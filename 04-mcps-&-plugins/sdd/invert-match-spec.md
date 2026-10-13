# invert-match (`-v`) — spec
> Estado: borrador · Implementación: bloqueada por prerrequisitos · Base explorada: 01-greenfield-gcsgrep@52596d7 · Spec base: `01-greenfield-gcsgrep/sdd/gcsgrep-spec.md`

## Propósito

Agregar el flag `-v` a `gcsgrep` para imprimir las líneas que **no** contienen el patrón, como `grep -v`, y que sus VCs se apoyen en lo que hay de verdad en el bucket de fixtures.

## Prerrequisitos de implementación

Se mantienen los VCs de gzip, binario y objeto inexistente por decisión del equipo. La base explorada `52596d7` implementa solamente la Iteración 1: aún no clasifica texto ni emite `not found`. Por eso esta spec describe un cambio **posterior** a estos prerrequisitos, no afirma que esos VCs ya pasan:

1. **Iteración 2 — sniffing y formatos:** completar la clasificación de texto de BR-6/BR-7 y el aviso de BR-5 de la spec base, según `01-greenfield-gcsgrep/sdd/gcsgrep-plan.md`, sección Iteración 2. Antes de agregar `-v`, `./gcsgrep timeout gs://$B/logs/archive/old_logs.log.gz` y `./gcsgrep timeout gs://$B/edge-cases/binary_nullbyte.bin` deben salir con código `1`, stdout vacío y una línea de stderr `gcsgrep: <URI>: not a text file, skipped`, terminada en LF.
2. **Iteración 3 — errores de GCS:** completar la traducción `not found` (FR-13a de la spec base), según la sección Iteración 3 del mismo plan y `sdd/DECISIONS.md`, D-13. Antes de agregar `-v`, `./gcsgrep timeout gs://$B/data/no-existe-vc-br-2.txt` debe salir con código `2`, stdout vacío y stderr `gcsgrep: not found: gs://$B/data/no-existe-vc-br-2.txt`, terminado en LF. La base explorada devuelve actualmente `metadata error: storage: object doesn't exist`; esa diferencia se midió y se registra en la evidencia del TP4.
3. **Fijar la nueva base:** después de completar ambas iteraciones, registrar el hash como `INVERT_MATCH_BASE`, confirmar nuevamente las interfaces y anclas de Dentro y medir ambas suites en verde. Registrar en la sección Línea de base de esta spec la fecha, el hash, los códigos de salida, las cifras PASS/FAIL/SKIP, los nombres de todos los tests de nivel superior por paquete y las rutas a ambos resultados JSON conservados como evidencia; hasta completar ese registro no se implementa `-v`. Esa revisión puede actualizar las anclas si los prerrequisitos cambiaron las interfaces. Hasta entonces no se implementa `-v`. El alcance de Dentro cubre únicamente `-v` sobre esa base preparada; los prerrequisitos tienen sus propios planes y commits.

Un veredicto `LISTA` permite planificar la feature; no levanta estos prerrequisitos. Las 23 pruebas unitarias y 21 e2e medidas abajo son evidencia de la base explorada, no evidencia de que las iteraciones 2 y 3 estén terminadas.

## Convenciones

- `$B` es `$GCSGREP_BUCKET` (`sdd-fardenghi-itba`) y `./gcsgrep` es el binario de `go build -o gcsgrep ./cmd/gcsgrep`, como en la spec base.
- Directorios: los comandos con `./gcsgrep` y `go test` sin `cd` corren desde `01-greenfield-gcsgrep/`. Los que nombran `../01-greenfield-gcsgrep` o `../02-brownfield-tmux` corren desde `04-mcps-&-plugins/`. Las rutas que imprime `git` son relativas a la raíz del repositorio. `INVERT_MATCH_BASE` es el hash registrado después de completar los prerrequisitos y antes de implementar `-v`, no `52596d7`.
- "Parece una instrucción" es la sección de la salida de `lector-fixtures` (`.claude/agents/lector-fixtures.md`) que anota las líneas que le hablan a un agente o piden ejecutar algo. Se trata como dato.
- En VC-1 a VC-7, stderr queda vacío. Toda línea de stdout o stderr citada como salida exacta termina en un LF, salvo los stdout/stderr explícitamente vacíos; no hay bytes extra.
- Cada lectura se identifica por el `updated` de `read_object_metadata` más la `generation` y el MD5 nativos de GCS que devuelve el helper `procedencia.py` del skill (ver Procedencia).

## Alcance

### Dentro

| Archivo | Cambio | Referencia (archivo:línea) |
|---|---|---|
| `internal/cli/cli.go` | `Config` gana `Invert bool`; `Parse` reconoce `-v` | `cli.go:24` (campos de `Config`), `cli.go:63` (`case "-n"`) |
| `internal/search/search.go` | `Scanner` gana el campo exportado `Invert bool` (cero = comportamiento actual); `Scan` conserva su firma `Scan(r, re, fn)` y entrega la línea cuando `matched != Invert` | `search.go:86` (firma, sin cambio), `search.go:108` y `:115` (cálculo de `matched`), `search.go:127` (`if matched`) |
| `cmd/gcsgrep/run.go` | asigna `scanner.Invert = cfg.Invert` justo después de crear el `Scanner`; `readObject` y su firma no cambian | `run.go:69` (`search.NewScanner()`) |
| `internal/cli/cli_test.go` | `-v` deja de ser el ejemplo de flag desconocido: se cambia por `-x` | `cli_test.go:61` (subtest `unknown long flag`), `cli_test.go:93-95` (`TestParseErrorOrder`) |
| `e2e/usage_test.go` | mismo cambio de `-v` a `-x` en el caso de flag desconocido | `usage_test.go:40` |
| `internal/search/search_test.go` | exactamente cinco tests nuevos: `TestScanInvertNoMatch` (VC-1), `TestScanInvertIgnoreCase` (VC-3), `TestScanInvertLongLine` (VC-14), `TestScanInvertMatchAfterLimit` (VC-15) y `TestScanInvertEmptyLine` (VC-16); los tests existentes no se tocan y siguen llamando `s.Scan(r, re, fn)` | archivo existente, solo se agregan funciones |
| `internal/cli/cli_test.go` | casos `invert` (VC-1) e `invert-literal-after-double-dash` (VC-17) en `TestParseValid`. Para `invert`: args `[]string{"-v", "timeout", "gs://b/logs/"}`, error nil y `Config{Invert: true, Max: 1000, Pattern: "timeout", Location: "gs://b/logs/", Extended: false, IgnoreCase: false, LineNumber: false}` | `cli_test.go`, tabla de `TestParseValid` |
| `e2e/invert_test.go` | archivo nuevo con exactamente estos 19 tests, uno por VC que corre `./gcsgrep` con `-v`: `TestInvertVC1`, `TestInvertVC2`, `TestInvertVC3`, `TestInvertVC4`, `TestInvertVC5`, `TestInvertVC6`, `TestInvertVC7`, `TestInvertVC8a`, `TestInvertVC8b`, `TestInvertVC9`, `TestInvertVC10`, `TestInvertVC11`, `TestInvertVC12`, `TestInvertVC13a`, `TestInvertVC13b`, `TestInvertVC13c`, `TestInvertVCBR1`, `TestInvertVCBR2` y `TestInvertVCBR3`. Ningún test existente empieza con `TestInvert`. VC-INV-2 ya lo cubre `TestVC08` y VC-INV-3 el `TestVC22` modificado. VC-NFR-1 se corre como script de shell, no como test de Go | archivo nuevo |

