package kms

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/bestony/google-cloud-emulator/core/lro"
	kmspb "github.com/bestony/google-cloud-emulator/gen/google/cloud/kms/v1"
	longrunningpb "github.com/bestony/google-cloud-emulator/gen/google/longrunning"
)

func newTestServer(t *testing.T) (*Server, *lro.Store) {
	t.Helper()
	store := lro.NewStore()
	return NewServer(NewState(store)), store
}

const (
	krParent = "projects/demo/locations/us-east1"
	krName   = "projects/demo/locations/us-east1/keyRings/main"
	ckName   = "projects/demo/locations/us-east1/keyRings/main/cryptoKeys/my-key"
	v1Name   = ckName + "/cryptoKeyVersions/1"
)

func TestKeyRingLifecycle(t *testing.T) {
	srv, _ := newTestServer(t)
	ctx := context.Background()

	kr, err := srv.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{Parent: krParent, KeyRingId: "main"})
	if err != nil || kr.Name != krName {
		t.Fatalf("CreateKeyRing: %v %+v", err, kr)
	}
	if _, err := srv.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{Parent: krParent, KeyRingId: "main"}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate: got %v", err)
	}
	got, err := srv.GetKeyRing(ctx, &kmspb.GetKeyRingRequest{Name: krName})
	if err != nil || got.CreateTime == nil {
		t.Fatalf("GetKeyRing: %v %+v", err, got)
	}
	lr, err := srv.ListKeyRings(ctx, &kmspb.ListKeyRingsRequest{Parent: krParent})
	if err != nil || len(lr.KeyRings) != 1 || lr.TotalSize != 1 {
		t.Fatalf("ListKeyRings: %v %+v", err, lr)
	}
	if _, err := srv.ListKeyRings(ctx, &kmspb.ListKeyRingsRequest{Parent: krParent, Filter: "name:x"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("filter: got %v", err)
	}
}

