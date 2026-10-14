package aireview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bhupenderbhardwaj/bhuppi/internal/procview"
)

func TestSensitivePath(t *testing.T) {
	if !SensitivePath("/Users/x/Library/Keychains/login.keychain-db") {
		t.Fatal("keychain")
	}
	if SensitivePath("/usr/bin/cursor") {
		t.Fatal("cursor binary")
	}
}

func TestExcessiveAndTree(t *testing.T) {
	var many []Sock
	for i := 0; i < 21; i++ {
		many = append(many, Sock{HasArrow: true, RemoteHost: "h" + itoa(i) + ".example", RemotePort: "443"})
	}
	page := Format(Report{Outbound: many}, false)
	if !strings.Contains(page, "! excessive outbound") || !strings.Contains(page, "21 hosts") {
		t.Fatalf("excessive:\n%s", page)
	}
	two := []Sock{
		{HasArrow: true, RemoteHost: "api.openai.com", RemotePort: "443"},
		{HasArrow: true, RemoteHost: "api2.cursor.sh", RemotePort: "443"},
	}
	quiet := Format(Report{Outbound: two}, false)
	if strings.Contains(quiet, "excessive") {
		t.Fatalf("ordinary marked:\n%s", quiet)
	}
	var dozen []Sock
	for i := 0; i < 12; i++ {
		dozen = append(dozen, Sock{HasArrow: true, RemoteHost: "api2.cursor.sh", RemotePort: "443"})
	}
	tree := Format(Report{Outbound: dozen}, false)
	if !strings.Contains(tree, "api2.cursor.sh") || !strings.Contains(tree, ":443") || !strings.Contains(tree, "(12)") || !strings.Contains(tree, "└──") {
		t.Fatalf("tree:\n%s", tree)
	}
}

func TestCronAndSecret(t *testing.T) {
	text := "# cursor backup\n* * * * * /usr/bin/cursor\n* * * * * echo hello\n"
	got := FilterCron(text, "cursor", "")
	if len(got) != 1 || !strings.Contains(got[0], "cursor") {
		t.Fatal(got)
	}
	if FilterCron("* * * * * echo hello", "cursor", "") != nil {
		t.Fatal("echo kept")
	}
	secret := "SUPERSECRET-SAMPLE-VALUE"
	page := Format(Report{
		Sensitive: []string{"/Users/x/Library/Keychains/login.keychain-db"},
		Blocks:    Blocks([]Snap{{PID: "1", PPID: "0", Command: "/usr/bin/cursor"}}, []Snap{{PID: "1", PPID: "0", Command: "/usr/bin/cursor"}}, nil),
	}, false)
	if strings.Contains(page, secret) {
		t.Fatal("printed a secret that was not a path")
	}
	if !strings.Contains(page, "! /Users/x/Library/Keychains/login.keychain-db  ! sensitive path") {
		t.Fatalf("mark:\n%s", page)
	}
	if strings.Contains(page, "! /usr/bin/cursor") {
		t.Fatal("safe path marked")
	}
}

