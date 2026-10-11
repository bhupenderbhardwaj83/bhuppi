package dispatch

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
	"github.com/bhupenderbhardwaj/bhuppi/internal/execx"
)

// Opt is the host view the dispatcher needs. Lookup answers "is this binary here?".
type Opt struct {
	GOOS       string
	DryRun     bool
	Lookup     func(name string) (string, bool)
	HasGit     func(name string) bool
	IsEmptyDir func(string) bool
	UID        string
}

// Plan is one resolved job.
type Plan struct {
	Kind      string // exec, dry, help, missing, refuse, docs, skip, cd, usage, recipe
	Argv      []string
	Exec      []string
	Steps     [][]string
	Execs     [][]string
	Text      string
	Code      int
	StdinNull bool
	Path      []string
	Node      *catalog.Node
	// Draw is "map" or "pmap" when bhuppi renders the tree itself.
	// Limit is how many levels under the root to draw. Note is the dry-run line.
	Draw  string
	Limit int
	Note  string
}

// Resolve turns `group cmd …` into a plan. It does not execute.
func Resolve(cat *catalog.Catalog, args []string, opt Opt) Plan {
	if opt.GOOS == "" {
		opt.GOOS = "linux"
	}
	if len(args) == 0 {
		return Plan{Kind: "help", Code: 0}
	}
	g := cat.Group(args[0])
	if g == nil {
		if e, ok := cat.Explain[args[0]]; ok {
			return Plan{Kind: "refuse", Code: 2, Text: fmt.Sprintf("%s is a Unix name, not a bhuppi group.\n%s\n%s\n", args[0], e.Target, e.Line)}
		}
		return Plan{Kind: "refuse", Code: 2, Text: fmt.Sprintf("unknown job %q\nsee: bhuppi help\n", args[0])}
	}
	r := descend(g, []string{g.Name}, args[1:])
	if r.unknown != "" {
		return Plan{Kind: "usage", Code: 2, Path: r.path, Text: fmt.Sprintf("unknown command %q\nsee: bhuppi help %s\n", r.unknown, strings.Join(r.path, " "))}
	}
	if r.help {
		return Plan{Kind: "help", Code: 0, Path: r.path, Node: r.node}
	}
	n := r.node
	switch n.Delivery {
	case "docs-only":
		return Plan{Kind: "docs", Code: 0, Path: r.path, Node: n, Text: "docs-only — not executed\n"}
	case "skip":
		return Plan{Kind: "skip", Code: 0, Path: r.path, Node: n, Text: "skip — help only, not executed\n"}
	}
	if wrongPlatform(n, opt.GOOS) {
		bin := n.Bin
		if bin == "" {
			bin = n.Name
		}
		return Plan{Kind: "missing", Code: 2, Path: r.path, Node: n, Text: missingText(n, bin, opt.GOOS)}
	}
	if n.MinArgs > 0 && len(r.args) < n.MinArgs {
		return Plan{Kind: "usage", Code: 2, Path: r.path, Node: n, Text: fmt.Sprintf("%s needs %d argument(s)\nsee: bhuppi help %s\n", strings.Join(r.path, " "), n.MinArgs, strings.Join(r.path, " "))}
	}
	argv, err := build(n, r.args, opt)
	if err != nil {
		return Plan{Kind: "refuse", Code: 2, Path: r.path, Node: n, Text: err.Error() + "\n"}
	}
	if n.Special == "cd" {
		if opt.DryRun {
			return Plan{Kind: "dry", Code: 0, Argv: argv, Path: r.path, Node: n}
		}
		return Plan{Kind: "cd", Code: 2, Argv: argv, Path: r.path, Node: n, Text: cdMessage(argv)}
	}
	shown, err := execx.Translate(opt.GOOS, argv, opt.HasGit)
	if err != nil {
		return Plan{Kind: "refuse", Code: 2, Path: r.path, Node: n, Text: err.Error() + "\n"}
	}
	steps := [][]string{shown}
	if n.Special == "stat" && len(shown) > 0 {
		path := shown[len(shown)-1]
		if isDirPath(path) {
			du, derr := execx.Translate(opt.GOOS, []string{"du", "-sh", path}, opt.HasGit)
			if derr != nil {
				return Plan{Kind: "refuse", Code: 2, Path: r.path, Node: n, Text: derr.Error() + "\n"}
			}
			steps = append(steps, du)
		}
	}
	if opt.DryRun {
		p := Plan{Kind: "dry", Code: 0, Argv: shown, Path: r.path, Node: n}
		if len(steps) > 1 {
			p.Steps = steps
		}
		return annotate(p, n, r.args)
	}
	if len(shown) == 0 {
		return Plan{Kind: "refuse", Code: 2, Text: "empty argv\n"}
	}
	execs := make([][]string, 0, len(steps))
	for _, st := range steps {
		if len(st) == 0 {
			return Plan{Kind: "refuse", Code: 2, Text: "empty argv\n"}
		}
		abs, ok := lookup(opt, st[0])
		if !ok {
			return Plan{Kind: "missing", Code: 2, Path: r.path, Node: n, Text: missingText(n, st[0], opt.GOOS)}
		}
		execs = append(execs, append([]string{abs}, st[1:]...))
	}
	p := Plan{
		Kind: "exec", Code: 0, Argv: shown, Exec: execs[0], Path: r.path, Node: n, StdinNull: n.StdinNull,
	}
	if len(execs) > 1 {
		p.Execs = execs
	}
	return annotate(p, n, r.args)
}

