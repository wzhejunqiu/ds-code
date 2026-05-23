//go:build unix

package versioninfo

import (
	"bytes"
	"strings"

	"golang.org/x/sys/unix"
)

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
