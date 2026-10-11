package dispatch

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
)

func build(n *catalog.Node, args []string, opt Opt) ([]string, error) {
	if n.Special == "" {
		return generic(n, args, opt)
	}
	fn, ok := specials[n.Special]
	if !ok {
		return nil, fmt.Errorf("unknown special %q", n.Special)
	}
	return fn(n, args, opt)
}

func generic(n *catalog.Node, args []string, opt Opt) ([]string, error) {
	base := append([]string{}, n.Argv...)
	switch opt.GOOS {
	case "linux":
		if len(n.ArgvLinux) > 0 {
			base = append([]string{}, n.ArgvLinux...)
		}
	case "darwin":
		if len(n.ArgvDarwin) > 0 {
			base = append([]string{}, n.ArgvDarwin...)
		}
	case "windows":
		if len(n.ArgvWindows) > 0 {
			base = append([]string{}, n.ArgvWindows...)
		}
	}
	if len(base) == 0 && n.Bin != "" {
		base = []string{n.Bin}
	}
	if len(n.Prefer) > 0 && len(base) > 0 {
		if bin := choose(n.Prefer, opt); bin != "" {
			base[0] = bin
		}
	}
	mode := n.UserArgs
	if mode == "" {
		mode = "append"
	}
	switch mode {
	case "none":
		if len(args) > 0 {
			return nil, fmt.Errorf("%s does not take extra arguments", n.Name)
		}
		return base, nil
	case "default":
		if len(args) == 0 {
			return base, nil
		}
		bin := n.Bin
		if bin == "" && len(base) > 0 {
			bin = base[0]
		}
		return append([]string{bin}, args...), nil
	default:
		return append(base, args...), nil
	}
}

func choose(names []string, opt Opt) string {
	for _, name := range names {
		if opt.present(name) {
			return name
		}
	}
	if len(names) == 0 {
		return ""
	}
	return names[len(names)-1]
}

var specials = map[string]func(*catalog.Node, []string, Opt) ([]string, error){}