func annotate(p Plan, n *catalog.Node, args []string) Plan {
	if n == nil {
		return p
	}
	if n.Special != "map" && n.Special != "pmap" {
		return p
	}
	p.Draw = n.Special
	p.Limit = limitOf(args)
	p.Note = "bhuppi draws the tree; this argv only discovers the entries\n"
	return p
}

func limitOf(args []string) int {
	n := 3
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--depth" && i+1 < len(args) {
			if v, err := strconv.Atoi(args[i+1]); err == nil && v > 0 {
				n = v
			}
		}
		if strings.HasPrefix(a, "--depth=") {
			if v, err := strconv.Atoi(strings.TrimPrefix(a, "--depth=")); err == nil && v > 0 {
				n = v
			}
		}
	}
	return n
}

type walked struct {
	node    *catalog.Node
	path    []string
	args    []string
	help    bool
	unknown string
}

func descend(n *catalog.Node, path, args []string) walked {
	if len(args) > 0 && len(n.Children) > 0 {
		if ch := catalog.Child(n, args[0]); ch != nil {
			return descend(ch, append(path, ch.Name), args[1:])
		}
		if n.Special == "" && isBranch(n) {
			return walked{node: n, path: path, unknown: args[0]}
		}
	}
	if len(args) == 0 && len(n.Children) > 0 && n.Special == "" && isBranch(n) {
		return walked{node: n, path: path, help: true}
	}
	return walked{node: n, path: path, args: args}
}

func isBranch(n *catalog.Node) bool {
	switch n.Delivery {
	case "group", "language-mode", "capability", "meta":
		return true
	default:
		return false
	}
}

func lookup(opt Opt, name string) (string, bool) {
	if opt.Lookup != nil {
		return opt.Lookup(name)
	}
	return execx.Look(name, opt.GOOS, os.Getenv("PATH"))
}

func (opt Opt) present(name string) bool {
	_, ok := lookup(opt, name)
	return ok
}

func missingText(n *catalog.Node, bin, goos string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s is not on PATH.\n", bin)
	switch {
	case n.Platform == "linux" && goos != "linux":
		fmt.Fprintf(&b, "Linux-only on this %s machine.\n", goos)
	case n.Platform == "darwin" && goos != "darwin":
		fmt.Fprintf(&b, "darwin-only on this %s machine.\n", goos)
	}
	if n.Fallback != "" {
		fmt.Fprintf(&b, "See: bhuppi help %s\n", n.Fallback)
	}
	if n.Hint != "" {
		fmt.Fprintf(&b, "%s\n", n.Hint)
	}
	if bin == "find" {
		fmt.Fprintf(&b, "find ships with macOS and with Ubuntu findutils.\n")
	}
	return b.String()
}

func wrongPlatform(n *catalog.Node, goos string) bool {
	switch n.Platform {
	case "linux":
		return goos != "linux"
	case "darwin":
		return goos != "darwin"
	default:
		return false
	}
}

