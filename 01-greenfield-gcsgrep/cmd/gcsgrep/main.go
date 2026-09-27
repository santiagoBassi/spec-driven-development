// Command gcsgrep busca texto en el contenido de objetos de Google Cloud
// Storage, con la experiencia de grep. Ver sdd/gcsgrep-spec.md.
package main

import "os"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
