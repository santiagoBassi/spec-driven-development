// Package location interpreta gs://bucket/[ruta] y decide el modo (bucket,
// prefijo u objeto), según la spec (sección "Ubicación"). No toca GCS.
package location

import (
	"fmt"
	"strings"
)

// Mode identifica qué se debe hacer con la ubicación.
type Mode int

const (
	// ModeBucket es gs://bucket/ : todo el bucket, recursivo (prefijo vacío).
	ModeBucket Mode = iota
	// ModePrefix es gs://bucket/prefijo/ : todo lo que está bajo el prefijo.
	ModePrefix
	// ModeObject es gs://bucket/objeto : un único objeto puntual.
	ModeObject
)

// Location es una ubicación gs:// ya interpretada.
type Location struct {
	Raw    string // el argumento tal como se recibió, para los mensajes de error
	Bucket string
	Path   string // prefijo (sin barra final) para ModeBucket/ModePrefix, o nombre de objeto para ModeObject
	Mode   Mode
}

// ErrInvalid se devuelve cuando la ubicación no tiene la forma
// gs://<bucket>/[<ruta>] con un <bucket> no vacío.
type ErrInvalid struct {
	Raw string
}

func (e *ErrInvalid) Error() string {
	return fmt.Sprintf("invalid location: %q", e.Raw)
}

const scheme = "gs://"

// Parse interpreta raw como una ubicación gs://. FR-14 fija el mensaje de
// error para todo lo que no encaje.
func Parse(raw string) (Location, error) {
	if !strings.HasPrefix(raw, scheme) {
		return Location{}, &ErrInvalid{Raw: raw}
	}
	rest := raw[len(scheme):]
	slash := strings.IndexByte(rest, '/')
	var bucket, path string
	if slash < 0 {
		// gs://bucket sin barra: se rechaza, no hay excepción a la regla.
		return Location{}, &ErrInvalid{Raw: raw}
	}
	bucket = rest[:slash]
	path = rest[slash+1:]
	if bucket == "" {
		return Location{}, &ErrInvalid{Raw: raw}
	}

	if path == "" {
		return Location{Raw: raw, Bucket: bucket, Path: "", Mode: ModeBucket}, nil
	}
	if strings.HasSuffix(path, "/") {
		return Location{Raw: raw, Bucket: bucket, Path: strings.TrimSuffix(path, "/"), Mode: ModePrefix}, nil
	}
	return Location{Raw: raw, Bucket: bucket, Path: path, Mode: ModeObject}, nil
}

// ListPrefix es el prefijo a pedirle al listado de GCS. Para ModePrefix
// incluye la barra final, así solo matchean los objetos bajo ese prefijo
// exacto (FR-11: "logs/app/" no debe matchear "logs/app-old/..."); para
// ModeBucket es el prefijo vacío (FR-10).
func (l Location) ListPrefix() string {
	if l.Mode == ModeBucket || l.Path == "" {
		return ""
	}
	return l.Path + "/"
}
