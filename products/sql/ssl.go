package sql

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	sqlpb "github.com/bestony/google-cloud-emulator/gen/google/cloud/sql/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func sslCertSelfLink(project, instance, fingerprint string) string {
	return fmt.Sprintf("%s/projects/%s/instances/%s/sslCerts/%s", selfLinkBase, project, instance, fingerprint)
}

func makeCertificate(commonName, instance string, serial uint64, lifetime time.Duration) (*sqlpb.SslCert, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", err
	}
	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber:          new(big.Int).SetUint64(serial + 1),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(lifetime),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, "", err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, "", err
	}
	fingerprint := sha1.Sum(der) //nolint:govet // SHA-1 is part of the Cloud SQL certificate identity API.
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return &sqlpb.SslCert{
		Kind:             "sql#sslCert",
		CertSerialNumber: template.SerialNumber.String(),
		Cert:             string(certPEM),
		CreateTime:       timestamppb.New(now),
		CommonName:       commonName,
		ExpirationTime:   timestamppb.New(template.NotAfter),
		Sha1Fingerprint:  hex.EncodeToString(fingerprint[:]),
		Instance:         instance,
	}, string(keyPEM), nil
}

func (s *State) ensureServerCaLocked(project, instance string) (*sqlpb.SslCert, error) {
	key := sslInstanceKey(project, instance)
	if ca := s.serverCaCerts[key]; ca != nil {
		return ca, nil
	}
	ca, _, err := makeCertificate(instance+" server-ca", instance, s.nextIDLocked(), 10*365*24*time.Hour)
	if err != nil {
		return nil, err
	}
	ca.SelfLink = sslCertSelfLink(project, instance, ca.Sha1Fingerprint)
	s.serverCaCerts[key] = ca
	return ca, nil
}

type sslCertsServer struct {
	sqlpb.UnimplementedSqlSslCertsServiceServer
	state *State
}

func (s *sslCertsServer) requireInstance(project, instance string) error {
	if project == "" || instance == "" {
		return invalidArg("project and instance are required")
	}
	return requireInstance(s.state, project, instance)
}

func (s *sslCertsServer) Insert(_ context.Context, req *sqlpb.SqlSslCertsInsertRequest) (*sqlpb.SslCertsInsertResponse, error) {
	if err := s.requireInstance(req.Project, req.Instance); err != nil {
		return nil, err
	}
	commonName := req.GetBody().GetCommonName()
	if commonName == "" {
		return nil, invalidArg("common_name is required")
	}
	s.state.mu.RLock()
	prefix := sslInstanceKey(req.Project, req.Instance) + "|"
	for key, cert := range s.state.sslCerts {
		if strings.HasPrefix(key, prefix) && cert.CommonName == commonName {
			s.state.mu.RUnlock()
			return nil, alreadyExists("ssl cert", req.Project, commonName)
		}
	}
	s.state.mu.RUnlock()

	s.state.mu.Lock()
	serial := s.state.nextIDLocked()
	s.state.mu.Unlock()
	cert, privateKey, err := makeCertificate(commonName, req.Instance, serial, 365*24*time.Hour)
	if err != nil {
		return nil, err
	}
	cert.SelfLink = sslCertSelfLink(req.Project, req.Instance, cert.Sha1Fingerprint)

	s.state.mu.Lock()
	s.state.sslCerts[sslCertKey(req.Project, req.Instance, cert.Sha1Fingerprint)] = cert
	ca, err := s.state.ensureServerCaLocked(req.Project, req.Instance)
	if err != nil {
		s.state.mu.Unlock()
		return nil, err
	}
	op := s.state.addOperation(req.Project, sqlpb.Operation_UPDATE, req.Instance, instanceSelfLink(req.Project, req.Instance))
	s.state.mu.Unlock()
	return &sqlpb.SslCertsInsertResponse{
		Kind:         "sql#sslCertsInsert",
		Operation:    op,
		ServerCaCert: proto.Clone(ca).(*sqlpb.SslCert),
		ClientCert:   &sqlpb.SslCertDetail{CertInfo: cert, CertPrivateKey: privateKey},
	}, nil
}