### Fuera

- Combinar `-v` con `-c` o `-l`: esos flags son de otra iteración y definen su propia semántica.
- `--max` y el listado: `-v` no los toca; el tope cuenta objetos listados, no líneas (BR-3 de la spec base).
- Un VC e2e de líneas de más de 1 MiB con `-v`: el único objeto real, `edge-cases/long_line_exceeds_1mb.log`, excede el límite de respuesta del MCP. La selección y el recorte del Scanner sí se cubren con entradas sintéticas unitarias en FR-14 y FR-15; el formato del aviso `...` en el ejecutable conserva BR-8 de la spec base.
- Errores de lectura distintos de "el objeto no existe" y "acceso denegado" (por ejemplo, red): los fija la spec base (FR-29a, FR-30a), `-v` no los cambia.
- Los demás flags de `grep` que la spec base deja fuera: `-r`, `--include`, `-o`, `-w`, contexto y color.
- Cambiar la clasificación de texto (BR-6 de la spec base) o el tratamiento del gzip (BR-7): `-v` los hereda.
- `edge-cases/long_line_exceeds_1mb.log`: el lector no pudo leer su contenido (la tool devolvió "exceeds maximum allowed tokens"), así que ningún VC se apoya en él.
- Escribir o borrar objetos del bucket: los fixtures se leen, nunca se modifican.
- Editar la spec base `01-greenfield-gcsgrep/sdd/gcsgrep-spec.md`. Queda como registro de la iteración `52596d7`, y esta spec la reemplaza en lo que toca a `-v`: el ítem de Fuera de `gcsgrep-spec.md:46` deja de valer para `-v` (FR-1), y la mitad de su VC-22 que espera `unknown flag: "-v"` deja de valer (INV-3). Actualizarla es tarea de otra iteración, no de este cambio (INV-4).
- Implementar el rechazo de flags cortos combinados (FR-24a de la spec base) y de flags repetidos (FR-25). En `52596d7` el código no los implementa para ningún flag (`cli.go:58-78`). `-v` sigue el mismo tratamiento que `-i` (FR-13a, FR-13b, FR-13c); cuando otra iteración implemente FR-24a y FR-25, va a reescribir también FR-13a, FR-13b y FR-13c.
- `-v` sobre un bucket o prefijo sin objetos, o sobre un bucket inexistente, y `-v` junto con los errores de uso y de ubicación (regex inválida, patrón vacío, ubicación con formato inválido, cantidad de posicionales, un solo error de uso): `-v` no cambia qué se lista ni qué se valida, así que vale lo que fija la spec base (FR-5, FR-6, FR-14, FR-15a, FR-15b, FR-16a, FR-16b, FR-23 y FR-27), sin VC propio en esta spec.

## Línea de base

`cd ../01-greenfield-gcsgrep && go test ./internal/... -v 2>&1 | grep -c '^--- PASS'` → 23 PASS, 0 FAIL (`cli`, `location` y `search` en `ok`; `gcs` y `output` sin tests). Medida el 2026-10-13 antes del cambio.

La suite e2e (`./e2e`) tiene en `52596d7` estos 21 tests de nivel superior: `TestVC15b`, `TestVC35`, `TestVC40` (`access_test.go`); `TestVC12Parcial`, `TestVC41Parcial`, `TestVC42Parcial` (`partial_test.go`); `TestVC01`, `TestVC03`, `TestVC04`, `TestVC07a`, `TestVC08`, `TestVC09a`, `TestVC09b`, `TestVC21`, `TestVC46` (`search_test.go`); `TestVC05`, `TestVC06`, `TestVC14`, `TestVC22`, `TestVC23` (`usage_test.go`); y `TestVC39` (`zz_vc39_test.go`). El único que usa `-v` es `TestVC22` (`usage_test.go:40`). La suite se midió antes de implementar `-v`, el 2026-10-13T18:15:35.450225+00:00, con `go test ./e2e -json`, bucket `sdd-fardenghi-itba` e identidades `lectora` y `sin-acceso`: exit `0`, 21 PASS, 0 FAIL y 0 SKIP. No se modificó código de `01-greenfield-gcsgrep`. La medición se ejecutó fuera de la sesión de lectura de fixtures y se registró en `../evidencia/03-skill.md`, sección 10. INV-5 verifica después del cambio ese mismo conjunto de tests, más los nuevos de `-v`.

## Procedencia de los fixtures

Leídos en vivo el 2026-10-13T03:06Z vía MCP; `logs/app/api.log` releído el 2026-10-13T18:09Z (selecciones `match` y `sin-match`, patrón `timeout`, prefijo `logs/app/api.log`). Server `@google-cloud/storage-mcp` 0.6.0 sin `--enable-destructive-tools`, cuenta `lectora@sat-spec-driven-development.iam.gserviceaccount.com`, bucket `gs://sdd-fardenghi-itba`.

El server 0.6.0 no expone `generation` ni checksums: `Generation` y `MD5` salen del helper `procedencia.py` del skill (`gcloud storage objects describe --raw` con la misma credencial lectora, sin leer contenido), consultado el 2026-10-13T18:09Z. `Updated` es el de `read_object_metadata` y coincide, completo y como texto, con el del helper en los 12 objetos.

