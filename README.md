# gcsgrep

`grep` sobre el contenido de objetos de Google Cloud Storage, sin bajarlos a disco, sin
escribir nada en GCS y con las credenciales de quien lo invoca.

```
gcsgrep [-E] [-i] [-n] [--max <N>|unlimited] [--] <patrón> <ubicación>
```

- Ubicaciones `gs://bucket/`, `gs://bucket/prefijo/` (recursivas) y `gs://bucket/objeto`.
- Patrón literal; con `-E`, regex RE2. `-i` ignora mayúsculas (también en `ñ`/`Ñ`, `á`/`Á`).
  `-n` numera las líneas.
- Lectura por streaming, solo lectura, autenticación por ADC.
- Tope de 1000 objetos por corrida (`--max`), que aborta antes de leer nada.
- Exit codes de `grep`: `0` hubo match, `1` no hubo, `2` error.

## Uso

```bash
go build -o gcsgrep ./cmd/gcsgrep
./gcsgrep timeout gs://<bucket>/logs/
```

## Verificación

```bash
go vet ./... && go test ./internal/...      # unitarios, sin red

export GCSGREP_E2E_BUCKET=sdd-fardenghi-itba
export GCSGREP_E2E_LECTORA=$PWD/lectora.json
export GCSGREP_E2E_SIN_ACCESO=$PWD/sin-acceso.json
go test ./e2e -count=1 -v                    # un test por VC, contra GCS real
```

La suite e2e necesita las claves de las identidades de prueba (no se versionan) y `gcloud`
en el PATH.

## Documentación

Los artefactos del proceso SDD están en [`sdd/`](./sdd):

| Archivo | Qué tiene |
|---|---|
| [`gcsgrep-spec.md`](./sdd/gcsgrep-spec.md) | Spec revisada: requisitos y VCs |
| [`gcsgrep-plan.md`](./sdd/gcsgrep-plan.md) | Plan de iteraciones |
| [`iterations/01-busqueda-punta-a-punta.md`](./sdd/iterations/01-busqueda-punta-a-punta.md) | Registro de la Iteración 1: comandos, resultados por VC y desvíos |
| [`gcsgrep-cobertura-vc.md`](./sdd/gcsgrep-cobertura-vc.md) | Tabla de cobertura: con qué se ejercita cada VC y qué se observó |
| [`DECISIONS.md`](./sdd/DECISIONS.md) | Decisiones no obvias tomadas al construir |
| [`CONTEXT.md`](./sdd/CONTEXT.md) | Estado actual, mapa de archivos y comandos |