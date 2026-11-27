package datastore

import "github.com/bestony/google-cloud-emulator/products/scaffold"

func init() {
	scaffold.SuppressServices("google.datastore.v1.Datastore")
}
