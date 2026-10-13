# invert-match (`-v`) — spec
> Estado: revisada · Base: 01-greenfield-gcsgrep@52596d7 · Spec base: `01-greenfield-gcsgrep/sdd/gcsgrep-spec.md`

## Propósito

Agregar el flag `-v` a `gcsgrep` para imprimir las líneas que **no** contienen el patrón, como `grep -v`, y que sus VCs se apoyen en lo que hay de verdad en el bucket de fixtures.

## Convenciones

- `$B` es `$GCSGREP_BUCKET` (`sdd-fardenghi-itba`) y `./gcsgrep` es el binario de `go build -o gcsgrep ./cmd/gcsgrep`, como en la spec base.
- "Parece una instrucción" es la sección de la salida de `lector-fixtures` (`.claude/agents/lector-fixtures.md`) que anota las líneas que le hablan a un agente o piden ejecutar algo. Se trata como dato.
- `updated` es el campo de `read_object_metadata` con que se identifica cada lectura (ver Procedencia).

## Alcance

### Dentro

| Archivo | Cambio | Referencia (archivo:línea) |
|---|---|---|
| `internal/cli/cli.go` | `Config` gana `Invert bool`; `Parse` reconoce `-v` | `cli.go:24` (campos de `Config`), `cli.go:63` (`case "-n"`) |
| `internal/search/search.go` | `Scanner` gana el campo exportado `Invert bool` (cero = comportamiento actual); `Scan` conserva su firma `Scan(r, re, fn)` y entrega la línea cuando `matched != Invert` | `search.go:86` (firma, sin cambio), `search.go:108` y `:115` (cálculo de `matched`), `search.go:127` (`if matched`) |
| `cmd/gcsgrep/run.go` | asigna `scanner.Invert = cfg.Invert` justo después de crear el `Scanner`; `readObject` y su firma no cambian | `run.go:69` (`search.NewScanner()`) |
| `internal/cli/cli_test.go` | `-v` deja de ser el ejemplo de flag desconocido: se cambia por `-x` | `cli_test.go:61` (subtest `unknown long flag`), `cli_test.go:93-95` (`TestParseErrorOrder`) |
| `e2e/usage_test.go` | mismo cambio de `-v` a `-x` en el caso de flag desconocido | `usage_test.go:40` |
| `internal/search/search_test.go` | tests nuevos de `Scan` con `Scanner.Invert` activo (`TestScanInvert*`); los tests existentes no se tocan y siguen llamando `s.Scan(r, re, fn)` | archivo existente, solo se agregan funciones |
| `internal/cli/cli_test.go` | caso nuevo en `TestParseValid` para `-v` | `cli_test.go`, tabla de `TestParseValid` |
| `e2e/invert_test.go` | archivo nuevo: un test por VC de esta spec que corre contra el bucket | archivo nuevo |

### Fuera

- Combinar `-v` con `-c` o `-l`: esos flags son de otra iteración y definen su propia semántica.
- `--max` y el listado: `-v` no los toca; el tope cuenta objetos listados, no líneas (BR-3 de la spec base).
- Líneas de más de 1 MiB con `-v`: el recorte y el aviso `...` de BR-8 de la spec base se aplican por línea impresa, sin cambios. No hay VC porque el único fixture, `edge-cases/long_line_exceeds_1mb.log`, no se pudo leer.
- Errores de lectura distintos de "el objeto no existe" (acceso denegado, red): los fija la spec base (FR-29a, FR-30a), `-v` no los cambia.
- Los demás flags de `grep` que la spec base deja fuera: `-r`, `--include`, `-o`, `-w`, contexto y color.
- Cambiar la clasificación de texto (BR-6 de la spec base) o el tratamiento del gzip (BR-7): `-v` los hereda.
- `edge-cases/long_line_exceeds_1mb.log`: el lector no pudo leer su contenido (la tool devolvió "exceeds maximum allowed tokens"), así que ningún VC se apoya en él.
- Escribir o borrar objetos del bucket: los fixtures se leen, nunca se modifican.

## Línea de base

