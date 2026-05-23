package textfile

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

const sniffLimit = 3072

var blockedExt = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}, ".ico": {},
	".pdf": {}, ".zip": {}, ".gz": {}, ".tar": {}, ".7z": {}, ".rar": {},
	".exe": {}, ".dll": {}, ".so": {}, ".dylib": {}, ".wasm": {},
	".o": {}, ".a": {}, ".pyc": {}, ".class": {}, ".jar": {},
	".woff": {}, ".woff2": {}, ".ttf": {}, ".otf": {},
	".mp3": {}, ".mp4": {}, ".mov": {}, ".avi": {}, ".mkv": {},
	".sqlite": {}, ".db": {},
}

var textApplicationMIMEs = []string{
	"application/json",
	"application/xml",
	"application/javascript",
	"application/x-yaml",
	"application/yaml",
	"application/sql",
	"application/x-sh",
	"application/x-httpd-php",
	"application/ld+json",
	"application/xhtml+xml",
}

// IsSearchable reports whether a file is likely text and safe to grep/glob-list.
// Errors opening or reading the file return false.
func IsSearchable(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if _, blocked := blockedExt[ext]; blocked {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, sniffLimit)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return false
	}
	buf = buf[:n]
	if n == 0 {
		return true
	}
	if bytes.IndexByte(buf, 0) >= 0 {
		return false
	}

	mime := mimetype.Detect(buf)
	if mime.Is("text/*") {
		return true
	}
	for _, t := range textApplicationMIMEs {
		if mime.Is(t) {
			return true
		}
	}
	if mime.Is("image/*") || mime.Is("video/*") || mime.Is("audio/*") {
		return false
	}
	if mime.Is("application/pdf") ||
		mime.Is("application/zip") ||
		mime.Is("application/gzip") ||
		mime.Is("application/x-gzip") ||
		mime.Is("application/x-tar") ||
		mime.Is("application/java-archive") ||
		strings.HasPrefix(mime.String(), "application/vnd.") {
		return false
	}
	return true
}
