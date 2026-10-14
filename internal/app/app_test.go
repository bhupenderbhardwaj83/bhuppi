package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
	"github.com/bhupenderbhardwaj/bhuppi/internal/complete"
	"github.com/bhupenderbhardwaj/bhuppi/internal/dispatch"
	"github.com/bhupenderbhardwaj/bhuppi/internal/help"
)

func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, &out, &errb, func(argv []string, stdinNull bool) (int, error) {
		cmd := exec.Command(argv[0], argv[1:]...)
		cmd.Stdout = &out
		cmd.Stderr = &errb
		if stdinNull {
			cmd.Stdin = nil
		}
		err := cmd.Run()
		if err == nil {
			return 0, nil
		}
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), nil
		}
		return 1, err
	})
	return out.String(), errb.String(), code
}

func TestHelpTACU(t *testing.T) {
	pages := [][]string{
		{"--plain", "help"},
		{"--plain", "--help"},
		{"--plain", "-h"},
		{"--plain", "h"},
		{"--plain", "help", "files"},
		{"--plain", "files", "--help"},
		{"--plain", "files", "-h"},
		{"--plain", "files", "help"},
		{"--plain", "help", "files", "list"},
		{"--plain", "files", "list", "--help"},
		{"--plain", "files", "list", "-h"},
		{"--plain", "files", "list", "help"},
		{"--plain", "help", "text", "awk"},
		{"--plain", "text", "awk", "--help"},
		{"--plain", "help", "text", "awk", "fields"},
		{"--plain", "text", "awk", "fields", "--help"},
		{"--plain", "text", "awk", "fields", "-h"},
		{"--plain", "text", "awk", "fields", "help"},
	}
	var overview, filesPage, listPage, awkPage, fieldsPage string
	for _, args := range pages {
		out, errb, code := run(t, args...)
		if code != 0 {
			t.Fatalf("%v code %d stderr %s", args, code, errb)
		}
		for _, want := range []string{"WHAT:", "WHEN:", "RULE:", "Shape:", "e.g.", "NOT THIS", "more:", "--dry-run", "--plain", "next page"} {
			if !strings.Contains(out, want) {
				t.Errorf("%v missing %q", args, want)
			}
		}
		switch strings.Join(args, " ") {
		case "--plain help", "--plain --help", "--plain -h", "--plain h":
			if overview == "" {
				overview = out
			} else if out != overview {
				t.Fatalf("overview mismatch for %v", args)
			}
		case "--plain help files", "--plain files --help", "--plain files -h", "--plain files help":
			if filesPage == "" {
				filesPage = out
			} else if out != filesPage {
				t.Fatalf("files help mismatch")
			}
		case "--plain help files list", "--plain files list --help", "--plain files list -h", "--plain files list help":
			if listPage == "" {
				listPage = out
			} else if out != listPage {
				t.Fatal("files list help mismatch")
			}
		case "--plain help text awk", "--plain text awk --help":
			if awkPage == "" {
				awkPage = out
			} else if out != awkPage {
				t.Fatal("awk help mismatch")
			}
		case "--plain help text awk fields", "--plain text awk fields --help", "--plain text awk fields -h", "--plain text awk fields help":
			if fieldsPage == "" {
				fieldsPage = out
			} else if out != fieldsPage {
				t.Fatal("fields help mismatch")
			}
		}
	}
	for _, want := range []string{"PICK ONE", "PROGRESSIVE HELP", "ALIASES", "GLOBAL OPTIONS", "b h u p p i", "Wrap the real binary. Teach the shape. Never pretend GNU is BSD.", "✨"} {
		if !strings.Contains(overview, want) {
			t.Errorf("overview missing %q", want)
		}
	}
	if strings.Contains(overview, "╭") {
		t.Fatal("plain overview should not draw the identity box")
	}
	boxed, _, code := run(t, "help")
	if code != 0 || !strings.Contains(boxed, "╭") || !strings.Contains(boxed, "┌ PROGRESSIVE HELP") || !strings.Contains(boxed, "└") {
		t.Fatalf("boxed overview missing TACU frame\n%s", boxed[:min(400, len(boxed))])
	}
	if !strings.Contains(filesPage, "bhuppi files — everyday files") {
		t.Fatal("files title")
	}
	if !strings.Contains(fieldsPage, "awk '{print $1,$3}'") && !strings.Contains(fieldsPage, "{print $1,$3}") {
		t.Fatalf("fields page should show the awk default:\n%s", fieldsPage)
	}
}

