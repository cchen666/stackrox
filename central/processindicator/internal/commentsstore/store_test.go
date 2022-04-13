package commentsstore

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stackrox/stackrox/central/analystnotes"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func getKey(deploymentSeed, containerSeed int) *analystnotes.ProcessNoteKey {
	key := &analystnotes.ProcessNoteKey{
		DeploymentID:  fmt.Sprintf("DEPLOY%d", deploymentSeed),
		ContainerName: fmt.Sprintf("CONTAINER%d", containerSeed),
		ExecFilePath:  "EXEC",
	}
	if deploymentSeed%2 == 0 {
		key.Args = "ARGS"
	}
	return key
}

func TestStore(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}

type StoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *StoreTestSuite) SetupTest() {
	suite.db = testutils.DBForSuite(suite)
	suite.store = New(suite.db)
}

func (suite *StoreTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *StoreTestSuite) mustGetComments(deploymentSeed, containerSeed int) []*storage.Comment {
	key := getKey(deploymentSeed, containerSeed)
	comments, err := suite.store.GetComments(key)
	suite.Require().NoError(err)
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].GetCommentMessage() < comments[j].GetCommentMessage()
	})

	// Validate the count every time
	count, err := suite.store.GetCommentsCount(key)
	suite.Require().NoError(err)
	suite.Equal(len(comments), count)
	return comments
}

func getBareComment(id, msg string) *storage.Comment {
	c := getBareCommentNoID(msg)
	c.CommentId = id
	return c
}

func getBareCommentNoID(msg string) *storage.Comment {
	return &storage.Comment{
		User:           &storage.Comment_User{Id: "blah"},
		CommentMessage: msg,
	}
}

func (suite *StoreTestSuite) mustAddComment(keyDeploymentSeed, keyContainerSeed int, msg string) string {
	id, err := suite.store.AddProcessComment(getKey(keyDeploymentSeed, keyContainerSeed), getBareCommentNoID(msg))
	suite.Require().NoError(err)
	return id
}

func (suite *StoreTestSuite) checkCommentIsHydratedVersionOf(
	gotComment, expectedBaseComment *storage.Comment,
	earliestCreated, latestCreated, earliestModifed, latestModified time.Time) {

	clonedGotComment := gotComment.Clone()
	testutils.ValidateTSInWindow(clonedGotComment.GetCreatedAt(), earliestCreated, latestCreated, suite.T())
	testutils.ValidateTSInWindow(clonedGotComment.GetLastModified(), earliestModifed, latestModified, suite.T())
	clonedGotComment.CreatedAt = nil
	clonedGotComment.LastModified = nil

	expectedBaseComment.ResourceType = storage.ResourceType_PROCESS
	suite.Equal(expectedBaseComment, clonedGotComment)
}

func (suite *StoreTestSuite) TestStore() {
	suite.Empty(suite.mustGetComments(1, 1))

	justBeforeAdd := time.Now()
	firstID := suite.mustAddComment(1, 1, "COMMENT1")
	secondID := suite.mustAddComment(1, 1, "COMMENT2")
	thirdID := suite.mustAddComment(1, 1, "COMMENT3")
	suite.mustAddComment(1, 2, "BLAH")
	justAfterAdd := time.Now()

	commentsAfterAdd := suite.mustGetComments(1, 1)
	suite.Len(commentsAfterAdd, 3)
	suite.checkCommentIsHydratedVersionOf(commentsAfterAdd[0], getBareComment(firstID, "COMMENT1"),
		justBeforeAdd, justAfterAdd, justBeforeAdd, justAfterAdd)
	suite.checkCommentIsHydratedVersionOf(commentsAfterAdd[1], getBareComment(secondID, "COMMENT2"),
		justBeforeAdd, justAfterAdd, justBeforeAdd, justAfterAdd)
	suite.checkCommentIsHydratedVersionOf(commentsAfterAdd[2], getBareComment(thirdID, "COMMENT3"),
		justBeforeAdd, justAfterAdd, justBeforeAdd, justAfterAdd)

	comment1ByID, err := suite.store.GetComment(getKey(1, 1), firstID)
	suite.NoError(err)
	suite.Equal(commentsAfterAdd[0], comment1ByID)

	justBeforeUpdate := time.Now()
	suite.NoError(suite.store.UpdateProcessComment(getKey(1, 1), getBareComment(firstID, "COMMENT1UPDATED")))
	justAfterUpdate := time.Now()

	commentsAfterUpdate := suite.mustGetComments(1, 1)
	suite.Len(commentsAfterUpdate, 3)
	suite.checkCommentIsHydratedVersionOf(commentsAfterUpdate[0], getBareComment(firstID, "COMMENT1UPDATED"),
		justBeforeAdd, justAfterAdd, justBeforeUpdate, justAfterUpdate)
	suite.checkCommentIsHydratedVersionOf(commentsAfterUpdate[1], getBareComment(secondID, "COMMENT2"),
		justBeforeAdd, justAfterAdd, justBeforeAdd, justAfterAdd)
	suite.checkCommentIsHydratedVersionOf(commentsAfterUpdate[2], getBareComment(thirdID, "COMMENT3"),
		justBeforeAdd, justAfterAdd, justBeforeAdd, justAfterAdd)

	suite.NoError(suite.store.RemoveProcessComment(getKey(1, 1), secondID))
	commentsAfterRemove := suite.mustGetComments(1, 1)
	suite.Len(commentsAfterRemove, 2)
	suite.checkCommentIsHydratedVersionOf(commentsAfterRemove[0], getBareComment(firstID, "COMMENT1UPDATED"),
		justBeforeAdd, justAfterAdd, justBeforeUpdate, justAfterUpdate)
	suite.checkCommentIsHydratedVersionOf(commentsAfterRemove[1], getBareComment(thirdID, "COMMENT3"),
		justBeforeAdd, justAfterAdd, justBeforeAdd, justAfterAdd)

	suite.NoError(suite.store.RemoveAllProcessComments(getKey(1, 1)))

	commentsAfterBigRemove := suite.mustGetComments(1, 1)
	suite.Len(commentsAfterBigRemove, 0)
}
