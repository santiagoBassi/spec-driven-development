// Package cli parsea argv y valida flags, patrón (longitud) y cantidad de
// posicionales, sin tocar GCS: todo lo que se puede decidir con argv se
// decide acá. El patrón (compilado como regex) y la ubicación se validan
// después, con internal/search y internal/location, en ese orden.
package cli

import (
	"fmt"
	"math"
	"strconv"
)

// Unlimited es el valor de Config.Max cuando --max unlimited desactivó el
// tope (BR-4): un tope que nunca se alcanza.
const Unlimited int64 = math.MaxInt64

// defaultMax es el tope por defecto (BR-3): 1000 objetos listados.
const defaultMax int64 = 1000

// Config es el resultado de un parseo válido.
type Config struct {
	Extended   bool // -E
	IgnoreCase bool // -i
	LineNumber bool // -n
	Max        int64
	Pattern    string
	Location   string
}

// ErrUsage es un error de uso (exit 2): el mensaje ya está formado, sin el
// prefijo "gcsgrep: " (eso lo agrega quien imprime el error).
type ErrUsage struct{ msg string }

func (e *ErrUsage) Error() string { return e.msg }

func usage(format string, args ...interface{}) error {
	return &ErrUsage{msg: fmt.Sprintf(format, args...)}
}

// Parse interpreta los argumentos de la invocación (sin el nombre del
// programa). Devuelve un error de uso ante el primer problema, en este
// orden: flags (en el orden de argv) → cantidad de posicionales → patrón
// vacío. El patrón inválido como regex y la ubicación inválida se
// comprueban después, fuera de este paquete.
func Parse(args []string) (Config, error) {
	var extended, ignoreCase, lineNumber bool
	max := defaultMax
	positional := make([]string, 0, 2)
	seenDoubleDash := false

	for i := 0; i < len(args); i++ {
		a := args[i]
		if !seenDoubleDash && a == "--" {
			seenDoubleDash = true
			continue
		}
		if !seenDoubleDash && len(a) > 0 && a[0] == '-' {
			switch a {
			case "-E":
				extended = true
			case "-i":
				ignoreCase = true
			case "-n":
				lineNumber = true
			case "--max":
				i++
				if i >= len(args) {
					return Config{}, usage("flag --max requires a value")
				}
				v := args[i]
				n, ok := parseMax(v)
				if !ok {
					return Config{}, usage("invalid value for --max: %q (integer >= 1 or unlimited)", v)
				}
				max = n
			default:
				return Config{}, usage("unknown flag: %q", a)
			}
			continue
		}
		positional = append(positional, a)
	}

	if len(positional) != 2 {
		return Config{}, usage("expected 2 arguments (pattern and location), got %d", len(positional))
	}
	pattern, location := positional[0], positional[1]
	if pattern == "" {
		return Config{}, usage("empty pattern")
	}

	return Config{
		Extended:   extended,
		IgnoreCase: ignoreCase,
		LineNumber: lineNumber,
		Max:        max,
		Pattern:    pattern,
		Location:   location,
	}, nil
}

// parseMax interpreta el valor de --max (BR-4): "unlimited", o solo dígitos
// decimales representando un entero >= 1 (un signo o un espacio lo
// invalidan). Un valor que desborda el entero devuelto es válido: es un
// tope que nunca se alcanza (Unlimited).
func parseMax(v string) (int64, bool) {
	if v == "unlimited" {
		return Unlimited, true
	}
	if v == "" {
		return 0, false
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		// El valor ya se validó dígito a dígito y no vacío: la única forma
		// de fallar acá es desbordar uint64.
		return Unlimited, true
	}
	if n == 0 {
		return 0, false
	}
	if n > uint64(math.MaxInt64) {
		return Unlimited, true
	}
	return int64(n), true
}