func (s *sslCertsServer) Get(_ context.Context, req *sqlpb.SqlSslCertsGetRequest) (*sqlpb.SslCert, error) {
	if err := s.requireInstance(req.Project, req.Instance); err != nil {
		return nil, err
	}
	s.state.mu.RLock()
	cert := s.state.sslCerts[sslCertKey(req.Project, req.Instance, req.Sha1Fingerprint)]
	s.state.mu.RUnlock()
	if cert == nil {
		return nil, notFound("ssl cert", req.Project, req.Sha1Fingerprint)
	}
	return proto.Clone(cert).(*sqlpb.SslCert), nil
}

func (s *sslCertsServer) List(_ context.Context, req *sqlpb.SqlSslCertsListRequest) (*sqlpb.SslCertsListResponse, error) {
	if err := s.requireInstance(req.Project, req.Instance); err != nil {
		return nil, err
	}
	prefix := sslInstanceKey(req.Project, req.Instance) + "|"
	s.state.mu.RLock()
	var items []*sqlpb.SslCert
	for key, cert := range s.state.sslCerts {
		if strings.HasPrefix(key, prefix) {
			items = append(items, proto.Clone(cert).(*sqlpb.SslCert))
		}
	}
	s.state.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Sha1Fingerprint < items[j].Sha1Fingerprint })
	return &sqlpb.SslCertsListResponse{Kind: "sql#sslCertsList", Items: items}, nil
}

func (s *sslCertsServer) Delete(_ context.Context, req *sqlpb.SqlSslCertsDeleteRequest) (*sqlpb.Operation, error) {
	if err := s.requireInstance(req.Project, req.Instance); err != nil {
		return nil, err
	}
	key := sslCertKey(req.Project, req.Instance, req.Sha1Fingerprint)
	s.state.mu.Lock()
	if s.state.sslCerts[key] == nil {
		s.state.mu.Unlock()
		return nil, notFound("ssl cert", req.Project, req.Sha1Fingerprint)
	}
	delete(s.state.sslCerts, key)
	op := s.state.addOperation(req.Project, sqlpb.Operation_UPDATE, req.Instance, instanceSelfLink(req.Project, req.Instance))
	s.state.mu.Unlock()
	return op, nil
}

func (s *sslCertsServer) UpdateCustomerManagedCertificate(_ context.Context, req *sqlpb.UpdateCustomerManagedCertificateRequest) (*sqlpb.Operation, error) {
	if err := s.requireInstance(req.Project, req.Instance); err != nil {
		return nil, err
	}
	if req.CertificateBundle == "" || req.CustomerManagedCertificateConfig == nil {
		return nil, invalidArg("certificate_bundle and customer_managed_certificate_config are required")
	}
	key := customerCertificateKey(req.Project, req.Instance, req.CertificateBundle)
	s.state.mu.Lock()
	if s.state.customerCertificates[key] == nil && !req.AllowMissing {
		s.state.mu.Unlock()
		return nil, notFound("customer managed certificate", req.Project, req.CertificateBundle)
	}
	cfg := proto.Clone(req.CustomerManagedCertificateConfig).(*sqlpb.CustomerManagedCertificateConfig)
	cfg.Bundle = req.CertificateBundle
	s.state.customerCertificates[key] = cfg
	op := s.state.addOperation(req.Project, sqlpb.Operation_UPDATE, req.Instance, instanceSelfLink(req.Project, req.Instance))
	s.state.mu.Unlock()
	return op, nil
}

func (s *sslCertsServer) DeleteCustomerManagedCertificate(_ context.Context, req *sqlpb.DeleteCustomerManagedCertificateRequest) (*sqlpb.Operation, error) {
	if err := s.requireInstance(req.Project, req.Instance); err != nil {
		return nil, err
	}
	key := customerCertificateKey(req.Project, req.Instance, req.CertificateBundle)
	s.state.mu.Lock()
	if s.state.customerCertificates[key] == nil {
		s.state.mu.Unlock()
		return nil, notFound("customer managed certificate", req.Project, req.CertificateBundle)
	}
	delete(s.state.customerCertificates, key)
	op := s.state.addOperation(req.Project, sqlpb.Operation_DELETE, req.Instance, instanceSelfLink(req.Project, req.Instance))
	s.state.mu.Unlock()
	return op, nil
}

