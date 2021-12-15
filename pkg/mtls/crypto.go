package mtls

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"time"

	"github.com/cloudflare/cfssl/config"
	cfcsr "github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	cflog "github.com/cloudflare/cfssl/log"
	cfsigner "github.com/cloudflare/cfssl/signer"
	"github.com/cloudflare/cfssl/signer/local"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/x509utils"
)

const (
	// CACertFileName is the canonical file name (basename) of the file storing the CA certificate.
	CACertFileName = "ca.pem"
	// CAKeyFileName is the canonical file name (basename) of the file storing the CA certificate private key.
	CAKeyFileName = "ca-key.pem"

	// ServiceCertFileName is the canonical file name (basename) of the file storing the public part of
	// an internal service certificate. Note that if files for several services are stored in the same
	// location (directory or file map), it is common to prefix the file name with the service name in
	// slug-case (e.g., `scanner-db-cert.pem`).
	ServiceCertFileName = "cert.pem"
	// ServiceKeyFileName is the canonical file name (basename) of the file storing the private key for
	// an internal service certificate. The same remark as above regarding prefixes applies.
	ServiceKeyFileName = "key.pem"

	// CertsPrefix is the filesystem prefix under which service certificates and keys are stored.
	CertsPrefix = "/run/secrets/stackrox.io/certs/"
	// defaultCACertFilePath is where the certificate is stored.
	defaultCACertFilePath = CertsPrefix + CACertFileName
	// defaultCAKeyFilePath is where the key is stored.
	defaultCAKeyFilePath = CertsPrefix + CAKeyFileName

	// defaultCertFilePath is where the certificate is stored.
	defaultCertFilePath = CertsPrefix + ServiceCertFileName
	// defaultKeyFilePath is where the key is stored.
	defaultKeyFilePath = CertsPrefix + ServiceKeyFileName

	// To account for clock skew, set certificates to be valid some time in the past.
	beforeGracePeriod = 1 * time.Hour

	certLifetime = 365 * 24 * time.Hour

	ephemeralProfile = "ephemeral"
	ephemeralInitBundleCertLifetime = 3 * time.Hour
)

var (
	// serialMax is the max value to be used with `rand.Int` to obtain a `*big.Int` with 64 bits of random data
	// (i.e., 1 << 64).
	serialMax = func() *big.Int {
		max := big.NewInt(1)
		max.Lsh(max, 64)
		return max
	}()
)

func init() {
	// The cfssl library prints logs at Info level when it processes a
	// Certificate Signing Request (CSR) or issues a new certificate.
	// These logs do not help the user understand anything, so here
	// we adjust the log level to exclude them.
	cflog.Level = cflog.LevelWarning
}

var (
	// CentralSubject is the identity used in certificates for Central.
	CentralSubject = Subject{ServiceType: storage.ServiceType_CENTRAL_SERVICE, Identifier: "Central"}

	// SensorSubject is the identity used in certificates for Sensor.
	SensorSubject = Subject{ServiceType: storage.ServiceType_SENSOR_SERVICE, Identifier: "Sensor"}

	// AdmissionControlSubject is the identity used in certificates for Admission Control.
	AdmissionControlSubject = Subject{ServiceType: storage.ServiceType_ADMISSION_CONTROL_SERVICE, Identifier: "Admission Control"}

	// ScannerSubject is the identity used in certificates for Scanner.
	ScannerSubject = Subject{ServiceType: storage.ServiceType_SCANNER_SERVICE, Identifier: "Scanner"}

	// ScannerDBSubject is the identity used in certificates for Scanners Postgres DB
	ScannerDBSubject = Subject{ServiceType: storage.ServiceType_SCANNER_DB_SERVICE, Identifier: "Scanner DB"}

	readCACertOnce     sync.Once
	caCert             *x509.Certificate
	caCertDER          []byte
	caCertFileContents []byte
	caCertErr          error

	readCAKeyOnce     sync.Once
	caKeyFileContents []byte
	caKeyErr          error
)

// IssuedCert is a representation of an issued certificate
type IssuedCert struct {
	CertPEM  []byte
	KeyPEM   []byte
	X509Cert *x509.Certificate
	ID       *storage.ServiceIdentity
}

// LeafCertificateFromFile reads a tls.Certificate (including private key and cert).
func LeafCertificateFromFile() (tls.Certificate, error) {
	return tls.LoadX509KeyPair(certFilePathSetting.Setting(), keyFilePathSetting.Setting())
}

// CACertPEM returns the PEM-encoded CA certificate.
func CACertPEM() ([]byte, error) {
	_, caDER, err := CACert()
	if err != nil {
		return nil, errors.Wrap(err, "CA cert loading")
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caDER,
	}), nil
}

func readCAKey() ([]byte, error) {
	readCAKeyOnce.Do(func() {
		caKeyBytes, err := os.ReadFile(caKeyFilePathSetting.Setting())
		if err != nil {
			caKeyErr = errors.Wrap(err, "reading CA key")
			return
		}
		caKeyFileContents = caKeyBytes
	})
	return caKeyFileContents, caKeyErr
}