func TestDangerMenus(t *testing.T) {
	top, _, code := run(t, "--plain")
	if code != 0 {
		t.Fatal(code)
	}
	if strings.Contains(top, "WHAT:") || strings.Contains(top, "\x1b") {
		t.Fatal("bare bhuppi should be the plain command menu")
	}
	if !strings.Contains(top, "── everyday ──") || !strings.Contains(top, "── machine ──") {
		t.Fatalf("missing section headers:\n%s", top)
	}
	for _, ln := range strings.Split(top, "\n") {
		trim := strings.TrimSpace(ln)
		if trim == "" || strings.HasPrefix(trim, "──") {
			continue
		}
		for _, sec := range []string{"guide", "machine", "everyday", "setup", "build"} {
			if strings.HasSuffix(trim, " "+sec) {
				t.Fatalf("section id on row: %q", ln)
			}
		}
	}
	filesRow := commandRow(top, "files")
	if !strings.Contains(filesRow, "bhuppi files") || !strings.Contains(filesRow, "e.g.") {
		t.Fatalf("files group row: %q", filesRow)
	}
	colored, err := help.Menu(catalogLoad(t), nil, help.Options{Color: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(colored, "\x1b") || strings.Contains(colored, "^[") {
		t.Fatal("colored menu should emit a real ESC, not a caret sequence")
	}
	files, _, code := run(t, "--plain", "files")
	if code != 0 {
		t.Fatal(code)
	}
	if strings.Contains(files, "WHAT:") {
		t.Fatal("bare files menu became the help page")
	}
	listRow := commandRow(files, "list")
	removeRow := commandRow(files, "remove")
	if !strings.Contains(listRow, "bhuppi files list") || !strings.Contains(listRow, "e.g.") {
		t.Fatalf("list row: %q", listRow)
	}
	if !strings.HasPrefix(strings.TrimSpace(removeRow), "!") || !strings.Contains(removeRow, "e.g.") || !strings.Contains(removeRow, "bhuppi files remove") {
		t.Fatalf("remove row: %q", removeRow)
	}
	if !menuBang(files, "remove") {
		t.Fatalf("files remove unmarked:\n%s", files)
	}
	if menuBang(files, "list") || menuBang(files, "show") || menuBang(files, "move") {
		t.Fatal("safe files verb was marked")
	}
	proc, _, code := run(t, "--plain", "proc")
	if code != 0 {
		t.Fatal(code)
	}
	if !menuBang(proc, "kill") || !menuBang(proc, "killall") || !menuBang(proc, "pkill") {
		t.Fatalf("proc signals unmarked:\n%s", proc)
	}
	if menuBang(proc, "list") || menuBang(proc, "view") || menuBang(proc, "agents") || menuBang(proc, "renice") {
		t.Fatal("safe proc verb was marked")
	}
	head := proc
	if lines := strings.Split(proc, "\n"); len(lines) > 30 {
		head = strings.Join(lines[:30], "\n")
	}
	if !strings.Contains(head, "! kill") {
		t.Fatalf("first lines miss ! kill:\n%s", head)
	}
	if strings.Contains(files, "\x1b") {
		t.Fatal("plain menu has ANSI")
	}
}

func commandRow(page, name string) string {
	for _, ln := range strings.Split(page, "\n") {
		f := strings.Fields(ln)
		if len(f) == 0 {
			continue
		}
		if f[0] == name || (f[0] == "!" && len(f) > 1 && f[1] == name) {
			return ln
		}
	}
	return ""
}

func menuBang(page, name string) bool {
	for _, ln := range strings.Split(page, "\n") {
		f := strings.Fields(ln)
		if len(f) >= 2 && f[0] == "!" && f[1] == name {
			return true
		}
	}
	return false
}

func TestDryRun(t *testing.T) {
	out, errb, code := run(t, "--dry-run", "files", "list", ".")
	if code != 0 {
		t.Fatal(code, errb)
	}
	if strings.TrimSpace(out) != "ls ." {
		t.Fatalf("got %q", out)
	}
	out, errb, code = run(t, "--dry-run", "text", "awk", "fields", "1,3", "a.log")
	if code != 0 {
		t.Fatal(errb)
	}
	if strings.TrimSpace(out) != "awk '{print $1,$3}' a.log" {
		t.Fatalf("got %q", out)
	}
	out, _, code = run(t, "--dry-run", "text", "sed", "inplace", "foo", "bar", "f")
	if code != 0 {
		t.Fatal(out)
	}
	got := strings.TrimSpace(out)
	switch runtime.GOOS {
	case "darwin":
		if got != "! sed -i '' s/foo/bar/g f  ! edits files in place" {
			t.Fatalf("darwin sed: %q", got)
		}
	default:
		if got != "! sed -i s/foo/bar/g f  ! edits files in place" {
			t.Fatalf("gnu sed: %q", got)
		}
	}
	out, _, code = run(t, "--dry-run", "pack", "tar", "compress", "logs.tgz", "logs")
	if code != 0 || strings.TrimSpace(out) != "tar -czf logs.tgz logs" {
		t.Fatalf("tar %q", out)
	}
	out, _, code = run(t, "--dry-run", "net", "ssh", "connect", "user@host")
	if code != 0 || strings.TrimSpace(out) != "ssh user@host" {
		t.Fatalf("ssh %q", out)
	}
}

func TestExplainAndNmap(t *testing.T) {
	out, errb, code := run(t, "explain", "awk")
	if code != 0 {
		t.Fatal(errb)
	}
	if !strings.HasPrefix(out, "text awk\n") {
		t.Fatalf("explain awk:\n%s", out)
	}
	out, errb, code = run(t, "net", "nmap", "localhost")
	if code != 0 {
		t.Fatal(code, errb)
	}
	for _, banned := range []string{"-sS", "-sV", "-O", "-p-"} {
		if strings.Contains(out, banned) {
			t.Fatalf("nmap page leaked %s\n%s", banned, out)
		}
	}
	if !strings.Contains(out, "docs-only") && !strings.Contains(errb, "docs-only") {
		t.Fatal("nmap should say docs-only")
	}
	if !strings.Contains(out, "https://nmap.org/book/man.html") {
		t.Fatal("nmap docs link missing")
	}
}

func TestCatalogInvariants(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Groups) != 15 {
		t.Fatal(len(cat.Groups))
	}
	if len(cat.Explain) < 164 {
		t.Fatal(len(cat.Explain))
	}
	if _, ok := cat.Explain["awk"]; !ok {
		t.Fatal("awk")
	}
	if cat.Explain["nmap"].Delivery != "docs-only" {
		t.Fatal(cat.Explain["nmap"])
	}
	files := cat.Group("files")
	want := []string{"list", "show", "where", "go", "make", "copy", "move", "remove", "find"}
	for _, name := range want {
		if catalog.Child(files, name) == nil {
			t.Errorf("files missing %s", name)
		}
	}
	if catalog.Child(files, "open") == nil {
		t.Fatal("files open should alias show")
	}
	if catalog.Child(files, "delete") != nil || catalog.Child(cat.Group("folders"), "delete") != nil {
		t.Fatal("delete verb")
	}
	if catalog.Child(cat.Group("folders"), "open") != nil {
		t.Fatal("folders open")
	}
	if cat.Explain["open"].Target != "files open" {
		t.Fatalf("explain open: %+v", cat.Explain["open"])
	}
	for _, gname := range []string{"files", "folders"} {
		g := cat.Group(gname)
		for _, name := range []string{"map", "stat"} {
			if catalog.Child(g, name) == nil {
				t.Errorf("%s missing %s", gname, name)
			}
		}
	}
	proc := cat.Group("proc")
	if catalog.Child(proc, "stat") == nil {
		t.Fatal("proc stat")
	}
	var walk func(*catalog.Node)
	walk = func(n *catalog.Node) {
		if !dispatch.KnownSpecial(n.Special) {
			t.Errorf("unknown special %s on %s", n.Special, n.Name)
		}
		for i := range n.Children {
			walk(&n.Children[i])
		}
	}
	for i := range cat.Groups {
		walk(&cat.Groups[i])
	}
}

func TestFindBackends(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	withFD := dispatch.Opt{GOOS: "linux", DryRun: true, Lookup: func(name string) (string, bool) {
		if name == "fd" || name == "find" || name == "ls" {
			return "/bin/" + name, true
		}
		return "", false
	}}
	p := dispatch.Resolve(cat, []string{"files", "find", "*.go", "."}, withFD)
	if dispatch.FormatArgv(p.Argv) != "fd --glob '*.go' ." {
		t.Fatalf("fd: %v", p.Argv)
	}
	noFD := withFD
	noFD.Lookup = func(name string) (string, bool) {
		if name == "find" {
			return "/bin/find", true
		}
		return "", false
	}
	p = dispatch.Resolve(cat, []string{"files", "find", "*.go", "."}, noFD)
	if dispatch.FormatArgv(p.Argv) != "find . -name '*.go'" {
		t.Fatalf("find: %#v %s", p, p.Text)
	}
}

func TestRemoveNeverAddsRF(t *testing.T) {
	cat, _ := catalog.Load()
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "a")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	opt := dispatch.Opt{GOOS: "linux", DryRun: true}
	p := dispatch.Resolve(cat, []string{"files", "remove", empty}, opt)
	if p.Argv[0] != "rmdir" {
		t.Fatalf("empty dir: %v", p.Argv)
	}
	p = dispatch.Resolve(cat, []string{"files", "remove", file}, opt)
	if strings.Join(p.Argv, " ") != "rm "+file {
		t.Fatalf("file: %v", p.Argv)
	}
	for _, a := range p.Argv {
		if a == "-rf" || a == "-r" || a == "-f" {
			t.Fatal("injected flag")
		}
	}
	p = dispatch.Resolve(cat, []string{"files", "remove", "-rf", file}, opt)
	if strings.Join(p.Argv, " ") != "rm -rf "+file {
		t.Fatalf("user -rf: %v", p.Argv)
	}
	p = dispatch.Resolve(cat, []string{"folders", "remove", empty}, opt)
	if p.Argv[0] != "rmdir" {
		t.Fatalf("folders: %v", p.Argv)
	}
}