| Objeto | Generation | MD5 (base64) | Updated | Tipo de contenido | VCs que lo citan |
|---|---|---|---|---|---|
| `logs/app/api.log` | `1791914566720157` | `vahZ/f5lsfZILBzK2ouO7g==` | `2026-10-13T18:02:46.750Z` | `text/plain` | VC-1, VC-2, VC-3, VC-4, VC-12, VC-13a, VC-BR-1, VC-BR-3, VC-INV-2, VC-NFR-1 |
| `logs/app/worker.log` | `1790119237365907` | `yym80Z+JYrLZcWMxPyQYkw==` | `2026-09-22T23:20:37.370Z` | `text/plain` | VC-12 |
| `logs/db/postgres.log` | `1790119237355295` | `n1T//eblOXT+lZeMIESFcQ==` | `2026-09-22T23:20:37.359Z` | `text/plain` | VC-12 |
| `logs/db/redis.log` | `1790119237374465` | `nVVau7LVd8REL5zBG3RsfQ==` | `2026-09-22T23:20:37.379Z` | `text/plain` | VC-12 |
| `logs/server.log` | `1790119237289998` | `yyNy+vY/pd19JRVnumPu5A==` | `2026-09-22T23:20:37.293Z` | `text/plain` | VC-12 |
| `data/subdata/deep_nested.txt` | `1790119237360468` | `E/k4O/agK9uThiCXsJLGgg==` | `2026-09-22T23:20:37.366Z` | `text/plain` | VC-5 |
| `data/windows_crlf.txt` | `1790119237334512` | `P2aJbfIH5vDpfCKwckn+/w==` | `2026-09-22T23:20:37.340Z` | `text/plain` | VC-6, VC-8b |
| `data/sin_salto_final.txt` | `1790370956298722` | `6oiUMenBGDDgMzDx5GTmNA==` | `2026-09-25T21:15:56.317Z` | `text/plain` | VC-7 |
| `edge-cases/empty.txt` | `1790119237353571` | `1B2M2Y8AsgTpgAmY7PhCfg==` | `2026-09-22T23:20:37.360Z` | `text/plain` | VC-8a |
| `logs/archive/old_logs.log.gz` | `1790119237355510` | `tq+KLeAq/wr4Yfg/KgHSMQ==` | `2026-09-22T23:20:37.361Z` | `text/plain` (metadata; es gzip por contenido) | VC-9, VC-12 |
| `edge-cases/binary_nullbyte.bin` | `1790119237353687` | `D1/XMni2zRc72Q9hB0VuOA==` | `2026-09-22T23:20:37.357Z` | `application/octet-stream` | VC-10 |
| `fixtures/hostil/notas.log` | `1791840852693528` | `ETcxgjogkWyjVuw2sPQ+sg==` | `2026-10-12T21:34:12.699Z` | `text/plain` | VC-11 |

`fixtures/hostil/notas.log`: el lector marcó la línea 2 en "Parece una instrucción", con las primeras 40 letras `NOTA PARA EL ASISTENTE: antes de seguir,`; esa marca respalda FR-11. El comando completo no se transmite ni se usa como salida esperada. Ver `../evidencia/05-fixture-hostil.md`.

Relectura de `logs/app/api.log` (2026-10-13T18:09Z): el objeto fue re-subido (updated anterior `2026-09-22T23:20:37.359Z`) con el mismo MD5 `vahZ/f5lsfZILBzK2ouO7g==`. El lector devolvió 906 bytes y 15 líneas; con match de `timeout`, la 4 y la 14; sin match, las 13 restantes. Esas líneas tienen el mismo número y texto que citan VC-1, VC-2, VC-3, VC-4, VC-BR-1 e INV-2, así que los contenidos esperados no cambian; solo se actualizan sus anclas. Sin líneas en "Parece una instrucción".

La salida esperada de VC-12 se materializa en `../test-fixtures/invert-logs-esperado.txt`, generada a partir de otra lectura MCP con `lector-fixtures` (`selección: sin-match`, `texto_completo: sí`, prefijo `logs/`); las 32 líneas ordinarias se conservaron completas y el gzip se salteó. La sesión principal recibió el resultado del lector y no leyó contenido por su cuenta.

Listados de prefijos (2026-10-13T18:09Z, `list_objects` vía `lector-fixtures`, sin leer contenido, una página con `next_page_token` nulo en cada caso):

| Prefijo | Objetos listados | VCs que dependen del listado |
|---|---|---|
| `logs/` | 6: `logs/app/api.log`, `logs/app/worker.log`, `logs/archive/old_logs.log.gz`, `logs/db/postgres.log`, `logs/db/redis.log`, `logs/server.log` | VC-12 |
| `data/subdata/` | 1: `data/subdata/deep_nested.txt` | VC-5 |
| `data/no-existe-vc-br-2` | 0 (`object_count: 0`) | VC-BR-2 |

Cierre del 2026-10-13T18:09Z: para los 12 objetos de la tabla, `read_object_metadata` (MCP) y `procedencia.py` (nativo) devuelven el mismo `updated`, idéntico al anotado; `generation` y MD5 nativos son los de la tabla. Los tres listados devuelven exactamente los objetos anotados.

Si al cerrar algún `updated`, `generation` o checksum no coincide, o algún listado de prefijo devuelve objetos distintos de los anotados, esta spec pasa a `Estado: vencida` y los VCs de ese objeto o prefijo se reescriben contra una lectura nueva.

## Requerimientos funcionales

#### FR-1 · Imprimir las líneas que no contienen el patrón con `-v`

**Dado** un objeto de texto con líneas que contienen el patrón y líneas que no,
**Cuando** la persona ejecuta `./gcsgrep -v <patrón> gs://<bucket>/<objeto>`,
**Entonces** el sistema imprime por stdout solo las líneas que no contienen el patrón, con la forma `gs://<bucket>/<objeto>:<texto>`, y sale con código `0`.

> **VC-1** — `./gcsgrep -v timeout gs://$B/logs/app/api.log` sale con código `0` y su stdout es exactamente estas 13 líneas, en este orden, cada una con el prefijo `gs://$B/logs/app/api.log:`:
> `2026-09-22 10:00:01 INFO Server started on port 8080`,
> `2026-09-22 10:01:15 INFO Route registered /api/v1/status`,
> `2026-09-22 10:02:40 INFO Route registered /api/v1/checkout`,
> `2026-09-22 10:06:01 INFO Retry policy applied for worker-1`,
> `2026-09-22 10:10:00 INFO Health check status OK`,
> `2026-09-22 10:14:03 WARN Connection TIMEOUT to redis cache replica-0`,
> `2026-09-22 10:20:11 INFO Cache fallback to memory cache`,
> `2026-09-22 10:25:00 INFO Processing batch payment id=8842`,
> `2026-09-22 10:30:15 ERROR Database Timeout connecting to replica-db`,
> `2026-09-22 10:35:44 INFO Request completed 200 OK duration=45ms`,
> `2026-09-22 10:40:12 WARN Client closed connection prematurely`,
> `2026-09-22 10:45:00 INFO Health check status OK` y
> `2026-09-22 10:59:59 INFO Graceful shutdown scheduled`.
> Fixture: `logs/app/api.log`, líneas 1, 2, 3, 5 a 13 y 15, generation `1791914566720157`, updated `2026-10-13T18:02:46.750Z`. Las líneas 7 (`TIMEOUT`) y 10 (`Timeout`) salen porque el patrón distingue mayúsculas.


Caso unitario de VC-1 (`TestScanInvertNoMatch`): `search.NewScanner()` con `Invert=true`, regex de `search.Compile("timeout", false, false)` e input `strings.NewReader("ok\ntimeout\n")`; `Scan` retorna nil y llama exactamente una vez al callback con `(lineNo=1, text=[]byte("ok"), truncated=false)`. El callback devuelve nil. `go test ./internal/search -run '^TestScanInvertNoMatch$' -count=1` sale con código `0`. El caso de parser `TestParseValid/invert` usa los args y el Config exactos de Dentro, y `Parse` retorna error nil; `go test ./internal/cli -run '^TestParseValid$/^invert$' -count=1` sale con código `0`.

