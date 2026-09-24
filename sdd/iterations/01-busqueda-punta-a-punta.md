# Iteración 1 — Búsqueda de punta a punta

> Registro **inmutable**. Lo aprendido que afecta a lo que sigue va a `CONTEXT.md`,
> `DECISIONS.md` y a las iteraciones futuras del plan; este archivo no se reescribe.

| | |
|---|---|
| Fecha | 2026-09-23 |
| Commit de partida | `840084a` |
| Toolchain | Go 1.22.2 (`GOTOOLCHAIN=local`) |
| Bucket `$B` | `gs://sdd-fardenghi-itba`, 23 objetos |
| Identidades | `lectora` (solo `storage.objects.get` y `storage.objects.list`), `sin-acceso` (ninguno) |
| Resultado | **Gate aprobado: 16 de 16 VCs pasan; evidencia parcial de 4 VCs que cierran en la Iteración 2** |

## Qué se planeó

Búsqueda de punta a punta contra GCS real, secuencial, solo lectura, sin ampliar el acceso y
con guardrail de costo: módulos `cli`, `location`, `gcs`, `search` y `output`; parser con
`-E`, `-i`, `-n`, `--max` y `--`; ADC explícito; listado con tope que aborta antes de leer;
lectura por streaming con recorte de `\r` y línea completa buscada reteniendo su primer MiB.
Fuera de alcance: servidor de prueba, sniffing, `-c`, `-l`, marcadores de carpeta, mensajes
de objeto o bucket inexistente, timeouts, concurrencia, progreso.

## Cómo se verificó

Cada VC es un test de `e2e/` con su número en el nombre; compila `./gcsgrep` y lo ejecuta como
proceso aparte contra el bucket real, con `lectora` como ADC (claves JSON en
`GOOGLE_APPLICATION_CREDENTIALS`).

```bash
export GOTOOLCHAIN=local
export GCSGREP_E2E_BUCKET=sdd-fardenghi-itba
export GCSGREP_E2E_LECTORA=$PWD/lectora.json
export GCSGREP_E2E_SIN_ACCESO=$PWD/sin-acceso.json
go vet ./... && go test ./internal/... -count=1
go build -o gcsgrep ./cmd/gcsgrep
go test ./e2e -count=1 -v
```

Resultado de la corrida final (`ok  gcsgrep/e2e  48.493s`):

| VC | Requisito | Test | Resultado | Observado |
|---|---|---|---|---|
| VC-1 | FR-1 | `TestVC01` | PASS | exit `0`, las 3 líneas exactas de `logs/app/` |
| VC-3 | FR-3 | `TestVC03` | PASS | `job_id=10.` → `1`; `(` sobre `logs/` → `1`, no `2` |
| VC-4 | FR-4 | `TestVC04` | PASS | `-E` → líneas `job_id=102` y `104`; con `job_id=10.` las 6 |
| VC-5 | FR-5 | `TestVC05` | PASS | `2`, `invalid pattern: …`, chequeo P: 0 conexiones |
| VC-6 | FR-6 | `TestVC06` | PASS | 3 casos, `2`, `gcsgrep: empty pattern`, chequeo P |
| VC-7 | FR-7 | `TestVC07` | PASS | 4 líneas con `-i`, 2 sin; `LANG=C -i 'ñandú árbol'` → `ÑANDÚ ÁRBOL`; sin `-i` → `1` |
| VC-8 | FR-8 | `TestVC08` | PASS | líneas `:4:` y `:14:`, en ese orden |
| VC-9 | FR-9 | `TestVC09` | PASS | `:2:Linea 2 con timeout y terminacion Windows`, sin `0x0d`; `Windows$` → `0` |
| VC-14 | FR-14 | `TestVC14` | PASS | 5 ubicaciones inválidas, `2`, mensaje exacto, chequeo P |
| VC-21 | FR-21 | `TestVC21` | PASS | `-- -1] …/postgres.log` → `0`, 5 líneas |
| VC-22 | FR-22 | `TestVC22` | PASS | `unknown flag: "-1]"` y `"-v"`, chequeo P |
| VC-23 | FR-23 | `TestVC23` | PASS | `got 0`, `1`, `3`, chequeo P |
| VC-35 | FR-35 | `TestVC35` | PASS | sin ADC: `2`, mensaje exacto, 0 conexiones al listener |
| VC-39 | BR-1 | `TestVC39` | PASS | 23 objetos, (nombre, generación, metageneración) idénticos antes y después |
| VC-40 | BR-2 | `TestVC40` | PASS | `sin-acceso`: `access denied` en prefijo y objeto; `lectora`: la salida de VC-1 |
| VC-46 | BR-8 | `TestVC46` | PASS | match temprano, tardío y regex que cruza el límite: misma línea, 1 048 576 bytes + `...` |

Evidencia parcial (**no** cuentan como "pasa"; cierran en la Iteración 2):

| VC | Test | Resultado | Falta observar |
|---|---|---|---|
| VC-12 | `TestVC12Parcial` | PASS | sin request de listado ni lectura de `a.log.bak` |
| VC-15 | `TestVC15Parcial` | PASS | bucket existente sin objetos (`fake/empty`) |
| VC-41 | `TestVC41Parcial` | PASS | tope por defecto de 1000 y que no se pida la página siguiente |
| VC-42 | `TestVC42Parcial` | PASS (valores inválidos) | `--max unlimited` sobre 2500 objetos |

