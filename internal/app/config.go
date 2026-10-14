package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bhupenderbhardwaj/bhuppi/internal/catalog"
	"github.com/bhupenderbhardwaj/bhuppi/internal/help"
	"github.com/bhupenderbhardwaj/bhuppi/internal/reputation"
	"golang.org/x/term"
)

func configCmd(cat *catalog.Catalog, args []string, opt help.Options, stdout, stderr io.Writer) int {
	if len(args) == 0 || (len(args) == 1 && args[0] == "reputation") {
		return showHelp(cat, []string{"config"}, opt, stdout, stderr)
	}
	if args[0] != "reputation" {
		fmt.Fprintf(stderr, "unknown command %q\nsee: bhuppi help config\n", args[0])
		return 2
	}
	return reputationCmd(args[1:], stdout, stderr)
}

func showHelp(cat *catalog.Catalog, path []string, opt help.Options, stdout, stderr io.Writer) int {
	s, err := help.Render(cat, path, opt)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}
	emit(stdout, s)
	return 0
}

func reputationCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(stderr, "see: bhuppi help config\n")
		return 2
	}
	switch args[0] {
	case "status":
		if len(args) != 1 {
			fmt.Fprintf(stderr, "see: bhuppi help config\n")
			return 2
		}
		keys, err := reputation.Load()
		if err != nil {
			fmt.Fprintf(stderr, "reputation: %v\n", err)
			return 1
		}
		fmt.Fprint(stdout, reputation.Status(keys))
		return 0
	case "set":
		if len(args) != 2 {
			fmt.Fprintf(stderr, "the key is not a command argument\nsee: bhuppi config reputation set vt\nsee: bhuppi config reputation set abuseip\n")
			return 2
		}
		if args[1] != "vt" && args[1] != "abuseip" {
			fmt.Fprintf(stderr, "reputation set is vt or abuseip\nsee: bhuppi help config\n")
			return 2
		}
		key, err := readSecret()
		if err != nil {
			fmt.Fprintf(stderr, "reputation: %v\n", err)
			return 1
		}
		if key == "" {
			fmt.Fprintf(stderr, "empty key\n")
			return 2
		}
		keys, err := reputation.Load()
		if err != nil {
			fmt.Fprintf(stderr, "reputation: %v\n", err)
			return 1
		}
		label := "virustotal"
		if args[1] == "vt" {
			keys.VirusTotal = key
		} else {
			keys.AbuseIP = key
			label = "abuseip"
		}
		if err := reputation.Save(keys); err != nil {
			fmt.Fprintf(stderr, "reputation: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s  set\n", label)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\nsee: bhuppi help config\n", args[0])
		return 2
	}
}

// readSecret takes the key from the terminal with echo off, or one line from stdin
// when stdin is not a terminal. The key is never an argument.
func readSecret() (string, error) {
	in := os.Stdin
	if term.IsTerminal(int(in.Fd())) {
		fmt.Fprint(os.Stderr, "key: ")
		b, err := term.ReadPassword(int(in.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
