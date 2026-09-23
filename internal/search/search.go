// Package search splits a byte stream into lines and matches each one whole.
//
// It knows nothing about GCS: it receives an io.Reader. Memory is bounded: a
// line is matched in full, but only its first MaxLine bytes are ever retained
// for printing (BR-8).
package search

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
)

const (
	// MaxLine is the most bytes of a line that are kept for output (1 MiB).
	MaxLine = 1 << 20

	// readSize is the size of the read buffer, and so of each piece of a long line.
	readSize = 64 << 10

	// keepCap holds one byte more than MaxLine: seeing that extra byte is what
	// tells a line of exactly MaxLine bytes from a longer one.
	keepCap = MaxLine + 1
)

// Compile builds the matcher for a pattern (FR-3, FR-4, FR-7). Without regex
// the pattern is a literal. ignoreCase uses RE2's simple Unicode case folding,
// so the result does not depend on the locale.
func Compile(pattern string, regex, ignoreCase bool) (*regexp.Regexp, error) {
	if !regex {
		pattern = regexp.QuoteMeta(pattern)
	}
	// Compile the pattern alone first so that a syntax error quotes what the
	// user typed, not the "(?i)" we would prepend.
	re, err := regexp.Compile(pattern)
	if err != nil || !ignoreCase {
		return re, err
	}
	return regexp.Compile("(?i)" + pattern)
}

// Line is a matching line. Text is only valid during the onMatch call.
type Line struct {
	// No is the 1-based line number.
	No int64
	// Text is the line without its terminator, and at most MaxLine bytes long.
	Text []byte
	// Truncated reports that the line was longer than MaxLine.
	Truncated bool
}

// Scanner reads lines. It is not safe for concurrent use; give each worker its
// own. Reusing one across objects reuses its buffers.
type Scanner struct {
	br   *bufio.Reader
	keep []byte
}

// NewScanner returns a Scanner ready to Scan.
func NewScanner() *Scanner {
	return &Scanner{
		br:   bufio.NewReaderSize(nil, readSize),
		keep: make([]byte, 0, keepCap),
	}
}

// Scan reads r to the end and calls onMatch for every line that re matches, in
// order. A line ends at "\n", or at the end of r; one "\r" just before the
// terminator is dropped (FR-9) before matching and before printing. The
// terminator is not part of the line, so "$" matches at its end.
//
// It stops at the first error from r or from onMatch and returns it. Lines
// already reported stay reported, but the line being read when r failed is not
// matched.
func (s *Scanner) Scan(r io.Reader, re *regexp.Regexp, onMatch func(Line) error) error {
	s.br.Reset(r)
	for no := int64(1); ; no++ {
		tail, err := s.readLine()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		var matched bool
		if tail == nil && len(s.keep) <= MaxLine {
			matched = re.Match(s.keep)
		} else {
			// The line does not fit in s.keep (or fills it exactly): match the
			// whole line as a stream. Matching only the retained prefix would
			// be wrong for "$", "\b" and friends.
			if matched, err = s.matchLong(re, tail); err != nil {
				return err
			}
		}
		if !matched {
			continue
		}

		line := Line{No: no, Text: s.keep}
		if len(s.keep) > MaxLine {
			line.Text = s.keep[:MaxLine]
			line.Truncated = true
		}
		if err := onMatch(line); err != nil {
			return err
		}
	}
}

// readLine reads the next line into s.keep, at most MaxLine+1 bytes of it. If
// the line has more than that, it returns a reader for the rest, which the
// caller must drain before the next readLine. It returns io.EOF when there are
// no more lines.
func (s *Scanner) readLine() (*restReader, error) {
	s.keep = s.keep[:0]
	first := true
	for {
		piece, end, err := s.chunk()
		if err == io.EOF {
			if first {
				return nil, io.EOF
			}
			return nil, nil // the last line ended at EOF, exactly on a piece boundary
		}
		if err != nil {
			return nil, err
		}
		first = false

		if room := keepCap - len(s.keep); len(piece) > room {
			s.keep = append(s.keep, piece[:room]...)
			return &restReader{s: s, buf: piece[room:], done: end}, nil
		}
		s.keep = append(s.keep, piece...)
		if end {
			return nil, nil
		}
		if len(s.keep) == keepCap {
			return &restReader{s: s}, nil
		}
	}
}

// chunk returns the next piece of the current line.
//
// end reports that the line is over, and the piece is then its last one, with
// the terminator and one trailing "\r" removed. A piece that is not the last
// never ends in "\r": that byte is pushed back, so it is either the trailing
// "\r" of the line or a regular byte, and the next piece can tell which.
// It returns io.EOF, with no piece, when the stream has no more bytes.
func (s *Scanner) chunk() (piece []byte, end bool, err error) {
	c, err := s.br.ReadSlice('\n')
	switch err {
	case nil:
		return trimCR(c[:len(c)-1]), true, nil
	case bufio.ErrBufferFull:
		if c[len(c)-1] == '\r' {
			_ = s.br.UnreadByte() // cannot fail right after ReadSlice
			c = c[:len(c)-1]
		}
		return c, false, nil
	case io.EOF:
		if len(c) == 0 {
			return nil, true, io.EOF
		}
		return trimCR(c), true, nil
	default:
		return nil, true, err
	}
}

func trimCR(b []byte) []byte {
	if n := len(b); n > 0 && b[n-1] == '\r' {
		return b[:n-1]
	}
	return b
}

// matchLong matches re against the whole line: what is in s.keep followed by
// what tail still has to give. It always consumes the rest of the line.
func (s *Scanner) matchLong(re *regexp.Regexp, tail *restReader) (bool, error) {
	if tail == nil {
		tail = &restReader{s: s, done: true}
	}
	src := bufio.NewReaderSize(io.MultiReader(bytes.NewReader(s.keep), tail), readSize)
	matched := re.MatchReader(src)
	// The regexp package takes any read error for the end of the text, so a
	// failed read could have produced a match on a partial line.
	if tail.err != nil {
		return false, tail.err
	}
	if _, err := io.Copy(io.Discard, tail); err != nil {
		return false, err
	}
	return matched, nil
}

// restReader yields the pieces of the current line that did not fit in
// Scanner.keep, then io.EOF.
type restReader struct {
	s    *Scanner
	buf  []byte
	done bool
	err  error
}

func (r *restReader) Read(p []byte) (int, error) {
	for len(r.buf) == 0 {
		if r.done {
			if r.err != nil {
				return 0, r.err
			}
			return 0, io.EOF
		}
		piece, end, err := r.s.chunk()
		switch {
		case err == io.EOF:
			r.done = true
		case err != nil:
			r.done, r.err = true, err
		default:
			r.buf, r.done = piece, end
		}
	}
	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}
