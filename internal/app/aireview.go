package app

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/aireview"
	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
	"github.com/bhupenderbhardwaj/bhuppi/internal/dispatch"
	"github.com/bhupenderbhardwaj/bhuppi/internal/help"
	"github.com/bhupenderbhardwaj/bhuppi/internal/procview"
)

func aiAgentCmd(args []string, dry bool, opt help.Options, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		cat, err := catalog.Load()
		if err != nil {
			fmt.Fprintf(stderr, "catalog: %v\n", err)
			return 1
		}
		s, err := help.Menu(cat, []string{"ai-agent"}, opt)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 2
		}
		emit(stdout, s)
		return 0
	}
	if args[0] != "review" {
		fmt.Fprintf(stderr, "unknown command %q\nsee: bhuppi help ai-agent\n", args[0])
		return 2
	}
	if len(args) > 2 || (len(args) == 2 && strings.HasPrefix(args[1], "-")) {
		fmt.Fprintf(stderr, "ai-agent review takes one name or pid\nsee: bhuppi help ai-agent review\n")
		return 2
	}
	query := ""
	if len(args) == 2 {
		query = args[1]
	}
	if dry {
		for _, st := range aireview.Argv(runtime.GOOS, query) {
			fmt.Fprintln(stdout, dispatch.FormatArgv(st))
		}
		return 0
	}
	return runAIReview(query, opt.Color, stdout, stderr)
}

func runAIReview(query string, color bool, stdout, stderr io.Writer) int {
	ps, err := absolute(aireview.Argv(runtime.GOOS, "")[0])
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	text, _, err := capture(ps)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	table := dropSelfSnaps(aireview.ParseSnaps(text), fmt.Sprint(os.Getpid()))
	cronText, _ := runQuiet([]string{"crontab", "-l"})
	launchText := ""
	if runtime.GOOS != "linux" {
		launchText, _ = runQuiet([]string{"launchctl", "list"})
	}
	if isPID(query) {
		var hit *aireview.Snap
		for i := range table {
			if table[i].PID == query {
				hit = &table[i]
				break
			}
		}
		if hit == nil {
			fmt.Fprintf(stderr, "no such process\n")
			return 2
		}
		emit(stdout, buildReport("pid "+query, namesOfSnap(*hit), []aireview.Snap{*hit}, aireview.FamilyPIDs(table, hit.PID), table, cronText, launchText, color))
		return 0
	}
	names := procview.AgentNames
	if query != "" {
		names = []string{query}
	}
	var blocks []string
	for _, name := range names {
		matched := aireview.MatchSnaps(table, name)
		if len(matched) == 0 {
			continue
		}
		ids := make([]string, len(matched))
		for i, s := range matched {
			ids[i] = s.PID
		}
		blocks = append(blocks, buildReport(name, []string{name}, matched, ids, table, cronText, launchText, color))
	}
	if len(blocks) == 0 {
		if query != "" {
			fmt.Fprintf(stderr, "no matches\n")
			return 2
		}
		emit(stdout, aireview.None())
		return 0
	}
	emit(stdout, strings.Join(blocks, "\n"))
	return 0
}

