package update

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

// VerifyEdDSASignature checks a Sparkle-style Ed25519 signature (base64) over message bytes.
func VerifyEdDSASignature(publicKeyBase64, signatureBase64 string, message []byte) error {
	pub, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return fmt.Errorf("decode public key: %w", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("public key length %d, want %d", len(pub), ed25519.PublicKeySize)
	}
	sig, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), message, sig) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}