func TestMakeAndSearch(t *testing.T) {
	cat, _ := catalog.Load()
	opt := dispatch.Opt{GOOS: "darwin", DryRun: true, Lookup: func(name string) (string, bool) {
		return "", name == "grep"
	}}
	p := dispatch.Resolve(cat, []string{"files", "make", "tmp"}, opt)
	if strings.Join(p.Argv, " ") != "mkdir tmp" {
		t.Fatal(p.Argv)
	}
	p = dispatch.Resolve(cat, []string{"files", "make", "a", "--empty"}, opt)
	if strings.Join(p.Argv, " ") != "touch a" {
		t.Fatal(p.Argv)
	}
	p = dispatch.Resolve(cat, []string{"text", "search", "TODO", "src"}, opt)
	if strings.Join(p.Argv, " ") != "grep -n -F TODO src" {
		t.Fatalf("grep %v", p.Argv)
	}
	opt.Lookup = func(name string) (string, bool) { return "/bin/" + name, name == "rg" }
	p = dispatch.Resolve(cat, []string{"text", "search", "TODO", "src"}, opt)
	if strings.Join(p.Argv, " ") != "rg -F TODO src" {
		t.Fatalf("rg %v", p.Argv)
	}
}

func TestLiveJobs(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(src, []byte("a b c\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errb, code := run(t, "files", "show", src)
	if code != 0 {
		t.Fatal(errb)
	}
	if out != "a b c\n" {
		t.Fatalf("show %q", out)
	}
	dst := filepath.Join(dir, "b.txt")
	if _, errb, code = run(t, "files", "copy", src, dst); code != 0 {
		t.Fatal(errb)
	}
	body, err := os.ReadFile(dst)
	if err != nil || string(body) != "a b c\n" {
		t.Fatalf("copy %q %v", body, err)
	}
	out, errb, code = run(t, "files", "list", dir)
	if code != 0 || !strings.Contains(out, "a.txt") {
		t.Fatalf("list %q %s", out, errb)
	}
	out, errb, code = run(t, "text", "awk", "fields", "1,3", src)
	if code != 0 {
		t.Fatal(errb)
	}
	if out != "a c\n" {
		t.Fatalf("awk %q", out)
	}
	logs := filepath.Join(dir, "logs")
	if err := os.Mkdir(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logs, "n.txt"), []byte("n"), 0o644); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "logs.tgz")
	if _, errb, code = run(t, "pack", "tar", "compress", archive, logs); code != 0 {
		t.Fatal(errb)
	}
	if _, err := os.Stat(archive); err != nil {
		t.Fatal(err)
	}
	if _, errb, code = run(t, "auth", "who"); code != 0 {
		t.Fatal(errb)
	}
	out, errb, code = run(t, "host", "os")
	if code != 0 || !(strings.Contains(out, "Darwin") || strings.Contains(out, "Linux")) {
		t.Fatalf("os %q %s", out, errb)
	}
}

