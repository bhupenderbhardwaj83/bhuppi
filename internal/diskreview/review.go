// Package diskreview explains what filled the disk. It reads sizes.
// It deletes only when clean is asked to, and only inside the allowlist.
package diskreview

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/bhupenderbhardwaj/bhuppi/internal/style"
)

// File is one path and its size in bytes.
type File struct {
	Path string
	Size int64
}

// Bucket is one recognizable category.
type Bucket struct {
	Total int64
	Paths []File
}

// ReviewArgv is the read-only argv for disk review. It does not run anything.
func ReviewArgv(home, goos string, args []string) ([][]string, error) {
	mode, sinceN, sinceUnit, err := parseReview(args)
	if err != nil {
		return nil, err
	}
	var steps [][]string
	if goos == "linux" {
		steps = [][]string{{"df", "-h", "/"}}
	} else {
		steps = [][]string{
			{"diskutil", "apfs", "list"},
			{"diskutil", "info", "/"},
		}
	}
	if mode == "all" {
		steps = append(steps, []string{"du", "-hd", "1", home})
	}
	switch mode {
	case "all":
		steps = append(steps, intervalArgv(home, goos, "-mmin", "-60")...)
		steps = append(steps, intervalArgv(home, goos, "-mtime", "-1")...)
		steps = append(steps, intervalArgv(home, goos, "-mtime", "-7")...)
		steps = append(steps, intervalArgv(home, goos, "-mtime", "-30")...)
	case "hour":
		steps = append(steps, intervalArgv(home, goos, "-mmin", "-60")...)
	case "day":
		steps = append(steps, intervalArgv(home, goos, "-mtime", "-1")...)
	case "week":
		steps = append(steps, intervalArgv(home, goos, "-mtime", "-7")...)
	case "month":
		steps = append(steps, intervalArgv(home, goos, "-mtime", "-30")...)
	case "since":
		flag, n := sinceFind(sinceN, sinceUnit)
		steps = append(steps, intervalArgv(home, goos, flag, n)...)
	}
	return steps, nil
}

// intervalArgv is the query for one window.
// On darwin, Spotlight answers the home in one pass. It returns nothing under
// ~/Library/Caches and often skips ~/Library/Logs, so those two roots get a
// bounded find instead of another walk of the whole home. On Linux the query
// is find of the home directory.
func intervalArgv(home, goos, flag, n string) [][]string {
	if goos == "linux" {
		return [][]string{findArgv(home, flag, n)}
	}
	out := [][]string{mdfindArgv(home, flag, n)}
	for _, rel := range []string{
		filepath.Join("Library", "Caches"),
		filepath.Join("Library", "Logs"),
	} {
		root := filepath.Join(home, rel)
		out = append(out, []string{"find", root, "-xdev", flag, n, "-type", "f", "-print"})
	}
	return out
}

// mdfindArgv is one Spotlight pass for files whose content changed in the window.
// Spotlight does not index .git object stores and often skips node_modules, so
// those files are missing unless the query returns the directory itself. That
// directory is then one du -sk, not a walk of every child. target/debug is not
// excluded: Spotlight returns those files, and they stay in the Desktop or
// Developer box. A directory mtime is not used to prune. On APFS the parent
// directory's mtime does not change when a file inside is edited.
func mdfindArgv(home, flag, n string) []string {
	num := strings.TrimPrefix(n, "-")
	v, _ := strconv.Atoi(num)
	if v < 1 {
		v = 1
	}
	var expr string
	if flag == "-mmin" {
		expr = fmt.Sprintf("kMDItemFSContentChangeDate >= $time.now(-%d)", v*60)
	} else {
		expr = fmt.Sprintf("kMDItemFSContentChangeDate >= $time.today(-%d)", v)
	}
	return []string{"mdfind", "-onlyin", home, expr}
}

func findArgv(home, flag, n string) []string {
	tool := filepath.Join(home, ".toolchain")
	return []string{
		"find", home, "-xdev",
		"(",
		"-path", tool, "-o", "-path", filepath.Join(tool, "*"),
		"-o", "-path", "*/bhuppi/.toolchain", "-o", "-path", "*/bhuppi/.toolchain/*",
		")",
		"-prune", "-o",
		flag, n, "-type", "f", "-print",
	}
}

