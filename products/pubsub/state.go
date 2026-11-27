package pubsub

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bestony/google-cloud-emulator/core/pagination"
	pubsubpb "github.com/bestony/google-cloud-emulator/gen/google/pubsub/v1"
)

var (
	projectPrefixRe = regexp.MustCompile(`^projects/[^/]+$`)
	topicIDRe       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._~+%-]{2,254}$`)
	subIDRe         = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._~+%-]{2,254}$`)
)

// State holds all Pub/Sub resources and message backlogs in memory.
type State struct {
	mu            sync.RWMutex
	topics        map[string]*pubsubpb.Topic
	subscriptions map[string]*pubsubpb.Subscription
	backlogs      map[string][]*subscriptionMessage
	ackIndex      map[string]map[string]*subscriptionMessage
	snapshots     map[string]*snapshotData
	nextMessageID int64
	nextAckID     int64
	nextSequence  int64
}

// NewState returns an empty State.
func NewState() *State {
	return &State{
		topics:        make(map[string]*pubsubpb.Topic),
		subscriptions: make(map[string]*pubsubpb.Subscription),
		backlogs:      make(map[string][]*subscriptionMessage),
		ackIndex:      make(map[string]map[string]*subscriptionMessage),
		snapshots:     make(map[string]*snapshotData),
		nextMessageID: 1,
		nextAckID:     1,
	}
}

// TopicResourceName builds projects/{p}/topics/{t}.
func TopicResourceName(parent, topicID string) string {
	return strings.TrimSuffix(parent, "/") + "/topics/" + topicID
}

// SubscriptionResourceName builds projects/{p}/subscriptions/{s}.
func SubscriptionResourceName(parent, subID string) string {
	return strings.TrimSuffix(parent, "/") + "/subscriptions/" + subID
}

// ParseTopicName splits projects/{p}/topics/{id} into project and id.
func ParseTopicName(name string) (project, id string, ok bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[1] == "" || parts[2] != "topics" || parts[3] == "" {
		return "", "", false
	}
	return parts[1], parts[3], true
}

// ParseSubscriptionName splits projects/{p}/subscriptions/{id}.
func ParseSubscriptionName(name string) (project, id string, ok bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[1] == "" || parts[2] != "subscriptions" || parts[3] == "" {
		return "", "", false
	}
	return parts[1], parts[3], true
}

// ValidTopicID reports whether id is a valid Pub/Sub topic id.
func ValidTopicID(id string) bool {
	if !topicIDRe.MatchString(id) {
		return false
	}
	return !strings.HasPrefix(strings.ToLower(id), "goog")
}

// ValidSubscriptionID reports whether id is a valid Pub/Sub subscription id.
func ValidSubscriptionID(id string) bool {
	if !subIDRe.MatchString(id) {
		return false
	}
	return !strings.HasPrefix(strings.ToLower(id), "goog")
}

// ValidProjectParent reports whether parent is projects/{project}.
func ValidProjectParent(parent string) bool {
	return projectPrefixRe.MatchString(parent)
}

func validResourceName(name string, kind string) bool {
	parts := strings.Split(name, "/")
	// projects/{p}/{kind}/{id}
	return len(parts) == 4 && parts[0] == "projects" && parts[1] != "" && parts[2] == kind && parts[3] != ""
}

// CreateTopic creates a topic named topic.Name (projects/{p}/topics/{id}).
// code is "" or "already-exists"/"invalid-name".
func (s *State) CreateTopic(topic *pubsubpb.Topic) (*pubsubpb.Topic, string) {
	project, id, ok := ParseTopicName(topic.GetName())
	if !ok || project == "" || id == "" {
		return nil, "invalid-name"
	}
	if !ValidTopicID(id) {
		return nil, "invalid-name"
	}
	name := topic.GetName()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.topics[name]; ok {
		return nil, "already-exists"
	}
	cp := proto.Clone(topic).(*pubsubpb.Topic)
	cp.State = pubsubpb.Topic_ACTIVE
	s.topics[name] = cp
	return proto.Clone(cp).(*pubsubpb.Topic), ""
}

// GetTopic returns a copy of the topic or nil.
func (s *State) GetTopic(name string) *pubsubpb.Topic {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.topics[name]
	if !ok {
		return nil
	}
	return proto.Clone(t).(*pubsubpb.Topic)
}

