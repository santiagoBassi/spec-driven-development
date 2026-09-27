package e2e

import "testing"

// TestVC39 cubre BR-1: toda la suite corre con lectora (solo
// roles/storage.objectViewer) y $B no cambia. El nombre del archivo
// (zz_) hace que este test corra último dentro del paquete (los archivos
// de un mismo paquete se ordenan alfabéticamente), así compara contra el
// estado real al final de la corrida completa.
func TestVC39(t *testing.T) {
	after, err := snapshotBucket()
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if after != vc39Before {
		t.Errorf("BR-1: el contenido de %s cambió durante la suite.\nantes:\n%s\ndespués:\n%s", bucket, vc39Before, after)
	}
}