func TestSummaryAndChain(t *testing.T) {
	page := Format(Report{
		Heading: "codex",
		Blocks: []Block{
			{
				Snap:    Snap{PID: "10", PPID: "1", Command: "/usr/bin/codex"},
				Program: "/usr/bin/codex",
				Chain: []Proc{
					{PID: "1", Program: "/sbin/launchd", Command: "/sbin/launchd"},
					{PID: "10", Program: "/Applications/Codex.app/Contents/MacOS/Autoupdate", Command: "/Applications/Codex.app/Contents/MacOS/Autoupdate"},
				},
			},
			{Snap: Snap{PID: "11", PPID: "1", Command: "/usr/bin/codex"}, Program: "/usr/bin/codex"},
		},
		Outbound: []Sock{
			{HasArrow: true, RemoteHost: "a.example", RemotePort: "443"},
			{HasArrow: true, RemoteHost: "a.example", RemotePort: "443"},
			{HasArrow: true, RemoteHost: "b.example", RemotePort: "443"},
		},
		Sensitive: []string{
			"/Users/x/Library/Cookies/Cookies.binarycookies",
			"/Users/x/.env",
		},
	}, false)
	if !strings.HasPrefix(page, "codex\n") {
		t.Fatalf("start:\n%s", page)
	}
	sumAt := strings.Index(page, "Summary")
	procAt := strings.Index(page, "── processes ──")
	if sumAt < 0 || procAt < 0 || sumAt > procAt {
		t.Fatalf("order:\n%s", page)
	}
	if !strings.Contains(page, "┌ 1/2") || !strings.Contains(page, "┌ 2/2") {
		t.Fatalf("boxes:\n%s", page)
	}
	if !strings.Contains(page, "└\n\n┌ 2/2") {
		t.Fatalf("boxes need a blank line between them:\n%s", page)
	}
	if !strings.Contains(page, "children  none") || !strings.Contains(page, "cwd  —") {
		t.Fatalf("empty lines:\n%s", page)
	}
	head := page[:procAt]
	for _, row := range []struct{ label, n string }{
		{"processes", "2"},
		{"outbound", "3"},
		{"sensitive files", "2"},
		{"credentials", "1"},
	} {
		if summaryValue(head, row.label) != row.n {
			t.Fatalf("%s = %q\n%s", row.label, summaryValue(head, row.label), head)
		}
	}
	if strings.Contains(page, "/sbin/launchd  /sbin/launchd") || strings.Contains(page, "Autoupdate  /Applications") {
		t.Fatalf("chain repeated the command:\n%s", page)
	}
	if strings.Count(page, "/sbin/launchd") != 1 {
		t.Fatalf("launchd count:\n%s", page)
	}
	if CredentialPath("/Users/x/Library/Cookies/Cookies.binarycookies") || CredentialPath("/Users/x/Library/Application Support/Trust Tokens") {
		t.Fatal("cookie or trust token counted as a credential")
	}
	if !CredentialPath("/Users/x/.env") {
		t.Fatal(".env")
	}
	rows := summaryRows(Report{
		Blocks:    []Block{{}, {}},
		Sensitive: []string{"/Users/x/Library/Cookies/Cookies.binarycookies", "/Users/x/.env"},
		Outbound: []Sock{
			{HasArrow: true, RemoteHost: "a.example", RemotePort: "443"},
			{HasArrow: true, RemoteHost: "a.example", RemotePort: "443"},
			{HasArrow: true, RemoteHost: "b.example", RemotePort: "443"},
		},
		Persist: []Persist{{Label: "com.codex"}},
		Launch:  []Job{{Label: "com.codex"}, {Label: "com.codex.helper"}, {Label: "com.codex.more"}},
	})
	for _, row := range rows {
		switch row.label {
		case "sensitive files", "credentials":
			if !row.warn || row.n == 0 {
				t.Fatalf("%s should be red: %+v", row.label, row)
			}
		case "outbound", "persistence", "scheduled tasks", "processes", "inbound":
			if row.warn {
				t.Fatalf("%s should stay cyan: %+v", row.label, row)
			}
		}
	}
	odd := summaryRows(Report{Outbound: []Sock{{HasArrow: true, RemoteHost: "1.2.3.4", RemotePort: "4444"}}})
	for _, row := range odd {
		if row.label == "outbound" && (!row.warn || row.n != 1) {
			t.Fatalf("abnormal outbound: %+v", row)
		}
		if row.label == "sensitive files" && (row.warn || row.n != 0) {
			t.Fatalf("zero sensitive: %+v", row)
		}
	}
}

