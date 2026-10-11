// Package pager shows one screen of output and waits for a key.
// Enter reveals one line, Space the next page, q stops.
// A non-terminal writer receives the full text.
package pager

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// ErrQuit means the reader pressed q or Ctrl-C. The rest of the text is dropped.
var ErrQuit = errors.New("pager quit")

// Options controls a page. Height is the terminal row count.
// Keys is nil in production; tests pass a fake reader.
type Options struct {
	TTY    bool
	Height int
	Width  int
	Keys   func() (byte, error)
}

// IsTerminal reports whether w is a terminal.
func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// Write pages text when dst is a terminal and the text is taller than the screen.
func Write(dst io.Writer, text string) error {
	return Copy(dst, strings.NewReader(text))
}

// Copy pages a stream. The destination's terminal size is used when it is a TTY.
func Copy(dst io.Writer, src io.Reader) error {
	opt := Options{TTY: IsTerminal(dst)}
	if opt.TTY {
		opt.Width, opt.Height = sizeOf(dst)
	}
	return Page(dst, src, opt)
}

// Page is Write with an explicit screen and key reader, for tests.
func Page(dst io.Writer, src io.Reader, opt Options) error {
	if !opt.TTY || opt.Height <= 0 {
		_, err := io.Copy(dst, src)
		return err
	}
	width := opt.Width
	if width <= 0 {
		width = 80
	}
	page := opt.Height - 1
	if page < 1 {
		page = 1
	}
	feed := &feeder{in: bufio.NewReader(src)}

	// First screen uses the full height when the text fits.
	// When it does not, one row is kept for the prompt.
	lines, more, err := feed.take(opt.Height, width, false)
	if err != nil {
		return err
	}
	if !more {
		return writeLines(dst, lines)
	}
	shown, left := splitRows(lines, page, width)
	if err := writeLines(dst, shown); err != nil {
		return err
	}
	feed.pushFront(left)

	oneLine := false
	for {
		fullPage, err := prompt(dst, opt)
		if err != nil {
			return err
		}
		oneLine = !fullPage
		var chunk []string
		if oneLine {
			chunk, more, err = feed.take(1, width, true)
		} else {
			chunk, more, err = feed.take(page, width, false)
		}
		if err != nil {
			return err
		}
		if err := writeLines(dst, chunk); err != nil {
			return err
		}
		if !more {
			return nil
		}
	}
}

func writeLines(dst io.Writer, lines []string) error {
	for _, ln := range lines {
		if _, err := fmt.Fprintln(dst, ln); err != nil {
			return err
		}
	}
	return nil
}

// prompt waits for one key. fullPage is true when the next chunk is a page (Space).
func prompt(dst io.Writer, opt Options) (fullPage bool, err error) {
	const hint = "-- Enter line · Space page · q stop --"
	if _, err = io.WriteString(dst, hint); err != nil {
		return false, err
	}
	var key byte
	if opt.Keys != nil {
		key, err = opt.Keys()
	} else {
		key, err = readTTY()
	}
	_, _ = io.WriteString(dst, "\r\033[K")
	if err != nil {
		return false, err
	}
	switch key {
	case 'q', 'Q', 3:
		return false, ErrQuit
	case ' ':
		return true, nil
	default:
		// Enter, newline, and anything else reveal one line.
		return false, nil
	}
}

type feeder struct {
	in  *bufio.Reader
	buf []string
}

func (f *feeder) pushFront(lines []string) {
	if len(lines) == 0 {
		return
	}
	f.buf = append(append([]string{}, lines...), f.buf...)
}

func (f *feeder) next() (string, bool, error) {
	if len(f.buf) > 0 {
		s := f.buf[0]
		f.buf = f.buf[1:]
		return s, true, nil
	}
	s, err := f.in.ReadString('\n')
	if len(s) > 0 {
		for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
			s = s[:len(s)-1]
		}
		return s, true, nil
	}
	if errors.Is(err, io.EOF) {
		return "", false, nil
	}
	return "", false, err
}

// take reads until maxRows visual rows, or exactly one line when oneLine is set.
// more is true when unread text remains.
func (f *feeder) take(maxRows, width int, oneLine bool) ([]string, bool, error) {
	if oneLine {
		s, ok, err := f.next()
		if err != nil || !ok {
			return nil, false, err
		}
		nxt, ok, err := f.next()
		if err != nil {
			return nil, false, err
		}
		if ok {
			f.pushFront([]string{nxt})
		}
		return []string{s}, ok, nil
	}
	var out []string
	rows := 0
	for {
		s, ok, err := f.next()
		if err != nil {
			return nil, false, err
		}
		if !ok {
			return out, false, nil
		}
		need := visualRows(s, width)
		if len(out) > 0 && rows+need > maxRows {
			f.pushFront([]string{s})
			return out, true, nil
		}
		out = append(out, s)
		rows += need
		if rows >= maxRows {
			nxt, ok, err := f.next()
			if err != nil {
				return nil, false, err
			}
			if ok {
				f.pushFront([]string{nxt})
				return out, true, nil
			}
			return out, false, nil
		}
	}
}

func splitRows(lines []string, maxRows, width int) (shown, left []string) {
	rows := 0
	for i, ln := range lines {
		need := visualRows(ln, width)
		if i > 0 && rows+need > maxRows {
			return lines[:i], lines[i:]
		}
		rows += need
		if rows >= maxRows {
			return lines[:i+1], lines[i+1:]
		}
	}
	return lines, nil
}

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func visualRows(line string, width int) int {
	n := 0
	for _, r := range ansiRE.ReplaceAllString(line, "") {
		if r == '\t' {
			n += 4
			continue
		}
		n++
	}
	if width <= 0 || n == 0 {
		return 1
	}
	rows := (n + width - 1) / width
	if rows < 1 {
		return 1
	}
	return rows
}

func sizeOf(w io.Writer) (width, height int) {
	f, ok := w.(*os.File)
	if !ok {
		return 80, 24
	}
	width, height, err := term.GetSize(int(f.Fd()))
	if err != nil || height <= 0 {
		return 80, 24
	}
	if width <= 0 {
		width = 80
	}
	return width, height
}

func readTTY() (byte, error) {
	fd, closer, err := console()
	if err != nil {
		return 0, err
	}
	if closer != nil {
		defer closer.Close()
	}
	state, err := term.MakeRaw(fd)
	if err != nil {
		return 0, err
	}
	restored := false
	restore := func() {
		if restored {
			return
		}
		restored = true
		_ = term.Restore(fd, state)
	}
	defer restore()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-sig:
			restore()
		case <-done:
		}
	}()

	var buf [1]byte
	_, err = syscall.Read(fd, buf[:])
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

func console() (int, io.Closer, error) {
	// Prefer the stdin terminal so a pty test and a normal shell share one
	// keyboard. /dev/tty is only for the case where stdin is a pipe and
	// stdout is still a terminal.
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return int(os.Stdin.Fd()), nil, nil
	}
	f, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
	if err != nil {
		return 0, nil, err
	}
	return int(f.Fd()), f, nil
}
