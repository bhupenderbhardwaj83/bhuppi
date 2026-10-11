package catalog

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data.yaml
var dataYAML []byte

// Node is one catalog page: a group, a command, or a language-mode child.
type Node struct {
	Name         string     `yaml:"name"`
	Title        string     `yaml:"title"`
	Shape        string     `yaml:"shape"`
	Example      string     `yaml:"example"`
	Examples     []string   `yaml:"examples"`
	What         string     `yaml:"what"`
	When         string     `yaml:"when"`
	Rule         string     `yaml:"rule"`
	Desc         string     `yaml:"desc"`
	NotThis      string     `yaml:"not_this"`
	More         string     `yaml:"more"`
	AlsoLinux    string     `yaml:"also_linux"`
	AlsoMacos    string     `yaml:"also_macos"`
	Sparkle      bool       `yaml:"sparkle"`
	Pick         string     `yaml:"pick"`
	Delivery     string     `yaml:"delivery"`
	Bin          string     `yaml:"bin"`
	Prefer       []string   `yaml:"prefer"`
	Platform     string     `yaml:"platform"`
	Hint         string     `yaml:"hint"`
	Docs         string     `yaml:"docs"`
	Special      string     `yaml:"special"`
	Unix         []string   `yaml:"unix"`
	Children     []Node     `yaml:"children"`
	Fallback     string     `yaml:"fallback"`
	MinArgs      int        `yaml:"min_args"`
	Complete     string     `yaml:"complete"`
	CompleteSkip int        `yaml:"complete_skip"`
	Section      string     `yaml:"section"`
	Argv         []string   `yaml:"argv"`
	ArgvLinux    []string   `yaml:"argv_linux"`
	ArgvDarwin   []string   `yaml:"argv_darwin"`
	ArgvWindows  []string   `yaml:"argv_windows"`
	UserArgs     string     `yaml:"user_args"`
	Runs         string     `yaml:"runs"`
	RunsLinux    string     `yaml:"runs_linux"`
	RunsDarwin   string     `yaml:"runs_darwin"`
	RunsWindows  string     `yaml:"runs_windows"`
	Optional     bool       `yaml:"optional"`
	StdinNull    bool       `yaml:"stdin_null"`
	ExplainLine  string     `yaml:"explain_line"`
	Steps        [][]string `yaml:"steps"`
	StepsLinux   [][]string `yaml:"steps_linux"`
	StepsDarwin  [][]string `yaml:"steps_darwin"`
	StepsWindows [][]string `yaml:"steps_windows"`
}

// Entry is one Unix name in the explain index.
type Entry struct {
	Name     string
	Target   string
	Line     string
	Delivery string
	Example  string
	What     string
	When     string
	Rule     string
	NotThis  string
	Docs     string
}

type explainExtra struct {
	Name     string `yaml:"name"`
	Target   string `yaml:"target"`
	Line     string `yaml:"line"`
	Delivery string `yaml:"delivery"`
	Example  string `yaml:"example"`
	What     string `yaml:"what"`
	When     string `yaml:"when"`
	Rule     string `yaml:"rule"`
	NotThis  string `yaml:"not_this"`
	Docs     string `yaml:"docs"`
}

// Identity is the overview contract.
type Identity struct {
	Name     string `yaml:"name"`
	Wordmark string `yaml:"wordmark"`
	Tagline  string `yaml:"tagline"`
	Rule     string `yaml:"rule"`
	What     string `yaml:"what"`
	When     string `yaml:"when"`
	NotThis  string `yaml:"not_this"`
	More     string `yaml:"more"`
}

// Catalog is the embedded job tree.
type Catalog struct {
	Identity  Identity       `yaml:"identity"`
	Groups    []Node         `yaml:"groups"`
	MetaPages []Node         `yaml:"meta"`
	Recipes   []Node         `yaml:"recipes"`
	Extras    []explainExtra `yaml:"explain"`

	byPath  map[string]*Node `yaml:"-"`
	Explain map[string]Entry `yaml:"-"`
}