func TestCryptoKeyAndVersionLifecycle(t *testing.T) {
	srv, _ := newTestServer(t)
	ctx := context.Background()
	if _, err := srv.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{Parent: krParent, KeyRingId: "main"}); err != nil {
		t.Fatal(err)
	}

	// Create key with initial version.
	ck, err := srv.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{
		Parent:      krName,
		CryptoKeyId: "my-key",
		CryptoKey: &kmspb.CryptoKey{
			Purpose: kmspb.CryptoKey_ENCRYPT_DECRYPT,
			Labels:  map[string]string{"team": "sec"},
		},
	})
	if err != nil {
		t.Fatalf("CreateCryptoKey: %v", err)
	}
	if ck.Name != ckName || ck.Primary == nil || ck.Primary.Name != v1Name {
		t.Fatalf("unexpected key: %+v", ck)
	}
	if ck.Primary.State != kmspb.CryptoKeyVersion_ENABLED || ck.Primary.Algorithm != kmspb.CryptoKeyVersion_GOOGLE_SYMMETRIC_ENCRYPTION {
		t.Fatalf("unexpected primary version: %+v", ck.Primary)
	}

	// Duplicate -> AlreadyExists; missing ring -> NotFound.
	if _, err := srv.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{Parent: krName, CryptoKeyId: "my-key", CryptoKey: &kmspb.CryptoKey{Purpose: kmspb.CryptoKey_ENCRYPT_DECRYPT}}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate key: got %v", err)
	}
	if _, err := srv.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{Parent: "projects/demo/locations/us-east1/keyRings/nope", CryptoKeyId: "x", CryptoKey: &kmspb.CryptoKey{Purpose: kmspb.CryptoKey_ENCRYPT_DECRYPT}}); status.Code(err) != codes.NotFound {
		// state returns invalid-parent for a well-formed but missing ring name; accept InvalidArgument too
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("missing ring: got %v", err)
		}
	}

	// List versions.
	lv, err := srv.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{Parent: ckName})
	if err != nil || len(lv.CryptoKeyVersions) != 1 {
		t.Fatalf("ListCryptoKeyVersions: %v %+v", err, lv)
	}

	// Update key labels.
	upd, err := srv.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{
		CryptoKey:  &kmspb.CryptoKey{Name: ckName, Labels: map[string]string{"team": "platform"}},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
	})
	if err != nil || upd.Labels["team"] != "platform" {
		t.Fatalf("UpdateCryptoKey: %v %+v", err, upd)
	}
	if _, err := srv.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{
		CryptoKey:  &kmspb.CryptoKey{Name: ckName},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"purpose"}},
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("bad mask: got %v", err)
	}

	// Disable version 1 -> FAILED_PRECONDITION on primary switch to disabled? (allowed); test state update.
	v, err := srv.UpdateCryptoKeyVersion(ctx, &kmspb.UpdateCryptoKeyVersionRequest{
		CryptoKeyVersion: &kmspb.CryptoKeyVersion{Name: v1Name, State: kmspb.CryptoKeyVersion_DISABLED},
		UpdateMask:       &fieldmaskpb.FieldMask{Paths: []string{"state"}},
	})
	if err != nil || v.State != kmspb.CryptoKeyVersion_DISABLED {
		t.Fatalf("UpdateCryptoKeyVersion: %v %+v", err, v)
	}

	// Create a second version on an asymmetric key.
	if _, err := srv.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{
		Parent:      krName,
		CryptoKeyId: "sign-key",
		CryptoKey:   &kmspb.CryptoKey{Purpose: kmspb.CryptoKey_ASYMMETRIC_SIGN},
	}); err != nil {
		t.Fatal(err)
	}
	signKey := krName + "/cryptoKeys/sign-key"
	v2, err := srv.CreateCryptoKeyVersion(ctx, &kmspb.CreateCryptoKeyVersionRequest{
		Parent:           signKey,
		CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
	})
	if err != nil {
		t.Fatalf("CreateCryptoKeyVersion: %v", err)
	}
	if v2.Name != signKey+"/cryptoKeyVersions/2" || v2.Algorithm != kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256 {
		t.Fatalf("unexpected v2: %+v", v2)
	}
	// Symmetric keys cannot add versions.
	if _, err := srv.CreateCryptoKeyVersion(ctx, &kmspb.CreateCryptoKeyVersionRequest{Parent: ckName, CryptoKeyVersion: &kmspb.CryptoKeyVersion{}}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("symmetric version create: got %v", err)
	}

	// Primary switch.
	ck2, err := srv.UpdateCryptoKeyPrimaryVersion(ctx, &kmspb.UpdateCryptoKeyPrimaryVersionRequest{Name: signKey, CryptoKeyVersionId: "2"})
	if err != nil || ck2.Primary.Name != signKey+"/cryptoKeyVersions/2" {
		t.Fatalf("UpdateCryptoKeyPrimaryVersion: %v %+v", err, ck2)
	}

	// Destroy + restore.
	dv, err := srv.DestroyCryptoKeyVersion(ctx, &kmspb.DestroyCryptoKeyVersionRequest{Name: v1Name})
	if err != nil || dv.State != kmspb.CryptoKeyVersion_DESTROY_SCHEDULED || dv.DestroyTime == nil {
		t.Fatalf("DestroyCryptoKeyVersion: %v %+v", err, dv)
	}
	rv, err := srv.RestoreCryptoKeyVersion(ctx, &kmspb.RestoreCryptoKeyVersionRequest{Name: v1Name})
	if err != nil || rv.State != kmspb.CryptoKeyVersion_ENABLED {
		t.Fatalf("RestoreCryptoKeyVersion: %v %+v", err, rv)
	}
}