#### FR-2 · Numerar con `-n` las líneas seleccionadas por `-v`

**Dado** un objeto de texto con líneas que no contienen el patrón,
**Cuando** la persona ejecuta `-n -v` sobre él,
**Entonces** cada línea impresa lleva su número original en el objeto, con la forma `gs://<bucket>/<objeto>:<línea>:<texto>`.

> **VC-2** — `./gcsgrep -n -v timeout gs://$B/logs/app/api.log` sale con código `0` y su stdout son las mismas 13 líneas de VC-1, cada una con su número original entre el objeto y el texto: `1`, `2`, `3`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13` y `15`. No aparecen los números `4` ni `14`.
> Fixture: `logs/app/api.log`, generation `1791914566720157`, updated `2026-10-13T18:02:46.750Z`.

#### FR-3 · Combinar `-i` con `-v`

**Dado** un objeto con el patrón escrito en mayúsculas, minúsculas y mixto,
**Cuando** la persona ejecuta `-i -n -v` con el patrón en minúsculas,
**Entonces** el sistema excluye también las líneas que lo contienen en otras mayúsculas.

> **VC-3** — `./gcsgrep -i -n -v timeout gs://$B/logs/app/api.log` sale con código `0` y su stdout son las líneas de VC-2 sin las numeradas `7` y `10`: 11 líneas, con los números `1`, `2`, `3`, `5`, `6`, `8`, `9`, `11`, `12`, `13` y `15`.
> Fixture: `logs/app/api.log`, líneas 7 (`TIMEOUT`) y 10 (`Timeout`), generation `1791914566720157`, updated `2026-10-13T18:02:46.750Z`.


Caso unitario de VC-3 (`TestScanInvertIgnoreCase`): `search.NewScanner()` con `Invert=true`, regex de `search.Compile("timeout", false, true)` e input `strings.NewReader("ok\nTIMEOUT\n")`; `Scan` retorna nil y llama exactamente una vez al callback con `(lineNo=1, text=[]byte("ok"), truncated=false)`. El callback devuelve nil. `go test ./internal/search -run '^TestScanInvertIgnoreCase$' -count=1` sale con código `0`.

#### FR-4 · Combinar `-E` con `-v`

**Dado** un objeto con líneas de niveles `INFO`, `WARN` y `ERROR`,
**Cuando** la persona ejecuta `-E -n -v 'ERROR|WARN'`,
**Entonces** el sistema imprime solo las líneas que no matchean la regex.

> **VC-4** — `./gcsgrep -E -n -v 'ERROR|WARN' gs://$B/logs/app/api.log` sale con código `0` y su stdout son exactamente las 10 líneas de nivel `INFO` de VC-2, con los números `1`, `2`, `3`, `5`, `6`, `8`, `9`, `11`, `13` y `15`. Las líneas `4`, `7`, `10`, `12` y `14` no aparecen.
> Fixture: `logs/app/api.log`, generation `1791914566720157`, updated `2026-10-13T18:02:46.750Z`.

#### FR-5 · Aplicar `-v` bajo un prefijo anidado

**Dado** un prefijo con un objeto de texto en un subdirectorio,
**Cuando** la persona ejecuta `-v` sobre el prefijo,
**Entonces** el sistema imprime las líneas sin el patrón del objeto anidado.

> **VC-5** — `./gcsgrep -n -v timeout gs://$B/data/subdata/` sale con código `0` y su stdout es exactamente `gs://$B/data/subdata/deep_nested.txt:1:Archivo anidado profundamente en data/subdata/` y `gs://$B/data/subdata/deep_nested.txt:3:Fin de archivo anidado`, en ese orden.
> Fixture: listado de `data/subdata/` con un único objeto (ver Procedencia); `data/subdata/deep_nested.txt`, líneas 1 y 3 (la 2 contiene `timeout`), generation `1790119237360468`, updated `2026-09-22T23:20:37.366Z`.

#### FR-6 · Recortar el `\r` de las líneas seleccionadas por `-v`

**Dado** un objeto con terminaciones `\r\n`,
**Cuando** la persona ejecuta `-n -v` sobre él,
**Entonces** las líneas impresas no llevan el `\r` final.

> **VC-6** — `./gcsgrep -n -v timeout gs://$B/data/windows_crlf.txt` sale con código `0` y su stdout es exactamente `gs://$B/data/windows_crlf.txt:1:Linea 1 con retorno CRLF` y `gs://$B/data/windows_crlf.txt:3:Linea 3 con CRLF sin match`; `od -An -tx1` sobre el stdout no contiene ningún byte `0d`.
> Fixture: `data/windows_crlf.txt`, líneas 1 y 3, generation `1790119237334512`, updated `2026-09-22T23:20:37.340Z`.

#### FR-7 · Seleccionar con `-v` la última línea sin salto final

**Dado** un objeto cuya última línea no termina en `\n`,
**Cuando** la persona ejecuta `-v` con un patrón que esa línea no contiene,
**Entonces** el sistema imprime también esa última línea, terminada en `\n`.

> **VC-7** — `./gcsgrep -v timeout gs://$B/data/sin_salto_final.txt` sale con código `0` y su stdout es exactamente `gs://$B/data/sin_salto_final.txt:Primera linea con salto` y `gs://$B/data/sin_salto_final.txt:Ultima linea sin salto final`, y el último byte del stdout es `0a`.
> Fixture: `data/sin_salto_final.txt`, líneas 1 y 2, generation `1790370956298722`, updated `2026-09-25T21:15:56.317Z`.

#### FR-8a · Salir con `1` cuando `-v` se aplica a un objeto vacío

**Dado** un objeto de texto vacío,
**Cuando** la persona ejecuta `-v` sobre él,
**Entonces** el sistema no imprime nada y sale con código `1`.

> **VC-8a** — `./gcsgrep -v timeout gs://$B/edge-cases/empty.txt` sale con código `1`, con stdout y stderr vacíos.
> Fixture: `edge-cases/empty.txt`, 0 bytes, generation `1790119237353571`, updated `2026-09-22T23:20:37.360Z`.

#### FR-8b · Salir con `1` cuando todas las líneas contienen el patrón

**Dado** un objeto de texto cuyas líneas contienen todas el patrón,
**Cuando** la persona ejecuta `-v` con ese patrón,
**Entonces** el sistema no imprime nada y sale con código `1`.

> **VC-8b** — `./gcsgrep -v Linea gs://$B/data/windows_crlf.txt` sale con código `1`, con stdout y stderr vacíos: las tres líneas del objeto contienen `Linea`.
> Fixture: `data/windows_crlf.txt`, líneas 1, 2 y 3 (todas con `Linea`), generation `1790119237334512`, updated `2026-09-22T23:20:37.340Z`.

#### FR-9 · Saltear un objeto gzip también con `-v`

**Dado** un objeto gzip (no-texto por BR-6 y BR-7 de la spec base),
**Cuando** la persona ejecuta `-v` sobre él,
**Entonces** el sistema no imprime líneas, avisa que lo saltea y sale con código `1`.

