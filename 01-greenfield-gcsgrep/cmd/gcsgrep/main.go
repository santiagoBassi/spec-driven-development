// Command gcsgrep searches the content of Google Cloud Storage objects with the
// feel of grep, without downloading them.
package main

import (
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