func TestShellHookUnalias(t *testing.T) {
	for _, sh := range []string{"zsh", "bash"} {
		s := complete.ShellInit(sh)
		unalias := strings.Index(s, "unalias bhuppi")
		fn := strings.Index(s, "bhuppi()")
		if unalias < 0 || fn < 0 || unalias > fn {
			t.Fatalf("%s hook must unalias bhuppi before defining the function", sh)
		}
		if strings.Contains(s, ".bhuppi") || strings.Contains(s, "source ~/.bhuppi") || strings.Contains(s, "source $HOME/.bhuppi") {
			t.Fatalf("%s hook sources ~/.bhuppi\n%s", sh, s)
		}
		if !strings.Contains(s, "--print-cd") {
			t.Fatalf("%s hook missing --print-cd", sh)
		}
	}
	share, err := os.ReadFile(filepath.Join("..", "..", "share", "bhuppi-shell.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(share)
	if strings.Index(text, "unalias bhuppi") > strings.Index(text, "bhuppi()") {
		t.Fatal("share hook unalias order")
	}
	if strings.Contains(text, ".bhuppi") {
		t.Fatal("share hook sources .bhuppi")
	}
}

func TestPrintCD(t *testing.T) {
	out, errb, code := run(t, "--print-cd", "files", "go", "/tmp")
	if code != 0 || out != "cd /tmp\n" {
		t.Fatalf("print-cd files go: %q %s code %d", out, errb, code)
	}
	out, errb, code = run(t, "--print-cd", "folders", "go", "/tmp")
	if code != 0 || out != "cd /tmp\n" {
		t.Fatalf("print-cd folders go: %q %s", out, errb)
	}
	out, _, code = run(t, "--print-cd", "--dry-run", "files", "go", "/tmp")
	if code != 0 || out != "" {
		t.Fatalf("print-cd dry-run must be silent, got %q", out)
	}
	out, _, code = run(t, "--print-cd", "files", "list", ".")
	if code != 0 || out != "" {
		t.Fatalf("print-cd list must be silent, got %q", out)
	}
	out, _, code = run(t, "--dry-run", "files", "list", ".")
	if code != 0 || strings.TrimSpace(out) != "ls ." {
		t.Fatalf("dry-run list: %q", out)
	}
}

func TestCompletionGroups(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })

	var files, folders int
	for _, it := range complete.Complete(cat, []string{"files", "show", ""}) {
		if it.Kind == "folder" {
			t.Fatalf("files show offered folder %s", it.Value)
		}
		if it.Kind == "file" {
			files++
		}
	}
	if files == 0 {
		t.Fatal("files show offered no files")
	}
	files, folders = 0, 0
	for _, it := range complete.Complete(cat, []string{"folders", "go", ""}) {
		if it.Kind == "file" {
			t.Fatalf("folders go offered file %s", it.Value)
		}
		if it.Kind == "folder" {
			folders++
			if !strings.HasSuffix(it.Value, "/") {
				t.Fatalf("folder missing slash: %s", it.Value)
			}
		}
	}
	if folders == 0 {
		t.Fatal("folders go offered no folders")
	}
	script := complete.Script("zsh")
	for _, want := range []string{"menu select", "group-name ''", "── %d ──", "compadd -V", "-l", `cmd_vals+=("$val")`, "files", "folders"} {
		if !strings.Contains(script, want) {
			t.Fatalf("zsh completion missing %s", want)
		}
	}
	if strings.Contains(script, `$val --`) || strings.Contains(script, `cmd_disp+=("$desc$section")`) || strings.Contains(script, `cmd_disp+=("$section")`) {
		t.Fatal("completion display appends the section id")
	}
	if strings.Contains(script, `cmd_vals+=("!`) || strings.Contains(script, "cmd_vals+=(\"\\x1b") {
		t.Fatal("completion inserts a mark")
	}
	var killDesc string
	for _, it := range complete.Complete(cat, []string{"proc", ""}) {
		if it.Value != "kill" {
			continue
		}
		if strings.Contains(it.Value, "!") || strings.Contains(it.Value, "\x1b") {
			t.Fatalf("inserted word %q", it.Value)
		}
		killDesc = it.Desc
	}
	if !strings.Contains(killDesc, "stops a process") || strings.Contains(killDesc, "\x1b") {
		t.Fatalf("kill description %q", killDesc)
	}
	var listDesc, filesDesc string
	var filesSection, removeSection string
	for _, it := range complete.Complete(cat, []string{"files", ""}) {
		if it.Value == "list" {
			listDesc = it.Desc
		}
		if it.Value == "remove" {
			removeSection = it.Section
			if strings.Contains(it.Value, "!") || strings.Contains(it.Value, "\x1b") {
				t.Fatalf("inserted word %q", it.Value)
			}
		}
	}
	if listDesc == "" || strings.Contains(listDesc, "\x1b") {
		t.Fatalf("list description %q", listDesc)
	}
	if removeSection != "careful" {
		t.Fatalf("remove section %q", removeSection)
	}
	for _, it := range complete.Complete(cat, []string{""}) {
		if it.Value == "files" {
			filesDesc = it.Desc
			filesSection = it.Section
		}
	}
	if filesDesc == "" || filesSection != "everyday" {
		t.Fatalf("files completion %q %q", filesDesc, filesSection)
	}
	if !strings.Contains(filesDesc, "bhuppi files") || !strings.Contains(filesDesc, "e.g.") {
		t.Fatalf("files display is not the four-column row: %q", filesDesc)
	}
	if strings.HasSuffix(strings.TrimSpace(filesDesc), "everyday") || strings.HasSuffix(strings.TrimSpace(filesDesc), "guide") {
		t.Fatalf("section id is a trailing column: %q", filesDesc)
	}
	var reviewAt, cleanAt = -1, -1
	for i, it := range complete.Complete(cat, []string{"disk", ""}) {
		if strings.HasSuffix(strings.TrimSpace(it.Desc), it.Section) {
			t.Fatalf("section id on display %q", it.Desc)
		}
		if it.Value == "review" {
			reviewAt = i
			if it.Section != "commands" || !strings.Contains(it.Desc, "e.g.") {
				t.Fatalf("review display %q %q", it.Section, it.Desc)
			}
		}
		if it.Value == "clean" {
			cleanAt = i
			if it.Section != "careful" || !strings.HasPrefix(strings.TrimSpace(it.Desc), "!") || strings.Contains(it.Value, "!") {
				t.Fatalf("clean completion %q %q", it.Value, it.Desc)
			}
		}
	}
	if reviewAt < 0 || cleanAt < 0 || reviewAt > cleanAt {
		t.Fatalf("disk order review=%d clean=%d", reviewAt, cleanAt)
	}
	colored := complete.CompleteWith(cat, []string{"disk", ""}, help.Options{Color: true})
	var sawClean bool
	for _, it := range colored {
		if it.Value != "clean" {
			continue
		}
		sawClean = true
		if !strings.Contains(it.Desc, "\x1b") || strings.Contains(it.Desc, "^[") || strings.Contains(it.Value, "\x1b") {
			t.Fatalf("colored clean display %q value %q", it.Desc, it.Value)
		}
	}
	if !sawClean {
		t.Fatal("colored disk completion missing clean")
	}
}

func TestDiskReviewAndMenu(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	page, err := help.Render(cat, []string{"files"}, help.Options{Color: true})
	if err != nil {
		t.Fatal(err)
	}
	var warnLine string
	var alias bool
	for _, ln := range strings.Split(page, "\n") {
		if strings.Contains(ln, "! remove") {
			warnLine = ln
		}
		if strings.Contains(ln, "open") && strings.Contains(ln, "show") {
			alias = true
		}
	}
	if !strings.Contains(warnLine, "\x1b") || !strings.Contains(warnLine, "38;2") || strings.Contains(warnLine, "^[") {
		t.Fatalf("warning line: %q", warnLine)
	}
	if !alias {
		t.Fatal("open and show are not one row")
	}
	plain, _, code := run(t, "--plain", "files")
	if code != 0 || !strings.Contains(plain, "! remove") || strings.Contains(plain, "^[") || strings.Contains(plain, "\x1b") {
		t.Fatalf("plain files menu:\n%s", plain)
	}
	helpDisk, _, code := run(t, "--plain", "help", "disk", "review")
	if code != 0 {
		t.Fatal(code)
	}
	flat := strings.Join(strings.Fields(helpDisk), " ")
	for _, want := range []string{"WHAT:", "WHEN:", "RULE:", "bhuppi disk review", "Not a wipe", "rm -rf", "without --yes"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("disk review help missing %q\n%s", want, helpDisk)
		}
	}
	dry, _, code := run(t, "--dry-run", "disk", "review", "--hour")
	if code != 0 || !strings.Contains(dry, "mdfind") || !strings.Contains(dry, "kMDItemFSContentChangeDate") || !strings.Contains(dry, "3600") {
		t.Fatalf("hour dry-run:\n%s", dry)
	}
	if strings.Contains(dry, "-mtime") {
		t.Fatalf("hour dry-run walked another interval:\n%s", dry)
	}
	diskPage, _, code := run(t, "--plain", "disk")
	if code != 0 || !strings.Contains(diskPage, "! clean") || !strings.Contains(diskPage, "review") {
		t.Fatalf("disk menu:\n%s", diskPage)
	}
	clean, _, code := run(t, "--dry-run", "disk", "clean")
	if code != 0 || !strings.Contains(clean, "deletes nothing") {
		t.Fatalf("clean dry-run:\n%s", clean)
	}
	yes, _, code := run(t, "--dry-run", "disk", "clean", "--yes")
	if code != 0 || !strings.Contains(yes, "!") || !strings.Contains(yes, "-delete") || strings.Contains(yes, "Documents") {
		t.Fatalf("clean --yes dry-run:\n%s", yes)
	}
	if len(cat.Groups) != 15 {
		t.Fatal(len(cat.Groups))
	}
}