func isDirPath(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func cdMessage(argv []string) string {
	dest := ""
	if len(argv) > 1 {
		dest = argv[1]
	}
	return fmt.Sprintf("a child process cannot cd your shell.\nThe shell hook runs: cd %s\nOpen a new terminal after install, or run: cd %s\n", dest, dest)
}

// RecipePlan builds a recipe pipeline. run is ignored here; the app decides.
func RecipePlan(cat *catalog.Catalog, name string, args []string, opt Opt) Plan {
	r := cat.Recipe(name)
	if r == nil {
		return Plan{Kind: "refuse", Code: 2, Text: fmt.Sprintf("no recipe %q\nsee: bhuppi help recipe\n", name)}
	}
	vars, err := recipeVars(name, args)
	if err != nil {
		return Plan{Kind: "usage", Code: 2, Path: []string{"recipe", name}, Node: r, Text: err.Error() + "\n"}
	}
	steps := r.Steps
	switch opt.GOOS {
	case "linux":
		if len(r.StepsLinux) > 0 {
			steps = r.StepsLinux
		}
	case "darwin":
		if len(r.StepsDarwin) > 0 {
			steps = r.StepsDarwin
		}
	case "windows":
		if len(r.StepsWindows) > 0 {
			steps = r.StepsWindows
		}
	}
	expanded := expandSteps(steps, vars)
	return Plan{Kind: "recipe", Code: 0, Steps: expanded, Path: []string{"recipe", name}, Node: r}
}

func recipeVars(name string, args []string) (map[string]string, error) {
	vars := map[string]string{"dir": ".", "file": "", "archive": ""}
	switch name {
	case "disk-hogs", "newest-in":
		if len(args) > 0 {
			vars["dir"] = args[0]
		}
	case "checksum":
		if len(args) < 1 {
			return nil, fmt.Errorf("checksum needs FILE")
		}
		vars["file"] = args[0]
	case "tar-dir":
		if len(args) < 1 {
			return nil, fmt.Errorf("tar-dir needs DIR")
		}
		vars["dir"] = args[0]
		base := filepath.Base(args[0])
		vars["archive"] = base + ".tar.gz"
	case "who-listens", "cron-mine", "conn-mine":
	default:
		return nil, fmt.Errorf("unknown recipe %s", name)
	}
	return vars, nil
}

func expandSteps(steps [][]string, vars map[string]string) [][]string {
	out := make([][]string, len(steps))
	for i, st := range steps {
		row := make([]string, len(st))
		for j, a := range st {
			row[j] = a
			for k, v := range vars {
				row[j] = strings.ReplaceAll(row[j], "{"+k+"}", v)
			}
		}
		out[i] = row
	}
	return out
}

// RunPipeline runs steps connected by pipes. stdout of the last step goes to w.
func RunPipeline(steps [][]string, stdout, stderr io.Writer) (int, error) {
	if len(steps) == 0 {
		return 2, fmt.Errorf("empty pipeline")
	}
	if len(steps) == 1 {
		cmd := execCommand(steps[0])
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err := cmd.Run()
		return exitCode(err), nil
	}
	cmds := make([]*exec.Cmd, len(steps))
	for i, st := range steps {
		cmds[i] = exec.Command(st[0], st[1:]...)
		cmds[i].Stderr = stderr
	}
	cmds[len(cmds)-1].Stdout = stdout
	writers := make([]io.Closer, len(steps)-1)
	for i := 0; i < len(steps)-1; i++ {
		r, w := io.Pipe()
		cmds[i].Stdout = w
		cmds[i+1].Stdin = r
		writers[i] = w
	}
	for _, c := range cmds {
		if err := c.Start(); err != nil {
			return 1, err
		}
	}
	for i := 0; i < len(steps)-1; i++ {
		i := i
		go func() {
			_ = cmds[i].Wait()
			_ = writers[i].Close()
		}()
	}
	err := cmds[len(cmds)-1].Wait()
	return exitCode(err), nil
}

func execCommand(argv []string) *exec.Cmd {
	return exec.Command(argv[0], argv[1:]...)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return 1
}