func init() {
	specials["args-bin"] = func(n *catalog.Node, args []string, opt Opt) ([]string, error) {
		return generic(n, args, opt)
	}
	specials["fixed"] = func(n *catalog.Node, args []string, opt Opt) ([]string, error) {
		if len(args) > 0 {
			return nil, fmt.Errorf("%s does not take extra arguments", strings.Join(append([]string{}, n.Name), " "))
		}
		optCopy := opt
		return generic(n, nil, optCopy)
	}
	specials["cd"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("go needs a path")
		}
		return []string{"cd", args[0]}, nil
	}
	specials["make"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		empty := false
		var rest []string
		for _, a := range args {
			if a == "--empty" {
				empty = true
				continue
			}
			rest = append(rest, a)
		}
		if len(rest) < 1 {
			return nil, fmt.Errorf("make needs a path")
		}
		if empty {
			return append([]string{"touch"}, rest...), nil
		}
		return append([]string{"mkdir"}, rest...), nil
	}
	specials["mkdir"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		for _, a := range args {
			if a == "--empty" {
				return nil, fmt.Errorf("folders make does not take --empty; use: bhuppi files make FILE --empty")
			}
		}
		return append([]string{"mkdir"}, args...), nil
	}
	specials["remove"] = func(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
		flags, paths := splitFlags(args)
		if len(paths) == 1 && !askedRecursive(flags) && isEmpty(opt, paths[0]) {
			return []string{"rmdir", paths[0]}, nil
		}
		out := append([]string{"rm"}, flags...)
		return append(out, paths...), nil
	}
	specials["rmdir"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		return append([]string{"rmdir"}, args...), nil
	}
	specials["find"] = specialFind
	specials["find-type"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("type needs a type letter")
		}
		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}
		return []string{"find", dir, "-type", args[0]}, nil
	}
	specials["find-size"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("size needs a find -size spec")
		}
		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}
		return []string{"find", dir, "-size", args[0]}, nil
	}
	specials["find-time"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("time needs a find -mtime number")
		}
		dir := "."
		if len(args) > 1 {
			dir = args[1]
		}
		return []string{"find", dir, "-mtime", args[0]}, nil
	}
	specials["find-exec"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 3 {
			return nil, fmt.Errorf("exec needs NAME PATH CMD")
		}
		pattern, dir, cmd := args[0], args[1], args[2:]
		out := []string{"find", dir, "-name", pattern, "-exec"}
		out = append(out, cmd...)
		if !contains(cmd, "{}") {
			out = append(out, "{}")
		}
		last := cmd[len(cmd)-1]
		if last != ";" && last != "+" {
			out = append(out, ";")
		}
		return out, nil
	}
	specials["search"] = specialSearch
	specials["awk-filter"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("filter needs a pattern")
		}
		pat := args[0]
		prog := pat
		if !strings.Contains(pat, "{") {
			prog = "/" + strings.ReplaceAll(pat, "/", `\/`) + "/"
		}
		return append([]string{"awk", prog}, args[1:]...), nil
	}
	specials["awk-fields"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("fields needs FIELDS and FILE")
		}
		var refs []string
		for _, f := range strings.Split(args[0], ",") {
			f = strings.TrimSpace(f)
			if f == "" || !isNum(f) {
				return nil, fmt.Errorf("fields must be comma-separated numbers, got %q", args[0])
			}
			refs = append(refs, "$"+f)
		}
		prog := "{print " + strings.Join(refs, ",") + "}"
		return append([]string{"awk", prog}, args[1:]...), nil
	}
	specials["awk-program"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("awk needs a program")
		}
		return append([]string{"awk", args[0]}, args[1:]...), nil
	}
	specials["awk-script"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("script needs a file")
		}
		return append([]string{"awk", "-f"}, args...), nil
	}
	specials["sed-replace"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 3 {
			return nil, fmt.Errorf("replace needs FROM TO FILE")
		}
		return append([]string{"sed", sedSub(args[0], args[1])}, args[2:]...), nil
	}
	specials["sed-delete"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("delete needs PAT FILE")
		}
		pat := strings.ReplaceAll(args[0], "/", `\/`)
		return append([]string{"sed", "/" + pat + "/d"}, args[1:]...), nil
	}
	specials["sed-inplace"] = func(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
		if len(args) < 3 {
			return nil, fmt.Errorf("inplace needs FROM TO FILE")
		}
		script := sedSub(args[0], args[1])
		if opt.GOOS == "darwin" {
			return append([]string{"sed", "-i", "", script}, args[2:]...), nil
		}
		return append([]string{"sed", "-i", script}, args[2:]...), nil
	}
	specials["hash"] = func(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
		algo := "sha256"
		files := args
		if len(args) > 0 {
			switch strings.ToLower(args[0]) {
			case "md5", "sha1", "sha256", "sha512":
				algo = strings.ToLower(args[0])
				files = args[1:]
			}
		}
		if len(files) < 1 {
			return nil, fmt.Errorf("hash needs a file")
		}
		if opt.GOOS == "darwin" {
			if algo == "md5" {
				return append([]string{"md5"}, files...), nil
			}
			bits := map[string]string{"sha1": "1", "sha256": "256", "sha512": "512"}[algo]
			return append([]string{"shasum", "-a", bits}, files...), nil
		}
		bin := map[string]string{"md5": "md5sum", "sha1": "sha1sum", "sha256": "sha256sum", "sha512": "sha512sum"}[algo]
		return append([]string{bin}, files...), nil
	}
	specials["ports"] = func(_ *catalog.Node, _ []string, opt Opt) ([]string, error) {
		switch opt.GOOS {
		case "darwin", "windows":
			return []string{"netstat", "-an"}, nil
		default:
			if opt.present("ss") {
				return []string{"ss", "-lptn"}, nil
			}
			return []string{"netstat", "-lptn"}, nil
		}
	}
	specials["ipaddr"] = func(_ *catalog.Node, _ []string, opt Opt) ([]string, error) {
		switch opt.GOOS {
		case "darwin":
			return []string{"ifconfig"}, nil
		case "windows":
			return []string{"ipconfig"}, nil
		default:
			if opt.present("ip") {
				return []string{"ip", "addr"}, nil
			}
			return []string{"ifconfig", "-a"}, nil
		}
	}
	specials["freemem"] = func(_ *catalog.Node, _ []string, opt Opt) ([]string, error) {
		if opt.GOOS == "darwin" {
			return []string{"vm_stat"}, nil
		}
		return []string{"free", "-h"}, nil
	}
	specials["ping"] = func(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
		flag := "-c"
		if opt.GOOS == "windows" {
			flag = "-n"
		}
		if len(args) > 0 && strings.HasPrefix(args[0], "-") {
			return append([]string{"ping"}, args...), nil
		}
		return append([]string{"ping", flag, "4"}, args...), nil
	}
	specials["dns"] = func(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
		for _, a := range args {
			if strings.Contains(strings.ToLower(a), "axfr") {
				return nil, fmt.Errorf("zone transfer is refused")
			}
		}
		if opt.present("dig") {
			return append([]string{"dig"}, args...), nil
		}
		if opt.present("host") {
			return append([]string{"host"}, args...), nil
		}
		return append([]string{"dig"}, args...), nil
	}
	specials["tcpdump"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("tcpdump needs an interface")
		}
		out := []string{"tcpdump", "-i", args[0], "-n"}
		if len(args) >= 2 {
			if _, err := strconv.Atoi(args[1]); err != nil {
				return nil, fmt.Errorf("COUNT must be a number")
			}
			out = append(out, "-c", args[1])
		}
		if len(args) > 2 {
			return nil, fmt.Errorf("tcpdump accepts IFACE and an optional COUNT only")
		}
		return out, nil
	}
	specials["pkg-search"] = func(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("search needs a package name")
		}
		prefix, err := pkgSearch(opt)
		if err != nil {
			return nil, err
		}
		return append(prefix, args[0]), nil
	}
	specials["pkg-install"] = func(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("install needs a package name")
		}
		prefix, err := pkgInstall(opt)
		if err != nil {
			return nil, err
		}
		return append(prefix, args[0]), nil
	}
	specials["openssl-key"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("key needs an output file")
		}
		return []string{"openssl", "genpkey", "-algorithm", "RSA", "-pkeyopt", "rsa_keygen_bits:2048", "-out", args[0]}, nil
	}
	specials["openssl-csr"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 3 {
			return nil, fmt.Errorf("csr needs KEY SUBJ OUT")
		}
		return []string{"openssl", "req", "-new", "-key", args[0], "-subj", args[1], "-out", args[2]}, nil
	}
	specials["openssl-cert"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("cert needs a file")
		}
		return []string{"openssl", "x509", "-in", args[0], "-noout", "-text"}, nil
	}
	specials["openssl-verify"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("verify needs CA and CERT")
		}
		return []string{"openssl", "verify", "-CAfile", args[0], args[1]}, nil
	}
	specials["openssl-handshake"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("handshake needs a host")
		}
		host := args[0]
		connect := host
		name := host
		if !strings.Contains(host, ":") {
			connect = host + ":443"
		} else {
			name = strings.SplitN(host, ":", 2)[0]
		}
		return []string{"openssl", "s_client", "-connect", connect, "-servername", name}, nil
	}
	specials["launchctl-load"] = func(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("load needs a plist")
		}
		uid := opt.UID
		if uid == "" {
			uid = strconv.Itoa(os.Getuid())
		}
		return []string{"launchctl", "bootstrap", "gui/" + uid, args[0]}, nil
	}
	specials["using"] = specialUsing
	specials["procs"] = specialProcs
	specials["nice"] = specialNice
	specials["pstat"] = specialPstat
	specials["route"] = func(_ *catalog.Node, _ []string, opt Opt) ([]string, error) {
		if opt.GOOS == "darwin" {
			return []string{"route", "-n", "get", "default"}, nil
		}
		if opt.present("ip") {
			return []string{"ip", "route"}, nil
		}
		return []string{"route", "-n"}, nil
	}
	specials["os"] = func(_ *catalog.Node, _ []string, opt Opt) ([]string, error) {
		if opt.GOOS == "windows" {
			return []string{"cmd", "/c", "ver"}, nil
		}
		return []string{"uname", "-a"}, nil
	}
	specials["arch"] = func(_ *catalog.Node, _ []string, opt Opt) ([]string, error) {
		if opt.GOOS == "darwin" || !opt.present("arch") {
			return []string{"uname", "-m"}, nil
		}
		return []string{"arch"}, nil
	}
	specials["otool"] = func(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
		if len(args) == 1 && !strings.HasPrefix(args[0], "-") {
			return []string{"otool", "-L", args[0]}, nil
		}
		return append([]string{"otool"}, args...), nil
	}
	specials["list"] = specialList
	specials["map"] = specialMap
	specials["pmap"] = specialPmap
	specials["stat"] = specialStat
}

