package kms

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"hash"
	"hash/crc32"
	"io"

	gcperr "github.com/bestony/google-cloud-emulator/core/errors"
	kmspb "github.com/bestony/google-cloud-emulator/gen/google/cloud/kms/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type keyMaterial struct {
	symmetric []byte
	rsa       *rsa.PrivateKey
	ecdsa     *ecdsa.PrivateKey
	ed25519   ed25519.PrivateKey
}

var kmsCRC32CTable = crc32.MakeTable(crc32.Castagnoli)

func crc32c(data []byte) int64                       { return int64(crc32.Checksum(data, kmsCRC32CTable)) }
func crc32cValue(data []byte) *wrapperspb.Int64Value { return wrapperspb.Int64(crc32c(data)) }

func validCRC(data []byte, want *wrapperspb.Int64Value) bool {
	return want == nil || want.GetValue() == crc32c(data)
}

func randomBytes(n int) ([]byte, error) {
	out := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, out)
	return out, err
}

func generateMaterial(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) (*keyMaterial, error) {
	m := &keyMaterial{}
	symmetricBits := 256
	switch alg {
	case kmspb.CryptoKeyVersion_GOOGLE_SYMMETRIC_ENCRYPTION, kmspb.CryptoKeyVersion_AES_256_GCM, kmspb.CryptoKeyVersion_AES_256_CBC, kmspb.CryptoKeyVersion_AES_256_CTR, kmspb.CryptoKeyVersion_HMAC_SHA256, kmspb.CryptoKeyVersion_HMAC_SHA1, kmspb.CryptoKeyVersion_HMAC_SHA224, kmspb.CryptoKeyVersion_HMAC_SHA384, kmspb.CryptoKeyVersion_HMAC_SHA512:
	case kmspb.CryptoKeyVersion_AES_128_GCM, kmspb.CryptoKeyVersion_AES_128_CBC, kmspb.CryptoKeyVersion_AES_128_CTR:
		symmetricBits = 128
	}
	if isSymmetricAlgorithm(alg) || isMACAlgorithm(alg) {
		key, err := randomBytes(symmetricBits / 8)
		if err != nil {
			return nil, err
		}
		m.symmetric = key
		return m, nil
	}
	if isRSASignAlgorithm(alg) || isRSADecryptAlgorithm(alg) {
		bits := 2048
		if bytes.Contains([]byte(alg.String()), []byte("3072")) {
			bits = 3072
		}
		if bytes.Contains([]byte(alg.String()), []byte("4096")) {
			bits = 4096
		}
		key, err := rsa.GenerateKey(rand.Reader, bits)
		if err != nil {
			return nil, err
		}
		m.rsa = key
		return m, nil
	}
	switch alg {
	case kmspb.CryptoKeyVersion_EC_SIGN_ED25519:
		_, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		m.ed25519 = key
	case kmspb.CryptoKeyVersion_EC_SIGN_P384_SHA384:
		key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if err != nil {
			return nil, err
		}
		m.ecdsa = key
	default:
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		m.ecdsa = key
	}
	return m, nil
}

func isSymmetricAlgorithm(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) bool {
	switch alg {
	case kmspb.CryptoKeyVersion_GOOGLE_SYMMETRIC_ENCRYPTION, kmspb.CryptoKeyVersion_AES_128_GCM, kmspb.CryptoKeyVersion_AES_256_GCM, kmspb.CryptoKeyVersion_AES_128_CBC, kmspb.CryptoKeyVersion_AES_256_CBC, kmspb.CryptoKeyVersion_AES_128_CTR, kmspb.CryptoKeyVersion_AES_256_CTR:
		return true
	}
	return false
}
func isGCM(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) bool {
	return alg == kmspb.CryptoKeyVersion_GOOGLE_SYMMETRIC_ENCRYPTION || alg == kmspb.CryptoKeyVersion_AES_128_GCM || alg == kmspb.CryptoKeyVersion_AES_256_GCM
}
func isCBC(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) bool {
	return alg == kmspb.CryptoKeyVersion_AES_128_CBC || alg == kmspb.CryptoKeyVersion_AES_256_CBC
}
func isCTR(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) bool {
	return alg == kmspb.CryptoKeyVersion_AES_128_CTR || alg == kmspb.CryptoKeyVersion_AES_256_CTR
}
func isMACAlgorithm(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) bool {
	switch alg {
	case kmspb.CryptoKeyVersion_HMAC_SHA1, kmspb.CryptoKeyVersion_HMAC_SHA224, kmspb.CryptoKeyVersion_HMAC_SHA256, kmspb.CryptoKeyVersion_HMAC_SHA384, kmspb.CryptoKeyVersion_HMAC_SHA512:
		return true
	}
	return false
}
func isRSASignAlgorithm(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) bool {
	return bytes.HasPrefix([]byte(alg.String()), []byte("RSA_SIGN_"))
}
func isRSAPSS(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) bool {
	return bytes.HasPrefix([]byte(alg.String()), []byte("RSA_SIGN_PSS"))
}
func isRSADecryptAlgorithm(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) bool {
	return bytes.HasPrefix([]byte(alg.String()), []byte("RSA_DECRYPT_"))
}
func isECAlgorithm(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) bool {
	return bytes.HasPrefix([]byte(alg.String()), []byte("EC_SIGN_"))
}

