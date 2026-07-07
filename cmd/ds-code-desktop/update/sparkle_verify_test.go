package update_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/wzhejunqiu/ds-code/cmd/ds-code-desktop/update"
)

func TestVerifyEdDSASignature_roundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("https://example.com/appcast.xml")
	sig := ed25519.Sign(priv, msg)
	if err := update.VerifyEdDSASignature(
		base64.StdEncoding.EncodeToString(pub),
		base64.StdEncoding.EncodeToString(sig),
		msg,
	); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyEdDSASignature_rejectsBadSig(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("payload")
	sig := ed25519.Sign(priv, msg)
	sig[0] ^= 0xff
	if err := update.VerifyEdDSASignature(
		base64.StdEncoding.EncodeToString(pub),
		base64.StdEncoding.EncodeToString(sig),
		msg,
	); err == nil {
		t.Fatal("expected verification error")
	}
}
