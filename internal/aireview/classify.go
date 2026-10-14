// Package aireview builds the local ai-agent review.
// It classifies paths and addresses. It does not open secret files.
package aireview

import (
	"runtime"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/procview"
)

// MaxHosts is how many outbound hosts the tree prints.
const MaxHosts = 30

var sensitiveNeedles = []string{
	"keychain", ".ssh/", "id_rsa", "id_ed25519", "id_ecdsa",
	"certificate", ".pem", ".p12", ".pfx", "passkey",
	".aws/credentials", ".aws/config", ".env", "credentials.json",
	".netrc", ".kube/config", ".npmrc", "cookies", "token",
	"api_key", "apikey",
}

// SensitivePath reports a credential-like path. It reads the path string only.
func SensitivePath(path string) bool {
	low := strings.ToLower(path)
	for _, n := range sensitiveNeedles {
		if strings.Contains(low, n) {
			return true
		}
	}
	return false
}

// credentialNeedles are the sensitive paths that are credentials.
// Cookies and Trust Tokens stay in the sensitive-file count only.
var credentialNeedles = []string{
	"keychain", "certificate",
	"id_rsa", "id_ed25519", "id_ecdsa", ".pem", ".p12", ".pfx",
	"passkey", ".env", "credentials.json", ".netrc", ".aws/credentials",
	"api_key", "apikey", "password",
}

// CredentialPath is a tighter subset of SensitivePath: keychain, certificate
// or private key, passkey, and credential files. It does not match cookies or tokens.
func CredentialPath(path string) bool {
	low := strings.ToLower(path)
	for _, n := range credentialNeedles {
		if strings.Contains(low, n) {
			return true
		}
	}
	return false
}

// Excessive is more than 20 unique outbound hosts, or more than 40 outbound connections.
func Excessive(hosts, conns int) bool {
	return hosts > 20 || conns > 40
}

// FilterCron keeps crontab lines that name the agent or its program. Comments are dropped.
func FilterCron(text, name, program string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if mentions(trim, name, program) {
			out = append(out, trim)
		}
	}
	return out
}

func mentions(text, name, program string) bool {
	if name != "" && procview.MatchesName(text, name) {
		return true
	}
	return program != "" && strings.Contains(text, program)
}

// Mentions is true when text names the agent or its program path.
func Mentions(text, name, program string) bool {
	return mentions(text, name, program)
}

// Argv is what --dry-run prints. PID is the real pid, or the word PID until ps has run.
func Argv(goos, query string) [][]string {
	if goos == "" {
		goos = runtime.GOOS
	}
	pid := "PID"
	if isDigits(query) {
		pid = query
	}
	steps := [][]string{psArgv(goos), lsofFiles(pid), lsofNet(pid), {"crontab", "-l"}}
	if goos == "linux" {
		steps = append(steps, []string{"systemctl", "--user", "list-unit-files", "--no-pager"})
		return steps
	}
	steps = append(steps, []string{"launchctl", "list"})
	return steps
}

func psArgv(goos string) []string {
	cols := "pid=,ppid=,user=,ni=,pri=,stat=,%cpu=,%mem=,etime=,command="
	if goos == "linux" {
		return []string{"ps", "-eww", "-o", "pid=,ppid=,user=,ni=,pri=,stat=,%cpu=,%mem=,etime=,args="}
	}
	return []string{"ps", "-axww", "-o", cols}
}

func lsofFiles(pid string) []string {
	return []string{"lsof", "-n", "-P", "-a", "-p", pid}
}

func lsofNet(pid string) []string {
	return []string{"lsof", "-n", "-P", "-a", "-p", pid, "-i"}
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Family is the seed processes plus every descendant in the local ps table.
func Family(table, seeds []procview.Row) []Proc {
	byPID := map[string]procview.Row{}
	byPPID := map[string][]procview.Row{}
	for _, r := range table {
		byPID[r.PID] = r
		byPPID[r.PPID] = append(byPPID[r.PPID], r)
	}
	seen := map[string]bool{}
	var out []Proc
	var walk func(pid string)
	walk = func(pid string) {
		if pid == "" || seen[pid] {
			return
		}
		seen[pid] = true
		r, ok := byPID[pid]
		if !ok {
			return
		}
		cmd := procview.RedactCommand(r.Command)
		out = append(out, Proc{PID: r.PID, PPID: r.PPID, Program: procview.ProgramPath(cmd), Command: cmd})
		for _, c := range byPPID[pid] {
			if c.PID != pid {
				walk(c.PID)
			}
		}
	}
	for _, s := range seeds {
		walk(s.PID)
	}
	return out
}

// Job is one launchctl list row.
type Job struct {
	PID, Status, Label string
}

// FilterLaunch keeps user-domain launchctl rows whose label contains name.
func FilterLaunch(text, name string) []Job {
	if name == "" {
		return nil
	}
	var out []Job
	for _, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 || f[0] == "PID" {
			continue
		}
		label := f[2]
		if !procview.MatchesName(label, name) {
			continue
		}
		out = append(out, Job{PID: f[0], Status: f[1], Label: label})
	}
	return out
}