func parseReview(args []string) (mode string, n int, unit string, err error) {
	if len(args) == 0 {
		return "all", 0, "", nil
	}
	if len(args) == 1 {
		switch args[0] {
		case "--hour":
			return "hour", 0, "", nil
		case "--day":
			return "day", 0, "", nil
		case "--week":
			return "week", 0, "", nil
		case "--month":
			return "month", 0, "", nil
		}
	}
	if len(args) == 2 && args[0] == "--since" {
		n, unit, err = parseSince(args[1])
		if err != nil {
			return "", 0, "", err
		}
		return "since", n, unit, nil
	}
	return "", 0, "", fmt.Errorf("disk review takes one of --hour, --day, --week, --month, or --since Nh|Nd|Nw")
}

func parseSince(s string) (int, string, error) {
	if len(s) < 2 {
		return 0, "", fmt.Errorf("disk review --since takes Nh, Nd, or Nw")
	}
	unit := s[len(s)-1:]
	if unit != "h" && unit != "d" && unit != "w" {
		return 0, "", fmt.Errorf("disk review --since takes Nh, Nd, or Nw")
	}
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n < 1 {
		return 0, "", fmt.Errorf("disk review --since takes Nh, Nd, or Nw")
	}
	return n, unit, nil
}

func sinceFind(n int, unit string) (flag, arg string) {
	switch unit {
	case "h":
		return "-mmin", "-" + strconv.Itoa(n*60)
	case "w":
		return "-mtime", "-" + strconv.Itoa(n*7)
	default:
		return "-mtime", "-" + strconv.Itoa(n)
	}
}

// IntervalTitle names a find or mdfind step for the report.
func IntervalTitle(argv []string) string {
	if len(argv) > 0 && filepath.Base(argv[0]) == "mdfind" {
		return titleFromQuery(strings.Join(argv, " "))
	}
	flag, n := findPredicate(argv)
	switch {
	case flag == "-mmin" && n == 60:
		return "last hour"
	case flag == "-mtime" && n == 1:
		return "last day"
	case flag == "-mtime" && n == 7:
		return "last week"
	case flag == "-mtime" && n == 30:
		return "last month"
	case flag == "-mmin" && n > 0 && n%60 == 0:
		return fmt.Sprintf("last %d hours", n/60)
	case flag == "-mtime" && n > 0 && n%7 == 0:
		return fmt.Sprintf("last %d weeks", n/7)
	case flag == "-mtime" && n > 0:
		return fmt.Sprintf("last %d days", n)
	default:
		return "changed"
	}
}

func titleFromQuery(q string) string {
	if n, ok := queryNum(q, "$time.now(-"); ok {
		switch {
		case n == 3600:
			return "last hour"
		case n > 0 && n%3600 == 0:
			return fmt.Sprintf("last %d hours", n/3600)
		}
	}
	if n, ok := queryNum(q, "$time.today(-"); ok {
		switch {
		case n == 1:
			return "last day"
		case n == 7:
			return "last week"
		case n == 30:
			return "last month"
		case n > 0 && n%7 == 0:
			return fmt.Sprintf("last %d weeks", n/7)
		case n > 0:
			return fmt.Sprintf("last %d days", n)
		}
	}
	return "changed"
}

