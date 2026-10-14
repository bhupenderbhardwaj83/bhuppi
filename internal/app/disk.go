package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/bhupenderbhardwaj/bhuppi/internal/diskreview"
	"github.com/bhupenderbhardwaj/bhuppi/internal/dispatch"
	"github.com/bhupenderbhardwaj/bhuppi/internal/help"
	"github.com/bhupenderbhardwaj/bhuppi/internal/warn"
)

func diskCmd(f parsed, opt help.Options, stdout, stderr io.Writer) int {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		fmt.Fprintf(stderr, "disk: home directory is not set\n")
		return 1
	}
	args := f.Args[2:]
	if f.Args[1] == "clean" {
		return diskClean(home, args, f.Dry, opt.Color, stdout, stderr)
	}
	steps, err := diskreview.ReviewArgv(home, runtime.GOOS, args)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", err.Error())
		return 2
	}
	if f.Dry {
		var b strings.Builder
		for _, st := range steps {
			fmt.Fprintln(&b, warn.Show(dispatch.FormatArgv(st), opt.Color))
		}
		emit(stdout, b.String())
		return 0
	}
	return diskReview(home, steps, opt.Color, stdout, stderr)
}

func diskReview(home string, steps [][]string, color bool, stdout, stderr io.Writer) int {
	var apfs diskreview.APFS
	var gotAPFS bool
	var purge string
	var dfSize, dfUsed, dfFree, dfMount string
	var dfOK, dfHuman bool
	for _, st := range steps {
		if len(st) == 0 {
			continue
		}
		switch st[0] {
		case "diskutil":
			text, err := runQuiet(st)
			if err != nil {
				continue
			}
			if len(st) >= 3 && st[1] == "apfs" {
				if parsed, ok := diskreview.ParseAPFS(text); ok {
					apfs = parsed
					gotAPFS = true
				}
				continue
			}
			if p, ok := diskreview.ParsePurgeable(text); ok {
				purge = p
			}
		case "df":
			text, err := runQuiet(st)
			if err != nil || len(st) < 2 {
				continue
			}
			mount := st[len(st)-1]
			size, used, free, ok := diskreview.ParseDFMount(text, mount)
			if !ok {
				continue
			}
			dfSize, dfUsed, dfFree, dfMount = size, used, free, mount
			dfOK = true
			dfHuman = st[1] == "-h"
		}
	}
	if !gotAPFS && runtime.GOOS == "darwin" {
		text, err := runQuiet([]string{"df", "-k", "/System/Volumes/Data"})
		if err == nil {
			size, used, free, ok := diskreview.ParseDFMount(text, "/System/Volumes/Data")
			if ok {
				dfSize, dfUsed, dfFree, dfMount = size, used, free, "/System/Volumes/Data"
				dfOK = true
				dfHuman = false
			}
		}
	}
	var summary diskreview.Summary
	if gotAPFS {
		summary = diskreview.FromAPFS(apfs, purge)
	} else if dfOK && dfHuman {
		summary = diskreview.FromDF(dfMount, dfSize, dfUsed, dfFree, purge, 0, true)
	} else if dfOK {
		summary = diskreview.FromDF(dfMount, dfSize, dfUsed, dfFree, purge, 1024, false)
	} else {
		summary = diskreview.FromDF("/", "", "", "", purge, 0, false)
	}
	var head strings.Builder
	head.WriteString(diskreview.FormatSummary(summary, color))
	if gotAPFS {
		head.WriteString(diskreview.FormatVolumes(apfs.Volumes, color))
	}
	emit(stdout, head.String())

	var duArgv []string
	var intervals [][]string
	for _, st := range steps {
		if len(st) == 0 {
			continue
		}
		switch st[0] {
		case "du":
			duArgv = st
		case "find", "mdfind":
			intervals = append(intervals, st)
		}
	}
	// The home total walks the disk. It runs beside the interval query so the
	// category boxes are not waiting on a second full walk.
	duText := make(chan string, 1)
	if duArgv != nil {
		go func() {
			text, _ := runQuiet(duArgv)
			duText <- text
		}()
	} else {
		close(duText)
	}
	for _, group := range groupIntervals(intervals) {
		title := diskreview.IntervalTitle(group.primary)
		fmt.Fprintln(stderr, title)
		type listing struct{ text string }
		extraCh := make(chan listing, len(group.extra))
		for _, ex := range group.extra {
			go func(ex []string) {
				text, _ := runQuiet(ex)
				extraCh <- listing{text}
			}(ex)
		}
		text, _ := runQuiet(group.primary)
		files, pending := statListing(diskreview.PathsOf(home, text))
		emit(stdout, diskreview.FormatInterval(title, home, files, color))
		var more []diskreview.File
		seen := map[string]bool{}
		for _, f := range files {
			seen[f.Path] = true
		}
		for range group.extra {
			got := <-extraCh
			add, _ := statListing(diskreview.PathsOf(home, got.text))
			for _, f := range add {
				if seen[f.Path] {
					continue
				}
				seen[f.Path] = true
				more = append(more, f)
			}
		}
		more = append(more, duDirectories(pending, files)...)
		if len(more) > 0 {
			emit(stdout, diskreview.FormatInterval(title, home, more, color))
		}
	}
	if duArgv != nil {
		emit(stdout, diskreview.FormatAllTime(home, diskreview.ParseDU(<-duText, home), color))
	}
	emit(stdout, diskreview.FormatSafe(home, safeRows(home), color))
	return 0
}

