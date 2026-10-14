package aireview

import (
	"encoding/xml"
	"strings"
)

// Plist is the small slice of a launchd plist the review is allowed to show.
type Plist struct {
	Label     string
	Program   string
	Args      []string
	RunAtLoad bool
}

// ParsePlist reads Label, Program, ProgramArguments, and RunAtLoad from an XML plist.
// It does not return the rest of the file.
func ParsePlist(data string) (Plist, bool) {
	dec := xml.NewDecoder(strings.NewReader(data))
	var p Plist
	var key string
	var inKey, inString, inArgs bool
	var buf strings.Builder
	ok := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "key":
				inKey = true
				buf.Reset()
			case "string":
				inString = true
				buf.Reset()
			case "array":
				if key == "ProgramArguments" {
					inArgs = true
				}
			case "true":
				if key == "RunAtLoad" {
					p.RunAtLoad = true
					ok = true
				}
			case "false":
				if key == "RunAtLoad" {
					p.RunAtLoad = false
					ok = true
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "key":
				key = strings.TrimSpace(buf.String())
				inKey = false
			case "string":
				val := strings.TrimSpace(buf.String())
				inString = false
				switch {
				case inArgs:
					p.Args = append(p.Args, val)
					ok = true
				case key == "Label":
					p.Label = val
					ok = true
				case key == "Program":
					p.Program = val
					ok = true
				}
			case "array":
				inArgs = false
			}
		case xml.CharData:
			if inKey || inString {
				buf.Write(t)
			}
		}
	}
	return p, ok && (p.Label != "" || p.Program != "" || len(p.Args) > 0)
}

// Strings are the program path and arguments, nothing else from the plist.
func (p Plist) Strings() []string {
	out := make([]string, 0, 1+len(p.Args))
	if p.Program != "" {
		out = append(out, p.Program)
	}
	out = append(out, p.Args...)
	return out
}

// Hit reports whether the plist names the agent or its program.
func (p Plist) Hit(name, program string) bool {
	if Mentions(p.Program, name, program) {
		return true
	}
	for _, a := range p.Args {
		if Mentions(a, name, program) {
			return true
		}
	}
	return false
}