func summaryValue(section, label string) string {
	for _, line := range strings.Split(section, "\n") {
		trim := strings.TrimSpace(line)
		idx := strings.Index(trim, label+" ")
		if idx < 0 {
			continue
		}
		if idx > 0 && trim[idx-1] != ' ' {
			continue
		}
		rest := strings.TrimSpace(trim[idx+len(label):])
		f := strings.Fields(rest)
		if len(f) == 0 {
			return ""
		}
		return f[0]
	}
	return ""
}

func TestSeventeenAndShortCommand(t *testing.T) {
	var matched []Snap
	for i := 0; i < 17; i++ {
		matched = append(matched, Snap{
			PID: itoa(100 + i), PPID: "1", User: "me", State: "S", Nice: "0", Priority: "20",
			CPU: "0.0", Mem: "0.1", Elapsed: "01:00",
			Command: "/Applications/ChatGPT.app/Contents/MacOS/ChatGPT --type=renderer --gpu-preferences=AAAA --disable-features=Foo",
		})
	}
	page := Format(Report{Blocks: Blocks(matched, matched, nil)}, false)
	if !strings.Contains(page, "── processes ──") || !strings.Contains(page, "── connections ──") || !strings.Contains(page, "processes 17") || !strings.Contains(page, "┌ 1/17") || !strings.Contains(page, "┌ 17/17") {
		t.Fatalf("page:\n%s", page)
	}
	if strings.Contains(page, "pass one pid") || strings.Contains(page, "--gpu-preferences") {
		t.Fatalf("page dumped the cap or the long argv:\n%s", page)
	}
	if !strings.Contains(page, "renderer") {
		t.Fatal(page)
	}
	long := "/Applications/ChatGPT.app/Contents/MacOS/ChatGPT --type=gpu-process --gpu-preferences=AAAA"
	short := ShortCommand("", long)
	if strings.Contains(short, "--gpu-preferences") || !strings.Contains(short, "gpu-process") {
		t.Fatal(short)
	}
}

func TestSpacedProgramBox(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "Codex Framework.framework", "Versions", "A", "Codex Framework")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "Codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := bin + " --type=renderer --gpu-preferences=AAAA"
	snap := Snap{PID: "9", PPID: "1", User: "me", State: "S", Nice: "0", Priority: "20", CPU: "0.0", Mem: "0.1", Elapsed: "01:00", Command: cmd}
	page := Format(Report{Blocks: Blocks([]Snap{snap}, []Snap{snap}, nil)}, false)
	if !strings.Contains(page, "program  "+bin) || !strings.Contains(page, "command  "+bin+" renderer") {
		t.Fatalf("path:\n%s", page)
	}
	if strings.Contains(page, "--gpu-preferences") || strings.Contains(page, "program  "+filepath.Join(dir, "Codex")+"\n") {
		t.Fatalf("truncated:\n%s", page)
	}
}

func TestLsofPidAndParenPath(t *testing.T) {
	header := "COMMAND    PID              USER   FD   TYPE DEVICE  SIZE/OFF                NODE NAME"
	want := "/Applications/Cursor.app/Contents/Frameworks/Cursor Helper (Plugin).app/Contents/MacOS/Cursor Helper (Plugin)"
	prefix := "Cursor    6486 bhupenderbhardwaj  txt      REG  1,1      10    1 "
	nameAt := strings.Index(header, "NAME")
	if len(prefix) < nameAt {
		prefix += strings.Repeat(" ", nameAt-len(prefix))
	}
	text := header + "\n" + prefix + want + "\n"
	got := ParseOpen(text)
	if len(got) != 1 || got[0].PID != "6486" || got[0].FD != "txt" {
		t.Fatalf("%+v", got)
	}
	if procview.PathOf(got[0].Name) != want {
		t.Fatal(procview.PathOf(got[0].Name))
	}
	if procview.PathOf(want+" (deleted)") != want {
		t.Fatal("deleted suffix")
	}
	jobs := FilterLaunch("2387\t0\tcom.apple.TextInputUI.xpc.CursorUIViewService\n-\t0\tcom.cursor.agent\n", "cursor")
	if len(jobs) != 1 || jobs[0].Label != "com.cursor.agent" {
		t.Fatalf("%+v", jobs)
	}
}

