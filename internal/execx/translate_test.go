package execx

import (
	"strings"
	"testing"
)

func TestWindowsDoesNotUseFindExe(t *testing.T) {
	_, err := Translate("windows", []string{"find", ".", "-name", "a"}, func(string) bool { return false })
	if err == nil || !strings.Contains(err.Error(), "find.exe") {
		t.Fatalf("err %v", err)
	}
	_, err = Translate("windows", []string{"sort", "f"}, func(string) bool { return false })
	if err == nil || !strings.Contains(err.Error(), "sort.exe") {
		t.Fatalf("err %v", err)
	}
	got, err := Translate("windows", []string{"find", ".", "-name", "a"}, func(name string) bool { return name == "find" })
	if err != nil || got[0] != "find" {
		t.Fatalf("git find: %v %v", got, err)
	}
	got, err = Translate("windows", []string{"ls", "."}, func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	if join(got) != "cmd /c dir ." {
		t.Fatalf("ls: %v", got)
	}
	got, err = Translate("windows", []string{"cat", "README"}, func(string) bool { return false })
	if err != nil || join(got) != "cmd /c type README" {
		t.Fatalf("cat: %v %v", got, err)
	}
	got, err = Translate("windows", []string{"grep", "-n", "TODO", "src"}, func(string) bool { return false })
	if err != nil || join(got) != "findstr /n TODO src" {
		t.Fatalf("grep: %v %v", got, err)
	}
	got, err = Translate("windows", []string{"ps", "aux"}, func(string) bool { return false })
	if err != nil || join(got) != "tasklist" {
		t.Fatalf("ps: %v %v", got, err)
	}
}

func join(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}
