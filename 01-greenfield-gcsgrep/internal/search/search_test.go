package search

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"
	"testing/iotest"
)

type hit struct {
	no        int64
	text      string
	truncated bool
}

func scan(t *testing.T, input, pattern string) []hit {
	t.Helper()
	return scanReader(t, strings.NewReader(input), regexp.MustCompile(pattern))
}

func scanReader(t *testing.T, r io.Reader, re *regexp.Regexp) []hit {
	t.Helper()
	var hits []hit
	err := NewScanner().Scan(r, re, func(l Line) error {
		hits = append(hits, hit{l.No, string(l.Text), l.Truncated})
		return nil
	})
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return hits
}

func TestScanLines(t *testing.T) {
	tests := []struct {
		name, input, pattern string
		want                 []hit
	}{
		{"empty object", "", "x", nil},
		{"only a newline", "\n", "x", nil},
		{"numbering starts at 1", "a\nx\nb\nx\n", "x", []hit{{2, "x", false}, {4, "x", false}}},
		{"no trailing newline", "a\nxy", "x", []hit{{2, "xy", false}}},
		{"empty lines count", "\n\nx\n", "x", []hit{{3, "x", false}}},
		{"empty line matches ^$", "a\n\nb\n", "^$", []hit{{2, "", false}}},
		{"CRLF trimmed", "one\r\ntwo x\r\n", "x", []hit{{2, "two x", false}}},
		{"CR trimmed before anchor", "Windows\r\n", "Windows$", []hit{{1, "Windows", false}}},
		{"only the last CR is trimmed", "a\r\r\nx\n", "a", []hit{{1, "a\r", false}}},
		{"CR in the middle stays", "a\rx\n", "x", []hit{{1, "a\rx", false}}},
		{"CR at EOF without newline", "x\r", "x$", []hit{{1, "x", false}}},
		{"lone CRLF line", "\r\nx\n", "x", []hit{{2, "x", false}}},
		{"no match", "a\nb\n", "z", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scan(t, tt.input, tt.pattern)
			if len(got) != len(tt.want) {
				t.Fatalf("hits = %+v, want %+v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("hit %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// A "\r" that lands as the last byte of a read buffer must not be lost, and
// must still be trimmed when it turns out to be the one before "\n".
func TestScanCRAtBufferBoundary(t *testing.T) {
	a := strings.Repeat("a", readSize-1)

	t.Run("CR then LF across the boundary is trimmed", func(t *testing.T) {
		got := scan(t, a+"\r\nnext\n", "a$")
		if len(got) != 1 || got[0].text != a {
			t.Fatalf("got %d hits, want the line without its CR", len(got))
		}
	})
	t.Run("CR followed by data across the boundary is kept", func(t *testing.T) {
		got := scan(t, a+"\rb\n", "a\rb$")
		if len(got) != 1 || got[0].text != a+"\rb" {
			t.Fatalf("got %d hits, want the line with its inner CR", len(got))
		}
	})
	t.Run("CR then EOF across the boundary is trimmed", func(t *testing.T) {
		got := scan(t, a+"\r", "a$")
		if len(got) != 1 || got[0].text != a {
			t.Fatalf("got %d hits, want the line without its CR", len(got))
		}
	})
	t.Run("line of exactly the buffer size, no terminator", func(t *testing.T) {
		line := strings.Repeat("a", readSize)
		got := scan(t, line, "a$")
		if len(got) != 1 || got[0].text != line {
			t.Fatalf("got %d hits", len(got))
		}
	})
}

func TestScanLongLines(t *testing.T) {
	long := func(n int, fill byte) string { return strings.Repeat(string(fill), n) }

	t.Run("exactly MaxLine is not truncated", func(t *testing.T) {
		line := long(MaxLine, 'a')
		got := scan(t, line+"\nnext\n", "a$")
		if len(got) != 1 || got[0].truncated || len(got[0].text) != MaxLine {
			t.Fatalf("got %d hits, first truncated=%v len=%d", len(got), got[0].truncated, len(got[0].text))
		}
	})
	t.Run("MaxLine plus CRLF is not truncated", func(t *testing.T) {
		got := scan(t, long(MaxLine, 'a')+"\r\nnext\n", "a$")
		if len(got) != 1 || got[0].truncated || len(got[0].text) != MaxLine {
			t.Fatalf("got %+v", len(got))
		}
	})
	t.Run("MaxLine plus CR then EOF is not truncated", func(t *testing.T) {
		got := scan(t, long(MaxLine, 'a')+"\r", "a$")
		if len(got) != 1 || got[0].truncated {
			t.Fatalf("got %d hits", len(got))
		}
	})
	t.Run("MaxLine+1 is truncated", func(t *testing.T) {
		got := scan(t, long(MaxLine+1, 'a')+"\n", "a")
		if len(got) != 1 || !got[0].truncated || len(got[0].text) != MaxLine {
			t.Fatalf("got %d hits", len(got))
		}
	})
	t.Run("MaxLine+1 plus CRLF is truncated", func(t *testing.T) {
		got := scan(t, long(MaxLine+1, 'a')+"\r\n", "a")
		if len(got) != 1 || !got[0].truncated || len(got[0].text) != MaxLine {
			t.Fatalf("got %d hits", len(got))
		}
	})
	t.Run("MaxLine+1 at EOF is truncated", func(t *testing.T) {
		got := scan(t, long(MaxLine+1, 'a'), "a")
		if len(got) != 1 || !got[0].truncated {
			t.Fatalf("got %d hits", len(got))
		}
	})
	t.Run("late match is reported with the first MiB", func(t *testing.T) {
		line := long(MaxLine+100, 'a') + "late" + long(10, 'b')
		got := scan(t, "x\n"+line+"\nlast\n", "late")
		if len(got) != 1 || got[0].no != 2 || !got[0].truncated {
			t.Fatalf("got %+v", got)
		}
		if got[0].text != line[:MaxLine] {
			t.Errorf("text is not the first MiB of the line")
		}
	})
	t.Run("early match is reported with the first MiB", func(t *testing.T) {
		line := "early" + long(MaxLine+100, 'a')
		got := scan(t, line+"\n", "early")
		if len(got) != 1 || !got[0].truncated || got[0].text != line[:MaxLine] {
			t.Fatalf("got %d hits", len(got))
		}
	})
	t.Run("regex crossing the limit matches", func(t *testing.T) {
		line := "early" + long(MaxLine, 'a') + "late"
		got := scan(t, line+"\n", "early.*late")
		if len(got) != 1 || !got[0].truncated {
			t.Fatalf("got %d hits", len(got))
		}
	})
	t.Run("regex crossing the limit does not match a different line", func(t *testing.T) {
		got := scan(t, "early"+long(MaxLine, 'a')+"\n"+"late\n", "early.*late")
		if len(got) != 0 {
			t.Fatalf("matched across lines: %+v", got)
		}
	})
	t.Run("end anchor looks at the real end of a long line", func(t *testing.T) {
		// The retained prefix (MaxLine+1 bytes) ends in 'a', the line does
		// not: a match on the prefix alone would be a false positive.
		line := long(MaxLine+1, 'a') + long(10, 'b')
		if got := scan(t, line+"\n", "a$"); len(got) != 0 {
			t.Fatalf("a$ matched a line that ends in b")
		}
		if got := scan(t, line+"\n", "b$"); len(got) != 1 {
			t.Fatalf("b$ did not match a line that ends in b")
		}
	})
	t.Run("end anchor after CRLF on a long line", func(t *testing.T) {
		line := long(MaxLine+50, 'a')
		if got := scan(t, line+"\r\n", "a$"); len(got) != 1 {
			t.Fatalf("a$ did not match a long CRLF line")
		}
	})
	t.Run("word boundary looks at the real next byte", func(t *testing.T) {
		// The retained prefix (MaxLine+1 bytes) ends exactly after "foo".
		line := long(MaxLine-2, 'x') + "foo" + "bar"
		if got := scan(t, line+"\n", `foo\b`); len(got) != 0 {
			t.Fatalf(`foo\b matched inside "foobar"`)
		}
	})
	t.Run("the line after a long one is read correctly", func(t *testing.T) {
		got := scan(t, long(MaxLine*2, 'a')+"\nhit\nhit\n", "hit")
		if len(got) != 2 || got[0].no != 2 || got[1].no != 3 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("a long line that does not match is still consumed", func(t *testing.T) {
		got := scan(t, long(MaxLine*3, 'a')+"\nhit\n", "hit")
		if len(got) != 1 || got[0].no != 2 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("several long lines in a row", func(t *testing.T) {
		in := long(MaxLine+5, 'a') + "\n" + long(MaxLine+7, 'b') + "\r\n" + "c\n"
		got := scan(t, in, "[abc]$")
		if len(got) != 3 || !got[0].truncated || !got[1].truncated || got[2].truncated {
			t.Fatalf("got %+v", len(got))
		}
	})
}

func TestScanErrors(t *testing.T) {
	boom := errors.New("boom")

	t.Run("read error is returned", func(t *testing.T) {
		r := io.MultiReader(strings.NewReader("x\nab"), iotest.ErrReader(boom))
		var hits int
		err := NewScanner().Scan(r, regexp.MustCompile("x"), func(Line) error { hits++; return nil })
		if !errors.Is(err, boom) {
			t.Fatalf("err = %v, want boom", err)
		}
		if hits != 1 {
			t.Errorf("hits = %d, want the one line read before the error", hits)
		}
	})
	t.Run("a partial line is not matched", func(t *testing.T) {
		r := io.MultiReader(strings.NewReader("ab"), iotest.ErrReader(boom))
		var hits int
		_ = NewScanner().Scan(r, regexp.MustCompile("b$"), func(Line) error { hits++; return nil })
		if hits != 0 {
			t.Errorf("matched a line that was cut by a read error")
		}
	})
	t.Run("read error in the middle of a long line", func(t *testing.T) {
		// "$" would match the partial line if the error were taken for EOF.
		// Enough bytes come first for the error to hit while the rest of the
		// line is being streamed, not while it is being retained.
		r := io.MultiReader(strings.NewReader(strings.Repeat("a", MaxLine+3*readSize)), iotest.ErrReader(boom))
		var hits int
		err := NewScanner().Scan(r, regexp.MustCompile("a$"), func(Line) error { hits++; return nil })
		if !errors.Is(err, boom) || hits != 0 {
			t.Fatalf("err = %v, hits = %d", err, hits)
		}
	})
	t.Run("onMatch error stops the scan", func(t *testing.T) {
		stop := errors.New("stop")
		var hits int
		err := NewScanner().Scan(strings.NewReader("x\nx\nx\n"), regexp.MustCompile("x"),
			func(Line) error { hits++; return stop })
		if !errors.Is(err, stop) || hits != 1 {
			t.Fatalf("err = %v, hits = %d", err, hits)
		}
	})
}

func TestScanReusesScanner(t *testing.T) {
	s := NewScanner()
	re := regexp.MustCompile("x")
	for i := 0; i < 3; i++ {
		var got []int64
		err := s.Scan(strings.NewReader("a\nx\n"), re, func(l Line) error { got = append(got, l.No); return nil })
		if err != nil || len(got) != 1 || got[0] != 2 {
			t.Fatalf("pass %d: err=%v got=%v", i, err, got)
		}
	}
}

// One byte at a time exercises every boundary the buffers have.
func TestScanOneByteReads(t *testing.T) {
	in := "a\r\nx one\r\n\r\nx two"
	got := scanReader(t, iotest.OneByteReader(strings.NewReader(in)), regexp.MustCompile("x"))
	if len(got) != 2 || got[0] != (hit{2, "x one", false}) || got[1] != (hit{4, "x two", false}) {
		t.Fatalf("got %+v", got)
	}
}

func TestScanTextIsIndependentOfNextRead(t *testing.T) {
	var texts []string
	err := NewScanner().Scan(strings.NewReader("first x\nsecond x\n"), regexp.MustCompile("x"),
		func(l Line) error { texts = append(texts, string(l.Text)); return nil })
	if err != nil || len(texts) != 2 || texts[0] != "first x" || texts[1] != "second x" {
		t.Fatalf("err=%v texts=%q", err, texts)
	}
}

func TestCompile(t *testing.T) {
	match := func(re *regexp.Regexp, s string) bool { return re.Match([]byte(s)) }

	t.Run("literal is not a regex", func(t *testing.T) {
		re, err := Compile("job_id=10.", false, false)
		if err != nil {
			t.Fatal(err)
		}
		if match(re, "job_id=101") || !match(re, "job_id=10.") {
			t.Error("the pattern was treated as a regex")
		}
	})
	t.Run("any non-empty literal is valid", func(t *testing.T) {
		for _, p := range []string{"(", "[", `\`, "*", "a{2", "(?"} {
			if _, err := Compile(p, false, false); err != nil {
				t.Errorf("literal %q rejected: %v", p, err)
			}
			if _, err := Compile(p, false, true); err != nil {
				t.Errorf("literal %q rejected with -i: %v", p, err)
			}
		}
	})
	t.Run("regex", func(t *testing.T) {
		re, err := Compile(`job_id=\d+ queue=\w+`, true, false)
		if err != nil {
			t.Fatal(err)
		}
		if !match(re, "job_id=102 queue=default") || match(re, "job_id=x queue=y") {
			t.Error("regex did not behave as RE2")
		}
	})
	t.Run("invalid regex quotes the user's pattern, not the -i prefix", func(t *testing.T) {
		_, err := Compile("(", true, true)
		if err == nil {
			t.Fatal("accepted an invalid regex")
		}
		if strings.Contains(err.Error(), "(?i)") {
			t.Errorf("error leaks the internal prefix: %v", err)
		}
	})
	t.Run("ignore case folds non-ASCII letters", func(t *testing.T) {
		re, err := Compile("ñandú árbol", false, true)
		if err != nil {
			t.Fatal(err)
		}
		if !match(re, "ÑANDÚ ÁRBOL") {
			t.Error("-i did not fold ñ/Ñ and á/Á")
		}
		re, _ = Compile("ñandú árbol", false, false)
		if match(re, "ÑANDÚ ÁRBOL") {
			t.Error("matched without -i")
		}
	})
	t.Run("ignore case applies to every alternative", func(t *testing.T) {
		re, err := Compile("foo|bar", true, true)
		if err != nil {
			t.Fatal(err)
		}
		if !match(re, "FOO") || !match(re, "BAR") {
			t.Error("(?i) did not cover both alternatives")
		}
	})
	t.Run("no multi-character folding", func(t *testing.T) {
		re, _ := Compile("ß", false, true)
		if match(re, "SS") {
			t.Error("ß matched SS")
		}
	})
}

func TestKeepBufferStaysBounded(t *testing.T) {
	s := NewScanner()
	in := bytes.Repeat([]byte("a"), MaxLine*3)
	if err := s.Scan(bytes.NewReader(in), regexp.MustCompile("a"), func(Line) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if cap(s.keep) != keepCap {
		t.Errorf("cap(keep) = %d after a 3 MiB line, want %d", cap(s.keep), keepCap)
	}
}
