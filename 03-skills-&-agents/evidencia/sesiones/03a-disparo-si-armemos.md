## Sesión nueva · cwd=<sandbox> · modelo=claude-opus-5-5

→ **Skill** {"skill":"write-spec","args":"gcsgrep: agregar flag --max-bytes que corte la lectura de cada objeto después de N bytes (brownfield, 01-greenfield-gcsgrep)"}

  ← resultado: Launching skill: write-spec

→ **Agent** subagent_type=Explore · prompt: Exploración de solo lectura (medium) en <sandbox>/01-greenfield-gcsgrep (CLI en Go que hace grep sobre objetos de GCS). Vamos a especificar un n …[recortado]

  ← resultado: Async agent launched successfully. (This tool result is internal metadata — never quote or paste any part of it, including the agentId below, into a user-facing reply.) ⏎ agentId: a5b6f2c5d60fa7245 (internal ID - do not  …[recortado]

→ **Bash** `cd 01-greenfield-gcsgrep && git rev-parse --short HEAD && ls sdd 2>/dev/null; go build ./... && go test ./internal/... 2>&1 | tail -20; go test -v ./internal/... 2>&1 | grep -cE '^(=== RUN|--- PASS)' ; go test -v ./internal/... 2>&1 | grep -E '^\s*--- (PASS|FAIL)' | awk '{print $2}' | sort | uniq -c`

    │ [subagent] → **Bash** `cd <sandbox>/01-greenfield-gcsgrep && find . -type f -not -path './.git/*' | head -100 && wc -l $(find . -name '*.go' -o -name '*.md' | grep -v  …[recortado]`

    │ [subagent]   ← resultado: ./go.mod ⏎ ./gcsgrep ⏎ ./go.sum ⏎ ./.gitignore ⏎ ./gcsgrep-requirements.md ⏎ ./docs/Lección 1 — Agentes de código y SDD greenfield (1).pdf ⏎ ./docs/guia-sdd.md ⏎ ./enunciado/revisi_n_de_spec.md ⏎ ./enunciado/enunciado.md …[recortado]

    │ [subagent] **Agente:** Code is small; read the Go files.

    │ [subagent] → **Bash** `cd <sandbox>/01-greenfield-gcsgrep && for f in internal/cli/cli.go cmd/gcsgrep/main.go cmd/gcsgrep/run.go internal/search/search.go internal/gcs …[recortado]`

  ← resultado: 22104ba ⏎ CONTEXT.md ⏎ DECISIONS.md ⏎ gcsgrep-base-context.md ⏎ gcsgrep-cobertura-vc.md ⏎ gcsgrep-plan.md ⏎ gcsgrep-spec.md ⏎ iterations ⏎ ok  	gcsgrep/internal/cli	(cached) ⏎ ?   	gcsgrep/internal/gcs	[no test files] ⏎  …[recortado]

    │ [subagent]   ← resultado: === internal/cli/cli.go ⏎      1	// Package cli parsea argv y valida flags, patrón (longitud) y cantidad de ⏎      2	// posicionales, sin tocar GCS: todo lo que se puede decidir con argv se ⏎      3	// decide acá. El pat …[recortado]

    │ [subagent] → **Bash** `cd <sandbox>/01-greenfield-gcsgrep && grep -n "^func Test\|t.Run(\"" internal/*/*_test.go e2e/*.go; echo; sed -n 1,80p e2e/main_test.go`

    │ [subagent]   ← resultado: internal/location/location_test.go:5:func TestParseValid(t *testing.T) { ⏎ internal/location/location_test.go:36:func TestVC15aParcial(t *testing.T) { ⏎ internal/location/location_test.go:47:func TestParseInvalid(t *test …[recortado]

→ **Bash** `go test -count=1 -v ./internal/... 2>&1 | grep -E -- '--- (PASS|FAIL|SKIP)' | awk '{print $2}' | sort | uniq -c; go test -count=1 -v ./internal/... 2>&1 | grep -cE -- '^--- PASS'; ls ../.claude/skills/write-spec/scripts/`

    │ [subagent] → **Bash** `cd <sandbox>/01-greenfield-gcsgrep && sed -n 80,276p e2e/main_test.go | grep -n "func \|STORAGE_EMULATOR\|net\.\|Listen" ; echo; cat -n e2e/part …[recortado]`

    │ [subagent]   ← resultado: 1:func snapshotBucket() (string, error) { ⏎ 27:// cero: sin LANG ni STORAGE_EMULATOR_HOST salvo que env los agregue. ⏎ 31:	env   []string // variables extra, p. ej. "STORAGE_EMULATOR_HOST=...", "LANG=C" ⏎ 40:func runTool …[recortado]

  ← resultado:   43 PASS: ⏎ 23 ⏎ vc-coverage.sh

    │ [subagent] → **Bash** `cd <sandbox>/01-greenfield-gcsgrep && grep -n "^#\|^\*\*\(FR\|BR\|NFR\|VC\|I\)-\|^- \*\*\(FR\|BR\|NFR\|VC\)\|^| *\(FR\|BR\|NFR\)" sdd/gcsgrep-sp …[recortado]`

    │ [subagent]   ← resultado: 1:# gcsgrep — spec ⏎ 18:## Propósito ⏎ 24:## Alcance ⏎ 26:### Dentro ⏎ 39:### Fuera ⏎ 66:### Restricciones técnicas ⏎ 73:## Actores ⏎ 82:## Entorno de verificación ⏎ 142:## Requerimientos funcionales ⏎ 144:### Búsqueda ⏎ …[recortado]

    │ [subagent] → **Bash** `cd <sandbox>/01-greenfield-gcsgrep && sed -n 1,72p sdd/gcsgrep-spec.md; echo ----; sed -n 146,160p sdd/gcsgrep-spec.md; echo ----; sed -n 449,76 …[recortado]`

    │ [subagent]   ← resultado: # gcsgrep — spec ⏎  ⏎ > **Estado: revisada.** Sin preguntas abiertas ni requisitos no atómicos. Habilitada ⏎ > para Planificar/Implementar. ⏎ > ⏎ > Construida a partir de [`gcsgrep-base-context.md`](./gcsgrep-base-contex …[recortado]

    │ [subagent] → **Bash** `cd <sandbox>/01-greenfield-gcsgrep && sed -n 997,1162p sdd/gcsgrep-spec.md; sed -n 1228,1246p sdd/gcsgrep-spec.md`

    │ [subagent]   ← resultado: ### BR-5 · Los objetos que no son texto se saltean sin contar como fallo ⏎  ⏎ Un objeto clasificado como no-texto (BR-6) no se busca: se escribe ⏎ `gcsgrep: gs://<bucket>/<objeto>: not a text file, skipped` por stderr y  …[recortado]

    │ [subagent] → **Bash** `cd <sandbox>/01-greenfield-gcsgrep && grep -n -i "por bytes\|bytes leídos\|límite por\|max-bytes\|byte limit" sdd/*.md sdd/iterations/*.md gcsgr …[recortado]`

    │ [subagent]   ← resultado: sdd/gcsgrep-plan.md:463:| Límite por bytes leídos | Diferido post-v1, como complemento del tope por objetos | ⏎ sdd/gcsgrep-spec.md:53:- Un límite por **bytes** leídos. El guardrail solo cuenta objetos (BR-3). ⏎ sdd/gcsg …[recortado]

    │ [subagent] → **Bash** `cd <sandbox>/01-greenfield-gcsgrep && sed -n 1,20p sdd/CONTEXT.md; sed -n 76,104p sdd/DECISIONS.md; sed -n 454,475p sdd/gcsgrep-plan.md; sed -n  …[recortado]`

    │ [subagent]   ← resultado: # gcsgrep — contexto ⏎  ⏎ > Se actualiza al cerrar cada iteración. Para el detalle de una decisión, ver ⏎ > `DECISIONS.md`; para lo que pasó en una iteración puntual, ver `iterations/NN-*.md`. ⏎  ⏎ ## Estado ⏎  ⏎ **Itera …[recortado]

    │ [subagent] → **Bash** `cd <sandbox>/01-greenfield-gcsgrep && sed -n 238,262p sdd/gcsgrep-plan.md; sed -n 52,97p internal/cli/cli_test.go; sed -n 60,78p test-fixtures/R …[recortado]`

    │ [subagent]   ← resultado: ## Iteración 2 — Servidor de prueba, sniffing y formatos de salida ⏎  ⏎ **Objetivo:** que la herramienta sirva sobre un bucket real, donde hay binarios, ⏎ gzip y carpetas, que tenga los dos formatos de resumen de `grep`, …[recortado]

    │ [subagent] **Agente:** Lo más importante primero: la spec actual deja **explícitamente fuera** un límite por bytes. Así que `--max-bytes` no entra como un FR más: hay que revisar la spec. Además, el código solo cubre la Iteración 1 (secuencial, sin `-c`/`-l`, sin `--concurrency`, sin sniff de binarios ni timeouts). Buena parte de lo que pide la spec todavía no existe en el código.

Raíz: `<sandbox>/01-greenfield-gcsgrep` (abajo, rutas relativas a esa raíz).

## 1. Parsing de flags
- **Librería:** ninguna. Es un parser a mano en `internal/cli/cli.go:45-100`: un `switch` de string exacto (`:58`), así que `--max-bytes` hoy cae en `unknown flag` y no choca con `--max`. `--` se maneja en `:53`.
- **Config:** el struct está en `cli.go:21-28` (`Extended`, `IgnoreCase`, `LineNumber`, `Max int64`, `Pattern`, `Location`). `Unlimited = math.MaxInt64` en `:15`, el default 1000 en `:18`.
- **Errores de uso:** son del tipo `*ErrUsage` (`cli.go:32-38`), sin el prefijo `gcsgrep: `. El prefijo lo agrega `output.Warn` (`internal/output/output.go:37-40`) vía `fail()` en `cmd/gcsgrep/run.go:129-132`, que siempre devuelve 2. Todo error de uso, de regex o de ubicación sale con exit 2, antes de tocar GCS (`run.go:25-36`).
- **Mensajes literales que hay hoy:**
  - `flag --max requires a value` (`cli.go:68`)
  - `invalid value for --max: %q (integer >= 1 or unlimited)` (`:73`)
  - `unknown flag: %q` (`:77`)
  - `expected 2 arguments (pattern and location), got %d` (`:85`)
  - `empty pattern` (`:89`)
  - `invalid pattern: %s` (`internal/search/search.go:29`)
- **Mensajes que la spec ya fija pero el código todavía no implementa:**
  - FR-24a/b: `unknown flag: "<arg>" (pass each flag separately, values after a space)` (incluye `--max=5`)
  - FR-25: `flag specified more than once: <flag>`
  - FR-26c: `invalid value for --concurrency: "<v>" (integer between 1 and 32)`
  - FR-26d: `flag --concurrency requires a value`
  - FR-20a-c: `flags <a> and <b> cannot be used together`
- **Validación de valores:** `parseMax` (`cli.go:106-131`) acepta solo dígitos (`+5` y ` 5` no valen), toma siempre el argumento siguiente aunque empiece con `-`, y un desborde se trata como `Unlimited`. Es el patrón natural a copiar para `--max-bytes`.
- **Decisiones del parser:** D-08 (`sdd/DECISIONS.md:83-102`). Hoy se aceptan flags repetidos y gana el último; FR-25 los rechaza recién en la Iteración 4. El orden del error único es: flags en orden de argv → cantidad de posicionales → patrón vacío → regex → ubicación.

## 2. Lectura de objetos
- **No hay interfaz para fakes.** `internal/gcs/gcs.go` usa directamente `*storage.Client`. La apertura es `gcs.Open` (`gcs.go:96-98`), un `NewReader` del objeto completo, sin rangos.
- **Configuración del cliente:** `WithJSONReads()` y `RetryNever` (`gcs.go:39-50`).
- **Dónde se lee cada objeto:** `readObject` (`cmd/gcsgrep/run.go:96-105`) abre, hace `defer rc.Close()` y llama a `scanner.Scan(rc, …)`. Es el punto natural para envolver el reader con `io.LimitReader`.
- **No usa `bufio.Scanner`.** `search.Scanner` (`search.go:65-137`) trabaja sobre un `bufio.Reader` de 64 KiB (`:73`) y lee byte a byte con `lineState.nextByte` (`:153-197`):
  - recorta el `\r` antes de `\n` (FR-9a);
  - busca la última línea aunque no termine en `\n` (FR-9b);
  - `MaxLine = 1 MiB` (`:16`). Retiene MaxLine+1 bytes y, si la línea es más larga, sigue el match por streaming con un `runeReader` propio (`:199-253`) y la imprime truncada con `...` (BR-8, D-10).
- **Errores de lectura:** un error que no sea `io.EOF` se devuelve tal cual y la línea en curso no se evalúa (`:122-124`, D-12). Un `io.EOF` se toma como fin limpio.
- **Binarios y gzip:** no hay detección de binarios ni descompresión gzip en el código. El sniff de 512 bytes (BR-5/6/7) es de la Iteración 2. Por spec, el gzip **no** se descomprime: se saltea como no-texto. La excepción es el *decompressive transcoding* que hace GCS (BR-7, `sdd/gcsgrep-spec.md:1039-1056`).
- **Cómo se testea la lectura:** `search.Scan` recibe un `io.Reader`. Los unit tests usan `strings.Reader`, `io.MultiReader` y `errAfterReader` (`internal/search/search_test.go:270-284`). No hay fake de GCS en Go. El "servidor de prueba" HTTP (`STORAGE_EMULATOR_HOST`) está planeado para la Iteración 2.

## 3. Salida, concurrencia y exit code
- **Formato:** `gs://<bucket>/<obj>:[<n>:]<texto>[...]\n`, armado en `formatLine` (`run.go:109-127`) y escrito con un solo `Write` bajo mutex (`output.go:16-33`).
- **Lo que define la spec y todavía no está en el código:**
  - `-c`: `gs://b/o:<n>`, con 0 incluido (FR-18a/b, `spec:449-472`).
  - `-l`: corta la lectura en el primer match (FR-19c, `spec:494-503`).
  - `--concurrency`: 4 por defecto, de 1 a 32 (FR-26a-d, `spec:635-685`).
  - Hoy hay un solo worker, en el bucle de `run.go:73-82` (D-07).
- **Exit code** (`run.go:84-91`): 2 si algún objeto falló, aunque haya habido matches (FR-29a); si no, 0 si hubo match; si no, 1. El aviso de error de lectura es `gs://b/o: read error: <err>` (`run.go:80`).
- **Fases:** primero se lista y después se lee (`run.go:52-66`, BR-3). El tope corta al ver el objeto N+1 (`gcs.go:66-87`).

## 4. Tests
- **`internal/cli/cli_test.go`:** `TestParseValid`, `TestParseUsageErrors` (tabla con el mensaje exacto y el tipo `*ErrUsage`, `:52-89`) y `TestParseErrorOrder`.
- **`internal/search/search_test.go`:** `TestScanCRLF`, `TestScanCRNotFollowedByLF`, `TestScanTrailingLoneCR`, `TestScanNoTrailingNewline`, `TestScanEmptyObject`, `TestScanExactlyMaxLineNotTruncated`, `TestScanOneByteOverMaxLineIsTruncated`, `TestScanEndAnchorPastLimit`, `TestScanWordBoundaryAcrossLimit`, `TestScanMatchAfterTruncation`, `TestScanReadErrorMidLine`, `TestScanReadErrorDuringLongLine`, más los `TestCompile*`.
- **`internal/location/location_test.go`:** cubre el parseo de ubicaciones.
- **Suite `e2e/`:** compila `./gcsgrep` y corre contra un **bucket GCS real** `$B`. Exige `GCSGREP_BUCKET`, `GCSGREP_LECTORA` y `GCSGREP_SIN_ACCESO` (`e2e/main_test.go:1-75`). Hay un test por VC:
  - `search_test.go`: VC01, 03, 04, 07a, 08, 09a, 09b, 21, 46.
  - `usage_test.go`: VC05, 06, 14, 22, 23.
  - `access_test.go`: VC15b, 35, 40.
  - `partial_test.go`: VC12, 41, 42 parciales.
  - `zz_vc39_test.go`: foto del bucket antes y después (solo lectura).
- **Helpers e2e:** `requireUsageError` y `checkUsage` (`main_test.go:~191-215`) implementan el "chequeo P": repiten el comando sin ADC, con `STORAGE_EMULATOR_HOST` apuntando a un listener que cuenta conexiones, para probar que un error de uso no toca la red.
- **Fixtures útiles para el flag:** `test-fixtures/edge-cases/long_line_exceeds_1mb.log`, `sniffing/utf8_split_at_512.txt`, `data/acentos.log`, `data/windows_crlf.txt`, `data/sin_salto_final.txt`, `logs/archive/old_logs.log.gz`.

## 5. Specs en `sdd/`
- **Archivos:** `gcsgrep-spec.md` (1246 líneas, marcada "revisada"), `gcsgrep-plan.md` (5 iteraciones), `gcsgrep-base-context.md`, `gcsgrep-cobertura-vc.md`, `CONTEXT.md` (Iteración 1 cerrada), `DECISIONS.md` (D-01..D-17), `iterations/01-busqueda-punta-a-punta.md`.
- **Dónde choca con `--max-bytes`:**
  - `spec:53`, en "Fuera": "Un límite por **bytes** leídos. El guardrail solo cuenta objetos (BR-3)."
  - `plan:463`: "Diferido post-v1, como complemento del tope por objetos". `plan:454-458` dice que si entra, entra por una revisión de spec.
  - `base-context:238` y `:386-387`, y BR-3 *Limitación conocida* (`spec:~962`).
  - `spec:29`, en el "Dentro", lista los flags permitidos.
- **IDs que el flag toca:**
  - Sintaxis completa: `spec:553`.
  - Parser: FR-22, FR-24b (`--max-bytes=5`), FR-25 (flag repetido), FR-27 (`spec:687-698`, lista de errores de uso).
  - Formatos: FR-18a/b (`-c`), FR-19a-c (`-l`).
  - Líneas: FR-9b (última línea sin `\n`), FR-28.
  - Fallos: FR-29b/c (corte a mitad de lectura), FR-32/FR-34 (timeouts).
  - Reglas de negocio: BR-3/BR-4 (patrón de flag numérico, `spec:945-995`), BR-5/6/7 (sniff y gzip), BR-8 (1 MiB), BR-9 (stderr).
  - VC-47 (`spec:~1095`): lista de VCs de falla que tienen que cumplir BR-9.
  - NFR-2 (memoria).
- **Formato:**
  - FR: `#### FR-N · Título`, con **Dado/Cuando/Entonces** en negrita y un VC en blockquote `> **VC-N** — …`, con comando y exit code exactos y "Cada caso cumple el chequeo P".
  - BR: `### BR-N · …`, con *Fundamento:*, *Excepciones:* y *Limitación conocida:*.
  - Hermanos con sufijo a/b/c.
  - Tabla de trazabilidad en `spec:1163+`, con columnas requisito | origen | VC | tipo, y conteo final: "70 requerimientos (58 FR, 9 BR, 3 NFR), 70 VCs".
  - Definición del chequeo P: `spec:~130`.

## 6. Docs de uso
- No hay README de la herramienta.
- Los flags figuran en `spec:29` y en la sintaxis de `spec:553`.
- `test-fixtures/README.md:~55-78` tiene ejemplos de invocación (`-i`, `-l`, `-c`, `-E`, `--max 5`).
- `CONTEXT.md` mapea archivos; no tiene ayuda de uso. El binario tampoco tiene `--help`.

## 7. Riesgos
- **Corte a mitad de línea:** con `io.LimitReader`, el corte llega como `io.EOF`. El scanner lo toma como fin limpio y **busca e imprime la línea parcial** como si fuera FR-9b (`search.go:99-101` y `:165-168`). Eso contradice el criterio de FR-29b, donde la línea cortada no se busca. Hay que decidir: ¿se busca la línea parcial o se descarta? ¿Se marca con `...`?
- **Corte entre `\r` y `\n`:** el `\r` queda en la línea (`search.go:182-188`).
- **Corte a mitad de un carácter UTF-8:** el regexp ve U+FFFD y la salida imprime bytes inválidos, porque se imprime byte a byte. Lo más delicado es el sniff de BR-6 con N < 512: si la muestra termina en una secuencia incompleta, ¿cuenta como "objeto ≤512 analizado entero", que lo vuelve no-texto, o como corte? El sniff tiene que hacerse sobre los bytes reales del objeto, no sobre los truncados, o la spec tiene que fijar la regla.
- **Gzip:** no se descomprime, así que un gzip se saltea antes de que el límite importe. Con *transcoding*, N contaría bytes descomprimidos si el límite se aplica del lado del cliente. Si se implementa con `NewRangeReader`, GCS no respeta rangos sobre objetos transcodificados (a verificar). Con `LimitReader`, el `bufio` de 64 KiB y el TCP pueden traer más de N bytes por la red, así que el ahorro de costo no es exacto y la spec no debería prometer "bytes transferidos".
- **Interacción con otros flags:**
  - `-l` ya corta en el primer match (FR-19c).
  - `-c` contaría solo las líneas dentro del límite, sin señal de que el resultado es parcial salvo que se especifique un aviso por stderr. Hay que definir si es aviso o error y su efecto en el exit code; para no romper el contrato de scripting, lo esperable es que no sea 2.
  - `-n`: los números de línea siguen siendo válidos.
  - BR-8: si N < 1 MiB, nunca se dispara el `...`. Hay que distinguir "línea >1 MiB" de "objeto cortado".
  - `--max` cuenta objetos y `--max-bytes` sería por objeto, no total: conviene nombrarlo y documentarlo sin ambigüedad.
  - Parser: definir el comportamiento con flag repetido (FR-25), con `--max-bytes=N` (FR-24b), con valor 0 o `unlimited`, y si acepta sufijos (`10M`).
  - Timeouts (FR-32/34) y progreso (FR-37/38): el progreso cuenta objetos, no bytes (`base-context:286`).
- **Testabilidad:** sin interfaz de cliente, el corte se testea en unidad con `io.LimitReader` sobre `search.Scan`. Para e2e hace falta el bucket real o el servidor de prueba de la Iteración 2 (por ejemplo, para mostrar que se cierra la conexión antes de recibir el cuerpo completo, como en VC-19c).

## Sesión nueva · cwd=<sandbox>/01-greenfield-gcsgrep · modelo=claude-opus-5-5

→ **Bash** `sed -n '25,32p;50,56p' sdd/gcsgrep-spec.md; echo ----; sed -n '450,466p' sdd/gcsgrep-plan.md; echo ----; sed -n '96,106p' cmd/gcsgrep/run.go; sed -n '95,102p;160,170p' internal/search/search.go`

  ← resultado: ### Dentro ⏎  ⏎ - Un único comando: `gcsgrep [flags] [--] <patrón> <ubicación>`. ⏎ - Búsqueda línea por línea, literal por defecto o RE2 con `-E`. ⏎ - Flags `-E`, `-i`, `-n`, `-c`, `-l`, `--max` y `--concurrency`. ⏎ - Ub …[recortado]

**Agente:** Todavía no escribí la spec: hay un bloqueo y tres decisiones de diseño que te tocan a vos. Esto es lo que salió de explorar el código y la spec.

## El bloqueo: la entrega 01 excluye este flag a propósito

- `sdd/gcsgrep-spec.md:53` lo pone en **Fuera** de forma explícita: "Un límite por **bytes** leídos. El guardrail solo cuenta objetos (BR-3)."
- `sdd/gcsgrep-plan.md:454-463` lo difiere para después de la v1 y aclara que, si entra, "entra por una revisión de spec, no por el plan".

Agregarlo implica escribir en `01-greenfield-gcsgrep/` y además cambiar esa línea de la spec revisada. El CLAUDE.md me pide no tocar `01-*` sin que me lo pidas.

## Línea de base (medida en `22104ba`, antes de cualquier cambio)

`go build ./...` compila. `go test -count=1 ./internal/...` da **23 tests de nivel superior y 43 PASS contando subtests, 0 FAIL**, en `cli`, `location` y `search`. Los paquetes `gcs` y `output` no tienen tests.

## Lo que tendría que hacer, según el código actual

- **Parser:** en `internal/cli/cli.go:58` se agregaría el caso `--max-bytes N`, con el mismo patrón que `parseMax` (`cli.go:106-131`):
  - acepta un entero ≥ 1 o `unlimited`;
  - por defecto `unlimited`, así el comportamiento actual no cambia;
  - no acepta sufijos como `10M`;
  - un error de uso sale con código 2 sin tocar la red.
- **Corte:** se envuelve el reader en `readObject` (`cmd/gcsgrep/run.go:96-105`). El límite es **por objeto**, no un total.
- **El riesgo principal:** un `io.LimitReader` común devuelve `io.EOF`. El scanner lo toma como un fin limpio (`search.go:165-168`) y **busca e imprime la línea cortada** como si fuera la última del objeto. Para distinguir un corte de un fin real, hace falta un reader que lea N+1 bytes y devuelva un error propio. Un objeto de exactamente N bytes no cuenta como cortado.
- **Lo que la spec no puede prometer:** que se descarguen solo N bytes. El buffer de 64 KiB y TCP traen más por la red. Lo verificable es que el scanner consuma como mucho N+1 bytes por objeto.
- **Desfase entre spec y código:** el código está en la Iteración 1. `-c`, `-l`, `--concurrency` y la detección de binarios (BR-5/6/7) están en la spec pero todavía no existen. Hay dos cruces que hay que definir:
  - con `-c`, se cuentan solo las líneas dentro del límite;
  - con la detección de BR-6 y N < 512, ¿se analiza la muestra real del objeto o la cortada?

## Lo que necesito que decidas

1. **¿Dónde va?** Mi recomendación: una spec aparte, `01-greenfield-gcsgrep/sdd/max-bytes-spec.md`, más un cambio de una línea en `gcsgrep-spec.md:53` que apunte a ella. La alternativa es reescribir la spec principal y su tabla de trazabilidad (70 requerimientos), que es más invasivo.
2. **¿Qué pasa con la línea cortada?** Recomiendo **no buscarla**, igual que el criterio de FR-29b para cortes a mitad de lectura. La alternativa es buscarla y marcarla con `...`.
3. **¿Cómo se avisa el corte?** Recomiendo un aviso por stderr, `gs://b/o: truncated at N bytes`, **sin cambiar el código de salida**. Si diera 2, rompería el uso en scripts. Las alternativas son no avisar o salir con 2.

Si me das el OK para tocar `01-*` y aceptás mis recomendaciones (o me decís cuáles cambiar), escribo la spec con un VC por requisito. Después la paso por `vc-coverage.sh` y por `review-spec` hasta que tenga `LISTA`. Recién ahí se escribe código.


---
Fin: error_max_turns · turnos=4 · costo USD=0.9506437999999999 · duración=25s

---
Fin: success · turnos=2 · costo USD=0.9506437999999999 · duración=35s