// Sock is one lsof -i name.
type Sock struct {
	Raw                    string
	Listen                 bool
	HasArrow               bool
	LocalHost, LocalPort   string
	RemoteHost, RemotePort string
}

// ParseSock reads one lsof NAME. Left of "->" is local. LISTEN has no remote.
func ParseSock(name string) Sock {
	s := Sock{Raw: strings.TrimSpace(name)}
	if strings.Contains(s.Raw, "(LISTEN)") {
		s.Listen = true
	}
	if i := strings.Index(s.Raw, "->"); i >= 0 {
		s.HasArrow = true
		s.LocalHost, s.LocalPort = hostPort(s.Raw[:i])
		s.RemoteHost, s.RemotePort = hostPort(s.Raw[i+2:])
		return s
	}
	if s.Listen {
		s.LocalHost, s.LocalPort = hostPort(s.Raw)
	}
	return s
}

func hostPort(s string) (host, port string) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return "", ""
	}
	token := fields[len(fields)-1]
	if strings.HasPrefix(token, "(") && len(fields) > 1 {
		token = fields[len(fields)-2]
	}
	if strings.HasPrefix(token, "[") {
		end := strings.Index(token, "]")
		if end < 0 || end+1 >= len(token) || token[end+1] != ':' {
			return "", ""
		}
		return token[1:end], token[end+2:]
	}
	i := strings.LastIndex(token, ":")
	if i <= 0 || i == len(token)-1 {
		return "", ""
	}
	return token[:i], token[i+1:]
}

// SplitDirection puts LISTEN, and connections into a listened port, on the inbound side.
// Other "->" rows are outbound. Rows with no arrow and no LISTEN are dropped.
func SplitDirection(socks []Sock) (in, out []Sock) {
	listen := map[string]bool{}
	for _, s := range socks {
		if s.Listen && s.LocalPort != "" {
			listen[s.LocalPort] = true
		}
	}
	for _, s := range socks {
		if s.Listen {
			in = append(in, s)
			continue
		}
		if !s.HasArrow {
			continue
		}
		if s.LocalPort != "" && listen[s.LocalPort] {
			in = append(in, s)
			continue
		}
		out = append(out, s)
	}
	return in, out
}

// PortCount is one remote port under a host.
type PortCount struct {
	Port string
	N    int
}

// HostGroup is one destination and its ports.
type HostGroup struct {
	Host  string
	Ports []PortCount
}

// GroupOutbound counts connections by remote host, then port.
func GroupOutbound(socks []Sock) []HostGroup {
	type key struct{ host, port string }
	n := map[key]int{}
	var hosts []string
	ports := map[string][]string{}
	seenH := map[string]bool{}
	seenP := map[key]bool{}
	for _, s := range socks {
		if s.RemoteHost == "" {
			continue
		}
		k := key{s.RemoteHost, s.RemotePort}
		n[k]++
		if !seenH[s.RemoteHost] {
			seenH[s.RemoteHost] = true
			hosts = append(hosts, s.RemoteHost)
		}
		if !seenP[k] {
			seenP[k] = true
			ports[s.RemoteHost] = append(ports[s.RemoteHost], s.RemotePort)
		}
	}
	var out []HostGroup
	for _, h := range hosts {
		g := HostGroup{Host: h}
		for _, p := range ports[h] {
			g.Ports = append(g.Ports, PortCount{Port: p, N: n[key{h, p}]})
		}
		out = append(out, g)
	}
	return out
}

// Counts returns unique hosts and the number of outbound connections.
func Counts(groups []HostGroup) (hosts, conns int) {
	hosts = len(groups)
	for _, g := range groups {
		for _, p := range g.Ports {
			conns += p.N
		}
	}
	return hosts, conns
}
