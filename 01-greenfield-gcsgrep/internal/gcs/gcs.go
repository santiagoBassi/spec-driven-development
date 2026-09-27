// Package gcs habla con Google Cloud Storage: verifica credenciales ADC,
// arma el cliente, lista objetos con tope y expone metadata y lectura por
// streaming. No sabe qué es un match; eso es de internal/search.
package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// ErrNoCredentials es el error fijo de FR-35: no hay ADC disponibles.
var ErrNoCredentials = errors.New("no Application Default Credentials found (run: gcloud auth application-default login)")

// Credentials busca las Application Default Credentials con scope de solo
// lectura (BR-1). Se llama siempre, antes de crear el cliente, aunque esté
// definido STORAGE_EMULATOR_HOST (FR-35).
func Credentials(ctx context.Context) (*google.Credentials, error) {
	creds, err := google.FindDefaultCredentials(ctx, storage.ScopeReadOnly)
	if err != nil {
		return nil, ErrNoCredentials
	}
	return creds, nil
}

// NewClient arma el cliente de storage. Con STORAGE_EMULATOR_HOST definido,
// el cliente de la librería ya se construye sin autenticación (las
// credenciales ya se comprobaron en Credentials); si no, usa las
// credenciales encontradas. Lecturas por la API JSON (para que el servidor
// de prueba de la Iteración 2 emule una sola API) y sin reintentos (FR-31).
func NewClient(ctx context.Context, creds *google.Credentials) (*storage.Client, error) {
	opts := []option.ClientOption{storage.WithJSONReads()}
	if os.Getenv("STORAGE_EMULATOR_HOST") == "" {
		opts = append(opts, option.WithCredentials(creds))
	}
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	client.SetRetry(storage.WithPolicy(storage.RetryNever))
	return client, nil
}

// ErrTooMany señala que el listado superó el tope vigente (BR-3): se vio el
// objeto N+1. Max es el tope vigente, para armar el mensaje de BR-3.
type ErrTooMany struct {
	Max int64
}

func (e *ErrTooMany) Error() string {
	return fmt.Sprintf("more than %d objects", e.Max)
}

// List devuelve los nombres de los objetos bajo bucket/prefix (prefix vacío
// = todo el bucket), sin importar si son texto o no (BR-3 cuenta todos).
// Se corta al ver el objeto max+1, sin pedir otra página (BR-3): devuelve
// ErrTooMany en ese caso, y no sigue leyendo el resto del listado.
func List(ctx context.Context, client *storage.Client, bucket, prefix string, max int64) ([]string, error) {
	q := &storage.Query{Prefix: prefix}
	if err := q.SetAttrSelection([]string{"Name"}); err != nil {
		return nil, err
	}
	it := client.Bucket(bucket).Objects(ctx, q)
	names := make([]string, 0, 64)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if int64(len(names)) >= max {
			return nil, &ErrTooMany{Max: max}
		}
		names = append(names, attrs.Name)
	}
	return names, nil
}

// Stat consulta la metadata de un objeto puntual.
func Stat(ctx context.Context, client *storage.Client, bucket, object string) (*storage.ObjectAttrs, error) {
	return client.Bucket(bucket).Object(object).Attrs(ctx)
}

// Open abre un lector por streaming del contenido de un objeto. El llamador
// debe cerrarlo.
func Open(ctx context.Context, client *storage.Client, bucket, object string) (io.ReadCloser, error) {
	return client.Bucket(bucket).Object(object).NewReader(ctx)
}

// IsAccessDenied reporta si err es un 403 de la API (BR-2).
func IsAccessDenied(err error) bool {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == 403
	}
	return false
}
