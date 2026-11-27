// Package pubsub emulates google.pubsub.v1 over gRPC and REST (via
// grpc-gateway). State is in-memory only.
//
// The implementation includes topic/subscription control-plane CRUD and the
// pull message data plane (publish, pull, acknowledge, deadlines, snapshots,
// seek, and streaming pull) with in-memory at-least-once semantics.
package pubsub

import (
	"context"
	"io"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
	"github.com/bestony/google-cloud-emulator/core/pagination"
	pubsubpb "github.com/bestony/google-cloud-emulator/gen/google/pubsub/v1"
)

func invalidArg(format string, args ...any) error {
	return gcperr.GRPC(gcperr.InvalidArgument, format, args...)
}

func notFound(format string, args ...any) error {
	return gcperr.GRPC(gcperr.NotFound, format, args...)
}

func stateErr(code string) error {
	switch code {
	case "not-found":
		return gcperr.GRPC(gcperr.NotFound, "resource not found")
	case "snapshot-not-found":
		return gcperr.GRPC(gcperr.NotFound, "snapshot not found")
	case "snapshot-subscription-mismatch":
		return gcperr.GRPC(gcperr.FailedPrecondition, "snapshot does not belong to the subscription")
	case "already-exists":
		return gcperr.GRPC(gcperr.AlreadyExists, "resource already exists")
	case "invalid-name", "invalid-parent", "invalid-topic":
		return gcperr.GRPC(gcperr.InvalidArgument, "invalid resource name or parent")
	case "topic-not-found":
		return gcperr.GRPC(gcperr.NotFound, "topic not found")
	case "invalid-page-token":
		return gcperr.GRPC(gcperr.InvalidArgument, "invalid page token")
	default:
		if strings.HasPrefix(code, "invalid-mask:") {
			return gcperr.GRPC(gcperr.InvalidArgument, "update mask path %q is not supported by the emulator", strings.TrimPrefix(code, "invalid-mask:"))
		}
		return gcperr.GRPC(gcperr.InvalidArgument, "%s", code)
	}
}

// PublisherServer implements google.pubsub.v1.Publisher.
type PublisherServer struct {
	pubsubpb.UnimplementedPublisherServer
	state *State
}

// NewPublisherServer returns a PublisherServer backed by state.
func NewPublisherServer(state *State) *PublisherServer {
	return &PublisherServer{state: state}
}

// SubscriberServer implements google.pubsub.v1.Subscriber.
type SubscriberServer struct {
	pubsubpb.UnimplementedSubscriberServer
	state *State
}

// NewSubscriberServer returns a SubscriberServer backed by state.
func NewSubscriberServer(state *State) *SubscriberServer {
	return &SubscriberServer{state: state}
}

// --- Publisher: topics ---

func (s *PublisherServer) CreateTopic(ctx context.Context, topic *pubsubpb.Topic) (*pubsubpb.Topic, error) {
	if topic == nil || topic.Name == "" {
		return nil, invalidArg("topic name is required")
	}
	out, code := s.state.CreateTopic(topic)
	if code != "" {
		return nil, stateErr(code)
	}
	return out, nil
}

func (s *PublisherServer) UpdateTopic(ctx context.Context, req *pubsubpb.UpdateTopicRequest) (*pubsubpb.Topic, error) {
	if req == nil || req.Topic == nil || req.Topic.Name == "" {
		return nil, invalidArg("topic name is required")
	}
	var paths []string
	if req.UpdateMask != nil {
		paths = req.UpdateMask.GetPaths()
	}
	out, code := s.state.UpdateTopic(req.Topic.Name, req.Topic, paths)
	if code != "" {
		return nil, stateErr(code)
	}
	return out, nil
}

func (s *PublisherServer) GetTopic(ctx context.Context, req *pubsubpb.GetTopicRequest) (*pubsubpb.Topic, error) {
	t := s.state.GetTopic(req.GetTopic())
	if t == nil {
		return nil, notFound("topic %q not found", req.GetTopic())
	}
	return t, nil
}

func (s *PublisherServer) ListTopics(ctx context.Context, req *pubsubpb.ListTopicsRequest) (*pubsubpb.ListTopicsResponse, error) {
	if !ValidProjectParent(req.GetProject()) {
		return nil, invalidArg("project must be of the form projects/{project}")
	}
	pageSize := pagination.EffectivePageSize(req.GetPageSize(), 100)
	topics, next, _, code := s.state.ListTopics(req.GetProject(), pageSize, req.GetPageToken())
	if code != "" {
		return nil, stateErr(code)
	}
	return &pubsubpb.ListTopicsResponse{Topics: topics, NextPageToken: next}, nil
}

