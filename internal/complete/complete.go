package complete

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
	"github.com/bhupenderbhardwaj/bhuppi/internal/help"
	"github.com/bhupenderbhardwaj/bhuppi/internal/warn"
)

// Item is one completion candidate. Kind is command, file, or folder.
// Section groups command names in the menu (everyday, look, careful, …).
type Item struct {
	Kind    string
	Value   string
	Desc    string
	Section string
}

// Complete returns three visual groups: commands, files, folders.
// words are the tokens after `bhuppi`. The last token is the prefix being typed.
func Complete(cat *catalog.Catalog, words []string) []Item {
	return CompleteWith(cat, words, help.Options{})
}

// CompleteWith is Complete with menu coloring. Color is for the display only.
func CompleteWith(cat *catalog.Catalog, words []string, opt help.Options) []Item {
	prefix := ""
	prev := words
	if len(words) > 0 {
		prefix = words[len(words)-1]
		prev = words[:len(words)-1]
	}
	if len(prev) == 0 {
		return filterPrefix(rootCommands(cat, opt), prefix)
	}
	if prev[0] == "recipe" {
		if len(prev) == 1 {
			var items []Item
			for i := range cat.Recipes {
				r := &cat.Recipes[i]
				items = append(items, commandItem(r.Name, help.Brief(r), help.SectionOf("recipe", r.Name, false)))
			}
			return filterPrefix(items, prefix)
		}
		return nil
	}
	if prev[0] == "explain" && len(prev) == 1 {
		var items []Item
		for _, name := range cat.ExplainNames() {
			items = append(items, commandItem(name, cat.Explain[name].Target, ""))
		}
		return filterPrefix(items, prefix)
	}
	if prev[0] != "help" && prev[0] != "h" {
		if m := cat.Meta(prev[0]); m != nil {
			if len(prev) == 1 && len(m.Children) > 0 {
				items := rowsToItems(help.CompletionRows(prev[0], m.Children, false, opt))
				return filterPrefix(items, prefix)
			}
			return nil
		}
	}
	args := prev
	if prev[0] == "help" || prev[0] == "h" {
		args = prev[1:]
		if len(args) == 0 {
			return filterPrefix(rootCommands(cat, opt), prefix)
		}
	}
	g := cat.Group(args[0])
	if g == nil {
		return filterPrefix(rootCommands(cat, opt), prefix)
	}
	node := g
	rest := args[1:]
	for len(rest) > 0 {
		ch := catalog.Child(node, rest[0])
		if ch == nil {
			break
		}
		node = ch
		rest = rest[1:]
	}
	var items []Item
	if len(node.Children) > 0 && len(rest) == 0 {
		items = orderSections(node.Name, rowsToItems(help.CompletionRows(node.Name, node.Children, false, opt)))
		if node.Special == "" {
			return filterPrefix(items, prefix)
		}
	}
	if len(rest) < node.CompleteSkip {
		return filterPrefix(items, prefix)
	}
	items = append(items, paths(node.Complete, prefix)...)
	return items
}

func isMeta(cat *catalog.Catalog, name string) bool {
	return cat.Meta(name) != nil
}

func rootCommands(cat *catalog.Catalog, opt help.Options) []Item {
	nodes := make([]catalog.Node, 0, len(cat.Groups)+len(cat.MetaPages))
	for i := range cat.Groups {
		nodes = append(nodes, cat.Groups[i])
	}
	for i := range cat.MetaPages {
		nodes = append(nodes, cat.MetaPages[i])
	}
	return orderSections("", rowsToItems(help.CompletionRows("", nodes, true, opt)))
}

func rowsToItems(rows []help.CompletionRow) []Item {
	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		sec := row.Section
		if sec == "" {
			sec = "commands"
		}
		items = append(items, Item{Kind: "command", Value: row.Name, Desc: row.Display, Section: sec})
	}
	return items
}

func commandItem(name, desc, section string) Item {
	if on, reason := warn.Name(name); on && desc != "" && !strings.Contains(desc, reason) {
		desc = desc + " — " + reason
	}
	if section == "" {
		section = "commands"
	}
	return Item{Kind: "command", Value: name, Desc: desc, Section: section}
}

func orderSections(parent string, items []Item) []Item {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != "command" || items[j].Kind != "command" {
			return false
		}
		si, sj := sectionRank(parent, items[i].Section), sectionRank(parent, items[j].Section)
		if si != sj {
			return si < sj
		}
		return nameRank(parent, items[i].Value) < nameRank(parent, items[j].Value)
	})
	return items
}

func sectionRank(parent, section string) int {
	order := []string{"everyday", "machine", "build", "guide", "setup", "look", "change", "commands", "careful"}
	if parent != "" && parent != "files" && parent != "proc" {
		order = []string{"commands", "careful"}
	}
	for i, name := range order {
		if name == section {
			return i
		}
	}
	return len(order) + 1
}

