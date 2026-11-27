package gke

import (
	"context"
	"fmt"
	"sort"
	"strings"

	containerpb "github.com/bestony/google-cloud-emulator/gen/google/container/v1"
	"google.golang.org/protobuf/proto"
)

func parseNodePoolPath(resource string) (project, location, cluster, pool string) {
	parts := strings.Split(strings.Trim(resource, "/"), "/")
	for i := 0; i+1 < len(parts); i += 2 {
		switch parts[i] {
		case "projects":
			project = parts[i+1]
		case "locations", "zones":
			location = parts[i+1]
		case "clusters":
			cluster = parts[i+1]
		case "nodePools":
			pool = parts[i+1]
		}
	}
	return project, location, cluster, pool
}

func nodePoolSelfLink(project, location, cluster, pool string) string {
	return fmt.Sprintf("%s/nodePools/%s", clusterSelfLink(project, location, cluster), pool)
}

func nodePoolRef(projectID, zone, parentOrName, clusterID, poolID string) (project, location, cluster, pool string) {
	project, location = resolveProjectZone(projectID, zone, "")
	cluster, pool = clusterID, poolID
	if parentOrName != "" {
		p, l, c, n := parseNodePoolPath(parentOrName)
		if p != "" {
			project = p
		}
		if l != "" {
			location = l
		}
		if c != "" {
			cluster = c
		}
		if n != "" {
			pool = n
		}
	}
	return project, location, cluster, pool
}

func findNodePool(c *containerpb.Cluster, name string) (int, *containerpb.NodePool) {
	for i, np := range c.GetNodePools() {
		if np.GetName() == name {
			return i, np
		}
	}
	return -1, nil
}

func (s *server) getClusterLocked(project, location, cluster string) (*containerpb.Cluster, error) {
	c, ok := s.state.clusters[clusterKey(project, location, cluster)]
	if !ok {
		return nil, notFound("cluster", project, cluster)
	}
	return c, nil
}

func (s *server) ListNodePools(_ context.Context, req *containerpb.ListNodePoolsRequest) (*containerpb.ListNodePoolsResponse, error) {
	project, location, cluster, _ := nodePoolRef(req.GetProjectId(), req.GetZone(), req.GetParent(), req.GetClusterId(), "")
	if project == "" || cluster == "" {
		return nil, invalidArg("project and cluster are required")
	}
	s.state.mu.RLock()
	c, err := s.getClusterLocked(project, location, cluster)
	if err != nil {
		s.state.mu.RUnlock()
		return nil, err
	}
	out := make([]*containerpb.NodePool, 0, len(c.GetNodePools()))
	for _, np := range c.GetNodePools() {
		out = append(out, proto.Clone(np).(*containerpb.NodePool))
	}
	s.state.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].GetName() < out[j].GetName() })
	return &containerpb.ListNodePoolsResponse{NodePools: out}, nil
}

func (s *server) GetNodePool(_ context.Context, req *containerpb.GetNodePoolRequest) (*containerpb.NodePool, error) {
	project, location, cluster, pool := nodePoolRef(req.GetProjectId(), req.GetZone(), req.GetName(), req.GetClusterId(), req.GetNodePoolId())
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	c, err := s.getClusterLocked(project, location, cluster)
	if err != nil {
		return nil, err
	}
	_, np := findNodePool(c, pool)
	if np == nil {
		return nil, notFound("node pool", project, pool)
	}
	return proto.Clone(np).(*containerpb.NodePool), nil
}

func (s *server) CreateNodePool(_ context.Context, req *containerpb.CreateNodePoolRequest) (*containerpb.Operation, error) {
	project, location, cluster, _ := nodePoolRef(req.GetProjectId(), req.GetZone(), req.GetParent(), req.GetClusterId(), "")
	body := req.GetNodePool()
	if body == nil || body.GetName() == "" {
		return nil, invalidArg("node_pool.name is required")
	}
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	c, err := s.getClusterLocked(project, location, cluster)
	if err != nil {
		return nil, err
	}
	if _, existing := findNodePool(c, body.GetName()); existing != nil {
		return nil, alreadyExists(body.GetName())
	}
	np := proto.Clone(body).(*containerpb.NodePool)
	np.SelfLink = nodePoolSelfLink(project, location, cluster, np.GetName())
	np.Status = containerpb.NodePool_RUNNING
	if np.Version == "" {
		np.Version = c.GetCurrentNodeVersion()
	}
	if np.Version == "" {
		np.Version = emulatorVersion
	}
	if len(np.Locations) == 0 {
		np.Locations = []string{location}
	}
	np.Etag = fmt.Sprintf("%d", s.state.nextIDLocked())
	copyCluster := proto.Clone(c).(*containerpb.Cluster)
	copyCluster.NodePools = append(copyCluster.NodePools, np)
	s.state.clusters[clusterKey(project, location, cluster)] = copyCluster
	return s.state.addOperation(project, location, containerpb.Operation_CREATE_NODE_POOL, np.SelfLink, "CreateNodePool"), nil
}

func (s *server) DeleteNodePool(_ context.Context, req *containerpb.DeleteNodePoolRequest) (*containerpb.Operation, error) {
	project, location, cluster, pool := nodePoolRef(req.GetProjectId(), req.GetZone(), req.GetName(), req.GetClusterId(), req.GetNodePoolId())
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	c, err := s.getClusterLocked(project, location, cluster)
	if err != nil {
		return nil, err
	}
	idx, np := findNodePool(c, pool)
	if np == nil {
		return nil, notFound("node pool", project, pool)
	}
	copyCluster := proto.Clone(c).(*containerpb.Cluster)
	copyCluster.NodePools = append(copyCluster.NodePools[:idx], copyCluster.NodePools[idx+1:]...)
	s.state.clusters[clusterKey(project, location, cluster)] = copyCluster
	return s.state.addOperation(project, location, containerpb.Operation_DELETE_NODE_POOL, np.GetSelfLink(), "DeleteNodePool"), nil
}