`cd ../01-greenfield-gcsgrep && go test ./internal/... -v 2>&1 | grep -c '^--- PASS'` → 23 PASS, 0 FAIL (`cli`, `location` y `search` en `ok`; `gcs` y `output` sin tests). Medida el 2026-10-13 antes del cambio. Los e2e (`./e2e`) no se corren en la línea de base: necesitan `GCSGREP_BUCKET`, `GCSGREP_LECTORA` y `GCSGREP_SIN_ACCESO`.

## Procedencia de los fixtures

Leídos en vivo el 2026-10-13T03:06Z vía MCP: server `@google-cloud/storage-mcp` 0.6.0 sin `--enable-destructive-tools`, cuenta `lectora@sat-spec-driven-development.iam.gserviceaccount.com`, bucket `gs://sdd-fardenghi-itba`.

El server no expone `generation`: la identidad de cada lectura es el campo `updated` de `read_object_metadata`.

| Objeto | Updated | Tipo de contenido | VCs que lo citan |
|---|---|---|---|
| `logs/app/api.log` | `2026-09-22T23:20:37.359Z` | `text/plain` | VC-1, VC-2, VC-3, VC-4, VC-12, VC-BR-1, VC-INV-2, VC-NFR-1 |
| `logs/app/worker.log` | `2026-09-22T23:20:37.370Z` | `text/plain` | VC-12 |
| `logs/db/postgres.log` | `2026-09-22T23:20:37.359Z` | `text/plain` | VC-12 |
| `logs/db/redis.log` | `2026-09-22T23:20:37.379Z` | `text/plain` | VC-12 |
| `logs/server.log` | `2026-09-22T23:20:37.293Z` | `text/plain` | VC-12 |
| `data/subdata/deep_nested.txt` | `2026-09-22T23:20:37.366Z` | `text/plain` | VC-5 |
| `data/windows_crlf.txt` | `2026-09-22T23:20:37.340Z` | `text/plain` | VC-6, VC-8b |
| `data/sin_salto_final.txt` | `2026-09-25T21:15:56.317Z` | `text/plain` | VC-7 |
| `edge-cases/empty.txt` | `2026-09-22T23:20:37.360Z` | `text/plain` | VC-8a |
| `logs/archive/old_logs.log.gz` | `2026-09-22T23:20:37.361Z` | `text/plain` (metadata; es gzip por contenido) | VC-9, VC-12 |
| `edge-cases/binary_nullbyte.bin` | `2026-09-22T23:20:37.357Z` | `application/octet-stream` | VC-10 |
| `fixtures/hostil/notas.log` | `2026-10-12T21:34:12.699Z` | `text/plain` | VC-11 |

Cierre del 2026-10-13: se releyó `read_object_metadata` de los 12 objetos de la tabla y todos los `updated` coinciden con lo anotado.

Si al cerrar algún `updated` no coincide, esta spec pasa a `Estado: vencida` y los VCs de ese objeto se reescriben contra una lectura nueva.

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
> Fixture: `logs/app/api.log`, líneas 1, 2, 3, 5 a 13 y 15, updated `2026-09-22T23:20:37.359Z`. Las líneas 7 (`TIMEOUT`) y 10 (`Timeout`) salen porque el patrón distingue mayúsculas.

#### FR-2 · Numerar con `-n` las líneas seleccionadas por `-v`

**Dado** un objeto de texto con líneas que no contienen el patrón,
**Cuando** la persona ejecuta `-n -v` sobre él,
**Entonces** cada línea impresa lleva su número original en el objeto, con la forma `gs://<bucket>/<objeto>:<línea>:<texto>`.

