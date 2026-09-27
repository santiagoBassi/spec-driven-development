// Package search compila el patrón (FR-3/FR-4/FR-7a) y delimita líneas en un
// stream de bytes, buscando cada línea completa y reteniendo como máximo su
// primer MiB para la salida (BR-8). No sabe que existe GCS: recibe un
// io.Reader.
package search

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"unicode/utf8"
)

// MaxLine es el límite de BR-8: 1 MiB (1 048 576 bytes).
const MaxLine = 1 << 20

// retainCap es un byte más que MaxLine: distingue una línea de exactamente
// MaxLine bytes (no se trunca) de una más larga (si el byte extra existe).
const retainCap = MaxLine + 1

// ErrInvalidPattern es el error de FR-5: el patrón (regex, con -E) no es una
// regex RE2 válida.
type ErrInvalidPattern struct {
	Detail string
}

func (e *ErrInvalidPattern) Error() string {
	return fmt.Sprintf("invalid pattern: %s", e.Detail)
}

// Compile arma la regex a usar para buscar. Sin extended, pattern se toma
// literal (FR-3): regexp.QuoteMeta lo escapa entero, así que ningún patrón
// no vacío es inválido. Con extended, pattern es una regex RE2 (FR-4/FR-5).
// Con ignoreCase, se antepone (?i) (case folding simple de Unicode, FR-7a),
// pero recién después de comprobar que el patrón compila solo: así el
// detalle de una regex inválida cita lo que la persona escribió.
func Compile(pattern string, extended, ignoreCase bool) (*regexp.Regexp, error) {
	expr := pattern
	if !extended {
		expr = regexp.QuoteMeta(pattern)
	}
	if _, err := regexp.Compile(expr); err != nil {
		return nil, &ErrInvalidPattern{Detail: err.Error()}
	}
	if ignoreCase {
		expr = "(?i)" + expr
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, &ErrInvalidPattern{Detail: err.Error()}
	}
	return re, nil
}

// LineFunc se llama una vez por línea que matchea, en orden, con el número
// de línea (1-based), el texto a imprimir (ya sin el \r final, recortado a
// MaxLine bytes si la línea era más larga) y si se truncó.
type LineFunc func(lineNo int, text []byte, truncated bool) error

// Scanner delimita líneas de un stream, recortando el \r final de las
// líneas \r\n (FR-9a) y buscando la última línea aunque no termine en \n
// (FR-9b). Se reusa entre objetos con Scan: mismo buffer retenido, sin
// volver a reservar memoria por objeto.
type Scanner struct {
	br     *bufio.Reader
	retain []byte
}

// NewScanner crea un Scanner listo para usarse con Scan.
func NewScanner() *Scanner {
	return &Scanner{
		br:     bufio.NewReaderSize(emptyReader{}, 64*1024),
		retain: make([]byte, 0, retainCap),
	}
}

type emptyReader struct{}

func (emptyReader) Read(p []byte) (int, error) { return 0, io.EOF }

// Scan lee r línea por línea. Por cada línea que matchea re, llama a fn con
// su número (desde 1), su texto a imprimir y si se truncó. Un error de
// lectura (que no sea el fin normal del stream) corta el escaneo y se
// devuelve tal cual.
func (s *Scanner) Scan(r io.Reader, re *regexp.Regexp, fn LineFunc) error {
	s.br.Reset(r)
	lineNo := 0
	for {
		ls := &lineState{br: s.br}
		s.retain = s.retain[:0]
		for len(s.retain) < retainCap {
			b, ok := ls.nextByte()
			if !ok {
				break
			}
			s.retain = append(s.retain, b)
		}
		if !ls.sawInput {
			// Fin limpio del objeto: no quedó ninguna línea parcial.
			return ls.err
		}

		truncated := len(s.retain) > MaxLine
		var matched bool
		if ls.ended {
			// La línea completa entró en el buffer retenido.
			matched = re.Match(s.retain)
		} else {
			// Línea más larga que retainCap: se busca en el resto por
			// streaming, sin cargarla entera en memoria, y siempre se
			// consume hasta el final de la línea (aunque el match ya se
			// haya decidido antes).
			rr := &runeReader{retain: s.retain, ls: ls}
			matched = re.MatchReader(rr)
			for {
				if _, ok := rr.readByte(); !ok {
					break
				}
			}
		}
		if ls.err != nil {
			return ls.err
		}

		lineNo++
		if matched {
			text := s.retain
			if truncated {
				text = s.retain[:MaxLine]
			}
			if err := fn(lineNo, text, truncated); err != nil {
				return err
			}
		}
	}
}