func (s *PublisherServer) ListTopicSubscriptions(ctx context.Context, req *pubsubpb.ListTopicSubscriptionsRequest) (*pubsubpb.ListTopicSubscriptionsResponse, error) {
	if req.GetTopic() == "" {
		return nil, invalidArg("topic is required")
	}
	pageSize := pagination.EffectivePageSize(req.GetPageSize(), 100)
	names, next, _, code := s.state.SubscriptionPageByTopic(req.GetTopic(), pageSize, req.GetPageToken())
	if code != "" {
		return nil, stateErr(code)
	}
	return &pubsubpb.ListTopicSubscriptionsResponse{Subscriptions: names, NextPageToken: next}, nil
}

func (s *PublisherServer) DeleteTopic(ctx context.Context, req *pubsubpb.DeleteTopicRequest) (*emptypb.Empty, error) {
	code := s.state.DeleteTopic(req.GetTopic())
	if code != "" {
		return nil, stateErr(code)
	}
	return &emptypb.Empty{}, nil
}

func (s *PublisherServer) DetachSubscription(ctx context.Context, req *pubsubpb.DetachSubscriptionRequest) (*pubsubpb.DetachSubscriptionResponse, error) {
	code := s.state.DetachSubscription(req.GetSubscription())
	if code != "" {
		return nil, stateErr(code)
	}
	return &pubsubpb.DetachSubscriptionResponse{}, nil
}

// --- Subscriber: subscriptions ---

func (s *SubscriberServer) CreateSubscription(ctx context.Context, sub *pubsubpb.Subscription) (*pubsubpb.Subscription, error) {
	if sub == nil || sub.Topic == "" {
		return nil, invalidArg("subscription topic is required")
	}
	out, code := s.state.CreateSubscription(sub)
	if code != "" {
		return nil, stateErr(code)
	}
	return out, nil
}

func (s *SubscriberServer) GetSubscription(ctx context.Context, req *pubsubpb.GetSubscriptionRequest) (*pubsubpb.Subscription, error) {
	sub := s.state.GetSubscription(req.GetSubscription())
	if sub == nil {
		return nil, notFound("subscription %q not found", req.GetSubscription())
	}
	return sub, nil
}

func (s *SubscriberServer) UpdateSubscription(ctx context.Context, req *pubsubpb.UpdateSubscriptionRequest) (*pubsubpb.Subscription, error) {
	if req == nil || req.Subscription == nil || req.Subscription.Name == "" {
		return nil, invalidArg("subscription name is required")
	}
	var paths []string
	if req.UpdateMask != nil {
		paths = req.UpdateMask.GetPaths()
	}
	out, code := s.state.UpdateSubscription(req.Subscription.Name, req.Subscription, paths)
	if code != "" {
		return nil, stateErr(code)
	}
	return out, nil
}

func (s *SubscriberServer) ListSubscriptions(ctx context.Context, req *pubsubpb.ListSubscriptionsRequest) (*pubsubpb.ListSubscriptionsResponse, error) {
	if !ValidProjectParent(req.GetProject()) {
		return nil, invalidArg("project must be of the form projects/{project}")
	}
	pageSize := pagination.EffectivePageSize(req.GetPageSize(), 100)
	subs, next, _, code := s.state.ListSubscriptions(req.GetProject(), pageSize, req.GetPageToken())
	if code != "" {
		return nil, stateErr(code)
	}
	return &pubsubpb.ListSubscriptionsResponse{Subscriptions: subs, NextPageToken: next}, nil
}

func (s *SubscriberServer) DeleteSubscription(ctx context.Context, req *pubsubpb.DeleteSubscriptionRequest) (*emptypb.Empty, error) {
	code := s.state.DeleteSubscription(req.GetSubscription())
	if code != "" {
		return nil, stateErr(code)
	}
	return &emptypb.Empty{}, nil
}

