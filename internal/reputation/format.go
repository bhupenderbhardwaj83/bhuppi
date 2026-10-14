package reputation

import (
	"fmt"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/style"
)

// Format renders one compact block per address.
// A malicious count above 0, or an AbuseIPDB confidence of 25 or higher, starts with ! and is red.
func Format(rows []Row, color bool) string {
	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(formatRow(row, color))
	}
	return b.String()
}

func formatRow(row Row, color bool) string {
	vtText, vtBad := vtRest(row)
	abText, abBad := abuseRest(row)
	label := row.IP
	if row.Name != "" {
		label = row.IP + "  " + row.Name
	}
	bad := vtBad || abBad
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", paintLine(label, bad, color))
	fmt.Fprintf(&b, "%s\n", paintLine(service("virustotal", vtText), vtBad, color))
	fmt.Fprintf(&b, "%s\n", paintLine(service("abuseipdb", abText), abBad, color))
	return b.String()
}

func vtRest(row Row) (string, bool) {
	if row.VTNote != "" {
		return row.VTNote, false
	}
	if !row.VT.OK {
		return "no score", false
	}
	text := fmt.Sprintf("malicious %d   suspicious %d   harmless %d", row.VT.Malicious, row.VT.Suspicious, row.VT.Harmless)
	return text, row.VT.Malicious > 0
}

func abuseRest(row Row) (string, bool) {
	if row.AbuseNote != "" {
		return row.AbuseNote, false
	}
	if !row.AbuseOK {
		return "no score", false
	}
	return fmt.Sprintf("confidence %d", row.AbuseScore), row.AbuseScore >= 25
}

func service(name, rest string) string {
	return fmt.Sprintf("  %-12s %s", name, rest)
}

func paintLine(text string, bad, color bool) string {
	if !bad {
		return text
	}
	text = "! " + strings.TrimSpace(text)
	if color {
		return style.Paint(style.Hidden, text, style.Auto, false)
	}
	return text
}
