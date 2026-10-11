package style

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

func TestColorOffConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "color")
	if err := os.WriteFile(path, []byte("off\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BHUPPI_COLOR_FILE", path)
	t.Setenv("NO_COLOR", "")
	got := Show(Hidden, ".secret", Dark, false)
	if strings.Contains(got, "\x1b") {
		t.Fatalf("config off produced ANSI: %q", got)
	}
	if got != ".secret" {
		t.Fatalf("got %q", got)
	}
}

func TestHiddenIsRedAndSizeIsCyan(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if NameRole(".secret", true) != Hidden {
		t.Fatal("hidden folder")
	}
	if NameRole(".secret", false) != Hidden {
		t.Fatal("hidden file")
	}
	hidden := Paint(Hidden, ".secret", Dark, false)
	if !strings.Contains(hidden, "38;2;231;163;163") || !strings.Contains(hidden, ".secret") {
		t.Fatalf("dark hidden: %q", hidden)
	}
	light := Paint(Hidden, ".secret", Light, false)
	if !strings.Contains(light, "38;2;142;47;47") {
		t.Fatalf("light hidden: %q", light)
	}
	if !LooksNumeric("4096") || !LooksNumeric("1.5K") || !LooksNumeric("12%") {
		t.Fatal("numeric detect")
	}
	size := Paint(Number, "4096", Dark, false)
	if !strings.Contains(size, "38;2;143;207;214") {
		t.Fatalf("size cyan: %q", size)
	}
	line := colorListLine("-rw-r--r--  1 me staff 4096 Oct 11 22:00 .secret", Dark)
	if !strings.Contains(line, "38;2;231;163;163") || !strings.Contains(line, "38;2;143;207;214") {
		t.Fatalf("list line: %q", line)
	}
}
