package e2e

import "testing"

// FR-35
func TestVC35(t *testing.T) {
	ln := newListener(t)
	r := execute(t, noADCEnv(t, ln.addr()), "timeout", gs(t, "logs/"))

	requireExit(t, r, 2)
	requireStdout(t, r, "")
	requireStderr(t, r, "gcsgrep: no Application Default Credentials found (run: gcloud auth application-default login)\n")
	ln.requireNoConnections(t)
}

// BR-2
func TestVC40(t *testing.T) {
	t.Run("sin-acceso is denied a listing", func(t *testing.T) {
		r := asSinAcceso(t, "timeout", gs(t, "logs/app/"))
		requireExit(t, r, 2)
		requireStdout(t, r, "")
		requireStderr(t, r, "gcsgrep: access denied: "+gs(t, "logs/app/")+"\n")
	})
	t.Run("sin-acceso is denied a single object", func(t *testing.T) {
		r := asSinAcceso(t, "timeout", gs(t, "logs/app/api.log"))
		requireExit(t, r, 2)
		requireStdout(t, r, "")
		requireStderr(t, r, "gcsgrep: access denied: "+gs(t, "logs/app/api.log")+"\n")
	})
	t.Run("lectora searches", func(t *testing.T) {
		r := asLectora(t, "timeout", gs(t, "logs/app/"))
		requireExit(t, r, 0)
		requireLines(t, r, vc1Lines(t))
	})
}