// lineState delimita una línea: entrega sus bytes con el \r de \r\n ya
// recortado, y distingue el fin de línea (\n consumido) del fin del stream.
type lineState struct {
	br          *bufio.Reader
	hasPushback bool
	pushback    byte
	sawInput    bool // se leyó al menos un byte crudo para esta línea
	ended       bool // \n consumido o EOF: no hay más bytes de esta línea
	err         error
}

// nextByte devuelve el siguiente byte lógico de la línea (ya sin \r cuando
// precede a \n), o ok=false cuando la línea terminó. ls.err queda con un
// error de lectura real (no io.EOF), si lo hubo.
func (ls *lineState) nextByte() (b byte, ok bool) {
	if ls.ended {
		return 0, false
	}
	var c byte
	if ls.hasPushback {
		c = ls.pushback
		ls.hasPushback = false
		ls.sawInput = true
	} else {
		rc, err := ls.br.ReadByte()
		if err != nil {
			ls.ended = true
			if err != io.EOF {
				ls.err = err
			}
			return 0, false
		}
		ls.sawInput = true
		c = rc
	}
	if c == '\n' {
		ls.ended = true
		return 0, false
	}
	if c != '\r' {
		return c, true
	}
	// c == '\r': hay que ver el siguiente byte para decidir si es \r\n.
	n, err := ls.br.ReadByte()
	if err != nil {
		ls.ended = true
		if err != io.EOF {
			ls.err = err
		}
		return '\r', true
	}
	if n == '\n' {
		ls.ended = true
		return 0, false
	}
	ls.pushback = n
	ls.hasPushback = true
	return '\r', true
}

// runeReader implementa io.RuneReader sobre el buffer retenido seguido del
// resto de la línea, tomado de ls a demanda, decodificando UTF-8 un rune a
// la vez sin leer más bytes de los necesarios. Así, drenar lo que
// MatchReader no haya consumido (llamando a readByte hasta el final) ve
// todos los bytes de la línea, sin que un buffer interno se los quede.
type runeReader struct {
	retain  []byte
	pos     int
	ls      *lineState
	pending []byte // bytes ya leídos de la fuente pero no consumidos por un rune
}

func (rr *runeReader) readByte() (byte, bool) {
	if len(rr.pending) > 0 {
		b := rr.pending[0]
		rr.pending = rr.pending[1:]
		return b, true
	}
	if rr.pos < len(rr.retain) {
		b := rr.retain[rr.pos]
		rr.pos++
		return b, true
	}
	return rr.ls.nextByte()
}

func (rr *runeReader) ReadRune() (r rune, size int, err error) {
	b0, ok := rr.readByte()
	if !ok {
		if rr.ls.err != nil {
			return 0, 0, rr.ls.err
		}
		return 0, 0, io.EOF
	}
	if b0 < utf8.RuneSelf {
		return rune(b0), 1, nil
	}
	var buf [utf8.UTFMax]byte
	buf[0] = b0
	n := 1
	for n < utf8.UTFMax && !utf8.FullRune(buf[:n]) {
		b, ok := rr.readByte()
		if !ok {
			break
		}
		buf[n] = b
		n++
	}
	r, size = utf8.DecodeRune(buf[:n])
	if size < n {
		leftover := append([]byte(nil), buf[size:n]...)
		rr.pending = append(leftover, rr.pending...)
	}
	return r, size, nil
}