func hashForAlgorithm(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) (crypto.Hash, []byte) {
	name := []byte(alg.String())
	switch {
	case bytes.Contains(name, []byte("SHA512")):
		return crypto.SHA512, nil
	case bytes.Contains(name, []byte("SHA384")):
		return crypto.SHA384, nil
	case bytes.Contains(name, []byte("SHA256")):
		return crypto.SHA256, nil
	case bytes.Contains(name, []byte("SHA224")):
		return crypto.SHA224, nil
	case bytes.Contains(name, []byte("SHA1")):
		return crypto.SHA1, nil
	default:
		return crypto.SHA256, nil
	}
}

func newHash(h crypto.Hash) hash.Hash {
	switch h {
	case crypto.SHA1:
		return sha1.New()
	case crypto.SHA224:
		return sha256.New224()
	case crypto.SHA256:
		return sha256.New()
	case crypto.SHA384:
		return sha512.New384()
	case crypto.SHA512:
		return sha512.New()
	default:
		return sha256.New()
	}
}

// materialForVersion returns the version proto and lazily generated key material.
func (s *State) materialForVersion(versionName string) (*kmspb.CryptoKeyVersion, *keyMaterial, error) {
	keyName, id, ok := SplitVersionName(versionName)
	if !ok {
		return nil, nil, gcperr.GRPC(gcperr.InvalidArgument, "invalid crypto key version name")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cryptoKeys[keyName]
	if !ok {
		return nil, nil, gcperr.GRPC(gcperr.NotFound, "crypto key not found")
	}
	v, ok := entry.Versions[id]
	if !ok {
		return nil, nil, gcperr.GRPC(gcperr.NotFound, "crypto key version not found")
	}
	if v.GetState() != kmspb.CryptoKeyVersion_ENABLED {
		return nil, nil, gcperr.GRPC(gcperr.FailedPrecondition, "crypto key version is not enabled")
	}
	if entry.Material == nil {
		entry.Material = map[string]*keyMaterial{}
	}
	material := entry.Material[id]
	if material == nil {
		var err error
		material, err = generateMaterial(v.GetAlgorithm())
		if err != nil {
			return nil, nil, gcperr.GRPC(gcperr.Internal, "generate key material: %v", err)
		}
		entry.Material[id] = material
	}
	return v, material, nil
}

// resolveCryptoVersion resolves either a CryptoKey name (primary version) or a CryptoKeyVersion name.
func (s *State) resolveCryptoVersion(name string) (string, error) {
	if ValidVersionName(name) {
		return name, nil
	}
	if !ValidCryptoKeyName(name) {
		return "", gcperr.GRPC(gcperr.InvalidArgument, "invalid crypto key name")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.cryptoKeys[name]
	if !ok {
		return "", gcperr.GRPC(gcperr.NotFound, "crypto key not found")
	}
	if entry.Proto.GetPrimary() == nil || entry.Proto.GetPrimary().GetName() == "" {
		return "", gcperr.GRPC(gcperr.FailedPrecondition, "crypto key has no primary version")
	}
	return entry.Proto.GetPrimary().GetName(), nil
}

var envelopeMagic = []byte("GCEKMS1\x00")

func encryptEnvelope(version string, plaintext, aad, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, err := randomBytes(gcm.NonceSize())
	if err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nil, nonce, plaintext, aad)
	out := append([]byte{}, envelopeMagic...)
	vb := []byte(version)
	var ln [2]byte
	binary.BigEndian.PutUint16(ln[:], uint16(len(vb)))
	out = append(out, ln[:]...)
	out = append(out, vb...)
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, nil
}

func decryptEnvelope(ciphertext, aad []byte, s *State, keyName string) ([]byte, string, bool, error) {
	if !bytes.HasPrefix(ciphertext, envelopeMagic) || len(ciphertext) < len(envelopeMagic)+2 {
		return nil, "", false, gcperr.GRPC(gcperr.InvalidArgument, "ciphertext is not an emulator KMS envelope")
	}
	pos := len(envelopeMagic)
	ln := int(binary.BigEndian.Uint16(ciphertext[pos : pos+2]))
	pos += 2
	if len(ciphertext) < pos+ln+12+16 {
		return nil, "", false, gcperr.GRPC(gcperr.InvalidArgument, "truncated ciphertext")
	}
	version := string(ciphertext[pos : pos+ln])
	pos += ln
	if owner, _, _ := SplitVersionName(version); owner != keyName {
		return nil, "", false, gcperr.GRPC(gcperr.InvalidArgument, "ciphertext belongs to a different crypto key")
	}
	v, material, err := s.materialForVersion(version)
	if err != nil {
		return nil, "", false, err
	}
	block, _ := aes.NewCipher(material.symmetric)
	gcm, _ := cipher.NewGCM(block)
	if len(ciphertext) < pos+gcm.NonceSize() {
		return nil, "", false, gcperr.GRPC(gcperr.InvalidArgument, "truncated ciphertext nonce")
	}
	nonce := ciphertext[pos : pos+gcm.NonceSize()]
	pos += gcm.NonceSize()
	plain, err := gcm.Open(nil, nonce, ciphertext[pos:], aad)
	if err != nil {
		return nil, "", false, gcperr.GRPC(gcperr.InvalidArgument, "ciphertext authentication failed")
	}
	return plain, version, version == v.GetName(), nil
}

func aesCrypt(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm, key, iv, input, aad []byte, encrypt bool) ([]byte, []byte, int32, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, 0, err
	}
	if isGCM(alg) {
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, nil, 0, err
		}
		if encrypt {
			if len(iv) == 0 {
				iv, err = randomBytes(gcm.NonceSize())
				if err != nil {
					return nil, nil, 0, err
				}
			}
			if len(iv) != gcm.NonceSize() {
				return nil, nil, 0, fmt.Errorf("invalid GCM IV length")
			}
			return gcm.Seal(nil, iv, input, aad), iv, 16, nil
		}
		out, err := gcm.Open(nil, iv, input, aad)
		return out, iv, 16, err
	}
	if len(iv) == 0 {
		iv, err = randomBytes(block.BlockSize())
		if err != nil {
			return nil, nil, 0, err
		}
	}
	if len(iv) != block.BlockSize() {
		return nil, nil, 0, fmt.Errorf("invalid IV length")
	}
	if isCBC(alg) {
		if encrypt {
			padlen := block.BlockSize() - len(input)%block.BlockSize()
			input = append(append([]byte{}, input...), bytes.Repeat([]byte{byte(padlen)}, padlen)...)
			out := make([]byte, len(input))
			cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, input)
			return out, iv, 0, nil
		}
		if len(input)%block.BlockSize() != 0 {
			return nil, nil, 0, fmt.Errorf("ciphertext is not block aligned")
		}
		out := make([]byte, len(input))
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, input)
		if len(out) == 0 {
			return nil, nil, 0, fmt.Errorf("empty plaintext")
		}
		padlen := int(out[len(out)-1])
		if padlen == 0 || padlen > block.BlockSize() || padlen > len(out) {
			return nil, nil, 0, fmt.Errorf("invalid padding")
		}
		for _, b := range out[len(out)-padlen:] {
			if int(b) != padlen {
				return nil, nil, 0, fmt.Errorf("invalid padding")
			}
		}
		return out[:len(out)-padlen], iv, 0, nil
	}
	out := make([]byte, len(input))
	cipher.NewCTR(block, iv).XORKeyStream(out, input)
	return out, iv, 0, nil
}