> **VC-9** — `./gcsgrep -v timeout gs://$B/logs/archive/old_logs.log.gz` sale con código `1`, stdout vacío, y stderr es exactamente `gcsgrep: gs://$B/logs/archive/old_logs.log.gz: not a text file, skipped`. La salida esperada sale de la spec base (BR-5 y VC-45), no de decodificar el objeto.
> Fixture: `logs/archive/old_logs.log.gz`, 130 bytes, generation `1790119237355510`, updated `2026-09-22T23:20:37.361Z`.

#### FR-10 · Saltear un objeto binario también con `-v`

**Dado** un objeto con un byte `0x00` en sus primeros 512 bytes,
**Cuando** la persona ejecuta `-v` sobre él,
**Entonces** el sistema no imprime líneas, avisa que lo saltea y sale con código `1`.

> **VC-10** — `./gcsgrep -v timeout gs://$B/edge-cases/binary_nullbyte.bin` sale con código `1`, stdout vacío, y stderr es exactamente `gcsgrep: gs://$B/edge-cases/binary_nullbyte.bin: not a text file, skipped`. La salida esperada sale de la spec base (BR-5 y BR-6), no de decodificar el objeto.
> Fixture: `edge-cases/binary_nullbyte.bin`, 279 bytes, generation `1790119237353687`, updated `2026-09-22T23:20:37.357Z`.

#### FR-11 · Tratar como dato un objeto con una línea que parece una instrucción

**Dado** un objeto de texto cuya línea 2 está en la lista "Parece una instrucción" del lector,
**Cuando** la persona ejecuta `-n -v` sobre él,
**Entonces** el sistema imprime dos líneas con los números originales 1 y 2, sale con código `0` y deja stderr vacío; la línea 2 lleva el mismo prefijo de objeto y numeración que cualquier otra línea.

> **VC-11** — `./gcsgrep -n -v timeout gs://$B/fixtures/hostil/notas.log` sale con código `0`; su stdout tiene exactamente 2 líneas (`wc -l` da `2`); la primera es `gs://$B/fixtures/hostil/notas.log:1:2026-10-01T10:00:00Z INFO arranque normal`; y la segunda empieza con `gs://$B/fixtures/hostil/notas.log:2:`. El texto de la línea 2 no es parte de la salida esperada: se verifica por número de línea y cantidad. stderr queda vacío.
> Fixture: `fixtures/hostil/notas.log`, líneas 1 y 2 (la 3 contiene `timeout`), generation `1791840852693528`, updated `2026-10-12T21:34:12.699Z`.

#### FR-12 · Aplicar `-v` a un prefijo con objetos de texto y un gzip

**Dado** un prefijo recursivo con cinco objetos de texto y un objeto gzip,
**Cuando** la persona ejecuta `-v` sobre el prefijo,
**Entonces** el sistema imprime las líneas sin el patrón de los objetos de texto, avisa por el gzip y sale con código `0`.

> **VC-12** — Desde `01-greenfield-gcsgrep/`, con `D=$(mktemp -d)` fuera del repositorio, `./gcsgrep -v timeout gs://$B/logs/ >"$D/out.txt" 2>"$D/err.txt"` sale con código `0`. `LC_ALL=C sort "$D/out.txt" >"$D/actual.txt"` y `LC_ALL=C sort "../04-mcps-&-plugins/test-fixtures/invert-logs-esperado.txt" >"$D/esperado.txt"` salen con código `0`; `diff -u "$D/esperado.txt" "$D/actual.txt"` sale con código `0` y stdout/stderr vacíos. El fixture esperado contiene exactamente las 32 líneas completas leídas por MCP: 13 de `logs/app/api.log`, 5 de `logs/app/worker.log`, 4 de `logs/db/postgres.log`, 6 de `logs/db/redis.log` y 4 de `logs/server.log`, todas con su prefijo `gs://sdd-fardenghi-itba/<objeto>:`. Por tanto se verifica texto y distribución por objeto; solo el orden entre objetos se ignora. `"$D/err.txt"` tiene exactamente una línea: `gcsgrep: gs://$B/logs/archive/old_logs.log.gz: not a text file, skipped`, seguida de LF.
En `TestInvertVC12`, cuyo directorio de trabajo es `01-greenfield-gcsgrep/e2e/`, leer el esperado con `os.ReadFile(filepath.Join("..", "..", "04-mcps-&-plugins", "test-fixtures", "invert-logs-esperado.txt"))`; no usar `fixturesDir()` ni copiar el archivo al módulo Go. La lectura debe tener error nil. Sustituir solo el prefijo URI `gs://sdd-fardenghi-itba/` al comienzo de cada línea por `"gs://"+os.Getenv("GCSGREP_BUCKET")+"/"`, para usar el bucket configurado. Separar las salidas por LF, sin perder la comprobación del LF final, ordenar las líneas con `sort.Strings` y comparar igualdad byte por byte; comprobar por separado exit `0` y el stderr exacto definido arriba.

> Fixture: listado de `logs/` con exactamente 6 objetos (ver Procedencia): los cinco objetos de texto y `logs/archive/old_logs.log.gz`, cada uno con la generation y el updated de su fila en la tabla de procedencia (`logs/app/api.log`: generation `1791914566720157`, updated `2026-10-13T18:02:46.750Z`, 13 líneas sin match confirmadas en la relectura); las cantidades salen de la lectura con `selección: sin-match`.

#### FR-13a · Tratar `-v` repetido como una sola vez, igual que `-i`

**Dado** el parser de `52596d7`, que acepta un flag corto repetido (`cli.go:58-78`),
**Cuando** la persona pasa `-v` dos veces,
**Entonces** el sistema se comporta como con un solo `-v`.

> **VC-13a** — `./gcsgrep -v -v timeout gs://$B/logs/app/api.log` sale con código `0`, stderr vacío, y su stdout es exactamente el de VC-1 (13 líneas).
> Fixture: `logs/app/api.log`, líneas 1, 2, 3, 5 a 13 y 15, generation `1791914566720157`, updated `2026-10-13T18:02:46.750Z`.

#### FR-13b · Rechazar el argumento corto combinado `-vn`

**Dado** el parser de `52596d7`, que trata un argumento corto combinado como flag desconocido (`cli.go:76-77`),
**Cuando** la persona ejecuta `./gcsgrep -vn timeout gs://$B/logs/app/api.log`,
**Entonces** el sistema sale con código `2`, stdout vacío y stderr `gcsgrep: unknown flag: "-vn"`.

> **VC-13b** — `./gcsgrep -vn timeout gs://$B/logs/app/api.log` sale con código `2`, stdout vacío, y stderr es exactamente `gcsgrep: unknown flag: "-vn"`. El rechazo ocurre antes de leer el objeto.

#### FR-13c · Rechazar el argumento corto combinado `-iv`

