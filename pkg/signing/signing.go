package signing

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// PublicKeyBase64 is the raw Ed25519 public key (32 bytes), base64-encoded.
// Set at compile time via ldflags. If empty, verification is skipped (dev mode).
var PublicKeyBase64 = ""

// Verify checks that signatureB64 is a valid Ed25519 signature over the payload
// string "{contentType}:{slug}:{sha256Hex}" using the embedded public key.
func Verify(contentType, slug, sha256Hex, signatureB64 string) error {
	if PublicKeyBase64 == "" {
		return nil
	}

	pub, err := publicKey()
	if err != nil {
		return err
	}

	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	payload := []byte(fmt.Sprintf("%s:%s:%s", contentType, slug, sha256Hex))
	if !ed25519.Verify(pub, payload, sig) {
		return fmt.Errorf("signature verification failed: file may have been tampered with")
	}

	return nil
}

// KeyID returns the key ID of the embedded public key: the first 16 hex
// characters of SHA-256(SPKI DER encoding). This matches the key_id field
// returned by the server's verify endpoint.
func KeyID() (string, error) {
	if PublicKeyBase64 == "" {
		return "", fmt.Errorf("no public key configured")
	}

	pub, err := publicKey()
	if err != nil {
		return "", err
	}

	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key to SPKI DER: %w", err)
	}

	h := sha256.Sum256(der)
	return hex.EncodeToString(h[:8]), nil
}

// HasPublicKey returns true if a public key is configured for verification.
func HasPublicKey() bool {
	return PublicKeyBase64 != ""
}

func publicKey() (ed25519.PublicKey, error) {
	pubBytes, err := base64.StdEncoding.DecodeString(PublicKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid embedded public key: %w", err)
	}

	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: got %d, want %d", len(pubBytes), ed25519.PublicKeySize)
	}

	return ed25519.PublicKey(pubBytes), nil
}