func (s *server) mutateNodePool(project, location, cluster, pool, detail string, opType containerpb.Operation_Type, mutate func(*containerpb.NodePool)) (*containerpb.Operation, error) {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	c, err := s.getClusterLocked(project, location, cluster)
	if err != nil {
		return nil, err
	}
	idx, np := findNodePool(c, pool)
	if np == nil {
		return nil, notFound("node pool", project, pool)
	}
	copyCluster := proto.Clone(c).(*containerpb.Cluster)
	mutated := proto.Clone(np).(*containerpb.NodePool)
	mutate(mutated)
	mutated.Etag = fmt.Sprintf("%d", s.state.nextIDLocked())
	copyCluster.NodePools[idx] = mutated
	s.state.clusters[clusterKey(project, location, cluster)] = copyCluster
	return s.state.addOperation(project, location, opType, mutated.GetSelfLink(), detail), nil
}

func (s *server) UpdateNodePool(_ context.Context, req *containerpb.UpdateNodePoolRequest) (*containerpb.Operation, error) {
	project, location, cluster, pool := nodePoolRef(req.GetProjectId(), req.GetZone(), req.GetName(), req.GetClusterId(), req.GetNodePoolId())
	return s.mutateNodePool(project, location, cluster, pool, "UpdateNodePool", containerpb.Operation_UPGRADE_NODES, func(np *containerpb.NodePool) {
		if req.GetNodeVersion() != "" {
			np.Version = req.GetNodeVersion()
		}
		if req.GetImageType() != "" {
			if np.Config == nil {
				np.Config = &containerpb.NodeConfig{}
			}
			np.Config.ImageType = req.GetImageType()
		}
		if len(req.GetLocations()) > 0 {
			np.Locations = append([]string(nil), req.GetLocations()...)
		}
		if req.GetWorkloadMetadataConfig() != nil {
			if np.Config == nil {
				np.Config = &containerpb.NodeConfig{}
			}
			np.Config.WorkloadMetadataConfig = proto.Clone(req.GetWorkloadMetadataConfig()).(*containerpb.WorkloadMetadataConfig)
		}
		if req.GetUpgradeSettings() != nil {
			np.UpgradeSettings = proto.Clone(req.GetUpgradeSettings()).(*containerpb.NodePool_UpgradeSettings)
		}
	})
}

func (s *server) SetNodePoolAutoscaling(_ context.Context, req *containerpb.SetNodePoolAutoscalingRequest) (*containerpb.Operation, error) {
	project, location, cluster, pool := nodePoolRef(req.GetProjectId(), req.GetZone(), req.GetName(), req.GetClusterId(), req.GetNodePoolId())
	return s.mutateNodePool(project, location, cluster, pool, "SetNodePoolAutoscaling", containerpb.Operation_TYPE_UNSPECIFIED, func(np *containerpb.NodePool) {
		np.Autoscaling = proto.Clone(req.GetAutoscaling()).(*containerpb.NodePoolAutoscaling)
	})
}

func (s *server) SetNodePoolManagement(_ context.Context, req *containerpb.SetNodePoolManagementRequest) (*containerpb.Operation, error) {
	project, location, cluster, pool := nodePoolRef(req.GetProjectId(), req.GetZone(), req.GetName(), req.GetClusterId(), req.GetNodePoolId())
	return s.mutateNodePool(project, location, cluster, pool, "SetNodePoolManagement", containerpb.Operation_SET_NODE_POOL_MANAGEMENT, func(np *containerpb.NodePool) {
		np.Management = proto.Clone(req.GetManagement()).(*containerpb.NodeManagement)
	})
}

func (s *server) SetNodePoolSize(_ context.Context, req *containerpb.SetNodePoolSizeRequest) (*containerpb.Operation, error) {
	project, location, cluster, pool := nodePoolRef(req.GetProjectId(), req.GetZone(), req.GetName(), req.GetClusterId(), req.GetNodePoolId())
	return s.mutateNodePool(project, location, cluster, pool, "SetNodePoolSize", containerpb.Operation_SET_NODE_POOL_SIZE, func(np *containerpb.NodePool) {
		np.InitialNodeCount = req.GetNodeCount()
	})
}

func (s *server) GetServerConfig(_ context.Context, req *containerpb.GetServerConfigRequest) (*containerpb.ServerConfig, error) {
	return &containerpb.ServerConfig{
		DefaultClusterVersion: emulatorVersion,
		ValidNodeVersions:     []string{emulatorVersion},
		ValidMasterVersions:   []string{emulatorVersion},
		DefaultImageType:      "COS_CONTAINERD",
		ValidImageTypes:       []string{"COS_CONTAINERD", "UBUNTU_CONTAINERD"},
	}, nil
}

func (s *server) GetJSONWebKeys(_ context.Context, req *containerpb.GetJSONWebKeysRequest) (*containerpb.GetJSONWebKeysResponse, error) {
	// A deterministic development-only key document. Clients that only need a
	// syntactically valid JWKS can consume it; it is not used for real signing.
	return &containerpb.GetJSONWebKeysResponse{Keys: []*containerpb.Jwk{{Kty: "RSA", Alg: "RS256", Use: "sig", Kid: "gcpex-development-key", N: "gcpex", E: "AQAB"}}}, nil
}
