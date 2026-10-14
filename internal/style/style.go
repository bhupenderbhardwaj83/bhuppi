// Package style is the sober palette. Every role has a light-background
// ink and a dark-background ink. Body text stays the terminal default.
package style

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Role is one colored job in help, listings, and the atlas.
type Role int

const (
	Hidden Role = iota // red: hidden names, warnings, missing
	Path               // amber: paths and folders
	Folder
	File // green: regular files and "present"
	Number
	Accent  // command names, group titles, section headers
	Label   // WHAT / WHEN / RULE
	Example // e.g. lines and use-case commands
	Muted   // tree branches, visible on light and dark
)

// Ground picks which ink to use. Auto asks the terminal.
type Ground int

const (
	Auto Ground = iota
	Light
	Dark
)

var lightInk = map[Role]string{
	Hidden:  "#8E2F2F",
	Path:    "#8A5A12",
	Folder:  "#8A5A12",
	File:    "#1F6B3A",
	Number:  "#0E5F6B",
	Accent:  "#3C3A78",
	Label:   "#6A3E1F",
	Example: "#1E5C52",
	Muted:   "#4B5563",
}

var darkInk = map[Role]string{
	Hidden:  "#E7A3A3",
	Path:    "#E2C48A",
	Folder:  "#E2C48A",
	File:    "#9DCF9A",
	Number:  "#8FCFD6",
	Accent:  "#C9C0EA",
	Label:   "#E2C2A4",
	Example: "#A9D4C8",
	Muted:   "#B7BDC6",
}

// TrueHidden paints a warning with the dark red as a real ESC sequence.
// The bytes are ESC, then "[38;2;231;163;163m". They are not the two characters caret and bracket.
func TrueHidden(text string) string {
	if text == "" {
		return ""
	}
	return "\x1b[38;2;231;163;163m" + text + "\x1b[0m"
}

// Paint colors text. It ignores the config file so a test can force a ground.
func Paint(role Role, text string, ground Ground, bold bool) string {
	if text == "" {
		return ""
	}
	st := lipgloss.NewStyle()
	if bold {
		st = st.Bold(true)
	}
	switch ground {
	case Light:
		st = st.Foreground(lipgloss.Color(lightInk[role]))
	case Dark:
		st = st.Foreground(lipgloss.Color(darkInk[role]))
	default:
		st = st.Foreground(lipgloss.AdaptiveColor{Light: lightInk[role], Dark: darkInk[role]})
	}
	return st.Render(text)
}

// Show colors text only when color is allowed (config on, NO_COLOR unset, not plain).
func Show(role Role, text string, ground Ground, bold bool) string {
	if !Allow(false) {
		return text
	}
	return Paint(role, text, ground, bold)
}

// Allow reports the persisted preference. It does not look at the TTY.
func Allow(plain bool) bool {
	if plain || os.Getenv("NO_COLOR") != "" {
		return false
	}
	return pref() != "off"
}

// Status is the word color on or color off, including a NO_COLOR note.
func Status() string {
	if os.Getenv("NO_COLOR") != "" {
		return "color off (NO_COLOR)\n"
	}
	if pref() == "off" {
		return "color off\n"
	}
	return "color on\n"
}

// Set writes ~/.config/bhuppi/color (or BHUPPI_COLOR_FILE).
func Set(on bool) error {
	path := file()
	if path == "" {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	word := "on\n"
	if !on {
		word = "off\n"
	}
	return os.WriteFile(path, []byte(word), 0o644)
}

func file() string {
	if p := os.Getenv("BHUPPI_COLOR_FILE"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "bhuppi", "color")
}

func pref() string {
	path := file()
	if path == "" {
		return "on"
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "on"
	}
	s := strings.ToLower(strings.TrimSpace(string(b)))
	if s == "off" || s == "0" || s == "false" {
		return "off"
	}
	return "on"
}

// NameRole classifies a file name. Hidden wins over file or folder.
func NameRole(base string, isDir bool) Role {
	if base != "." && base != ".." && strings.HasPrefix(base, ".") {
		return Hidden
	}
	if isDir || base == "." || base == ".." {
		return Folder
	}
	return File
}

// LooksNumeric reports sizes, counts, pids, ports, percents, and IPs.
func LooksNumeric(tok string) bool {
	if tok == "" {
		return false
	}
	t := strings.Trim(tok, "()[]{},")
	if t == "" {
		return false
	}
	if strings.HasSuffix(t, "%") {
		t = strings.TrimSuffix(t, "%")
	}
	if isIP(t) {
		return true
	}
	// size suffix: 12K 1.5M 96B
	if len(t) > 1 {
		last := t[len(t)-1]
		if strings.ContainsRune("KMGTPEBkmgtpeb", rune(last)) && isNumBody(t[:len(t)-1]) {
			return true
		}
	}
	return isNumBody(t)
}

func isNumBody(s string) bool {
	if s == "" {
		return false
	}
	dots := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
		case r == '.':
			dots++
		case r == ':':
			// elapsed times 10-13:07:09 and ports already handled
		default:
			return false
		}
	}
	// colon form: digits and colons only, at least one digit
	if strings.Contains(s, ":") {
		digit := false
		for _, r := range s {
			if r >= '0' && r <= '9' {
				digit = true
				continue
			}
			if r != ':' && r != '-' {
				return false
			}
		}
		return digit
	}
	return dots <= 1
}

func isIP(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" || len(p) > 3 {
			return false
		}
		n := 0
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
			n = n*10 + int(r-'0')
		}
		if n > 255 {
			return false
		}
	}
	return true
}
