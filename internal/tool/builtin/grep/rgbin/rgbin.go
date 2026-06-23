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

	"github.com/wzhejunqiu/ds-code/internal/datadir"
)

//go:embed rg.tar.gz
var embeddedTarGz []byte

var (
	loadOnce sync.Once
	loadErr  error
	rgBytes  []byte
	rgHash   [32]byte
)

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
	if b, err := os.ReadFile(dest); err == nil {
		gotHash := sha256.Sum256(b)
		if bytes.Equal(gotHash[:], rgHash[:]) {
			return dest, nil
		}
	}
	if err := writeAtomic(dest, rgBytes, 0o700); err != nil {
		return "", err
	}
	return dest, nil
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
