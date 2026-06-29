package datadir

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/wzhejunqiu/ds-code/internal/version"
)

var (
	identifierOnce  sync.Once
	identifierValue string
)

// Identifier returns the install-scoped LLM user id, creating ~/.ds-code/identifier if needed.
func Identifier() string {
	identifierOnce.Do(func() {
		identifierValue, _ = loadOrCreateIdentifier()
	})
	return identifierValue
}

func resetIdentifierForTest() {
	identifierOnce = sync.Once{}
	identifierValue = ""
}

// ResetIdentifierForTest clears the in-process identifier cache (tests only).
func ResetIdentifierForTest() {
	resetIdentifierForTest()
}

func identifierPath() (string, error) {
	root, err := UserDataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "identifier"), nil
}

func loadOrCreateIdentifier() (string, error) {
	path, err := identifierPath()
	if err != nil {
		return "", err
	}
	if raw, err := os.ReadFile(path); err == nil {
		if id := strings.TrimSpace(string(raw)); validIdentifier(id) {
			return id, nil
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return createIdentifier(path)
}

func createIdentifier(path string) (string, error) {
	root := filepath.Dir(path)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", err
	}
	id := computeIdentifier(uuid.NewString(), currentUsername())
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(id+"\n"), 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return id, nil
}

func computeIdentifier(uuidV4, whoami string) string {
	sum := sha256.Sum256([]byte(uuidV4 + whoami + version.Name))
	return hex.EncodeToString(sum[:])
}

func validIdentifier(id string) bool {
	if len(id) != 64 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func currentUsername() string {
	if u, err := user.Current(); err == nil {
		if name := strings.TrimSpace(u.Username); name != "" {
			return name
		}
	}
	out, err := exec.Command("whoami").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