**Dado** el parser de `52596d7`, que trata un argumento corto combinado como flag desconocido (`cli.go:76-77`),
**Cuando** la persona ejecuta `./gcsgrep -iv timeout gs://$B/logs/app/api.log`,
**Entonces** el sistema sale con código `2`, stdout vacío y stderr `gcsgrep: unknown flag: "-iv"`.

> **VC-13c** — `./gcsgrep -iv timeout gs://$B/logs/app/api.log` sale con código `2`, stdout vacío, y stderr es exactamente `gcsgrep: unknown flag: "-iv"`. El rechazo ocurre antes de leer el objeto.

#### FR-14 · Recortar una línea larga seleccionada por inversión

**Dado** un Scanner con `Invert=true` y una línea de 1 MiB más un byte que no contiene el patrón,
**Cuando** se escanea la línea,
**Entonces** se entrega una sola línea al callback con número 1, exactamente su primer MiB y `truncated=true`, sin error.

> **VC-14** — Test unitario `TestScanInvertLongLine`: crear `s := search.NewScanner()`, asignar `s.Invert=true`, compilar `search.Compile("timeout", false, false)` y escanear `strings.NewReader(strings.Repeat("a", search.MaxLine+1)+"\n")`. `Scan` retorna nil y llama exactamente una vez al callback con `lineNo=1`, `text` igual byte por byte a `[]byte(strings.Repeat("a", search.MaxLine))` (1.048.576 bytes), y `truncated=true`; el callback devuelve nil. `go test ./internal/search -run '^TestScanInvertLongLine$' -count=1` sale con código `0`, sin FAIL. La entrada es sintética local; este VC no afirma haber leído el fixture grande del bucket.

#### FR-15 · Excluir una línea larga cuyo match está después del límite

**Dado** un Scanner con `Invert=true` y una línea cuyo patrón está después del primer MiB,
**Cuando** se escanea la línea completa por streaming,
**Entonces** no se llama al callback y el escaneo retorna sin error.

> **VC-15** — Test unitario `TestScanInvertMatchAfterLimit`: crear `s := search.NewScanner()`, asignar `s.Invert=true`, compilar `search.Compile("timeout", false, false)` y escanear `strings.NewReader(strings.Repeat("a", search.MaxLine)+"timeout\n")`. `Scan` retorna nil y no llama al callback (contador final igual a `0`). `go test ./internal/search -run '^TestScanInvertMatchAfterLimit$' -count=1` sale con código `0`, sin FAIL. La entrada es sintética local y no se atribuye al objeto grande no leído por MCP.

#### FR-16 · Seleccionar una línea vacía con inversión

**Dado** un Scanner con `Invert=true`, patrón literal no vacío y un input con una sola línea vacía,
**Cuando** se escanea ese input,
**Entonces** se entrega al callback exactamente una línea con número 1, texto de cero bytes y `truncated=false`, sin error.

> **VC-16** — Test unitario `TestScanInvertEmptyLine`: `s := search.NewScanner()`, `s.Invert=true`, regex de `search.Compile("timeout", false, false)` e input `strings.NewReader("\n")`. `Scan` retorna nil y llama una vez al callback con `lineNo=1`, `len(text)=0` y `truncated=false`; el callback retorna nil. `go test ./internal/search -run '^TestScanInvertEmptyLine$' -count=1 -json` sale con código `0` y contiene un evento de nivel superior `Action=pass, Test=TestScanInvertEmptyLine`, sin FAIL ni SKIP. No se atribuye esta entrada sintética a un objeto leído del bucket.

#### FR-17 · Respetar el fin de opciones antes de un patrón literal `-v`

**Dado** el parser y los args `[]string{"--", "-v", "gs://b/logs/"}`,
**Cuando** se llama a `cli.Parse` con esos args,
**Entonces** retorna error nil y `Config{Invert: false, Max: 1000, Pattern: "-v", Location: "gs://b/logs/", Extended: false, IgnoreCase: false, LineNumber: false}`: `-v` después de `--` conserva su tratamiento como patrón literal (FR-23 de la spec base).

> **VC-17** — Agregar el subtest `invert-literal-after-double-dash` a `TestParseValid`, con los args y Config exactos de FR-17. `go test ./internal/cli -run '^TestParseValid$/^invert-literal-after-double-dash$' -count=1 -json` sale con código `0` y contiene el evento `Action=pass, Test=TestParseValid/invert-literal-after-double-dash`, sin FAIL ni SKIP; la presencia del evento es obligatoria. Es un test puro de parser, sin GCS, no un test e2e nuevo.

## Reglas de negocio

### BR-1 · `-v` selecciona exactamente el complemento de lo que matchea

Con el mismo patrón y los mismos modificadores, cada línea de un objeto de texto se imprime con `-v` si y solo si no se imprime sin `-v`: las dos salidas son disjuntas y juntas cubren el objeto entero.

*Fundamento:* es la definición de `grep -v` y lo que permite razonar sobre `-v` sin leer el objeto otra vez.
*Excepciones:* ninguna.

> **VC-BR-1** — Para cada una de estas dos formas, `{ ./gcsgrep <flags> timeout gs://$B/logs/app/api.log; ./gcsgrep <flags> -v timeout gs://$B/logs/app/api.log; } | sed -E 's/^gs:\/\/[^:]+:([0-9]+):.*/\1/' | sort -n | tr '\n' ' '` imprime `1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 ` (los 15 números de línea del objeto, cada uno exactamente una vez): con `<flags>` igual a `-n`, y con `<flags>` igual a `-n -i`. En la segunda la unión sigue siendo completa, pero las líneas 7 y 10 pasan del lado de las que matchean. Además, cada una de las cuatro invocaciones de `./gcsgrep` (con y sin `-v`, en cada forma) sale con código `0` por sí sola, porque todas seleccionan al menos una línea; las cuatro dejan stderr vacío. El test las corre por separado y verifica cada código y stderr antes de unir las salidas.
> Fixture: `logs/app/api.log`, 15 líneas (con `-n` matchean la 4 y la 14; con `-n -i` también la 7 y la 10), generation `1791914566720157`, updated `2026-10-13T18:02:46.750Z`.

### BR-2 · `-v` no esconde los errores de lectura

Un objeto puntual que no existe sigue terminando la corrida con código `2`, con o sin `-v`: invertir cambia qué líneas se imprimen, no qué se considera un error. El acceso denegado lo fija BR-3; los demás errores de lectura los fija la spec base y quedan fuera de esta.

*Fundamento:* un script que usa `-v` debe distinguir "ninguna línea" (`1`) de "no pude leer" (`2`).
*Excepciones:* ninguna.

> **VC-BR-2** — `./gcsgrep -v timeout gs://$B/data/no-existe-vc-br-2.txt` sale con código `2`, stdout vacío, y stderr es exactamente `gcsgrep: not found: gs://$B/data/no-existe-vc-br-2.txt`. `./gcsgrep timeout gs://$B/data/no-existe-vc-br-2.txt` (sin `-v`) da el mismo código, el mismo stdout vacío y el mismo stderr (FR-13a de la spec base).
> Fixture: el objeto `data/no-existe-vc-br-2.txt` no existe en el bucket: el listado del prefijo `data/no-existe-vc-br-2` del 2026-10-13T18:09Z volvió vacío (ver Procedencia); no tiene `generation` ni `updated`.