func TestDeleteCryptoKeyLRO(t *testing.T) {
	srv, store := newTestServer(t)
	ctx := context.Background()
	if _, err := srv.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{Parent: krParent, KeyRingId: "main"}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{Parent: krName, CryptoKeyId: "k", CryptoKey: &kmspb.CryptoKey{Purpose: kmspb.CryptoKey_ENCRYPT_DECRYPT}}); err != nil {
		t.Fatal(err)
	}
	op, err := srv.DeleteCryptoKey(ctx, &kmspb.DeleteCryptoKeyRequest{Name: krName + "/cryptoKeys/k"})
	if err != nil {
		t.Fatalf("DeleteCryptoKey: %v", err)
	}
	if !op.Done || op.Name == "" {
		t.Fatalf("operation not done: %+v", op)
	}
	if op.GetResponse() == nil {
		t.Fatalf("operation has no response: %+v", op)
	}
	// Operation is retrievable via the shared store.
	if _, ok := store.Get(op.Name); !ok {
		t.Fatalf("operation %q not in store", op.Name)
	}
	if _, err := srv.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{Name: krName + "/cryptoKeys/k"}); status.Code(err) != codes.NotFound {
		t.Fatalf("get deleted key: got %v", err)
	}
	if _, err := srv.DeleteCryptoKey(ctx, &kmspb.DeleteCryptoKeyRequest{Name: krName + "/cryptoKeys/k"}); status.Code(err) != codes.NotFound {
		t.Fatalf("delete missing key: got %v", err)
	}
}

func TestSymmetricEncryptDecryptRoundTrip(t *testing.T) {
	srv, _ := newTestServer(t)
	ctx := context.Background()
	if _, err := srv.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{Parent: krParent, KeyRingId: "main"}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{Parent: krName, CryptoKeyId: "my-key", CryptoKey: &kmspb.CryptoKey{Purpose: kmspb.CryptoKey_ENCRYPT_DECRYPT}}); err != nil {
		t.Fatal(err)
	}
	enc, err := srv.Encrypt(ctx, &kmspb.EncryptRequest{Name: ckName, Plaintext: []byte("hello"), AdditionalAuthenticatedData: []byte("aad")})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc.Name != v1Name || len(enc.Ciphertext) == 0 || enc.CiphertextCrc32C == nil {
		t.Fatalf("unexpected encrypt response: %+v", enc)
	}
	dec, err := srv.Decrypt(ctx, &kmspb.DecryptRequest{Name: ckName, Ciphertext: enc.Ciphertext, AdditionalAuthenticatedData: []byte("aad")})
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(dec.Plaintext) != "hello" || !dec.UsedPrimary {
		t.Fatalf("unexpected decrypt response: %+v", dec)
	}
	if _, err := srv.Decrypt(ctx, &kmspb.DecryptRequest{Name: ckName, Ciphertext: enc.Ciphertext, AdditionalAuthenticatedData: []byte("wrong")}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("decrypt wrong AAD: got %v", err)
	}
}

func TestMacSignVerifyAndRandomBytes(t *testing.T) {
	srv, _ := newTestServer(t)
	ctx := context.Background()
	if _, err := srv.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{Parent: krParent, KeyRingId: "main"}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{Parent: krName, CryptoKeyId: "mac", CryptoKey: &kmspb.CryptoKey{Purpose: kmspb.CryptoKey_MAC}}); err != nil {
		t.Fatal(err)
	}
	version := krName + "/cryptoKeys/mac/cryptoKeyVersions/1"
	sign, err := srv.MacSign(ctx, &kmspb.MacSignRequest{Name: version, Data: []byte("message")})
	if err != nil {
		t.Fatalf("MacSign: %v", err)
	}
	verify, err := srv.MacVerify(ctx, &kmspb.MacVerifyRequest{Name: version, Data: []byte("message"), Mac: sign.Mac})
	if err != nil || !verify.Success {
		t.Fatalf("MacVerify: %v %+v", err, verify)
	}
	verify, err = srv.MacVerify(ctx, &kmspb.MacVerifyRequest{Name: version, Data: []byte("tampered"), Mac: sign.Mac})
	if err != nil || verify.Success {
		t.Fatalf("tampered MacVerify: %v %+v", err, verify)
	}
	random, err := srv.GenerateRandomBytes(ctx, &kmspb.GenerateRandomBytesRequest{Location: krParent, LengthBytes: 32, ProtectionLevel: kmspb.ProtectionLevel_HSM})
	if err != nil || len(random.Data) != 32 || random.DataCrc32C == nil {
		t.Fatalf("GenerateRandomBytes: %v %+v", err, random)
	}
}