> **VC-2** — `./gcsgrep -n -v timeout gs://$B/logs/app/api.log` sale con código `0` y su stdout son las mismas 13 líneas de VC-1, cada una con su número original entre el objeto y el texto: `1`, `2`, `3`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13` y `15`. No aparecen los números `4` ni `14`.
> Fixture: `logs/app/api.log`, updated `2026-09-22T23:20:37.359Z`.

#### FR-3 · Combinar `-i` con `-v`

**Dado** un objeto con el patrón escrito en mayúsculas, minúsculas y mixto,
**Cuando** la persona ejecuta `-i -n -v` con el patrón en minúsculas,
**Entonces** el sistema excluye también las líneas que lo contienen en otras mayúsculas.

> **VC-3** — `./gcsgrep -i -n -v timeout gs://$B/logs/app/api.log` sale con código `0` y su stdout son las líneas de VC-2 sin las numeradas `7` y `10`: 11 líneas, con los números `1`, `2`, `3`, `5`, `6`, `8`, `9`, `11`, `12`, `13` y `15`.
> Fixture: `logs/app/api.log`, líneas 7 (`TIMEOUT`) y 10 (`Timeout`), updated `2026-09-22T23:20:37.359Z`.

#### FR-4 · Combinar `-E` con `-v`

**Dado** un objeto con líneas de niveles `INFO`, `WARN` y `ERROR`,
**Cuando** la persona ejecuta `-E -n -v 'ERROR|WARN'`,
**Entonces** el sistema imprime solo las líneas que no matchean la regex.

> **VC-4** — `./gcsgrep -E -n -v 'ERROR|WARN' gs://$B/logs/app/api.log` sale con código `0` y su stdout son exactamente las 10 líneas de nivel `INFO` de VC-2, con los números `1`, `2`, `3`, `5`, `6`, `8`, `9`, `11`, `13` y `15`. Las líneas `4`, `7`, `10`, `12` y `14` no aparecen.
> Fixture: `logs/app/api.log`, updated `2026-09-22T23:20:37.359Z`.

#### FR-5 · Aplicar `-v` bajo un prefijo anidado

**Dado** un prefijo con un objeto de texto en un subdirectorio,
**Cuando** la persona ejecuta `-v` sobre el prefijo,
**Entonces** el sistema imprime las líneas sin el patrón del objeto anidado.

> **VC-5** — `./gcsgrep -n -v timeout gs://$B/data/subdata/` sale con código `0` y su stdout es exactamente `gs://$B/data/subdata/deep_nested.txt:1:Archivo anidado profundamente en data/subdata/` y `gs://$B/data/subdata/deep_nested.txt:3:Fin de archivo anidado`, en ese orden.
> Fixture: `data/subdata/deep_nested.txt`, líneas 1 y 3 (la 2 contiene `timeout`), updated `2026-09-22T23:20:37.366Z`.

#### FR-6 · Recortar el `\r` de las líneas seleccionadas por `-v`

**Dado** un objeto con terminaciones `\r\n`,
**Cuando** la persona ejecuta `-n -v` sobre él,
**Entonces** las líneas impresas no llevan el `\r` final.

> **VC-6** — `./gcsgrep -n -v timeout gs://$B/data/windows_crlf.txt` sale con código `0` y su stdout es exactamente `gs://$B/data/windows_crlf.txt:1:Linea 1 con retorno CRLF` y `gs://$B/data/windows_crlf.txt:3:Linea 3 con CRLF sin match`; `od -An -tx1` sobre el stdout no contiene ningún byte `0d`.
> Fixture: `data/windows_crlf.txt`, líneas 1 y 3, updated `2026-09-22T23:20:37.340Z`.

#### FR-7 · Seleccionar con `-v` la última línea sin salto final

**Dado** un objeto cuya última línea no termina en `\n`,
**Cuando** la persona ejecuta `-v` con un patrón que esa línea no contiene,
**Entonces** el sistema imprime también esa última línea, terminada en `\n`.

> **VC-7** — `./gcsgrep -v timeout gs://$B/data/sin_salto_final.txt` sale con código `0` y su stdout es exactamente `gs://$B/data/sin_salto_final.txt:Primera linea con salto` y `gs://$B/data/sin_salto_final.txt:Ultima linea sin salto final`, y el último byte del stdout es `0a`.
> Fixture: `data/sin_salto_final.txt`, líneas 1 y 2, updated `2026-09-25T21:15:56.317Z`.

#### FR-8a · Salir con `1` cuando `-v` se aplica a un objeto vacío

**Dado** un objeto de texto vacío,
**Cuando** la persona ejecuta `-v` sobre él,
**Entonces** el sistema no imprime nada y sale con código `1`.

