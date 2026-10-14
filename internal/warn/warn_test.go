package warn

import (
	"strings"
	"testing"
)

func TestMark(t *testing.T) {
	if on, _ := Mark("rm -rf /tmp/x"); !on {
		t.Fatal("rm")
	}
	if on, _ := Mark("find . -type f -name '*.py' -print"); on {
		t.Fatal("find -print")
	}
	if on, _ := Mark("kill 1"); !on {
		t.Fatal("kill")
	}
	if on, _ := Mark("renice 10 1"); on {
		t.Fatal("renice")
	}
	if on, reason := Mark("bhuppi files remove a"); !on || reason != "deletes files" {
		t.Fatalf("files remove: %v %s", on, reason)
	}
	if on, reason := Mark("bhuppi proc kill 1"); !on || reason != "stops a process" {
		t.Fatalf("proc kill: %v %s", on, reason)
	}
	if on, _ := Name("remove"); !on {
		t.Fatal("remove name")
	}
	if on, _ := Name("list"); on {
		t.Fatal("list name")
	}
	if on, _ := Name("rsync"); on {
		t.Fatal("bare rsync")
	}
	if on, _ := Mark("rsync -a --delete src dst"); !on {
		t.Fatal("rsync --delete")
	}
	if on, _ := Mark("chmod 644 a"); !on {
		t.Fatal("chmod")
	}
	if on, _ := Mark("sed s/a/b/ f"); on {
		t.Fatal("sed filter")
	}
	if on, reason := Mark("dd if=/dev/zero of=/dev/disk0"); !on || reason != "overwrites a disk" {
		t.Fatalf("dd: %v %s", on, reason)
	}
	plain := Show("rm -rf /tmp/x", false)
	if !strings.Contains(plain, "!") || strings.Contains(plain, "\x1b") {
		t.Fatalf("plain: %q", plain)
	}
	if !strings.Contains(plain, "rm -rf /tmp/x") || !strings.Contains(plain, "deletes files") {
		t.Fatalf("plain text: %q", plain)
	}
}
