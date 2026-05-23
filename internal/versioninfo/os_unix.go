//go:build unix && !darwin

package versioninfo

import (
	"bufio"
	"bytes"
	"os"
	"runtime"
	"strings"
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