func specialUsing(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
	if len(args) != 1 {
		return append([]string{"lsof"}, args...), nil
	}
	a := args[0]
	if a == "" || (strings.HasPrefix(a, "-") && a != "-") {
		return append([]string{"lsof"}, args...), nil
	}
	if port, ok := usingPort(a); ok {
		return []string{"lsof", "-i", port}, nil
	}
	if strings.Contains(a, "/") || pathExists(a) {
		return []string{"lsof", "--", a}, nil
	}
	return []string{"lsof", "-c", a}, nil
}

func usingPort(a string) (string, bool) {
	if strings.HasPrefix(a, ":") && len(a) > 1 && !strings.Contains(a[1:], "/") {
		return a, true
	}
	if isNum(a) {
		return ":" + a, true
	}
	return "", false
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func specialNice(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
	if len(args) == 0 {
		// macOS /usr/bin/nice with no utility prints usage. The niceness is the ps ni column.
		if opt.GOOS == "darwin" {
			return []string{"ps", "-o", "ni=", "-p", strconv.Itoa(os.Getpid())}, nil
		}
		return []string{"nice"}, nil
	}
	return append([]string{"nice"}, args...), nil
}

func specialProcs(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
	if len(args) == 0 {
		return []string{"ps", "aux"}, nil
	}
	if strings.HasPrefix(args[0], "-") {
		return append([]string{"ps"}, args...), nil
	}
	if len(args) == 1 {
		return []string{"pgrep", "-lf", args[0]}, nil
	}
	return append([]string{"ps"}, args...), nil
}

func specialPstat(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
	if len(args) != 1 || !isNum(args[0]) {
		return nil, fmt.Errorf("proc stat takes one pid")
	}
	pid := args[0]
	if opt.GOOS == "darwin" {
		return []string{"ps", "-p", pid, "-o", "pid=,user=,uid=,ni=,pri=,stat=,%cpu=,%mem=,etime=,command="}, nil
	}
	return []string{"ps", "-p", pid, "-o", "pid,user,uid,ni,pri,stat,%cpu,%mem,etime,args"}, nil
}

func specialList(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
	long, hidden := false, false
	var rest []string
	for _, a := range args {
		switch a {
		case "--long":
			long = true
		case "--hidden":
			hidden = true
		default:
			rest = append(rest, a)
		}
	}
	if !long && !hidden {
		return append([]string{"ls"}, args...), nil
	}
	var paths, flags []string
	for _, a := range rest {
		if strings.HasPrefix(a, "-") && a != "-" {
			flags = append(flags, a)
			continue
		}
		paths = append(paths, a)
	}
	if hidden {
		if len(flags) > 0 {
			return nil, fmt.Errorf("list --hidden takes a path, not extra flags")
		}
		if len(paths) > 1 {
			return nil, fmt.Errorf("list --hidden takes one path")
		}
		path := "."
		if len(paths) == 1 {
			path = paths[0]
		}
		argv := []string{"find", path, "-mindepth", "1", "-maxdepth", "1", "-name", ".*", "!", "-name", ".", "!", "-name", ".."}
		if long {
			argv = append(argv, "-exec", "ls", "-lahd", "{}", "+")
		}
		return argv, nil
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	argv := []string{"ls", "-lah"}
	argv = append(argv, paths...)
	argv = append(argv, flags...)
	return argv, nil
}

func specialMap(_ *catalog.Node, args []string, _ Opt) ([]string, error) {
	depth := "3"
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--depth" {
			if i+1 >= len(args) || !positiveInt(args[i+1]) {
				return nil, fmt.Errorf("map --depth needs a positive integer")
			}
			depth = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(a, "--depth=") {
			v := strings.TrimPrefix(a, "--depth=")
			if !positiveInt(v) {
				return nil, fmt.Errorf("map --depth needs a positive integer")
			}
			depth = v
			continue
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			return nil, fmt.Errorf("map does not take %s", a)
		}
		paths = append(paths, a)
	}
	if len(paths) > 1 {
		return nil, fmt.Errorf("map takes one path")
	}
	path := "."
	if len(paths) == 1 {
		path = paths[0]
	}
	// find only discovers entries. bhuppi draws the tree afterwards.
	return []string{"find", path, "-maxdepth", depth}, nil
}

func specialPmap(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
	if _, err := takeDepth(args); err != nil {
		return nil, err
	}
	if opt.GOOS == "linux" {
		return []string{"ps", "-eo", "pid=,ppid=,user=,ni=,stat=,args="}, nil
	}
	return []string{"ps", "-axo", "pid=,ppid=,user=,ni=,stat=,command="}, nil
}

func takeDepth(args []string) (string, error) {
	depth := "3"
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--depth" {
			if i+1 >= len(args) || !positiveInt(args[i+1]) {
				return "", fmt.Errorf("map --depth needs a positive integer")
			}
			depth = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(a, "--depth=") {
			v := strings.TrimPrefix(a, "--depth=")
			if !positiveInt(v) {
				return "", fmt.Errorf("map --depth needs a positive integer")
			}
			depth = v
			continue
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			return "", fmt.Errorf("map does not take %s", a)
		}
	}
	return depth, nil
}