Unitarios: `ok` en `internal/cli`, `internal/location` e `internal/search`. `go vet ./...` y
`gofmt -l .` sin hallazgos.

**Las demos del plan** (`lectora`, `$B`):

```
$ ./gcsgrep timeout gs://$B/logs/app/                 3 líneas   [exit 0]
$ ./gcsgrep -n -i timeout gs://$B/logs/app/api.log    4 líneas numeradas (4, 7, 10, 14)   [exit 0]
$ ./gcsgrep -E 'job_id=\d+ queue=\w+ status=ERROR' gs://$B/logs/app/worker.log   102 y 104   [exit 0]
$ ./gcsgrep timeout gs://$B/no-existe/                sin salida   [exit 1]
$ ./gcsgrep --max 5 timeout gs://$B/edge-cases/       gcsgrep: more than 5 objects under gs://$B/edge-cases/; use --max <N> or --max unlimited   [exit 2]
$ ./gcsgrep timeout logs/                             gcsgrep: invalid location: "logs/"   [exit 2]
```

**Permisos efectivos** (consulta `testIamPermissions`, que no escribe): `lectora` →
`storage.objects.get`, `storage.objects.list`; no tiene `create`, `update`, `delete`, `buckets.get`
ni `setIamPolicy`. `sin-acceso` → ninguno. Es lo que hace que VC-39 demuestre algo.

## Qué pasó de distinto a lo planeado (desvíos)

1. **BR-8: las líneas largas se buscan siempre como stream.** El plan decía `re.Match(prefijo)`
   primero y `MatchReader` solo si no daba match. Un match sobre el prefijo retenido da falsos
   positivos con `$` o `\b`. Ver `DECISIONS.md` D-10. Una mutación de prueba confirmó que los tests
   lo detectan.
2. **Dependencias fijadas** (`storage v1.50.0`) para conservar `go 1.22` (D-01). Con `@latest`, Go
   descargaba una toolchain 1.26 sin avisar.
3. **Claves JSON en vez de impersonación.** El plan admitía las dos vías. Se usaron claves de service
   account, en `lectora.json` y `sin-acceso.json` en la raíz del repo, **ignoradas por git** y con permisos `0600` (D-16). El script de impersonación
   (`tools/e2e-creds.sh`) que se había escrito nunca se probó contra GCS real y se eliminó.
4. **Sin valor en la spec:** mensaje de `--max` sin argumento, y flags repetidos aceptados hasta la
   Iteración 4 (D-08).

## Dificultades

- Un primer intento de subir las dependencias con `@latest` movió `go.mod` a Go 1.26; se corrigió (D-01).

## Sin verificar en esta iteración

- **El gate corre dentro de una sola máquina.** El chequeo P y VC-35 asumen que la máquina está fuera de
  GCP (sin servidor de metadata); acá lo está.

## Handoff a la Iteración 2

Leer, en orden: la spec, el plan, `CONTEXT.md`, `DECISIONS.md` y este registro.

1. **El servidor de prueba** (`internal/fakegcs`, con `httptest`). El cliente pide, contra un endpoint
   emulado: listado en `GET /storage/v1/b/{bucket}/o` (parámetros `prefix` y `fields`, y `pageToken`);
   metadata en `GET /storage/v1/b/{bucket}/o/{objeto}`; contenido en la misma ruta con `?alt=media`. Un
   servidor mínimo con esas tres rutas alcanzó para las pruebas de esta iteración. Recordar D-04:
   con `STORAGE_EMULATOR_HOST` el cliente se crea sin credenciales, pero `gcs.Credentials` se sigue
   llamando y exige ADC (los VCs contra el servidor corren con ADC válidas).
2. **Riesgo de transcoding (VC-45).** Sigue abierto. Hay que confirmar contra `$B` qué `Accept-Encoding`
   manda el cliente con lecturas JSON, y si `sniffing/transcoded.log` llega comprimido. Si no se puede
   corregir con la configuración del cliente, cambia el *qué*: se vuelve a la spec.
3. **Volver a verificar VC-3 y VC-41** con sniffing: hoy los no-texto se leen como bytes (`old_logs.log.gz`
   no contiene `(`; VC-41 con `--max 6` solo mira el exit code).
4. **Marcadores de carpeta** (FR-17): hoy se leen como objetos vacíos y cuentan para el tope (D-14).
5. **Cerrar VC-12, 15, 41 y 42** con el servidor: cambiar `TestVCnnParcial` por `TestVCnn` completo.
6. **Nota para la Iteración 3 (medida contra GCS real).** `gcsgrep timeout gs://<bucket-inexistente>/` sale
   como `list error: storage: bucket doesn't exist` (el cliente distingue el bucket en el listado), pero
   con una ubicación de **objeto**, tanto `gs://<bucket-inexistente>/a.log` como un objeto inexistente de
   un bucket real salen como `metadata error: storage: object doesn't exist`. El cliente de Go convierte
   todo `404` de `Attrs` en `ErrObjectNotExist` y **descarta el cuerpo del error**, así que la
   distinción por cuerpo que propone el plan no es posible con `Attrs`. Candidatos: un request propio, o
   el listado con `maxResults=1` solo en el camino de error (la spec no lo prohíbe: FR-12 habla del
   caso exitoso). Confirmarlo al empezar esa iteración.