func queryNum(q, prefix string) (int, bool) {
	i := strings.Index(q, prefix)
	if i < 0 {
		return 0, false
	}
	rest := q[i+len(prefix):]
	j := strings.Index(rest, ")")
	if j < 0 {
		return 0, false
	}
	n, err := strconv.Atoi(rest[:j])
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func findPredicate(argv []string) (flag string, n int) {
	for i := 0; i < len(argv)-1; i++ {
		if argv[i] == "-mmin" || argv[i] == "-mtime" {
			raw := strings.TrimPrefix(argv[i+1], "-")
			v, _ := strconv.Atoi(raw)
			return argv[i], v
		}
	}
	return "", 0
}

// ParseDF reads size, used, and free for the / mount. Free is the free column, not free plus purgeable.
func ParseDF(text string) (size, used, free string, ok bool) {
	return ParseDFMount(text, "/")
}

// ParseDFMount reads size, used, and free for one mount. Free is the free column.
func ParseDFMount(text, mount string) (size, used, free string, ok bool) {
	lines := strings.Split(text, "\n")
	sizeAt, usedAt, freeAt := 1, 2, 3
	for _, ln := range lines {
		fields := strings.Fields(ln)
		if len(fields) == 0 {
			continue
		}
		if strings.EqualFold(fields[0], "Filesystem") {
			for i, f := range fields {
				switch strings.ToLower(f) {
				case "size", "1024-blocks":
					sizeAt = i
				case "used":
					usedAt = i
				case "avail", "available":
					freeAt = i
				}
			}
			continue
		}
		if fields[len(fields)-1] != mount {
			continue
		}
		if len(fields) <= freeAt || len(fields) <= sizeAt || len(fields) <= usedAt {
			continue
		}
		return fields[sizeAt], fields[usedAt], fields[freeAt], true
	}
	return "", "", "", false
}

// APFSVol is one volume inside the container that holds /.
type APFSVol struct {
	Name     string
	Consumed string
	Bytes    int64
}

// APFS is the container that holds /, using the figures diskutil printed.
type APFS struct {
	Ref       string
	Size      string
	Used      string
	Free      string
	SizeB     int64
	UsedB     int64
	FreeB     int64
	HaveBytes bool
	Volumes   []APFSVol
}

// ParseAPFS reads `diskutil apfs list` and keeps the container whose mount is /.
func ParseAPFS(text string) (APFS, bool) {
	var all []apfsBox
	cur := -1
	vol := -1
	for _, ln := range strings.Split(text, "\n") {
		if ref := containerHeader(ln); ref != "" {
			all = append(all, apfsBox{ref: ref})
			cur = len(all) - 1
			vol = -1
			continue
		}
		if cur < 0 {
			continue
		}
		if ref := afterLabel(ln, "APFS Container Reference:"); ref != "" {
			all[cur].ref = ref
			vol = -1
			continue
		}
		if strings.Contains(ln, "Size (Capacity Ceiling):") {
			all[cur].size, all[cur].sizeB, all[cur].sizeOK = measure(ln)
			vol = -1
			continue
		}
		if strings.Contains(ln, "Capacity In Use By Volumes:") {
			all[cur].used, all[cur].usedB, all[cur].usedOK = measure(ln)
			vol = -1
			continue
		}
		if strings.Contains(ln, "Capacity Not Allocated:") {
			all[cur].free, all[cur].freeB, all[cur].freeOK = measure(ln)
			vol = -1
			continue
		}
		if strings.Contains(ln, "APFS Volume Disk") {
			all[cur].vols = append(all[cur].vols, APFSVol{})
			vol = len(all[cur].vols) - 1
			continue
		}
		if vol < 0 {
			if mountIsRoot(ln) {
				all[cur].root = true
			}
			continue
		}
		if all[cur].vols[vol].Name == "" && strings.Contains(ln, "Name:") {
			all[cur].vols[vol].Name = volumeName(ln)
		}
		if strings.Contains(ln, "Capacity Consumed:") {
			all[cur].vols[vol].Consumed, all[cur].vols[vol].Bytes, _ = measure(ln)
		}
		if mountIsRoot(ln) {
			all[cur].root = true
		}
	}
	for i := range all {
		c := &all[i]
		if !c.root || c.ref == "" || c.size == "" || c.used == "" || c.free == "" {
			continue
		}
		out := APFS{
			Ref: c.ref, Size: c.size, Used: c.used, Free: c.free,
			SizeB: c.sizeB, UsedB: c.usedB, FreeB: c.freeB,
			HaveBytes: c.sizeOK && c.usedOK && c.freeOK,
		}
		for _, v := range c.vols {
			if v.Name == "" || v.Consumed == "" {
				continue
			}
			out.Volumes = append(out.Volumes, v)
		}
		return out, true
	}
	return APFS{}, false
}

type apfsBox struct {
	ref                    string
	size, used, free       string
	sizeB, usedB, freeB    int64
	sizeOK, usedOK, freeOK bool
	root                   bool
	vols                   []APFSVol
}

func containerHeader(ln string) string {
	if strings.Contains(ln, "Container Reference") || strings.Contains(ln, "Physical Store") {
		return ""
	}
	i := strings.Index(ln, "Container disk")
	if i < 0 {
		return ""
	}
	rest := strings.TrimSpace(ln[i+len("Container "):])
	fields := strings.Fields(rest)
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "disk") {
		return ""
	}
	return fields[0]
}

func afterLabel(ln, label string) string {
	i := strings.Index(ln, label)
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(ln[i+len(label):])
}

func mountIsRoot(ln string) bool {
	if !strings.Contains(ln, "Mount Point:") {
		return false
	}
	return strings.TrimSpace(afterColon(ln)) == "/"
}

func afterColon(ln string) string {
	_, val, ok := strings.Cut(ln, ":")
	if !ok {
		return ""
	}
	return strings.TrimSpace(val)
}

func volumeName(ln string) string {
	val := afterColon(ln)
	if i := strings.Index(val, " ("); i > 0 {
		val = strings.TrimSpace(val[:i])
	}
	return val
}

