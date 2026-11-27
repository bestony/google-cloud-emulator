package pubsub

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pubsubpb "github.com/bestony/google-cloud-emulator/gen/google/pubsub/v1"
)

func newTestServers(t *testing.T) (*PublisherServer, *SubscriberServer, *State) {
	t.Helper()
	st := NewState()
	return NewPublisherServer(st), NewSubscriberServer(st), st
}

func TestTopicLifecycle(t *testing.T) {
	pub, _, _ := newTestServers(t)
	ctx := context.Background()

	topic, err := pub.CreateTopic(ctx, &pubsubpb.Topic{
		Name:   "projects/demo/topics/orders",
		Labels: map[string]string{"env": "dev"},
	})
	if err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	if topic.Name != "projects/demo/topics/orders" || topic.State != pubsubpb.Topic_ACTIVE {
		t.Fatalf("unexpected topic: %+v", topic)
	}

	// Duplicate -> AlreadyExists.
	if _, err := pub.CreateTopic(ctx, &pubsubpb.Topic{Name: "projects/demo/topics/orders"}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate create: got %v", err)
	}
	// Invalid name -> InvalidArgument.
	if _, err := pub.CreateTopic(ctx, &pubsubpb.Topic{Name: "projects/demo/topics/goog-reserved"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("reserved name: got %v", err)
	}

	got, err := pub.GetTopic(ctx, &pubsubpb.GetTopicRequest{Topic: "projects/demo/topics/orders"})
	if err != nil || got.Labels["env"] != "dev" {
		t.Fatalf("GetTopic: %v %+v", err, got)
	}
	if _, err := pub.GetTopic(ctx, &pubsubpb.GetTopicRequest{Topic: "projects/demo/topics/missing"}); status.Code(err) != codes.NotFound {
		t.Fatalf("get missing: got %v", err)
	}

	// Update labels via mask.
	upd, err := pub.UpdateTopic(ctx, &pubsubpb.UpdateTopicRequest{
		Topic:      &pubsubpb.Topic{Name: "projects/demo/topics/orders", Labels: map[string]string{"env": "prod"}},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
	})
	if err != nil || upd.Labels["env"] != "prod" {
		t.Fatalf("UpdateTopic: %v %+v", err, upd)
	}
	// Unsupported mask path -> InvalidArgument.
	if _, err := pub.UpdateTopic(ctx, &pubsubpb.UpdateTopicRequest{
		Topic:      &pubsubpb.Topic{Name: "projects/demo/topics/orders"},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad mask: got %v", err)
	}

	// List with pagination.
	for _, id := range []string{"a-topic", "b-topic", "c-topic"} {
		if _, err := pub.CreateTopic(ctx, &pubsubpb.Topic{Name: "projects/demo/topics/" + id}); err != nil {
			t.Fatal(err)
		}
	}
	page1, err := pub.ListTopics(ctx, &pubsubpb.ListTopicsRequest{Project: "projects/demo", PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Topics) != 2 || page1.NextPageToken == "" {
		t.Fatalf("unexpected page1: %+v", page1)
	}
	page2, err := pub.ListTopics(ctx, &pubsubpb.ListTopicsRequest{Project: "projects/demo", PageSize: 2, PageToken: page1.NextPageToken})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Topics) != 2 || page2.NextPageToken != "" {
		t.Fatalf("unexpected page2: %+v", page2)
	}

	// ListTopicSubscriptions empty.
	lss, err := pub.ListTopicSubscriptions(ctx, &pubsubpb.ListTopicSubscriptionsRequest{Topic: "projects/demo/topics/orders"})
	if err != nil || len(lss.Subscriptions) != 0 {
		t.Fatalf("ListTopicSubscriptions: %v %+v", err, lss)
	}
}

func TestSubscriptionLifecycle(t *testing.T) {
	pub, sub, _ := newTestServers(t)
	ctx := context.Background()
	topicName := "projects/demo/topics/events"
	if _, err := pub.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName}); err != nil {
		t.Fatal(err)
	}

	// Subscription on missing topic -> NotFound.
	if _, err := sub.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name:  "projects/demo/subscriptions/early",
		Topic: "projects/demo/topics/nope",
	}); status.Code(err) != codes.NotFound {
		t.Fatalf("missing topic: got %v", err)
	}

	// gRPC-style create without name -> server assigns one.
	created, err := sub.CreateSubscription(ctx, &pubsubpb.Subscription{
		Topic:              topicName,
		AckDeadlineSeconds: 30,
	})
	if err != nil {
		t.Fatalf("CreateSubscription (auto name): %v", err)
	}
	if !strings.HasPrefix(created.Name, "projects/demo/subscriptions/") || created.Name == "projects/demo/subscriptions/" {
		t.Fatalf("auto name not assigned: %q", created.Name)
	}

	// Named create.
	named, err := sub.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name:  "projects/demo/subscriptions/push-sub",
		Topic: topicName,
		PushConfig: &pubsubpb.PushConfig{
			PushEndpoint: "https://example.com/push",
		},
	})
	if err != nil {
		t.Fatalf("CreateSubscription: %v", err)
	}
	if named.AckDeadlineSeconds != 0 {
		t.Fatalf("unexpected ack deadline %d", named.AckDeadlineSeconds)
	}
	if _, err := sub.CreateSubscription(ctx, &pubsubpb.Subscription{Name: "projects/demo/subscriptions/push-sub", Topic: topicName}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate sub: got %v", err)
	}
	// Cross-project name -> InvalidArgument.
	if _, err := sub.CreateSubscription(ctx, &pubsubpb.Subscription{Name: "projects/other/subscriptions/x", Topic: topicName}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("cross-project: got %v", err)
	}

	// ListTopicSubscriptions returns both.
	lss, err := pub.ListTopicSubscriptions(ctx, &pubsubpb.ListTopicSubscriptionsRequest{Topic: topicName})
	if err != nil || len(lss.Subscriptions) != 2 {
		t.Fatalf("ListTopicSubscriptions: %v %+v", err, lss)
	}

	// Update ack deadline + labels.
	upd, err := sub.UpdateSubscription(ctx, &pubsubpb.UpdateSubscriptionRequest{
		Subscription: &pubsubpb.Subscription{
			Name:               named.Name,
			AckDeadlineSeconds: 60,
			Labels:             map[string]string{"team": "sre"},
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"ack_deadline_seconds", "labels"}},
	})
	if err != nil || upd.AckDeadlineSeconds != 60 || upd.Labels["team"] != "sre" {
		t.Fatalf("UpdateSubscription: %v %+v", err, upd)
	}

	// ModifyPushConfig.
	if _, err := sub.ModifyPushConfig(ctx, &pubsubpb.ModifyPushConfigRequest{
		Subscription: named.Name,
		PushConfig:   &pubsubpb.PushConfig{PushEndpoint: "https://example.com/new"},
	}); err != nil {
		t.Fatalf("ModifyPushConfig: %v", err)
	}
	got, err := sub.GetSubscription(ctx, &pubsubpb.GetSubscriptionRequest{Subscription: named.Name})
	if err != nil || got.PushConfig.PushEndpoint != "https://example.com/new" {
		t.Fatalf("push endpoint not updated: %v %+v", err, got)
	}
	// Missing push endpoint -> InvalidArgument.
	if _, err := sub.ModifyPushConfig(ctx, &pubsubpb.ModifyPushConfigRequest{
		Subscription: named.Name,
		PushConfig:   &pubsubpb.PushConfig{},
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty endpoint: got %v", err)
	}

	// Detach (Publisher service).
	if _, err := pub.DetachSubscription(ctx, &pubsubpb.DetachSubscriptionRequest{Subscription: named.Name}); err != nil {
		t.Fatalf("DetachSubscription: %v", err)
	}
	got, _ = sub.GetSubscription(ctx, &pubsubpb.GetSubscriptionRequest{Subscription: named.Name})
	if !got.Detached {
		t.Fatal("subscription not detached")
	}

	// Delete topic -> attached subscriptions get _deleted-topic_.
	if _, err := pub.DeleteTopic(ctx, &pubsubpb.DeleteTopicRequest{Topic: topicName}); err != nil {
		t.Fatalf("DeleteTopic: %v", err)
	}
	got, _ = sub.GetSubscription(ctx, &pubsubpb.GetSubscriptionRequest{Subscription: named.Name})
	if got.Topic != "_deleted-topic_" {
		t.Fatalf("expected _deleted-topic_, got %q", got.Topic)
	}

	// Delete subscription.
	if _, err := sub.DeleteSubscription(ctx, &pubsubpb.DeleteSubscriptionRequest{Subscription: named.Name}); err != nil {
		t.Fatalf("DeleteSubscription: %v", err)
	}
	if _, err := sub.GetSubscription(ctx, &pubsubpb.GetSubscriptionRequest{Subscription: named.Name}); status.Code(err) != codes.NotFound {
		t.Fatalf("get deleted sub: got %v", err)
	}
}

