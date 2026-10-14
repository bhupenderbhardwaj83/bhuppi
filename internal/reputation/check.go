package reputation

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Stats are VirusTotal last_analysis_stats counts. OK is false when the body had no stats.
type Stats struct {
	Malicious  int
	Suspicious int
	Harmless   int
	OK         bool
}

// Row is one public address and the two service results.
// A note replaces a score. Scores are never invented.
type Row struct {
	IP         string
	Name       string
	VT         Stats
	VTNote     string
	AbuseOK    bool
	AbuseScore int
	AbuseNote  string
}

// ParseVT reads malicious, suspicious, and harmless from a VirusTotal v3 body.
func ParseVT(body []byte) (Stats, error) {
	var doc struct {
		Data struct {
			Attributes struct {
				Last struct {
					Malicious  *int `json:"malicious"`
					Suspicious *int `json:"suspicious"`
					Harmless   *int `json:"harmless"`
				} `json:"last_analysis_stats"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return Stats{}, err
	}
	s := doc.Data.Attributes.Last
	if s.Malicious == nil && s.Suspicious == nil && s.Harmless == nil {
		return Stats{}, fmt.Errorf("no score")
	}
	out := Stats{OK: true}
	if s.Malicious != nil {
		out.Malicious = *s.Malicious
	}
	if s.Suspicious != nil {
		out.Suspicious = *s.Suspicious
	}
	if s.Harmless != nil {
		out.Harmless = *s.Harmless
	}
	return out, nil
}

// ParseAbuse reads abuseConfidenceScore from an AbuseIPDB v2 body.
func ParseAbuse(body []byte) (int, error) {
	var doc struct {
		Data struct {
			Score *int `json:"abuseConfidenceScore"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return 0, err
	}
	if doc.Data.Score == nil {
		return 0, fmt.Errorf("no score")
	}
	return *doc.Data.Score, nil
}

type trip func(req *http.Request) (int, []byte, error)

// Check asks VirusTotal and AbuseIPDB once per public IP.
// A missing key skips that service. HTTP 429 stops further calls to that service
// and keeps rows already gathered. It does not sleep.
func Check(ips []string, names map[string]string, keys Keys) []Row {
	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return check(ips, names, keys, func(req *http.Request) (int, []byte, error) {
		resp, err := client.Do(req)
		if err != nil {
			return 0, nil, err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return resp.StatusCode, b, err
	})
}

func check(ips []string, names map[string]string, keys Keys, get trip) []Row {
	vtOff := !keys.HasVT()
	abOff := !keys.HasAbuse()
	vtLimit := false
	abLimit := false
	rows := make([]Row, 0, len(ips))
	for _, ip := range ips {
		row := Row{IP: ip, Name: names[ip]}
		if vtOff {
			row.VTNote = "not configured"
		} else if vtLimit {
			row.VTNote = "rate limited"
		} else if req, err := vtRequest(ip, keys.VirusTotal); err != nil {
			row.VTNote = "unavailable"
		} else {
			code, body, err := get(req)
			row.VT, row.VTNote, vtLimit = applyVT(code, body, err, vtLimit)
		}
		if abOff {
			row.AbuseNote = "not configured"
		} else if abLimit {
			row.AbuseNote = "rate limited"
		} else if req, err := abuseRequest(ip, keys.AbuseIP); err != nil {
			row.AbuseNote = "unavailable"
		} else {
			code, body, err := get(req)
			row.AbuseScore, row.AbuseOK, row.AbuseNote, abLimit = applyAbuse(code, body, err, abLimit)
		}
		rows = append(rows, row)
	}
	return rows
}

func applyVT(code int, body []byte, err error, limited bool) (Stats, string, bool) {
	if err != nil {
		return Stats{}, "unavailable", limited
	}
	if code == http.StatusTooManyRequests {
		return Stats{}, "rate limited", true
	}
	if code != http.StatusOK {
		return Stats{}, "unavailable", limited
	}
	st, perr := ParseVT(body)
	if perr != nil {
		return Stats{}, "no score", limited
	}
	return st, "", limited
}

func applyAbuse(code int, body []byte, err error, limited bool) (int, bool, string, bool) {
	if err != nil {
		return 0, false, "unavailable", limited
	}
	if code == http.StatusTooManyRequests {
		return 0, false, "rate limited", true
	}
	if code != http.StatusOK {
		return 0, false, "unavailable", limited
	}
	n, perr := ParseAbuse(body)
	if perr != nil {
		return 0, false, "no score", limited
	}
	return n, true, "", limited
}

func vtRequest(ip, key string) (*http.Request, error) {
	u := "https://www.virustotal.com/api/v3/ip_addresses/" + url.PathEscape(ip)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apikey", key)
	return req, nil
}

func abuseRequest(ip, key string) (*http.Request, error) {
	u := "https://api.abuseipdb.com/api/v2/check?ipAddress=" + url.QueryEscape(ip)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Key", key)
	req.Header.Set("Accept", "application/json")
	return req, nil
}