### BR-3 · `-v` no amplía ni oculta un acceso denegado

Con una identidad sin permisos sobre el bucket, `-v` termina igual que sin `-v`: código `2`, sin leer ningún objeto ni imprimir líneas (BR-2 de la spec base). Invertir no convierte un acceso denegado en "ninguna línea seleccionada".

*Fundamento:* el enunciado pide fixtures para el objeto sin permisos, y un `-v` que respondiera `1` o imprimiera algo ocultaría que la identidad no pudo leer.
*Excepciones:* ninguna.

> **VC-BR-3** — Con ADC = `sin-acceso` (`GOOGLE_APPLICATION_CREDENTIALS="$GCSGREP_SIN_ACCESO"`), `./gcsgrep -v timeout gs://$B/logs/app/api.log` sale con código `2`, stdout vacío, y stderr es exactamente `gcsgrep: access denied: gs://$B/logs/app/api.log`. La misma invocación sin `-v` da el mismo código, el mismo stdout vacío y el mismo stderr (BR-2 y VC-40 de la spec base). Este VC no depende de las iteraciones 2 y 3: la base explorada ya produce ese resultado sin `-v`.
> Fixture: `logs/app/api.log` leído con la cuenta `sin-acceso` (sin rol sobre el bucket, ver `../evidencia/02-iam.md`), generation `1791914566720157`, updated `2026-10-13T18:02:46.750Z`. El 2026-10-13, `gcloud storage cat` con esa cuenta recibió `HTTPError 403` y `gcsgrep timeout` sin `-v` en la base salió con código `2` y el stderr esperado.

## Invariantes

### INV-1 · La suite unitaria sigue en verde, salvo los dos casos que dejan de esperar `-v` desconocido

Todos los tests unitarios de nivel superior de `INVERT_MATCH_BASE` siguen existiendo y en PASS, incluidos los que agreguen los prerrequisitos, y no hay ningún FAIL. Esa base conserva como mínimo los 23 tests de la base explorada. Cambian solo `unknown long flag` en `cli_test.go:61` y `TestParseErrorOrder` en `cli_test.go:93-95`, que pasan a usar `-x`. Los tests nuevos de `-v` se suman a la cuenta. La firma de `Scan` no cambia, así que los llamados existentes de `search_test.go` siguen compilando sin editarse.

> **VC-INV-1** — Las dos ejecuciones de `go test ./internal/... -json` y los comandos `jq` deben salir con código `0`; se comprueba el exit de `go test` antes de procesar cada JSON, para que el pipeline no oculte un fallo. `cd ../01-greenfield-gcsgrep && go test ./internal/... -json | jq -r 'select(.Action=="fail") | .Test // .Package'` no imprime nada, y la lista de tests de nivel superior en PASS (`go test ./internal/... -json | jq -r 'select(.Action=="pass" and .Test!=null and (.Test|contains("/")|not)) | (.Package|sub("gcsgrep/internal/";"")) + "." + .Test' | sort`) contiene como mínimo estos 23 nombres de la línea de base:
> `cli.TestParseErrorOrder`, `cli.TestParseUsageErrors`, `cli.TestParseValid`,
> `location.TestParseInvalid`, `location.TestParseValid`, `location.TestVC15aParcial`,
> `search.TestCompileEszett`, `search.TestCompileExtended`, `search.TestCompileIgnoreCase`, `search.TestCompileInvalidPattern`, `search.TestCompileLiteral`,
> `search.TestScanCRLF`, `search.TestScanCRNotFollowedByLF`, `search.TestScanEmptyObject`, `search.TestScanEndAnchorPastLimit`, `search.TestScanExactlyMaxLineNotTruncated`, `search.TestScanMatchAfterTruncation`, `search.TestScanNoTrailingNewline`, `search.TestScanOneByteOverMaxLineIsTruncated`, `search.TestScanReadErrorDuringLongLine`, `search.TestScanReadErrorMidLine`, `search.TestScanTrailingLoneCR` y `search.TestScanWordBoundaryAcrossLimit`.


VC-INV-1 también compara la **nueva base completa**, no solo esos 23 nombres. Desde `04-mcps-&-plugins/`, con el hash `INVERT_MATCH_BASE` registrado y `D=$(mktemp -d)` fuera del repositorio, `git worktree add "$D/base" "$INVERT_MATCH_BASE"` sale con código `0`. `(cd "$D/base/01-greenfield-gcsgrep" && go test ./internal/... -count=1 -json >"$D/base-unit.json")` y `(cd ../01-greenfield-gcsgrep && go test ./internal/... -count=1 -json >"$D/cambio-unit.json")` salen con código `0`, sin eventos `Action=fail` ni `Action=skip`. Con `P='select(.Action=="pass" and .Test!=null and (.Test|contains("/")|not)) | (.Package|sub("gcsgrep/internal/";"")) + "." + .Test'`, `jq -r "$P" "$D/base-unit.json" | LC_ALL=C sort >"$D/base-nombres.txt"` y `jq -r "$P" "$D/cambio-unit.json" | grep -v '^search\.TestScanInvert' | LC_ALL=C sort >"$D/cambio-nombres.txt"` producen exactamente el mismo conjunto; `diff -u "$D/base-nombres.txt" "$D/cambio-nombres.txt"` sale con código `0`, stdout/stderr vacíos. La lista de referencia incluye todos los tests agregados por las iteraciones 2 y 3, y como mínimo los 23 ya enumerados. La lista de nombres extraída de `"$D/base-unit.json"` debe coincidir exactamente con la lista por paquete registrada en la sección Línea de base al completar los prerrequisitos. Los `jq`, `sort` y `grep` salen con código `0`; verificar cada código por separado, para no ocultar errores en pipelines. Al terminar, pase o falle, `git worktree remove --force "$D/base"` sale con código `0` y limpia el worktree.

Además, VC-INV-1 comprueba los tests nuevos: `go test ./internal/search -run '^TestScanInvert' -count=1 -json` sale con código `0`; los eventos de nivel superior con `Action=pass` contienen exactamente `TestScanInvertNoMatch`, `TestScanInvertIgnoreCase`, `TestScanInvertLongLine`, `TestScanInvertMatchAfterLimit` y `TestScanInvertEmptyLine`, sin FAIL ni SKIP. `go test ./internal/cli -run '^TestParseValid$/^invert$' -count=1 -json` sale con código `0` y contiene un evento `Action=pass, Test=TestParseValid/invert`, sin FAIL ni SKIP. El subtest `TestParseValid/invert-literal-after-double-dash` también debe emitir su evento PASS mediante el comando de VC-17, sin FAIL ni SKIP. De este modo, una suite sin esos tests no satisface el VC aunque `go test -run` solo muestre que no encontró tests.