func TestDataPlaneUnimplemented(t *testing.T) {
	pub, sub, _ := newTestServers(t)
	ctx := context.Background()
	if _, err := pub.Publish(ctx, &pubsubpb.PublishRequest{Topic: "projects/demo/topics/t"}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("Publish: got %v, want Unimplemented", err)
	}
	if _, err := sub.Pull(ctx, &pubsubpb.PullRequest{Subscription: "projects/demo/subscriptions/s"}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("Pull: got %v, want Unimplemented", err)
	}
}

func TestGRPCTransport(t *testing.T) {
	pub, sub, st := newTestServers(t)
	lis := bufconn.Listen(1024 * 1024)
	gs := grpc.NewServer()
	pubsubpb.RegisterPublisherServer(gs, pub)
	pubsubpb.RegisterSubscriberServer(gs, sub)
	go func() { _ = gs.Serve(lis) }()
	defer gs.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	pclient := pubsubpb.NewPublisherClient(conn)
	sclient := pubsubpb.NewSubscriberClient(conn)

	if _, err := pclient.CreateTopic(context.Background(), &pubsubpb.Topic{Name: "projects/demo/topics/grpc-topic"}); err != nil {
		t.Fatalf("gRPC CreateTopic: %v", err)
	}
	created, err := sclient.CreateSubscription(context.Background(), &pubsubpb.Subscription{
		Name:  "projects/demo/subscriptions/grpc-sub",
		Topic: "projects/demo/topics/grpc-topic",
	})
	if err != nil {
		t.Fatalf("gRPC CreateSubscription: %v", err)
	}
	if created.Name != "projects/demo/subscriptions/grpc-sub" {
		t.Fatalf("unexpected sub name %q", created.Name)
	}
	lt, err := pclient.ListTopics(context.Background(), &pubsubpb.ListTopicsRequest{Project: "projects/demo"})
	if err != nil || len(lt.Topics) != 1 {
		t.Fatalf("gRPC ListTopics: %v %+v", err, lt)
	}
	_ = st
}