func readCA() (*x509.Certificate, []byte, []byte, error) {
	readCACertOnce.Do(func() {
		caBytes, err := os.ReadFile(caFilePathSetting.Setting())
		if err != nil {
			caCertErr = errors.Wrap(err, "reading CA file")
			return
		}

		der, err := x509utils.ConvertPEMToDERs(caBytes)
		if err != nil {
			caCertErr = errors.Wrap(err, "CA cert could not be decoded")
			return
		}
		if len(der) == 0 {
			caCertErr = errors.New("reading CA file failed")
			return
		}

		cert, err := x509.ParseCertificate(der[0])
		if err != nil {
			caCertErr = errors.Wrap(err, "CA cert could not be parsed")
			return
		}
		caCertFileContents = caBytes
		caCert = cert
		caCertDER = der[0]
	})
	return caCert, caCertFileContents, caCertDER, caCertErr
}

// CACert reads the cert from the local file system and returns the cert and the DER encoding.
func CACert() (*x509.Certificate, []byte, error) {
	caCert, _, caCertDER, caCertErr := readCA()
	return caCert, caCertDER, caCertErr
}

func signer() (cfsigner.Signer, error) {
	return local.NewSignerFromFile(caFilePathSetting.Setting(), caKeyFilePathSetting.Setting(), signingPolicy())
}

func signingPolicy() *config.Signing {
	return &config.Signing{
		Default: profile(certLifetime, beforeGracePeriod),
		Profiles: map[string]*config.SigningProfile{
			ephemeralProfile: profile(ephemeralInitBundleCertLifetime, 0),
		},
	}
}

func profile(lifetime time.Duration, gracePeriod time.Duration) *config.SigningProfile {
	return &config.SigningProfile{
		Usage:    []string{"signing", "key encipherment", "server auth", "client auth"},
		Expiry:   lifetime + gracePeriod,
		Backdate: gracePeriod,
		CSRWhitelist: &config.CSRWhitelist{
			PublicKey:          true,
			PublicKeyAlgorithm: true,
			SignatureAlgorithm: true,
		},
		ClientProvidesSerialNumbers: true,
	}
}

func validateSubject(subj Subject) error {
	errorList := errorhelpers.NewErrorList("")
	if subj.ServiceType == storage.ServiceType_UNKNOWN_SERVICE {
		errorList.AddString("Subject service type must be known")
	}
	if subj.Identifier == "" {
		errorList.AddString("Subject Identifier must be non-empty")
	}
	return errorList.ToError()
}

func issueNewCertFromSigner(subj Subject, signer cfsigner.Signer, opts []IssueCertOption) (*IssuedCert, error) {
	if err := validateSubject(subj); err != nil {
		// Purposefully didn't use returnErr because errorList.ToError() returned from validateSubject is already prefixed
		return nil, err
	}

	serial, err := RandomSerial()
	if err != nil {
		return nil, errors.Wrap(err, "serial generation")
	}

	csr := &cfcsr.CertificateRequest{
		KeyRequest:   cfcsr.NewBasicKeyRequest(),
		SerialNumber: serial.String(),
	}
	csrBytes, keyBytes, err := cfcsr.ParseRequest(csr)
	if err != nil {
		return nil, errors.Wrap(err, "request parsing")
	}

	var issueOpts issueOptions
	issueOpts.apply(opts)

	var hosts []string
	hosts = append(hosts, subj.AllHostnames()...)
	if ns := issueOpts.namespace; ns != "" && ns != namespaces.StackRox {
		hosts = append(hosts, subj.AllHostnamesForNamespace(ns)...)
	}

	req := cfsigner.SignRequest{
		Hosts:   hosts,
		Request: string(csrBytes),
		Subject: &cfsigner.Subject{
			CN:           subj.CN(),
			Names:        []cfcsr.Name{subj.Name()},
			SerialNumber: serial.String(),
		},
		Serial: serial,
		Profile: issueOpts.profile,
	}
	certBytes, err := signer.Sign(req)
	if err != nil {
		return nil, errors.Wrap(err, "signing")
	}

	x509Cert, err := helpers.ParseCertificatePEM(certBytes)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse generated PEM cert")
	}

	id := generateIdentity(subj, serial)

	return &IssuedCert{
		CertPEM:  certBytes,
		KeyPEM:   keyBytes,
		X509Cert: x509Cert,
		ID:       id,
	}, nil
}

// IssueNewCert generates a new key and certificate chain for a sensor.
func IssueNewCert(subj Subject, opts ...IssueCertOption) (cert *IssuedCert, err error) {
	s, err := signer()
	if err != nil {
		return nil, errors.Wrap(err, "signer creation")
	}
	return issueNewCertFromSigner(subj, s, opts)
}

// RandomSerial returns a new integer that can be used as a certificate serial number (i.e., it is positive and contains
// 64 bits of random data).
func RandomSerial() (*big.Int, error) {
	serial, err := rand.Int(rand.Reader, serialMax)
	if err != nil {
		return nil, errors.Wrap(err, "serial number generation")
	}
	serial.Add(serial, big.NewInt(1)) // Serial numbers must be positive.
	return serial, nil
}

func generateIdentity(subj Subject, serial *big.Int) *storage.ServiceIdentity {
	return &storage.ServiceIdentity{
		Id:   subj.Identifier,
		Type: subj.ServiceType,
		Srl: &storage.ServiceIdentity_SerialStr{
			SerialStr: serial.String(),
		},
	}
}