type intervalGroup struct {
	primary []string
	extra   [][]string
}

// groupIntervals keeps one Spotlight query as the primary and attaches the
// bounded finds for the same window. Those finds must not delay the first box.
func groupIntervals(steps [][]string) []intervalGroup {
	var out []intervalGroup
	for _, st := range steps {
		if len(st) == 0 {
			continue
		}
		if st[0] == "mdfind" || len(out) == 0 || diskreview.IntervalTitle(st) != diskreview.IntervalTitle(out[len(out)-1].primary) {
			out = append(out, intervalGroup{primary: st})
			continue
		}
		out[len(out)-1].extra = append(out[len(out)-1].extra, st)
	}
	return out
}

// statListing sizes files with stat. A node_modules or .git directory is left
// unsized so it can be one du -sk later. Every other directory is skipped:
// its files, when Spotlight listed them, already carry the bytes.
func statListing(paths []string) (files []diskreview.File, pending []diskreview.File) {
	type item struct {
		path string
		size int64
		dir  bool
		ok   bool
	}
	out := make([]item, len(paths))
	jobs := make(chan int)
	var wg sync.WaitGroup
	workers := 32
	if len(paths) < workers {
		workers = len(paths)
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				st, err := os.Stat(paths[i])
				if err != nil {
					continue
				}
				if st.IsDir() {
					if dirUnit(paths[i]) {
						out[i] = item{path: paths[i], size: -1, dir: true, ok: true}
					}
					continue
				}
				out[i] = item{path: paths[i], size: st.Size(), ok: true}
			}
		}()
	}
	for i := range paths {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	for _, it := range out {
		if !it.ok {
			continue
		}
		if it.dir {
			pending = append(pending, diskreview.File{Path: it.path, Size: -1})
			continue
		}
		files = append(files, diskreview.File{Path: it.path, Size: it.size})
	}
	return files, pending
}

func dirUnit(path string) bool {
	base := filepath.Base(path)
	return base == "node_modules" || base == ".git"
}

func duDirectories(pending, files []diskreview.File) []diskreview.File {
	var extra []diskreview.File
	for _, cmd := range diskreview.MeasureArgv(pending) {
		if len(cmd) < 3 || covered(cmd[2], files) {
			continue
		}
		text, err := runQuiet(cmd)
		if err != nil && strings.TrimSpace(text) == "" {
			continue
		}
		kb, ok := parseDuK(text)
		if !ok {
			continue
		}
		extra = append(extra, diskreview.File{Path: cmd[2], Size: kb * 1024})
	}
	return extra
}

func covered(dir string, files []diskreview.File) bool {
	prefix := dir + string(filepath.Separator)
	for _, f := range files {
		if f.Path == dir || strings.HasPrefix(f.Path, prefix) {
			return true
		}
	}
	return false
}

func parseDuK(text string) (int64, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func safeRows(home string) []diskreview.Safe {
	var rows []diskreview.Safe
	for _, path := range diskreview.Allowlist(home) {
		st, err := os.Stat(path)
		if err != nil || !st.IsDir() {
			continue
		}
		text, err := runQuiet([]string{"du", "-sh", path})
		if err != nil && strings.TrimSpace(text) == "" {
			continue
		}
		size := strings.TrimSpace(text)
		if i := strings.IndexAny(size, " \t"); i > 0 {
			size = size[:i]
		}
		if size == "" {
			size = "0B"
		}
		rows = append(rows, diskreview.Safe{Path: path, Size: size})
	}
	return rows
}

func diskClean(home string, args []string, dry, color bool, stdout, stderr io.Writer) int {
	if len(args) > 1 || (len(args) == 1 && args[0] != "--yes") {
		fmt.Fprintf(stderr, "disk clean takes --yes, or nothing\n")
		return 2
	}
	yes := len(args) == 1
	if !yes {
		if dry {
			fmt.Fprintln(stdout, "bhuppi disk clean deletes nothing without --yes")
			return 0
		}
		emit(stdout, diskreview.FormatSafe(home, safeRows(home), color))
		return 0
	}
	var lines []string
	for _, dir := range diskreview.Allowlist(home) {
		st, err := os.Stat(dir)
		if err != nil || !st.IsDir() {
			continue
		}
		lines = append(lines, warn.Show(dispatch.FormatArgv([]string{"find", dir, "-mindepth", "1", "-delete"}), color))
	}
	if dry {
		if len(lines) == 0 {
			fmt.Fprintln(stdout, "bhuppi disk clean --yes has nothing in the allowlist")
			return 0
		}
		emit(stdout, strings.Join(lines, "\n")+"\n")
		return 0
	}
	for _, ln := range lines {
		fmt.Fprintln(stderr, ln)
	}
	if err := diskreview.RemoveContents(home, stdout); err != nil {
		fmt.Fprintf(stderr, "disk clean: %v\n", err)
		return 1
	}
	return 0
}