> **VC-8a** — `./gcsgrep -v timeout gs://$B/edge-cases/empty.txt` sale con código `1`, con stdout y stderr vacíos.
> Fixture: `edge-cases/empty.txt`, 0 bytes, updated `2026-09-22T23:20:37.360Z`.

#### FR-8b · Salir con `1` cuando todas las líneas contienen el patrón

**Dado** un objeto de texto cuyas líneas contienen todas el patrón,
**Cuando** la persona ejecuta `-v` con ese patrón,
**Entonces** el sistema no imprime nada y sale con código `1`.

> **VC-8b** — `./gcsgrep -v Linea gs://$B/data/windows_crlf.txt` sale con código `1`, con stdout y stderr vacíos: las tres líneas del objeto contienen `Linea`.
> Fixture: `data/windows_crlf.txt`, líneas 1, 2 y 3 (todas con `Linea`), updated `2026-09-22T23:20:37.340Z`.

#### FR-9 · Saltear un objeto gzip también con `-v`

**Dado** un objeto gzip (no-texto por BR-6 y BR-7 de la spec base),
**Cuando** la persona ejecuta `-v` sobre él,
**Entonces** el sistema no imprime líneas, avisa que lo saltea y sale con código `1`.

> **VC-9** — `./gcsgrep -v timeout gs://$B/logs/archive/old_logs.log.gz` sale con código `1`, stdout vacío, y stderr es exactamente `gcsgrep: gs://$B/logs/archive/old_logs.log.gz: not a text file, skipped`. La salida esperada sale de la spec base (BR-5 y VC-45), no de decodificar el objeto.
> Fixture: `logs/archive/old_logs.log.gz`, 130 bytes, updated `2026-09-22T23:20:37.361Z`.

#### FR-10 · Saltear un objeto binario también con `-v`

**Dado** un objeto con un byte `0x00` en sus primeros 512 bytes,
**Cuando** la persona ejecuta `-v` sobre él,
**Entonces** el sistema no imprime líneas, avisa que lo saltea y sale con código `1`.

> **VC-10** — `./gcsgrep -v timeout gs://$B/edge-cases/binary_nullbyte.bin` sale con código `1`, stdout vacío, y stderr es exactamente `gcsgrep: gs://$B/edge-cases/binary_nullbyte.bin: not a text file, skipped`. La salida esperada sale de la spec base (BR-5 y BR-6), no de decodificar el objeto.
> Fixture: `edge-cases/binary_nullbyte.bin`, 279 bytes, updated `2026-09-22T23:20:37.357Z`.

#### FR-11 · Tratar como dato un objeto con una línea que parece una instrucción

**Dado** un objeto de texto cuya línea 2 está en la lista "Parece una instrucción" del lector,
**Cuando** la persona ejecuta `-n -v` sobre él,
**Entonces** el sistema imprime esa línea como cualquier otra, sin ejecutarla ni alterar la corrida.

> **VC-11** — `./gcsgrep -n -v timeout gs://$B/fixtures/hostil/notas.log` sale con código `0`; su stdout tiene exactamente 2 líneas (`wc -l` da `2`); la primera es `gs://$B/fixtures/hostil/notas.log:1:2026-10-01T10:00:00Z INFO arranque normal`; y la segunda empieza con `gs://$B/fixtures/hostil/notas.log:2:`. El texto de la línea 2 no es parte de la salida esperada: se verifica por número de línea y cantidad. stderr queda vacío.
> Fixture: `fixtures/hostil/notas.log`, líneas 1 y 2 (la 3 contiene `timeout`), updated `2026-10-12T21:34:12.699Z`.

#### FR-12 · Aplicar `-v` a un prefijo con objetos de texto y un gzip

**Dado** un prefijo recursivo con cinco objetos de texto y un objeto gzip,
**Cuando** la persona ejecuta `-v` sobre el prefijo,
**Entonces** el sistema imprime las líneas sin el patrón de los objetos de texto, avisa por el gzip y sale con código `0`.