// Load reads the embedded YAML and indexes explain names.
func Load() (*Catalog, error) {
	var c Catalog
	if err := yaml.Unmarshal(dataYAML, &c); err != nil {
		return nil, err
	}
	if err := c.compile(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Catalog) compile() error {
	want := []string{"files", "folders", "text", "lang", "proc", "net", "pack", "disk", "auth", "svc", "logs", "host", "data", "dev", "docs"}
	if len(c.Groups) != len(want) {
		return fmt.Errorf("catalog has %d groups, want %d", len(c.Groups), len(want))
	}
	for i, name := range want {
		if c.Groups[i].Name != name {
			return fmt.Errorf("group %d is %s, want %s", i, c.Groups[i].Name, name)
		}
	}
	c.byPath = map[string]*Node{}
	c.Explain = map[string]Entry{}
	var walk func(n *Node, path string) error
	walk = func(n *Node, path string) error {
		for _, key := range []struct{ k, v string }{
			{"what", n.What}, {"when", n.When}, {"rule", n.Rule},
			{"desc", n.Desc}, {"not_this", n.NotThis}, {"shape", n.Shape},
			{"example", n.Example}, {"more", n.More},
		} {
			if strings.TrimSpace(key.v) == "" {
				return fmt.Errorf("%s missing %s", path, key.k)
			}
		}
		if n.Name == "delete" {
			if strings.HasPrefix(path, "files") || strings.HasPrefix(path, "folders") {
				return fmt.Errorf("banned verb %s at %s", n.Name, path)
			}
		}
		c.byPath[path] = n
		line := n.ExplainLine
		if line == "" {
			line = n.What
		}
		for _, u := range n.Unix {
			if prev, ok := c.Explain[u]; ok && prev.Target != path {
				return fmt.Errorf("duplicate explain %s (%s vs %s)", u, prev.Target, path)
			}
			c.Explain[u] = Entry{
				Name: u, Target: path, Line: line, Delivery: n.Delivery,
				Example: n.Example, What: n.What, When: n.When, Rule: n.Rule,
				NotThis: n.NotThis, Docs: n.Docs,
			}
		}
		for i := range n.Children {
			ch := &n.Children[i]
			if err := walk(ch, path+" "+ch.Name); err != nil {
				return err
			}
		}
		return nil
	}
	for i := range c.Groups {
		g := &c.Groups[i]
		if err := walk(g, g.Name); err != nil {
			return err
		}
	}
	for i := range c.MetaPages {
		m := &c.MetaPages[i]
		c.byPath["meta "+m.Name] = m
	}
	for i := range c.Recipes {
		r := &c.Recipes[i]
		path := "recipe " + r.Name
		if err := walk(r, path); err != nil {
			return err
		}
	}
	for _, ex := range c.Extras {
		if _, ok := c.Explain[ex.Name]; ok {
			continue
		}
		target := ex.Target
		if target == "" {
			target = "skip"
		}
		c.Explain[ex.Name] = Entry{
			Name: ex.Name, Target: target, Line: ex.Line, Delivery: ex.Delivery,
			Example: ex.Example, What: ex.What, When: ex.When, Rule: ex.Rule,
			NotThis: ex.NotThis, Docs: ex.Docs,
		}
	}
	for i := range c.Groups {
		g := &c.Groups[i]
		if _, ok := c.Explain[g.Name]; ok {
			continue
		}
		c.Explain[g.Name] = Entry{
			Name: g.Name, Target: g.Name, Line: g.What, Delivery: "group",
			Example: "bhuppi help " + g.Name, What: g.What, When: g.When, Rule: g.Rule, NotThis: g.NotThis,
		}
	}
	if len(c.Explain) < 164 {
		return fmt.Errorf("explain index has %d names, want at least 164", len(c.Explain))
	}
	return nil
}

// Group returns a top-level job group.
func (c *Catalog) Group(name string) *Node {
	for i := range c.Groups {
		if c.Groups[i].Name == name {
			return &c.Groups[i]
		}
	}
	return nil
}

// Meta returns a meta page (help, doctor, explain, …).
func (c *Catalog) Meta(name string) *Node {
	for i := range c.MetaPages {
		if c.MetaPages[i].Name == name {
			return &c.MetaPages[i]
		}
	}
	return nil
}

// Recipe returns a named recipe.
func (c *Catalog) Recipe(name string) *Node {
	for i := range c.Recipes {
		if c.Recipes[i].Name == name {
			return &c.Recipes[i]
		}
	}
	return nil
}

// At returns the node at a space-separated path such as "text awk fields".
func (c *Catalog) At(path string) *Node {
	return c.byPath[path]
}

// ExplainNames returns sorted Unix names.
func (c *Catalog) ExplainNames() []string {
	out := make([]string, 0, len(c.Explain))
	for name := range c.Explain {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Child finds a direct child by name.
func Child(n *Node, name string) *Node {
	if n == nil {
		return nil
	}
	for i := range n.Children {
		if n.Children[i].Name == name {
			return &n.Children[i]
		}
	}
	return nil
}

// Tool is one binary the doctor table reports.
type Tool struct {
	Name     string
	Platform string
	Fallback string
	Hint     string
	Optional bool
}

// Tools lists binaries referenced by the catalog, de-duplicated, stable order.
func (c *Catalog) Tools() []Tool {
	seen := map[string]bool{}
	var out []Tool
	add := func(name, platform, fallback, hint string, optional bool) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, Tool{Name: name, Platform: platform, Fallback: fallback, Hint: hint, Optional: optional})
	}
	var walk func(n *Node)
	walk = func(n *Node) {
		if len(n.Prefer) > 0 {
			for _, p := range n.Prefer {
				add(p, n.Platform, n.Fallback, n.Hint, true)
			}
		}
		add(n.Bin, n.Platform, n.Fallback, n.Hint, n.Optional)
		for i := range n.Children {
			walk(&n.Children[i])
		}
	}
	for i := range c.Groups {
		walk(&c.Groups[i])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
