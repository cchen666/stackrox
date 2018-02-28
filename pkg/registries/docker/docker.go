package docker

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/urlfmt"
	manifestV1 "github.com/docker/distribution/manifest/schema1"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/heroku/docker-registry-client/registry"
)

var (
	log = logging.LoggerForModule()
)

type dockerRegistry struct {
	protoRegistry *v1.Registry

	registry string

	getClientOnce sync.Once
	clientObj     client

	url      string
	username string
	password string
}

type v1Config struct {
	Cmd []string `json:"Cmd"`
}

// Parse out the layer JSON
type v1Compatibility struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
	Author  string    `json:"author"`
	Config  v1Config  `json:"container_config"`
}

type client interface {
	Manifest(repository, reference string) (*manifestV1.SignedManifest, error)
	Repositories() ([]string, error)
	Ping() error
}

type nilClient struct {
	error error
}

func (n nilClient) Manifest(repository, reference string) (*manifestV1.SignedManifest, error) {
	return nil, n.error
}

func (n nilClient) Repositories() ([]string, error) {
	return nil, n.error
}

func (n nilClient) Ping() error {
	return n.error
}

func newRegistry(protoRegistry *v1.Registry) (*dockerRegistry, error) {
	if protoRegistry.GetImageRegistry() == "" {
		return nil, errors.New("Image registry must be specified")
	}

	username, hasUsername := protoRegistry.Config["username"]
	password, hasPassword := protoRegistry.Config["password"]

	if hasUsername != hasPassword {
		if !hasUsername {
			return nil, errors.New("Config parameter 'username' must be defined for all non Docker Hub registries")
		}
		return nil, errors.New("Config parameter 'password' must be defined for all non Docker Hub registries")
	}

	if (!hasUsername && !hasPassword) && !strings.Contains(protoRegistry.Endpoint, "docker.io") {
		return nil, errors.New("Config parameters 'username' and 'password' must be defined for all non Docker Hub registries")
	}

	url, err := urlfmt.FormatURL(protoRegistry.Endpoint, true, false)
	if err != nil {
		return nil, err
	}
	return &dockerRegistry{
		protoRegistry: protoRegistry,
		registry:      protoRegistry.Endpoint,
		url:           url,
		username:      username,
		password:      password,
	}, nil
}

func (d *dockerRegistry) client() (c client) {
	d.getClientOnce.Do(func() {
		reg, err := registry.New(d.url, d.username, d.password)
		if err != nil {
			d.clientObj = nilClient{err}
			return
		}
		d.clientObj = reg
	})
	return d.clientObj
}

var scrubPrefixes = []string{
	"/bin/sh -c #(nop)",
	"/bin/sh -c ",
}

func scrubDockerfileLines(compat v1Compatibility) *v1.ImageLayer {
	line := strings.Join(compat.Config.Cmd, " ")
	for _, scrubPrefix := range scrubPrefixes {
		line = strings.TrimPrefix(line, scrubPrefix)
	}
	line = strings.Join(strings.Fields(line), " ")
	var lineInstruction string
	for instruction := range registries.DockerfileInstructionSet {
		if strings.HasPrefix(line, instruction) {
			lineInstruction = instruction
			line = strings.TrimPrefix(line, instruction+" ")
			break
		}
	}
	if lineInstruction == "" {
		lineInstruction = "RUN"
	}
	protoTS, err := ptypes.TimestampProto(compat.Created)
	if err != nil {
		log.Error(err)
	}
	return &v1.ImageLayer{
		Instruction: lineInstruction,
		Value:       line,
		Created:     protoTS,
		Author:      compat.Author,
	}
}

func compareProtoTimestamps(t1, t2 *timestamp.Timestamp) bool {
	if t1 == nil {
		return true
	}
	if t2 == nil {
		return false
	}
	if t1.Seconds < t2.Seconds {
		return true
	} else if t2.Seconds > t1.Seconds {
		return false
	}
	return t1.Nanos < t2.Nanos
}

func (d *dockerRegistry) getV2Metadata(image *v1.Image) *v1.V2Metadata {
	metadata, err := d.client().(*registry.Registry).ManifestV2(image.GetName().GetRemote(), image.GetName().GetTag())
	if err != nil {
		return nil
	}
	layers := make([]string, 0, len(metadata.Layers))
	for _, layer := range metadata.Layers {
		layers = append(layers, layer.Digest.String())
	}
	return &v1.V2Metadata{
		Digest: metadata.Config.Digest.String(),
		Layers: layers,
	}
}

// Metadata returns the metadata via this registries implementation
func (d *dockerRegistry) Metadata(image *v1.Image) (*v1.ImageMetadata, error) {
	log.Infof("Getting metadata for image %s", images.Wrapper{Image: image})
	if image == nil {
		return nil, nil
	}

	manifest, err := d.client().Manifest(image.GetName().GetRemote(), image.GetName().GetTag())
	if err != nil {
		return nil, err
	}

	// Get the latest layer and author
	var latest v1.ImageLayer
	var layers []*v1.ImageLayer
	for _, layer := range manifest.History {
		var compat v1Compatibility
		if err := json.Unmarshal([]byte(layer.V1Compatibility), &compat); err != nil {
			return nil, err
		}
		layer := scrubDockerfileLines(compat)
		if compareProtoTimestamps(latest.Created, layer.Created) {
			latest = *layer
		}
		layers = append(layers, layer)
	}
	fsLayers := make([]string, 0, len(manifest.FSLayers))
	for _, fsLayer := range manifest.FSLayers {
		fsLayers = append(fsLayers, fsLayer.BlobSum.String())
	}
	imageMetadata := &v1.ImageMetadata{
		Created:  latest.Created,
		Author:   latest.Author,
		Layers:   layers,
		FsLayers: fsLayers,
		V2:       d.getV2Metadata(image),
	}
	return imageMetadata, nil
}

func (d *dockerRegistry) ProtoRegistry() *v1.Registry {
	return d.protoRegistry
}

// Test tests the current registry and makes sure that it is working properly
func (d *dockerRegistry) Test() error {
	return d.client().Ping()
}

// Match decides if the image is contained within this registry
func (d *dockerRegistry) Match(image *v1.Image) bool {
	return d.protoRegistry.GetImageRegistry() == image.GetName().GetRegistry()
}

func (d *dockerRegistry) Global() bool {
	return len(d.protoRegistry.GetClusters()) == 0
}

func init() {
	f := func(registry *v1.Registry) (registries.ImageRegistry, error) {
		reg, err := newRegistry(registry)
		return reg, err
	}
	registries.Registry["docker"] = f
	registries.Registry["quay"] = f
}