// measure keeps the first size diskutil printed in parentheses, and the byte count beside it.
func measure(ln string) (display string, bytes int64, hasBytes bool) {
	val := afterColon(ln)
	fields := strings.Fields(val)
	if len(fields) >= 2 && fields[1] == "B" {
		n, err := strconv.ParseInt(fields[0], 10, 64)
		if err == nil {
			bytes = n
			hasBytes = true
		}
	}
	rest := val
	for {
		i := strings.Index(rest, "(")
		if i < 0 {
			break
		}
		j := strings.Index(rest[i:], ")")
		if j < 0 {
			break
		}
		inner := strings.TrimSpace(rest[i+1 : i+j])
		rest = rest[i+j+1:]
		if strings.Contains(inner, "%") || !strings.ContainsAny(inner, "0123456789") {
			continue
		}
		return inner, bytes, hasBytes
	}
	if hasBytes {
		return fields[0] + " B", bytes, true
	}
	return "", 0, false
}

// ParsePurgeable returns Apple's purgeable figure when the line has a number.
// It is not added to free.
func ParsePurgeable(text string) (string, bool) {
	for _, ln := range strings.Split(text, "\n") {
		if !strings.Contains(strings.ToLower(ln), "purgeable") {
			continue
		}
		_, val, ok := strings.Cut(ln, ":")
		if !ok {
			val = ln
		}
		val = strings.TrimSpace(val)
		if !strings.ContainsAny(val, "0123456789") {
			continue
		}
		if i := strings.Index(val, " ("); i > 0 {
			val = strings.TrimSpace(val[:i])
		}
		return val, true
	}
	return "", false
}

// ParseDU reads `du -hd 1` text. The home total is dropped. Largest first.
func ParseDU(text, home string) []File {
	home = filepath.Clean(home)
	var out []File
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		size, path, ok := splitSizePath(ln)
		if !ok || filepath.Clean(path) == home {
			continue
		}
		out = append(out, File{Path: path, Size: size})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Size == out[j].Size {
			return out[i].Path < out[j].Path
		}
		return out[i].Size > out[j].Size
	})
	return out
}

// ParseSized reads lines of "bytes path" or "4.0K path".
func ParseSized(text string) []File {
	var out []File
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		size, path, ok := splitSizePath(ln)
		if !ok {
			continue
		}
		out = append(out, File{Path: path, Size: size})
	}
	return out
}

func splitSizePath(ln string) (int64, string, bool) {
	ln = strings.TrimSpace(ln)
	if ln == "" {
		return 0, "", false
	}
	sizeText, path, ok := strings.Cut(ln, "\t")
	if !ok {
		sizeText, path, ok = strings.Cut(ln, " ")
	}
	if !ok {
		return 0, "", false
	}
	size, ok := parseHuman(strings.TrimSpace(sizeText))
	path = strings.TrimSpace(path)
	if !ok || path == "" {
		return 0, "", false
	}
	return size, path, true
}

func parseHuman(s string) (int64, bool) {
	if s == "" {
		return 0, false
	}
	mult := 1.0
	last := s[len(s)-1]
	switch last {
	case 'K', 'k':
		mult = 1024
		s = s[:len(s)-1]
	case 'M', 'm':
		mult = 1024 * 1024
		s = s[:len(s)-1]
	case 'G', 'g':
		mult = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case 'T', 't':
		mult = 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case 'B', 'b':
		s = s[:len(s)-1]
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 {
		return 0, false
	}
	return int64(f*mult + 0.5), true
}

// Human prints a byte count.
func Human(n int64) string {
	if n < 1024 {
		return strconv.FormatInt(n, 10) + "B"
	}
	units := []string{"K", "M", "G", "T"}
	v := float64(n)
	u := -1
	for v >= 1024 && u < len(units)-1 {
		v /= 1024
		u++
	}
	return fmt.Sprintf("%.1f%s", v, units[u])
}

// Category names the bucket for a path under home. First match wins.
// A Desktop tree stays Desktop, including build output that lives there.
func Category(home, path string) string {
	_ = home
	p := filepath.ToSlash(filepath.Clean(path))
	switch {
	case strings.Contains(p, "/Library/Caches") || strings.Contains(p, "/.npm/_cacache") || strings.Contains(p, "/Caches/Homebrew") || strings.Contains(p, "/Cache/"):
		return "Caches"
	case strings.Contains(p, "/Library/Logs") || strings.Contains(p, "/.npm/_logs"):
		return "Logs"
	case strings.Contains(p, "/Applications") || strings.Contains(p, "Library/Group Containers"):
		return "Applications"
	case strings.Contains(p, "/Documents"):
		return "Documents"
	case strings.Contains(p, "/Desktop"):
		return "Desktop"
	case strings.Contains(p, "/Downloads"):
		return "Downloads"
	case strings.Contains(p, "node_modules") || strings.Contains(p, "/go/pkg") || strings.Contains(p, "Library/Developer") || strings.Contains(p, "target/debug") || strings.Contains(p, "target/release"):
		return "Developer"
	default:
		return "Other"
	}
}

// PathsOf reads a find or mdfind listing. It does not stat and does not run du.
// The toolchain is dropped so the scan does not measure bhuppi's own compiler.
func PathsOf(home, text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || skipScan(line) {
			continue
		}
		out = append(out, line)
	}
	_ = home
	return out
}