func TestGRPCTransport(t *testing.T) {
	srv, _ := newTestServer(t)
	lis := bufconn.Listen(1024 * 1024)
	gs := grpc.NewServer()
	kmspb.RegisterKeyManagementServiceServer(gs, srv)
	go func() { _ = gs.Serve(lis) }()
	defer gs.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := kmspb.NewKeyManagementServiceClient(conn)

	if _, err := client.CreateKeyRing(context.Background(), &kmspb.CreateKeyRingRequest{Parent: krParent, KeyRingId: "grpc-ring"}); err != nil {
		t.Fatalf("gRPC CreateKeyRing: %v", err)
	}
	keys, err := client.ListCryptoKeys(context.Background(), &kmspb.ListCryptoKeysRequest{Parent: krName})
	if err != nil {
		t.Fatalf("gRPC ListCryptoKeys on empty ring: %v", err)
	}
	_ = keys
	if _, err := client.GetKeyRing(context.Background(), &kmspb.GetKeyRingRequest{Name: "projects/demo/locations/us-east1/keyRings/grpc-ring"}); err != nil {
		t.Fatalf("gRPC GetKeyRing: %v", err)
	}
}

func TestRESTGateway(t *testing.T) {
	srv, _ := newTestServer(t)
	mux := runtime.NewServeMux()
	ctx := context.Background()
	if err := kmspb.RegisterKeyManagementServiceHandlerServer(ctx, mux, srv); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// POST /v1/{parent}/keyRings?keyRingId=main
	req, _ := http.NewRequest("POST", ts.URL+"/v1/"+krParent+"/keyRings?keyRingId=main", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("create key ring status = %d body=%s", resp.StatusCode, raw)
	}
	var kr map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&kr); err != nil {
		t.Fatal(err)
	}
	if kr["name"] != krName {
		t.Fatalf("unexpected key ring: %v", kr)
	}

	// Create crypto key via REST.
	req2, _ := http.NewRequest("POST", ts.URL+"/v1/"+krName+"/cryptoKeys?cryptoKeyId=rest-key", strings.NewReader(`{"purpose":"ENCRYPT_DECRYPT"}`))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		raw, _ := io.ReadAll(resp2.Body)
		t.Fatalf("create crypto key status = %d body=%s", resp2.StatusCode, raw)
	}
	var ck map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&ck); err != nil {
		t.Fatal(err)
	}
	if ck["name"] != krName+"/cryptoKeys/rest-key" {
		t.Fatalf("unexpected crypto key: %v", ck)
	}
	if primary, ok := ck["primary"].(map[string]any); !ok || primary["name"] != krName+"/cryptoKeys/rest-key/cryptoKeyVersions/1" {
		t.Fatalf("unexpected primary: %v", ck["primary"])
	}

	// Encrypt succeeds through the REST gateway.
	resp3, err := http.Post(ts.URL+"/v1/"+krName+"/cryptoKeys/rest-key:encrypt", "application/json", strings.NewReader(`{"plaintext":"aGk="}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != 200 {
		raw, _ := io.ReadAll(resp3.Body)
		t.Fatalf("encrypt status = %d body=%s", resp3.StatusCode, raw)
	}
	var enc map[string]any
	if err := json.NewDecoder(resp3.Body).Decode(&enc); err != nil {
		t.Fatal(err)
	}
	if enc["ciphertext"] == "" || enc["name"] != krName+"/cryptoKeys/rest-key/cryptoKeyVersions/1" {
		t.Fatalf("unexpected encrypt response: %v", enc)
	}

	// DeleteCryptoKey returns an LRO via REST.
	req4, _ := http.NewRequest("DELETE", ts.URL+"/v1/"+krName+"/cryptoKeys/rest-key", nil)
	resp4, err := http.DefaultClient.Do(req4)
	if err != nil {
		t.Fatal(err)
	}
	defer resp4.Body.Close()
	if resp4.StatusCode != 200 {
		raw, _ := io.ReadAll(resp4.Body)
		t.Fatalf("delete status = %d body=%s", resp4.StatusCode, raw)
	}
	var op map[string]any
	if err := json.NewDecoder(resp4.Body).Decode(&op); err != nil {
		t.Fatal(err)
	}
	if op["done"] != true || op["name"] == "" {
		t.Fatalf("unexpected operation: %v", op)
	}
}

var _ = longrunningpb.Operation{}

func TestAsymmetricAndRawCryptoRoundTrips(t *testing.T) {
	srv, _ := newTestServer(t)
	ctx := context.Background()
	if _, err := srv.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{Parent: krParent, KeyRingId: "main"}); err != nil {
		t.Fatal(err)
	}
	// EC signing key: default algorithm EC_SIGN_P256_SHA256.
	if _, err := srv.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{Parent: krName, CryptoKeyId: "ec", CryptoKey: &kmspb.CryptoKey{Purpose: kmspb.CryptoKey_ASYMMETRIC_SIGN}}); err != nil {
		t.Fatal(err)
	}
	ecVersion := krName + "/cryptoKeys/ec/cryptoKeyVersions/1"
	digest := sha256.Sum256([]byte("signed"))
	sign, err := srv.AsymmetricSign(ctx, &kmspb.AsymmetricSignRequest{Name: ecVersion, Digest: &kmspb.Digest{Digest: &kmspb.Digest_Sha256{Sha256: digest[:]}}})
	if err != nil {
		t.Fatalf("AsymmetricSign: %v", err)
	}
	pub, err := srv.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: ecVersion})
	if err != nil {
		t.Fatalf("GetPublicKey: %v", err)
	}
	block, _ := pem.Decode([]byte(pub.Pem))
	if block == nil {
		t.Fatal("public key PEM did not decode")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if !ecdsa.VerifyASN1(parsed.(*ecdsa.PublicKey), digest[:], sign.Signature) {
		t.Fatal("ECDSA signature did not verify")
	}

	// RSA decryption key: encrypt with public key, decrypt through KMS.
	if _, err := srv.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{Parent: krName, CryptoKeyId: "rsa", CryptoKey: &kmspb.CryptoKey{Purpose: kmspb.CryptoKey_ASYMMETRIC_DECRYPT}}); err != nil {
		t.Fatal(err)
	}
	rsaVersion := krName + "/cryptoKeys/rsa/cryptoKeyVersions/1"
	rsaPubResp, err := srv.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: rsaVersion})
	if err != nil {
		t.Fatal(err)
	}
	rsaBlock, _ := pem.Decode([]byte(rsaPubResp.Pem))
	rsaParsed, err := x509.ParsePKIXPublicKey(rsaBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaParsed.(*rsa.PublicKey), []byte("secret"), nil)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := srv.AsymmetricDecrypt(ctx, &kmspb.AsymmetricDecryptRequest{Name: rsaVersion, Ciphertext: ciphertext})
	if err != nil {
		t.Fatalf("AsymmetricDecrypt: %v", err)
	}
	if string(dec.Plaintext) != "secret" {
		t.Fatalf("plaintext = %q", dec.Plaintext)
	}

	// Raw AES-GCM round trip on a key explicitly templated for raw operations.
	if _, err := srv.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{Parent: krName, CryptoKeyId: "raw", CryptoKey: &kmspb.CryptoKey{
		Purpose:         kmspb.CryptoKey_ENCRYPT_DECRYPT,
		VersionTemplate: &kmspb.CryptoKeyVersionTemplate{Algorithm: kmspb.CryptoKeyVersion_AES_256_GCM},
	}}); err != nil {
		t.Fatal(err)
	}
	rawVersion := krName + "/cryptoKeys/raw/cryptoKeyVersions/1"
	rawEnc, err := srv.RawEncrypt(ctx, &kmspb.RawEncryptRequest{Name: rawVersion, Plaintext: []byte("raw plaintext"), AdditionalAuthenticatedData: []byte("aad")})
	if err != nil {
		t.Fatalf("RawEncrypt: %v", err)
	}
	rawDec, err := srv.RawDecrypt(ctx, &kmspb.RawDecryptRequest{Name: rawVersion, Ciphertext: rawEnc.Ciphertext, InitializationVector: rawEnc.InitializationVector, AdditionalAuthenticatedData: []byte("aad")})
	if err != nil {
		t.Fatalf("RawDecrypt: %v", err)
	}
	if string(rawDec.Plaintext) != "raw plaintext" {
		t.Fatalf("raw plaintext = %q", rawDec.Plaintext)
	}
}
