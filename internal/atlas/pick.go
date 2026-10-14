package atlas

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// ErrNoCase means that use-case number is not on the page.
var ErrNoCase = errors.New("no use case")

var caseTitle = regexp.MustCompile(`(?i)use case\s+0*(\d+)`)
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// CaseCommands is the 1-based use-case command list for a named page.
// Index 0 is use case 1. An empty slot means that number has no command.
func CaseCommands(name string) []string {
	c := find(name)
	if c == nil {
		return nil
	}
	nums := make([]int, len(c.Cases))
	max := 0
	for i, uc := range c.Cases {
		n := i + 1
		if m := caseTitle.FindStringSubmatch(uc.Title); len(m) == 2 {
			if v, err := strconv.Atoi(m[1]); err == nil && v > 0 {
				n = v
			}
		}
		nums[i] = n
		if n > max {
			max = n
		}
	}
	if max == 0 {
		return nil
	}
	out := make([]string, max)
	for i, uc := range c.Cases {
		if uc.Command != "" {
			out[nums[i]-1] = uc.Command
		}
	}
	return out
}

// CommandsIn reads use-case command lines out of an atlas page body.
// The command is the first non-empty line under each "Use Case NN" heading.
func CommandsIn(body string) []string {
	lines := strings.Split(body, "\n")
	found := map[int]string{}
	max := 0
	for i := 0; i < len(lines); i++ {
		m := caseTitle.FindStringSubmatch(strip(lines[i]))
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil || n < 1 {
			continue
		}
		cmd := ""
		for j := i + 1; j < len(lines); j++ {
			s := strings.TrimSpace(strip(lines[j]))
			if s == "" {
				continue
			}
			if caseTitle.MatchString(s) {
				break
			}
			cmd = s
			break
		}
		found[n] = cmd
		if n > max {
			max = n
		}
	}
	out := make([]string, max)
	for n, cmd := range found {
		out[n-1] = cmd
	}
	return out
}

// Pick returns the 1-based use-case command.
func Pick(commands []string, n int) (string, bool) {
	if n < 1 || n > len(commands) {
		return "", false
	}
	cmd := commands[n-1]
	if cmd == "" {
		return "", false
	}
	return cmd, true
}

// Deliver copies use case n from an atlas body through write.
// A missing number does not call write.
func Deliver(body string, n int, write func(string) error) (string, error) {
	cmd, ok := Pick(CommandsIn(body), n)
	if !ok {
		return "", ErrNoCase
	}
	if write != nil {
		if err := write(cmd); err != nil {
			return "", err
		}
	}
	return cmd, nil
}

func strip(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}
