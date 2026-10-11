package pager

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func script(keys ...byte) func() (byte, error) {
	i := 0
	return func() (byte, error) {
		if i >= len(keys) {
			return 'q', nil
		}
		b := keys[i]
		i++
		return b, nil
	}
}

func manyLines(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "L%02d\n", i)
	}
	return b.String()
}

func TestShortBufferIsComplete(t *testing.T) {
	called := false
	var dst bytes.Buffer
	err := Page(&dst, strings.NewReader("one\ntwo\n"), Options{
		TTY: true, Height: 10, Width: 80,
		Keys: func() (byte, error) {
			called = true
			return 'q', nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("short output waited for a key")
	}
	if dst.String() != "one\ntwo\n" {
		t.Fatalf("%q", dst.String())
	}
}

func TestLongBufferStopsAtOnePage(t *testing.T) {
	var dst bytes.Buffer
	err := Page(&dst, strings.NewReader(manyLines(30)), Options{
		TTY: true, Height: 10, Width: 40, Keys: script('q'),
	})
	if err != ErrQuit {
		t.Fatalf("err %v", err)
	}
	got := dst.String()
	if !strings.Contains(got, "L09\n") || strings.Contains(got, "L10\n") {
		t.Fatalf("first page:\n%s", got)
	}
}

func TestEnterRevealsOneLine(t *testing.T) {
	var dst bytes.Buffer
	err := Page(&dst, strings.NewReader(manyLines(30)), Options{
		TTY: true, Height: 10, Width: 40, Keys: script('\n', 'q'),
	})
	if err != ErrQuit {
		t.Fatal(err)
	}
	got := dst.String()
	if !strings.Contains(got, "L10\n") || strings.Contains(got, "L11\n") {
		t.Fatalf("enter:\n%s", got)
	}
}

func TestSpaceRevealsNextPage(t *testing.T) {
	var dst bytes.Buffer
	err := Page(&dst, strings.NewReader(manyLines(30)), Options{
		TTY: true, Height: 10, Width: 40, Keys: script(' ', 'q'),
	})
	if err != ErrQuit {
		t.Fatal(err)
	}
	got := dst.String()
	if !strings.Contains(got, "L18\n") || strings.Contains(got, "L19\n") {
		t.Fatalf("space:\n%s", got)
	}
}

func TestNonTTYPrintsEverything(t *testing.T) {
	called := false
	var dst bytes.Buffer
	text := manyLines(30)
	err := Page(&dst, strings.NewReader(text), Options{
		TTY: false, Height: 10, Width: 40,
		Keys: func() (byte, error) {
			called = true
			return 'q', nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("non-TTY read a key")
	}
	if dst.String() != text {
		t.Fatalf("len %d want %d", dst.Len(), len(text))
	}
}
