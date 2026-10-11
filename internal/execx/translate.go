package execx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Translate maps a Unix argv onto the Windows native command when Git usr/bin
// is not providing the real binary. find.exe and sort.exe in System32 are
// never treated as Unix find or sort.
func Translate(goos string, argv []string, hasGit func(string) bool) ([]string, error) {
	if goos != "windows" || len(argv) == 0 {
		return argv, nil
	}
	bin := argv[0]
	if hasGit != nil && hasGit(bin) {
		return append([]string{}, argv...), nil
	}
	switch bin {
	case "find", "sort":
		return nil, fmt.Errorf("Windows %s.exe is not Unix %s. Install Git for Windows (usr\\bin) so the real binary is used. bhuppi will not run System32\\%s.exe", bin, bin, bin)
	case "grep":
		rest := argv[1:]
		if len(rest) > 0 && rest[0] == "-n" {
			rest = rest[1:]
		}
		return append([]string{"findstr", "/n"}, rest...), nil
	case "ps":
		if len(argv) == 1 || (len(argv) == 2 && argv[1] == "aux") {
			return []string{"tasklist"}, nil
		}
		return nil, fmt.Errorf("ps flags need Git for Windows usr\\bin\\ps.exe. tasklist is only the no-flag form")
	case "ping":
		out := append([]string{}, argv...)
		for i, a := range out {
			if a == "-c" {
				out[i] = "-n"
			}
		}
		return out, nil
	}
	if pref, ok := nativePrefix[bin]; ok {
		out := append([]string{}, pref...)
		return append(out, argv[1:]...), nil
	}
	return append([]string{}, argv...), nil
}

var nativePrefix = map[string][]string{
	"ls":    {"cmd", "/c", "dir"},
	"cat":   {"cmd", "/c", "type"},
	"cp":    {"cmd", "/c", "copy"},
	"mv":    {"cmd", "/c", "move"},
	"rm":    {"cmd", "/c", "del"},
	"rmdir": {"cmd", "/c", "rmdir"},
	"mkdir": {"cmd", "/c", "mkdir"},
	"pwd":   {"cmd", "/c", "cd"},
	"touch": {"cmd", "/c", "copy", "/y", "NUL"},
}

// HasGitBin reports whether Git for Windows usr/bin provides this tool.
func HasGitBin(bin, pathEnv string) bool {
	return gitUsr(bin, pathEnv) != ""
}

func gitUsr(bin, pathEnv string) string {
	if pathEnv == "" {
		pathEnv = os.Getenv("PATH")
	}
	names := []string{bin + ".exe", bin}
	for _, dir := range filepath.SplitList(pathEnv) {
		low := strings.ToLower(filepath.ToSlash(dir))
		git := strings.Contains(low, "git/usr/bin")
		msys := (strings.Contains(low, "msys") || strings.Contains(low, "cygwin")) && strings.Contains(low, "usr/bin")
		if !git && !msys {
			continue
		}
		for _, name := range names {
			cand := filepath.Join(dir, name)
			if st, err := os.Stat(cand); err == nil && !st.IsDir() {
				return cand
			}
		}
	}
	return ""
}

// Look finds bin. On Windows it prefers Git usr/bin and rejects System32 find/sort.
func Look(bin, goos, pathEnv string) (string, bool) {
	if goos == "windows" {
		if p := gitUsr(bin, pathEnv); p != "" {
			return p, true
		}
	}
	if pathEnv != "" {
		for _, dir := range filepath.SplitList(pathEnv) {
			for _, name := range []string{bin, bin + ".exe"} {
				cand := filepath.Join(dir, name)
				st, err := os.Stat(cand)
				if err == nil && !st.IsDir() {
					if goos == "windows" && falseFriend(bin, cand) {
						return "", false
					}
					return cand, true
				}
			}
		}
	}
	p, err := exec.LookPath(bin)
	if err != nil {
		return "", false
	}
	if goos == "windows" && falseFriend(bin, p) {
		return "", false
	}
	return p, true
}

func falseFriend(bin, path string) bool {
	if bin != "find" && bin != "sort" {
		return false
	}
	low := strings.ToLower(filepath.ToSlash(path))
	return strings.Contains(low, "/windows/system32/") || strings.Contains(low, "/windows/syswow64/")
}