func skipScan(path string) bool {
	p := filepath.ToSlash(path)
	return strings.Contains(p, "/.toolchain/") || strings.HasSuffix(p, "/.toolchain") || strings.Contains(p, "/bhuppi/.toolchain")
}

// MeasureArgv is the size commands still required after stat.
// A path that already has a size spawns nothing. A directory with no size
// spawns one du -sk of that directory, never one du per file inside it.
func MeasureArgv(files []File) [][]string {
	var out [][]string
	for _, f := range files {
		if f.Size >= 0 || f.Path == "" {
			continue
		}
		out = append(out, []string{"du", "-sk", f.Path})
	}
	return out
}

// Group sums sized paths into categories and rolls a folder up when many files inside it changed.
func Group(home string, files []File) map[string]Bucket {
	byCat := map[string][]File{}
	for _, f := range files {
		cat := Category(home, f.Path)
		byCat[cat] = append(byCat[cat], f)
	}
	out := map[string]Bucket{}
	for cat, list := range byCat {
		rolled := rollup(list)
		var total int64
		for _, f := range list {
			total += f.Size
		}
		sort.Slice(rolled, func(i, j int) bool {
			if rolled[i].Size == rolled[j].Size {
				return rolled[i].Path < rolled[j].Path
			}
			return rolled[i].Size > rolled[j].Size
		})
		out[cat] = Bucket{Total: total, Paths: rolled}
	}
	return out
}

func rollup(files []File) []File {
	byParent := map[string][]File{}
	var order []string
	for _, f := range files {
		parent := filepath.Dir(f.Path)
		if _, ok := byParent[parent]; !ok {
			order = append(order, parent)
		}
		byParent[parent] = append(byParent[parent], f)
	}
	var out []File
	for _, parent := range order {
		kids := byParent[parent]
		if len(kids) >= 3 {
			var sum int64
			for _, k := range kids {
				sum += k.Size
			}
			out = append(out, File{Path: parent, Size: sum})
			continue
		}
		out = append(out, kids...)
	}
	return out
}

// Allowlist is the only set of directories whose contents clean may remove.
func Allowlist(home string) []string {
	return []string{
		filepath.Join(home, "Library", "Caches"),
		filepath.Join(home, "Library", "Logs"),
		filepath.Join(home, ".npm", "_cacache"),
		filepath.Join(home, "Library", "Caches", "Homebrew"),
	}
}

func allowRoot(home, path string) bool {
	path = filepath.Clean(path)
	for _, dir := range Allowlist(home) {
		if filepath.Clean(dir) == path {
			return true
		}
	}
	return false
}

// InsideAllow reports whether path is strictly inside an allowlist directory.
func InsideAllow(home, path string) bool {
	path = filepath.Clean(path)
	for _, dir := range Allowlist(home) {
		dir = filepath.Clean(dir)
		rel, err := filepath.Rel(dir, path)
		if err != nil || rel == "." || rel == "" || strings.HasPrefix(rel, "..") {
			continue
		}
		return true
	}
	return false
}

// SafeNote is one sentence about a rebuildable cache. It is not a document.
func SafeNote(path string) string {
	switch {
	case strings.HasSuffix(path, "Library/Logs"):
		return "Logs are not documents. Removing this does not delete documents."
	case strings.HasSuffix(path, ".npm/_cacache"):
		return "npm recreates this download cache. Removing this does not delete documents."
	case strings.HasSuffix(path, "Library/Caches/Homebrew"):
		return "Homebrew recreates downloaded bottles. Removing this does not delete documents."
	default:
		return "Apps recreate caches. Removing this does not delete documents."
	}
}

