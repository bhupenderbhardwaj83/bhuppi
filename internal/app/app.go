package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/atlas"
	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
	"github.com/bhupenderbhardwaj/bhuppi/internal/clip"
	"github.com/bhupenderbhardwaj/bhuppi/internal/complete"
	"github.com/bhupenderbhardwaj/bhuppi/internal/dispatch"
	"github.com/bhupenderbhardwaj/bhuppi/internal/execx"
	"github.com/bhupenderbhardwaj/bhuppi/internal/help"
	"github.com/bhupenderbhardwaj/bhuppi/internal/pager"
	"github.com/bhupenderbhardwaj/bhuppi/internal/platform"
	"github.com/bhupenderbhardwaj/bhuppi/internal/style"
	"github.com/bhupenderbhardwaj/bhuppi/internal/treeview"
	"github.com/bhupenderbhardwaj/bhuppi/internal/warn"
)

// Version is the bhuppi release stamped into `bhuppi version`.
const Version = "0.1.0"

// Runner executes a prepared argv. Production replaces the process on Unix.
type Runner func(argv []string, stdinNull bool) (int, error)

// Run is the CLI entry. args are os.Args[1:].
func Run(args []string, stdout, stderr io.Writer, runner Runner) int {
	cat, err := catalog.Load()
	if err != nil {
		fmt.Fprintf(stderr, "catalog: %v\n", err)
		return 1
	}
	f, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	if f.PrintCD {
		return printCD(cat, f, stdout)
	}
	opt := help.Options{
		Plain: f.Plain,
		Color: help.ColorEnabled(stdout, f.Plain),
		OS:    f.OS,
	}
	if len(f.Args) == 0 && !f.Help {
		s, err := help.Menu(cat, nil, opt)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 2
		}
		emit(stdout, s)
		return 0
	}
	if len(f.Args) == 0 || (f.Help && len(f.Args) == 0) {
		s, _ := help.Render(cat, nil, opt)
		emit(stdout, s)
		return 0
	}
	if f.Help || positionalHelp(f.Args) {
		path, err := helpPath(cat, f.Args)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 2
		}
		s, err := help.Render(cat, path, opt)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 2
		}
		emit(stdout, s)
		return 0
	}
	switch f.Args[0] {
	case "version":
		fmt.Fprintf(stdout, "bhuppi %s\n", Version)
		return 0
	case "doctor":
		text := platform.Report(cat, platform.OSName(), runtime.GOARCH, nil, nil)
		if opt.Color {
			text = style.ColorDoctor(text)
		}
		emit(stdout, text)
		return 0
	case "explain":
		return explain(cat, f.Args[1:], stdout, stderr)
	case "ai-agent":
		return aiAgentCmd(f.Args[1:], f.Dry, opt, stdout, stderr)
	case "atlas":
		return atlasCmd(f.Args[1:], opt, stdout, stderr)
	case "color":
		return colorCmd(f.Args[1:], stdout, stderr)
	case "config":
		return configCmd(cat, f.Args[1:], opt, stdout, stderr)
	case "completion":
		shell := "zsh"
		if len(f.Args) > 1 {
			shell = f.Args[1]
		}
		emit(stdout, complete.Script(shell))
		return 0
	case "shell-init":
		shell := "zsh"
		if len(f.Args) > 1 {
			shell = f.Args[1]
		}
		emit(stdout, complete.ShellInit(shell))
		return 0
	case "complete":
		emit(stdout, complete.Format(complete.CompleteWith(cat, f.Args[1:], help.Options{Color: style.Allow(f.Plain)})))
		return 0
	case "recipe":
		if len(f.Args) == 1 {
			s, err := help.Render(cat, []string{"recipe"}, opt)
			if err != nil {
				fmt.Fprintf(stderr, "%v\n", err)
				return 2
			}
			emit(stdout, s)
			return 0
		}
		return runRecipe(cat, f.Args[1:], f, stdout, stderr)
	}
	if len(f.Args) >= 2 && f.Args[0] == "disk" && (f.Args[1] == "review" || f.Args[1] == "clean") {
		return diskCmd(f, opt, stdout, stderr)
	}
	dopt := dispatch.Opt{
		GOOS:   runtime.GOOS,
		DryRun: f.Dry,
		UID:    fmt.Sprint(os.Getuid()),
	}
	plan := dispatch.Resolve(cat, f.Args, dopt)
	return finish(cat, plan, opt, stdout, stderr, runner)
}

