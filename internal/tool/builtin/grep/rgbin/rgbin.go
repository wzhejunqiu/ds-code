package rgbin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
)

//go:embed rg.tar.gz
var embeddedTarGz []byte

var (
	loadOnce sync.Once
	loadErr  error
	rgBytes  []byte
	rgHash   [32]byte

	validatedMu sync.Mutex
	validated   *validatedRG
)

type validatedRG struct {
	path string
	size int64
	mod  time.Time
}

// Path returns the bundled ripgrep executable path (~/.ds-code/bin/rg).
func Path() (string, error) {
	loadOnce.Do(func() {
		rgBytes, loadErr = extractRGFromTarGz(embeddedTarGz)
		if loadErr == nil {
			rgHash = sha256.Sum256(rgBytes)
		}
	})
	if loadErr != nil {
		return "", loadErr
	}
	dest, err := datadir.RipgrepBinPath()
	if err != nil {
		return "", err
	}
	if path, ok := cachedPathIfValid(dest); ok {
		return path, nil
	}
	if ok, err := verifyOnDisk(dest); err != nil {
		return "", err
	} else if ok {
		setValidated(dest)
		return dest, nil
	}
	if err := writeAtomic(dest, rgBytes, 0o700); err != nil {
		return "", err
	}
	setValidated(dest)
	return dest, nil
}

func cachedPathIfValid(dest string) (string, bool) {
	validatedMu.Lock()
	v := validated
	validatedMu.Unlock()
	if v == nil || v.path != dest {
		return "", false
	}
	info, err := os.Stat(dest)
	if err != nil {
		return "", false
	}
	if info.Size() == v.size && info.ModTime().Equal(v.mod) {
		return dest, true
	}
	return "", false
}

func verifyOnDisk(dest string) (bool, error) {
	b, err := os.ReadFile(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	gotHash := sha256.Sum256(b)
	return bytes.Equal(gotHash[:], rgHash[:]), nil
}

func setValidated(dest string) {
	info, err := os.Stat(dest)
	if err != nil {
		validatedMu.Lock()
		validated = nil
		validatedMu.Unlock()
		return
	}
	validatedMu.Lock()
	validated = &validatedRG{
		path: dest,
		size: info.Size(),
		mod:  info.ModTime(),
	}
	validatedMu.Unlock()
}

func extractRGFromTarGz(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("rgbin: gzip: %w", err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("rgbin: tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		base := filepath.Base(hdr.Name)
		if base == "rg" || base == "rg.exe" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("rgbin: rg binary not found in embedded tarball")
}

func writeAtomic(dest string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".rg-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, dest)
}
