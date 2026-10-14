package reputation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Keys are the local reputation credentials. Empty means that service is missing.
// Status and error text never include these values.
type Keys struct {
	VirusTotal string `json:"virustotal,omitempty"`
	AbuseIP    string `json:"abuseip,omitempty"`
}

// HasVT reports whether a VirusTotal key is stored.
func (k Keys) HasVT() bool { return strings.TrimSpace(k.VirusTotal) != "" }

// HasAbuse reports whether an AbuseIPDB key is stored.
func (k Keys) HasAbuse() bool { return strings.TrimSpace(k.AbuseIP) != "" }

// Path is ~/.config/bhuppi/reputation.json, or BHUPPI_REPUTATION_FILE in tests.
func Path() string {
	if p := os.Getenv("BHUPPI_REPUTATION_FILE"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "bhuppi", "reputation.json")
}

// Load reads the local store. A missing file is an empty store, not an error.
func Load() (Keys, error) {
	path := Path()
	if path == "" {
		return Keys{}, fmt.Errorf("reputation store path is empty")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Keys{}, nil
		}
		return Keys{}, err
	}
	var k Keys
	if err := json.Unmarshal(b, &k); err != nil {
		return Keys{}, fmt.Errorf("reputation store is unreadable")
	}
	return k, nil
}

// Save writes both keys. The directory is mode 0700 and the file is mode 0600.
func Save(k Keys) error {
	path := Path()
	if path == "" {
		return fmt.Errorf("reputation store path is empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	k.VirusTotal = strings.TrimSpace(k.VirusTotal)
	k.AbuseIP = strings.TrimSpace(k.AbuseIP)
	b, err := json.Marshal(k)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// Status prints set or missing. It does not print a key.
func Status(k Keys) string {
	return "virustotal  " + presence(k.HasVT()) + "\nabuseip     " + presence(k.HasAbuse()) + "\n"
}

func presence(set bool) string {
	if set {
		return "set"
	}
	return "missing"
}

// Missing is the exit-2 text when neither key is stored.
func Missing() string {
	return "virustotal is not configured\n  bhuppi config reputation set vt\nabuseip is not configured\n  bhuppi config reputation set abuseip\n"
}

// Preface names a service that will be skipped because its key is missing.
func Preface(k Keys) string {
	var b strings.Builder
	if !k.HasVT() {
		b.WriteString("virustotal is not configured\n  bhuppi config reputation set vt\n")
	}
	if !k.HasAbuse() {
		b.WriteString("abuseip is not configured\n  bhuppi config reputation set abuseip\n")
	}
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
	return b.String()
}
