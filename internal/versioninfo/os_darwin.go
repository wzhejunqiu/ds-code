//go:build darwin

package versioninfo

import (
	"strings"

	"golang.org/x/sys/unix"
)

func platformOSVersion() string {
	if v, err := unix.Sysctl("kern.osproductversion"); err == nil {
		return strings.TrimSpace(string(v))
	}
	return unameSysnameReleaseFields()
}

func platformKernelVersion() string {
	if v, err := unix.Sysctl("kern.osrelease"); err == nil {
		return strings.TrimSpace(string(v))
	}
	return unameRelease()
}