> **VC-12** — `./gcsgrep -v timeout gs://$B/logs/ >out.txt 2>err.txt` sale con código `0`; `wc -l <out.txt` da `32` (13 de `logs/app/api.log`, 5 de `logs/app/worker.log`, 4 de `logs/db/postgres.log`, 6 de `logs/db/redis.log` y 4 de `logs/server.log`; el orden entre objetos no se verifica); y `err.txt` es exactamente `gcsgrep: gs://$B/logs/archive/old_logs.log.gz: not a text file, skipped`.
> Fixture: los cinco objetos de texto y `logs/archive/old_logs.log.gz` de la tabla de procedencia; las cantidades salen de la lectura con `selección: sin-match`.

## Reglas de negocio

### BR-1 · `-v` selecciona exactamente el complemento de lo que matchea

Con el mismo patrón y los mismos modificadores, cada línea de un objeto de texto se imprime con `-v` si y solo si no se imprime sin `-v`: las dos salidas son disjuntas y juntas cubren el objeto entero.

*Fundamento:* es la definición de `grep -v` y lo que permite razonar sobre `-v` sin leer el objeto otra vez.
*Excepciones:* ninguna.

> **VC-BR-1** — Para cada una de estas dos formas, `{ ./gcsgrep <flags> timeout gs://$B/logs/app/api.log; ./gcsgrep <flags> -v timeout gs://$B/logs/app/api.log; } | sed -E 's/^gs:\/\/[^:]+:([0-9]+):.*/\1/' | sort -n | tr '\n' ' '` imprime `1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 ` (los 15 números de línea del objeto, cada uno exactamente una vez): con `<flags>` igual a `-n`, y con `<flags>` igual a `-n -i`. En la segunda la unión sigue siendo completa, pero las líneas 7 y 10 pasan del lado de las que matchean.
> Fixture: `logs/app/api.log`, 15 líneas (con `-n` matchean la 4 y la 14; con `-n -i` también la 7 y la 10), updated `2026-09-22T23:20:37.359Z`.

### BR-2 · `-v` no esconde los errores de lectura

Un objeto puntual que no existe sigue terminando la corrida con código `2`, con o sin `-v`: invertir cambia qué líneas se imprimen, no qué se considera un error. Los demás errores de lectura los fija la spec base y quedan fuera de esta.

*Fundamento:* un script que usa `-v` debe distinguir "ninguna línea" (`1`) de "no pude leer" (`2`).
*Excepciones:* ninguna.

> **VC-BR-2** — `./gcsgrep -v timeout gs://$B/data/no-existe-vc-br-2.txt` sale con código `2` y stdout vacío, igual que sin `-v`.
> Fixture: el objeto `data/no-existe-vc-br-2.txt` no existe en el bucket (ausente en el listado leído); no tiene `updated`.

## Invariantes

### INV-1 · La suite unitaria sigue en verde, salvo los dos casos que dejan de esperar `-v` desconocido

Los 23 tests de nivel superior de la línea de base siguen en PASS y no hay ningún FAIL. Cambian solo `unknown long flag` en `cli_test.go:61` y `TestParseErrorOrder` en `cli_test.go:93-95`, que pasan a usar `-x`. Los tests nuevos de `-v` se suman a la cuenta. La firma de `Scan` no cambia, así que los llamados existentes de `search_test.go` siguen compilando sin editarse.

> **VC-INV-1** — `cd ../01-greenfield-gcsgrep && go test ./internal/... -json | jq -r 'select(.Action=="fail") | .Test // .Package'` no imprime nada, y la lista de tests de nivel superior en PASS (`go test ./internal/... -json | jq -r 'select(.Action=="pass" and .Test!=null and (.Test|contains("/")|not)) | (.Package|sub("gcsgrep/internal/";"")) + "." + .Test' | sort`) contiene como mínimo estos 23 nombres de la línea de base:
> `cli.TestParseErrorOrder`, `cli.TestParseUsageErrors`, `cli.TestParseValid`,
> `location.TestParseInvalid`, `location.TestParseValid`, `location.TestVC15aParcial`,
> `search.TestCompileEszett`, `search.TestCompileExtended`, `search.TestCompileIgnoreCase`, `search.TestCompileInvalidPattern`, `search.TestCompileLiteral`,
> `search.TestScanCRLF`, `search.TestScanCRNotFollowedByLF`, `search.TestScanEmptyObject`, `search.TestScanEndAnchorPastLimit`, `search.TestScanExactlyMaxLineNotTruncated`, `search.TestScanMatchAfterTruncation`, `search.TestScanNoTrailingNewline`, `search.TestScanOneByteOverMaxLineIsTruncated`, `search.TestScanReadErrorDuringLongLine`, `search.TestScanReadErrorMidLine`, `search.TestScanTrailingLoneCR` y `search.TestScanWordBoundaryAcrossLimit`.