func namesOfSnap(s aireview.Snap) []string {
	var out []string
	low := strings.ToLower(s.Command)
	for _, n := range procview.AgentNames {
		if strings.Contains(low, n) {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		base := filepath.Base(procview.ProgramPath(s.Command))
		if base != "" && base != "." {
			out = append(out, base)
		}
	}
	return out
}

func dropSelfSnaps(rows []aireview.Snap, self string) []aireview.Snap {
	var out []aireview.Snap
	for _, r := range rows {
		if r.PID == self || strings.Contains(r.Command, "ai-agent review") {
			continue
		}
		out = append(out, r)
	}
	return out
}

func buildReport(heading string, names []string, matched []aireview.Snap, pids []string, table []aireview.Snap, cronText, launchText string, color bool) string {
	opens := aireview.ParseOpen(lsofBatched(pids, false))
	var socks []aireview.Sock
	for _, o := range aireview.ParseOpen(lsofBatched(pids, true)) {
		socks = append(socks, aireview.ParseSock(o.Name))
	}
	in, out := aireview.SplitDirection(socks)
	files := map[string]aireview.Files{}
	sens := map[string]bool{}
	var sensitive []string
	addSens := func(p string) {
		if p == "" || !aireview.SensitivePath(p) || sens[p] {
			return
		}
		sens[p] = true
		sensitive = append(sensitive, p)
	}
	for _, o := range opens {
		f := files[o.PID]
		path := procview.PathOf(o.Name)
		if o.FD == "txt" && f.Program == "" {
			if path != "" {
				f.Program = path
			} else {
				f.Program = strings.TrimSpace(o.Name)
			}
		}
		if o.FD == "cwd" && f.Cwd == "" {
			if path != "" {
				f.Cwd = path
			} else {
				f.Cwd = strings.TrimSpace(o.Name)
			}
		}
		files[o.PID] = f
		addSens(path)
	}
	var programs []string
	for _, p := range matched {
		prog := files[p.PID].Program
		if prog == "" {
			prog = procview.ProgramPath(p.Command)
		}
		if prog != "" {
			programs = append(programs, prog)
		}
	}
	var persist []aireview.Persist
	for _, item := range readPersistence(names, programs) {
		for _, s := range item.paths {
			addSens(s)
		}
		persist = append(persist, item.row)
	}
	var cron []string
	var jobs []aireview.Job
	seenCron := map[string]bool{}
	seenJob := map[string]bool{}
	for _, name := range names {
		prog := ""
		if len(programs) > 0 {
			prog = programs[0]
		}
		for _, line := range aireview.FilterCron(cronText, name, prog) {
			if !seenCron[line] {
				seenCron[line] = true
				cron = append(cron, line)
			}
		}
		for _, j := range aireview.FilterLaunch(launchText, name) {
			if !seenJob[j.Label] {
				seenJob[j.Label] = true
				jobs = append(jobs, j)
			}
		}
	}
	return aireview.Format(aireview.Report{
		Heading: heading, Blocks: aireview.Blocks(table, matched, files),
		Inbound: in, Outbound: out,
		Sensitive: sensitive, Persist: persist, Cron: cron, Launch: jobs,
	}, color)
}

type persistHit struct {
	row   aireview.Persist
	paths []string
}

func readPersistence(names, programs []string) []persistHit {
	var hits []persistHit
	if runtime.GOOS == "linux" {
		return systemdHits(names, programs)
	}
	for _, path := range plistPaths() {
		p, ok := loadPlist(path)
		if !ok || !plistHit(p, names, programs) {
			continue
		}
		prog := p.Program
		if prog == "" && len(p.Args) > 0 {
			prog = p.Args[0]
		}
		var sens []string
		rowSens := false
		for _, s := range p.Strings() {
			if aireview.SensitivePath(s) {
				sens = append(sens, s)
				rowSens = true
			}
		}
		hits = append(hits, persistHit{
			row:   aireview.Persist{Label: p.Label, Program: prog, RunAtLoad: p.RunAtLoad, Sensitive: rowSens},
			paths: sens,
		})
	}
	return hits
}

func plistHit(p aireview.Plist, names, programs []string) bool {
	for _, name := range names {
		if p.Hit(name, "") {
			return true
		}
	}
	for _, prog := range programs {
		if prog != "" && p.Hit("", prog) {
			return true
		}
	}
	return false
}

func loadPlist(path string) (aireview.Plist, bool) {
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 || len(b) > 1<<20 {
		return aireview.Plist{}, false
	}
	if bytes.HasPrefix(b, []byte("bplist")) {
		out, err := exec.Command("plutil", "-convert", "xml1", "-o", "-", path).Output()
		if err != nil {
			return aireview.Plist{}, false
		}
		b = out
	}
	return aireview.ParsePlist(string(b))
}

func plistPaths() []string {
	var out []string
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		matches, _ := filepath.Glob(filepath.Join(home, "Library", "LaunchAgents", "*.plist"))
		out = append(out, matches...)
	}
	ents, err := os.ReadDir("/Library/LaunchDaemons")
	if err != nil {
		return out
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".plist") {
			continue
		}
		p := filepath.Join("/Library/LaunchDaemons", e.Name())
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		f.Close()
		out = append(out, p)
	}
	return out
}

func systemdHits(names, programs []string) []persistHit {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	dir := filepath.Join(home, ".config", "systemd", "user")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var hits []persistHit
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".service") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil || len(b) > 1<<20 {
			continue
		}
		text := string(b)
		execStart := execStartOf(text)
		joined := e.Name() + "\n" + execStart
		if !anyMention(joined, names, programs) {
			continue
		}
		var sens []string
		rowSens := false
		for _, field := range strings.Fields(execStart) {
			if aireview.SensitivePath(field) {
				sens = append(sens, field)
				rowSens = true
			}
		}
		hits = append(hits, persistHit{
			row: aireview.Persist{
				Label:     e.Name(),
				Program:   execStart,
				RunAtLoad: strings.Contains(text, "WantedBy="),
				Sensitive: rowSens,
			},
			paths: sens,
		})
	}
	return hits
}

func anyMention(text string, names, programs []string) bool {
	for _, n := range names {
		if aireview.Mentions(text, n, "") {
			return true
		}
	}
	for _, p := range programs {
		if aireview.Mentions(text, "", p) {
			return true
		}
	}
	return false
}

func execStartOf(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trim := strings.TrimSpace(line)
		if len(trim) >= 10 && strings.EqualFold(trim[:10], "execstart=") {
			return strings.TrimSpace(trim[10:])
		}
	}
	return ""
}

func lsofBatched(pids []string, net bool) string {
	var b strings.Builder
	for _, argv := range aireview.LsofArgv(pids, net) {
		text, _ := runQuiet(argv)
		b.WriteString(text)
		if text != "" && !strings.HasSuffix(text, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func runQuiet(argv []string) (string, error) {
	abs, err := absolute(argv)
	if err != nil {
		return "", err
	}
	text, _, err := capture(abs)
	return text, err
}
