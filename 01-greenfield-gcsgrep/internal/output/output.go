// Package output writes results to stdout and everything else to stderr (BR-9).
package output

import (
	"fmt"
	"io"
	"strconv"
	"sync"
)

// Errorf writes "gcsgrep: <message>" and a newline to w in a single write.
func Errorf(w io.Writer, format string, args ...any) {
	_, _ = io.WriteString(w, "gcsgrep: "+fmt.Sprintf(format, args...)+"\n")
}

// Printer writes matches to stdout and warnings to stderr. It is safe for
// concurrent use: each record goes out whole, in one write, so lines from
// different objects can interleave but never mix.
type Printer struct {
	mu          sync.Mutex
	out, err    io.Writer
	lineNumbers bool
	buf         []byte
}

// New returns a Printer. With lineNumbers, records carry the line number (-n).
func New(out, err io.Writer, lineNumbers bool) *Printer {
	return &Printer{out: out, err: err, lineNumbers: lineNumbers}
}

// Match writes gs://<bucket>/<object>:[<n>:]<text>[...] and a newline. The text
// is followed by "..." when it is the truncated start of a longer line (BR-8).
func (p *Printer) Match(uri string, lineNo int64, text []byte, truncated bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	b := append(p.buf[:0], uri...)
	b = append(b, ':')
	if p.lineNumbers {
		b = strconv.AppendInt(b, lineNo, 10)
		b = append(b, ':')
	}
	b = append(b, text...)
	if truncated {
		b = append(b, "..."...)
	}
	b = append(b, '\n')
	p.buf = b

	_, err := p.out.Write(b)
	return err
}

// Warnf writes "gcsgrep: <message>" to stderr.
func (p *Printer) Warnf(format string, args ...any) {
	p.mu.Lock()
	defer p.mu.Unlock()
	Errorf(p.err, format, args...)
}