func TestCursorCaret(t *testing.T) {
	table := []Snap{
		{PID: "1", Command: "/System/Library/PrivateFrameworks/TextInputUIMacHelper.framework/Versions/A/XPCServices/CursorUIViewService.xpc/Contents/MacOS/CursorUIViewService"},
		{PID: "2", Command: "com.apple.TextInputUI cursor"},
		{PID: "3", Command: "/Applications/Cursor.app/Contents/MacOS/Cursor"},
		{PID: "4", Command: "/Users/x/.local/bin/cursor-agent"},
	}
	got := MatchSnaps(table, "cursor")
	if len(got) != 2 || got[0].PID != "3" || got[1].PID != "4" {
		t.Fatalf("%+v", got)
	}
}

func TestProcListBoxes(t *testing.T) {
	text := "" +
		"10 1 me 0 20 S 0.1 0.2 01:00 /bin/one\n" +
		"11 1 me 0 31 S 0.0 0.4 02:00 /Applications/Cursor.app/Contents/MacOS/Cursor --type=renderer --user-data-dir=/tmp/profile --resume settings.json\n"
	page := FormatList(nil, ParseSnaps(text), nil, false)
	if !strings.Contains(page, "Summary") || !strings.Contains(page, "processes 2") || !strings.Contains(page, "┌ 1/2") || !strings.Contains(page, "┌ 2/2") {
		t.Fatalf("boxes:\n%s", page)
	}
	if strings.Contains(page, "user-data-dir") || strings.Contains(page, "--resume") || strings.Contains(page, "settings.json") {
		t.Fatalf("long argv leaked:\n%s", page)
	}
	if !strings.Contains(page, "renderer") {
		t.Fatalf("role:\n%s", page)
	}
}

func TestNetConnections(t *testing.T) {
	header := "COMMAND    PID              USER   FD   TYPE DEVICE  SIZE/OFF                NODE NAME"
	row := func(cmd, pid, fd, name string) string {
		b := []byte(strings.Repeat(" ", strings.Index(header, "NAME")))
		put := func(at int, s string) {
			for i := 0; i < len(s) && at+i < len(b); i++ {
				b[at+i] = s[i]
			}
		}
		put(0, cmd)
		put(strings.Index(header, "PID"), pid)
		put(strings.Index(header, "USER"), "me")
		put(strings.Index(header, "FD"), fd)
		return string(b) + name
	}
	text := header + "\n" +
		row("sshd", "1", "3u", "TCP 127.0.0.1:22 (LISTEN)") + "\n" +
		row("curl", "2", "4u", "TCP 10.0.0.8:50000->1.2.3.4:4444 (ESTABLISHED)") + "\n" +
		row("curl", "2", "5u", "TCP 10.0.0.8:50001->api.openai.com:443 (ESTABLISHED)") + "\n"
	page := FormatLsof(text, false)
	if !strings.Contains(page, "Summary") || !strings.Contains(page, "── inbound ──") || !strings.Contains(page, "── outbound ──") {
		t.Fatalf("sections:\n%s", page)
	}
	if summaryValue(page, "inbound") != "1" || summaryValue(page, "outbound") != "2" {
		t.Fatalf("counts:\n%s", page)
	}
	if !strings.Contains(page, "LISTEN") || !strings.Contains(page, "! 1.2.3.4") || !strings.Contains(page, "4444") {
		t.Fatalf("marks:\n%s", page)
	}
	for _, line := range strings.Split(page, "\n") {
		if strings.Contains(line, "api.openai.com") && strings.Contains(line, "!") {
			t.Fatalf("ordinary host marked: %s", line)
		}
	}
}

