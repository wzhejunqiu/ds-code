//go:build unix

package versioninfo

import (
	"bufio"
	"bytes"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func linuxPrettyName() string {
	b, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
		}
	}
	return ""
}

func unameRelease() string {
	var u unix.Utsname
	if err := unix.Uname(&u); err != nil {
		return ""
	}
	return unixByteSliceToString(u.Release[:])
}

func unameSysnameReleaseFields() string {
	var u unix.Utsname
	if err := unix.Uname(&u); err != nil {
		return ""
	}
	sys := unixByteSliceToString(u.Sysname[:])
	rel := unixByteSliceToString(u.Release[:])
	if sys == "" {
		return rel
	}
	if rel == "" {
		return sys
	}
	return sys + " " + rel
}

func unixByteSliceToString(bs []byte) string {
	if i := bytes.IndexByte(bs, 0); i >= 0 {
		bs = bs[:i]
	}
	return strings.TrimSpace(string(bs))
}