func (s *sslCertsServer) ListCustomerManagedCertificates(_ context.Context, req *sqlpb.ListCustomerManagedCertificatesRequest) (*sqlpb.ListCustomerManagedCertificatesResponse, error) {
	if err := s.requireInstance(req.Project, req.Instance); err != nil {
		return nil, err
	}
	prefix := sslInstanceKey(req.Project, req.Instance) + "|"
	s.state.mu.RLock()
	var items []*sqlpb.CustomerManagedCertificateConfig
	for key, cfg := range s.state.customerCertificates {
		if strings.HasPrefix(key, prefix) {
			items = append(items, proto.Clone(cfg).(*sqlpb.CustomerManagedCertificateConfig))
		}
	}
	s.state.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Bundle < items[j].Bundle })
	return &sqlpb.ListCustomerManagedCertificatesResponse{CustomerManagedCertificates: items}, nil
}

func (s *instancesServer) instanceOperation(project, instance string, opType sqlpb.Operation_SqlOperationType, mutate func(*sqlpb.DatabaseInstance)) (*sqlpb.Operation, error) {
	if project == "" || instance == "" {
		return nil, invalidArg("project and instance are required")
	}
	key := instanceKey(project, instance)
	s.state.mu.Lock()
	inst := s.state.instances[key]
	if inst == nil {
		s.state.mu.Unlock()
		return nil, notFound("instance", project, instance)
	}
	if mutate != nil {
		mutate(inst)
		inst.Etag = fmt.Sprint(s.state.nextIDLocked())
	}
	op := s.state.addOperation(project, opType, instance, inst.SelfLink)
	s.state.mu.Unlock()
	return op, nil
}

func (s *instancesServer) Restart(_ context.Context, req *sqlpb.SqlInstancesRestartRequest) (*sqlpb.Operation, error) {
	return s.instanceOperation(req.Project, req.Instance, sqlpb.Operation_RESTART, func(inst *sqlpb.DatabaseInstance) {
		inst.State = sqlpb.DatabaseInstance_RUNNABLE
	})
}

func (s *instancesServer) Start(_ context.Context, req *sqlpb.SqlInstancesStartRequest) (*sqlpb.Operation, error) {
	return s.instanceOperation(req.Project, req.Instance, sqlpb.Operation_UPDATE, func(inst *sqlpb.DatabaseInstance) {
		inst.State = sqlpb.DatabaseInstance_RUNNABLE
	})
}

func (s *instancesServer) Stop(_ context.Context, req *sqlpb.SqlInstancesStopRequest) (*sqlpb.Operation, error) {
	return s.instanceOperation(req.Project, req.Instance, sqlpb.Operation_UPDATE, func(inst *sqlpb.DatabaseInstance) {
		inst.State = sqlpb.DatabaseInstance_SUSPENDED
	})
}

func (s *instancesServer) ResetSslConfig(_ context.Context, req *sqlpb.SqlInstancesResetSslConfigRequest) (*sqlpb.Operation, error) {
	if err := requireInstance(s.state, req.Project, req.Instance); err != nil {
		return nil, err
	}
	s.state.mu.Lock()
	serial := s.state.nextIDLocked()
	s.state.mu.Unlock()
	ca, _, err := makeCertificate(req.Instance+" server-ca", req.Instance, serial, 10*365*24*time.Hour)
	if err != nil {
		return nil, err
	}
	ca.SelfLink = sslCertSelfLink(req.Project, req.Instance, ca.Sha1Fingerprint)
	instKey := sslInstanceKey(req.Project, req.Instance)
	prefix := instKey + "|"
	s.state.mu.Lock()
	for key := range s.state.sslCerts {
		if strings.HasPrefix(key, prefix) {
			delete(s.state.sslCerts, key)
		}
	}
	s.state.serverCaCerts[instKey] = ca
	op := s.state.addOperation(req.Project, sqlpb.Operation_UPDATE, req.Instance, instanceSelfLink(req.Project, req.Instance))
	s.state.mu.Unlock()
	return op, nil
}