func TestProcViewAndAgents(t *testing.T) {
	out, errb, code := run(t, "--dry-run", "proc", "view", "1")
	if code != 0 {
		t.Fatal(code, errb)
	}
	if !strings.Contains(out, "ps ") || !strings.Contains(out, "lsof") {
		t.Fatalf("dry-run:\n%s", out)
	}
	if strings.Contains(out, "\x1b") {
		t.Fatal("dry-run ansi")
	}
	helpView, _, code := run(t, "--plain", "help", "proc", "view")
	if code != 0 {
		t.Fatal(code)
	}
	flat := strings.Join(strings.Fields(helpView), " ")
	for _, want := range []string{"WHAT:", "WHEN:", "RULE:", "bhuppi proc view 1", "Not strace", "Not dtruss"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("view help missing %q\n%s", want, helpView)
		}
	}
	helpAgents, _, code := run(t, "--plain", "help", "proc", "agents")
	if code != 0 {
		t.Fatal(code)
	}
	flat = strings.Join(strings.Fields(helpAgents), " ")
	for _, want := range []string{"WHAT:", "Not a secret dump", "Not dtruss", "Not a remote scan"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("agents help missing %q\n%s", want, helpAgents)
		}
	}
	menu, _, code := run(t, "--plain", "proc")
	if code != 0 || !strings.Contains(menu, "agents") || !strings.Contains(menu, "view") {
		t.Fatal("proc menu missing view or agents")
	}
	helpNet, _, code := run(t, "--plain", "help", "net", "connections")
	if code != 0 {
		t.Fatal(code)
	}
	flat = strings.Join(strings.Fields(helpNet), " ")
	for _, want := range []string{
		"WHAT:", "WHEN:", "RULE:", "bhuppi net connections", "Not nmap", "Not a remote scan",
		"Reputation-check is not the default", "Keys are not command arguments",
		"Private addresses are not sent", "bhuppi config reputation set vt",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("connections help missing %q\n%s", want, helpNet)
		}
	}
	if strings.Contains(helpNet, "net reputation") {
		t.Fatal("connections help offers net reputation")
	}
	dry, _, code := run(t, "--dry-run", "net", "connections")
	if code != 0 || !strings.Contains(dry, "lsof -n -P -i") || !strings.Contains(dry, "bhuppi then resolves names") {
		t.Fatalf("connections dry-run: %q", dry)
	}
	if strings.Contains(strings.ToLower(dry), "virustotal") || strings.Contains(strings.ToLower(dry), "abuseip") {
		t.Fatalf("plain dry-run mentions a reputation service: %q", dry)
	}
	netMenu, _, code := run(t, "--plain", "net")
	if code != 0 || !strings.Contains(netMenu, "connections") {
		t.Fatal("net menu missing connections")
	}
}

