package treeview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	os.Exit(m.Run())
}

func TestDirTree(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := []string{
		filepath.Join(dir, "a.txt"),
		filepath.Join(dir, ".secret"),
		filepath.Join(dir, "sub", ".hidden"),
		filepath.Join(dir, "sub", "deep", "c.txt"),
	}
	var listing []string
	listing = append(listing, dir)
	for _, p := range files {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		listing = append(listing, p)
	}
	listing = append(listing, filepath.Join(dir, "sub"), filepath.Join(dir, "sub", "deep"))
	out := Dir(dir, strings.Join(listing, "\n"), 2, false)
	if !strings.Contains(out, "──") || !strings.Contains(out, "(Level 1)") {
		t.Fatalf("drawing:\n%s", out)
	}
	if strings.Contains(out, filepath.Join(dir, "a.txt")) {
		t.Fatalf("printed a full path:\n%s", out)
	}
	if !strings.Contains(out, ".secret") || !strings.Contains(out, ".hidden") {
		t.Fatalf("hidden names:\n%s", out)
	}
	if strings.Contains(out, "c.txt") {
		t.Fatalf("depth 2 included level 3:\n%s", out)
	}
	if !strings.HasSuffix(strings.Split(out, "\n")[0], "/") {
		t.Fatalf("root: %q", strings.Split(out, "\n")[0])
	}
	colored := Dir(dir, strings.Join(listing, "\n"), 2, true)
	if !strings.Contains(colored, "\x1b") || !strings.Contains(colored, "(Level 1)") {
		t.Fatalf("color:\n%s", colored)
	}
}

func TestProcTree(t *testing.T) {
	procs := []Proc{
		{PID: "1", PPID: "0", User: "root", Command: "/sbin/launchd"},
		{PID: "8", PPID: "1", User: "root", Command: "/usr/sbin/syslogd"},
		{PID: "9", PPID: "8", User: "bhupi", Command: "/usr/bin/logger hello"},
		{PID: "10", PPID: "9", User: "bhupi", Command: "/bin/sleep 100"},
	}
	out := ProcTree(procs, 2, false)
	if !strings.Contains(out, "──") || !strings.Contains(out, "(Level 1)") {
		t.Fatalf("proc drawing:\n%s", out)
	}
	if !strings.Contains(out, "syslogd") || !strings.Contains(out, "logger") {
		t.Fatalf("parent and child:\n%s", out)
	}
	if !strings.Contains(out, "(Level 2)") {
		t.Fatalf("child level:\n%s", out)
	}
	shallow := ProcTree(procs, 1, false)
	if strings.Contains(shallow, "logger") || strings.Contains(shallow, "sleep") {
		t.Fatalf("depth 1 too deep:\n%s", shallow)
	}
	long := strings.Repeat("argument ", 20)
	wide := ProcTree([]Proc{
		{PID: "1", PPID: "0", User: "root", Command: "/sbin/launchd"},
		{PID: "2", PPID: "1", User: "me", Command: "/bin/cmd " + long},
	}, 1, false)
	if !strings.Contains(wide, "…") || !strings.Contains(wide, "(Level 1)") {
		t.Fatalf("trim:\n%s", wide)
	}
	table := "1 0 root 0 Ss /sbin/launchd\n8 1 root 0 S /usr/sbin/syslogd\n"
	parsed := ParsePS(table)
	if len(parsed) != 2 || parsed[1].Command != "/usr/sbin/syslogd" {
		t.Fatalf("parse: %#v", parsed)
	}
}
