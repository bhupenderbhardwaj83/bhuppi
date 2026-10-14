package diskreview

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestGroupCachesAndDocuments(t *testing.T) {
	home := "/Users/ada"
	text := "" +
		"2048 /Users/ada/Library/Caches/com.apple/Cache.db\n" +
		"9216 /Users/ada/Documents/notes.txt\n"
	got := Group(home, ParseSized(text))
	if got["Caches"].Total != 2048 {
		t.Fatalf("caches %d", got["Caches"].Total)
	}
	if got["Documents"].Total != 9216 {
		t.Fatalf("documents %d", got["Documents"].Total)
	}
	if Category(home, "/Users/ada/Library/Caches/com.apple/Cache.db") != "Caches" {
		t.Fatal("caches category")
	}
	if Category(home, "/Users/ada/Documents/notes.txt") != "Documents" {
		t.Fatal("documents category")
	}
	page := FormatInterval("last hour", home, ParseSized(text), false)
	if !strings.Contains(page, "Caches") || !strings.Contains(page, "Documents") || !strings.Contains(page, "2.0K") || !strings.Contains(page, "9.0K") {
		t.Fatalf("page:\n%s", page)
	}
}

func TestBlocksAndNpmCache(t *testing.T) {
	home := "/Users/ada"
	npm := "/Users/ada/.npm/_cacache/content-v2/sha512/abcd"
	logs := "/Users/ada/Library/Logs/system.log"
	if Category(home, npm) != "Caches" {
		t.Fatal("npm cache must be Caches")
	}
	if Category(home, logs) != "Logs" {
		t.Fatal("library logs")
	}
	if Category(home, "/Users/ada/Desktop/app/src-tauri/target/debug/app") != "Desktop" {
		t.Fatal("desktop build output stays Desktop")
	}
	if Category(home, "/Users/ada/src/app/target/release/app") != "Developer" {
		t.Fatal("target/release")
	}
	page := FormatInterval("last 3 days", home, []File{{Path: npm, Size: 2048}, {Path: logs, Size: 512}}, false)
	if !strings.Contains(page, "┌ Caches") || !strings.Contains(page, "┌ Logs") {
		t.Fatalf("blocks:\n%s", page)
	}
	if !strings.Contains(page, "└\n\n┌") {
		t.Fatalf("blocks ran together:\n%s", page)
	}
	caches := strings.Index(page, "┌ Caches")
	logAt := strings.Index(page, "┌ Logs")
	if caches < 0 || logAt < caches {
		t.Fatalf("order:\n%s", page)
	}
	if strings.Contains(page[logAt:], "_cacache") {
		t.Fatal("npm cache listed under Logs")
	}
	report := FormatSummary(Summary{Size: "460Gi", Used: "12Gi", Free: "80Gi", Purge: "12.3 GB"}, false) + page
	summary := strings.Index(report, "Summary")
	if summary < 0 || strings.TrimLeft(report, "\n")[:7] != "Summary" {
		t.Fatal("summary is not first")
	}
	for _, ln := range strings.Split(report[:summary], "\n") {
		if strings.TrimSpace(ln) == "volume" {
			t.Fatal("bare volume before summary")
		}
	}
}

func TestShortenKeepsEndsDistinct(t *testing.T) {
	home := "/Users/ada"
	mid := strings.Repeat("segment/", 15)
	one := "/Users/ada/Library/Caches/" + mid + "Resources/File.bin"
	got := fitPaths(home, []File{{Path: one, Size: 2048}})
	line := "│ " + Human(int64(2048)) + "  " + got[0]
	if !strings.HasPrefix(got[0], "~/") || !strings.HasSuffix(got[0], "Resources/File.bin") || !strings.Contains(got[0], "…") {
		t.Fatalf("shortened %s", got[0])
	}
	if utf8.RuneCountInString(line) > 100 {
		t.Fatalf("line wraps %d %s", utf8.RuneCountInString(line), line)
	}
	a := "/Users/ada/Library/Caches/" + mid + "alpha/Resources/File.bin"
	b := "/Users/ada/Library/Caches/" + mid + "beta/Resources/File.bin"
	pair := fitPaths(home, []File{{Path: a, Size: 2048}, {Path: b, Size: 2048}})
	if pair[0] == pair[1] || !strings.Contains(pair[0], "alpha") || !strings.Contains(pair[1], "beta") {
		t.Fatalf("collapsed:\n%s\n%s", pair[0], pair[1])
	}
	short := fitPaths(home, []File{{Path: "/Users/ada/Documents/notes.txt", Size: 100}})
	if short[0] != "~/Documents/notes.txt" {
		t.Fatal(short[0])
	}
}

