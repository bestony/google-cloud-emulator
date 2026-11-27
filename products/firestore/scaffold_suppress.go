package firestore

import "github.com/bestony/google-cloud-emulator/products/scaffold"

func init() {
	scaffold.SuppressServices("google.firestore.v1.Firestore")
}