// RemoveContents deletes the entries inside existing allowlist directories.
// It prints each path before it is removed. It does not remove the directories
// themselves, and it refuses any path outside the allowlist.
func RemoveContents(home string, w io.Writer) error {
	for _, dir := range Allowlist(home) {
		st, err := os.Stat(dir)
		if err != nil || !st.IsDir() {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			p := filepath.Join(dir, e.Name())
			if allowRoot(home, p) || !InsideAllow(home, p) {
				continue
			}
			fmt.Fprintln(w, p)
			if err := os.RemoveAll(p); err != nil {
				return err
			}
		}
	}
	return nil
}

const reviewWidth = 100

// Summary is the disk block. NoteKey is unaccounted or reserved, never a rewritten total.
type Summary struct {
	Key, Label              string
	Size, Used, Free, Purge string
	NoteKey, NoteVal        string
}

// FormatSummary is the disk block. Purgeable is omitted when empty and is not added to free.
func FormatSummary(s Summary, color bool) string {
	if s.Key == "" {
		s.Key = "volume"
	}
	if s.Label == "" {
		s.Label = "/"
	}
	var b strings.Builder
	fmt.Fprintln(&b, paint(style.Accent, "Summary", color))
	writeKV(&b, s.Key, s.Label, style.Path, color)
	writeKV(&b, "size", s.Size, style.Number, color)
	writeKV(&b, "used", s.Used, style.Number, color)
	writeKV(&b, "free", s.Free, style.Number, color)
	if s.Purge != "" {
		writeKV(&b, "purgeable", s.Purge, style.Number, color)
	}
	if s.NoteKey != "" && s.NoteVal != "" {
		writeKV(&b, s.NoteKey, s.NoteVal, style.Number, color)
	}
	fmt.Fprintln(&b)
	return b.String()
}

// FromAPFS labels the container. The three figures stay as diskutil printed them.
func FromAPFS(a APFS, purge string) Summary {
	s := Summary{
		Key: "disk", Label: "APFS container " + a.Ref,
		Size: a.Size, Used: a.Used, Free: a.Free, Purge: purge,
	}
	s.NoteKey, s.NoteVal = gapNote(a.SizeB, a.UsedB, a.FreeB, a.HaveBytes, "unaccounted")
	return s
}

// FromDF labels a df mount. Linux keeps /, and a root reserve is named reserved.
// scale is 1 for a human df field already in bytes, or 1024 for df -k blocks.
func FromDF(mount, size, used, free, purge string, scale int64, human bool) Summary {
	if mount == "" {
		mount = "/"
	}
	s := Summary{Key: "volume", Label: mount, Size: size, Used: used, Free: free, Purge: purge}
	var sb, ub, fb int64
	var ok bool
	if human {
		sb, ub, fb, ok = humanBytes(size, used, free)
	} else if scale > 0 {
		sb, ub, fb, ok = blockBytes(size, used, free, scale)
	}
	s.NoteKey, s.NoteVal = gapNote(sb, ub, fb, ok, "reserved")
	return s
}

func gapNote(sizeB, usedB, freeB int64, have bool, key string) (string, string) {
	if !have {
		return "", ""
	}
	signed := sizeB - usedB - freeB
	abs := signed
	if abs < 0 {
		abs = -abs
	}
	if abs <= 1<<30 {
		return "", ""
	}
	if key == "reserved" && signed <= 0 {
		return "", ""
	}
	return key, Human(abs)
}

func humanBytes(size, used, free string) (int64, int64, int64, bool) {
	sb, ok1 := parseDFHuman(size)
	ub, ok2 := parseDFHuman(used)
	fb, ok3 := parseDFHuman(free)
	return sb, ub, fb, ok1 && ok2 && ok3
}

func blockBytes(size, used, free string, scale int64) (int64, int64, int64, bool) {
	sb, ok1 := strconv.ParseInt(strings.TrimSpace(size), 10, 64)
	ub, ok2 := strconv.ParseInt(strings.TrimSpace(used), 10, 64)
	fb, ok3 := strconv.ParseInt(strings.TrimSpace(free), 10, 64)
	if ok1 != nil || ok2 != nil || ok3 != nil {
		return 0, 0, 0, false
	}
	return sb * scale, ub * scale, fb * scale, true
}

func parseDFHuman(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
		i++
	}
	if i == 0 {
		return 0, false
	}
	f, err := strconv.ParseFloat(s[:i], 64)
	if err != nil || f < 0 {
		return 0, false
	}
	unit := strings.ToLower(strings.TrimSpace(s[i:]))
	var mult float64
	switch unit {
	case "", "b":
		mult = 1
	case "k", "ki", "kb":
		mult = 1024
	case "m", "mi", "mb":
		mult = 1024 * 1024
	case "g", "gi", "gb":
		mult = 1024 * 1024 * 1024
	case "t", "ti", "tb":
		mult = 1024 * 1024 * 1024 * 1024
	default:
		return 0, false
	}
	return int64(f*mult + 0.5), true
}

