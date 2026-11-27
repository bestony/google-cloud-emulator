package secretmanager

import (
	smpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/secretmanager/v1"
)

// OperationsServer implements SecretOperationsServiceServer. The backup
// surface (ListBackups/GetBackup/CreateBackup/UpdateBackup/DeleteBackup/
// RestoreBackup) is not implemented by the emulator yet; every method
// inherits the generated UNIMPLEMENTED response.
type OperationsServer struct {
	smpb.UnimplementedSecretOperationsServiceServer
}

// NewOperationsServer returns an OperationsServer.
func NewOperationsServer() *OperationsServer { return &OperationsServer{} }
