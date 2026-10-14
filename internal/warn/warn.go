// Package warn marks command lines that can delete data, stop a process,
// or overwrite a disk. It does not refuse the command.
package warn

import (
	"path"
	"regexp"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/style"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// Name reports a bhuppi verb or binary that is detrimental on its own.
// rsync, sed, and fdisk are not: those depend on the argv.
func Name(name string) (bool, string) {
	switch strings.ToLower(base(name)) {
	case "remove", "rm", "rmdir", "unlink", "shred", "clean":
		return true, "deletes files"
	case "chmod":
		return true, "changes permissions"
	case "chown", "chgrp":
		return true, "changes ownership"
	case "kill", "killall", "pkill":
		return true, "stops a process"
	case "dd", "wipefs":
		return true, "overwrites a disk"
	case "shutdown", "reboot", "halt", "poweroff":
		return true, "stops a process"
	default:
		if strings.HasPrefix(strings.ToLower(base(name)), "mkfs") {
			return true, "overwrites a disk"
		}
	}
	return false, ""
}

// Mark reports whether line can damage the system, and a short reason.
func Mark(line string) (bool, string) {
	raw := strings.TrimSpace(ansiRE.ReplaceAllString(line, ""))
	if raw == "" {
		return false, ""
	}
	for _, seg := range segments(raw) {
		if on, reason := segment(seg); on {
			return true, reason
		}
	}
	if truncates(raw) {
		return true, "truncates a file"
	}
	return false, ""
}

// Show prefixes a damaging command with ! and, when color is on, paints it red.
// A safe line is returned unchanged. Plain mode has no ANSI.
func Show(line string, color bool) string {
	on, reason := Mark(line)
	if !on {
		return line
	}
	plain := strings.TrimSpace(ansiRE.ReplaceAllString(line, ""))
	shown := "! " + plain
	if reason != "" {
		shown += "  ! " + reason
	}
	if !color {
		return shown
	}
	return style.TrueHidden(shown)
}

func segments(line string) []string {
	line = strings.ReplaceAll(line, "&&", "\n")
	line = strings.ReplaceAll(line, "||", "\n")
	line = strings.ReplaceAll(line, ";", "\n")
	line = strings.ReplaceAll(line, "|", "\n")
	return strings.Split(line, "\n")
}

func segment(seg string) (bool, string) {
	fields := strings.Fields(seg)
	fields = preamble(fields)
	if len(fields) == 0 {
		return false, ""
	}
	if fields[0] == "e.g." {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return false, ""
	}
	if fields[0] == "bhuppi" {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return false, ""
	}
	switch fields[0] {
	case "files", "folders", "proc":
		if len(fields) > 1 {
			if on, reason := Name(fields[1]); on {
				return true, reason
			}
		}
	case "text":
		if len(fields) > 2 && fields[1] == "sed" && fields[2] == "inplace" {
			return true, "edits files in place"
		}
	case "disk":
		// bhuppi disk fdisk is the list-only page (fdisk -l).
		if len(fields) > 2 && fields[1] == "diskutil" && strings.HasPrefix(strings.ToLower(fields[2]), "erase") {
			return true, "overwrites a disk"
		}
		return false, ""
	}
	return tool(base(fields[0]), fields[1:])
}

func tool(name string, args []string) (bool, string) {
	if on, reason := Name(name); on {
		return true, reason
	}
	switch name {
	case "renice", "rsync":
		if name == "rsync" {
			for _, a := range args {
				if a == "--delete" || strings.HasPrefix(a, "--delete-") || a == "--remove-source-files" {
					return true, "deletes files"
				}
			}
		}
		return false, ""
	case "fdisk":
		if fdiskList(args) {
			return false, ""
		}
		return true, "overwrites a disk"
	case "diskutil":
		for _, a := range args {
			if strings.HasPrefix(strings.ToLower(a), "erase") {
				return true, "overwrites a disk"
			}
		}
		return false, ""
	case "sed":
		if sedInPlace(args) {
			return true, "edits files in place"
		}
		return false, ""
	case "find":
		if findDeletes(args) {
			return true, "deletes files"
		}
		return false, ""
	case "xargs":
		if cmd := xargsCmd(args); cmd != "" {
			return tool(cmd, nil)
		}
	default:
		if strings.HasPrefix(name, "mkfs") {
			return true, "overwrites a disk"
		}
	}
	return false, ""
}

func preamble(fields []string) []string {
	for len(fields) > 0 {
		switch fields[0] {
		case "sudo", "command", "nocorrect", "time":
			fields = fields[1:]
			continue
		}
		if strings.Contains(fields[0], "=") && !strings.HasPrefix(fields[0], "-") && !strings.Contains(fields[0], "/") {
			fields = fields[1:]
			continue
		}
		break
	}
	return fields
}

func base(name string) string {
	name = strings.Trim(name, `"'`)
	if strings.Contains(name, "/") {
		name = path.Base(name)
	}
	return name
}

func fdiskList(args []string) bool {
	if len(args) == 0 {
		return false
	}
	for _, a := range args {
		if a != "-l" && a != "--list" {
			return false
		}
	}
	return true
}

func sedInPlace(args []string) bool {
	for _, a := range args {
		if a == "-i" || a == "--in-place" || strings.HasPrefix(a, "-i") {
			return true
		}
	}
	return false
}

func findDeletes(args []string) bool {
	for i, a := range args {
		if a == "-delete" {
			return true
		}
		if (a == "-exec" || a == "-ok") && i+1 < len(args) {
			switch base(args[i+1]) {
			case "rm", "rmdir", "shred", "unlink":
				return true
			}
		}
	}
	return false
}

func xargsCmd(args []string) string {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		return base(a)
	}
	return ""
}

func truncates(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '>' {
			continue
		}
		if i+1 < len(s) && (s[i+1] == '>' || s[i+1] == '&') {
			if s[i+1] == '>' {
				i++
			}
			continue
		}
		if i > 0 && s[i-1] == '>' {
			continue
		}
		return true
	}
	return false
}
