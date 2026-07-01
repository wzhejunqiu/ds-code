package permission

import (
	"fmt"
	"net"
	"strings"
)

const (
	errHostNotAllowlist = "主机 %q 不在 web.allowlist 中"
	errBlockedFetchHost = "web_fetch: 禁止访问的主机 %q"
	errDNSLookup        = "web_fetch: 解析主机 %q 失败: %w"
	errBlockedIP        = "web_fetch: 禁止访问的 IP %s（主机 %q）"

	// cloudMetadataIPv4Addr is the link-local address shared by AWS, GCP, Azure,
	// and other clouds for the instance metadata service (IMDS). SSRF to this host
	// can leak temporary IAM credentials and other instance secrets.
	cloudMetadataIPv4Addr = "169.254.169.254"
)

var cloudMetadataIPv4 = net.ParseIP(cloudMetadataIPv4Addr)

// CheckFetchSSRF blocks localhost, metadata, and private/reserved IPs (all modes).
func CheckFetchSSRF(host string) error {
	if isBlockedFetchHost(host) {
		return fmt.Errorf(errBlockedFetchHost, host)
	}
	return nil
}

// CheckResolvedFetchHost validates a resolved dial target (DNS + private IP).
func CheckResolvedFetchHost(host string) error {
	if isBlockedFetchHost(host) {
		return fmt.Errorf(errBlockedFetchHost, host)
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf(errDNSLookup, host, err)
	}
	for _, ip := range ips {
		if IsPrivateOrMetadataIP(ip) {
			return fmt.Errorf(errBlockedIP, ip, host)
		}
	}
	return nil
}

// IsPrivateOrMetadataIP reports loopback, link-local, private, or cloud metadata IPs.
func IsPrivateOrMetadataIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
		return true
	}
	return ip.Equal(cloudMetadataIPv4)
}

func normalizeFetchHost(host string) string {
	return strings.ToLower(strings.TrimSpace(host))
}

func hostAllowed(host string, allowlist []string) bool {
	host = normalizeFetchHost(host)
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

func (e *Engine) hostInAllowlist(host string) bool {
	return hostAllowed(host, e.WebAllowlist)
}

func isBlockedFetchHost(host string) bool {
	host = normalizeFetchHost(host)
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	if host == "metadata.google.internal" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return IsPrivateOrMetadataIP(ip)
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return true
	}
	for _, ip := range ips {
		if IsPrivateOrMetadataIP(ip) {
			return true
		}
	}
	return false
}

func appendUniqueAllowlist(list []string, host string) []string {
	host = normalizeFetchHost(host)
	if host == "" {
		return list
	}
	for _, entry := range list {
		if normalizeFetchHost(entry) == host {
			return list
		}
	}
	return append(list, host)
}
