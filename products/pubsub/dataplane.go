package pubsub

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bestony/google-cloud-emulator/core/pagination"
	pubsubpb "github.com/bestony/google-cloud-emulator/gen/google/pubsub/v1"
)

const (
	defaultAckDeadlineSeconds = int32(10)
	maxMessageBytes           = 10 * 1024 * 1024
)

// subscriptionMessage is one delivery queue entry. Acked messages are retained
// only when the subscription has retain_acked_messages enabled; unacked
// messages are retained for the subscription retention window.
type subscriptionMessage struct {
	message         *pubsubpb.PubsubMessage
	sequence        int64
	acked           bool
	outstanding     bool
	ackID           string
	deadline        time.Time
	deliveryAttempt int32
}

type snapshotData struct {
	resource     *pubsubpb.Snapshot
	subscription string
	messages     []*subscriptionMessage
}

func cloneSubscriptionMessage(in *subscriptionMessage) *subscriptionMessage {
	if in == nil {
		return nil
	}
	out := *in
	out.message = proto.Clone(in.message).(*pubsubpb.PubsubMessage)
	return &out
}

// ParseSnapshotName splits projects/{p}/snapshots/{id}.
func ParseSnapshotName(name string) (project, id string, ok bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[1] == "" || parts[2] != "snapshots" || parts[3] == "" {
		return "", "", false
	}
	return parts[1], parts[3], true
}

var (
	attributeExistsRe = regexp.MustCompile(`^attributes:([A-Za-z0-9_-]+)$`)
	attributeEqualsRe = regexp.MustCompile(`^attributes\.([A-Za-z0-9_-]+)\s*=\s*"([^"]*)"$`)
	attributePrefixRe = regexp.MustCompile(`^hasPrefix\(attributes\.([A-Za-z0-9_-]+),\s*"([^"]*)"\)$`)
)

// subscriptionFilterMatches implements the commonly used subset of Pub/Sub's
// subscription filter language. Unknown expressions are accepted permissively;
// control-plane validation of the full CEL dialect is outside the emulator's
// scope, but the common attribute equality/existence/prefix forms behave like
// the real service.
func subscriptionFilterMatches(filter string, msg *pubsubpb.PubsubMessage) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	if m := attributeExistsRe.FindStringSubmatch(filter); m != nil {
		_, ok := msg.GetAttributes()[m[1]]
		return ok
	}
	if m := attributeEqualsRe.FindStringSubmatch(filter); m != nil {
		return msg.GetAttributes()[m[1]] == m[2]
	}
	if m := attributePrefixRe.FindStringSubmatch(filter); m != nil {
		return strings.HasPrefix(msg.GetAttributes()[m[1]], m[2])
	}
	return true
}

func (s *State) appendMessageLocked(subscriptionName string, msg *pubsubpb.PubsubMessage) {
	s.nextSequence++
	entry := &subscriptionMessage{
		message:  proto.Clone(msg).(*pubsubpb.PubsubMessage),
		sequence: s.nextSequence,
	}
	s.backlogs[subscriptionName] = append(s.backlogs[subscriptionName], entry)
}

func (s *State) fanoutLocked(topic string, msg *pubsubpb.PubsubMessage) {
	names := make([]string, 0, len(s.subscriptions))
	for name := range s.subscriptions {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		sub := s.subscriptions[name]
		if sub.GetTopic() != topic || sub.GetDetached() {
			continue
		}
		if !subscriptionFilterMatches(sub.GetFilter(), msg) {
			continue
		}
		s.appendMessageLocked(name, msg)
	}
}