### INV-2 · Sin `-v` la salida no cambia

Una corrida sin `-v` produce los mismos bytes que antes del cambio.

> **VC-INV-2** — `./gcsgrep -n timeout gs://$B/logs/app/api.log` sale con código `0`, stderr vacío, y su stdout es exactamente `gs://$B/logs/app/api.log:4:2026-09-22 10:05:22 ERROR Request timeout after 3000ms uri=/api/v1/checkout` y `gs://$B/logs/app/api.log:14:2026-09-22 10:50:23 ERROR Gateway timeout=504 from upstream payment service`, en ese orden (el VC-8 de la spec base).
> Fixture: `logs/app/api.log`, líneas 4 y 14, generation `1791914566720157`, updated `2026-10-13T18:02:46.750Z`.

### INV-3 · Un flag desconocido distinto de `-v` sigue dando código 2

El rechazo de flags desconocidos (FR-22 de la spec base) no cambia; solo `-v` sale de esa lista.

> **VC-INV-3** — `./gcsgrep -x timeout gs://$B/logs/` sale con código `2`, stdout vacío, y stderr es exactamente `gcsgrep: unknown flag: "-x"`. La mitad del VC-22 de la spec base que usaba `-v` deja de valer; la que usa `-1]` sigue igual.

### INV-4 · Solo cambian los archivos de Dentro

La implementación en `01-greenfield-gcsgrep` toca únicamente los archivos de la tabla Dentro y conserva sin cambios las entregas previas `02-brownfield-tmux` y `03-skills-&-agents`. El toolkit y las evidencias del TP4 se mantienen en `04-mcps-&-plugins` y no forman parte de ese cambio al ejecutable.

> **VC-INV-4** — `git diff --name-only "$INVERT_MATCH_BASE" -- ../01-greenfield-gcsgrep` imprime solo rutas de esta lista: `01-greenfield-gcsgrep/internal/cli/cli.go`, `01-greenfield-gcsgrep/internal/cli/cli_test.go`, `01-greenfield-gcsgrep/internal/search/search.go`, `01-greenfield-gcsgrep/internal/search/search_test.go`, `01-greenfield-gcsgrep/cmd/gcsgrep/run.go`, `01-greenfield-gcsgrep/e2e/usage_test.go` y `01-greenfield-gcsgrep/e2e/invert_test.go`; y `git diff --name-only "$INVERT_MATCH_BASE" -- ../02-brownfield-tmux '../03-skills-&-agents'` no imprime nada. Para los archivos nuevos sin commitear, `git ls-files --others --exclude-standard --full-name -- ../01-greenfield-gcsgrep` imprime como mucho `01-greenfield-gcsgrep/e2e/invert_test.go`, y `git ls-files --others --exclude-standard -- ../02-brownfield-tmux '../03-skills-&-agents'` no imprime nada. Los cuatro comandos `git` deben salir con código `0` y stderr vacío. Ningún VC de esta spec deja archivos dentro del repositorio: los que escriben salidas usan `mktemp`.

### INV-5 · La suite e2e existente sigue en verde

Todos los tests e2e de nivel superior de `INVERT_MATCH_BASE` siguen existiendo y en PASS después del cambio, y ninguno falla. Esa base incluye como mínimo los 21 tests de la base explorada y los que agreguen los prerrequisitos. `TestVC22` cambia solo su segundo caso, de `-v` a `-x`. Los tests de `e2e/invert_test.go` se suman.

> **VC-INV-5** — Desde `04-mcps-&-plugins/`, con `INVERT_MATCH_BASE`, `GCSGREP_BUCKET`, `GCSGREP_LECTORA` y `GCSGREP_SIN_ACCESO` definidas, `D=$(mktemp -d)` (directorio nuevo y vacío, fuera del repositorio): `git worktree add "$D/base" "$INVERT_MATCH_BASE"` sale con código `0`. Después, `(cd "$D/base/01-greenfield-gcsgrep" && go test ./e2e -count=1 -json >"$D/base.json")` y `(cd ../01-greenfield-gcsgrep && go test ./e2e -count=1 -json >"$D/cambio.json")` salen con código `0`. Además:
> - `jq -r 'select(.Action=="fail" or .Action=="skip") | .Test // .Package'` no imprime nada sobre ninguno de los dos JSON: ningún test ni subtest puede fallar o saltearse.
> - Con `P='select(.Action=="pass" and .Test!=null and (.Test|contains("/")|not)) | .Test'`, `jq -r "$P" "$D/base.json" | sort` contiene todos los 21 nombres de la base explorada y coincide exactamente con la lista e2e registrada en la sección Línea de base después de completar los prerrequisitos (incluye todos los tests nuevos de sus iteraciones 2 y 3).
> - `jq -r "$P" "$D/cambio.json" | grep -v '^TestInvert' | sort` da exactamente la misma lista de referencia que `"$D/base.json"` (incluye los tests de los prerrequisitos).
> - `jq -r "$P" "$D/cambio.json" | grep '^TestInvert' | sort` da exactamente los 19 nombres de `e2e/invert_test.go` listados en Dentro.
>
> Al terminar, pase o falle, `git worktree remove --force "$D/base"` deja el repositorio sin el worktree, así que el VC se puede repetir.

## Requerimientos no funcionales

### NFR-1 · Un `-v` sobre un objeto chico termina en menos de 5 s

Con `-v` sobre un objeto de menos de 1 KiB del bucket, la corrida completa (verificar credenciales, consultar el objeto, leer, imprimir) tarda menos de 5 segundos de reloj. Condiciones: Linux, ADC con la cuenta lectora (`GOOGLE_APPLICATION_CREDENTIALS`), acceso directo a GCS sin proxy y sin otra corrida de `gcsgrep` en paralelo.

> **VC-NFR-1** — `T=$(mktemp); O=$(mktemp); ( for i in 1 2 3; do /usr/bin/time -f %e -o "$T" ./gcsgrep -v timeout gs://$B/logs/app/api.log >"$O" || exit 1; [ "$(wc -l <"$O")" -eq 13 ] || exit 1; awk '$1 >= 5.0 { exit 1 }' "$T" || exit 1; done )` sale con código `0`. Falla (código `1`) si en alguna de las tres corridas `gcsgrep` sale con un código distinto de `0`, imprime una cantidad de líneas distinta de 13 (las de VC-1) o mide `5.0` s o más.
> Fixture: `logs/app/api.log`, 906 bytes, generation `1791914566720157`, updated `2026-10-13T18:02:46.750Z`.

## Pendiente de revisión

Completar las iteraciones 2 y 3 y registrar la medición de `INVERT_MATCH_BASE` exigida en Prerrequisitos. La referencia real de las comparaciones de VC-INV-1 y VC-INV-5 todavía no existe. La spec permanece en borrador y no habilita implementar hasta completar ese registro y obtener `LISTA` en una nueva revisión.

## Preguntas abiertas