func (s *Server) Encrypt(ctx context.Context, req *kmspb.EncryptRequest) (*kmspb.EncryptResponse, error) {
	if !validCRC(req.GetPlaintext(), req.GetPlaintextCrc32C()) || !validCRC(req.GetAdditionalAuthenticatedData(), req.GetAdditionalAuthenticatedDataCrc32C()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "CRC32C checksum mismatch")
	}
	versionName, err := s.state.resolveCryptoVersion(req.GetName())
	if err != nil {
		return nil, err
	}
	v, material, err := s.state.materialForVersion(versionName)
	if err != nil {
		return nil, err
	}
	if !isGCM(v.GetAlgorithm()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "algorithm does not support Encrypt; use RawEncrypt")
	}
	ct, err := encryptEnvelope(v.GetName(), req.GetPlaintext(), req.GetAdditionalAuthenticatedData(), material.symmetric)
	if err != nil {
		return nil, gcperr.GRPC(gcperr.Internal, "encrypt: %v", err)
	}
	return &kmspb.EncryptResponse{Name: v.GetName(), Ciphertext: ct, CiphertextCrc32C: crc32cValue(ct), VerifiedPlaintextCrc32C: req.GetPlaintextCrc32C() != nil, VerifiedAdditionalAuthenticatedDataCrc32C: req.GetAdditionalAuthenticatedDataCrc32C() != nil, ProtectionLevel: v.GetProtectionLevel()}, nil
}

