// Package procview composes the one-shot process pages.
// It classifies paths and addresses. It does not open files or read secrets.
package procview

import (
	"net"
	"strings"
)

// AgentNames are the local command lines proc agents looks for.
var AgentNames = []string{
	"cursor", "claude", "codex", "antigravity", "deepseek", "qwen",
	"chatgpt", "copilot", "windsurf", "ollama",
}

// sensitiveNeedles are path fragments. A match is the path itself, never file contents.
var sensitiveNeedles = []string{
	".ssh/", "id_rsa", "id_ed25519", "id_ecdsa",
	".aws/credentials", ".aws/config", ".env", "credentials.json",
	".netrc", ".kube/config", ".npmrc", "cookies", "login keychain", "token",
}

// Sensitive reports whether a path looks like a credential location.
// It compares the path string only. It does not open the file.
func Sensitive(path string) bool {
	low := strings.ToLower(path)
	for _, n := range sensitiveNeedles {
		if strings.Contains(low, n) {
			return true
		}
	}
	return false
}

// Abnormal reports an outbound remote that is a bare IP, or a port other than 80 or 443.
func Abnormal(host, port string) bool {
	if net.ParseIP(host) != nil {
		return true
	}
	return port != "80" && port != "443"
}

// Reason is a short warning for an abnormal remote. Empty when the remote is ordinary HTTPS or HTTP.
func Reason(host, port string) string {
	if !Abnormal(host, port) {
		return ""
	}
	if net.ParseIP(host) != nil {
		return "bare address"
	}
	return "unusual port"
}
