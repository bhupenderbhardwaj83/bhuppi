package reputation

import (
	"net"
	"sort"
)

// Checkable reports a public IP. Loopback, RFC1918, link-local, multicast,
// and unspecified addresses are not sent to a resolver or a reputation service.
func Checkable(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return false
	}
	return true
}

// Public returns each checkable address once, in lexical order.
// The key is the canonical IP, so 127.0.0.1 never appears.
func Public(hosts []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, host := range hosts {
		if !Checkable(host) {
			continue
		}
		key := net.ParseIP(host).String()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