func (s *Server) Decrypt(ctx context.Context, req *kmspb.DecryptRequest) (*kmspb.DecryptResponse, error) {
	if !validCRC(req.GetCiphertext(), req.GetCiphertextCrc32C()) || !validCRC(req.GetAdditionalAuthenticatedData(), req.GetAdditionalAuthenticatedDataCrc32C()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "CRC32C checksum mismatch")
	}
	if !ValidCryptoKeyName(req.GetName()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "Decrypt name must be a crypto key")
	}
	plain, version, _, err := decryptEnvelope(req.GetCiphertext(), req.GetAdditionalAuthenticatedData(), s.state, req.GetName())
	if err != nil {
		return nil, err
	}
	v, material, err := s.state.materialForVersion(version)
	_ = material
	if err != nil {
		return nil, err
	}
	primary := ""
	if ck := s.state.GetCryptoKey(req.GetName()); ck != nil && ck.GetPrimary() != nil {
		primary = ck.GetPrimary().GetName()
	}
	return &kmspb.DecryptResponse{Plaintext: plain, PlaintextCrc32C: crc32cValue(plain), UsedPrimary: version == primary, ProtectionLevel: v.GetProtectionLevel()}, nil
}

func (s *Server) RawEncrypt(ctx context.Context, req *kmspb.RawEncryptRequest) (*kmspb.RawEncryptResponse, error) {
	if !validCRC(req.GetPlaintext(), req.GetPlaintextCrc32C()) || !validCRC(req.GetAdditionalAuthenticatedData(), req.GetAdditionalAuthenticatedDataCrc32C()) || !validCRC(req.GetInitializationVector(), req.GetInitializationVectorCrc32C()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "CRC32C checksum mismatch")
	}
	v, material, err := s.state.materialForVersion(req.GetName())
	if err != nil {
		return nil, err
	}
	if !isSymmetricAlgorithm(v.GetAlgorithm()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "algorithm does not support raw symmetric encryption")
	}
	ct, iv, tag, err := aesCrypt(v.GetAlgorithm(), material.symmetric, req.GetInitializationVector(), req.GetPlaintext(), req.GetAdditionalAuthenticatedData(), true)
	if err != nil {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "raw encrypt: %v", err)
	}
	return &kmspb.RawEncryptResponse{Name: v.GetName(), Ciphertext: ct, InitializationVector: iv, TagLength: tag, CiphertextCrc32C: crc32cValue(ct), InitializationVectorCrc32C: crc32cValue(iv), VerifiedPlaintextCrc32C: req.GetPlaintextCrc32C() != nil, VerifiedAdditionalAuthenticatedDataCrc32C: req.GetAdditionalAuthenticatedDataCrc32C() != nil, VerifiedInitializationVectorCrc32C: req.GetInitializationVectorCrc32C() != nil, ProtectionLevel: v.GetProtectionLevel()}, nil
}