func TestRESTGateway(t *testing.T) {
	st := NewState()
	pub := NewPublisherServer(st)
	sub := NewSubscriberServer(st)
	mux := runtime.NewServeMux()
	ctx := context.Background()
	if err := pubsubpb.RegisterPublisherHandlerServer(ctx, mux, pub); err != nil {
		t.Fatal(err)
	}
	if err := pubsubpb.RegisterSubscriberHandlerServer(ctx, mux, sub); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// PUT /v1/projects/{p}/topics/{t}
	req, _ := http.NewRequest("PUT", ts.URL+"/v1/projects/demo/topics/rest-topic", strings.NewReader(`{"labels":{"via":"rest"}}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("create topic status = %d body=%s", resp.StatusCode, raw)
	}
	var topic map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&topic); err != nil {
		t.Fatal(err)
	}
	if topic["name"] != "projects/demo/topics/rest-topic" {
		t.Fatalf("unexpected topic: %v", topic)
	}

	// Create subscription via REST (PUT full name).
	req2, _ := http.NewRequest("PUT", ts.URL+"/v1/projects/demo/subscriptions/rest-sub", strings.NewReader(`{"topic":"projects/demo/topics/rest-topic","ackDeadlineSeconds":20}`))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		raw, _ := io.ReadAll(resp2.Body)
		t.Fatalf("create sub status = %d body=%s", resp2.StatusCode, raw)
	}

	// List topics.
	resp3, err := http.Get(ts.URL + "/v1/projects/demo/topics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	var list map[string]any
	if err := json.NewDecoder(resp3.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	topics, _ := list["topics"].([]any)
	if len(topics) != 1 {
		t.Fatalf("expected 1 topic, got %v", list)
	}

	// Publish must be 501 via gateway.
	resp4, err := http.Post(ts.URL+"/v1/projects/demo/topics/rest-topic:publish", "application/json", strings.NewReader(`{"messages":[{"data":"aGk="}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp4.Body.Close()
	if resp4.StatusCode != 501 {
		raw, _ := io.ReadAll(resp4.Body)
		t.Fatalf("publish status = %d, want 501 body=%s", resp4.StatusCode, raw)
	}
}