func specialStat(_ *catalog.Node, args []string, opt Opt) ([]string, error) {
	if len(args) != 1 || strings.HasPrefix(args[0], "-") {
		return nil, fmt.Errorf("stat takes one path")
	}
	if opt.GOOS == "darwin" {
		return []string{"stat", "-x", args[0]}, nil
	}
	return []string{"stat", args[0]}, nil
}

func positiveInt(s string) bool {
	if !isNum(s) || s == "0" {
		return false
	}
	return true
}

func pkgSearch(opt Opt) ([]string, error) {
	switch {
	case opt.GOOS == "darwin" && opt.present("brew"):
		return []string{"brew", "search"}, nil
	case opt.present("apt-get"):
		return []string{"apt-cache", "search"}, nil
	case opt.present("dnf"):
		return []string{"dnf", "search"}, nil
	case opt.present("yum"):
		return []string{"yum", "search"}, nil
	case opt.present("brew"):
		return []string{"brew", "search"}, nil
	default:
		return nil, fmt.Errorf("no package manager on PATH (brew, apt, dnf, or yum)")
	}
}

func pkgInstall(opt Opt) ([]string, error) {
	switch {
	case opt.GOOS == "darwin" && opt.present("brew"):
		return []string{"brew", "install"}, nil
	case opt.present("apt-get"):
		return []string{"sudo", "apt-get", "install"}, nil
	case opt.present("dnf"):
		return []string{"sudo", "dnf", "install"}, nil
	case opt.present("yum"):
		return []string{"sudo", "yum", "install"}, nil
	case opt.present("brew"):
		return []string{"brew", "install"}, nil
	default:
		return nil, fmt.Errorf("no package manager on PATH (brew, apt, dnf, or yum)")
	}
}

