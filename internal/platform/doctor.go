package platform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
	"github.com/bhupenderbhardwaj/bhuppi/internal/execx"
)

// OSName is linux, darwin, or windows.
func OSName() string {
	switch runtime.GOOS {
	case "linux", "darwin", "windows":
		return runtime.GOOS
	default:
		return runtime.GOOS
	}
}

// Row is one doctor table line.
type Row struct {
	Tool  string
	State string
	Note  string
}

// Report builds the doctor table. present reports whether a binary is on PATH.
func Report(cat *catalog.Catalog, goos, arch string, present func(string) bool, flavor func(string) string) string {
	if present == nil {
		present = func(name string) bool {
			_, ok := execx.Look(name, goos, os.Getenv("PATH"))
			return ok
		}
	}
	if flavor == nil {
		flavor = Flavor
	}
	var b strings.Builder
	b.WriteString("bhuppi doctor — what this machine has\n")
	b.WriteString("WHAT: Show the OS and whether each cataloged binary is on PATH.\n")
	b.WriteString("WHEN: Something failed with exit 2 and you want the missing name.\n")
	b.WriteString("RULE: Missing stays missing. GNU and BSD are labeled, not rewritten.\n\n")
	fmt.Fprintf(&b, "os     %s\narch   %s\n\n", goos, arch)
	fmt.Fprintf(&b, "%-16s %-10s %s\n", "tool", "state", "note")
	for _, tool := range []string{"sed", "tar", "awk", "stat"} {
		state := "missing"
		note := flavor(tool)
		if present(tool) {
			state = "present"
		} else if note == "" {
			note = "not on PATH"
		}
		fmt.Fprintf(&b, "%-16s %-10s %s\n", tool, state, note)
	}
	for _, t := range cat.Tools() {
		if t.Name == "sed" || t.Name == "tar" || t.Name == "awk" || t.Name == "stat" {
			continue
		}
		state := "missing"
		note := t.Hint
		if present(t.Name) {
			state = "present"
			note = strings.TrimSpace(t.Platform)
		} else {
			switch {
			case t.Platform == "linux" && goos != "linux":
				note = "Linux-only"
			case t.Platform == "darwin" && goos != "darwin":
				note = "darwin-only"
			case t.Optional:
				note = "optional"
			}
			if t.Fallback != "" {
				if note != "" {
					note += " "
				}
				note += "→ bhuppi help " + t.Fallback
			}
			if note == "" {
				note = "not on PATH"
			}
		}
		fmt.Fprintf(&b, "%-16s %-10s %s\n", t.Name, state, note)
	}
	return b.String()
}

// Flavor probes sed, tar, awk, and stat. Other names return "".
func Flavor(bin string) string {
	switch bin {
	case "sed", "awk", "tar", "stat":
		out := probe(bin, "--version")
		low := strings.ToLower(out)
		switch {
		case strings.Contains(low, "gnu"):
			return "GNU"
		case strings.Contains(low, "bsdtar"), strings.Contains(low, "libarchive"):
			return "BSD"
		case bin == "tar" && strings.Contains(low, "bsd"):
			return "BSD"
		case out == "" || strings.Contains(low, "illegal") || strings.Contains(low, "invalid"):
			return "BSD"
		default:
			if strings.Contains(low, "awk version") {
				return "BSD"
			}
			return "other"
		}
	default:
		return ""
	}
}

func probe(bin string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	out, _ := cmd.CombinedOutput()
	if len(out) > 400 {
		out = out[:400]
	}
	return string(out)
}