func (s *Server) RawDecrypt(ctx context.Context, req *kmspb.RawDecryptRequest) (*kmspb.RawDecryptResponse, error) {
	if !validCRC(req.GetCiphertext(), req.GetCiphertextCrc32C()) || !validCRC(req.GetAdditionalAuthenticatedData(), req.GetAdditionalAuthenticatedDataCrc32C()) || !validCRC(req.GetInitializationVector(), req.GetInitializationVectorCrc32C()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "CRC32C checksum mismatch")
	}
	v, material, err := s.state.materialForVersion(req.GetName())
	if err != nil {
		return nil, err
	}
	plain, _, _, err := aesCrypt(v.GetAlgorithm(), material.symmetric, req.GetInitializationVector(), req.GetCiphertext(), req.GetAdditionalAuthenticatedData(), false)
	if err != nil {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "raw decrypt: %v", err)
	}
	return &kmspb.RawDecryptResponse{Plaintext: plain, PlaintextCrc32C: crc32cValue(plain), ProtectionLevel: v.GetProtectionLevel()}, nil
}

func publicKeyMaterial(m *keyMaterial) (any, error) {
	switch {
	case m.rsa != nil:
		return &m.rsa.PublicKey, nil
	case m.ecdsa != nil:
		return &m.ecdsa.PublicKey, nil
	case m.ed25519 != nil:
		return m.ed25519.Public().(ed25519.PublicKey), nil
	default:
		return nil, fmt.Errorf("version has no public key")
	}
}

func (s *Server) GetPublicKey(ctx context.Context, req *kmspb.GetPublicKeyRequest) (*kmspb.PublicKey, error) {
	v, material, err := s.state.materialForVersion(req.GetName())
	if err != nil {
		return nil, err
	}
	pub, err := publicKeyMaterial(material)
	if err != nil {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "%v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, gcperr.GRPC(gcperr.Internal, "marshal public key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	format := req.GetPublicKeyFormat()
	if format == kmspb.PublicKey_PUBLIC_KEY_FORMAT_UNSPECIFIED {
		format = kmspb.PublicKey_PEM
	}
	return &kmspb.PublicKey{Name: v.GetName(), Pem: string(pemBytes), Algorithm: v.GetAlgorithm(), PemCrc32C: crc32cValue(pemBytes), ProtectionLevel: v.GetProtectionLevel(), PublicKeyFormat: format, PublicKey: &kmspb.ChecksummedData{Data: pemBytes, Crc32CChecksum: crc32cValue(pemBytes)}}, nil
}

func digestBytesAndHash(req *kmspb.AsymmetricSignRequest, alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) ([]byte, crypto.Hash, error) {
	h, _ := hashForAlgorithm(alg)
	if req.GetDigest() != nil && len(req.GetData()) > 0 {
		return nil, 0, fmt.Errorf("digest and data are mutually exclusive")
	}
	if len(req.GetData()) > 0 {
		if alg == kmspb.CryptoKeyVersion_EC_SIGN_ED25519 {
			return req.GetData(), crypto.Hash(0), nil
		}
		hh := newHash(h)
		hh.Write(req.GetData())
		return hh.Sum(nil), h, nil
	}
	d := req.GetDigest()
	if d == nil {
		return nil, 0, fmt.Errorf("digest or data is required")
	}
	switch {
	case len(d.GetSha256()) > 0:
		return d.GetSha256(), crypto.SHA256, nil
	case len(d.GetSha384()) > 0:
		return d.GetSha384(), crypto.SHA384, nil
	case len(d.GetSha512()) > 0:
		return d.GetSha512(), crypto.SHA512, nil
	}
	return nil, 0, fmt.Errorf("unsupported digest type")
}

func (s *Server) AsymmetricSign(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
	v, material, err := s.state.materialForVersion(req.GetName())
	if err != nil {
		return nil, err
	}
	digest, hashID, err := digestBytesAndHash(req, v.GetAlgorithm())
	if err != nil {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "%v", err)
	}
	var sig []byte
	switch {
	case material.ed25519 != nil:
		if len(req.GetData()) == 0 {
			return nil, gcperr.GRPC(gcperr.InvalidArgument, "Ed25519 signing requires data")
		}
		sig = ed25519.Sign(material.ed25519, req.GetData())
	case material.ecdsa != nil:
		sig, err = ecdsa.SignASN1(rand.Reader, material.ecdsa, digest)
	case material.rsa != nil && isRSAPSS(v.GetAlgorithm()):
		sig, err = rsa.SignPSS(rand.Reader, material.rsa, hashID, digest, nil)
	case material.rsa != nil:
		sig, err = rsa.SignPKCS1v15(rand.Reader, material.rsa, hashID, digest)
	default:
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "algorithm does not support asymmetric signing")
	}
	if err != nil {
		return nil, gcperr.GRPC(gcperr.Internal, "sign: %v", err)
	}
	return &kmspb.AsymmetricSignResponse{Name: v.GetName(), Signature: sig, SignatureCrc32C: crc32cValue(sig), VerifiedDigestCrc32C: req.GetDigestCrc32C() != nil, VerifiedDataCrc32C: req.GetDataCrc32C() != nil, ProtectionLevel: v.GetProtectionLevel()}, nil
}

