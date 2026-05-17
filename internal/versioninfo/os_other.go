//go:build !unix

package versioninfo

func platformOSVersion() string {
	return ""
}

func platformKernelVersion() string {
	return ""
}