func (s *SubscriberServer) ModifyPushConfig(ctx context.Context, req *pubsubpb.ModifyPushConfigRequest) (*emptypb.Empty, error) {
	if req.GetSubscription() == "" {
		return nil, invalidArg("subscription is required")
	}
	if req.GetPushConfig() == nil || req.GetPushConfig().GetPushEndpoint() == "" {
		return nil, invalidArg("push_config.push_endpoint is required")
	}
	code := s.state.SetPushConfig(req.GetSubscription(), req.GetPushConfig())
	if code != "" {
		return nil, stateErr(code)
	}
	return &emptypb.Empty{}, nil
}

// --- Data plane ---

func (s *PublisherServer) Publish(ctx context.Context, req *pubsubpb.PublishRequest) (*pubsubpb.PublishResponse, error) {
	if req.GetTopic() == "" {
		return nil, invalidArg("topic is required")
	}
	ids, code := s.state.PublishMessages(req.GetTopic(), req.GetMessages())
	if code != "" {
		return nil, stateErr(code)
	}
	return &pubsubpb.PublishResponse{MessageIds: ids}, nil
}

func (s *SubscriberServer) Pull(ctx context.Context, req *pubsubpb.PullRequest) (*pubsubpb.PullResponse, error) {
	if req.GetSubscription() == "" {
		return nil, invalidArg("subscription is required")
	}
	messages, code := s.state.PullMessages(req.GetSubscription(), req.GetMaxMessages(), 0)
	if code != "" {
		return nil, stateErr(code)
	}
	return &pubsubpb.PullResponse{ReceivedMessages: messages}, nil
}

func (s *SubscriberServer) Acknowledge(ctx context.Context, req *pubsubpb.AcknowledgeRequest) (*emptypb.Empty, error) {
	if req.GetSubscription() == "" {
		return nil, invalidArg("subscription is required")
	}
	if code := s.state.Acknowledge(req.GetSubscription(), req.GetAckIds()); code != "" {
		return nil, stateErr(code)
	}
	return &emptypb.Empty{}, nil
}

func (s *SubscriberServer) ModifyAckDeadline(ctx context.Context, req *pubsubpb.ModifyAckDeadlineRequest) (*emptypb.Empty, error) {
	if req.GetSubscription() == "" {
		return nil, invalidArg("subscription is required")
	}
	if code := s.state.ModifyAckDeadline(req.GetSubscription(), req.GetAckIds(), req.GetAckDeadlineSeconds()); code != "" {
		return nil, stateErr(code)
	}
	return &emptypb.Empty{}, nil
}

func applyStreamingControl(state *State, subscription string, req *pubsubpb.StreamingPullRequest) error {
	if len(req.GetModifyDeadlineAckIds()) != len(req.GetModifyDeadlineSeconds()) {
		return invalidArg("modify_deadline_ack_ids and modify_deadline_seconds must have the same length")
	}
	if len(req.GetAckIds()) > 0 {
		if code := state.Acknowledge(subscription, req.GetAckIds()); code != "" {
			return stateErr(code)
		}
	}
	for i, id := range req.GetModifyDeadlineAckIds() {
		if code := state.ModifyAckDeadline(subscription, []string{id}, req.GetModifyDeadlineSeconds()[i]); code != "" {
			return stateErr(code)
		}
	}
	return nil
}

// StreamingPull implements a small but useful streaming receiver: the first
// request initializes the subscription, subsequent requests carry acks and
// deadline modifications, and the server continuously sends available batches.
func (s *SubscriberServer) StreamingPull(stream grpc.BidiStreamingServer[pubsubpb.StreamingPullRequest, pubsubpb.StreamingPullResponse]) error {
	first, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	subscription := first.GetSubscription()
	if subscription == "" {
		return invalidArg("subscription is required in the first streaming pull request")
	}
	if s.state.GetSubscription(subscription) == nil {
		return notFound("subscription %q not found", subscription)
	}
	deadline := first.GetStreamAckDeadlineSeconds()
	if deadline != 0 && (deadline < 10 || deadline > 600) {
		return invalidArg("stream_ack_deadline_seconds must be between 10 and 600")
	}
	if err := applyStreamingControl(s.state, subscription, first); err != nil {
		return err
	}

	recvDone := make(chan error, 1)
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				recvDone <- err
				return
			}
			if req.GetSubscription() != "" {
				recvDone <- invalidArg("subscription must only be set in the first streaming pull request")
				return
			}
			if req.GetStreamAckDeadlineSeconds() != 0 && (req.GetStreamAckDeadlineSeconds() < 10 || req.GetStreamAckDeadlineSeconds() > 600) {
				recvDone <- invalidArg("stream_ack_deadline_seconds must be between 10 and 600")
				return
			}
			if err := applyStreamingControl(s.state, subscription, req); err != nil {
				recvDone <- err
				return
			}
		}
	}()

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case err := <-recvDone:
			if err == io.EOF {
				return nil
			}
			return err
		default:
		}

		messages, code := s.state.PullMessages(subscription, 10, deadline)
		if code != "" {
			return stateErr(code)
		}
		if len(messages) > 0 {
			if err := stream.Send(&pubsubpb.StreamingPullResponse{ReceivedMessages: messages}); err != nil {
				return err
			}
			continue
		}

		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case err := <-recvDone:
			if err == io.EOF {
				return nil
			}
			return err
		case <-ticker.C:
		}
	}
}

