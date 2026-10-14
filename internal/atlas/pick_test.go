package atlas

import (
	"strings"
	"testing"
)

func TestLabScopeOnce(t *testing.T) {
	nmap, ok := Render("nmap", "", false)
	if !ok {
		t.Fatal("nmap")
	}
	if n := strings.Count(nmap, "approved lab scope"); n != 1 {
		t.Fatalf("nmap copies: %d", n)
	}
	sentence := "Authorization is required for every target."
	if strings.Count(nmap, sentence) != 1 {
		t.Fatalf("sentence copies: %d", strings.Count(nmap, sentence))
	}
	if strings.Index(nmap, sentence) > strings.Index(nmap, "Syntax") {
		t.Fatal("sentence is not above the syntax")
	}
	awk, ok := Render("awk", "", false)
	if !ok || strings.Contains(awk, "approved lab scope") {
		t.Fatal("awk has the nmap lab-scope sentence")
	}
}

func TestDeliverUseCase(t *testing.T) {
	body := "" +
		"Use Case 01: Find JSON files\n" +
		"  find . -type f -name '*.json' -print\n" +
		"  Find JSON files.\n\n" +
		"Use Case 02: Discover source files\n" +
		"  find src -type f -name '*.py' -print\n" +
		"  · -name tests basenames\n\n" +
		"Use Case 03: Locate large artifacts\n" +
		"  find . -type f -size +50M -print\n"
	var got string
	cmd, err := Deliver(body, 2, func(s string) error {
		got = s
		return nil
	})
	want := "find src -type f -name '*.py' -print"
	if err != nil || cmd != want || got != want {
		t.Fatalf("copy: cmd %q got %q err %v", cmd, got, err)
	}
	called := false
	if _, err := Deliver(body, 9, func(string) error {
		called = true
		return nil
	}); err == nil || called {
		t.Fatalf("missing number wrote=%v err=%v", called, err)
	}
	if strings.Contains(got, "Discover") {
		t.Fatal("copied the title")
	}
	cmd, ok := Pick(CaseCommands("find"), 2)
	if !ok || cmd != want {
		t.Fatalf("find case 2: %q %v", cmd, ok)
	}
}