// PublishMessages validates and publishes messages to a topic, fanning each
// message out to every attached subscription. The returned IDs are in request
// order.
func (s *State) PublishMessages(topic string, messages []*pubsubpb.PubsubMessage) ([]string, string) {
	if !validResourceName(topic, "topics") {
		return nil, "invalid-name"
	}
	if len(messages) == 0 {
		return nil, "empty-messages"
	}
	if len(messages) > 1000 {
		return nil, "too-many-messages"
	}
	for _, msg := range messages {
		if msg == nil || (len(msg.GetData()) == 0 && len(msg.GetAttributes()) == 0) {
			return nil, "empty-message"
		}
		if len(msg.GetData()) > maxMessageBytes {
			return nil, "message-too-large"
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.topics[topic]; !ok {
		return nil, "not-found"
	}
	ids := make([]string, 0, len(messages))
	for _, input := range messages {
		msg := proto.Clone(input).(*pubsubpb.PubsubMessage)
		msg.MessageId = fmt.Sprint(s.nextMessageID)
		msg.PublishTime = timestamppb.Now()
		s.nextMessageID++
		ids = append(ids, msg.GetMessageId())
		s.fanoutLocked(topic, msg)
	}
	return ids, ""
}

func ackDeadlineFor(sub *pubsubpb.Subscription, override int32) time.Duration {
	seconds := override
	if seconds <= 0 {
		seconds = sub.GetAckDeadlineSeconds()
	}
	if seconds <= 0 {
		seconds = defaultAckDeadlineSeconds
	}
	return time.Duration(seconds) * time.Second
}

func (s *State) rebuildAckIndexLocked(subscription string) {
	index := map[string]*subscriptionMessage{}
	for _, entry := range s.backlogs[subscription] {
		if entry.outstanding && !entry.acked && entry.ackID != "" {
			index[entry.ackID] = entry
		}
	}
	s.ackIndex[subscription] = index
}

func subscriptionRetention(sub *pubsubpb.Subscription) time.Duration {
	if d := sub.GetMessageRetentionDuration(); d != nil {
		if err := d.CheckValid(); err == nil && d.AsDuration() > 0 {
			return d.AsDuration()
		}
	}
	return 7 * 24 * time.Hour
}

func (s *State) compactBacklogLocked(subscription string, sub *pubsubpb.Subscription) {
	retention := subscriptionRetention(sub)
	cutoff := time.Now().Add(-retention)
	kept := s.backlogs[subscription][:0]
	for _, entry := range s.backlogs[subscription] {
		if entry.message.GetPublishTime().AsTime().Before(cutoff) {
			continue
		}
		if entry.acked && !sub.GetRetainAckedMessages() {
			continue
		}
		kept = append(kept, entry)
	}
	for i := len(kept); i < len(s.backlogs[subscription]); i++ {
		s.backlogs[subscription][i] = nil
	}
	s.backlogs[subscription] = kept
	s.rebuildAckIndexLocked(subscription)
}

// PullMessages returns up to maxMessages available messages and marks them
// outstanding until their ack deadline. Expired outstanding messages become
// eligible for redelivery.
func (s *State) PullMessages(subscription string, maxMessages, streamAckDeadlineSeconds int32) ([]*pubsubpb.ReceivedMessage, string) {
	if !validResourceName(subscription, "subscriptions") {
		return nil, "invalid-name"
	}
	if maxMessages <= 0 {
		return nil, "invalid-max-messages"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[subscription]
	if !ok {
		return nil, "not-found"
	}
	s.compactBacklogLocked(subscription, sub)
	deadline := ackDeadlineFor(sub, streamAckDeadlineSeconds)
	now := time.Now()
	out := []*pubsubpb.ReceivedMessage{}
	for _, entry := range s.backlogs[subscription] {
		if len(out) >= int(maxMessages) {
			break
		}
		if entry.acked {
			continue
		}
		if entry.outstanding && entry.deadline.After(now) {
			continue
		}
		if entry.ackID != "" {
			delete(s.ackIndex[subscription], entry.ackID)
		}
		entry.outstanding = true
		entry.deadline = now.Add(deadline)
		entry.ackID = fmt.Sprintf("ack-%d", s.nextAckID)
		s.nextAckID++
		entry.deliveryAttempt++
		if s.ackIndex[subscription] == nil {
			s.ackIndex[subscription] = map[string]*subscriptionMessage{}
		}
		s.ackIndex[subscription][entry.ackID] = entry
		attempt := entry.deliveryAttempt
		if sub.GetDeadLetterPolicy() == nil {
			attempt = 0
		}
		out = append(out, &pubsubpb.ReceivedMessage{
			AckId:           entry.ackID,
			Message:         proto.Clone(entry.message).(*pubsubpb.PubsubMessage),
			DeliveryAttempt: attempt,
		})
	}
	return out, ""
}

func (s *State) removeMessageLocked(subscription string, target *subscriptionMessage) {
	backlog := s.backlogs[subscription]
	for i, entry := range backlog {
		if entry == target {
			copy(backlog[i:], backlog[i+1:])
			backlog[len(backlog)-1] = nil
			s.backlogs[subscription] = backlog[:len(backlog)-1]
			return
		}
	}
}

// Acknowledge acknowledges messages by ack ID. Unknown or expired IDs are
// ignored, matching Pub/Sub's idempotent acknowledge behavior.
func (s *State) Acknowledge(subscription string, ackIDs []string) string {
	if !validResourceName(subscription, "subscriptions") {
		return "invalid-name"
	}
	if len(ackIDs) == 0 {
		return "empty-ack-ids"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[subscription]
	if !ok {
		return "not-found"
	}
	for _, id := range ackIDs {
		entry := s.ackIndex[subscription][id]
		if entry == nil {
			continue
		}
		entry.acked = true
		entry.outstanding = false
		entry.deadline = time.Time{}
		delete(s.ackIndex[subscription], id)
		if !sub.GetRetainAckedMessages() {
			s.removeMessageLocked(subscription, entry)
		}
	}
	return ""
}

// ModifyAckDeadline changes the deadline for outstanding messages. A zero
// deadline is a NACK and makes the message immediately eligible for redelivery.
func (s *State) ModifyAckDeadline(subscription string, ackIDs []string, seconds int32) string {
	if !validResourceName(subscription, "subscriptions") {
		return "invalid-name"
	}
	if len(ackIDs) == 0 {
		return "empty-ack-ids"
	}
	if seconds < 0 || seconds > 600 {
		return "invalid-deadline"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subscriptions[subscription]; !ok {
		return "not-found"
	}
	for _, id := range ackIDs {
		entry := s.ackIndex[subscription][id]
		if entry == nil {
			continue
		}
		if seconds == 0 {
			entry.outstanding = false
			entry.deadline = time.Time{}
			delete(s.ackIndex[subscription], id)
			continue
		}
		entry.outstanding = true
		entry.deadline = time.Now().Add(time.Duration(seconds) * time.Second)
	}
	return ""
}

// CreateSnapshot captures the subscription backlog and its acknowledgement
// state. Messages are cloned so later subscription mutations do not rewrite
// the snapshot.
func (s *State) CreateSnapshot(name, subscription string, labels map[string]string) (*pubsubpb.Snapshot, string) {
	if !validResourceName(subscription, "subscriptions") {
		return nil, "invalid-subscription"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[subscription]
	if !ok {
		return nil, "not-found"
	}
	if name == "" {
		project, _, _ := ParseSubscriptionName(subscription)
		name = "projects/" + project + "/snapshots/" + randomSubscriptionID()
	}
	project, id, valid := ParseSnapshotName(name)
	if !valid || !ValidSubscriptionID(id) {
		return nil, "invalid-name"
	}
	subProject, _, _ := ParseSubscriptionName(subscription)
	if project != subProject {
		return nil, "invalid-name"
	}
	if _, exists := s.snapshots[name]; exists {
		return nil, "already-exists"
	}
	messages := make([]*subscriptionMessage, 0, len(s.backlogs[subscription]))
	for _, entry := range s.backlogs[subscription] {
		messages = append(messages, cloneSubscriptionMessage(entry))
	}
	resource := &pubsubpb.Snapshot{
		Name:       name,
		Topic:      sub.GetTopic(),
		ExpireTime: timestamppb.New(time.Now().Add(7 * 24 * time.Hour)),
		Labels:     cloneStringMap(labels),
	}
	s.snapshots[name] = &snapshotData{resource: proto.Clone(resource).(*pubsubpb.Snapshot), subscription: subscription, messages: messages}
	return proto.Clone(resource).(*pubsubpb.Snapshot), ""
}

func (s *State) GetSnapshot(name string) *pubsubpb.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.snapshots[name]
	if !ok {
		return nil
	}
	return proto.Clone(snap.resource).(*pubsubpb.Snapshot)
}

func (s *State) ListSnapshots(parent string, pageSize int, pageToken string) ([]*pubsubpb.Snapshot, string, string) {
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", "invalid-page-token"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := []string{}
	for name := range s.snapshots {
		if strings.HasPrefix(name, parent+"/snapshots/") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if offset >= len(names) {
		return []*pubsubpb.Snapshot{}, "", ""
	}
	end := offset + pageSize
	if end > len(names) {
		end = len(names)
	}
	out := make([]*pubsubpb.Snapshot, 0, end-offset)
	for _, name := range names[offset:end] {
		out = append(out, proto.Clone(s.snapshots[name].resource).(*pubsubpb.Snapshot))
	}
	next := ""
	if end < len(names) {
		next = pagination.Encode(end)
	}
	return out, next, ""
}

func (s *State) UpdateSnapshot(name string, update *pubsubpb.Snapshot, paths []string) (*pubsubpb.Snapshot, string) {
	if !validResourceName(name, "snapshots") {
		return nil, "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, ok := s.snapshots[name]
	if !ok {
		return nil, "not-found"
	}
	for _, path := range paths {
		switch path {
		case "labels":
			snap.resource.Labels = cloneStringMap(update.GetLabels())
		default:
			return nil, fmt.Sprintf("invalid-mask:%s", path)
		}
	}
	return proto.Clone(snap.resource).(*pubsubpb.Snapshot), ""
}

func (s *State) DeleteSnapshot(name string) string {
	if !validResourceName(name, "snapshots") {
		return "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.snapshots[name]; !ok {
		return "not-found"
	}
	delete(s.snapshots, name)
	return ""
}

// SeekToSnapshot restores the backlog captured by snapshot. The snapshot must
// belong to the subscription in this emulator.
func (s *State) SeekToSnapshot(subscription, snapshot string) string {
	if !validResourceName(subscription, "subscriptions") || !validResourceName(snapshot, "snapshots") {
		return "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subscriptions[subscription]; !ok {
		return "not-found"
	}
	snap, ok := s.snapshots[snapshot]
	if !ok {
		return "snapshot-not-found"
	}
	if snap.subscription != subscription {
		return "snapshot-subscription-mismatch"
	}
	messages := make([]*subscriptionMessage, 0, len(snap.messages))
	for _, entry := range snap.messages {
		cp := cloneSubscriptionMessage(entry)
		if !cp.acked {
			cp.outstanding = false
			cp.deadline = time.Time{}
			cp.ackID = ""
		}
		messages = append(messages, cp)
	}
	s.backlogs[subscription] = messages
	s.rebuildAckIndexLocked(subscription)
	return ""
}

// SeekToTime marks retained messages published before time as acknowledged and
// messages published at or after time as unacknowledged.
func (s *State) SeekToTime(subscription string, target time.Time) string {
	if !validResourceName(subscription, "subscriptions") {
		return "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[subscription]
	if !ok {
		return "not-found"
	}
	s.compactBacklogLocked(subscription, sub)
	for _, entry := range s.backlogs[subscription] {
		if entry.message.GetPublishTime().AsTime().Before(target) {
			entry.acked = true
			entry.outstanding = false
			entry.deadline = time.Time{}
		} else {
			entry.acked = false
			entry.outstanding = false
			entry.deadline = time.Time{}
		}
		if entry.ackID != "" {
			delete(s.ackIndex[subscription], entry.ackID)
			entry.ackID = ""
		}
	}
	s.rebuildAckIndexLocked(subscription)
	return ""
}
