package commentsstore

import (
	"github.com/stackrox/stackrox/central/analystnotes"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"go.etcd.io/bbolt"
)

var (
	processCommentsBucket = []byte("process_comments")
)

// Store stores process comments.
//go:generate mockgen-wrapper
type Store interface {
	AddProcessComment(key *analystnotes.ProcessNoteKey, comment *storage.Comment) (string, error)
	UpdateProcessComment(key *analystnotes.ProcessNoteKey, comment *storage.Comment) error

	GetComment(key *analystnotes.ProcessNoteKey, commentID string) (*storage.Comment, error)
	GetComments(key *analystnotes.ProcessNoteKey) ([]*storage.Comment, error)
	GetCommentsCount(key *analystnotes.ProcessNoteKey) (int, error)

	RemoveProcessComment(key *analystnotes.ProcessNoteKey, commentID string) error
	RemoveAllProcessComments(key *analystnotes.ProcessNoteKey) error
}

// New returns a new, ready-to-use, store.
func New(db *bbolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, processCommentsBucket)
	return &storeImpl{
		bucketRef: bolthelper.TopLevelRef(db, processCommentsBucket),
	}
}
