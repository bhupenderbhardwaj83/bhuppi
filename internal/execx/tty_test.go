package execx

import "testing"

func TestNeedsTTY(t *testing.T) {
	if !NeedsTTY("/usr/bin/ssh", true) || !NeedsTTY("top", true) || !NeedsTTY("vim", true) {
		t.Fatal("interactive tools must keep the tty")
	}
	if NeedsTTY("/bin/ls", true) || NeedsTTY("cat", true) || NeedsTTY("awk", true) {
		t.Fatal("passthrough tools are paged when stdout is a terminal")
	}
	if !NeedsTTY("/bin/ls", false) {
		t.Fatal("a pipe or file must exec so redirection is unchanged")
	}
	if NeedsTTY("/usr/bin/ssh-keygen", true) {
		t.Fatal("ssh-keygen is not ssh")
	}
}
