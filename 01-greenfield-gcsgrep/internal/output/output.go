// Package output escribe registros completos a stdout y avisos a stderr
// (BR-9): stdout contiene solo resultados, y todo error o aviso va a
// stderr con el prefijo "gcsgrep: ".
package output

import (
	"fmt"
	"io"
	"sync"
)

// Printer escribe cada registro con un único Write, bajo mutex. Con un
// solo worker (N=1, Iteración 1) el mutex no hace falta para que las
// líneas no se corten, pero ya se escribe así para no reabrir esto cuando
// la Iteración 4 suba N.
type Printer struct {
	mu sync.Mutex
	w  io.Writer
}

// NewPrinter crea un Printer que escribe en w.
func NewPrinter(w io.Writer) *Printer {
	return &Printer{w: w}
}

// Print escribe exactamente los bytes de line, en un único Write. line debe
// incluir su propio terminador si lo lleva.
func (p *Printer) Print(line []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err := p.w.Write(line)
	return err
}

// Warn escribe "gcsgrep: <msg>\n" en w (BR-9: todo aviso empieza con
// "gcsgrep: ").
func Warn(w io.Writer, msg string) error {
	_, err := fmt.Fprintf(w, "gcsgrep: %s\n", msg)
	return err
}
