package dtr

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/httputil/proxy"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/scanners/types"
	"github.com/stackrox/stackrox/pkg/urlfmt"
	"github.com/stackrox/stackrox/pkg/utils"
)

const (
	requestTimeout = 30 * time.Second
	typeString     = "dtr"
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

type dtr struct {
	client *http.Client

	conf     config
	registry string

	protoImageIntegration *storage.ImageIntegration
	types.ScanSemaphore
}

type config storage.DTRConfig

func (c config) validate() error {
	errorList := errorhelpers.NewErrorList("Validation")
	if c.Username == "" {
		errorList.AddString("username parameter must be defined for DTR")
	}
	if c.Password == "" {
		errorList.AddString("password parameter must be defined for DTR")
	}
	if c.Endpoint == "" {
		errorList.AddString("endpoint parameter must be defined for DTR")
	}
	return errorList.ToError()
}

func newScanner(protoImageIntegration *storage.ImageIntegration) (*dtr, error) {
	dtrConfig, ok := protoImageIntegration.IntegrationConfig.(*storage.ImageIntegration_Dtr)
	if !ok {
		return nil, errors.New("DTR configuration required")
	}
	conf := config(*dtrConfig.Dtr)
	if err := conf.validate(); err != nil {
		return nil, err
	}

	// Trim any trailing slashes as the expectation will be that the input is in the form
	// https://12.12.12.12:8080 or https://dtr.com
	conf.Endpoint = urlfmt.FormatURL(conf.Endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	registry := urlfmt.GetServerFromURL(conf.Endpoint)
	client := &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: conf.Insecure,
			},
			Proxy: proxy.FromConfig(),
		},
	}

	scanner := &dtr{
		client:                client,
		registry:              registry,
		conf:                  conf,
		protoImageIntegration: protoImageIntegration,

		ScanSemaphore: types.NewDefaultSemaphore(),
	}

	return scanner, nil
}

func (d *dtr) sendRequest(method, urlPrefix string) ([]byte, error) {
	req, err := http.NewRequest(method, d.conf.Endpoint+urlPrefix, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(d.conf.Username, d.conf.Password)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	if err := errorFromStatusCode(resp.StatusCode); err != nil {
		return nil, err
	}
	defer utils.IgnoreError(resp.Body.Close)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Error reading Docker Trusted Registry response body")
	}
	return body, nil
}

// getScan takes in an id and returns the image scan for that id if applicable
func (d *dtr) getScan(image *storage.Image) (*storage.ImageScan, error) {
	if image == nil || image.GetName().GetRemote() == "" || image.GetName().GetTag() == "" {
		return nil, nil
	}
	getScanURL := fmt.Sprintf("/api/v0/imagescan/repositories/%v/%v?detailed=true", image.GetName().GetRemote(), image.GetName().GetTag())
	body, err := d.sendRequest(http.MethodGet, getScanURL)
	if err != nil {
		return nil, err
	}

	scans, err := parseDTRImageScans(body)
	if err != nil {
		scanErrors, err := parseDTRImageScanErrors(body)
		if err != nil {
			return nil, err
		}
		var errMsg string
		for _, scanErr := range scanErrors.Errors {
			errMsg += scanErr.Message + "\n"
		}
		return nil, errors.New(errMsg)
	}
	if len(scans) == 0 {
		return nil, fmt.Errorf("expected to receive at least one scan for %v", image.GetName().GetFullName())
	}

	// Find the last scan time
	lastScan := scans[0]
	for _, s := range scans {
		if s.CheckCompletedAt.After(lastScan.CheckCompletedAt) {
			lastScan = s
		}
	}
	if lastScan.CheckCompletedAt.IsZero() {
		return nil, fmt.Errorf("expected to receive at least one scan for %s", image.GetName().GetFullName())
	}

	scan := convertTagScanSummaryToImageScan(image, lastScan)
	return scan, nil
}

func errorFromStatusCode(status int) error {
	switch status {
	case 400:
		return errors.New("HTTP 400: Scanning is not enabled")
	case 401:
		return errors.New("HTTP 401: The client is not authenticated")
	case 405:
		return errors.New("HTTP 405: Method Not Allowed")
	case 406:
		return errors.New("HTTP 406: Not Acceptable")
	case 415:
		return errors.New("HTTP 415: Unsupported Media Type")
	case 200:
	default:
		return nil
	}
	return nil
}

// Test initiates a test of the DTR which verifies that we have the proper scan permissions
func (d *dtr) Test() error {
	_, err := d.sendRequest(http.MethodGet, "/api/v0/imagescan/status")
	return err
}

// GetScan retrieves the most recent scan
func (d *dtr) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	log.Infof("Getting latest scan for image %s", image.GetName().GetFullName())
	return d.getScan(image)
}

// Match decides if the image is contained within this registry
func (d *dtr) Match(image *storage.ImageName) bool {
	return d.registry == image.GetRegistry()
}

func (d *dtr) Type() string {
	return typeString
}

func (d *dtr) Name() string {
	return d.protoImageIntegration.GetName()
}

func (d *dtr) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return nil, nil
}
