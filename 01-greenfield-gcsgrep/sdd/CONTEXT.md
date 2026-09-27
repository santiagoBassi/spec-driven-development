# gcsgrep — contexto

> Se actualiza al cerrar cada iteración. Para el detalle de una decisión, ver
> `DECISIONS.md`; para lo que pasó en una iteración puntual, ver `iterations/NN-*.md`.

## Estado

**Iteración 1 cerrada.** Búsqueda de punta a punta contra GCS real (bucket de
fixtures `$B`), secuencial (`N = 1`), con el guardrail de costo activo desde el
primer día. 18 VCs pasan, 4 quedan con evidencia parcial (VC-12, VC-15a, VC-41,
VC-42): les falta la mitad que solo se observa con el servidor de prueba de la
Iteración 2.

Próximo paso: Iteración 2 (servidor de prueba, sniffing, `-c`/`-l`). Ver
`gcsgrep-plan.md`.

## Mapa de archivos

```
go.mod / go.sum          module gcsgrep, go 1.22, cloud.google.com/go/storage v1.50.0
cmd/gcsgrep/
  main.go                 os.Exit(run(...))
  run.go                  arma las piezas y traduce el resultado a exit code
internal/cli/             Parse(argv) → Config o error de uso; no toca GCS
internal/location/         Parse("gs://…") → Location{Bucket, Path, Mode}
internal/gcs/              Credentials, NewClient, List (con tope), Stat, Open,
                            IsAccessDenied
internal/search/            Compile(patrón) y Scanner (delimita líneas, BR-8)
internal/output/            Printer (stdout) y Warn (stderr)
e2e/                        un test por VC (o …Parcial); TestMain compila el
                            binario y toma la foto de VC-39
sdd/                        spec, plan, cobertura, este archivo, DECISIONS.md,
                            iterations/
```

Módulos que el plan reserva para iteraciones posteriores y todavía no existen:
`internal/sniff/` (Iteración 2) e `internal/fakegcs/` (Iteración 2).

## Cómo correr el gate

```bash
cd 01-greenfield-gcsgrep
GOTOOLCHAIN=local go build -o gcsgrep ./cmd/gcsgrep
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go test ./internal/...

GCSGREP_BUCKET=sdd-fardenghi-itba \
GCSGREP_LECTORA=$PWD/lectora.json \
GCSGREP_SIN_ACCESO=$PWD/sin-acceso.json \
GOTOOLCHAIN=local go test ./e2e -v -count=1
```

`GOTOOLCHAIN=local` evita que `go` intente bajar otra toolchain (la máquina de
desarrollo tiene 1.22.2, que ya cumple el piso `go 1.22` del módulo).

Variables de entorno de `e2e` (obligatorias; si falta alguna, `TestMain` falla,
no las saltea):

| Variable | Qué es |
|---|---|
| `GCSGREP_BUCKET` | Nombre de `$B`, sin `gs://` (hoy `sdd-fardenghi-itba`) |
| `GCSGREP_LECTORA` | Ruta a la clave JSON de la service account `lectora` |
| `GCSGREP_SIN_ACCESO` | Ruta a la clave JSON de la service account `sin-acceso` |

También hace falta `gcloud` en el `PATH` (lo usa la foto de VC-39).

## Entorno ya preparado

- `$B = gs://sdd-fardenghi-itba`, `us-east1`, con los 24 fixtures de
  `test-fixtures/` (incluidos `data/acentos.log` y `data/sin_salto_final.txt`).
- `lectora` y `sin-acceso`: claves JSON en la raíz del repo (`lectora.json`,
  `sin-acceso.json`; en `.gitignore`, nunca se versionan).
- No hace falta servidor de prueba ni listener manual: `e2e` arma su propio
  listener TCP en proceso para el chequeo P.

## Qué falta para que los 4 parciales cierren (Iteración 2)

- **VC-12**: que un objeto puntual no genere request de listado y no lea
  `a.log.bak` — necesita el servidor de prueba para observar los requests.
- **VC-15a**: un bucket `empty` real (sin objetos) — hoy la evidencia es que
  `gs://b/` parsea al mismo modo/prefijo vacío que un prefijo sin objetos
  (`TestVC15aParcial`), más VC-15b que sí corre contra `$B`.
- **VC-41**: el tope por defecto de 1000 sin pedir la página siguiente —
  necesita 1000+ objetos, que no se pueden crear en `$B` sin violar su
  composición fija.
- **VC-42**: `--max unlimited` leyendo 2500 objetos reales — mismo motivo.