func TestConfigReputation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reputation.json")
	t.Setenv("BHUPPI_REPUTATION_FILE", path)
	helpCfg, _, code := run(t, "--plain", "help", "config")
	if code != 0 {
		t.Fatal(code)
	}
	flat := strings.Join(strings.Fields(helpCfg), " ")
	for _, want := range []string{
		"WHAT:", "WHEN:", "RULE:",
		"bhuppi config reputation set vt",
		"bhuppi config reputation set abuseip",
		"bhuppi config reputation status",
		"Reputation-check is not the default",
		"Keys are not command arguments",
		"Private addresses are not sent",
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("config help missing %q\n%s", want, helpCfg)
		}
	}
	if strings.Contains(helpCfg, "net reputation") {
		t.Fatal("config help offers net reputation")
	}
	_, errb, code := run(t, "net", "reputation", "status")
	if code == 0 || !strings.Contains(errb, "unknown command") {
		t.Fatalf("net reputation ran: %d %s", code, errb)
	}
	out, errb, code := run(t, "config", "reputation", "status")
	if code != 0 || !strings.Contains(out, "virustotal  missing") || !strings.Contains(out, "abuseip     missing") {
		t.Fatalf("status: %d\n%s\n%s", code, out, errb)
	}
	_, errb, code = run(t, "net", "connections", "reputation-check")
	if code != 2 || !strings.Contains(errb, "bhuppi config reputation set vt") || !strings.Contains(errb, "bhuppi config reputation set abuseip") {
		t.Fatalf("unconfigured check: %d %s", code, errb)
	}
	if strings.Contains(errb, "net reputation") {
		t.Fatal(errb)
	}
	dry, _, code := run(t, "--dry-run", "net", "connections", "reputation-check")
	if code != 0 || !strings.Contains(dry, "lsof -n -P -i") || !strings.Contains(dry, "resolves names") {
		t.Fatalf("reputation dry-run: %q", dry)
	}
	_, errb, code = run(t, "config", "reputation", "set", "vt", "NO-KEY-ON-ARGV")
	if code == 0 || strings.Contains(errb, "NO-KEY-ON-ARGV") {
		t.Fatalf("argv key: %d %s", code, errb)
	}
	const sample = "SAMPLE-KEY-DO-NOT-PRINT"
	if err := os.WriteFile(path, []byte(`{"virustotal":"`+sample+`"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, errb, code = run(t, "config", "reputation", "status")
	if strings.Contains(out+errb, sample) {
		t.Fatal("status contains the key")
	}
	if code != 0 || !strings.Contains(out, "virustotal  set") || !strings.Contains(out, "abuseip     missing") {
		t.Fatalf("status:\n%s\n%s", out, errb)
	}
}

func TestAIAgentReview(t *testing.T) {
	out, _, code := run(t, "--plain", "help", "ai-agent")
	if code != 0 {
		t.Fatal(code)
	}
	flat := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{"WHAT:", "WHEN:", "RULE:", "bhuppi ai-agent review cursor", "Not a secret dump", "Not a remote scan", "Not dtruss"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ai-agent help missing %q", want)
		}
	}
	rev, _, code := run(t, "--plain", "help", "ai-agent", "review")
	if code != 0 {
		t.Fatal(code)
	}
	flat = strings.Join(strings.Fields(rev), " ")
	for _, want := range []string{"WHAT:", "WHEN:", "RULE:", "bhuppi ai-agent review cursor", "Not a secret dump", "Not dtruss"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("review help missing %q", want)
		}
	}
	over, _, code := run(t, "--plain")
	if code != 0 || !strings.Contains(over, "ai-agent") {
		t.Fatal("overview missing ai-agent")
	}
	if len(catalogLoad(t).Groups) != 15 {
		t.Fatal("group count")
	}
	dry, _, code := run(t, "--dry-run", "ai-agent", "review", "cursor")
	if code != 0 {
		t.Fatal(code)
	}
	for _, want := range []string{"ps ", "lsof", "crontab"} {
		if !strings.Contains(dry, want) {
			t.Fatalf("dry-run missing %s\n%s", want, dry)
		}
	}
	if runtime.GOOS == "linux" {
		if !strings.Contains(dry, "systemctl") || strings.Contains(dry, "launchctl") {
			t.Fatal(dry)
		}
	} else if !strings.Contains(dry, "launchctl") {
		t.Fatal(dry)
	}
}

func catalogLoad(t *testing.T) *catalog.Catalog {
	t.Helper()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestEveryHelpPage(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	opt := help.Options{Plain: true}
	s, err := help.Render(cat, nil, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "WHAT:") {
		t.Fatal("overview")
	}
	var walk func(path []string, n *catalog.Node)
	walk = func(path []string, n *catalog.Node) {
		page, err := help.Render(cat, path, opt)
		if err != nil {
			t.Fatalf("%v: %v", path, err)
		}
		for _, want := range []string{"WHAT:", "WHEN:", "RULE:", "Shape:", "e.g.", "NOT THIS", "more:"} {
			if !strings.Contains(page, want) {
				t.Errorf("%v missing %s", path, want)
			}
		}
		for i := range n.Children {
			ch := &n.Children[i]
			walk(append(append([]string{}, path...), ch.Name), ch)
		}
	}
	for i := range cat.Groups {
		g := &cat.Groups[i]
		walk([]string{g.Name}, g)
	}
}

func TestOpenListMapStat(t *testing.T) {
	out, _, code := run(t, "--dry-run", "files", "show", "h")
	if code != 0 || strings.TrimSpace(out) != "cat h" {
		t.Fatalf("show h: %q", out)
	}
	openOut, _, code := run(t, "--dry-run", "files", "open", "README")
	showOut, _, code2 := run(t, "--dry-run", "files", "show", "README")
	if code != 0 || code2 != 0 || strings.TrimSpace(openOut) != "cat README" || openOut != showOut {
		t.Fatalf("open %q show %q", openOut, showOut)
	}
	out, _, code = run(t, "--dry-run", "files", "list", ".", "--long")
	if code != 0 || strings.TrimSpace(out) != "ls -lah ." {
		t.Fatalf("long: %q", out)
	}
	out, _, code = run(t, "--dry-run", "files", "list", ".", "--hidden")
	hidden := "find . -mindepth 1 -maxdepth 1 -name '.*' ! -name . ! -name .."
	if code != 0 || strings.TrimSpace(out) != hidden {
		t.Fatalf("hidden: %q", out)
	}
	if strings.Contains(out, "ls") {
		t.Fatalf("hidden listed with ls: %q", out)
	}
	out, _, code = run(t, "--dry-run", "files", "list", ".", "--long", "--hidden")
	longHidden := hidden + " -exec ls -lahd {} +"
	if code != 0 || strings.TrimSpace(out) != longHidden {
		t.Fatalf("long hidden: %q", out)
	}
	out, _, code = run(t, "--dry-run", "files", "list", "--hidden", "--long", ".")
	if code != 0 || strings.TrimSpace(out) != longHidden {
		t.Fatalf("flag order: %q", out)
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, code = run(t, "--dry-run", "files", "stat", file)
	if code != 0 {
		t.Fatal(out)
	}
	if runtime.GOOS == "darwin" {
		if strings.TrimSpace(out) != "stat -x "+file {
			t.Fatalf("file stat: %q", out)
		}
	} else if strings.TrimSpace(out) != "stat "+file {
		t.Fatalf("file stat: %q", out)
	}
	out, _, code = run(t, "--os", "linux", "--dry-run", "files", "stat", file)
	if runtime.GOOS == "darwin" && strings.TrimSpace(out) != "stat -x "+file {
		t.Fatalf("--os must not change exec argv: %q", out)
	}
	out, _, code = run(t, "--dry-run", "folders", "stat", dir)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if code != 0 || len(lines) != 2 {
		t.Fatalf("dir stat: %q", out)
	}
	if runtime.GOOS == "darwin" && lines[0] != "stat -x "+dir {
		t.Fatalf("dir stat line: %q", lines[0])
	}
	if lines[1] != "du -sh "+dir {
		t.Fatalf("du: %q", lines[1])
	}

	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	withTree := dispatch.Opt{GOOS: "darwin", DryRun: true, Lookup: func(name string) (string, bool) {
		return "/bin/" + name, name == "tree" || name == "find"
	}}
	p := dispatch.Resolve(cat, []string{"folders", "map", ".", "--depth", "2"}, withTree)
	if dispatch.FormatArgv(p.Argv) != "find . -maxdepth 2" {
		t.Fatalf("tree map: %q %s", dispatch.FormatArgv(p.Argv), p.Text)
	}
	if !strings.Contains(p.Note, "bhuppi draws the tree") {
		t.Fatalf("map note: %q", p.Note)
	}
	noTree := withTree
	noTree.Lookup = func(name string) (string, bool) {
		return "/bin/" + name, name == "find"
	}
	p = dispatch.Resolve(cat, []string{"files", "map", "src"}, noTree)
	if dispatch.FormatArgv(p.Argv) != "find src -maxdepth 3" {
		t.Fatalf("find map: %q", dispatch.FormatArgv(p.Argv))
	}
	page, _, code := run(t, "--plain", "help", "folders", "map")
	for _, want := range []string{"WHAT:", "WHEN:", "RULE:", "bhuppi folders map . --depth 2", "NOT THIS", "flat find", "(Level N)"} {
		if !strings.Contains(page, want) {
			t.Errorf("map help missing %q", want)
		}
	}
	page, _, code = run(t, "--plain", "help", "files", "stat")
	for _, want := range []string{"kind", "size", "mode", "owner", "group", "links", "inode", "also linux:", "also macos:", "NOT THIS"} {
		if !strings.Contains(page, want) {
			t.Errorf("stat help missing %q", want)
		}
	}
}

func TestProcMac(t *testing.T) {
	out, errb, code := run(t, "--dry-run", "proc", "list")
	if code != 0 || !strings.Contains(out, "ps ") || !strings.Contains(out, "bhuppi draws the boxes") || strings.Contains(out, "ps aux") {
		t.Fatalf("list: %q %s", out, errb)
	}
	out, _, code = run(t, "--dry-run", "proc", "list", "ssh")
	if code != 0 || !strings.Contains(out, "ps ") || !strings.Contains(out, "bhuppi draws the boxes") || !strings.Contains(out, "ssh") {
		t.Fatalf("keyword: %q", out)
	}
	out, _, code = run(t, "--dry-run", "proc", "list", "-ef")
	if code != 0 || strings.TrimSpace(out) != "ps -ef" {
		t.Fatalf("ps flags: %q", out)
	}
	out, _, code = run(t, "--dry-run", "proc", "stat", "123")
	want := "ps -p 123 -o pid=,user=,uid=,ni=,pri=,stat=,%cpu=,%mem=,etime=,command="
	if runtime.GOOS != "darwin" {
		want = "ps -p 123 -o pid,user,uid,ni,pri,stat,%cpu,%mem,etime,args"
	}
	if code != 0 || strings.TrimSpace(out) != want {
		t.Fatalf("pstat: %q", out)
	}
	out, _, code = run(t, "--dry-run", "proc", "nice")
	if runtime.GOOS == "darwin" {
		if code != 0 || !strings.HasPrefix(strings.TrimSpace(out), "ps -o ni= -p ") {
			t.Fatalf("nice: %q", out)
		}
	} else if code != 0 || strings.TrimSpace(out) != "nice" {
		t.Fatalf("nice: %q", out)
	}
	out, _, code = run(t, "--dry-run", "proc", "nice", "-n", "10", "ls")
	if code != 0 || strings.TrimSpace(out) != "nice -n 10 ls" {
		t.Fatalf("nice cmd: %q", out)
	}
	out, _, code = run(t, "--dry-run", "proc", "using", "npm")
	if code != 0 || strings.TrimSpace(out) != "lsof -c npm" {
		t.Fatalf("using npm: %q", out)
	}
	out, _, code = run(t, "--dry-run", "proc", "using", "python")
	if code != 0 || strings.TrimSpace(out) != "lsof -c python" {
		t.Fatalf("using python: %q", out)
	}
	out, _, code = run(t, "--dry-run", "proc", "using", ":443")
	if code != 0 || strings.TrimSpace(out) != "lsof -i :443" {
		t.Fatalf("using port: %q", out)
	}
	out, _, code = run(t, "--dry-run", "proc", "using", "/etc/hosts")
	if code != 0 || strings.TrimSpace(out) != "lsof -- /etc/hosts" {
		t.Fatalf("using path: %q", out)
	}
	out, _, code = run(t, "--dry-run", "proc", "renice", "10", "123")
	if code != 0 || strings.TrimSpace(out) != "renice 10 123" {
		t.Fatalf("renice: %q", out)
	}

	_, errb, code = run(t, "proc", "strace")
	if runtime.GOOS == "darwin" {
		if code != 2 || !strings.Contains(errb, "strace") || strings.Contains(errb, "needs 1 argument") {
			t.Fatalf("strace darwin: code %d %s", code, errb)
		}
	}
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	p := dispatch.Resolve(cat, []string{"proc", "strace"}, dispatch.Opt{GOOS: "linux", DryRun: true})
	if p.Kind != "usage" || !strings.Contains(p.Text, "needs 1 argument") {
		t.Fatalf("linux strace arity: %+v", p)
	}
	p = dispatch.Resolve(cat, []string{"proc", "strace", "ls"}, dispatch.Opt{GOOS: "darwin", DryRun: true})
	if p.Kind != "missing" || !strings.Contains(p.Text, "strace") || strings.Contains(p.Text, "needs 1 argument") {
		t.Fatalf("darwin strace with args: %+v", p)
	}
	page, _, code := run(t, "--plain", "help", "proc", "using")
	for _, want := range []string{"WHAT:", "WHEN:", "RULE:", "bhuppi proc using python", "NOT THIS", "bare word is not a filename", ":443", "lsof -c"} {
		if !strings.Contains(page, want) {
			t.Errorf("using help missing %q\n%s", want, page)
		}
	}
	page, _, _ = run(t, "--plain", "help", "proc", "strace")
	for _, want := range []string{"Linux", "dtruss", "docs-only", "NOT THIS", "Not dtruss", "Not a stealth tracer"} {
		if !strings.Contains(page, want) {
			t.Errorf("strace help missing %q", want)
		}
	}
	page, _, _ = run(t, "--plain", "help", "proc", "nice")
	if !strings.Contains(page, "bhuppi proc nice") || !strings.Contains(page, "-n") {
		t.Fatalf("nice help:\n%s", page)
	}
	page, _, _ = run(t, "--plain", "help", "proc", "stat")
	for _, want := range []string{"nice", "priority", "also linux:", "also macos:"} {
		if !strings.Contains(page, want) {
			t.Errorf("pstat help missing %q", want)
		}
	}
}

func TestAtlasAndColor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "color")
	t.Setenv("BHUPPI_COLOR_FILE", path)
	t.Setenv("NO_COLOR", "")
	out, errb, code := run(t, "color", "off")
	if code != 0 || strings.TrimSpace(out) != "color off" {
		t.Fatalf("color off: %q %s", out, errb)
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != "off\n" {
		t.Fatalf("config %q %v", body, err)
	}
	out, _, code = run(t, "color")
	if code != 0 || !strings.Contains(out, "color off") {
		t.Fatalf("status %q", out)
	}
	out, _, code = run(t, "--plain", "atlas")
	if code != 0 || !strings.Contains(out, "awk") {
		t.Fatalf("index: %s", out[:min(200, len(out))])
	}
	if strings.Contains(out, "awk [OPTION]") {
		t.Fatal("index dumped the awk syntax")
	}
	out, errb, code = run(t, "--plain", "atlas", "awk")
	if code != 0 {
		t.Fatal(errb)
	}
	for _, want := range []string{"Syntax", "awk [OPTION]", "Description", "/etc/passwd", "ERROR", "printf"} {
		if !strings.Contains(out, want) {
			t.Errorf("atlas awk missing %q", want)
		}
	}
	out, _, code = run(t, "--plain", "atlas", "awk", "fields")
	if code != 0 || !strings.Contains(out, "field") {
		t.Fatalf("fields jump: %s", out)
	}
	out, _, code = run(t, "--plain", "explain", "awk")
	if !strings.Contains(out, "guide: bhuppi atlas awk") {
		t.Fatalf("explain guide:\n%s", out)
	}
	out, _, code = run(t, "--plain", "help", "atlas")
	for _, want := range []string{"WHAT:", "WHEN:", "RULE:", "bhuppi atlas awk", "NOT THIS", "use-case"} {
		if !strings.Contains(out, want) {
			t.Errorf("atlas help missing %q", want)
		}
	}
	page, _, _ := run(t, "--plain", "help")
	if !strings.Contains(page, "bhuppi color off|on") {
		t.Fatal("global options missing color")
	}
	if !strings.Contains(page, "a red ! marks a command") {
		t.Fatal("global options missing the warning mark")
	}
	_, errb, code = run(t, "atlas", "not-a-command")
	if code != 2 || !strings.Contains(errb, "closest") {
		t.Fatalf("missing atlas: %d %s", code, errb)
	}
	if _, errb, code = run(t, "color", "on"); code != 0 {
		t.Fatal(errb)
	}
}

func TestFindShapes(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	opt := dispatch.Opt{GOOS: "darwin", DryRun: true, Lookup: func(name string) (string, bool) {
		return "/bin/" + name, name == "find" || name == "fd" || name == "rg"
	}}
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"files", "find", ".", "--contains", "go"}, "find . -name '*go*'"},
		{[]string{"files", "find", ".", "--startswith", "go"}, "find . -name 'go*'"},
		{[]string{"files", "find", ".", "--endswith", "go"}, "find . -name '*go'"},
		{[]string{"files", "find", ".", "--anyof", "a-z"}, "find . -name '*[a-z]*'"},
		{[]string{"files", "find", ".", "--oneof", "a,b,c"}, "find . '(' -name a -o -name b -o -name c ')'"},
		{[]string{"files", "find", ".", "--regex", `.*\.go`}, `find . -regex '.*\.go'`},
		{[]string{"files", "find", ".", "--contains", "readme", "--insensitivecase"}, "find . -iname '*readme*'"},
		{[]string{"files", "find", ".", "--contains", "readme", "--sensitivecase"}, "find . -name '*readme*'"},
		{[]string{"files", "find", ".", "--regex", ".*readme", "--insensitivecase"}, "find . -iregex '.*readme'"},
		{[]string{"files", "find", "readme", ".", "--word"}, "find . -name readme"},
		{[]string{"files", "find", ".", "--contains", "readme", "--invert"}, "find . ! -name '*readme*'"},
	}
	for _, tc := range cases {
		p := dispatch.Resolve(cat, tc.args, opt)
		if p.Kind != "dry" || dispatch.FormatArgv(p.Argv) != tc.want {
			t.Fatalf("%v: %s (%s) %s", tc.args, dispatch.FormatArgv(p.Argv), p.Kind, p.Text)
		}
	}
	p := dispatch.Resolve(cat, []string{"text", "search", "--regex", "TO.O", "src"}, opt)
	if strings.Join(p.Argv, " ") != "rg TO.O src" {
		t.Fatalf("regex search: %v %s", p.Argv, p.Text)
	}
	p = dispatch.Resolve(cat, []string{"text", "search", "--startswith", "TODO", "src"}, opt)
	if strings.Join(p.Argv, " ") != "rg ^TODO src" {
		t.Fatalf("anchor: %v", p.Argv)
	}
	p = dispatch.Resolve(cat, []string{"text", "search", "readme", ".", "--insensitivecase", "--invert"}, opt)
	if strings.Join(p.Argv, " ") != "rg -i -v -F readme ." {
		t.Fatalf("rg case: %v %s", p.Argv, p.Text)
	}
	noRG := opt
	noRG.Lookup = func(name string) (string, bool) {
		return "/bin/" + name, name == "grep" || name == "find"
	}
	p = dispatch.Resolve(cat, []string{"text", "search", "readme", ".", "--insensitivecase", "--invert"}, noRG)
	if strings.Join(p.Argv, " ") != "grep -n -i -v -F readme ." {
		t.Fatalf("grep case: %v %s", p.Argv, p.Text)
	}
	p = dispatch.Resolve(cat, []string{"text", "search", "readme", ".", "--sensitivecase"}, opt)
	if strings.Join(p.Argv, " ") != "rg -s -F readme ." {
		t.Fatalf("rg sensitive: %v", p.Argv)
	}
	_, errb, code := run(t, "--dry-run", "files", "find", ".", "--contains", "readme", "--sensitivecase", "--insensitivecase")
	if code == 0 || strings.TrimSpace(errb) != "use one of --sensitivecase or --insensitivecase" {
		t.Fatalf("both cases: %d %q", code, errb)
	}
	_, errb, code = run(t, "--dry-run", "files", "find", ".", "--line")
	if code == 0 || !strings.Contains(errb, "--line is for text search") {
		t.Fatalf("line on find: %d %q", code, errb)
	}
	page, _, code := run(t, "--plain", "help", "files", "find")
	if code != 0 {
		t.Fatal(page)
	}
	flat := strings.Join(strings.Fields(page), " ")
	for _, want := range []string{"WHAT:", "WHEN:", "RULE:", "--contains", "--startswith", "--endswith", "--anyof", "--oneof", "--regex", "--sensitivecase", "--insensitivecase", "--word", "--invert", "also linux:", "regextype posix-extended", "not a second regex language", "--regex is still the raw pattern"} {
		if !strings.Contains(flat, want) {
			t.Errorf("files find help missing %q", want)
		}
	}
	page, _, code = run(t, "--plain", "help", "text", "search")
	flat = strings.Join(strings.Fields(page), " ")
	for _, want := range []string{"WHAT:", "WHEN:", "RULE:", "--contains", "--regex", "--sensitivecase", "--insensitivecase", "--word", "--invert", "--line", "not a second regex language", "--regex is still the raw pattern"} {
		if !strings.Contains(flat, want) {
			t.Errorf("text search help missing %q", want)
		}
	}
}

func TestMapDrawing(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(dir, "a.txt"),
		filepath.Join(dir, "sub", ".hidden"),
		filepath.Join(dir, "sub", "deep", "c.txt"),
	} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, errb, code := run(t, "folders", "map", dir, "--depth", "2")
	if code != 0 {
		t.Fatal(errb)
	}
	if !strings.Contains(out, "──") || !strings.Contains(out, "(Level 1)") {
		t.Fatalf("not a drawn tree:\n%s", out)
	}
	if strings.Contains(out, filepath.Join(dir, "a.txt")) {
		t.Fatalf("flat path list:\n%s", out)
	}
	if !strings.Contains(out, ".hidden") || strings.Contains(out, "c.txt") {
		t.Fatalf("depth:\n%s", out)
	}
	out, _, code = run(t, "--dry-run", "proc", "map", "--depth", "2")
	if code != 0 || !strings.Contains(out, "bhuppi draws the tree") {
		t.Fatalf("proc map dry: %q", out)
	}
	if runtime.GOOS == "darwin" && !strings.Contains(out, "ps -axo pid=,ppid=,user=,ni=,stat=,command=") {
		t.Fatalf("proc argv: %q", out)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
