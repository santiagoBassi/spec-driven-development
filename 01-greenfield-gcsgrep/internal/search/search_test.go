package search

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"
)

func TestCompileLiteral(t *testing.T) {
	re, err := Compile("job_id=10.", false, false)
	if err != nil {
		t.Fatalf("Compile: unexpected error: %v", err)
	}
	if re.MatchString("job_id=10.") == false {
		t.Errorf("literal pattern should match itself verbatim")
	}
	if re.MatchString("job_id=100") {
		t.Errorf("literal pattern must not treat '.' as a wildcard")
	}
}

func TestCompileExtended(t *testing.T) {
	re, err := Compile(`job_id=\d+ status=ERROR`, true, false)
	if err != nil {
		t.Fatalf("Compile: unexpected error: %v", err)
	}
	if !re.MatchString("job_id=102 status=ERROR") {
		t.Errorf("regex should match")
	}
}

func TestCompileInvalidPattern(t *testing.T) {
	_, err := Compile("(", true, false)
	if err == nil {
		t.Fatalf("Compile: expected error for invalid regex")
	}
	if !strings.HasPrefix(err.Error(), "invalid pattern: ") {
		t.Errorf("error = %q, want prefix %q", err.Error(), "invalid pattern: ")
	}
}

func TestCompileIgnoreCase(t *testing.T) {
	re, err := Compile("timeout", false, true)
	if err != nil {
		t.Fatalf("Compile: unexpected error: %v", err)
	}
	for _, s := range []string{"timeout", "TIMEOUT", "Timeout"} {
		if !re.MatchString(s) {
			t.Errorf("-i should match %q", s)
		}
	}
	// FR-7a: no depende del locale; Unicode simple case folding cubre ñ/Ñ y á/Á.
	reAccents, err := Compile("ñandú árbol", false, true)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !reAccents.MatchString("ÑANDÚ ÁRBOL") {
		t.Errorf("-i debe matchear letras no ASCII sin distinguir mayúsculas")
	}
}

// TestCompileEszett cubre FR-7b: el case folding simple de RE2 no pliega
// ß a SS.
func TestCompileEszett(t *testing.T) {
	re, err := Compile("strasse", false, true)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if re.MatchString("straße") {
		t.Errorf("-i no debe plegar straße a strasse (equivalencia de más de un carácter)")
	}
}

// scanLines es un helper que junta todas las líneas que matchean.
type matchedLine struct {
	n         int
	text      string
	truncated bool
}

func scanAll(t *testing.T, r io.Reader, re *regexp.Regexp) []matchedLine {
	t.Helper()
	var got []matchedLine
	s := NewScanner()
	if err := s.Scan(r, re, func(n int, text []byte, truncated bool) error {
		got = append(got, matchedLine{n, string(text), truncated})
		return nil
	}); err != nil {
		t.Fatalf("Scan: unexpected error: %v", err)
	}
	return got
}

func TestScanCRLF(t *testing.T) {
	re := mustCompile(t, "timeout", false, false)
	data := "Linea 1\r\nLinea 2 con timeout\r\nLinea 3\r\n"
	got := scanAll(t, strings.NewReader(data), re)
	want := []matchedLine{{2, "Linea 2 con timeout", false}}
	assertLines(t, got, want)
}

func TestScanCRNotFollowedByLF(t *testing.T) {
	// Un \r que no precede a \n es parte de la línea (no se recorta).
	re := mustCompile(t, `a\rb`, true, false)
	data := "a\rb\n"
	got := scanAll(t, strings.NewReader(data), re)
	want := []matchedLine{{1, "a\rb", false}}
	assertLines(t, got, want)
}

func TestScanTrailingLoneCR(t *testing.T) {
	// Un \r como último byte del stream, sin nada después, es literal (no
	// hay \n que lo convierta en fin de línea).
	re := mustCompile(t, "x", false, false)
	data := "x\r"
	got := scanAll(t, strings.NewReader(data), re)
	want := []matchedLine{{1, "x\r", false}}
	assertLines(t, got, want)
}

func TestScanNoTrailingNewline(t *testing.T) {
	re := mustCompile(t, "salto final$", true, false)
	data := "Primera linea con salto\nUltima linea sin salto final"
	got := scanAll(t, strings.NewReader(data), re)
	want := []matchedLine{{2, "Ultima linea sin salto final", false}}
	assertLines(t, got, want)
}

