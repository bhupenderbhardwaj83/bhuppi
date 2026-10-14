package procview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSensitiveAndAbnormal(t *testing.T) {
	if !Sensitive("/Users/x/.ssh/id_ed25519") {
		t.Fatal("ssh key path")
	}
	if Sensitive("/usr/bin/cursor") {
		t.Fatal("cursor binary")
	}
	if !Abnormal("1.2.3.4", "4444") {
		t.Fatal("bare ip")
	}
	if Abnormal("api.openai.com", "443") {
		t.Fatal("vendor https")
	}
	secret := "BEGIN OPENSSH PRIVATE KEY"
	page := Format(Page{
		Agents:  true,
		Detail:  Detail{OK: true, PID: "9", User: "x", State: "S", Command: "/usr/bin/cursor"},
		Program: "/usr/bin/cursor",
		Files:   []string{"/Users/x/.ssh/id_ed25519", "/usr/bin/cursor"},
		Outbound: []End{
			{Host: "1.2.3.4", Port: "4444"},
			{Host: "api.openai.com", Port: "443"},
		},
	}, false)
	if !strings.Contains(page, "! /Users/x/.ssh/id_ed25519") || !strings.Contains(page, "sensitive path") {
		t.Fatalf("sensitive mark:\n%s", page)
	}
	if strings.Contains(page, "! /usr/bin/cursor") {
		t.Fatalf("safe path marked:\n%s", page)
	}
	if !strings.Contains(page, "! 1.2.3.4:4444") {
		t.Fatalf("abnormal conn:\n%s", page)
	}
	if strings.Contains(page, "! api.openai.com:443") {
		t.Fatalf("normal conn marked:\n%s", page)
	}
	if strings.Contains(page, secret) {
		t.Fatal("classifier printed file contents")
	}
}

func TestChainAndChildren(t *testing.T) {
	rows := []Row{
		{PID: "1", PPID: "0", Command: "/sbin/launchd"},
		{PID: "10", PPID: "1", Command: "parent"},
		{PID: "20", PPID: "10", Command: "child"},
		{PID: "21", PPID: "10", Command: "other"},
	}
	chain := Ancestors(rows, "20")
	if len(chain) != 3 || chain[0].PID != "1" || chain[2].PID != "20" {
		t.Fatalf("chain %+v", chain)
	}
	text := Format(Page{Detail: Detail{OK: true, PID: "20"}, Chain: chain, Kids: Children(rows, "10")}, false)
	if !strings.Contains(text, "(Level 1)") || !strings.Contains(text, "(Level 2)") {
		t.Fatalf("levels:\n%s", text)
	}
	if strings.Contains(text, "(Level 0)") {
		t.Fatal("root tagged")
	}
	kids := Children(rows, "10")
	if len(kids) != 2 {
		t.Fatal(kids)
	}
	one := Format(Page{Detail: Detail{OK: true, PID: "1", Command: "/sbin/launchd"}, Chain: Ancestors(rows, "1")}, false)
	if strings.Contains(one, "(Level") {
		t.Fatalf("pid 1 chain:\n%s", one)
	}
}

func TestAgentsNoneAndCap(t *testing.T) {
	none := FormatAgents(nil, false)
	if !strings.Contains(none, "no matching agents") || !strings.Contains(none, "cursor") || !strings.Contains(none, "ollama") {
		t.Fatal(none)
	}
	rows := []Row{{PID: "3", User: "me", Command: "/Applications/Cursor.app/Contents/MacOS/Cursor"}}
	got := MatchAgents(rows)
	if len(got) != 1 || got[0].Name != "cursor" {
		t.Fatal(got)
	}
	dir := t.TempDir()
	sub := filepath.Join(dir, "Application Support")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(sub, "tool")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := bin + " --api-key s3cretvalue --flag"
	if ProgramPath(cmd) != bin {
		t.Fatal(ProgramPath(cmd))
	}
	spaced := filepath.Join(dir, "Cursor Helper (Plugin).app", "Contents", "MacOS", "Cursor Helper (Plugin)")
	if err := os.MkdirAll(filepath.Dir(spaced), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spaced, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "Cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	spacedCmd := spaced + " --type=gpu-process --gpu-preferences=AAAA"
	if ProgramPath(spacedCmd) != spaced {
		t.Fatal(ProgramPath(spacedCmd))
	}
	caret := "/System/Library/PrivateFrameworks/TextInputUIMacHelper.framework/Versions/A/XPCServices/CursorUIViewService.xpc/Contents/MacOS/CursorUIViewService"
	if MatchesName(caret, "cursor") || MatchesName("com.apple.TextInputUI cursor", "cursor") {
		t.Fatal("text caret matched cursor")
	}
	if !MatchesName("/Applications/Cursor.app/Contents/MacOS/Cursor", "cursor") || !MatchesName("/Users/x/.local/bin/cursor-agent", "cursor") {
		t.Fatal("real cursor missed")
	}
	if strings.Contains(RedactCommand(cmd), "s3cretvalue") || !strings.Contains(RedactCommand(cmd), "--api-key ***") {
		t.Fatal(RedactCommand(cmd))
	}
	if strings.Contains(RedactCommand("tool --api-key=s3cretvalue"), "s3cretvalue") {
		t.Fatal("inline key")
	}
	choice := FormatChoice("cursor", []Row{{PID: "1", Command: "cursor --token s3cretvalue"}}, "pass one pid")
	if strings.Contains(choice, "s3cretvalue") {
		t.Fatal(choice)
	}
	shown, omitted := CapPaths([]string{"/a", "/a", "/b"}, 1)
	if len(shown) != 1 || omitted != 1 {
		t.Fatalf("%v %d", shown, omitted)
	}
}
