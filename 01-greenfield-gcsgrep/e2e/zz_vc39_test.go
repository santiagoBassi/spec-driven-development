package e2e

import (
	"strings"
	"testing"
)

// BR-1 (VC-39). The file name starts with zz_ so that this runs after every
// other test in the package: it compares the bucket with how it was before the
// suite, so it only means something in a full `go test ./e2e` run.
//
// Every test above already ran with ADC = lectora, which has only
// roles/storage.objectViewer on $B: if the tool tried to write, GCS would have
// refused and that test would have failed.
func TestVC39(t *testing.T) {
	if snapshotErr != nil {
		t.Fatalf("snapshot before the suite: %v", snapshotErr)
	}
	after, err := takeSnapshot()
	if err != nil {
		t.Fatalf("snapshot after the suite: %v", err)
	}
	if snapshotBefore == "" {
		t.Fatal("the bucket listing before the suite is empty")
	}
	if after != snapshotBefore {
		t.Errorf("the bucket changed during the suite\nbefore:\n%s\nafter:\n%s", snapshotBefore, after)
		return
	}
	t.Logf("%d objects in the bucket, identical (name, generation, metageneration) before and after the suite",
		len(strings.Split(after, "\n")))
}
