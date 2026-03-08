package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withKey(t *testing.T, fn func(pub ed25519.PublicKey, priv ed25519.PrivateKey)) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	origKey := PublicKeyBase64
	defer func() { PublicKeyBase64 = origKey }()
	PublicKeyBase64 = base64.StdEncoding.EncodeToString(pub)

	fn(pub, priv)
}

func TestVerify_ValidSignature(t *testing.T) {
	withKey(t, func(_ ed25519.PublicKey, priv ed25519.PrivateKey) {
		payload := "map:test-map:abc123def456"
		sig := ed25519.Sign(priv, []byte(payload))

		err := Verify("map", "test-map", "abc123def456", base64.StdEncoding.EncodeToString(sig))
		assert.NoError(t, err)
	})
}

func TestVerify_InvalidSignature(t *testing.T) {
	withKey(t, func(_ ed25519.PublicKey, _ ed25519.PrivateKey) {
		// Sign with a different key
		_, otherPriv, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		payload := "map:test-map:abc123def456"
		sig := ed25519.Sign(otherPriv, []byte(payload))

		err = Verify("map", "test-map", "abc123def456", base64.StdEncoding.EncodeToString(sig))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature verification failed")
	})
}

func TestVerify_TamperedHash(t *testing.T) {
	withKey(t, func(_ ed25519.PublicKey, priv ed25519.PrivateKey) {
		// Sign with the correct hash
		payload := "map:test-map:originalhash"
		sig := ed25519.Sign(priv, []byte(payload))

		// Verify with a different hash (simulating tampered file)
		err := Verify("map", "test-map", "tamperedhash", base64.StdEncoding.EncodeToString(sig))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature verification failed")
	})
}

func TestVerify_NoPublicKey(t *testing.T) {
	origKey := PublicKeyBase64
	defer func() { PublicKeyBase64 = origKey }()
	PublicKeyBase64 = ""

	err := Verify("map", "test", "hash", "sig")
	assert.NoError(t, err, "should skip verification when no public key is set")
}

func TestVerify_BadBase64Signature(t *testing.T) {
	withKey(t, func(_ ed25519.PublicKey, _ ed25519.PrivateKey) {
		err := Verify("map", "test", "hash", "not-valid-base64!!!")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature encoding")
	})
}

func TestHasPublicKey(t *testing.T) {
	origKey := PublicKeyBase64
	defer func() { PublicKeyBase64 = origKey }()

	PublicKeyBase64 = ""
	assert.False(t, HasPublicKey())

	PublicKeyBase64 = "something"
	assert.True(t, HasPublicKey())
}

func TestKeyID(t *testing.T) {
	withKey(t, func(pub ed25519.PublicKey, _ ed25519.PrivateKey) {
		keyID, err := KeyID()
		require.NoError(t, err)

		// Compute expected key ID manually
		der, err := x509.MarshalPKIXPublicKey(pub)
		require.NoError(t, err)
		h := sha256.Sum256(der)
		expected := hex.EncodeToString(h[:8])

		assert.Equal(t, expected, keyID)
		assert.Len(t, keyID, 16)
	})
}

func TestKeyID_NoKey(t *testing.T) {
	origKey := PublicKeyBase64
	defer func() { PublicKeyBase64 = origKey }()
	PublicKeyBase64 = ""

	_, err := KeyID()
	assert.Error(t, err)
}

func TestVerify_FullPayloadFormat(t *testing.T) {
	withKey(t, func(_ ed25519.PublicKey, priv ed25519.PrivateKey) {
		// Simulate the exact server-side flow
		contentType := "map"
		slug := "0-degrees"
		sha256Hex := "d651bc88dcc9cb873395dac902df03b152180547c1068eda55ee48e44efbac0a"

		payload := fmt.Sprintf("%s:%s:%s", contentType, slug, sha256Hex)
		sig := ed25519.Sign(priv, []byte(payload))

		err := Verify(contentType, slug, sha256Hex, base64.StdEncoding.EncodeToString(sig))
		assert.NoError(t, err)
	})
}