func TestScanEmptyObject(t *testing.T) {
	re := mustCompile(t, "x", false, false)
	s := NewScanner()
	var calls int
	if err := s.Scan(strings.NewReader(""), re, func(int, []byte, bool) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if calls != 0 {
		t.Errorf("un objeto vacío no debe producir líneas, se llamó a fn %d veces", calls)
	}
}

func TestScanExactlyMaxLineNotTruncated(t *testing.T) {
	re := mustCompile(t, "^a+$", true, false)
	line := strings.Repeat("a", MaxLine)
	got := scanAll(t, strings.NewReader(line+"\n"), re)
	if len(got) != 1 {
		t.Fatalf("got %d matches, want 1", len(got))
	}
	if got[0].truncated {
		t.Errorf("una línea de exactamente MaxLine bytes no se trunca")
	}
	if len(got[0].text) != MaxLine {
		t.Errorf("text len = %d, want %d", len(got[0].text), MaxLine)
	}
}

func TestScanOneByteOverMaxLineIsTruncated(t *testing.T) {
	re := mustCompile(t, "^a+b$", true, false)
	line := strings.Repeat("a", MaxLine) + "b"
	got := scanAll(t, strings.NewReader(line+"\n"), re)
	if len(got) != 1 {
		t.Fatalf("got %d matches, want 1", len(got))
	}
	if !got[0].truncated {
		t.Errorf("una línea de MaxLine+1 bytes debe truncarse")
	}
	if got[0].text != strings.Repeat("a", MaxLine) {
		t.Errorf("el texto retenido debe ser exactamente los primeros MaxLine bytes")
	}
}

// TestScanEndAnchorPastLimit reproduce el falso positivo que evita D-10: un
// $ no puede matchear sobre el prefijo retenido si el final real de la
// línea es distinto.
func TestScanEndAnchorPastLimit(t *testing.T) {
	line := strings.Repeat("a", MaxLine) + "b" // termina en 'b', no en 'a'

	reA := mustCompile(t, "a$", true, false)
	gotA := scanAll(t, strings.NewReader(line+"\n"), reA)
	if len(gotA) != 0 {
		t.Fatalf("a$ no debe matchear: la línea real termina en 'b', got %v", gotA)
	}

	reB := mustCompile(t, "b$", true, false)
	gotB := scanAll(t, strings.NewReader(line+"\n"), reB)
	if len(gotB) != 1 {
		t.Fatalf("b$ debe matchear: la línea real termina en 'b', got %v", gotB)
	}
}

// TestScanWordBoundaryAcrossLimit cubre \b para una palabra que cruza el
// límite de MaxLine bytes.
func TestScanWordBoundaryAcrossLimit(t *testing.T) {
	prefix := strings.Repeat("a", MaxLine-2) + " wo"
	suffix := "rd zzzzzzzzzz"
	line := prefix + suffix // "word" cruza el byte MaxLine
	re := mustCompile(t, `\bword\b`, true, false)
	got := scanAll(t, strings.NewReader(line+"\n"), re)
	if len(got) != 1 {
		t.Fatalf(`\bword\b debe matchear cruzando el límite, got %v`, got)
	}
}

// TestScanMatchAfterTruncation cubre BR-8: un match después del primer MiB
// se reporta igual, aunque no sea visible en la salida truncada.
func TestScanMatchAfterTruncation(t *testing.T) {
	line := strings.Repeat("a", MaxLine+100) + "NEEDLE"
	re := mustCompile(t, "NEEDLE", false, false)
	got := scanAll(t, strings.NewReader(line+"\n"), re)
	if len(got) != 1 {
		t.Fatalf("got %d matches, want 1", len(got))
	}
	if !got[0].truncated {
		t.Errorf("truncated debe ser true")
	}
	if strings.Contains(got[0].text, "NEEDLE") {
		t.Errorf("el texto truncado no debe incluir el match posterior al límite")
	}
}

// TestScanReadErrorMidLine comprueba que un error de lectura (no io.EOF) se
// devuelve tal cual, sin evaluarlo como si fuera match.
func TestScanReadErrorMidLine(t *testing.T) {
	boom := errors.New("boom")
	r := io.MultiReader(strings.NewReader("first ok\n"), &errAfterReader{err: boom, data: []byte("partial")})
	re := mustCompile(t, ".*", true, false)
	s := NewScanner()
	var got []matchedLine
	err := s.Scan(r, re, func(n int, text []byte, truncated bool) error {
		got = append(got, matchedLine{n, string(text), truncated})
		return nil
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Scan error = %v, want %v", err, boom)
	}
	if len(got) != 1 || got[0].text != "first ok" {
		t.Errorf("la línea completa antes del error debe reportarse igual, got %v", got)
	}
}

// TestScanReadErrorDuringLongLine ejercita el mismo caso pero con una línea
// que ya pasó a modo streaming (> MaxLine bytes).
func TestScanReadErrorDuringLongLine(t *testing.T) {
	boom := errors.New("boom")
	long := bytes.Repeat([]byte("a"), MaxLine+10)
	r := io.MultiReader(bytes.NewReader(long), &errAfterReader{err: boom, data: []byte("more")})
	re := mustCompile(t, ".*", true, false)
	s := NewScanner()
	err := s.Scan(r, re, func(int, []byte, bool) error { return nil })
	if !errors.Is(err, boom) {
		t.Fatalf("Scan error = %v, want %v", err, boom)
	}
}

// errAfterReader entrega data y después siempre falla con err.
type errAfterReader struct {
	data []byte
	err  error
	sent bool
}

func (r *errAfterReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		n := copy(p, r.data)
		return n, nil
	}
	return 0, r.err
}

func mustCompile(t *testing.T, pattern string, extended, ignoreCase bool) *regexp.Regexp {
	t.Helper()
	re, err := Compile(pattern, extended, ignoreCase)
	if err != nil {
		t.Fatalf("Compile(%q): %v", pattern, err)
	}
	return re
}

func assertLines(t *testing.T, got, want []matchedLine) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("line %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}
