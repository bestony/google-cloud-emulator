package compute

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	computepb "github.com/bestony/google-cloud-emulator/gen/google/cloud/compute/v1"
)

// Register wires the implemented Compute Engine services into the gRPC
// server and REST gateway. Core resources use hand-written behavior; the
// remaining control-plane services use the stateful generic implementation.
func Register(grpcServer *grpc.Server, gateway *runtime.ServeMux, state *State) error {
	ctx := context.Background()

	instances := &instancesServer{state: state}
	disks := &disksServer{state: state}
	networks := &networksServer{state: state}
	subnetworks := &subnetworksServer{state: state}
	firewalls := &firewallsServer{state: state}
	addresses := &addressesServer{state: state}
	images := &imagesServer{state: state}
	snapshots := &snapshotsServer{state: state}
	templates := &instanceTemplatesServer{state: state}
	healthChecks := &healthChecksServer{state: state}
	machineTypes := &machineTypesServer{state: state}
	zones := &zonesServer{state: state}
	regions := &regionsServer{state: state}
	globalOps := &globalOperationsServer{state: state}
	zoneOps := &zoneOperationsServer{state: state}
	regionOps := &regionOperationsServer{state: state}

	computepb.RegisterInstancesServer(grpcServer, instances)
	computepb.RegisterDisksServer(grpcServer, disks)
	computepb.RegisterNetworksServer(grpcServer, networks)
	computepb.RegisterSubnetworksServer(grpcServer, subnetworks)
	computepb.RegisterFirewallsServer(grpcServer, firewalls)
	computepb.RegisterAddressesServer(grpcServer, addresses)
	computepb.RegisterImagesServer(grpcServer, images)
	computepb.RegisterSnapshotsServer(grpcServer, snapshots)
	computepb.RegisterInstanceTemplatesServer(grpcServer, templates)
	computepb.RegisterHealthChecksServer(grpcServer, healthChecks)
	computepb.RegisterMachineTypesServer(grpcServer, machineTypes)
	computepb.RegisterZonesServer(grpcServer, zones)
	computepb.RegisterRegionsServer(grpcServer, regions)
	computepb.RegisterGlobalOperationsServer(grpcServer, globalOps)
	computepb.RegisterZoneOperationsServer(grpcServer, zoneOps)
	computepb.RegisterRegionOperationsServer(grpcServer, regionOps)

	if err := computepb.RegisterInstancesHandlerServer(ctx, gateway, instances); err != nil {
		return err
	}
	if err := computepb.RegisterDisksHandlerServer(ctx, gateway, disks); err != nil {
		return err
	}
	if err := computepb.RegisterNetworksHandlerServer(ctx, gateway, networks); err != nil {
		return err
	}
	if err := computepb.RegisterSubnetworksHandlerServer(ctx, gateway, subnetworks); err != nil {
		return err
	}
	if err := computepb.RegisterFirewallsHandlerServer(ctx, gateway, firewalls); err != nil {
		return err
	}
	if err := computepb.RegisterAddressesHandlerServer(ctx, gateway, addresses); err != nil {
		return err
	}
	if err := computepb.RegisterImagesHandlerServer(ctx, gateway, images); err != nil {
		return err
	}
	if err := computepb.RegisterSnapshotsHandlerServer(ctx, gateway, snapshots); err != nil {
		return err
	}
	if err := computepb.RegisterInstanceTemplatesHandlerServer(ctx, gateway, templates); err != nil {
		return err
	}
	if err := computepb.RegisterHealthChecksHandlerServer(ctx, gateway, healthChecks); err != nil {
		return err
	}
	if err := computepb.RegisterMachineTypesHandlerServer(ctx, gateway, machineTypes); err != nil {
		return err
	}
	if err := computepb.RegisterZonesHandlerServer(ctx, gateway, zones); err != nil {
		return err
	}
	if err := computepb.RegisterRegionsHandlerServer(ctx, gateway, regions); err != nil {
		return err
	}
	if err := computepb.RegisterGlobalOperationsHandlerServer(ctx, gateway, globalOps); err != nil {
		return err
	}
	if err := computepb.RegisterZoneOperationsHandlerServer(ctx, gateway, zoneOps); err != nil {
		return err
	}
	if err := computepb.RegisterRegionOperationsHandlerServer(ctx, gateway, regionOps); err != nil {
		return err
	}
	return registerGenericComputeServices(grpcServer, gateway, state)
}