// --- Snapshots and seek ---

func (s *SubscriberServer) CreateSnapshot(ctx context.Context, req *pubsubpb.CreateSnapshotRequest) (*pubsubpb.Snapshot, error) {
	if req.GetSubscription() == "" {
		return nil, invalidArg("subscription is required")
	}
	snap, code := s.state.CreateSnapshot(req.GetName(), req.GetSubscription(), req.GetLabels())
	if code != "" {
		return nil, stateErr(code)
	}
	return snap, nil
}

func (s *SubscriberServer) GetSnapshot(ctx context.Context, req *pubsubpb.GetSnapshotRequest) (*pubsubpb.Snapshot, error) {
	snap := s.state.GetSnapshot(req.GetSnapshot())
	if snap == nil {
		return nil, notFound("snapshot %q not found", req.GetSnapshot())
	}
	return snap, nil
}

func (s *SubscriberServer) ListSnapshots(ctx context.Context, req *pubsubpb.ListSnapshotsRequest) (*pubsubpb.ListSnapshotsResponse, error) {
	if !ValidProjectParent(req.GetProject()) {
		return nil, invalidArg("project must be of the form projects/{project}")
	}
	pageSize := pagination.EffectivePageSize(req.GetPageSize(), 100)
	snaps, next, code := s.state.ListSnapshots(req.GetProject(), pageSize, req.GetPageToken())
	if code != "" {
		return nil, stateErr(code)
	}
	return &pubsubpb.ListSnapshotsResponse{Snapshots: snaps, NextPageToken: next}, nil
}

func (s *SubscriberServer) UpdateSnapshot(ctx context.Context, req *pubsubpb.UpdateSnapshotRequest) (*pubsubpb.Snapshot, error) {
	if req.GetSnapshot() == nil || req.GetSnapshot().GetName() == "" {
		return nil, invalidArg("snapshot.name is required")
	}
	var paths []string
	if req.GetUpdateMask() != nil {
		paths = req.GetUpdateMask().GetPaths()
	}
	snap, code := s.state.UpdateSnapshot(req.GetSnapshot().GetName(), req.GetSnapshot(), paths)
	if code != "" {
		return nil, stateErr(code)
	}
	return snap, nil
}

func (s *SubscriberServer) DeleteSnapshot(ctx context.Context, req *pubsubpb.DeleteSnapshotRequest) (*emptypb.Empty, error) {
	if code := s.state.DeleteSnapshot(req.GetSnapshot()); code != "" {
		return nil, stateErr(code)
	}
	return &emptypb.Empty{}, nil
}

func (s *SubscriberServer) Seek(ctx context.Context, req *pubsubpb.SeekRequest) (*pubsubpb.SeekResponse, error) {
	if req.GetSubscription() == "" {
		return nil, invalidArg("subscription is required")
	}
	var code string
	switch target := req.GetTarget().(type) {
	case *pubsubpb.SeekRequest_Time:
		if target.Time == nil {
			return nil, invalidArg("seek time is required")
		}
		code = s.state.SeekToTime(req.GetSubscription(), target.Time.AsTime())
	case *pubsubpb.SeekRequest_Snapshot:
		if target.Snapshot == "" {
			return nil, invalidArg("seek snapshot is required")
		}
		code = s.state.SeekToSnapshot(req.GetSubscription(), target.Snapshot)
	default:
		return nil, invalidArg("seek target is required")
	}
	if code != "" {
		return nil, stateErr(code)
	}
	return &pubsubpb.SeekResponse{}, nil
}
