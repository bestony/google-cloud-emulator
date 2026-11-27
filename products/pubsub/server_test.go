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
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

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

func TestMessageDataPlane(t *testing.T) {
	pub, sub, _ := newTestServers(t)
	ctx := context.Background()
	topicName := "projects/demo/topics/data-topic"
	subName := "projects/demo/subscriptions/data-sub"
	if _, err := pub.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName}); err != nil {
		t.Fatal(err)
	}
	if _, err := sub.CreateSubscription(ctx, &pubsubpb.Subscription{Name: subName, Topic: topicName, AckDeadlineSeconds: 30}); err != nil {
		t.Fatal(err)
	}

	published, err := pub.Publish(ctx, &pubsubpb.PublishRequest{
		Topic: topicName,
		Messages: []*pubsubpb.PubsubMessage{
			{Data: []byte("one"), Attributes: map[string]string{"kind": "kept"}},
			{Data: []byte("two"), Attributes: map[string]string{"kind": "dropped"}},
		},
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if len(published.MessageIds) != 2 || published.MessageIds[0] == published.MessageIds[1] {
		t.Fatalf("bad message ids: %v", published.MessageIds)
	}

	pulled, err := sub.Pull(ctx, &pubsubpb.PullRequest{Subscription: subName, MaxMessages: 10})
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if len(pulled.ReceivedMessages) != 2 {
		t.Fatalf("expected 2 messages, got %+v", pulled.ReceivedMessages)
	}
	first := pulled.ReceivedMessages[0]
	if string(first.Message.Data) != "one" || first.Message.MessageId != published.MessageIds[0] || first.Message.PublishTime == nil {
		t.Fatalf("bad first message: %+v", first.Message)
	}
	if first.AckId == "" || pulled.ReceivedMessages[1].AckId == "" {
		t.Fatalf("missing ack ids: %+v", pulled.ReceivedMessages)
	}

	// Outstanding messages are not redelivered until their deadline expires or
	// they are nacked with a zero ModifyAckDeadline.
	again, err := sub.Pull(ctx, &pubsubpb.PullRequest{Subscription: subName, MaxMessages: 10})
	if err != nil || len(again.ReceivedMessages) != 0 {
		t.Fatalf("outstanding messages redelivered early: %v %+v", err, again)
	}
	if _, err := sub.ModifyAckDeadline(ctx, &pubsubpb.ModifyAckDeadlineRequest{
		Subscription:       subName,
		AckIds:             []string{first.AckId},
		AckDeadlineSeconds: 0,
	}); err != nil {
		t.Fatalf("ModifyAckDeadline nack: %v", err)
	}
	redelivered, err := sub.Pull(ctx, &pubsubpb.PullRequest{Subscription: subName, MaxMessages: 10})
	if err != nil || len(redelivered.ReceivedMessages) != 1 {
		t.Fatalf("redelivery after nack: %v %+v", err, redelivered)
	}
	if redelivered.ReceivedMessages[0].Message.MessageId != first.Message.MessageId {
		t.Fatalf("redelivered wrong message: %+v", redelivered.ReceivedMessages[0])
	}

	// Acking both deliveries removes them from the pull backlog.
	ackIDs := []string{redelivered.ReceivedMessages[0].AckId, pulled.ReceivedMessages[1].AckId}
	if _, err := sub.Acknowledge(ctx, &pubsubpb.AcknowledgeRequest{Subscription: subName, AckIds: ackIDs}); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}
	empty, err := sub.Pull(ctx, &pubsubpb.PullRequest{Subscription: subName, MaxMessages: 10})
	if err != nil || len(empty.ReceivedMessages) != 0 {
		t.Fatalf("acked messages redelivered: %v %+v", err, empty)
	}

	// Attribute filters select only matching messages for a subscription.
	filteredName := "projects/demo/subscriptions/filtered-sub"
	if _, err := sub.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name:   filteredName,
		Topic:  topicName,
		Filter: `attributes.kind = "kept"`,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pub.Publish(ctx, &pubsubpb.PublishRequest{
		Topic: topicName,
		Messages: []*pubsubpb.PubsubMessage{
			{Data: []byte("kept"), Attributes: map[string]string{"kind": "kept"}},
			{Data: []byte("dropped"), Attributes: map[string]string{"kind": "dropped"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	filtered, err := sub.Pull(ctx, &pubsubpb.PullRequest{Subscription: filteredName, MaxMessages: 10})
	if err != nil || len(filtered.ReceivedMessages) != 1 || string(filtered.ReceivedMessages[0].Message.Data) != "kept" {
		t.Fatalf("filtered pull: %v %+v", err, filtered)
	}
}

func TestSnapshotsAndSeek(t *testing.T) {
	pub, sub, _ := newTestServers(t)
	ctx := context.Background()
	topicName := "projects/demo/topics/snap-topic"
	subName := "projects/demo/subscriptions/snap-sub"
	if _, err := pub.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName}); err != nil {
		t.Fatal(err)
	}
	if _, err := sub.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name:                subName,
		Topic:               topicName,
		RetainAckedMessages: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pub.Publish(ctx, &pubsubpb.PublishRequest{
		Topic:    topicName,
		Messages: []*pubsubpb.PubsubMessage{{Data: []byte("one")}, {Data: []byte("two")}},
	}); err != nil {
		t.Fatal(err)
	}
	pulled, err := sub.Pull(ctx, &pubsubpb.PullRequest{Subscription: subName, MaxMessages: 2})
	if err != nil || len(pulled.ReceivedMessages) != 2 {
		t.Fatalf("initial pull: %v %+v", err, pulled)
	}
	firstID := pulled.ReceivedMessages[0].Message.MessageId
	secondID := pulled.ReceivedMessages[1].Message.MessageId
	if _, err := sub.Acknowledge(ctx, &pubsubpb.AcknowledgeRequest{Subscription: subName, AckIds: []string{pulled.ReceivedMessages[0].AckId}}); err != nil {
		t.Fatal(err)
	}

	snapName := "projects/demo/snapshots/snap-one"
	snap, err := sub.CreateSnapshot(ctx, &pubsubpb.CreateSnapshotRequest{
		Name:         snapName,
		Subscription: subName,
		Labels:       map[string]string{"purpose": "test"},
	})
	if err != nil {
		t.Fatalf("CreateSnapshot: %v", err)
	}
	if snap.Topic != topicName || snap.Labels["purpose"] != "test" || snap.ExpireTime == nil {
		t.Fatalf("bad snapshot: %+v", snap)
	}
	if got, err := sub.GetSnapshot(ctx, &pubsubpb.GetSnapshotRequest{Snapshot: snapName}); err != nil || got.Name != snapName {
		t.Fatalf("GetSnapshot: %v %+v", err, got)
	}
	listed, err := sub.ListSnapshots(ctx, &pubsubpb.ListSnapshotsRequest{Project: "projects/demo"})
	if err != nil || len(listed.Snapshots) != 1 {
		t.Fatalf("ListSnapshots: %v %+v", err, listed)
	}

	// Acknowledge the second message after the snapshot. Seeking back restores
	// it as unacknowledged while the first remains acknowledged.
	if _, err := sub.Acknowledge(ctx, &pubsubpb.AcknowledgeRequest{Subscription: subName, AckIds: []string{pulled.ReceivedMessages[1].AckId}}); err != nil {
		t.Fatal(err)
	}
	if _, err := sub.Seek(ctx, &pubsubpb.SeekRequest{Subscription: subName, Target: &pubsubpb.SeekRequest_Snapshot{Snapshot: snapName}}); err != nil {
		t.Fatalf("Seek snapshot: %v", err)
	}
	replayed, err := sub.Pull(ctx, &pubsubpb.PullRequest{Subscription: subName, MaxMessages: 10})
	if err != nil || len(replayed.ReceivedMessages) != 1 {
		t.Fatalf("snapshot replay: %v %+v", err, replayed)
	}
	if replayed.ReceivedMessages[0].Message.MessageId != secondID {
		t.Fatalf("snapshot replayed wrong message: got %q want %q", replayed.ReceivedMessages[0].Message.MessageId, secondID)
	}

	// Seeking to the beginning of time marks every retained message unacked.
	if _, err := sub.Seek(ctx, &pubsubpb.SeekRequest{
		Subscription: subName,
		Target:       &pubsubpb.SeekRequest_Time{Time: timestamppb.New(time.Unix(0, 0))},
	}); err != nil {
		t.Fatalf("Seek time: %v", err)
	}
	replayed, err = sub.Pull(ctx, &pubsubpb.PullRequest{Subscription: subName, MaxMessages: 10})
	if err != nil || len(replayed.ReceivedMessages) != 2 {
		t.Fatalf("time replay: %v %+v", err, replayed)
	}
	ids := map[string]bool{}
	for _, got := range replayed.ReceivedMessages {
		ids[got.Message.MessageId] = true
	}
	if !ids[firstID] || !ids[secondID] {
		t.Fatalf("time replay ids: %v", ids)
	}

	if _, err := sub.DeleteSnapshot(ctx, &pubsubpb.DeleteSnapshotRequest{Snapshot: snapName}); err != nil {
		t.Fatal(err)
	}
	if _, err := sub.GetSnapshot(ctx, &pubsubpb.GetSnapshotRequest{Snapshot: snapName}); status.Code(err) != codes.NotFound {
		t.Fatalf("deleted snapshot: got %v", err)
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

	// StreamingPull delivers published messages and accepts acknowledgements on
	// the same bidirectional stream.
	if _, err := pclient.Publish(context.Background(), &pubsubpb.PublishRequest{
		Topic:    "projects/demo/topics/grpc-topic",
		Messages: []*pubsubpb.PubsubMessage{{Data: []byte("streamed")}},
	}); err != nil {
		t.Fatalf("gRPC Publish: %v", err)
	}
	streamCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stream, err := sclient.StreamingPull(streamCtx)
	if err != nil {
		t.Fatalf("StreamingPull open: %v", err)
	}
	if err := stream.Send(&pubsubpb.StreamingPullRequest{
		Subscription:             "projects/demo/subscriptions/grpc-sub",
		StreamAckDeadlineSeconds: 10,
	}); err != nil {
		t.Fatalf("StreamingPull init: %v", err)
	}
	received, err := stream.Recv()
	if err != nil {
		t.Fatalf("StreamingPull recv: %v", err)
	}
	if len(received.ReceivedMessages) != 1 || string(received.ReceivedMessages[0].Message.Data) != "streamed" {
		t.Fatalf("StreamingPull messages: %+v", received)
	}
	if err := stream.Send(&pubsubpb.StreamingPullRequest{AckIds: []string{received.ReceivedMessages[0].AckId}}); err != nil {
		t.Fatalf("StreamingPull ack: %v", err)
	}
	_ = stream.CloseSend()
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

	// Publish, pull and acknowledge exercise the REST data-plane gateway.
	resp4, err := http.Post(ts.URL+"/v1/projects/demo/topics/rest-topic:publish", "application/json", strings.NewReader(`{"messages":[{"data":"aGk="}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp4.Body.Close()
	if resp4.StatusCode != 200 {
		raw, _ := io.ReadAll(resp4.Body)
		t.Fatalf("publish status = %d body=%s", resp4.StatusCode, raw)
	}
	var publishedREST map[string]any
	if err := json.NewDecoder(resp4.Body).Decode(&publishedREST); err != nil {
		t.Fatal(err)
	}
	if ids, _ := publishedREST["messageIds"].([]any); len(ids) != 1 {
		t.Fatalf("bad publish response: %v", publishedREST)
	}

	resp5, err := http.Post(ts.URL+"/v1/projects/demo/subscriptions/rest-sub:pull", "application/json", strings.NewReader(`{"maxMessages":10}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp5.Body.Close()
	if resp5.StatusCode != 200 {
		raw, _ := io.ReadAll(resp5.Body)
		t.Fatalf("pull status = %d body=%s", resp5.StatusCode, raw)
	}
	var pullREST map[string]any
	if err := json.NewDecoder(resp5.Body).Decode(&pullREST); err != nil {
		t.Fatal(err)
	}
	received, _ := pullREST["receivedMessages"].([]any)
	if len(received) != 1 {
		t.Fatalf("bad pull response: %v", pullREST)
	}
	ackID, _ := received[0].(map[string]any)["ackId"].(string)
	if ackID == "" {
		t.Fatalf("missing REST ack id: %v", received[0])
	}

	resp6, err := http.Post(ts.URL+"/v1/projects/demo/subscriptions/rest-sub:acknowledge", "application/json", strings.NewReader(`{"ackIds":["`+ackID+`"]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp6.Body.Close()
	if resp6.StatusCode != 200 {
		raw, _ := io.ReadAll(resp6.Body)
		t.Fatalf("ack status = %d body=%s", resp6.StatusCode, raw)
	}
}
