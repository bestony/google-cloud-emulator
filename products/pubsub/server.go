// Package pubsub emulates google.pubsub.v1 over gRPC and REST (via
// grpc-gateway). State is in-memory only.
//
// M1 scope: topic and subscription control-plane CRUD. The data plane
// (Publish/Pull/Acknowledge/StreamingPull/ModifyAckDeadline/Seek/snapshots)
// returns UNIMPLEMENTED.
package pubsub

import (
	"context"
	"strings"

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