func finish(cat *catalog.Catalog, plan dispatch.Plan, opt help.Options, stdout, stderr io.Writer, runner Runner) int {
	switch plan.Kind {
	case "help":
		s, err := help.Menu(cat, plan.Path, opt)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 2
		}
		emit(stdout, s)
		return 0
	case "docs", "skip":
		s, err := help.Render(cat, plan.Path, opt)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 2
		}
		emit(stdout, s)
		if plan.Text != "" {
			fmt.Fprint(stderr, plan.Text)
		}
		return 0
	case "usage":
		fmt.Fprint(stderr, plan.Text)
		if len(plan.Path) > 0 {
			s, err := help.Render(cat, plan.Path, opt)
			if err == nil {
				emit(stdout, s)
			}
		}
		return 2
	case "dry":
		var b strings.Builder
		if len(plan.Steps) > 1 {
			for _, st := range plan.Steps {
				fmt.Fprintln(&b, warn.Show(dispatch.FormatArgv(st), opt.Color))
			}
		} else {
			fmt.Fprintln(&b, warn.Show(dispatch.FormatArgv(plan.Argv), opt.Color))
		}
		if plan.Note != "" {
			b.WriteString(plan.Note)
		}
		emit(stdout, b.String())
		return 0
	case "cd", "missing", "refuse":
		text := plan.Text
		if opt.Color && plan.Kind == "missing" {
			text = style.ColorMissing(text)
		}
		fmt.Fprint(stderr, text)
		return plan.Code
	case "exec":
		if plan.Draw == "pview" || plan.Draw == "pagents" || plan.Draw == "procs" || plan.Draw == "conns" {
			return renderProc(plan, stdout, stderr, opt.Color)
		}
		if plan.Draw != "" {
			return drawTree(plan, stdout, stderr, opt.Color)
		}
		if len(plan.Execs) > 1 {
			return runExecs(plan.Execs, plan.StdinNull, stdout, stderr, runner, opt.Color, plan.Path)
		}
		if runner == nil && !execx.NeedsTTY(plan.Exec[0], pager.IsTerminal(stdout)) {
			if opt.Color && style.ColorableJob(plan.Path) {
				var buf bytes.Buffer
				code, err := execx.RunCaptured(plan.Exec, plan.StdinNull, &buf, stderr)
				emit(stdout, style.ColorJobIn(plan.Path, buf.String(), style.ListRoot(plan.Exec)))
				if err != nil {
					fmt.Fprintf(stderr, "%v\n", err)
					if code == 0 {
						return 1
					}
				}
				return code
			}
			code, err := execx.RunCaptured(plan.Exec, plan.StdinNull, stdout, stderr)
			if err != nil {
				fmt.Fprintf(stderr, "%v\n", err)
				if code == 0 {
					return 1
				}
			}
			return code
		}
		if runner == nil {
			runner = defaultRunner
		}
		code, err := runner(plan.Exec, plan.StdinNull)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			if code == 0 {
				return 1
			}
		}
		return code
	default:
		fmt.Fprintf(stderr, "unhandled plan %s\n", plan.Kind)
		return 2
	}
}

func runRecipe(cat *catalog.Catalog, args []string, f parsed, stdout, stderr io.Writer) int {
	opt := help.Options{Plain: f.Plain, Color: help.ColorEnabled(stdout, f.Plain), OS: f.OS}
	if f.Help {
		path := []string{"recipe"}
		if len(args) > 0 {
			path = append(path, args[0])
		}
		s, err := help.Render(cat, path, opt)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 2
		}
		emit(stdout, s)
		return 0
	}
	plan := dispatch.RecipePlan(cat, args[0], args[1:], dispatch.Opt{GOOS: runtime.GOOS})
	if plan.Kind != "recipe" {
		fmt.Fprint(stderr, plan.Text)
		return plan.Code
	}
	emit(stdout, dispatch.FormatPipeline(plan.Steps)+"\n")
	if !f.Run || f.Dry {
		return 0
	}
	out := stdout
	var captured bytes.Buffer
	if pager.IsTerminal(stdout) {
		out = &captured
	}
	code, err := dispatch.RunPipeline(plan.Steps, out, stderr)
	if pager.IsTerminal(stdout) {
		emit(stdout, captured.String())
	}
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	return code
}

