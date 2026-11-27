package scaffold

import (
	"google.golang.org/grpc"

	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/bestony/google-cloud-emulator/gen/google/appengine/v1"
)

func init() {
	registerEntry("google.appengine.v1.Applications",
		func(s grpc.ServiceRegistrar) { pb.RegisterApplicationsServer(s, &pb.UnimplementedApplicationsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterApplicationsHandlerServer(ctx, mux, &pb.UnimplementedApplicationsServer{})
		},
	)
	registerEntry("google.appengine.v1.Services",
		func(s grpc.ServiceRegistrar) { pb.RegisterServicesServer(s, &pb.UnimplementedServicesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterServicesHandlerServer(ctx, mux, &pb.UnimplementedServicesServer{})
		},
	)
	registerEntry("google.appengine.v1.Versions",
		func(s grpc.ServiceRegistrar) { pb.RegisterVersionsServer(s, &pb.UnimplementedVersionsServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterVersionsHandlerServer(ctx, mux, &pb.UnimplementedVersionsServer{})
		},
	)
	registerEntry("google.appengine.v1.Instances",
		func(s grpc.ServiceRegistrar) { pb.RegisterInstancesServer(s, &pb.UnimplementedInstancesServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterInstancesHandlerServer(ctx, mux, &pb.UnimplementedInstancesServer{})
		},
	)
	registerEntry("google.appengine.v1.Firewall",
		func(s grpc.ServiceRegistrar) { pb.RegisterFirewallServer(s, &pb.UnimplementedFirewallServer{}) },
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterFirewallHandlerServer(ctx, mux, &pb.UnimplementedFirewallServer{})
		},
	)
	registerEntry("google.appengine.v1.AuthorizedDomains",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterAuthorizedDomainsServer(s, &pb.UnimplementedAuthorizedDomainsServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterAuthorizedDomainsHandlerServer(ctx, mux, &pb.UnimplementedAuthorizedDomainsServer{})
		},
	)
	registerEntry("google.appengine.v1.AuthorizedCertificates",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterAuthorizedCertificatesServer(s, &pb.UnimplementedAuthorizedCertificatesServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterAuthorizedCertificatesHandlerServer(ctx, mux, &pb.UnimplementedAuthorizedCertificatesServer{})
		},
	)
	registerEntry("google.appengine.v1.DomainMappings",
		func(s grpc.ServiceRegistrar) {
			pb.RegisterDomainMappingsServer(s, &pb.UnimplementedDomainMappingsServer{})
		},
		func(ctx context.Context, mux *runtime.ServeMux) error {
			return pb.RegisterDomainMappingsHandlerServer(ctx, mux, &pb.UnimplementedDomainMappingsServer{})
		},
	)
}