func (s *instancesServer) RotateServerCa(_ context.Context, req *sqlpb.SqlInstancesRotateServerCaRequest) (*sqlpb.Operation, error) {
	if err := requireInstance(s.state, req.Project, req.Instance); err != nil {
		return nil, err
	}
	s.state.mu.Lock()
	serial := s.state.nextIDLocked()
	s.state.mu.Unlock()
	ca, _, err := makeCertificate(req.Instance+" server-ca", req.Instance, serial, 10*365*24*time.Hour)
	if err != nil {
		return nil, err
	}
	ca.SelfLink = sslCertSelfLink(req.Project, req.Instance, ca.Sha1Fingerprint)
	s.state.mu.Lock()
	s.state.serverCaCerts[sslInstanceKey(req.Project, req.Instance)] = ca
	op := s.state.addOperation(req.Project, sqlpb.Operation_UPDATE, req.Instance, instanceSelfLink(req.Project, req.Instance))
	s.state.mu.Unlock()
	return op, nil
}

func (s *instancesServer) CreateEphemeral(_ context.Context, req *sqlpb.SqlInstancesCreateEphemeralCertRequest) (*sqlpb.SslCert, error) {
	if err := requireInstance(s.state, req.Project, req.Instance); err != nil {
		return nil, err
	}
	name := "ephemeral"
	if req.GetBody().GetAccessToken() != "" {
		name = "ephemeral-access-token"
	}
	s.state.mu.Lock()
	serial := s.state.nextIDLocked()
	s.state.mu.Unlock()
	cert, _, err := makeCertificate(name, req.Instance, serial, 10*time.Minute)
	if err != nil {
		return nil, err
	}
	cert.SelfLink = sslCertSelfLink(req.Project, req.Instance, cert.Sha1Fingerprint)
	return cert, nil
}

func (s *instancesServer) Clone(_ context.Context, req *sqlpb.SqlInstancesCloneRequest) (*sqlpb.Operation, error) {
	if req.Project == "" || req.Instance == "" {
		return nil, invalidArg("project and instance are required")
	}
	destination := req.GetBody().GetCloneContext().GetDestinationInstanceName()
	if destination == "" {
		return nil, invalidArg("destination instance name is required")
	}
	sourceKey := instanceKey(req.Project, req.Instance)
	destKey := instanceKey(req.Project, destination)
	s.state.mu.Lock()
	source := s.state.instances[sourceKey]
	if source == nil {
		s.state.mu.Unlock()
		return nil, notFound("instance", req.Project, req.Instance)
	}
	if s.state.instances[destKey] != nil {
		s.state.mu.Unlock()
		return nil, alreadyExists("instance", req.Project, destination)
	}
	clone := proto.Clone(source).(*sqlpb.DatabaseInstance)
	clone.Name = destination
	clone.SelfLink = instanceSelfLink(req.Project, destination)
	clone.Etag = fmt.Sprint(s.state.nextIDLocked())
	clone.CreateTime = timestamppb.Now()
	clone.State = sqlpb.DatabaseInstance_RUNNABLE
	s.state.instances[destKey] = clone
	op := s.state.addOperation(req.Project, sqlpb.Operation_CLONE, destination, clone.SelfLink)
	s.state.mu.Unlock()
	return op, nil
}

func (s *instancesServer) Failover(_ context.Context, req *sqlpb.SqlInstancesFailoverRequest) (*sqlpb.Operation, error) {
	return s.instanceOperation(req.Project, req.Instance, sqlpb.Operation_FAILOVER, nil)
}

func (s *instancesServer) PromoteReplica(_ context.Context, req *sqlpb.SqlInstancesPromoteReplicaRequest) (*sqlpb.Operation, error) {
	return s.instanceOperation(req.Project, req.Instance, sqlpb.Operation_PROMOTE_REPLICA, func(inst *sqlpb.DatabaseInstance) {
		inst.InstanceType = sqlpb.SqlInstanceType_CLOUD_SQL_INSTANCE
		inst.MasterInstanceName = ""
	})
}

func (s *instancesServer) RestoreBackup(_ context.Context, req *sqlpb.SqlInstancesRestoreBackupRequest) (*sqlpb.Operation, error) {
	return s.instanceOperation(req.Project, req.Instance, sqlpb.Operation_RESTORE_VOLUME, nil)
}
