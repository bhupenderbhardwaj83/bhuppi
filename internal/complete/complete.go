package complete

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
)

// Item is one completion candidate. Kind is command, file, or folder.
type Item struct {
	Kind  string
	Value string
	Desc  string
}

// Complete returns three visual groups: commands, files, folders.
// words are the tokens after `bhuppi`. The last token is the prefix being typed.
func Complete(cat *catalog.Catalog, words []string) []Item {
	prefix := ""
	prev := words
	if len(words) > 0 {
		prefix = words[len(words)-1]
		prev = words[:len(words)-1]
	}
	if len(prev) == 0 {
		return filterPrefix(rootCommands(cat), prefix)
	}
	if prev[0] == "recipe" {
		if len(prev) == 1 {
			var items []Item
			for i := range cat.Recipes {
				r := &cat.Recipes[i]
				items = append(items, Item{Kind: "command", Value: r.Name, Desc: r.Title})
			}
			return filterPrefix(items, prefix)
		}
		return nil
	}
	if prev[0] == "explain" && len(prev) == 1 {
		var items []Item
		for _, name := range cat.ExplainNames() {
			items = append(items, Item{Kind: "command", Value: name, Desc: cat.Explain[name].Target})
		}
		return filterPrefix(items, prefix)
	}
	if isMeta(cat, prev[0]) && prev[0] != "help" && prev[0] != "h" {
		return nil
	}
	args := prev
	if prev[0] == "help" || prev[0] == "h" {
		args = prev[1:]
		if len(args) == 0 {
			return filterPrefix(rootCommands(cat), prefix)
		}
	}
	g := cat.Group(args[0])
	if g == nil {
		return filterPrefix(rootCommands(cat), prefix)
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
		for i := range node.Children {
			ch := &node.Children[i]
			items = append(items, Item{Kind: "command", Value: ch.Name, Desc: ch.Title})
		}
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

func rootCommands(cat *catalog.Catalog) []Item {
	var items []Item
	for i := range cat.Groups {
		g := &cat.Groups[i]
		items = append(items, Item{Kind: "command", Value: g.Name, Desc: g.Title})
	}
	for i := range cat.MetaPages {
		m := &cat.MetaPages[i]
		items = append(items, Item{Kind: "command", Value: m.Name, Desc: m.Title})
	}
	return items
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

// Format prints candidates as kind, value, desc separated by tabs.
func Format(items []Item) string {
	var b strings.Builder
	for _, it := range items {
		fmt.Fprintf(&b, "%s\t%s\t%s\n", it.Kind, it.Value, it.Desc)
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
  if [ -x /usr/local/bin/bhuppi ]; then
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

const zshScript = `# bhuppi completion — commands, files, and folders stay in three groups.
_bhuppi() {
  zstyle ':completion:*:*:bhuppi:*' group-name ''
  zstyle ':completion:*:*:bhuppi:*:descriptions' format '%F{8}── %d ──%f'
  local -a cmds files folders
  local kind val desc
  while IFS=$'\t' read -r kind val desc; do
    case "$kind" in
      command) cmds+=("${val}:${desc}") ;;
      file) files+=("$val") ;;
      folder) folders+=("$val") ;;
    esac
  done < <(command bhuppi complete -- "${words[@]:1}")
  _describe -t commands 'commands' cmds
  if (( ${#files} )); then
    _describe -t files 'files' files
  fi
  if (( ${#folders} )); then
    _describe -t folders 'folders' folders
  fi
}
if (( $+functions[compdef] )); then
  compdef _bhuppi bhuppi
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
