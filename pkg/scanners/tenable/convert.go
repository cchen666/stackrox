package tenable

import (
	"fmt"
	"strconv"
	"strings"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/scans"
	"github.com/stackrox/stackrox/pkg/stringutils"
)

func convertNVDFindingsAndPackagesToComponents(findings []*finding, packages []pkg) (components []*storage.EmbeddedImageScanComponent) {
	packagesToVulnerabilities := make(map[pkg][]*storage.EmbeddedVulnerability)
	for _, finding := range findings {
		// If there's an error then we are going to take the default value of 0.0 anyways
		cvssScore, err := strconv.ParseFloat(finding.NVDFinding.CVSSScore, 32)
		if err != nil {
			log.Warn(err)
		}
		vulnerability := &storage.EmbeddedVulnerability{
			Cvss:              float32(cvssScore),
			Cve:               finding.NVDFinding.CVE,
			Summary:           finding.NVDFinding.Description,
			Link:              scans.GetVulnLink(finding.NVDFinding.CVE),
			CvssV2:            convertCVSS(finding.NVDFinding),
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		}
		for _, affectedPackage := range finding.Packages {
			packagesToVulnerabilities[affectedPackage] = append(packagesToVulnerabilities[affectedPackage], vulnerability)
		}
	}
	for _, p := range packages {
		if _, ok := packagesToVulnerabilities[p]; !ok {
			packagesToVulnerabilities[p] = nil
		}
	}
	for p, vulns := range packagesToVulnerabilities {
		components = append(components, &storage.EmbeddedImageScanComponent{
			Name:    p.Name,
			Version: p.Version,
			Vulns:   vulns,
		})
	}
	return components
}

func convertScanToImageScan(s *scanResult) *storage.ImageScan {
	completedAt, err := ptypes.TimestampProto(s.UpdatedAt)
	if err != nil {
		log.Error(err)
	}

	os := stringutils.GetAfterLast(s.OS, "_")
	if os == "" {
		os = "unknown"
	} else {
		os = fmt.Sprintf("%s:%s", strings.ToLower(os), s.OSVersion)
	}

	return &storage.ImageScan{
		OperatingSystem: os,
		ScanTime:        completedAt,
		Components:      convertNVDFindingsAndPackagesToComponents(s.Findings, s.InstalledPackages),
	}
}

func getImpact(i string) storage.CVSSV2_Impact {
	i = strings.ToLower(i)
	switch {
	case strings.Contains(i, "none"):
		return storage.CVSSV2_IMPACT_NONE
	case strings.Contains(i, "partial"):
		return storage.CVSSV2_IMPACT_PARTIAL
	case strings.Contains(i, "complete"):
		return storage.CVSSV2_IMPACT_COMPLETE
	default:
		log.Errorf("Impact could not be parsed: %v", i)
		return storage.CVSSV2_IMPACT_COMPLETE
	}
}

func convertCVSS(nvd nvdFinding) *storage.CVSSV2 {
	var cvss storage.CVSSV2
	av := strings.ToLower(nvd.AccessVector)
	switch {
	case strings.Contains(av, "network"):
		cvss.AttackVector = storage.CVSSV2_ATTACK_NETWORK
	case strings.Contains(av, "adjacent"):
		cvss.AttackVector = storage.CVSSV2_ATTACK_ADJACENT
	case strings.Contains(av, "local"):
		cvss.AttackVector = storage.CVSSV2_ATTACK_LOCAL
	default:
		log.Errorf("Could not parse access vector %v", nvd.AccessVector)
		cvss.AttackVector = storage.CVSSV2_ATTACK_NETWORK
	}
	auth := strings.ToLower(nvd.Auth)
	switch {
	case strings.Contains(auth, "none"):
		cvss.Authentication = storage.CVSSV2_AUTH_NONE
	case strings.Contains(auth, "single"):
		cvss.Authentication = storage.CVSSV2_AUTH_SINGLE
	case strings.Contains(auth, "multiple"):
		cvss.Authentication = storage.CVSSV2_AUTH_MULTIPLE
	default:
		log.Errorf("Could not parse auth vector %v", nvd.Auth)
		cvss.Authentication = storage.CVSSV2_AUTH_MULTIPLE
	}
	access := strings.ToLower(nvd.AccessComplexity)
	switch {
	case strings.Contains(access, "low"):
		cvss.AccessComplexity = storage.CVSSV2_ACCESS_LOW
	case strings.Contains(access, "med"):
		cvss.AccessComplexity = storage.CVSSV2_ACCESS_MEDIUM
	case strings.Contains(access, "high"):
		cvss.AccessComplexity = storage.CVSSV2_ACCESS_HIGH
	default:
		log.Errorf("Could not parse access complexity %v", nvd.AccessComplexity)
		cvss.AccessComplexity = storage.CVSSV2_ACCESS_HIGH
	}
	cvss.Availability = getImpact(nvd.AvailabilityImpact)
	cvss.Confidentiality = getImpact(nvd.ConfidentialityImpact)
	cvss.Integrity = getImpact(nvd.IntegrityImpact)
	return &cvss
}