func sedSub(from, to string) string {
	delim := "/"
	if strings.Contains(from, "/") || strings.Contains(to, "/") {
		delim = "|"
	}
	esc := func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, delim, `\`+delim)
		s = strings.ReplaceAll(s, "&", `\&`)
		s = strings.ReplaceAll(s, "\n", "")
		return s
	}
	return "s" + delim + esc(from) + delim + esc(to) + delim + "g"
}

func splitFlags(args []string) (flags, paths []string) {
	dash := false
	for _, a := range args {
		if a == "--" {
			dash = true
			continue
		}
		if !dash && strings.HasPrefix(a, "-") && a != "-" {
			flags = append(flags, a)
			continue
		}
		paths = append(paths, a)
	}
	return flags, paths
}

func askedRecursive(flags []string) bool {
	for _, f := range flags {
		if f == "--recursive" {
			return true
		}
		if strings.HasPrefix(f, "--") {
			continue
		}
		if strings.HasPrefix(f, "-") && strings.ContainsAny(strings.TrimLeft(f, "-"), "rR") {
			return true
		}
	}
	return false
}

func isEmpty(opt Opt, path string) bool {
	if opt.IsEmptyDir != nil {
		return opt.IsEmptyDir(path)
	}
	st, err := os.Stat(path)
	if err != nil || !st.IsDir() {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	return err != nil
}

func isNum(s string) bool {
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

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

// KnownSpecial reports whether the catalog special is implemented.
func KnownSpecial(name string) bool {
	if name == "" {
		return true
	}
	_, ok := specials[name]
	return ok
}