// UpdateTopic applies whitelisted field-mask paths to an existing topic.
// code is "" or "not-found"/"invalid-name"/"invalid-mask".
func (s *State) UpdateTopic(name string, topic *pubsubpb.Topic, maskPaths []string) (*pubsubpb.Topic, string) {
	if !validResourceName(name, "topics") {
		return nil, "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.topics[name]
	if !ok {
		return nil, "not-found"
	}
	for _, p := range maskPaths {
		switch p {
		case "labels":
			t.Labels = cloneStringMap(topic.Labels)
		case "message_storage_policy":
			t.MessageStoragePolicy = cloneMessageStoragePolicy(topic.MessageStoragePolicy)
		case "kms_key_name":
			t.KmsKeyName = topic.KmsKeyName
		case "message_retention_duration":
			t.MessageRetentionDuration = cloneDuration(topic.MessageRetentionDuration)
		case "schema_settings":
			t.SchemaSettings = cloneSchemaSettings(topic.SchemaSettings)
		default:
			return nil, fmt.Sprintf("invalid-mask:%s", p)
		}
	}
	return proto.Clone(t).(*pubsubpb.Topic), ""
}

// ListTopics returns a sorted page of topic names under parent.
func (s *State) ListTopics(parent string, pageSize int, pageToken string) ([]*pubsubpb.Topic, string, int, string) {
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", 0, "invalid-page-token"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var names []string
	for name := range s.topics {
		if strings.HasPrefix(name, parent+"/topics/") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	total := len(names)
	if offset >= total {
		return []*pubsubpb.Topic{}, "", total, ""
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	page := names[offset:end]
	out := make([]*pubsubpb.Topic, 0, len(page))
	for _, n := range page {
		out = append(out, proto.Clone(s.topics[n]).(*pubsubpb.Topic))
	}
	next := ""
	if end < total {
		next = pagination.Encode(end)
	}
	return out, next, total, ""
}

// DeleteTopic removes a topic and marks attached subscriptions' topic as _deleted-topic_.
func (s *State) DeleteTopic(name string) string {
	if !validResourceName(name, "topics") {
		return "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.topics[name]; !ok {
		return "not-found"
	}
	delete(s.topics, name)
	for _, sub := range s.subscriptions {
		if sub.Topic == name {
			sub.Topic = "_deleted-topic_"
		}
	}
	return ""
}

// SubscriptionNamesByTopic returns sorted subscription names attached to topic.
func (s *State) SubscriptionNamesByTopic(topic string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []string
	for name, sub := range s.subscriptions {
		if sub.Topic == topic {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// SubscriptionPageByTopic returns a page of subscription names attached to topic.
func (s *State) SubscriptionPageByTopic(topic string, pageSize int, pageToken string) ([]string, string, int, string) {
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", 0, "invalid-page-token"
	}
	names := s.SubscriptionNamesByTopic(topic)
	total := len(names)
	if offset >= total {
		return []string{}, "", total, ""
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	next := ""
	if end < total {
		next = pagination.Encode(end)
	}
	return names[offset:end], next, total, ""
}

// CreateSubscription creates a subscription. When sub.Name is empty a random
// name is generated on the topic's project (gRPC allows this; REST requires a
// name). code is "" or "already-exists"/"invalid-name"/"invalid-topic"/"topic-not-found".
func (s *State) CreateSubscription(sub *pubsubpb.Subscription) (*pubsubpb.Subscription, string) {
	if sub.GetTopic() == "" {
		return nil, "invalid-topic"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// The topic must exist.
	if _, ok := s.topics[sub.GetTopic()]; !ok {
		return nil, "topic-not-found"
	}
	name := sub.GetName()
	if name == "" {
		tp, _, _ := ParseTopicName(sub.GetTopic())
		if tp == "" {
			return nil, "invalid-topic"
		}
		name = SubscriptionResourceName("projects/"+tp, randomSubscriptionID())
	} else {
		_, id, ok := ParseSubscriptionName(name)
		if !ok || id == "" {
			return nil, "invalid-name"
		}
		if !ValidSubscriptionID(id) {
			return nil, "invalid-name"
		}
		// Subscription and topic must live in the same project.
		tp, _, _ := ParseTopicName(sub.GetTopic())
		sp, _, _ := ParseSubscriptionName(name)
		if tp != sp {
			return nil, "invalid-name"
		}
	}
	if _, ok := s.subscriptions[name]; ok {
		return nil, "already-exists"
	}
	cp := proto.Clone(sub).(*pubsubpb.Subscription)
	cp.Name = name
	s.subscriptions[name] = cp
	return proto.Clone(cp).(*pubsubpb.Subscription), ""
}

// GetSubscription returns a copy of the subscription or nil.
func (s *State) GetSubscription(name string) *pubsubpb.Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sub, ok := s.subscriptions[name]
	if !ok {
		return nil
	}
	return proto.Clone(sub).(*pubsubpb.Subscription)
}

// UpdateSubscription applies whitelisted field-mask paths.
func (s *State) UpdateSubscription(name string, sub *pubsubpb.Subscription, maskPaths []string) (*pubsubpb.Subscription, string) {
	if !validResourceName(name, "subscriptions") {
		return nil, "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.subscriptions[name]
	if !ok {
		return nil, "not-found"
	}
	for _, p := range maskPaths {
		switch p {
		case "labels":
			cur.Labels = cloneStringMap(sub.Labels)
		case "push_config":
			cur.PushConfig = clonePushConfig(sub.PushConfig)
		case "ack_deadline_seconds":
			cur.AckDeadlineSeconds = sub.AckDeadlineSeconds
		case "retain_acked_messages":
			cur.RetainAckedMessages = sub.RetainAckedMessages
		case "message_retention_duration":
			cur.MessageRetentionDuration = cloneDuration(sub.MessageRetentionDuration)
		case "enable_message_ordering":
			cur.EnableMessageOrdering = sub.EnableMessageOrdering
		case "expiration_policy":
			cur.ExpirationPolicy = cloneExpirationPolicy(sub.ExpirationPolicy)
		case "filter":
			cur.Filter = sub.Filter
		case "dead_letter_policy":
			cur.DeadLetterPolicy = cloneDeadLetterPolicy(sub.DeadLetterPolicy)
		case "retry_policy":
			cur.RetryPolicy = cloneRetryPolicy(sub.RetryPolicy)
		case "enable_exactly_once_delivery":
			cur.EnableExactlyOnceDelivery = sub.EnableExactlyOnceDelivery
		default:
			return nil, fmt.Sprintf("invalid-mask:%s", p)
		}
	}
	return proto.Clone(cur).(*pubsubpb.Subscription), ""
}

// ListSubscriptions returns a sorted page of subscription names under parent.
func (s *State) ListSubscriptions(parent string, pageSize int, pageToken string) ([]*pubsubpb.Subscription, string, int, string) {
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", 0, "invalid-page-token"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var names []string
	for name := range s.subscriptions {
		if strings.HasPrefix(name, parent+"/subscriptions/") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	total := len(names)
	if offset >= total {
		return []*pubsubpb.Subscription{}, "", total, ""
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	page := names[offset:end]
	out := make([]*pubsubpb.Subscription, 0, len(page))
	for _, n := range page {
		out = append(out, proto.Clone(s.subscriptions[n]).(*pubsubpb.Subscription))
	}
	next := ""
	if end < total {
		next = pagination.Encode(end)
	}
	return out, next, total, ""
}

// DeleteSubscription removes a subscription.
func (s *State) DeleteSubscription(name string) string {
	if !validResourceName(name, "subscriptions") {
		return "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subscriptions[name]; !ok {
		return "not-found"
	}
	delete(s.subscriptions, name)
	delete(s.backlogs, name)
	delete(s.ackIndex, name)
	return ""
}

// DetachSubscription marks a subscription detached.
func (s *State) DetachSubscription(name string) string {
	if !validResourceName(name, "subscriptions") {
		return "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[name]
	if !ok {
		return "not-found"
	}
	sub.Detached = true
	return ""
}

// SetPushConfig replaces the push config of a subscription.
func (s *State) SetPushConfig(name string, push *pubsubpb.PushConfig) string {
	if !validResourceName(name, "subscriptions") {
		return "invalid-name"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[name]
	if !ok {
		return "not-found"
	}
	sub.PushConfig = proto.Clone(push).(*pubsubpb.PushConfig)
	return ""
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneDuration(in *durationpb.Duration) *durationpb.Duration {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*durationpb.Duration)
}

func cloneMessageStoragePolicy(in *pubsubpb.MessageStoragePolicy) *pubsubpb.MessageStoragePolicy {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*pubsubpb.MessageStoragePolicy)
}

func cloneSchemaSettings(in *pubsubpb.SchemaSettings) *pubsubpb.SchemaSettings {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*pubsubpb.SchemaSettings)
}

func clonePushConfig(in *pubsubpb.PushConfig) *pubsubpb.PushConfig {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*pubsubpb.PushConfig)
}

func cloneExpirationPolicy(in *pubsubpb.ExpirationPolicy) *pubsubpb.ExpirationPolicy {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*pubsubpb.ExpirationPolicy)
}

func cloneDeadLetterPolicy(in *pubsubpb.DeadLetterPolicy) *pubsubpb.DeadLetterPolicy {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*pubsubpb.DeadLetterPolicy)
}

func cloneRetryPolicy(in *pubsubpb.RetryPolicy) *pubsubpb.RetryPolicy {
	if in == nil {
		return nil
	}
	return proto.Clone(in).(*pubsubpb.RetryPolicy)
}

// randomSubscriptionID returns a random id similar to the server-assigned ones.
func randomSubscriptionID() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "sub0000"
	}
	out := make([]byte, len(b))
	for i, v := range b {
		out[i] = alphabet[int(v)%len(alphabet)]
	}
	return string(out)
}
