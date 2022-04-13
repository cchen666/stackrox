package tenable

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	dockerRegistry "github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/httputil/proxy"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/scanners/types"
	"github.com/stackrox/stackrox/pkg/transports"
	"github.com/stackrox/stackrox/pkg/utils"
)

const (
	requestTimeout = 5 * time.Second

	typeString = "tenable"
)

// Variables so we can modify during unit tests
var (
	registry         = "registry.cloud.tenable.com"
	registryEndpoint = "https://" + registry
	apiEndpoint      = "https://cloud.tenable.com"
)

var (
	log = logging.LoggerForModule()
)

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Scanner, error)) {
	return typeString, func(integration *storage.ImageIntegration) (types.Scanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}

type tenable struct {
	types.ScanSemaphore

	client *http.Client

	config      *storage.TenableConfig
	reg         *dockerRegistry.Registry
	integration *storage.ImageIntegration
}

func validate(config *storage.TenableConfig) error {
	errorList := errorhelpers.NewErrorList("Tenable Validation")
	if config.GetAccessKey() == "" {
		errorList.AddString("Access key must be specified for Tenable scanner")
	}
	if config.GetSecretKey() == "" {
		errorList.AddString("Secret Key must be specified for Tenable scanner")
	}
	return errorList.ToError()
}

func newScanner(integration *storage.ImageIntegration) (*tenable, error) {
	tenableConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Tenable)
	if !ok {
		return nil, errors.New("Tenable configuration required")
	}
	config := tenableConfig.Tenable
	if err := validate(config); err != nil {
		return nil, err
	}

	tran, err := transports.NewPersistentTokenTransport(registryEndpoint, config.GetAccessKey(), config.GetSecretKey())
	if err != nil {
		return nil, err
	}
	reg, err := dockerRegistry.NewFromTransport(registryEndpoint, tran, dockerRegistry.Log)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			Proxy: proxy.FromConfig(),
		},
	}
	scanner := &tenable{
		client:      client,
		reg:         reg,
		config:      config,
		integration: integration,

		ScanSemaphore: types.NewDefaultSemaphore(),
	}
	return scanner, nil
}

func (d *tenable) sendRequest(method, urlPrefix string) ([]byte, int, error) {
	req, err := http.NewRequest(method, apiEndpoint+urlPrefix, nil)
	if err != nil {
		return nil, -1, err
	}
	req.Header.Add("X-ApiKeys", fmt.Sprintf("accessKey=%v; secretKey=%v", d.config.GetAccessKey(), d.config.GetSecretKey()))
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, -1, err
	}
	defer utils.IgnoreError(resp.Body.Close)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// Test initiates a test of the Tenable Scanner which verifies that we have the proper scan permissions
func (d *tenable) Test() error {
	body, status, err := d.sendRequest("GET", "/container-security/api/v1/container/list")
	if err != nil {
		return err
	} else if status != http.StatusOK {
		log.Errorf("unexpected status code '%d' when calling %s. Body: %s",
			status, apiEndpoint+"/container-security/api/v1/container/list", string(body))
		return errors.Errorf("unexpected status code '%d' when calling %s.",
			status, apiEndpoint+"/container-security/api/v1/container/list")
	}
	return nil
}

func getShortenedDigest(s string) string {
	s = strings.TrimPrefix(s, "sha256:")
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// GetScan retrieves the most recent scan
func (d *tenable) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	if image == nil || image.GetName().GetRemote() == "" || image.GetName().GetTag() == "" {
		return nil, nil
	}

	v1Digest := image.GetMetadata().GetV1().GetDigest()
	if v1Digest == "" {
		return nil, fmt.Errorf("could not get scan for image %q as we have not retrieved a valid v1 digest", image.GetName().GetFullName())
	}

	getScanURL := fmt.Sprintf("/container-security/api/v1/reports/by_image?image_id=%s", getShortenedDigest(v1Digest))

	body, status, err := d.sendRequest("GET", getScanURL)
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %v when retrieving image scan: %v", status, string(body))
	}
	scan, err := parseImageScan(body)
	if err != nil {
		return nil, err
	}
	return convertScanToImageScan(scan), nil
}

// Match decides if the image is contained within this registry
func (d *tenable) Match(image *storage.ImageName) bool {
	return registry == image.GetRegistry()
}

func (d *tenable) Type() string {
	return typeString
}

func (d *tenable) Name() string {
	return d.integration.GetName()
}

func (d *tenable) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return nil, nil
}
