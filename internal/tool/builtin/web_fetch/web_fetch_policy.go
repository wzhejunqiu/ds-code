package web_fetch

import (
	"fmt"
	"net"
	"strings"
)

func hostAllowed(host string, allowlist []string) bool {
	if len(allowlist) == 0 {
		return false
	}
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || strings.Contains(host, "/") {
		return false
	}
	if isBlockedFetchHost(host) {
		return false
	}
	for _, entry := range allowlist {
		entry = strings.ToLower(strings.TrimSpace(entry))
		if entry == "" {
			continue
		}
		if strings.HasPrefix(entry, "*.") {
			base := entry[2:]
			if len(base) < 3 {
				continue
			}
			if host == base || strings.HasSuffix(host, "."+base) {
				return true
			}
			continue
		}
		if entry == host {
			return true
		}
	}
	return false
}

func isBlockedFetchHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if host == "metadata.google.internal" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return isPrivateOrMetadataIP(ip)
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return true
	}
	for _, ip := range ips {
		if isPrivateOrMetadataIP(ip) {
			return true
		}
	}
	return false
}

func isPrivateOrMetadataIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
		return true
	}
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return true
	}
	return false
}

func validateFetchURLHost(host string, allowlist []string) error {
	if !hostAllowed(host, allowlist) {
		return fmt.Errorf(ErrHostNotAllowlist, host)
	}
	return nil
}
