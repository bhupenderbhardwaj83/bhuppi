package reputation

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicSkipsLoopback(t *testing.T) {
	got := Public([]string{
		"127.0.0.1", "::1", "10.1.2.3", "192.168.0.9", "172.16.0.1", "172.31.255.1",
		"169.254.1.1", "0.0.0.0", "api.openai.com", "140.82.112.25", "140.82.112.25",
	})
	if len(got) != 1 || got[0] != "140.82.112.25" {
		t.Fatalf("reputation set: %v", got)
	}
	if !Checkable("140.82.112.25") || Checkable("127.0.0.1") {
		t.Fatal("checkable")
	}
}

func TestMaliciousMarked(t *testing.T) {
	st, err := ParseVT([]byte(`{"data":{"attributes":{"last_analysis_stats":{"malicious":2,"suspicious":1,"harmless":70}}}}`))
	if err != nil || !st.OK || st.Malicious != 2 || st.Suspicious != 1 || st.Harmless != 70 {
		t.Fatalf("%+v %v", st, err)
	}
	page := Format([]Row{{
		IP: "140.82.112.25", Name: "github.com", VT: st, AbuseOK: true, AbuseScore: 0,
	}}, false)
	if !strings.Contains(page, "! 140.82.112.25  github.com") || !strings.Contains(page, "malicious 2") || !strings.Contains(page, "! virustotal") {
		t.Fatalf("unmarked:\n%s", page)
	}
	for _, line := range strings.Split(page, "\n") {
		if strings.Contains(line, "abuseipdb") && strings.HasPrefix(strings.TrimSpace(line), "!") {
			t.Fatalf("clean abuse line marked: %s", line)
		}
	}
	hot := Format([]Row{{
		IP:      "203.0.113.8",
		VT:      Stats{OK: true, Malicious: 0, Suspicious: 0, Harmless: 3},
		AbuseOK: true, AbuseScore: 25,
	}}, false)
	if !strings.HasPrefix(hot, "! 203.0.113.8") || !strings.Contains(hot, "! abuseipdb") || !strings.Contains(hot, "confidence 25") {
		t.Fatalf("confidence:\n%s", hot)
	}
}

func TestStatusHidesKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reputation.json")
	t.Setenv("BHUPPI_REPUTATION_FILE", path)
	const sample = "SAMPLE-KEY-DO-NOT-PRINT"
	if err := Save(Keys{VirusTotal: sample}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode %v", info.Mode().Perm())
	}
	dinfo, err := os.Stat(dir)
	if err != nil || dinfo.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode %v", dinfo.Mode().Perm())
	}
	keys, err := Load()
	if err != nil || !keys.HasVT() || keys.HasAbuse() {
		t.Fatal(err)
	}
	text := Status(keys)
	if strings.Contains(text, sample) {
		t.Fatal("status contains the key")
	}
	if !strings.Contains(text, "virustotal  set") || !strings.Contains(text, "abuseip     missing") {
		t.Fatalf("status:\n%s", text)
	}
	raw, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(raw), sample) {
		t.Fatal("store did not keep the key")
	}
}

func TestMissingKeyAndRateLimit(t *testing.T) {
	vtCalls := 0
	rows := check([]string{"140.82.112.25"}, map[string]string{"140.82.112.25": "github.com"}, Keys{AbuseIP: "local-test"}, func(req *http.Request) (int, []byte, error) {
		if strings.Contains(req.URL.Host, "virustotal") {
			t.Fatal("virustotal called without a key")
		}
		if req.Header.Get("Key") == "" || req.Header.Get("Accept") != "application/json" {
			t.Fatal("abuse headers")
		}
		return 200, []byte(`{"data":{"abuseConfidenceScore":0}}`), nil
	})
	page := Format(rows, false)
	if !strings.Contains(page, "140.82.112.25  github.com") || !strings.Contains(page, "not configured") || !strings.Contains(page, "confidence 0") {
		t.Fatalf("page:\n%s", page)
	}
	if strings.Contains(page, "!") {
		t.Fatalf("clean row marked:\n%s", page)
	}

	limited := check([]string{"140.82.112.25", "1.1.1.1"}, nil, Keys{VirusTotal: "local-test", AbuseIP: "local-test"}, func(req *http.Request) (int, []byte, error) {
		if strings.Contains(req.URL.Host, "virustotal") {
			vtCalls++
			if req.Header.Get("x-apikey") == "" {
				t.Fatal("missing vt header")
			}
			if vtCalls == 1 {
				return 200, []byte(`{"data":{"attributes":{"last_analysis_stats":{"malicious":0,"suspicious":0,"harmless":70}}}}`), nil
			}
			return 429, []byte(`{}`), nil
		}
		return 200, []byte(`{"data":{"abuseConfidenceScore":0}}`), nil
	})
	if vtCalls != 2 {
		t.Fatalf("vt calls %d", vtCalls)
	}
	if !limited[0].VT.OK || limited[0].VT.Harmless != 70 {
		t.Fatalf("first row lost: %+v", limited[0])
	}
	if limited[1].VTNote != "rate limited" || !limited[1].AbuseOK || limited[1].AbuseScore != 0 {
		t.Fatalf("second row: %+v", limited[1])
	}
	shown := Format(limited, false)
	if !strings.Contains(shown, "rate limited") || !strings.Contains(shown, "harmless 70") || !strings.Contains(shown, "confidence 0") {
		t.Fatalf("kept rows:\n%s", shown)
	}
}