func (s *Server) AsymmetricDecrypt(ctx context.Context, req *kmspb.AsymmetricDecryptRequest) (*kmspb.AsymmetricDecryptResponse, error) {
	if !validCRC(req.GetCiphertext(), req.GetCiphertextCrc32C()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "CRC32C checksum mismatch")
	}
	v, material, err := s.state.materialForVersion(req.GetName())
	if err != nil {
		return nil, err
	}
	if material.rsa == nil || !isRSADecryptAlgorithm(v.GetAlgorithm()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "algorithm does not support asymmetric decryption")
	}
	h, _ := hashForAlgorithm(v.GetAlgorithm())
	hh := newHash(h)
	plain, err := rsa.DecryptOAEP(hh, rand.Reader, material.rsa, req.GetCiphertext(), nil)
	if err != nil {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "asymmetric decrypt failed")
	}
	return &kmspb.AsymmetricDecryptResponse{Plaintext: plain, PlaintextCrc32C: crc32cValue(plain), ProtectionLevel: v.GetProtectionLevel()}, nil
}

func hmacHash(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm) func() hash.Hash {
	h, _ := hashForAlgorithm(alg)
	return func() hash.Hash { return newHash(h) }
}
func computeMAC(alg kmspb.CryptoKeyVersion_CryptoKeyVersionAlgorithm, key, data []byte) []byte {
	mac := hmac.New(hmacHash(alg), key)
	mac.Write(data)
	return mac.Sum(nil)
}

func (s *Server) MacSign(ctx context.Context, req *kmspb.MacSignRequest) (*kmspb.MacSignResponse, error) {
	if !validCRC(req.GetData(), req.GetDataCrc32C()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "CRC32C checksum mismatch")
	}
	v, material, err := s.state.materialForVersion(req.GetName())
	if err != nil {
		return nil, err
	}
	if !isMACAlgorithm(v.GetAlgorithm()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "algorithm does not support MAC signing")
	}
	mac := computeMAC(v.GetAlgorithm(), material.symmetric, req.GetData())
	return &kmspb.MacSignResponse{Name: v.GetName(), Mac: mac, MacCrc32C: crc32cValue(mac), VerifiedDataCrc32C: req.GetDataCrc32C() != nil, ProtectionLevel: v.GetProtectionLevel()}, nil
}

func (s *Server) MacVerify(ctx context.Context, req *kmspb.MacVerifyRequest) (*kmspb.MacVerifyResponse, error) {
	if !validCRC(req.GetData(), req.GetDataCrc32C()) || !validCRC(req.GetMac(), req.GetMacCrc32C()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "CRC32C checksum mismatch")
	}
	v, material, err := s.state.materialForVersion(req.GetName())
	if err != nil {
		return nil, err
	}
	if !isMACAlgorithm(v.GetAlgorithm()) {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "algorithm does not support MAC verification")
	}
	want := computeMAC(v.GetAlgorithm(), material.symmetric, req.GetData())
	ok := hmac.Equal(want, req.GetMac())
	return &kmspb.MacVerifyResponse{Name: v.GetName(), Success: ok, VerifiedDataCrc32C: req.GetDataCrc32C() != nil, VerifiedMacCrc32C: req.GetMacCrc32C() != nil, VerifiedSuccessIntegrity: true, ProtectionLevel: v.GetProtectionLevel()}, nil
}

func (s *Server) GenerateRandomBytes(ctx context.Context, req *kmspb.GenerateRandomBytesRequest) (*kmspb.GenerateRandomBytesResponse, error) {
	if req.GetLengthBytes() < 8 || req.GetLengthBytes() > 1024 {
		return nil, gcperr.GRPC(gcperr.InvalidArgument, "length_bytes must be between 8 and 1024")
	}
	data, err := randomBytes(int(req.GetLengthBytes()))
	if err != nil {
		return nil, gcperr.GRPC(gcperr.Internal, "random bytes: %v", err)
	}
	return &kmspb.GenerateRandomBytesResponse{Data: data, DataCrc32C: crc32cValue(data)}, nil
}
