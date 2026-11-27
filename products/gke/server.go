package gke

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	containerpb "github.com/bestony/google-cloud-emulator/gen/google/container/v1"
)

func notFound(what, project, name string) error {
	return status.Errorf(codes.NotFound, "%s %q in project %q was not found", what, name, project)
}

func invalidArg(format string, args ...any) error {
	return status.Errorf(codes.InvalidArgument, format, args...)
}

func alreadyExists(name string) error {
	return status.Errorf(codes.AlreadyExists, "resource %q already exists", name)
}

// parseParent splits a parent resource like
// "projects/p/locations/l" or "projects/p/zones/z" into project and location.
func parseParent(parent string) (project, location string) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	for i := 0; i+1 < len(parts); i += 2 {
		switch parts[i] {
		case "projects":
			project = parts[i+1]
		case "locations", "zones":
			location = parts[i+1]
		}
	}
	return project, location
}

// parseClusterName extracts the cluster id from a full resource name
// "projects/p/locations/l/clusters/c" (or the legacy zones form).
func parseClusterName(resource string) string {
	resource = strings.Trim(resource, "/")
	if i := strings.LastIndex(resource, "/clusters/"); i >= 0 {
		return resource[i+len("/clusters/"):]
	}
	parts := strings.Split(resource, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return resource
}

// parseOperationName extracts the operation id from a full resource name
// "projects/p/locations/l/operations/op-1".
func parseOperationName(resource string) string {
	resource = strings.Trim(resource, "/")
	if i := strings.LastIndex(resource, "/operations/"); i >= 0 {
		return resource[i+len("/operations/"):]
	}
	parts := strings.Split(resource, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return resource
}

func resolveProjectZone(projectID, zone, parent string) (project, location string) {
	project, location = parseParent(parent)
	if projectID != "" {
		project = projectID
	}
	if zone != "" {
		location = zone
	}
	return project, location
}

type server struct {
	containerpb.UnimplementedClusterManagerServer
	state *State
}

func byName(items []*containerpb.Cluster) {
	sort.Slice(items, func(i, j int) bool { return items[i].GetName() < items[j].GetName() })
}

func (s *server) ListClusters(_ context.Context, req *containerpb.ListClustersRequest) (*containerpb.ListClustersResponse, error) {
	project, location := resolveProjectZone(req.GetProjectId(), req.GetZone(), req.GetParent())
	if project == "" {
		return nil, invalidArg("project_id or parent is required")
	}
	s.state.mu.RLock()
	var items []*containerpb.Cluster
	for key, c := range s.state.clusters {
		parts := strings.Split(key, "|")
		if parts[0] != project {
			continue
		}
		if location != "" && parts[1] != location {
			continue
		}
		items = append(items, c)
	}
	s.state.mu.RUnlock()
	byName(items)
	return &containerpb.ListClustersResponse{Clusters: items}, nil
}

func (s *server) GetCluster(_ context.Context, req *containerpb.GetClusterRequest) (*containerpb.Cluster, error) {
	project, location := resolveProjectZone(req.GetProjectId(), req.GetZone(), "")
	name := req.GetClusterId()
	if req.GetName() != "" {
		p, l := parseParent(req.GetName())
		project, location = p, l
		name = parseClusterName(req.GetName())
	}
	if project == "" || name == "" {
		return nil, invalidArg("project and cluster name are required")
	}
	if location == "" {
		// Fall back to a project-wide search when no location was given.
		s.state.mu.RLock()
		var found *containerpb.Cluster
		for key, c := range s.state.clusters {
			parts := strings.Split(key, "|")
			if parts[0] == project && parts[2] == name {
				found = c
				break
			}
		}
		s.state.mu.RUnlock()
		if found == nil {
			return nil, notFound("cluster", project, name)
		}
		return proto.Clone(found).(*containerpb.Cluster), nil
	}
	key := clusterKey(project, location, name)
	s.state.mu.RLock()
	c, ok := s.state.clusters[key]
	s.state.mu.RUnlock()
	if !ok {
		return nil, notFound("cluster", project, name)
	}
	return proto.Clone(c).(*containerpb.Cluster), nil
}

func (s *server) CreateCluster(_ context.Context, req *containerpb.CreateClusterRequest) (*containerpb.Operation, error) {
	project, location := resolveProjectZone(req.GetProjectId(), req.GetZone(), req.GetParent())
	if req.GetCluster() == nil {
		return nil, invalidArg("cluster body is required")
	}
	body := req.GetCluster()
	name := body.GetName()
	if project == "" || location == "" {
		return nil, invalidArg("project and zone (or parent) are required")
	}
	if name == "" {
		return nil, invalidArg("cluster name is required")
	}
	if strings.Contains(name, "/") {
		return nil, invalidArg("cluster name %q must not contain '/'", name)
	}

	key := clusterKey(project, location, name)
	s.state.mu.Lock()
	if _, ok := s.state.clusters[key]; ok {
		s.state.mu.Unlock()
		return nil, alreadyExists(name)
	}
	c := proto.Clone(body).(*containerpb.Cluster)
	c.Id = strconv.FormatUint(s.state.nextIDLocked(), 10)
	c.Name = name
	c.Zone = location
	c.Location = location
	c.SelfLink = clusterSelfLink(project, location, name)
	now := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	c.CreateTime = now
	c.Status = containerpb.Cluster_RUNNING
	if c.InitialClusterVersion == "" {
		c.InitialClusterVersion = emulatorVersion
	}
	if c.CurrentMasterVersion == "" {
		c.CurrentMasterVersion = emulatorVersion
	}
	c.CurrentNodeVersion = emulatorVersion
	c.Etag = strconv.FormatUint(s.state.nextIDLocked(), 10)
	s.state.clusters[key] = c
	op := s.state.addOperation(project, location, containerpb.Operation_CREATE_CLUSTER,
		c.SelfLink, "CreateCluster")
	s.state.mu.Unlock()
	return op, nil
}

func (s *server) UpdateCluster(_ context.Context, req *containerpb.UpdateClusterRequest) (*containerpb.Operation, error) {
	project, location := resolveProjectZone(req.GetProjectId(), req.GetZone(), "")
	name := req.GetClusterId()
	if req.GetName() != "" {
		p, l := parseParent(req.GetName())
		project, location = p, l
		name = parseClusterName(req.GetName())
	}
	if project == "" || name == "" {
		return nil, invalidArg("project and cluster name are required")
	}
	if req.GetUpdate() == nil {
		return nil, invalidArg("update body is required")
	}
	update := req.GetUpdate()

	key := clusterKey(project, location, name)
	s.state.mu.Lock()
	stored, ok := s.state.clusters[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("cluster", project, name)
	}
	c := proto.Clone(stored).(*containerpb.Cluster)
	opType := containerpb.Operation_TYPE_UNSPECIFIED
	if update.DesiredNodeVersion != "" {
		c.CurrentMasterVersion = update.DesiredNodeVersion
		opType = containerpb.Operation_UPGRADE_MASTER
	}
	if update.DesiredMonitoringService != "" {
		c.MonitoringService = update.DesiredMonitoringService
	}
	if update.DesiredAddonsConfig != nil {
		c.AddonsConfig = proto.Clone(update.DesiredAddonsConfig).(*containerpb.AddonsConfig)
	}
	if update.DesiredDatabaseEncryption != nil {
		c.DatabaseEncryption = proto.Clone(update.DesiredDatabaseEncryption).(*containerpb.DatabaseEncryption)
	}
	if update.DesiredWorkloadIdentityConfig != nil {
		c.WorkloadIdentityConfig = proto.Clone(update.DesiredWorkloadIdentityConfig).(*containerpb.WorkloadIdentityConfig)
	}
	if update.DesiredMeshCertificates != nil {
		c.MeshCertificates = proto.Clone(update.DesiredMeshCertificates).(*containerpb.MeshCertificates)
	}
	if update.DesiredShieldedNodes != nil {
		c.ShieldedNodes = proto.Clone(update.DesiredShieldedNodes).(*containerpb.ShieldedNodes)
	}
	if update.DesiredCostManagementConfig != nil {
		c.CostManagementConfig = proto.Clone(update.DesiredCostManagementConfig).(*containerpb.CostManagementConfig)
	}
	if len(update.DesiredLocations) > 0 {
		c.Locations = append([]string(nil), update.DesiredLocations...)
	}
	c.Etag = fmt.Sprintf("%d", s.state.nextIDLocked())
	s.state.clusters[key] = c
	detail := "UpdateCluster"
	if opType == containerpb.Operation_UPGRADE_MASTER {
		detail = "UpgradeMaster"
	}
	op := s.state.addOperation(project, location, opType, c.SelfLink, detail)
	s.state.mu.Unlock()
	return op, nil
}

func (s *server) DeleteCluster(_ context.Context, req *containerpb.DeleteClusterRequest) (*containerpb.Operation, error) {
	project, location := resolveProjectZone(req.GetProjectId(), req.GetZone(), "")
	name := req.GetClusterId()
	if req.GetName() != "" {
		p, l := parseParent(req.GetName())
		project, location = p, l
		name = parseClusterName(req.GetName())
	}
	if project == "" || name == "" {
		return nil, invalidArg("project and cluster name are required")
	}
	key := clusterKey(project, location, name)
	s.state.mu.Lock()
	c, ok := s.state.clusters[key]
	if !ok {
		s.state.mu.Unlock()
		return nil, notFound("cluster", project, name)
	}
	target := c.SelfLink
	delete(s.state.clusters, key)
	op := s.state.addOperation(project, location, containerpb.Operation_DELETE_CLUSTER, target, "DeleteCluster")
	s.state.mu.Unlock()
	return op, nil
}

func (s *server) ListOperations(_ context.Context, req *containerpb.ListOperationsRequest) (*containerpb.ListOperationsResponse, error) {
	project, location := resolveProjectZone(req.GetProjectId(), req.GetZone(), req.GetParent())
	if project == "" {
		return nil, invalidArg("project_id or parent is required")
	}
	return &containerpb.ListOperationsResponse{
		Operations: s.state.listOperations(project, location),
	}, nil
}

func (s *server) GetOperation(_ context.Context, req *containerpb.GetOperationRequest) (*containerpb.Operation, error) {
	project, location := resolveProjectZone(req.GetProjectId(), req.GetZone(), "")
	name := req.GetOperationId()
	if req.GetName() != "" {
		p, l := parseParent(req.GetName())
		project, location = p, l
		name = parseOperationName(req.GetName())
	}
	if project == "" || name == "" {
		return nil, invalidArg("project and operation name are required")
	}
	op, ok := s.state.getOperation(project, location, name)
	if !ok {
		return nil, notFound("operation", project, name)
	}
	return proto.Clone(op).(*containerpb.Operation), nil
}
