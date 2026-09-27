package e2e

import (
	"fmt"
	"strings"
	"testing"
)

// TestVC12Parcial es la evidencia de la Iteración 1 para VC-12 (FR-12): un
// objeto puntual busca solo en ese objeto. La otra mitad (que no haya
// request de listado y que no se lea a.log.bak) necesita el servidor de
// prueba y cierra en la Iteración 2.
func TestVC12Parcial(t *testing.T) {
	loc := "gs://" + bucket + "/logs/app/api.log"
	r := runTool(t, runOpts{args: []string{"timeout", loc}, creds: lectoraCreds})
	requireExit(t, r, 0)
	lines := stdoutLines(r.stdout)
	if len(lines) == 0 {
		t.Fatalf("se esperaban matches en api.log")
	}
	prefix := "gs://" + bucket + "/logs/app/api.log:"
	for _, l := range lines {
		if !strings.HasPrefix(l, prefix) {
			t.Errorf("línea fuera de api.log: %q", l)
		}
	}
}

// TestVC41Parcial es la evidencia de la Iteración 1 para VC-41 (BR-3): el
// guardrail aborta antes de leer al superar el tope, con el mensaje exacto.
// Falta observar el tope por defecto de 1000 sin paginar de más, que
// necesita el servidor de prueba (Iteración 2).
func TestVC41Parcial(t *testing.T) {
	loc := "gs://" + bucket + "/edge-cases/"
	r := runTool(t, runOpts{args: []string{"--max", "5", "timeout", loc}, creds: lectoraCreds})
	requireExit(t, r, 2)
	if len(r.stdout) != 0 {
		t.Errorf("stdout = %q, want empty", r.stdout)
	}
	want := fmt.Sprintf("gcsgrep: more than 5 objects under %s; use --max <N> or --max unlimited\n", loc)
	if string(r.stderr) != want {
		t.Errorf("stderr = %q, want %q", r.stderr, want)
	}

	r2 := runTool(t, runOpts{args: []string{"--max", "6", "timeout", loc}, creds: lectoraCreds})
	requireExit(t, r2, 0)
}

// TestVC42Parcial es la evidencia de la Iteración 1 para VC-42 (BR-4): los
// valores inválidos de --max, --max sin valor (con el chequeo P en ambos
// casos) y un N que desborda el entero representable como tope que nunca se
// alcanza. Falta observar --max unlimited leyendo 2500 objetos, que necesita
// el servidor de prueba (Iteración 2).
func TestVC42Parcial(t *testing.T) {
	loc := "gs://" + bucket + "/logs/"
	for _, v := range []string{"0", "-3", "abc", "1.5", "+5", " 5"} {
		requireUsageError(t, []string{"--max", v, "timeout", loc}, fmt.Sprintf("gcsgrep: invalid value for --max: %q (integer >= 1 or unlimited)", v))
	}
	requireUsageError(t, []string{"timeout", loc, "--max"}, "gcsgrep: flag --max requires a value")

	r := runTool(t, runOpts{args: []string{"--max", "99999999999999999999", "timeout", "gs://" + bucket + "/logs/app/"}, creds: lectoraCreds})
	requireExit(t, r, 0)
}