func explain(cat *catalog.Catalog, args []string, stdout, stderr io.Writer) int {
	var b strings.Builder
	if len(args) == 0 {
		fmt.Fprintln(&b, "bhuppi explain NAME")
		fmt.Fprintln(&b, "Unix name → job. Skips and docs-only names say so.")
		for _, name := range cat.ExplainNames() {
			e := cat.Explain[name]
			fmt.Fprintf(&b, "%-16s %s\n", name, e.Target)
		}
		emit(stdout, b.String())
		return 0
	}
	e, ok := cat.Explain[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "no explain entry for %s\nsee: bhuppi explain\n", args[0])
		return 2
	}
	fmt.Fprintln(&b, e.Target)
	if e.Line != "" {
		fmt.Fprintln(&b, e.Line)
	} else if e.What != "" {
		fmt.Fprintln(&b, e.What)
	}
	if e.Example != "" {
		fmt.Fprintf(&b, "e.g. %s\n", e.Example)
	}
	if atlas.Has(args[0]) {
		fmt.Fprintf(&b, "guide: bhuppi atlas %s\n", args[0])
	}
	switch e.Delivery {
	case "docs-only":
		fmt.Fprintln(&b, "docs-only — not executed")
		if e.Docs != "" {
			fmt.Fprintf(&b, "docs: %s\n", e.Docs)
		}
	case "skip":
		fmt.Fprintln(&b, "skip — help only, not executed")
		if e.Docs != "" {
			fmt.Fprintf(&b, "docs: %s\n", e.Docs)
		}
	}
	emit(stdout, b.String())
	return 0
}

// emit pages when stdout is a terminal. --plain does not turn this off.
func drawTree(plan dispatch.Plan, stdout, stderr io.Writer, color bool) int {
	var buf bytes.Buffer
	code, err := execx.RunCaptured(plan.Exec, plan.StdinNull, &buf, stderr)
	limit := plan.Limit
	if limit < 1 {
		limit = 3
	}
	var text string
	if plan.Draw == "pmap" {
		text = treeview.ProcTree(treeview.ParsePS(buf.String()), limit, color)
	} else {
		root := "."
		if len(plan.Argv) >= 2 {
			root = plan.Argv[1]
		}
		text = treeview.Dir(root, buf.String(), limit, color)
	}
	if text != "" {
		emit(stdout, text)
	}
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		if code == 0 {
			return 1
		}
	}
	return code
}

func emit(w io.Writer, s string) {
	emitOpt(w, s, pager.Options{})
}

func emitOpt(w io.Writer, s string, opt pager.Options) {
	if err := pager.WriteWith(w, s, opt); err != nil && !errors.Is(err, pager.ErrQuit) {
		fmt.Fprint(w, s)
	}
}

func positionalHelp(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if args[0] == "help" || args[0] == "h" {
		return true
	}
	for _, a := range args[1:] {
		if a == "help" {
			return true
		}
	}
	return false
}

func helpPath(cat *catalog.Catalog, args []string) ([]string, error) {
	if len(args) > 0 && (args[0] == "help" || args[0] == "h") {
		args = args[1:]
	}
	filtered := make([]string, 0, len(args))
	for _, a := range args {
		if a == "help" {
			continue
		}
		filtered = append(filtered, a)
	}
	args = filtered
	if len(args) == 0 {
		return nil, nil
	}
	if args[0] == "recipe" {
		if len(args) == 1 {
			return []string{"recipe"}, nil
		}
		if cat.Recipe(args[1]) == nil {
			return nil, fmt.Errorf("no recipe %q\nsee: bhuppi help recipe", args[1])
		}
		return []string{"recipe", args[1]}, nil
	}
	if m := cat.Meta(args[0]); m != nil {
		path := []string{args[0]}
		n := m
		for _, part := range args[1:] {
			ch := catalog.Child(n, part)
			if ch == nil {
				break
			}
			path = append(path, part)
			n = ch
		}
		return path, nil
	}
	g := cat.Group(args[0])
	if g == nil {
		return nil, fmt.Errorf("no help page for %q\nsee: bhuppi help", args[0])
	}
	path := []string{args[0]}
	n := g
	for _, part := range args[1:] {
		ch := catalog.Child(n, part)
		if ch == nil {
			break
		}
		path = append(path, part)
		n = ch
	}
	return path, nil
}

type parsed struct {
	Dry     bool
	Plain   bool
	Help    bool
	Run     bool
	PrintCD bool
	OS      string
	Args    []string
}