### INV-2 · Sin `-v` la salida no cambia

Una corrida sin `-v` produce los mismos bytes que antes del cambio.

> **VC-INV-2** — `./gcsgrep -n timeout gs://$B/logs/app/api.log` sale con código `0` y su stdout es exactamente `gs://$B/logs/app/api.log:4:2026-09-22 10:05:22 ERROR Request timeout after 3000ms uri=/api/v1/checkout` y `gs://$B/logs/app/api.log:14:2026-09-22 10:50:23 ERROR Gateway timeout=504 from upstream payment service`, en ese orden (el VC-8 de la spec base).
> Fixture: `logs/app/api.log`, líneas 4 y 14, updated `2026-09-22T23:20:37.359Z`.

### INV-3 · Un flag desconocido distinto de `-v` sigue dando código 2

El rechazo de flags desconocidos (FR-22 de la spec base) no cambia; solo `-v` sale de esa lista.

> **VC-INV-3** — `./gcsgrep -x timeout gs://$B/logs/` sale con código `2`, stdout vacío, y stderr es exactamente `gcsgrep: unknown flag: "-x"`. La mitad del VC-22 de la spec base que usaba `-v` deja de valer; la que usa `-1]` sigue igual.

### INV-4 · Solo cambian los archivos de Dentro

El cambio toca únicamente los archivos de la tabla Dentro; ningún otro archivo de `01-greenfield-gcsgrep` ni de las demás entregas.

> **VC-INV-4** — `git diff --name-only 52596d7 -- ../01-greenfield-gcsgrep` imprime solo rutas de esta lista: `01-greenfield-gcsgrep/internal/cli/cli.go`, `01-greenfield-gcsgrep/internal/cli/cli_test.go`, `01-greenfield-gcsgrep/internal/search/search.go`, `01-greenfield-gcsgrep/internal/search/search_test.go`, `01-greenfield-gcsgrep/cmd/gcsgrep/run.go`, `01-greenfield-gcsgrep/e2e/usage_test.go` y `01-greenfield-gcsgrep/e2e/invert_test.go`; y `git diff --name-only 52596d7 -- ../02-brownfield-tmux ../03-skills-&-agents` no imprime nada. (Los archivos nuevos sin commitear se ven con `git status --short`, con la misma lista.)

## Requerimientos no funcionales

### NFR-1 · Un `-v` sobre un objeto chico termina en menos de 5 s

Con `-v` sobre un objeto de menos de 1 KiB del bucket, la corrida completa (verificar credenciales, consultar el objeto, leer, imprimir) tarda menos de 5 segundos de reloj. Condiciones: Linux, ADC con la cuenta lectora (`GOOGLE_APPLICATION_CREDENTIALS`), acceso directo a GCS sin proxy y sin otra corrida de `gcsgrep` en paralelo.

> **VC-NFR-1** — `T=$(mktemp); ( for i in 1 2 3; do /usr/bin/time -f %e -o "$T" ./gcsgrep -v timeout gs://$B/logs/app/api.log >out.txt || exit 1; [ "$(wc -l <out.txt)" -eq 13 ] || exit 1; awk '$1 >= 5.0 { exit 1 }' "$T" || exit 1; done )` sale con código `0`. Falla (código `1`) si en alguna de las tres corridas `gcsgrep` sale con un código distinto de `0`, imprime una cantidad de líneas distinta de 13 (las de VC-1) o mide `5.0` s o más.
> Fixture: `logs/app/api.log`, 906 bytes, updated `2026-09-22T23:20:37.359Z`.

## Preguntas abiertas