func TestCleanStaysInsideAllowlist(t *testing.T) {
	home := t.TempDir()
	cache := filepath.Join(home, "Library", "Caches")
	if err := os.MkdirAll(filepath.Join(cache, "junk"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "junk", "a"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	brew := filepath.Join(cache, "Homebrew")
	if err := os.MkdirAll(brew, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brew, "a.tar"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	docs := filepath.Join(home, "Documents", "keep.txt")
	if err := os.MkdirAll(filepath.Dir(docs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(docs, []byte("doc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if InsideAllow(home, docs) {
		t.Fatal("documents are not in the allowlist")
	}
	if !InsideAllow(home, filepath.Join(cache, "junk")) {
		t.Fatal("cache child should be allowed")
	}
	var log strings.Builder
	if err := RemoveContents(home, &log); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(docs); err != nil {
		t.Fatal("document removed")
	}
	if _, err := os.Stat(cache); err != nil {
		t.Fatal("cache directory itself was removed")
	}
	if _, err := os.Stat(brew); err != nil {
		t.Fatal("homebrew directory itself was removed")
	}
	if _, err := os.Stat(filepath.Join(brew, "a.tar")); !os.IsNotExist(err) {
		t.Fatal("homebrew cache contents remain")
	}
	if _, err := os.Stat(filepath.Join(cache, "junk")); !os.IsNotExist(err) {
		t.Fatal("cache contents remain")
	}
	if !strings.Contains(log.String(), filepath.Join(cache, "junk")) {
		t.Fatal("path was not printed before removal")
	}
}

func TestReviewHourArgv(t *testing.T) {
	steps, err := ReviewArgv("/Users/ada", "darwin", []string{"--hour"})
	if err != nil {
		t.Fatal(err)
	}
	flat := ""
	for _, st := range steps {
		flat += strings.Join(st, " ") + "\n"
	}
	if !strings.Contains(flat, "diskutil apfs list") || !strings.Contains(flat, "mdfind") || !strings.Contains(flat, "kMDItemFSContentChangeDate") || !strings.Contains(flat, "$time.now(-3600)") {
		t.Fatalf("hour argv:\n%s", flat)
	}
	if strings.Contains(flat, "df -h /") || strings.Contains(flat, "find /Users/ada -xdev") {
		t.Fatalf("darwin hour still walks the home directory:\n%s", flat)
	}
	if !strings.Contains(flat, "find /Users/ada/Library/Caches") {
		t.Fatalf("hour missing bounded cache find:\n%s", flat)
	}
	if strings.Contains(flat, "-mtime") || strings.Contains(flat, "du ") {
		t.Fatalf("hour included another interval:\n%s", flat)
	}
	all, err := ReviewArgv("/Users/ada", "darwin", nil)
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for _, st := range all {
		joined += strings.Join(st, " ") + "\n"
	}
	for _, want := range []string{"$time.now(-3600)", "$time.today(-1)", "$time.today(-7)", "$time.today(-30)", "du -hd 1", "diskutil apfs list"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("default missing %s\n%s", want, joined)
		}
	}
	linux, err := ReviewArgv("/Users/ada", "linux", []string{"--hour"})
	if err != nil {
		t.Fatal(err)
	}
	linuxFlat := ""
	for _, st := range linux {
		linuxFlat += strings.Join(st, " ") + "\n"
	}
	if !strings.Contains(linuxFlat, "df -h /") || strings.Contains(linuxFlat, "apfs") {
		t.Fatalf("linux argv:\n%s", linuxFlat)
	}
}

func TestStatSizesDoNotDuEachFile(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&b, "%d /Users/ada/Desktop/proj/target/debug/file%d\n", 1000+i, i)
	}
	b.WriteString("2048 /Users/ada/Library/Caches/com.apple/Cache.db\n")
	b.WriteString("9216 /Users/ada/Documents/notes.txt\n")
	files := ParseSized(b.String())
	if len(MeasureArgv(files)) != 0 {
		t.Fatalf("stat text spawned %d du commands", len(MeasureArgv(files)))
	}
	got := Group("/Users/ada", files)
	if got["Desktop"].Total == 0 || got["Caches"].Total != 2048 || got["Documents"].Total != 9216 {
		t.Fatalf("totals %+v", got)
	}
	pending := []File{{Path: "/Users/ada/proj/node_modules", Size: -1}}
	cmds := MeasureArgv(pending)
	if len(cmds) != 1 || strings.Join(cmds[0], " ") != "du -sk /Users/ada/proj/node_modules" {
		t.Fatalf("directory measure: %v", cmds)
	}
	page := FormatInterval("last 3 days", "/Users/ada", files, false)
	if !strings.Contains(page, "┌ Desktop") || !strings.Contains(page, "target/debug") {
		t.Fatalf("desktop target hidden:\n%s", page)
	}
}

func TestAPFSContainerNotSystemVolume(t *testing.T) {
	text := "" +
		"APFS Containers (2 found)\n" +
		"+-- Container disk3 AAAA\n" +
		"    APFS Container Reference:     disk3\n" +
		"    Size (Capacity Ceiling):      493921239040 B (460Gi)\n" +
		"    Capacity In Use By Volumes:   472446402560 B (440Gi)\n" +
		"    Capacity Not Allocated:       21474836480 B (20Gi)\n" +
		"    +-> Volume disk3s1 BBBB\n" +
		"        APFS Volume Disk (Role):   disk3s1 (System)\n" +
		"        Name:                      Macintosh HD (Case-insensitive)\n" +
		"        Mount Point:               Not Mounted\n" +
		"        Capacity Consumed:         13958643712 B (13Gi)\n" +
		"        Snapshot Mount Point:      /\n" +
		"    +-> Volume disk3s5 CCCC\n" +
		"        APFS Volume Disk (Role):   disk3s5 (Data)\n" +
		"        Name:                      Data (Case-insensitive)\n" +
		"        Mount Point:               /System/Volumes/Data\n" +
		"        Capacity Consumed:         458466426880 B (427Gi)\n" +
		"    +-> Volume disk3s6 DDDD\n" +
		"        APFS Volume Disk (Role):   disk3s6 (No specific role)\n" +
		"        Name:                      Empty\n" +
		"+-- Container disk5 EEEE\n" +
		"    APFS Container Reference:     disk5\n" +
		"    Size (Capacity Ceiling):      1000 B (1 KB)\n" +
		"    Capacity In Use By Volumes:   1000 B (1 KB)\n" +
		"    Capacity Not Allocated:       0 B (0 B)\n" +
		"    +-> Volume disk5s1 FFFF\n" +
		"        APFS Volume Disk (Role):   disk5s1 (No specific role)\n" +
		"        Name:                      Other\n" +
		"        Mount Point:               /Volumes/Other\n" +
		"        Capacity Consumed:         1000 B (1 KB)\n"
	got, ok := ParseAPFS(text)
	if !ok || got.Ref != "disk3" || got.Used != "440Gi" || got.Size != "460Gi" || got.Free != "20Gi" {
		t.Fatalf("%+v ok=%v", got, ok)
	}
	page := FormatSummary(FromAPFS(got, ""), false) + FormatVolumes(got.Volumes, false)
	if summaryField(page, "used") != "440Gi" {
		t.Fatalf("used was the system volume:\n%s", page)
	}
	if summaryField(page, "disk") != "APFS container disk3" {
		t.Fatalf("label:\n%s", page)
	}
	if strings.Contains(page[:strings.Index(page, "── volumes ──")], "13Gi") {
		t.Fatal("13Gi used as the disk")
	}
	if !strings.Contains(page, "┌ Macintosh HD") || !strings.Contains(page, "13Gi") || !strings.Contains(page, "┌ Data") || !strings.Contains(page, "427Gi") {
		t.Fatalf("volumes:\n%s", page)
	}
	if strings.Contains(page, "Empty") || strings.Contains(page, "Other") {
		t.Fatal("skipped volume was shown")
	}
	dataAt := strings.Index(page, "┌ Data")
	hdAt := strings.Index(page, "┌ Macintosh HD")
	if dataAt < 0 || hdAt < dataAt {
		t.Fatal("volumes are not largest first")
	}
	if strings.Contains(page, "unaccounted") {
		t.Fatal("matching container printed unaccounted")
	}

	bad := strings.Replace(text, "472446402560 B (440Gi)", "429496729600 B (400Gi)", 1)
	miss, ok := ParseAPFS(bad)
	if !ok || miss.Used != "400Gi" || miss.Size != "460Gi" || miss.Free != "20Gi" {
		t.Fatalf("rewritten %+v", miss)
	}
	again := FormatSummary(FromAPFS(miss, "12.3 GB"), false)
	if summaryField(again, "used") != "400Gi" || summaryField(again, "size") != "460Gi" || summaryField(again, "free") != "20Gi" {
		t.Fatalf("figures changed:\n%s", again)
	}
	if !strings.Contains(again, "unaccounted") || strings.Contains(again, "440Gi") {
		t.Fatalf("unaccounted:\n%s", again)
	}
	if strings.Contains(summaryField(again, "free")+summaryField(again, "used"), "12.3") {
		t.Fatal("purgeable was added")
	}
}

func summaryField(page, key string) string {
	for _, ln := range strings.Split(page, "\n") {
		f := strings.Fields(ln)
		if len(f) >= 2 && f[0] == key {
			return strings.Join(f[1:], " ")
		}
	}
	return ""
}

func TestPurgeableStaysSeparate(t *testing.T) {
	df := "Filesystem Size Used Avail Capacity Mounted on\n/dev/disk1 460Gi 12Gi 80Gi 14% /\n"
	size, used, free, ok := ParseDF(df)
	if !ok || size != "460Gi" || used != "12Gi" || free != "80Gi" {
		t.Fatalf("%q %q %q", size, used, free)
	}
	purge, ok := ParsePurgeable("Volume Name: Macintosh HD\nPurgeable Space: 12.3 GB (999 Bytes)\n")
	if !ok || purge != "12.3 GB" || strings.Contains(free+purge, "92") {
		t.Fatal(purge)
	}
	page := FormatSummary(Summary{Size: size, Used: used, Free: free, Purge: purge}, false)
	if strings.Contains(page, "92") {
		t.Fatal("purgeable was added to free")
	}
}