// FormatVolumes lists container volumes, largest first, in the same boxes as categories.
func FormatVolumes(vols []APFSVol, color bool) string {
	if len(vols) == 0 {
		return ""
	}
	list := append([]APFSVol(nil), vols...)
	sort.Slice(list, func(i, j int) bool {
		if list[i].Bytes == list[j].Bytes {
			return list[i].Name < list[j].Name
		}
		return list[i].Bytes > list[j].Bytes
	})
	var b strings.Builder
	fmt.Fprintln(&b, paint(style.Accent, "── volumes ──", color))
	fmt.Fprintln(&b)
	for _, v := range list {
		writeBox(&b, v.Name, v.Consumed, nil, color)
	}
	return b.String()
}

func writeKV(b *strings.Builder, key, val string, role style.Role, color bool) {
	fmt.Fprintf(b, "  %-12s %s\n", key, paint(role, val, color))
}

// FormatAllTime lists the largest depth-1 folders in one box.
func FormatAllTime(home string, files []File, color bool) string {
	var b strings.Builder
	fmt.Fprintln(&b, paint(style.Accent, "── all-time ──", color))
	fmt.Fprintln(&b)
	if len(files) == 0 {
		fmt.Fprintln(&b, "  none")
		fmt.Fprintln(&b)
		return b.String()
	}
	var sum int64
	for _, f := range files {
		sum += f.Size
	}
	shown, omitted := capFiles(files, 15)
	writeBox(&b, "folders", Human(sum), fileRows(home, shown), color)
	if omitted > 0 {
		fmt.Fprintf(&b, "%d paths omitted\n\n", omitted)
	}
	return b.String()
}

// FormatInterval prints one time window. Each category is its own box, largest first.
func FormatInterval(title, home string, files []File, color bool) string {
	var b strings.Builder
	fmt.Fprintln(&b, paint(style.Accent, "── "+title+" ──", color))
	fmt.Fprintln(&b)
	grouped := Group(home, files)
	type sized struct {
		name   string
		bucket Bucket
	}
	var cats []sized
	for name, bucket := range grouped {
		if bucket.Total == 0 {
			continue
		}
		cats = append(cats, sized{name: name, bucket: bucket})
	}
	sort.Slice(cats, func(i, j int) bool {
		if cats[i].bucket.Total == cats[j].bucket.Total {
			return cats[i].name < cats[j].name
		}
		return cats[i].bucket.Total > cats[j].bucket.Total
	})
	if len(cats) == 0 {
		fmt.Fprintln(&b, "  none")
		fmt.Fprintln(&b)
		return b.String()
	}
	var omitted int
	for _, cat := range cats {
		shown, skip := capFiles(cat.bucket.Paths, 10)
		omitted += skip
		writeBox(&b, cat.name, Human(cat.bucket.Total), fileRows(home, shown), color)
	}
	if omitted > 0 {
		fmt.Fprintf(&b, "%d paths omitted\n\n", omitted)
	}
	return b.String()
}

func fileRows(home string, files []File) []boxRow {
	shown := fitPaths(home, files)
	rows := make([]boxRow, len(files))
	for i, f := range files {
		rows[i] = boxRow{size: Human(f.Size), text: shown[i], role: style.Path}
	}
	return rows
}

func capFiles(files []File, n int) ([]File, int) {
	if len(files) <= n {
		return files, 0
	}
	return files[:n], len(files) - n
}

// Safe is one rebuildable directory.
type Safe struct {
	Path string
	Size string
}

// FormatSafe lists caches that can be recreated and the clean command, which does not run.
func FormatSafe(home string, rows []Safe, color bool) string {
	var b strings.Builder
	fmt.Fprintln(&b, paint(style.Accent, "── safe to remove ──", color))
	fmt.Fprintln(&b)
	if len(rows) == 0 {
		fmt.Fprintln(&b, "  none")
		fmt.Fprintln(&b)
	}
	for _, row := range rows {
		path, _ := shrinkPath(homePath(home, row.Path), reviewWidth-2)
		writeBox(&b, safeTitle(row.Path), row.Size, []boxRow{
			{text: path, role: style.Path},
			{text: SafeNote(row.Path), role: style.Muted},
		}, color)
	}
	fmt.Fprintln(&b, "bhuppi disk clean")
	fmt.Fprintln(&b, "does not run unless you add --yes")
	return b.String()
}

