// Package dns emulates the Cloud DNS REST API (dns.googleapis.com/dns/v1).
// Cloud DNS has no public protobuf definition; the surface is driven by its
// Discovery document. Mutations in this API are synchronous: create returns
// the resource itself, and record edits go through Change resources.
package dns

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bestony/google-cloud-emulator/core/iam"
	"github.com/bestony/google-cloud-emulator/core/pagination"
)

// nameServers are the authoritative name servers assigned to every public
// zone, mirroring the real service's naming pattern.
var nameServers = []string{
	"ns-cloud-a1.googledomains.com",
	"ns-cloud-a2.googledomains.com",
	"ns-cloud-a3.googledomains.com",
	"ns-cloud-a4.googledomains.com",
}

var zoneNameRe = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)
var rrTypeRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)

// rrsetKey identifies an RRset within a zone.
type rrsetKey struct {
	name string // lowercased FQDN
	typ  string // uppercased type
}

// rrset is one ResourceRecordSet.
type rrset struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	TTL      int64    `json:"ttl"`
	Rrdatas  []string `json:"rrdatas"`
}

// change is one applied Change resource.
type change struct {
	ID         string   `json:"id"`
	StartTime  string   `json:"startTime"`
	Status     string   `json:"status"`
	IsServing  bool     `json:"isServing"`
	Additions  []*rrset `json:"additions,omitempty"`
	Deletions  []*rrset `json:"deletions,omitempty"`
}

// State holds zones, record sets and changes, keyed by
// "projects/{project}/managedZones/{zone}".
type State struct {
	mu        sync.RWMutex
	zones     map[string]map[string]any
	rrsets    map[string]map[rrsetKey]*rrset
	changes   map[string][]*change
	nextChangeID map[string]int64
	iam       *iam.Store
}

// NewState returns an empty State backed by the shared IAM store.
func NewState(iamStore *iam.Store) *State {
	return &State{
		zones:        make(map[string]map[string]any),
		rrsets:       make(map[string]map[rrsetKey]*rrset),
		changes:      make(map[string][]*change),
		nextChangeID: make(map[string]int64),
		iam:          iamStore,
	}
}

// IAMStore returns the shared IAM store.
func (s *State) IAMStore() *iam.Store { return s.iam }

// ZoneKey builds the internal key for a zone.
func ZoneKey(project, zone string) string {
	return "projects/" + project + "/managedZones/" + zone
}

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

func cloneZone(z map[string]any) map[string]any {
	raw, _ := json.Marshal(z)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

// GetZone returns a copy of the zone object, or nil.
func (s *State) GetZone(project, zone string) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	z, ok := s.zones[ZoneKey(project, zone)]
	if !ok {
		return nil
	}
	return cloneZone(z)
}

// ZoneExists reports whether the zone exists.
func (s *State) ZoneExists(project, zone string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.zones[ZoneKey(project, zone)]
	return ok
}

// CreateZone stores a new zone. body is the decoded request object; server
// fields (kind/id/nameServers/creationTime) are set here.
func (s *State) CreateZone(project, zone string, body map[string]any) (map[string]any, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ZoneKey(project, zone)
	if _, exists := s.zones[key]; exists {
		return nil, "already-exists"
	}
	z := map[string]any{}
	for k, v := range body {
		// Output-only fields are never taken from the client.
		switch k {
		case "kind", "id", "nameServers", "creationTime":
			continue
		}
		z[k] = v
	}
	z["kind"] = "dns#managedZone"
	z["id"] = zone
	z["name"] = zone
	ns := make([]any, len(nameServers))
	for i, n := range nameServers {
		ns[i] = n
	}
	z["nameServers"] = ns
	z["creationTime"] = nowRFC3339()
	if _, ok := z["dnsName"]; !ok {
		z["dnsName"] = ""
	}
	s.zones[key] = z
	// Seed the apex NS RRset, as the real service does.
	s.rrsets[key] = map[rrsetKey]*rrset{}
	apex := rrsetKey{name: normalizeDNSName(z["dnsName"].(string)), typ: "NS"}
	s.rrsets[key][apex] = &rrset{
		Name:    ensureTrailingDot(z["dnsName"].(string)),
		Type:    "NS",
		TTL:     21600,
		Rrdatas: append([]string(nil), nameServers...),
	}
	return cloneZone(z), ""
}

