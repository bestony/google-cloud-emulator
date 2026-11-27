package compute

import "github.com/bestony/google-cloud-emulator/products/scaffold"

func init() {
	scaffold.SuppressPrefix("google.cloud.compute.v1.")
}