func safeTitle(path string) string {
	switch {
	case strings.HasSuffix(filepath.ToSlash(path), "Library/Caches/Homebrew"):
		return "Homebrew"
	case strings.HasSuffix(filepath.ToSlash(path), ".npm/_cacache"):
		return "npm"
	case strings.HasSuffix(filepath.ToSlash(path), "Library/Logs"):
		return "Logs"
	case strings.HasSuffix(filepath.ToSlash(path), "Library/Caches"):
		return "Caches"
	default:
		return filepath.Base(path)
	}
}

type boxRow struct {
	size string
	text string
	role style.Role
}

func writeBox(b *strings.Builder, title, total string, rows []boxRow, color bool) {
	fmt.Fprintf(b, "┌ %s  %s\n", paint(style.Accent, title, color), paint(style.Number, total, color))
	for _, row := range rows {
		if row.size != "" {
			fmt.Fprintf(b, "│ %s  %s\n", paint(style.Number, row.size, color), paint(row.role, row.text, color))
			continue
		}
		fmt.Fprintf(b, "│ %s\n", paint(row.role, row.text, color))
	}
	fmt.Fprintln(b, "└")
	fmt.Fprintln(b)
}

// fitPaths shortens the middle of a path when the row would pass 100 columns.
// The home-relative start and the final two components stay, and two different
// paths in the same box do not collapse to one string.
func fitPaths(home string, files []File) []string {
	shown := make([]string, len(files))
	keep := make([]int, len(files))
	for i, f := range files {
		full := homePath(home, f.Path)
		shown[i], keep[i] = shrinkPath(full, reviewWidth-rowChrome(Human(f.Size)))
	}
	for guard := 0; guard < 64; guard++ {
		dup := map[string][]int{}
		for i, s := range shown {
			dup[s] = append(dup[s], i)
		}
		grew := false
		for _, idx := range dup {
			if len(idx) < 2 {
				continue
			}
			for _, i := range idx {
				next, k, ok := expandPath(homePath(home, files[i].Path), keep[i])
				if !ok || next == shown[i] {
					continue
				}
				shown[i] = next
				keep[i] = k
				grew = true
			}
		}
		if !grew {
			break
		}
	}
	return shown
}

func rowChrome(size string) int {
	// "│ " + size + "  "
	return 4 + utf8.RuneCountInString(size)
}

func homePath(home, path string) string {
	cleanPath := filepath.Clean(path)
	cleanHome := filepath.Clean(home)
	if home != "" && (cleanPath == cleanHome || strings.HasPrefix(cleanPath, cleanHome+string(filepath.Separator))) {
		rel := strings.TrimPrefix(cleanPath, cleanHome)
		rel = strings.TrimPrefix(rel, string(filepath.Separator))
		if rel == "" {
			return "~"
		}
		return "~" + string(filepath.Separator) + rel
	}
	return filepath.ToSlash(cleanPath)
}

func pathParts(shown string) []string {
	rest := strings.TrimPrefix(shown, "~/")
	if rest == shown || rest == "" {
		return strings.Split(strings.Trim(filepath.ToSlash(shown), "/"), "/")
	}
	return strings.Split(rest, "/")
}

func shrinkPath(full string, budget int) (string, int) {
	if utf8.RuneCountInString(full) <= budget {
		return full, -1
	}
	parts := pathParts(full)
	if len(parts) < 4 {
		return full, -1
	}
	best := full
	bestKeep := 1
	for keep := len(parts) - 3; keep >= 1; keep-- {
		cand := joinShrink(full, parts, keep)
		best = cand
		bestKeep = keep
		if utf8.RuneCountInString(cand) <= budget {
			return cand, keep
		}
	}
	return best, bestKeep
}

func expandPath(full string, keep int) (string, int, bool) {
	if keep < 0 {
		return full, -1, false
	}
	parts := pathParts(full)
	if keep+1 > len(parts)-3 {
		return full, -1, full != joinShrink(full, parts, keep)
	}
	return joinShrink(full, parts, keep+1), keep + 1, true
}

func joinShrink(full string, parts []string, keep int) string {
	head := strings.Join(parts[:keep], "/")
	tail := strings.Join(parts[len(parts)-2:], "/")
	if strings.HasPrefix(full, "~/") {
		return "~/" + head + "/…/" + tail
	}
	return "/" + head + "/…/" + tail
}

func paint(role style.Role, text string, color bool) string {
	if !color || text == "" {
		return text
	}
	return style.Paint(role, text, style.Dark, false)
}
