package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/stackrox/central/analystnotes"
	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/generated/storage"
	pkgMetrics "github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("processComments(key: ProcessNoteKey!): [Comment!]!"),
		schema.AddQuery("processCommentsCount(key: ProcessNoteKey!): Int!"),
		schema.AddMutation("addProcessComment(key: ProcessNoteKey!, commentMessage: String!): String!"),
		schema.AddMutation("updateProcessComment(key: ProcessNoteKey!, commentId: ID!, commentMessage: String!): Boolean!"),
		schema.AddMutation("removeProcessComment(key: ProcessNoteKey!, commentId: ID!): Boolean!"),
	)
}

// ProcessComments returns a list of comments for a process.
func (resolver *Resolver) ProcessComments(ctx context.Context, args struct {
	Key analystnotes.ProcessNoteKey
}) ([]*commentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ProcessComments")
	if err := readIndicators(ctx); err != nil {
		return nil, err
	}
	return resolver.wrapComments(resolver.ProcessIndicatorStore.GetCommentsForProcess(ctx, &args.Key))
}

// ProcessCommentsCount returns the count of comments for a process key.
func (resolver *Resolver) ProcessCommentsCount(ctx context.Context, args struct {
	Key analystnotes.ProcessNoteKey
}) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ProcessCommentsCount")
	if err := readIndicators(ctx); err != nil {
		return 0, err
	}
	count, err := resolver.ProcessIndicatorStore.GetCommentsCountForProcess(ctx, &args.Key)
	if err != nil {
		return 0, err
	}
	return int32(count), nil
}

// AddProcessComment adds a process comment.
func (resolver *Resolver) AddProcessComment(ctx context.Context, args struct {
	Key            analystnotes.ProcessNoteKey
	CommentMessage string
}) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "AddProcessComment")
	if err := writeIndicators(ctx); err != nil {
		return "", err
	}

	comment := &storage.Comment{
		CommentMessage: args.CommentMessage,
	}
	commentID, err := resolver.ProcessIndicatorStore.AddProcessComment(ctx, &args.Key, comment)
	if err != nil {
		return "", err
	}
	return commentID, nil
}

// UpdateProcessComment updates a process comment.
func (resolver *Resolver) UpdateProcessComment(ctx context.Context, args struct {
	Key            analystnotes.ProcessNoteKey
	CommentID      graphql.ID
	CommentMessage string
}) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "UpdateProcessComment")
	if err := writeIndicators(ctx); err != nil {
		return false, err
	}
	request := &storage.Comment{
		CommentId:      string(args.CommentID),
		CommentMessage: args.CommentMessage,
	}

	err := resolver.ProcessIndicatorStore.UpdateProcessComment(ctx, &args.Key, request)
	if err != nil {
		return false, err
	}

	return true, nil
}

// RemoveProcessComment removes a process comment.
func (resolver *Resolver) RemoveProcessComment(ctx context.Context, args struct {
	Key       analystnotes.ProcessNoteKey
	CommentID graphql.ID
}) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "RemoveProcessComment")
	if err := writeIndicators(ctx); err != nil {
		return false, err
	}

	err := resolver.ProcessIndicatorStore.RemoveProcessComment(ctx, &args.Key, string(args.CommentID))
	if err != nil {
		return false, err
	}

	return true, nil
}
