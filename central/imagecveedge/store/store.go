package store

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// Store provides storage functionality for ImageCVEEdges.
//go:generate mockgen-wrapper
type Store interface {
	Count() (int, error)
	Exists(id string) (bool, error)

	GetAll() ([]*storage.ImageCVEEdge, error)
	Get(id string) (*storage.ImageCVEEdge, bool, error)
	GetMany(ids []string) ([]*storage.ImageCVEEdge, []int, error)
	UpdateVulnState(cve string, images []string, state storage.VulnerabilityState) error
}