func parseArgs(args []string) (parsed, error) {
	var f parsed
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			f.Args = append(f.Args, args[i+1:]...)
			break
		}
		switch {
		case a == "--dry-run":
			f.Dry = true
		case a == "--print-cd":
			f.PrintCD = true
		case a == "--plain":
			f.Plain = true
		case a == "--run":
			f.Run = true
		case a == "-h" || a == "--help":
			f.Help = true
		case a == "--version":
			f.Args = append(f.Args, "version")
		case a == "--os":
			i++
			if i >= len(args) {
				return f, fmt.Errorf("--os needs linux or darwin")
			}
			f.OS = args[i]
		case strings.HasPrefix(a, "--os="):
			f.OS = strings.TrimPrefix(a, "--os=")
		default:
			f.Args = append(f.Args, a)
		}
	}
	switch f.OS {
	case "", "linux", "darwin", "windows":
	default:
		return f, fmt.Errorf("--os is linux or darwin")
	}
	return f, nil
}

// printCD is the shell-hook probe. It prints one `cd PATH` line for files go
// and folders go, and nothing for every other command, including --dry-run.
func printCD(cat *catalog.Catalog, f parsed, stdout io.Writer) int {
	if f.Dry || f.Help || len(f.Args) == 0 {
		return 0
	}
	switch f.Args[0] {
	case "help", "h", "version", "doctor", "explain", "completion", "shell-init", "complete", "recipe":
		return 0
	}
	plan := dispatch.Resolve(cat, f.Args, dispatch.Opt{
		GOOS: runtime.GOOS,
		UID:  fmt.Sprint(os.Getuid()),
	})
	if plan.Kind == "cd" && len(plan.Argv) >= 2 {
		fmt.Fprintf(stdout, "cd %s\n", plan.Argv[1])
	}
	return 0
}

func atlasCmd(args []string, opt help.Options, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		emit(stdout, atlas.RenderIndex(opt.Color))
		return 0
	}
	section := ""
	if len(args) > 1 {
		section = strings.Join(args[1:], " ")
	}
	text, ok := atlas.Render(args[0], section, opt.Color)
	if !ok {
		fmt.Fprint(stderr, text)
		return 2
	}
	if section != "" {
		emit(stdout, text)
		return 0
	}
	commands := atlas.CaseCommands(args[0])
	emitOpt(stdout, text, pager.Options{
		Cases: len(commands),
		Hint:  pager.AtlasHint,
		Copy: func(n int) (string, bool) {
			cmd, ok := atlas.Pick(commands, n)
			if !ok {
				return "", false
			}
			if err := clip.Copy(cmd, nil); err != nil {
				return "clipboard tool missing", true
			}
			return copyConfirm(cmd, opt.Color), true
		},
	})
	return 0
}

func copyConfirm(cmd string, color bool) string {
	if on, _ := warn.Mark(cmd); on {
		line := "! copied: " + cmd
		if color {
			return style.Paint(style.Hidden, line, style.Auto, false)
		}
		return line
	}
	return "copied: " + cmd
}

func colorCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, style.Status())
		return 0
	}
	switch args[0] {
	case "on", "off":
		if err := style.Set(args[0] == "on"); err != nil {
			fmt.Fprintf(stderr, "color: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "color %s\n", args[0])
		return 0
	default:
		fmt.Fprintf(stderr, "color is on or off\nsee: bhuppi color\n")
		return 2
	}
}

func runExecs(execs [][]string, stdinNull bool, stdout, stderr io.Writer, runner Runner, color bool, path []string) int {
	if runner != nil {
		code := 0
		for _, argv := range execs {
			c, err := runner(argv, stdinNull)
			if err != nil {
				fmt.Fprintf(stderr, "%v\n", err)
				if c == 0 {
					c = 1
				}
			}
			if c != 0 && code == 0 {
				code = c
			}
		}
		return code
	}
	out := stdout
	var buf bytes.Buffer
	page := pager.IsTerminal(stdout)
	if page {
		out = &buf
	}
	code := 0
	for _, argv := range execs {
		c, err := execx.RunCaptured(argv, stdinNull, out, stderr)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			if c == 0 {
				c = 1
			}
		}
		if c != 0 && code == 0 {
			code = c
		}
	}
	if page {
		text := buf.String()
		if color {
			root := ""
			if len(execs) > 0 {
				root = style.ListRoot(execs[0])
			}
			text = style.ColorJobIn(path, text, root)
		}
		emit(stdout, text)
	}
	return code
}

func defaultRunner(argv []string, stdinNull bool) (int, error) {
	err := execx.Exec(argv, stdinNull)
	if err == nil {
		return 0, nil
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), nil
	}
	return 1, err
}