// ListZones returns one page of zones sorted by name, optionally filtered by
// exact dnsName (case-insensitive).
func (s *State) ListZones(project, dnsNameFilter string, pageSize int, pageToken string) ([]map[string]any, string, int, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := "projects/" + project + "/managedZones/"
	var names []string
	for key := range s.zones {
		if strings.HasPrefix(key, prefix) {
			names = append(names, key)
		}
	}
	sort.Strings(names)
	var filtered []map[string]any
	for _, key := range names {
		z := s.zones[key]
		if dnsNameFilter != "" {
			dn, _ := z["dnsName"].(string)
			if !dnsNameEqual(dn, dnsNameFilter) {
				continue
			}
		}
		filtered = append(filtered, cloneZone(z))
	}
	total := len(filtered)
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", 0, "invalid-page-token"
	}
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	next := pagination.PageToken(offset, pageSize, total)
	return filtered[offset:end], next, total, ""
}

// PatchZone merges the provided fields into the zone (PATCH semantics).
func (s *State) PatchZone(project, zone string, fields map[string]any) (map[string]any, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ZoneKey(project, zone)
	z, ok := s.zones[key]
	if !ok {
		return nil, "not-found"
	}
	for k, v := range fields {
		switch k {
		case "kind", "id", "name", "nameServers", "creationTime":
			continue // output-only / identity
		}
		z[k] = v
	}
	return cloneZone(z), ""
}

// DeleteZone removes a zone, its record sets and its changes. It fails when
// record sets other than NS/SOA remain.
func (s *State) DeleteZone(project, zone string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := ZoneKey(project, zone)
	if _, ok := s.zones[key]; !ok {
		return "not-found"
	}
	for k := range s.rrsets[key] {
		if k.typ != "NS" && k.typ != "SOA" {
			return "not-empty"
		}
	}
	delete(s.zones, key)
	delete(s.rrsets, key)
	delete(s.changes, key)
	delete(s.nextChangeID, key)
	return ""
}

// rrsetJSON renders an rrset for API responses.
func rrsetJSON(r *rrset) map[string]any {
	out := map[string]any{
		"kind":    "dns#resourceRecordSet",
		"name":    r.Name,
		"type":    r.Type,
		"ttl":     r.TTL,
		"rrdatas": append([]string(nil), r.Rrdatas...),
	}
	return out
}

// changeJSON renders a change for API responses.
func changeJSON(c *change) map[string]any {
	out := map[string]any{
		"kind":       "dns#change",
		"id":         c.ID,
		"startTime":  c.StartTime,
		"status":     c.Status,
		"isServing":  c.IsServing,
	}
	if len(c.Additions) > 0 {
		adds := make([]any, 0, len(c.Additions))
		for _, a := range c.Additions {
			adds = append(adds, rrsetJSON(a))
		}
		out["additions"] = adds
	}
	if len(c.Deletions) > 0 {
		dels := make([]any, 0, len(c.Deletions))
		for _, d := range c.Deletions {
			dels = append(dels, rrsetJSON(d))
		}
		out["deletions"] = dels
	}
	return out
}

// CreateChange validates and applies a change atomically, returning the
// completed Change resource.
func (s *State) CreateChange(project, zone string, additions, deletions []*rrset) (map[string]any, string) {
	if !s.ZoneExists(project, zone) {
		return nil, "zone-not-found"
	}
	key := ZoneKey(project, zone)
	// Validate before mutating.
	seenAdd := map[rrsetKey]bool{}
	seenDel := map[rrsetKey]bool{}
	var addKeys, delKeys []rrsetKey
	for _, r := range additions {
		k := rrsetKey{name: normalizeDNSName(r.Name), typ: strings.ToUpper(r.Type)}
		if seenAdd[k] || seenDel[k] {
			return nil, "conflicting-rrset"
		}
		seenAdd[k] = true
		addKeys = append(addKeys, k)
	}
	for _, r := range deletions {
		k := rrsetKey{name: normalizeDNSName(r.Name), typ: strings.ToUpper(r.Type)}
		if seenAdd[k] || seenDel[k] {
			return nil, "conflicting-rrset"
		}
		seenDel[k] = true
		delKeys = append(delKeys, k)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	sets := s.rrsets[key]
	if sets == nil {
		sets = map[rrsetKey]*rrset{}
		s.rrsets[key] = sets
	}
	// Deletions must reference existing record sets.
	for _, k := range delKeys {
		if _, ok := sets[k]; !ok {
			return nil, "rrset-not-found"
		}
	}
	for i, k := range addKeys {
		r := additions[i]
		sets[k] = &rrset{
			Name:    ensureTrailingDot(r.Name),
			Type:    strings.ToUpper(r.Type),
			TTL:     r.TTL,
			Rrdatas: append([]string(nil), r.Rrdatas...),
		}
	}
	for _, k := range delKeys {
		delete(sets, k)
	}
	s.nextChangeID[key]++
	c := &change{
		ID:        fmt.Sprintf("%d", s.nextChangeID[key]),
		StartTime: nowRFC3339(),
		Status:    "done",
		IsServing: true,
		Additions: cloneRRsets(additions),
		Deletions: cloneRRsets(deletions),
	}
	// Normalize stored change copies the same way as applied sets.
	for _, r := range c.Additions {
		r.Type = strings.ToUpper(r.Type)
		r.Name = ensureTrailingDot(r.Name)
	}
	for _, r := range c.Deletions {
		r.Type = strings.ToUpper(r.Type)
		r.Name = ensureTrailingDot(r.Name)
	}
	s.changes[key] = append(s.changes[key], c)
	return changeJSON(c), ""
}

// GetChange returns a change by ID.
func (s *State) GetChange(project, zone, id string) (map[string]any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.changes[ZoneKey(project, zone)] {
		if c.ID == id {
			return changeJSON(c), true
		}
	}
	return nil, false
}

// ListChanges returns one page of changes, newest first.
func (s *State) ListChanges(project, zone string, pageSize int, pageToken string) ([]map[string]any, string, int, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.changes[ZoneKey(project, zone)]
	total := len(list)
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", 0, "invalid-page-token"
	}
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	out := make([]map[string]any, 0, end-offset)
	// Newest first.
	for i := total - 1 - offset; i >= 0 && len(out) < end-offset; i-- {
		out = append(out, changeJSON(list[i]))
	}
	next := pagination.PageToken(offset, pageSize, total)
	return out, next, total, ""
}

