// Package netname resolves public addresses already present on this machine.
// It uses the system resolver. It does not query VirusTotal or AbuseIPDB.
package netname

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/bhupenderbhardwaj/bhuppi/internal/reputation"
)

// Resolve looks up each public address. One failure or timeout leaves that
// address unnamed and does not fail the others. Each lookup is bounded to two seconds.
func Resolve(ips []string) map[string]string {
	out := map[string]string{}
	var mu sync.Mutex
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for _, ip := range ips {
		if !reputation.Checkable(ip) {
			continue
		}
		ip := ip
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			names, err := net.DefaultResolver.LookupAddr(ctx, ip)
			if err != nil || len(names) == 0 {
				return
			}
			name := strings.TrimSuffix(names[0], ".")
			if name == "" {
				return
			}
			mu.Lock()
			out[ip] = name
			if parsed := net.ParseIP(ip); parsed != nil {
				out[parsed.String()] = name
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}
