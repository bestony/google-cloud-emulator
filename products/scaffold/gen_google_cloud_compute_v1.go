package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/cloud/compute/v1"
)

func init() {
	registerEntry("google.cloud.compute.v1.AcceleratorTypes",
		func(s grpc.ServiceRegistrar) { pb.RegisterAcceleratorTypesServer(s, &pb.UnimplementedAcceleratorTypesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAcceleratorTypesHandlerServer(ctx, mux, &pb.UnimplementedAcceleratorTypesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Addresses",
		func(s grpc.ServiceRegistrar) { pb.RegisterAddressesServer(s, &pb.UnimplementedAddressesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAddressesHandlerServer(ctx, mux, &pb.UnimplementedAddressesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Advice",
		func(s grpc.ServiceRegistrar) { pb.RegisterAdviceServer(s, &pb.UnimplementedAdviceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAdviceHandlerServer(ctx, mux, &pb.UnimplementedAdviceServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Autoscalers",
		func(s grpc.ServiceRegistrar) { pb.RegisterAutoscalersServer(s, &pb.UnimplementedAutoscalersServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterAutoscalersHandlerServer(ctx, mux, &pb.UnimplementedAutoscalersServer{}) },
	)
	registerEntry("google.cloud.compute.v1.BackendBuckets",
		func(s grpc.ServiceRegistrar) { pb.RegisterBackendBucketsServer(s, &pb.UnimplementedBackendBucketsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterBackendBucketsHandlerServer(ctx, mux, &pb.UnimplementedBackendBucketsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.BackendServices",
		func(s grpc.ServiceRegistrar) { pb.RegisterBackendServicesServer(s, &pb.UnimplementedBackendServicesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterBackendServicesHandlerServer(ctx, mux, &pb.UnimplementedBackendServicesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.CrossSiteNetworks",
		func(s grpc.ServiceRegistrar) { pb.RegisterCrossSiteNetworksServer(s, &pb.UnimplementedCrossSiteNetworksServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterCrossSiteNetworksHandlerServer(ctx, mux, &pb.UnimplementedCrossSiteNetworksServer{}) },
	)
	registerEntry("google.cloud.compute.v1.DiskTypes",
		func(s grpc.ServiceRegistrar) { pb.RegisterDiskTypesServer(s, &pb.UnimplementedDiskTypesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDiskTypesHandlerServer(ctx, mux, &pb.UnimplementedDiskTypesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Disks",
		func(s grpc.ServiceRegistrar) { pb.RegisterDisksServer(s, &pb.UnimplementedDisksServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterDisksHandlerServer(ctx, mux, &pb.UnimplementedDisksServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ExternalVpnGateways",
		func(s grpc.ServiceRegistrar) { pb.RegisterExternalVpnGatewaysServer(s, &pb.UnimplementedExternalVpnGatewaysServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterExternalVpnGatewaysHandlerServer(ctx, mux, &pb.UnimplementedExternalVpnGatewaysServer{}) },
	)
	registerEntry("google.cloud.compute.v1.FirewallPolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterFirewallPoliciesServer(s, &pb.UnimplementedFirewallPoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterFirewallPoliciesHandlerServer(ctx, mux, &pb.UnimplementedFirewallPoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Firewalls",
		func(s grpc.ServiceRegistrar) { pb.RegisterFirewallsServer(s, &pb.UnimplementedFirewallsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterFirewallsHandlerServer(ctx, mux, &pb.UnimplementedFirewallsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ForwardingRules",
		func(s grpc.ServiceRegistrar) { pb.RegisterForwardingRulesServer(s, &pb.UnimplementedForwardingRulesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterForwardingRulesHandlerServer(ctx, mux, &pb.UnimplementedForwardingRulesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.FutureReservations",
		func(s grpc.ServiceRegistrar) { pb.RegisterFutureReservationsServer(s, &pb.UnimplementedFutureReservationsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterFutureReservationsHandlerServer(ctx, mux, &pb.UnimplementedFutureReservationsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.GlobalAddresses",
		func(s grpc.ServiceRegistrar) { pb.RegisterGlobalAddressesServer(s, &pb.UnimplementedGlobalAddressesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterGlobalAddressesHandlerServer(ctx, mux, &pb.UnimplementedGlobalAddressesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.GlobalForwardingRules",
		func(s grpc.ServiceRegistrar) { pb.RegisterGlobalForwardingRulesServer(s, &pb.UnimplementedGlobalForwardingRulesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterGlobalForwardingRulesHandlerServer(ctx, mux, &pb.UnimplementedGlobalForwardingRulesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.GlobalFrontendSettingsService",
		func(s grpc.ServiceRegistrar) { pb.RegisterGlobalFrontendSettingsServiceServer(s, &pb.UnimplementedGlobalFrontendSettingsServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterGlobalFrontendSettingsServiceHandlerServer(ctx, mux, &pb.UnimplementedGlobalFrontendSettingsServiceServer{}) },
	)
	registerEntry("google.cloud.compute.v1.GlobalNetworkEndpointGroups",
		func(s grpc.ServiceRegistrar) { pb.RegisterGlobalNetworkEndpointGroupsServer(s, &pb.UnimplementedGlobalNetworkEndpointGroupsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterGlobalNetworkEndpointGroupsHandlerServer(ctx, mux, &pb.UnimplementedGlobalNetworkEndpointGroupsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.GlobalOperations",
		func(s grpc.ServiceRegistrar) { pb.RegisterGlobalOperationsServer(s, &pb.UnimplementedGlobalOperationsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterGlobalOperationsHandlerServer(ctx, mux, &pb.UnimplementedGlobalOperationsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.GlobalOrganizationOperations",
		func(s grpc.ServiceRegistrar) { pb.RegisterGlobalOrganizationOperationsServer(s, &pb.UnimplementedGlobalOrganizationOperationsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterGlobalOrganizationOperationsHandlerServer(ctx, mux, &pb.UnimplementedGlobalOrganizationOperationsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.GlobalPublicDelegatedPrefixes",
		func(s grpc.ServiceRegistrar) { pb.RegisterGlobalPublicDelegatedPrefixesServer(s, &pb.UnimplementedGlobalPublicDelegatedPrefixesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterGlobalPublicDelegatedPrefixesHandlerServer(ctx, mux, &pb.UnimplementedGlobalPublicDelegatedPrefixesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.GlobalVmExtensionPolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterGlobalVmExtensionPoliciesServer(s, &pb.UnimplementedGlobalVmExtensionPoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterGlobalVmExtensionPoliciesHandlerServer(ctx, mux, &pb.UnimplementedGlobalVmExtensionPoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.HealthChecks",
		func(s grpc.ServiceRegistrar) { pb.RegisterHealthChecksServer(s, &pb.UnimplementedHealthChecksServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterHealthChecksHandlerServer(ctx, mux, &pb.UnimplementedHealthChecksServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Hosts",
		func(s grpc.ServiceRegistrar) { pb.RegisterHostsServer(s, &pb.UnimplementedHostsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterHostsHandlerServer(ctx, mux, &pb.UnimplementedHostsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ImageFamilyViews",
		func(s grpc.ServiceRegistrar) { pb.RegisterImageFamilyViewsServer(s, &pb.UnimplementedImageFamilyViewsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterImageFamilyViewsHandlerServer(ctx, mux, &pb.UnimplementedImageFamilyViewsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ImageViews",
		func(s grpc.ServiceRegistrar) { pb.RegisterImageViewsServer(s, &pb.UnimplementedImageViewsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterImageViewsHandlerServer(ctx, mux, &pb.UnimplementedImageViewsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Images",
		func(s grpc.ServiceRegistrar) { pb.RegisterImagesServer(s, &pb.UnimplementedImagesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterImagesHandlerServer(ctx, mux, &pb.UnimplementedImagesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InstanceGroupManagerResizeRequests",
		func(s grpc.ServiceRegistrar) { pb.RegisterInstanceGroupManagerResizeRequestsServer(s, &pb.UnimplementedInstanceGroupManagerResizeRequestsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInstanceGroupManagerResizeRequestsHandlerServer(ctx, mux, &pb.UnimplementedInstanceGroupManagerResizeRequestsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InstanceGroupManagers",
		func(s grpc.ServiceRegistrar) { pb.RegisterInstanceGroupManagersServer(s, &pb.UnimplementedInstanceGroupManagersServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInstanceGroupManagersHandlerServer(ctx, mux, &pb.UnimplementedInstanceGroupManagersServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InstanceGroups",
		func(s grpc.ServiceRegistrar) { pb.RegisterInstanceGroupsServer(s, &pb.UnimplementedInstanceGroupsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInstanceGroupsHandlerServer(ctx, mux, &pb.UnimplementedInstanceGroupsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InstanceSettingsService",
		func(s grpc.ServiceRegistrar) { pb.RegisterInstanceSettingsServiceServer(s, &pb.UnimplementedInstanceSettingsServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInstanceSettingsServiceHandlerServer(ctx, mux, &pb.UnimplementedInstanceSettingsServiceServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InstanceTemplates",
		func(s grpc.ServiceRegistrar) { pb.RegisterInstanceTemplatesServer(s, &pb.UnimplementedInstanceTemplatesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInstanceTemplatesHandlerServer(ctx, mux, &pb.UnimplementedInstanceTemplatesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Instances",
		func(s grpc.ServiceRegistrar) { pb.RegisterInstancesServer(s, &pb.UnimplementedInstancesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInstancesHandlerServer(ctx, mux, &pb.UnimplementedInstancesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InstantSnapshotGroups",
		func(s grpc.ServiceRegistrar) { pb.RegisterInstantSnapshotGroupsServer(s, &pb.UnimplementedInstantSnapshotGroupsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInstantSnapshotGroupsHandlerServer(ctx, mux, &pb.UnimplementedInstantSnapshotGroupsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InstantSnapshots",
		func(s grpc.ServiceRegistrar) { pb.RegisterInstantSnapshotsServer(s, &pb.UnimplementedInstantSnapshotsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInstantSnapshotsHandlerServer(ctx, mux, &pb.UnimplementedInstantSnapshotsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InterconnectAttachmentGroups",
		func(s grpc.ServiceRegistrar) { pb.RegisterInterconnectAttachmentGroupsServer(s, &pb.UnimplementedInterconnectAttachmentGroupsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInterconnectAttachmentGroupsHandlerServer(ctx, mux, &pb.UnimplementedInterconnectAttachmentGroupsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InterconnectAttachments",
		func(s grpc.ServiceRegistrar) { pb.RegisterInterconnectAttachmentsServer(s, &pb.UnimplementedInterconnectAttachmentsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInterconnectAttachmentsHandlerServer(ctx, mux, &pb.UnimplementedInterconnectAttachmentsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InterconnectGroups",
		func(s grpc.ServiceRegistrar) { pb.RegisterInterconnectGroupsServer(s, &pb.UnimplementedInterconnectGroupsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInterconnectGroupsHandlerServer(ctx, mux, &pb.UnimplementedInterconnectGroupsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InterconnectLocations",
		func(s grpc.ServiceRegistrar) { pb.RegisterInterconnectLocationsServer(s, &pb.UnimplementedInterconnectLocationsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInterconnectLocationsHandlerServer(ctx, mux, &pb.UnimplementedInterconnectLocationsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.InterconnectRemoteLocations",
		func(s grpc.ServiceRegistrar) { pb.RegisterInterconnectRemoteLocationsServer(s, &pb.UnimplementedInterconnectRemoteLocationsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInterconnectRemoteLocationsHandlerServer(ctx, mux, &pb.UnimplementedInterconnectRemoteLocationsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Interconnects",
		func(s grpc.ServiceRegistrar) { pb.RegisterInterconnectsServer(s, &pb.UnimplementedInterconnectsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterInterconnectsHandlerServer(ctx, mux, &pb.UnimplementedInterconnectsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.LicenseCodes",
		func(s grpc.ServiceRegistrar) { pb.RegisterLicenseCodesServer(s, &pb.UnimplementedLicenseCodesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterLicenseCodesHandlerServer(ctx, mux, &pb.UnimplementedLicenseCodesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Licenses",
		func(s grpc.ServiceRegistrar) { pb.RegisterLicensesServer(s, &pb.UnimplementedLicensesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterLicensesHandlerServer(ctx, mux, &pb.UnimplementedLicensesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.MachineImages",
		func(s grpc.ServiceRegistrar) { pb.RegisterMachineImagesServer(s, &pb.UnimplementedMachineImagesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterMachineImagesHandlerServer(ctx, mux, &pb.UnimplementedMachineImagesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.MachineTypes",
		func(s grpc.ServiceRegistrar) { pb.RegisterMachineTypesServer(s, &pb.UnimplementedMachineTypesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterMachineTypesHandlerServer(ctx, mux, &pb.UnimplementedMachineTypesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ManagedRulesets",
		func(s grpc.ServiceRegistrar) { pb.RegisterManagedRulesetsServer(s, &pb.UnimplementedManagedRulesetsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterManagedRulesetsHandlerServer(ctx, mux, &pb.UnimplementedManagedRulesetsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.NetworkAttachments",
		func(s grpc.ServiceRegistrar) { pb.RegisterNetworkAttachmentsServer(s, &pb.UnimplementedNetworkAttachmentsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNetworkAttachmentsHandlerServer(ctx, mux, &pb.UnimplementedNetworkAttachmentsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.NetworkEdgeSecurityServices",
		func(s grpc.ServiceRegistrar) { pb.RegisterNetworkEdgeSecurityServicesServer(s, &pb.UnimplementedNetworkEdgeSecurityServicesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNetworkEdgeSecurityServicesHandlerServer(ctx, mux, &pb.UnimplementedNetworkEdgeSecurityServicesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.NetworkEndpointGroups",
		func(s grpc.ServiceRegistrar) { pb.RegisterNetworkEndpointGroupsServer(s, &pb.UnimplementedNetworkEndpointGroupsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNetworkEndpointGroupsHandlerServer(ctx, mux, &pb.UnimplementedNetworkEndpointGroupsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.NetworkFirewallPolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterNetworkFirewallPoliciesServer(s, &pb.UnimplementedNetworkFirewallPoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNetworkFirewallPoliciesHandlerServer(ctx, mux, &pb.UnimplementedNetworkFirewallPoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.NetworkProfiles",
		func(s grpc.ServiceRegistrar) { pb.RegisterNetworkProfilesServer(s, &pb.UnimplementedNetworkProfilesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNetworkProfilesHandlerServer(ctx, mux, &pb.UnimplementedNetworkProfilesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Networks",
		func(s grpc.ServiceRegistrar) { pb.RegisterNetworksServer(s, &pb.UnimplementedNetworksServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNetworksHandlerServer(ctx, mux, &pb.UnimplementedNetworksServer{}) },
	)
	registerEntry("google.cloud.compute.v1.NodeGroups",
		func(s grpc.ServiceRegistrar) { pb.RegisterNodeGroupsServer(s, &pb.UnimplementedNodeGroupsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNodeGroupsHandlerServer(ctx, mux, &pb.UnimplementedNodeGroupsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.NodeTemplates",
		func(s grpc.ServiceRegistrar) { pb.RegisterNodeTemplatesServer(s, &pb.UnimplementedNodeTemplatesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNodeTemplatesHandlerServer(ctx, mux, &pb.UnimplementedNodeTemplatesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.NodeTypes",
		func(s grpc.ServiceRegistrar) { pb.RegisterNodeTypesServer(s, &pb.UnimplementedNodeTypesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterNodeTypesHandlerServer(ctx, mux, &pb.UnimplementedNodeTypesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.OrganizationSecurityPolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterOrganizationSecurityPoliciesServer(s, &pb.UnimplementedOrganizationSecurityPoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterOrganizationSecurityPoliciesHandlerServer(ctx, mux, &pb.UnimplementedOrganizationSecurityPoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.PacketMirrorings",
		func(s grpc.ServiceRegistrar) { pb.RegisterPacketMirroringsServer(s, &pb.UnimplementedPacketMirroringsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterPacketMirroringsHandlerServer(ctx, mux, &pb.UnimplementedPacketMirroringsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.PreviewFeatures",
		func(s grpc.ServiceRegistrar) { pb.RegisterPreviewFeaturesServer(s, &pb.UnimplementedPreviewFeaturesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterPreviewFeaturesHandlerServer(ctx, mux, &pb.UnimplementedPreviewFeaturesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ProjectViews",
		func(s grpc.ServiceRegistrar) { pb.RegisterProjectViewsServer(s, &pb.UnimplementedProjectViewsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterProjectViewsHandlerServer(ctx, mux, &pb.UnimplementedProjectViewsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Projects",
		func(s grpc.ServiceRegistrar) { pb.RegisterProjectsServer(s, &pb.UnimplementedProjectsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterProjectsHandlerServer(ctx, mux, &pb.UnimplementedProjectsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.PublicAdvertisedPrefixes",
		func(s grpc.ServiceRegistrar) { pb.RegisterPublicAdvertisedPrefixesServer(s, &pb.UnimplementedPublicAdvertisedPrefixesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterPublicAdvertisedPrefixesHandlerServer(ctx, mux, &pb.UnimplementedPublicAdvertisedPrefixesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.PublicDelegatedPrefixes",
		func(s grpc.ServiceRegistrar) { pb.RegisterPublicDelegatedPrefixesServer(s, &pb.UnimplementedPublicDelegatedPrefixesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterPublicDelegatedPrefixesHandlerServer(ctx, mux, &pb.UnimplementedPublicDelegatedPrefixesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionAutoscalers",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionAutoscalersServer(s, &pb.UnimplementedRegionAutoscalersServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionAutoscalersHandlerServer(ctx, mux, &pb.UnimplementedRegionAutoscalersServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionBackendBuckets",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionBackendBucketsServer(s, &pb.UnimplementedRegionBackendBucketsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionBackendBucketsHandlerServer(ctx, mux, &pb.UnimplementedRegionBackendBucketsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionBackendServices",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionBackendServicesServer(s, &pb.UnimplementedRegionBackendServicesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionBackendServicesHandlerServer(ctx, mux, &pb.UnimplementedRegionBackendServicesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionCommitments",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionCommitmentsServer(s, &pb.UnimplementedRegionCommitmentsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionCommitmentsHandlerServer(ctx, mux, &pb.UnimplementedRegionCommitmentsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionCompositeHealthChecks",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionCompositeHealthChecksServer(s, &pb.UnimplementedRegionCompositeHealthChecksServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionCompositeHealthChecksHandlerServer(ctx, mux, &pb.UnimplementedRegionCompositeHealthChecksServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionDiskTypes",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionDiskTypesServer(s, &pb.UnimplementedRegionDiskTypesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionDiskTypesHandlerServer(ctx, mux, &pb.UnimplementedRegionDiskTypesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionDisks",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionDisksServer(s, &pb.UnimplementedRegionDisksServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionDisksHandlerServer(ctx, mux, &pb.UnimplementedRegionDisksServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionHealthAggregationPolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionHealthAggregationPoliciesServer(s, &pb.UnimplementedRegionHealthAggregationPoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionHealthAggregationPoliciesHandlerServer(ctx, mux, &pb.UnimplementedRegionHealthAggregationPoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionHealthCheckServices",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionHealthCheckServicesServer(s, &pb.UnimplementedRegionHealthCheckServicesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionHealthCheckServicesHandlerServer(ctx, mux, &pb.UnimplementedRegionHealthCheckServicesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionHealthChecks",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionHealthChecksServer(s, &pb.UnimplementedRegionHealthChecksServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionHealthChecksHandlerServer(ctx, mux, &pb.UnimplementedRegionHealthChecksServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionHealthSources",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionHealthSourcesServer(s, &pb.UnimplementedRegionHealthSourcesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionHealthSourcesHandlerServer(ctx, mux, &pb.UnimplementedRegionHealthSourcesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionInstanceGroupManagerResizeRequests",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionInstanceGroupManagerResizeRequestsServer(s, &pb.UnimplementedRegionInstanceGroupManagerResizeRequestsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionInstanceGroupManagerResizeRequestsHandlerServer(ctx, mux, &pb.UnimplementedRegionInstanceGroupManagerResizeRequestsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionInstanceGroupManagers",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionInstanceGroupManagersServer(s, &pb.UnimplementedRegionInstanceGroupManagersServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionInstanceGroupManagersHandlerServer(ctx, mux, &pb.UnimplementedRegionInstanceGroupManagersServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionInstanceGroups",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionInstanceGroupsServer(s, &pb.UnimplementedRegionInstanceGroupsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionInstanceGroupsHandlerServer(ctx, mux, &pb.UnimplementedRegionInstanceGroupsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionInstanceTemplates",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionInstanceTemplatesServer(s, &pb.UnimplementedRegionInstanceTemplatesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionInstanceTemplatesHandlerServer(ctx, mux, &pb.UnimplementedRegionInstanceTemplatesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionInstances",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionInstancesServer(s, &pb.UnimplementedRegionInstancesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionInstancesHandlerServer(ctx, mux, &pb.UnimplementedRegionInstancesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionInstantSnapshotGroups",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionInstantSnapshotGroupsServer(s, &pb.UnimplementedRegionInstantSnapshotGroupsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionInstantSnapshotGroupsHandlerServer(ctx, mux, &pb.UnimplementedRegionInstantSnapshotGroupsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionInstantSnapshots",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionInstantSnapshotsServer(s, &pb.UnimplementedRegionInstantSnapshotsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionInstantSnapshotsHandlerServer(ctx, mux, &pb.UnimplementedRegionInstantSnapshotsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionNetworkEndpointGroups",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionNetworkEndpointGroupsServer(s, &pb.UnimplementedRegionNetworkEndpointGroupsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionNetworkEndpointGroupsHandlerServer(ctx, mux, &pb.UnimplementedRegionNetworkEndpointGroupsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionNetworkFirewallPolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionNetworkFirewallPoliciesServer(s, &pb.UnimplementedRegionNetworkFirewallPoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionNetworkFirewallPoliciesHandlerServer(ctx, mux, &pb.UnimplementedRegionNetworkFirewallPoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionNotificationEndpoints",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionNotificationEndpointsServer(s, &pb.UnimplementedRegionNotificationEndpointsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionNotificationEndpointsHandlerServer(ctx, mux, &pb.UnimplementedRegionNotificationEndpointsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionOperations",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionOperationsServer(s, &pb.UnimplementedRegionOperationsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionOperationsHandlerServer(ctx, mux, &pb.UnimplementedRegionOperationsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionSecurityPolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionSecurityPoliciesServer(s, &pb.UnimplementedRegionSecurityPoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionSecurityPoliciesHandlerServer(ctx, mux, &pb.UnimplementedRegionSecurityPoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionSnapshotSettings",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionSnapshotSettingsServer(s, &pb.UnimplementedRegionSnapshotSettingsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionSnapshotSettingsHandlerServer(ctx, mux, &pb.UnimplementedRegionSnapshotSettingsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionSnapshots",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionSnapshotsServer(s, &pb.UnimplementedRegionSnapshotsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionSnapshotsHandlerServer(ctx, mux, &pb.UnimplementedRegionSnapshotsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionSslCertificates",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionSslCertificatesServer(s, &pb.UnimplementedRegionSslCertificatesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionSslCertificatesHandlerServer(ctx, mux, &pb.UnimplementedRegionSslCertificatesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionSslPolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionSslPoliciesServer(s, &pb.UnimplementedRegionSslPoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionSslPoliciesHandlerServer(ctx, mux, &pb.UnimplementedRegionSslPoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionTargetHttpProxies",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionTargetHttpProxiesServer(s, &pb.UnimplementedRegionTargetHttpProxiesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionTargetHttpProxiesHandlerServer(ctx, mux, &pb.UnimplementedRegionTargetHttpProxiesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionTargetHttpsProxies",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionTargetHttpsProxiesServer(s, &pb.UnimplementedRegionTargetHttpsProxiesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionTargetHttpsProxiesHandlerServer(ctx, mux, &pb.UnimplementedRegionTargetHttpsProxiesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionTargetTcpProxies",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionTargetTcpProxiesServer(s, &pb.UnimplementedRegionTargetTcpProxiesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionTargetTcpProxiesHandlerServer(ctx, mux, &pb.UnimplementedRegionTargetTcpProxiesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionUrlMaps",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionUrlMapsServer(s, &pb.UnimplementedRegionUrlMapsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionUrlMapsHandlerServer(ctx, mux, &pb.UnimplementedRegionUrlMapsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RegionZones",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionZonesServer(s, &pb.UnimplementedRegionZonesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionZonesHandlerServer(ctx, mux, &pb.UnimplementedRegionZonesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Regions",
		func(s grpc.ServiceRegistrar) { pb.RegisterRegionsServer(s, &pb.UnimplementedRegionsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRegionsHandlerServer(ctx, mux, &pb.UnimplementedRegionsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ReliabilityRisks",
		func(s grpc.ServiceRegistrar) { pb.RegisterReliabilityRisksServer(s, &pb.UnimplementedReliabilityRisksServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterReliabilityRisksHandlerServer(ctx, mux, &pb.UnimplementedReliabilityRisksServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ReservationBlocks",
		func(s grpc.ServiceRegistrar) { pb.RegisterReservationBlocksServer(s, &pb.UnimplementedReservationBlocksServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterReservationBlocksHandlerServer(ctx, mux, &pb.UnimplementedReservationBlocksServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ReservationSlots",
		func(s grpc.ServiceRegistrar) { pb.RegisterReservationSlotsServer(s, &pb.UnimplementedReservationSlotsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterReservationSlotsHandlerServer(ctx, mux, &pb.UnimplementedReservationSlotsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ReservationSubBlocks",
		func(s grpc.ServiceRegistrar) { pb.RegisterReservationSubBlocksServer(s, &pb.UnimplementedReservationSubBlocksServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterReservationSubBlocksHandlerServer(ctx, mux, &pb.UnimplementedReservationSubBlocksServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Reservations",
		func(s grpc.ServiceRegistrar) { pb.RegisterReservationsServer(s, &pb.UnimplementedReservationsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterReservationsHandlerServer(ctx, mux, &pb.UnimplementedReservationsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ResourcePolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterResourcePoliciesServer(s, &pb.UnimplementedResourcePoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterResourcePoliciesHandlerServer(ctx, mux, &pb.UnimplementedResourcePoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.RolloutPlans",
		func(s grpc.ServiceRegistrar) { pb.RegisterRolloutPlansServer(s, &pb.UnimplementedRolloutPlansServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRolloutPlansHandlerServer(ctx, mux, &pb.UnimplementedRolloutPlansServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Rollouts",
		func(s grpc.ServiceRegistrar) { pb.RegisterRolloutsServer(s, &pb.UnimplementedRolloutsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRolloutsHandlerServer(ctx, mux, &pb.UnimplementedRolloutsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Routers",
		func(s grpc.ServiceRegistrar) { pb.RegisterRoutersServer(s, &pb.UnimplementedRoutersServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRoutersHandlerServer(ctx, mux, &pb.UnimplementedRoutersServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Routes",
		func(s grpc.ServiceRegistrar) { pb.RegisterRoutesServer(s, &pb.UnimplementedRoutesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterRoutesHandlerServer(ctx, mux, &pb.UnimplementedRoutesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.SecurityPolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterSecurityPoliciesServer(s, &pb.UnimplementedSecurityPoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSecurityPoliciesHandlerServer(ctx, mux, &pb.UnimplementedSecurityPoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ServiceAttachments",
		func(s grpc.ServiceRegistrar) { pb.RegisterServiceAttachmentsServer(s, &pb.UnimplementedServiceAttachmentsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterServiceAttachmentsHandlerServer(ctx, mux, &pb.UnimplementedServiceAttachmentsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.SnapshotSettingsService",
		func(s grpc.ServiceRegistrar) { pb.RegisterSnapshotSettingsServiceServer(s, &pb.UnimplementedSnapshotSettingsServiceServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSnapshotSettingsServiceHandlerServer(ctx, mux, &pb.UnimplementedSnapshotSettingsServiceServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Snapshots",
		func(s grpc.ServiceRegistrar) { pb.RegisterSnapshotsServer(s, &pb.UnimplementedSnapshotsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSnapshotsHandlerServer(ctx, mux, &pb.UnimplementedSnapshotsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.SslCertificates",
		func(s grpc.ServiceRegistrar) { pb.RegisterSslCertificatesServer(s, &pb.UnimplementedSslCertificatesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSslCertificatesHandlerServer(ctx, mux, &pb.UnimplementedSslCertificatesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.SslPolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterSslPoliciesServer(s, &pb.UnimplementedSslPoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSslPoliciesHandlerServer(ctx, mux, &pb.UnimplementedSslPoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.StoragePoolTypes",
		func(s grpc.ServiceRegistrar) { pb.RegisterStoragePoolTypesServer(s, &pb.UnimplementedStoragePoolTypesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterStoragePoolTypesHandlerServer(ctx, mux, &pb.UnimplementedStoragePoolTypesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.StoragePools",
		func(s grpc.ServiceRegistrar) { pb.RegisterStoragePoolsServer(s, &pb.UnimplementedStoragePoolsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterStoragePoolsHandlerServer(ctx, mux, &pb.UnimplementedStoragePoolsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Subnetworks",
		func(s grpc.ServiceRegistrar) { pb.RegisterSubnetworksServer(s, &pb.UnimplementedSubnetworksServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterSubnetworksHandlerServer(ctx, mux, &pb.UnimplementedSubnetworksServer{}) },
	)
	registerEntry("google.cloud.compute.v1.TargetGrpcProxies",
		func(s grpc.ServiceRegistrar) { pb.RegisterTargetGrpcProxiesServer(s, &pb.UnimplementedTargetGrpcProxiesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTargetGrpcProxiesHandlerServer(ctx, mux, &pb.UnimplementedTargetGrpcProxiesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.TargetHttpProxies",
		func(s grpc.ServiceRegistrar) { pb.RegisterTargetHttpProxiesServer(s, &pb.UnimplementedTargetHttpProxiesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTargetHttpProxiesHandlerServer(ctx, mux, &pb.UnimplementedTargetHttpProxiesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.TargetHttpsProxies",
		func(s grpc.ServiceRegistrar) { pb.RegisterTargetHttpsProxiesServer(s, &pb.UnimplementedTargetHttpsProxiesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTargetHttpsProxiesHandlerServer(ctx, mux, &pb.UnimplementedTargetHttpsProxiesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.TargetInstances",
		func(s grpc.ServiceRegistrar) { pb.RegisterTargetInstancesServer(s, &pb.UnimplementedTargetInstancesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTargetInstancesHandlerServer(ctx, mux, &pb.UnimplementedTargetInstancesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.TargetPools",
		func(s grpc.ServiceRegistrar) { pb.RegisterTargetPoolsServer(s, &pb.UnimplementedTargetPoolsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTargetPoolsHandlerServer(ctx, mux, &pb.UnimplementedTargetPoolsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.TargetSslProxies",
		func(s grpc.ServiceRegistrar) { pb.RegisterTargetSslProxiesServer(s, &pb.UnimplementedTargetSslProxiesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTargetSslProxiesHandlerServer(ctx, mux, &pb.UnimplementedTargetSslProxiesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.TargetTcpProxies",
		func(s grpc.ServiceRegistrar) { pb.RegisterTargetTcpProxiesServer(s, &pb.UnimplementedTargetTcpProxiesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTargetTcpProxiesHandlerServer(ctx, mux, &pb.UnimplementedTargetTcpProxiesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.TargetVpnGateways",
		func(s grpc.ServiceRegistrar) { pb.RegisterTargetVpnGatewaysServer(s, &pb.UnimplementedTargetVpnGatewaysServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterTargetVpnGatewaysHandlerServer(ctx, mux, &pb.UnimplementedTargetVpnGatewaysServer{}) },
	)
	registerEntry("google.cloud.compute.v1.UrlMaps",
		func(s grpc.ServiceRegistrar) { pb.RegisterUrlMapsServer(s, &pb.UnimplementedUrlMapsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterUrlMapsHandlerServer(ctx, mux, &pb.UnimplementedUrlMapsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.VpnGateways",
		func(s grpc.ServiceRegistrar) { pb.RegisterVpnGatewaysServer(s, &pb.UnimplementedVpnGatewaysServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterVpnGatewaysHandlerServer(ctx, mux, &pb.UnimplementedVpnGatewaysServer{}) },
	)
	registerEntry("google.cloud.compute.v1.VpnTunnels",
		func(s grpc.ServiceRegistrar) { pb.RegisterVpnTunnelsServer(s, &pb.UnimplementedVpnTunnelsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterVpnTunnelsHandlerServer(ctx, mux, &pb.UnimplementedVpnTunnelsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.WireGroups",
		func(s grpc.ServiceRegistrar) { pb.RegisterWireGroupsServer(s, &pb.UnimplementedWireGroupsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterWireGroupsHandlerServer(ctx, mux, &pb.UnimplementedWireGroupsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ZoneOperations",
		func(s grpc.ServiceRegistrar) { pb.RegisterZoneOperationsServer(s, &pb.UnimplementedZoneOperationsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterZoneOperationsHandlerServer(ctx, mux, &pb.UnimplementedZoneOperationsServer{}) },
	)
	registerEntry("google.cloud.compute.v1.ZoneVmExtensionPolicies",
		func(s grpc.ServiceRegistrar) { pb.RegisterZoneVmExtensionPoliciesServer(s, &pb.UnimplementedZoneVmExtensionPoliciesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterZoneVmExtensionPoliciesHandlerServer(ctx, mux, &pb.UnimplementedZoneVmExtensionPoliciesServer{}) },
	)
	registerEntry("google.cloud.compute.v1.Zones",
		func(s grpc.ServiceRegistrar) { pb.RegisterZonesServer(s, &pb.UnimplementedZonesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error { return pb.RegisterZonesHandlerServer(ctx, mux, &pb.UnimplementedZonesServer{}) },
	)
}