// GetRRSet returns one record set.
func (s *State) GetRRSet(project, zone, name, typ string) (map[string]any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sets := s.rrsets[ZoneKey(project, zone)]
	r, ok := sets[rrsetKey{name: normalizeDNSName(name), typ: strings.ToUpper(typ)}]
	if !ok {
		return nil, false
	}
	return rrsetJSON(r), true
}

// ListRRSets returns one page of record sets, optionally filtered by exact
// name (case-insensitive) and type.
func (s *State) ListRRSets(project, zone, nameFilter, typeFilter string, pageSize int, pageToken string) ([]map[string]any, string, int, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sets := s.rrsets[ZoneKey(project, zone)]
	var all []*rrset
	for _, r := range sets {
		if nameFilter != "" && !dnsNameEqual(r.Name, nameFilter) {
			continue
		}
		if typeFilter != "" && !strings.EqualFold(r.Type, typeFilter) {
			continue
		}
		all = append(all, r)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Name != all[j].Name {
			return all[i].Name < all[j].Name
		}
		return all[i].Type < all[j].Type
	})
	total := len(all)
	offset, err := pagination.Decode(pageToken)
	if err != nil {
		return nil, "", 0, "invalid-page-token"
	}
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	out := make([]map[string]any, 0, end-offset)
	for _, r := range all[offset:end] {
		out = append(out, rrsetJSON(r))
	}
	next := pagination.PageToken(offset, pageSize, total)
	return out, next, total, ""
}

// InZone reports whether name is the zone apex or a subdomain of dnsName.
func InZone(name, dnsName string) bool {
	n := normalizeDNSName(name)
	z := normalizeDNSName(dnsName)
	if z == "" {
		return false
	}
	return n == z || strings.HasSuffix(n, "."+z)
}

// normalizeDNSName lowercases and trims a trailing dot for comparisons.
func normalizeDNSName(name string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
}

// dnsNameEqual compares two DNS names case-insensitively, ignoring a
// trailing dot.
func dnsNameEqual(a, b string) bool {
	return normalizeDNSName(a) == normalizeDNSName(b)
}

// ensureTrailingDot returns name with exactly one trailing dot.
func ensureTrailingDot(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, ".")
	return name + "."
}

func cloneRRsets(in []*rrset) []*rrset {
	if len(in) == 0 {
		return nil
	}
	out := make([]*rrset, len(in))
	for i, r := range in {
		out[i] = &rrset{
			Name:    r.Name,
			Type:    r.Type,
			TTL:     r.TTL,
			Rrdatas: append([]string(nil), r.Rrdatas...),
		}
	}
	return out
}

// ValidateZoneName reports whether name is a valid managed zone name.
func ValidateZoneName(name string) bool { return zoneNameRe.MatchString(name) }

// ValidateRRSet performs basic validation of a record set against the zone.
func ValidateRRSet(r *rrset, dnsName string) string {
	if r == nil {
		return "record set is required"
	}
	if strings.TrimSpace(r.Name) == "" {
		return "rrset name is required"
	}
	if !rrTypeRe.MatchString(r.Type) {
		return "invalid rrset type"
	}
	if r.TTL < 0 {
		return "ttl must not be negative"
	}
	if len(r.Rrdatas) == 0 {
		return "rrdatas must not be empty"
	}
	if !InZone(r.Name, dnsName) {
		return fmt.Sprintf("rrset name %q is not within zone %q", r.Name, dnsName)
	}
	return ""
}