func TestNamedAddress(t *testing.T) {
	header := "COMMAND    PID              USER   FD   TYPE DEVICE  SIZE/OFF                NODE NAME"
	row := func(cmd, pid, fd, name string) string {
		b := []byte(strings.Repeat(" ", strings.Index(header, "NAME")))
		put := func(at int, s string) {
			for i := 0; i < len(s) && at+i < len(b); i++ {
				b[at+i] = s[i]
			}
		}
		put(0, cmd)
		put(strings.Index(header, "PID"), pid)
		put(strings.Index(header, "USER"), "me")
		put(strings.Index(header, "FD"), fd)
		return string(b) + name
	}
	text := header + "\n" +
		row("sshd", "1", "3u", "TCP 127.0.0.1:22 (LISTEN)") + "\n" +
		row("curl", "2", "4u", "TCP 10.0.0.8:50000->140.82.112.25:443 (ESTABLISHED)") + "\n"
	page := FormatLsofNamed(text, map[string]string{"140.82.112.25": "github.com"}, false)
	if !strings.Contains(page, "140.82.112.25  github.com") {
		t.Fatalf("name missing:\n%s", page)
	}
	if !strings.Contains(page, "127.0.0.1") {
		t.Fatalf("loopback dropped:\n%s", page)
	}
	for _, line := range strings.Split(page, "\n") {
		if strings.Contains(line, "140.82.112.25") && !strings.Contains(line, "140.82.112.25  github.com") {
			t.Fatalf("address replaced:\n%s", line)
		}
		if strings.Contains(line, "140.82.112.25") && strings.Contains(line, "!") {
			t.Fatalf("named address marked: %s", line)
		}
		if strings.Contains(line, "127.0.0.1") && strings.Contains(line, "github.com") {
			t.Fatalf("loopback renamed: %s", line)
		}
	}
}

func TestFamilyAndArgv(t *testing.T) {
	table := []procview.Row{
		{PID: "1", PPID: "0", Command: "/sbin/launchd"},
		{PID: "10", PPID: "1", Command: "/Applications/Cursor.app/Contents/MacOS/Cursor"},
		{PID: "11", PPID: "10", Command: "Cursor Helper"},
	}
	fam := Family(table, []procview.Row{table[1]})
	if len(fam) != 2 || fam[0].PID != "10" || fam[1].PID != "11" {
		t.Fatalf("%+v", fam)
	}
	argv := Argv("darwin", "cursor")
	joined := ""
	for _, st := range argv {
		joined += strings.Join(st, " ") + "\n"
	}
	for _, want := range []string{"ps ", "lsof", "crontab", "launchctl"} {
		if !strings.Contains(joined, want) {
			t.Fatal(joined)
		}
	}
	linux := Argv("linux", "1")
	flat := ""
	for _, st := range linux {
		flat += strings.Join(st, " ") + "\n"
	}
	if strings.Contains(flat, "launchctl") || !strings.Contains(flat, "systemctl") || !strings.Contains(flat, "-p 1") {
		t.Fatal(flat)
	}
}

func TestPlist(t *testing.T) {
	raw := `<?xml version="1.0"?>
<plist><dict>
<key>Label</key><string>com.example.cursor</string>
<key>ProgramArguments</key><array><string>/usr/bin/cursor</string><string>/Users/x/.ssh/id_ed25519</string></array>
<key>RunAtLoad</key><true/>
<key>Note</key><string>SUPERSECRET-SAMPLE-VALUE</string>
</dict></plist>`
	p, ok := ParsePlist(raw)
	if !ok || p.Label != "com.example.cursor" || !p.RunAtLoad || len(p.Args) != 2 {
		t.Fatalf("%+v %v", p, ok)
	}
	if strings.Contains(strings.Join(p.Strings(), " "), "SUPERSECRET") {
		t.Fatal("plist leaked an unrelated string")
	}
	if !p.Hit("cursor", "") {
		t.Fatal("hit")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
