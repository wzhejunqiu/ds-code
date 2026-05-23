//go:build unix && !darwin

package versioninfo

import (
	"runtime"
)

func platformOSVersion() string {
	switch runtime.GOOS {
	case "linux":
		if v := linuxPrettyName(); v != "" {
			return v
		}
	}
	return unameSysnameReleaseFields()
}

func platformKernelVersion() string {
	return unameRelease()
}