func nameRank(parent, name string) int {
	var lists [][]string
	switch parent {
	case "":
		lists = [][]string{
			{"files", "folders", "text"},
			{"proc", "disk", "net", "host", "svc", "logs", "pack", "auth"},
			{"dev", "lang", "data", "docs"},
			{"help", "explain", "atlas", "recipe", "doctor", "ai-agent"},
			{"config", "color", "completion", "shell-init", "version"},
		}
	case "files":
		lists = [][]string{
			{"list", "show", "open", "find", "map", "stat", "file", "where", "less", "more", "readlink", "realpath", "basename", "dirname"},
			{"copy", "move", "make", "link", "install", "mktemp", "go", "rsync"},
			{"remove", "chmod", "chown", "chgrp"},
		}
	}
	n := 0
	for _, list := range lists {
		for i, item := range list {
			if item == name {
				return n + i
			}
		}
		n += len(list) + 1
	}
	return 10000
}

func filterPrefix(items []Item, prefix string) []Item {
	if prefix == "" {
		return items
	}
	var out []Item
	for _, it := range items {
		if strings.HasPrefix(it.Value, prefix) {
			out = append(out, it)
		}
	}
	return out
}

func paths(mode, prefix string) []Item {
	if mode == "" || mode == "none" {
		return nil
	}
	dir, base := ".", prefix
	if strings.Contains(prefix, "/") {
		dir = filepath.Dir(prefix)
		base = filepath.Base(prefix)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var items []Item
	for _, e := range entries {
		name := e.Name()
		if base != "" && base != "." && !strings.HasPrefix(name, base) {
			continue
		}
		rel := name
		if dir != "." {
			rel = filepath.Join(dir, name)
		}
		if e.IsDir() {
			if mode == "files" {
				continue
			}
			items = append(items, Item{Kind: "folder", Value: rel + "/", Desc: "folder"})
			continue
		}
		if mode == "folders" {
			continue
		}
		items = append(items, Item{Kind: "file", Value: rel, Desc: "file"})
	}
	return items
}

// Format prints candidates as kind, value, display, section separated by tabs.
// The section is the group header. It is not part of the display.
func Format(items []Item) string {
	var b strings.Builder
	for _, it := range items {
		desc := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(it.Desc)
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", it.Kind, it.Value, desc, it.Section)
	}
	return b.String()
}

// Script prints shell completion that keeps commands, files, and folders apart.
func Script(shell string) string {
	switch shell {
	case "bash":
		return bashScript
	case "powershell", "pwsh":
		return psScript
	default:
		return zshScript
	}
}

const hook = `
# zsh expands an existing alias while it reads the next command.
# Remove that alias before the function below is defined.
unalias bhuppi 2>/dev/null || true
if [ -z "${BHUPPI_BIN:-}" ]; then
  if [ -x "$HOME/.local/bin/bhuppi" ]; then
    BHUPPI_BIN="$HOME/.local/bin/bhuppi"
  elif [ -x /usr/local/bin/bhuppi ]; then
    BHUPPI_BIN=/usr/local/bin/bhuppi
  else
    BHUPPI_BIN=$(command -v bhuppi 2>/dev/null || true)
  fi
fi

bhuppi() {
  _line=
  if [ -n "${BHUPPI_BIN:-}" ] && [ -x "$BHUPPI_BIN" ]; then
    _line=$("$BHUPPI_BIN" --print-cd "$@" 2>/dev/null) || _line=
  fi
  case "$_line" in
    cd\ *)
      cd -- "${_line#cd }" || return
      return
      ;;
  esac
  if [ -n "${BHUPPI_BIN:-}" ] && [ -x "$BHUPPI_BIN" ]; then
    "$BHUPPI_BIN" "$@"
    return
  fi
  command bhuppi "$@"
}
`

const zshScript = `# bhuppi completion — menu-select stays. Each row matches the Enter menu.
# The inserted word is the plain verb. The section is a header, not a column.
_bhuppi() {
  zstyle ':completion:*:*:bhuppi:*' menu select
  zstyle ':completion:*:*:bhuppi:*' group-name ''
  if [[ -z ${NO_COLOR:-} ]]; then
    zstyle ':completion:*:*:bhuppi:*:descriptions' format $'\e[38;2;226;194;164m── %d ──\e[0m'
  else
    zstyle ':completion:*:*:bhuppi:*:descriptions' format '── %d ──'
  fi
  local -a cmd_vals cmd_disp files folders
  local kind val desc section shown cur=""
  local bin=${BHUPPI_BIN:-$HOME/.local/bin/bhuppi}
  _bhuppi_flush() {
    (( ${#cmd_vals} )) || return
    local expl
    _description "$cur" expl "$cur"
    compadd -V "$cur" -l "$expl[@]" -d cmd_disp -a cmd_vals
    cmd_vals=()
    cmd_disp=()
  }
  while IFS=$'\t' read -r kind val desc section; do
    [[ -z $kind ]] && continue
    case "$kind" in
      command)
        [[ -z $section ]] && section=commands
        if [[ -n $cur && $section != $cur ]]; then
          _bhuppi_flush
        fi
        cur=$section
        # Display is already the four-column row. Do not append the section.
        shown=$desc
        if [[ $desc != *$'\e'* && $desc != "$val"* && $desc != "! $val"* ]]; then
          shown="$val  $desc"
        fi
        cmd_vals+=("$val")
        cmd_disp+=("$shown")
        ;;
      file) files+=("$val") ;;
      folder) folders+=("$val") ;;
    esac
  done < <(command "$bin" complete -- "${words[@]:1}")
  _bhuppi_flush
  local expl
  if (( ${#files} )); then
    _description files expl 'files'
    compadd -V files -l "$expl[@]" -a files
  fi
  if (( ${#folders} )); then
    _description folders expl 'folders'
    compadd -V folders -l "$expl[@]" -a folders
  fi
}
# zcompdump may restore an older _bhuppi. Rebind after this file, and after a later compinit.
_bhuppi_rebind() {
  if (( $+functions[compdef] )); then
    compdef _bhuppi bhuppi
  fi
}
_bhuppi_rebind
if (( $+functions[compinit] )) && [[ -z ${_bhuppi_compinit_hook:-} ]]; then
  _bhuppi_compinit_hook=1
  functions[_bhuppi_compinit_orig]=$functions[compinit]
  compinit() {
    _bhuppi_compinit_orig "$@"
    _bhuppi_rebind
  }
fi
`

const bashScript = `# bhuppi completion — commands, files, and folders are built separately.
_bhuppi_bash() {
  local cur words
  cur="${COMP_WORDS[COMP_CWORD]}"
  words=("${COMP_WORDS[@]:1:COMP_CWORD}")
  local kind val desc
  local -a cmds files folders
  while IFS=$'\t' read -r kind val desc; do
    case "$kind" in
      command) cmds+=("$val") ;;
      file) files+=("$val") ;;
      folder) folders+=("$val") ;;
    esac
  done < <(command bhuppi complete -- "${words[@]}")
  COMPREPLY=()
  local c
  for c in "${cmds[@]}"; do
    [[ $c == "$cur"* ]] && COMPREPLY+=("$c")
  done
  for c in "${files[@]}"; do
    [[ $c == "$cur"* ]] && COMPREPLY+=("$c")
  done
  for c in "${folders[@]}"; do
    [[ $c == "$cur"* ]] && COMPREPLY+=("$c")
  done
}
complete -o filenames -F _bhuppi_bash bhuppi
`

const psScript = `# bhuppi completion — three groups: commands, files, folders.
Register-ArgumentCompleter -CommandName bhuppi -ScriptBlock {
  param($commandName, $wordToComplete, $cursorPosition)
  $line = $wordToComplete
  $raw = & bhuppi complete -- @($wordToComplete)
  foreach ($row in $raw) {
    $bits = $row.Split([char]9)
    if ($bits.Length -lt 2) { continue }
    $kind = $bits[0]
    $val = $bits[1]
    $tip = $kind
    [System.Management.Automation.CompletionResult]::new($val, $val, 'ParameterValue', $tip)
  }
}
`

// ShellInit prints the cd hook plus completion for one shell.
func ShellInit(shell string) string {
	switch shell {
	case "powershell", "pwsh":
		return psHook + "\n" + psScript
	case "bash":
		return hook + "\n" + bashScript
	default:
		return hook + "\n" + zshScript
	}
}

const psHook = `
function bhuppi {
  $a = @($args)
  $dry = $false
  $help = $false
  $i = 0
  while ($i -lt $a.Count) {
    switch ($a[$i]) {
      '--dry-run' { $dry = $true; $i++; continue }
      '--plain' { $i++; continue }
      '--run' { $i++; continue }
      '-h' { $help = $true; $i++; continue }
      '--help' { $help = $true; $i++; continue }
      '--os' { $i += 2; continue }
      default { break }
    }
  }
  if (-not $dry -and -not $help -and ($i + 2) -lt $a.Count) {
    if (($a[$i] -eq 'files' -or $a[$i] -eq 'folders') -and $a[$i+1] -eq 'go') {
      Set-Location -LiteralPath $a[$i+2]
      return
    }
  }
  $exe = Join-Path $env:LOCALAPPDATA 'bhuppi\lib\bhuppi.exe'
  if (-not (Test-Path $exe)) { $exe = 'bhuppi.exe' }
  & $exe @args
}
`
