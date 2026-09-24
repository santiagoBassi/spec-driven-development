# gcsgrep — contexto

> Estado actual, mapa de archivos y comandos. Se actualiza al cerrar cada iteración.
> Las decisiones no obvias viven en [`DECISIONS.md`](./DECISIONS.md); el contrato, en
> [`gcsgrep-spec.md`](./gcsgrep-spec.md); el orden de construcción, en
> [`gcsgrep-plan.md`](./gcsgrep-plan.md).

## Estado

**Iteración 1 cerrada.** Sus 16 VCs pasan contra GCS real (2026-09-23). El registro, con
comandos, resultados, desvíos y handoff, está en
[`iterations/01-busqueda-punta-a-punta.md`](./iterations/01-busqueda-punta-a-punta.md). La
tabla por VC está en [`gcsgrep-cobertura-vc.md`](./gcsgrep-cobertura-vc.md).

| Iteración | Estado |
|---|---|
| 1 · Búsqueda de punta a punta | **Cerrada.** 16 de 16 VCs pasan |
| 2 · Servidor de prueba, sniffing, `-c`/`-l` | Siguiente. No empezada |
| 3 a 5 | No empezadas |

**Spec revisada de nuevo tras cerrar la Iteración 1** (sigue en 50 requisitos y 50 VCs):
- FR-13 y FR-16 informan un recurso inexistente con un solo mensaje,
  `gcsgrep: not found: <ubicación>` (D-17).
- BR-4 y FR-26 fijan qué pasa con `--max` o `--concurrency` sin valor (D-18).

El **código no cambió**: sigue con los mensajes provisionales de D-13, que la Iteración 3
reemplaza. El registro `iterations/01` no se toca (es inmutable): lo aprendido vive acá, en
`DECISIONS.md` y en el plan.

## Qué hace hoy

```
gcsgrep [-E] [-i] [-n] [--max <N>|unlimited] [--] <patrón> <ubicación>
```

- Ubicaciones `gs://bucket/`, `gs://bucket/prefijo/` y `gs://bucket/objeto`.
- Patrón literal; con `-E`, RE2. `-i` con case folding de Unicode. Líneas `\r\n` recortadas.
- Tope de 1000 objetos (`--max`), que aborta antes de leer nada. Solo ADC. Solo lectura.
- Exit codes de `grep`: `0` match, `1` sin match, `2` error.

**Todavía no** (planeado): `-c`, `-l`, sniffing de binarios (los no-texto se leen como
bytes), marcadores de carpeta, el mensaje `not found` para un recurso inexistente (hoy sale
como `list error` o `metadata error`, D-13), timeouts, concurrencia, progreso, flags
combinados o repetidos y `--concurrency`. Ver el plan.

## Mapa de archivos

```
cmd/gcsgrep/main.go        main: os.Exit(run(...))
cmd/gcsgrep/run.go         orquesta: parseo → ADC → resolver objetos → pool de lectura → exit code
internal/cli/              Parse(argv) → Config | UsageError; valida sin tocar GCS
internal/location/         gs://bucket/[ruta] → modo prefijo (incluye bucket) u objeto
internal/gcs/              ADC, cliente sin reintentos, List con tope, Stat, Open
internal/search/           Compile del patrón; Scanner: líneas, recorte de \r, BR-8
internal/output/           Printer (un Write por registro, con mutex) y Errorf
e2e/                       un test por VC contra ./gcsgrep (ver DECISIONS D-15)
test-fixtures/             los fixtures que se suben a $B
lectora.json, sin-acceso.json   claves de las identidades de prueba. IGNORADAS por git, nunca se versionan

sdd/                       todos los artefactos del pipeline SDD de este proyecto (sin la teoría)
  gcsgrep-base-context.md    base context: requerimientos refinados, diseño, arquitectura
  gcsgrep-spec.md            la spec revisada: 50 requisitos, 50 VCs
  gcsgrep-plan.md            el plan de iteraciones y el alcance diferido
  gcsgrep-cobertura-vc.md    una fila por VC: con qué se ejercita y qué se observó
  CONTEXT.md                 este archivo: estado vivo, se reescribe
  DECISIONS.md               decisiones no obvias, solo se agrega
  iterations/NN-*.md         un registro inmutable por iteración cerrada
```

Dependencias entre paquetes: `cli` usa `location` y `search`; `gcs` no conoce `search`;
`search` no conoce GCS (recibe un `io.Reader`); `cmd/gcsgrep` une todo.

## Comandos

```bash
# Unitarios y vet (sin red)
GOTOOLCHAIN=local go vet ./... && go test ./internal/...

# Compilar
go build -o gcsgrep ./cmd/gcsgrep

# Suite completa contra el bucket real (necesita las claves y `gcloud` en el PATH)
export GCSGREP_E2E_BUCKET=sdd-fardenghi-itba
export GCSGREP_E2E_LECTORA=$PWD/lectora.json
export GCSGREP_E2E_SIN_ACCESO=$PWD/sin-acceso.json
go test ./e2e -count=1 -v        # ~50 s

# Solo lo que no toca GCS (sirve una ruta ficticia para las claves)
GCSGREP_E2E_BUCKET=sdd-fardenghi-itba GCSGREP_E2E_LECTORA=/x GCSGREP_E2E_SIN_ACCESO=/x \
  go test ./e2e -v -run 'TestVC(05|06|14|22|23|35|42Parcial)$'

# Usar la herramienta a mano con una identidad de prueba
GOOGLE_APPLICATION_CREDENTIALS=$PWD/lectora.json ./gcsgrep timeout gs://sdd-fardenghi-itba/logs/app/
```

`go test ./e2e` compila el binario solo. La suite completa corre unos 50 s contra GCS real y
consume unas pocas operaciones de listado y lectura (cabe de sobra en el free tier).

## Entorno

- **Bucket `$B`:** `gs://sdd-fardenghi-itba` (`us-east1`, Standard). 23 objetos: los fixtures de
  `test-fixtures/` sin el `README.md`.
- **`lectora`:** solo `storage.objects.get` y `storage.objects.list`. **`sin-acceso`:** nada.
- `go.mod` está fijado a `go 1.22` a propósito (D-01).
- El binario `./gcsgrep` está en `.gitignore`.

## Lo que sigue (Iteración 2)

Ver el handoff en el registro de la Iteración 1. En corto: servidor de prueba con `httptest`,
sniffing (BR-6/BR-7), `-c` y `-l`, marcadores de carpeta, y cerrar VC-12, 15, 41 y 42.

**A resolver primero: transcoding (VC-45).** Ya está medido (D-19): `transcoded.log` llega como
texto, pero el request envía `Accept-Encoding: gzip`, que agrega el transporte HTTP de Go y BR-7 no
admite. Hay que ajustar el cliente para que no lo pida y verificarlo con el servidor de prueba
(que registra headers) y con VC-45 contra `$B`.

**Para no perder de vista:**
- El caso de `--max` sin valor (D-18) ya está en `TestVC42Parcial` y pasa (D-20); a VC-42 solo le falta
  `--max unlimited` sobre 2500 objetos.
- VC-13 y VC-16 de la Iteración 3 cambiaron con la spec (un solo mensaje `not found`, D-17). No hay
  nada que hacer antes de esa iteración.
