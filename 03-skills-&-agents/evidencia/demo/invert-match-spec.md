# `invert-match` (`-v`) — spec

> Estado: revisada (2026-10-04, `review-spec`: LISTA en la 5.ª pasada) · Base: `01-greenfield-gcsgrep/`@`22104ba` · Notas:
> [notas-exploracion.md](./notas-exploracion.md) (se citan como "hallazgo N")
>
> Toda referencia `archivo:línea` es relativa a `01-greenfield-gcsgrep/` en ese commit.
> Los IDs `FR-n`, `BR-n` y `D-n` que se citan con el prefijo "base" son los de
> `sdd/gcsgrep-spec.md` y `sdd/DECISIONS.md`.
>
> Regla estructural: cada FR, BR, INV y NFR tiene un VC con su mismo ID (VC-3 ↔ FR-3,
> VC-BR-1 ↔ BR-1, VC-INV-1 ↔ INV-1, VC-NFR-1 ↔ NFR-1).

## Propósito

Agregar a `gcsgrep` el flag `-v`, que invierte la selección como `grep -v`: se imprimen
las líneas que **no** matchean el patrón, con el mismo formato, numeración y exit codes
que una búsqueda normal, sin cambiar nada de lo que hace la herramienta sin `-v`.

## Alcance

### Dentro

| Archivo | Cambio | Referencia |
|---|---|---|
| `internal/cli/cli.go` | Campo `Invert bool // -v` en `Config`; `case "-v"` en el `switch` de flags; `Invert` en el `Config` devuelto | `cli.go:21-28`, `cli.go:58-78`, `cli.go:92-99` |
| `internal/search/search.go` | Campo exportado `Invert bool` en `Scanner` (el valor cero conserva el comportamiento actual); la condición de emisión pasa de `matched` a `matched != s.Invert`, en el mismo lugar, después del chequeo de error de lectura; la doc de `LineFunc` y `Scan` dice "línea seleccionada" en lugar de "línea que matchea" | `search.go:56-59`, `search.go:65-68`, `search.go:82-85`, `search.go:122-127` |
| `cmd/gcsgrep/run.go` | `scanner.Invert = cfg.Invert` después de crear el `Scanner`. Nada más: el exit code ya depende de "se emitió al menos una línea" (hallazgo 4) | `run.go:69` |
| `internal/cli/cli_test.go` | Casos nuevos de `-v` (VC-12, VC-13, VC-15). Los dos usos de `-v` como flag desconocido pasan a `-w`, sin cambiar el nombre del caso: el caso `"unknown long flag"` y `TestParseErrorOrder` (argumento y mensaje esperado) | `cli_test.go:61`, `cli_test.go:93-95` |
| `internal/search/search_test.go` | Tests nuevos de inversión (VC-10, VC-BR-3, VC-NFR-1, VC-NFR-2) | archivo existente, tests al final |
| `e2e/usage_test.go` | `TestVC22`: el segundo caso pasa de `-v` a `-w` | `usage_test.go:40` |
| `e2e/invert_test.go` (nuevo) | Automatiza, con los helpers de `e2e/main_test.go`, los VCs de esta spec que corren un solo binario contra el bucket: VC-1 a VC-9, VC-11, VC-14, VC-16, VC-17, VC-BR-1, VC-BR-2 y VC-INV-5 (que compara dos invocaciones del mismo binario). Nombre: `TestInvertVC` + el ID del VC sin guiones (`TestInvertVC1`, `TestInvertVCBR1`, `TestInvertVCINV5`), para no chocar con los `TestVC<n>` de la base (base D-17). VC-INV-2 y VC-INV-3 no van acá: comparan contra el binario o la suite de `22104ba` y se corren como dicen sus VCs | — |
| `sdd/gcsgrep-spec.md` | Sacar `-v` de la lista de Fuera; sumarlo a la lista de flags y a la sintaxis; en VC-22 reemplazar `-v` por `-w` | `gcsgrep-spec.md:30`, `:46`, `:553`, `:573-576` |
| `sdd/gcsgrep-cobertura-vc.md` | En la fila de VC-22, `-v` pasa a `-w` | `gcsgrep-cobertura-vc.md:53` |
| `sdd/gcsgrep-base-context.md` | En "¿Qué queda fuera?", sacar `-v` de los diferidos y enlazar esta spec | `gcsgrep-base-context.md:59` |

### Fuera

- **Saltear binarios (base BR-5/BR-6/BR-7).** No está implementado en `22104ba`
  (hallazgo 5): hoy un binario se busca como texto con o sin flags, y con `-v` pasa lo
  mismo. Cuando entre el sniffing, la clasificación del objeto ocurre antes de la
  selección de líneas y un objeto salteado no aporta líneas con ni sin `-v` (BR-1 lo
  sigue garantizando). Esta spec no adelanta esa iteración.
- **`-c`, `-l` y `--concurrency`.** No existen en la base; su interacción con `-v` se
  especifica cuando se implementen (base FR-18a/b, FR-19a/b/c, FR-20a/b/c).
- **Pista para flags combinados y rechazo de repetidos (base FR-24a, FR-25).** No existen
  en la base. `-v` se comporta como `-E`, `-i` y `-n` hoy (FR-14, FR-15) y va a cambiar
  junto con ellos.
- **Exit codes y precedencia** (`run.go:84-91`): no se tocan (INV-6). Un error de
  lectura sigue dando `2` aunque se hayan impreso líneas (base FR-29a). **Observar** ese
  exit `2` en una ejecución con `-v` requiere inyectar un error de lectura contra GCS, y
  eso necesita el servidor de prueba de la Iteración 2 de la base (base D-15), que no
  existe en `22104ba`. Construirlo no es parte de este cambio. Cuando exista, base VC-29a
  se corre también con `-v`.
- **Formato de salida** (`run.go:109-127`, `internal/output/output.go`): no se toca.
- **Listado, `--max`, ubicaciones y credenciales** (`internal/gcs`, `internal/location`,
  `run.go:37-66`): `-v` actúa después del listado, sobre las líneas.
- **`search.Compile`** (`search.go:38-54`): el patrón se compila igual con y sin `-v`.
- **Forma larga `--invert-match`**: el pedido es `-v`; agregarla suma superficie sin uso.
- **Las demás entregas** (`02-*`).
- **`sdd/gcsgrep-plan.md`, `sdd/DECISIONS.md`, `sdd/CONTEXT.md` y
  `sdd/iterations/`.** Son el registro histórico de cómo se planificó y construyó la v1:
  "`-v` diferido post-v1" (`gcsgrep-plan.md:466`) sigue siendo cierto sobre ese plan. Lo
  que describe el comportamiento vigente (la spec base, su cobertura de VCs y su base
  context) sí se actualiza, en Dentro.

## Línea de base

Medida en `22104ba`, antes del cambio, desde `01-greenfield-gcsgrep/`:

- `go build ./...` → exit `0`.
- `go test -v ./internal/... | grep -E '^\s*--- (PASS|FAIL)'` → **43 PASS, 0 FAIL**
  (paquetes `cli`, `location`, `search`).
- `go test ./e2e/` → **no medida**: no hay `GCSGREP_BUCKET`, `GCSGREP_LECTORA` ni
  `GCSGREP_SIN_ACCESO` en la máquina donde se escribió la spec. Por eso VC-INV-1,
  VC-INV-2 y VC-INV-3 son diferenciales: comparan `22104ba` contra el commit con el
  cambio, en la misma máquina y contra el mismo bucket.

## Entorno de verificación

- `./gcsgrep` es el binario construido con `go build -o gcsgrep ./cmd/gcsgrep` desde
  `01-greenfield-gcsgrep/`; los comandos corren desde ese directorio.
- `$B` es el bucket de fixtures de la base, cargado como indica
  `test-fixtures/README.md`, y las ADC son las de la identidad lectora.
- `./gcsgrep-base` es el binario de `22104ba`, dejado junto a `./gcsgrep` con estos dos
  comandos desde `01-greenfield-gcsgrep/`: `git worktree add /tmp/gcsgrep-base 22104ba` y
  `go -C /tmp/gcsgrep-base/01-greenfield-gcsgrep build -o "$PWD/gcsgrep-base" ./cmd/gcsgrep`.
- **Salvo que el VC diga otra cosa, stderr tiene que estar vacío.**
- Los archivos auxiliares que crean los VCs (`out`, `m`, `v`, `bench.txt`, `base.txt`,
  `nuevo.txt`, `permitidos.txt`, `gcsgrep-base`) no se commitean; si se commitearan,
  VC-INV-4 fallaría.
- Los VCs listados en la fila de `e2e/invert_test.go` se automatizan ahí, uno por test:
  `go test -v ./e2e/ -run '^TestInvertVC'` tiene que dar un `--- PASS` por cada uno y
  ningún `--- FAIL`. El VC escrito en esta spec es la referencia; el test lo implementa
  sin agregar ni quitar nada.
- `$INICIO` es el commit desde el que arranca la implementación (el que tiene esta spec
  con `Estado: revisada`); `$CAMBIO` es el último commit de la implementación. Los
  comandos `git` con pathspec corren desde el directorio que dice cada VC.
- "Chequeo P" es el de la base (`sdd/gcsgrep-spec.md:130-135`): el comando se repite sin
  ADC y con `STORAGE_EMULATOR_HOST` apuntando a un listener, que no registra ninguna
  conexión.
- "stdout exacto" sobre un prefijo compara las líneas sin importar el orden entre
  objetos distintos (base FR-28).

## Requerimientos funcionales

### Selección

#### FR-1 · Imprimir las líneas que no matchean

**Dado** un objeto de texto con líneas que matchean el patrón y líneas que no,
**Cuando** la persona ejecuta `gcsgrep -v <patrón> gs://<bucket>/<objeto>`,
**Entonces** el sistema imprime, en el orden del objeto, solo las líneas que no matchean,
cada una como `gs://<bucket>/<objeto>:<texto>`, y sale con código `0`.

> **VC-1** — `./gcsgrep -v ' 200 ' gs://$B/logs/server.log` sale con código `0`, stderr
> vacío, y stdout es exactamente
> `gs://$B/logs/server.log:192.168.1.52 - - [22/Sep/2026:10:05:44 -0300] "GET /slow-query HTTP/1.1" 504 312`
> seguido de `\n`.

#### FR-2 · Salir con 1 si todas las líneas matchean

**Dado** un objeto de texto en el que todas las líneas matchean el patrón,
**Cuando** la persona ejecuta la búsqueda con `-v`,
**Entonces** el sistema no imprime nada por stdout y sale con código `1`.

> **VC-2** — `./gcsgrep -v HTTP/1.1 gs://$B/logs/server.log` sale con código `1`, stdout
> vacío y stderr vacío.

#### FR-3 · Numerar con la línea original al combinar con `-n`

**Dado** un objeto de texto,
**Cuando** la persona ejecuta la búsqueda con `-v -n`,
**Entonces** cada línea impresa lleva su número de línea dentro del objeto (contando
también las líneas que matchean y no se imprimen), como
`gs://<bucket>/<objeto>:<n>:<texto>`.

> **VC-3** — `./gcsgrep -v -n checkpoint gs://$B/logs/db/postgres.log` sale con código
> `0` y stdout es exactamente estas tres líneas:
> ```
> gs://$B/logs/db/postgres.log:3:2026-09-22 08:15:33 UTC [1450]: [1-1] user=analytics,db=warehouse ERROR: canceling statement due to statement timeout
> gs://$B/logs/db/postgres.log:4:2026-09-22 08:30:12 UTC [1602]: [1-1] user=app,db=production ERROR: deadlock detected
> gs://$B/logs/db/postgres.log:5:2026-09-22 08:45:00 UTC [1201]: [3-1] user=app,db=production LOG: automatic vacuum of table accounts
> ```

#### FR-4 · Excluir sin distinguir mayúsculas al combinar con `-i`

**Dado** un objeto con líneas que contienen el patrón con otra capitalización,
**Cuando** la persona ejecuta la búsqueda con `-v -i`,
**Entonces** esas líneas no se imprimen (se consideran matches, como sin `-v`).

> **VC-4** — `./gcsgrep -v -i error gs://$B/logs/db/postgres.log` sale con código `0` y
> stdout es exactamente las líneas 1, 2 y 5 del objeto, cada una como
> `gs://$B/logs/db/postgres.log:<texto>`, en ese orden. (Sin `-i`, el mismo comando
> imprimiría las 5 líneas, porque ninguna contiene `error` en minúscula.)

#### FR-5 · Excluir por regex al combinar con `-E`

**Dado** un patrón RE2 válido,
**Cuando** la persona ejecuta la búsqueda con `-v -E`,
**Entonces** se imprimen las líneas que la regex no matchea.

> **VC-5** — `./gcsgrep -v -E '\[[12]-1\]' gs://$B/logs/db/postgres.log` sale con
> código `0` y stdout es exactamente
> `gs://$B/logs/db/postgres.log:2026-09-22 08:45:00 UTC [1201]: [3-1] user=app,db=production LOG: automatic vacuum of table accounts`
> seguido de `\n`.

#### FR-6 · No imprimir nada para un objeto vacío

**Dado** un objeto de 0 bytes,
**Cuando** la persona ejecuta la búsqueda con `-v` sobre ese objeto,
**Entonces** el sistema no imprime nada (el objeto no tiene líneas) y sale con código `1`.

> **VC-6** — `./gcsgrep -v x gs://$B/edge-cases/empty.txt` sale con código `1`, stdout
> vacío y stderr vacío.

#### FR-7 · Imprimir sin el `\r` las líneas CRLF que no matchean

**Dado** un objeto con saltos de línea `\r\n`,
**Cuando** la persona ejecuta la búsqueda con `-v`,
**Entonces** las líneas impresas no llevan el `\r` final (base FR-9a).

> **VC-7** — `./gcsgrep -v timeout gs://$B/data/windows_crlf.txt > out` sale con código
> `0`, y
> `cmp out <(printf 'gs://%s/data/windows_crlf.txt:Linea 1 con retorno CRLF\ngs://%s/data/windows_crlf.txt:Linea 3 con CRLF sin match\n' "$B" "$B")`
> sale con código `0` (sin diferencias; como la referencia no tiene `\r`, `out` tampoco).

#### FR-8 · Imprimir la última línea sin salto final si no matchea

**Dado** un objeto cuya última línea no termina en `\n`,
**Cuando** la persona ejecuta la búsqueda con `-v` y esa línea no matchea,
**Entonces** se imprime terminada en `\n` (base FR-9b).

> **VC-8** — `./gcsgrep -v Primera gs://$B/data/sin_salto_final.txt` sale con código
> `0` y stdout es exactamente `gs://$B/data/sin_salto_final.txt:Ultima linea sin salto final`
> seguido de `\n`.

#### FR-9 · Truncar una línea de más de 1 MiB que no matchea

**Dado** un objeto con una línea de más de 1 MiB que no matchea el patrón,
**Cuando** la persona ejecuta la búsqueda con `-v`,
**Entonces** esa línea se imprime con sus primeros 1 048 576 bytes seguidos de `...`
(base BR-8).

> **VC-9** — `./gcsgrep -v 'Linea normal' gs://$B/edge-cases/long_line_exceeds_1mb.log > out`
> sale con código `0`, y
> `cmp out <(printf 'gs://%s/edge-cases/long_line_exceeds_1mb.log:' "$B"; head -c 1048576 test-fixtures/edge-cases/long_line_exceeds_1mb.log; printf '...\n')`
> sale con código `0` (sin diferencias).

#### FR-10 · Imprimir las líneas vacías que no matchean

**Dado** un objeto con una línea vacía,
**Cuando** se busca con `-v` un patrón que la línea vacía no matchea,
**Entonces** la línea vacía se selecciona con su número y texto vacío, y no aparece una
línea vacía extra por el `\n` final del objeto.

> **VC-10** — `go test ./internal/search -run '^TestScanInvertEmptyLine$' -v` sale con
> código `0` y `--- PASS: TestScanInvertEmptyLine`. El test escanea `"a\n\nb\n"` con
> `mustCompile(t, "a", false, false)` y un `Scanner` con `Invert = true`, y espera
> exactamente `[{2, "", false}, {3, "b", false}]` con `assertLines`. (No hay fixture
> con líneas vacías en el bucket; el formato `gs://<bucket>/<objeto>:` sale de
> `formatLine`, que no cambia.)

#### FR-11 · Invertir en cada objeto de un prefijo

**Dado** un prefijo con varios objetos de texto, alguno sin ninguna línea que matchee,
**Cuando** la persona ejecuta la búsqueda con `-v` sobre el prefijo,
**Entonces** se imprimen las líneas que no matchean de cada objeto, incluidas todas las
del objeto que no tiene matches, y sale con código `0`.

> **VC-11** — `./gcsgrep -v ERROR gs://$B/logs/db/ > out` sale con código `0`, y
> `diff <(sort out) <( { sed -n '1p;2p;5p' test-fixtures/logs/db/postgres.log | sed "s#^#gs://$B/logs/db/postgres.log:#"; sed "s#^#gs://$B/logs/db/redis.log:#" test-fixtures/logs/db/redis.log; } | sort)`
> sale con código `0` (9 líneas, sin diferencias).

### Invocación

#### FR-12 · Reconocer `-v` como flag

**Dado** el argumento `-v` antes de `--`,
**Cuando** se parsea la invocación,
**Entonces** el resultado tiene `Invert = true` y el resto de los campos como sin `-v`.

> **VC-12** — `go test ./internal/cli -run '^TestParseValid$/^(invert|all_flags_with_invert)$' -v`
> sale con código `0` con `--- PASS: TestParseValid/invert` y
> `--- PASS: TestParseValid/all_flags_with_invert`. Son dos casos nuevos de la tabla de
> `TestParseValid` (el caso existente "all flags" no se toca):
> - `"invert"`: `{"-v", "pat", "gs://b/"}` →
>   `Config{Invert: true, Max: defaultMax, Pattern: "pat", Location: "gs://b/"}`.
> - `"all flags with invert"`: `{"-E", "-i", "-n", "-v", "--max", "5", "pat", "gs://b/"}` →
>   `Config{Extended: true, IgnoreCase: true, LineNumber: true, Invert: true, Max: 5, Pattern: "pat", Location: "gs://b/"}`.

#### FR-13 · Tratar `-v` después de `--` como patrón

**Dado** el argumento `-v` después de `--`,
**Cuando** se parsea la invocación,
**Entonces** `-v` es el patrón literal y la inversión queda desactivada.

> **VC-13** — `go test ./internal/cli -run '^TestParseValid$/^invert_after_double_dash_is_the_pattern$' -v`
> sale con código `0` con
> `--- PASS: TestParseValid/invert_after_double_dash_is_the_pattern`. El caso nuevo
> `"invert after double dash is the pattern"` parsea `{"--", "-v", "gs://b/"}` y espera
> `Config{Max: defaultMax, Pattern: "-v", Location: "gs://b/"}` (`Invert` en `false`).

#### FR-14 · Rechazar `-v` combinado con otro flag corto

**Dado** un argumento anterior a `--` que junta `v` con otro flag corto en un solo
argumento, en cualquier orden (`-vn`, `-nv`, `-Ev`, `-iv`, …),
**Cuando** la persona ejecuta el comando,
**Entonces** el sistema sale con código `2`, escribe
`gcsgrep: unknown flag: "<argumento tal como vino>"` por stderr y no opera contra GCS
(base FR-22: solo se reconoce `-v` exacto, igual que `-in` hoy).

> **VC-14** — Para cada `<a>` en `-vn`, `-nv`, `-Ev` e `-iv`,
> `./gcsgrep <a> timeout gs://$B/logs/` sale con código `2`, stdout vacío y stderr
> exactamente `gcsgrep: unknown flag: "<a>"` seguido de `\n` (por ejemplo
> `gcsgrep: unknown flag: "-nv"`). Cada caso cumple el chequeo P.

#### FR-15 · Aceptar `-v` repetido

**Dado** `-v` más de una vez antes de `--`,
**Cuando** se parsea la invocación,
**Entonces** el resultado es el mismo que con un solo `-v` (base D-08: los repetidos se
aceptan, como `-E`, `-i` y `-n`).

> **VC-15** — `go test ./internal/cli -run '^TestParseValid$/^invert_repeated$' -v` sale
> con código `0` con `--- PASS: TestParseValid/invert_repeated`. El caso nuevo
> `"invert repeated"` parsea `{"-v", "-v", "pat", "gs://b/"}` y espera
> `Config{Invert: true, Max: defaultMax, Pattern: "pat", Location: "gs://b/"}`.

#### FR-16 · Rechazar el patrón vacío con `-v`

**Dado** `-v` y un patrón vacío,
**Cuando** la persona ejecuta el comando,
**Entonces** el sistema sale con código `2` con `gcsgrep: empty pattern` (base FR-6) y no
opera contra GCS: `-v ''` no es "imprimir todo".

> **VC-16** — `./gcsgrep -v '' gs://$B/logs/` sale con código `2`, stdout vacío y stderr
> exactamente `gcsgrep: empty pattern` seguido de `\n`. Cumple el chequeo P.

#### FR-17 · Rechazar una regex inválida con `-v`

**Dado** `-v -E` con un patrón que no es RE2 válido,
**Cuando** la persona ejecuta el comando,
**Entonces** el sistema falla igual que sin `-v` (base FR-5): mismo stderr, código `2`, sin
operar contra GCS.

> **VC-17** — `./gcsgrep -v -E '(' gs://$B/logs/` sale con código `2`, stdout vacío, y su
> stderr es byte a byte igual al de `./gcsgrep -E '(' gs://$B/logs/` (que empieza con
> `gcsgrep: invalid pattern: `). Cumple el chequeo P.

## Reglas de negocio

### BR-1 · `-v` parte las líneas: cada línea sale en exactamente una de las dos búsquedas

Para el mismo patrón, flags y ubicación, la búsqueda con `-v` y la búsqueda sin `-v`
recorren los mismos objetos y las mismas líneas; cada línea de cada objeto buscado aparece
en exactamente una de las dos salidas. `-v` no cambia qué objetos se buscan.

> **VC-BR-1** — Con `./gcsgrep -n ERROR gs://$B/logs/db/ > m` (sale `0`) y
> `./gcsgrep -v -n ERROR gs://$B/logs/db/ > v` (sale `0`):
> `sort m v | uniq -d` imprime nada, y
> `cat m v | sed -E "s#^gs://$B/##; s/^([^:]+:[0-9]+):.*/\1/" | sort` es exactamente
> `logs/db/postgres.log:1` … `logs/db/postgres.log:5` y `logs/db/redis.log:1` …
> `logs/db/redis.log:6` (11 líneas, cada una una vez).

### BR-2 · El match se decide sobre la línea entera, no sobre lo que se imprime

Una línea de más de 1 MiB cuyo único match está después del primer MiB matchea (base
BR-8), así que con `-v` **no** se imprime, aunque el texto truncado no muestre el patrón.

> **VC-BR-2** — `./gcsgrep -v -n late_after_1mb gs://$B/edge-cases/long_line_exceeds_1mb.log`
> sale con código `0` y stdout es exactamente:
> ```
> gs://$B/edge-cases/long_line_exceeds_1mb.log:2:2026-09-22 12:00:01 INFO Linea normal siguiente sin timeout
> gs://$B/edge-cases/long_line_exceeds_1mb.log:3:2026-09-22 12:00:02 INFO Linea normal final con timeout detectado
> ```

### BR-3 · Una línea cortada por un error de lectura no se selecciona

Si la lectura de un objeto falla a mitad de una línea, esa línea no se evalúa ni se
imprime con `-v` (base D-12); las líneas completas anteriores sí, y el error se devuelve.

> **VC-BR-3** — `go test ./internal/search -run '^TestScanInvertReadError(MidLine|DuringLongLine)$' -v`
> sale con código `0` con `--- PASS` para los dos tests, uno por cada rama de `Scan`
> (`search.go:106-108` y `search.go:109-121`). Ambos usan `Invert = true` y
> `mustCompile(t, "zzz", false, false)`, y esperan que `Scan` devuelva un error con
> `errors.Is(err, boom)`:
> - `TestScanInvertReadErrorMidLine` escanea
>   `io.MultiReader(strings.NewReader("first ok\n"), &errAfterReader{data: []byte("partial"), err: boom})`
>   y espera que `fn` haya recibido exactamente `[{1, "first ok", false}]`.
> - `TestScanInvertReadErrorDuringLongLine` escanea
>   `io.MultiReader(bytes.NewReader(bytes.Repeat([]byte("a"), MaxLine+10)), &errAfterReader{data: []byte("more"), err: boom})`
>   y espera que `fn` no se haya llamado nunca.

## Invariantes

### INV-1 · La suite unitaria da lo mismo que en la línea de base

Los 43 tests y subtests que pasan en `22104ba` siguen pasando, con el mismo nombre, y
ninguno falla. Los casos de `cli_test.go:61` y `:93` cambian `-v` por `-w` sin cambiar de
nombre.

> **VC-INV-1** — En cada commit (`/tmp/gcsgrep-base/01-greenfield-gcsgrep/` en `22104ba` y `01-greenfield-gcsgrep/` en el commit con el
> cambio) se corre
> `go test -v ./internal/... 2>&1 | grep -E '^\s*--- (PASS|FAIL)' | sed -E 's/ \([0-9.]+s\)//' | sort`
> a `base.txt` y `nuevo.txt`. `comm -23 base.txt nuevo.txt` no imprime nada, y
> `grep -c -- '--- FAIL' nuevo.txt` imprime `0`.

### INV-2 · Sin `-v`, la herramienta se comporta byte a byte igual

Cualquier invocación sin `-v` produce el mismo stdout, stderr y exit code que en `22104ba`.

> **VC-INV-2** — Para cada una de estas invocaciones, `./gcsgrep-base` y `./gcsgrep`
> producen stdout y stderr idénticos (`cmp` sale `0`) y el mismo exit code:
> `' 200 ' gs://$B/logs/server.log` · `HTTP/1.1 gs://$B/logs/server.log` ·
> `-n checkpoint gs://$B/logs/db/postgres.log` · `-i error gs://$B/logs/db/postgres.log` ·
> `-E '\[[12]-1\]' gs://$B/logs/db/postgres.log` · `x gs://$B/edge-cases/empty.txt` ·
> `timeout gs://$B/data/windows_crlf.txt` · `Primera gs://$B/data/sin_salto_final.txt` ·
> `'Linea normal' gs://$B/edge-cases/long_line_exceeds_1mb.log` ·
> `-n late_after_1mb gs://$B/edge-cases/long_line_exceeds_1mb.log` ·
> `ERROR gs://$B/logs/db/` · `-n timeout gs://$B/` · `-- -v gs://$B/logs/` ·
> `'' gs://$B/logs/` · `-E '(' gs://$B/logs/` · `-in timeout gs://$B/logs/` ·
> `--max 5 timeout gs://$B/edge-cases/`. (Para `gs://$B/` y `logs/db/`, stdout se compara
> después de `sort`, por base FR-28.)

### INV-3 · La suite e2e da lo mismo que en la línea de base

Todo `TestVC*` de `e2e/` que pasa en `22104ba` pasa después del cambio (`TestVC22` con
`-w` en lugar de `-v`).

> **VC-INV-3** — Con las mismas `GCSGREP_BUCKET`, `GCSGREP_LECTORA` y
> `GCSGREP_SIN_ACCESO`, `go test -v ./e2e/ -run '^TestVC' 2>&1 | grep -E '^\s*--- PASS' | sed -E 's/ \([0-9.]+s\)//' | sort`
> en `/tmp/gcsgrep-base/01-greenfield-gcsgrep/` y en `01-greenfield-gcsgrep/` con el cambio da `base-e2e.txt` y
> `nuevo-e2e.txt`; `comm -23 base-e2e.txt nuevo-e2e.txt` no imprime nada.

### INV-4 · Solo cambian los archivos de Dentro

El diff total del cambio (`$INICIO`..`$CAMBIO`, ver Entorno) toca únicamente los
archivos de Dentro y esta spec con sus notas. Ningún otro archivo del repo, incluidas
las otras entregas. Y `01-greenfield-gcsgrep/` llega a `$INICIO` igual que en `22104ba`,
así que las comparaciones contra `22104ba` son comparaciones contra el punto de partida.

> **VC-INV-4** — Tres comandos, desde la raíz del repo:
> 1. `git diff --name-only 22104ba "$INICIO" -- 01-greenfield-gcsgrep/` no imprime nada.
> 2. `git diff --name-only "$INICIO" "$CAMBIO" | grep -c '^01-greenfield-gcsgrep/internal/cli/cli.go$'`
>    imprime `1` (el diff no está vacío).
> 3. `git diff --name-only "$INICIO" "$CAMBIO" | grep -vxF -f permitidos.txt` no imprime
>    nada y sale con código `1`.
>
> `permitidos.txt` tiene estas 12 rutas, una por línea:
> `01-greenfield-gcsgrep/internal/cli/cli.go`,
> `01-greenfield-gcsgrep/internal/cli/cli_test.go`,
> `01-greenfield-gcsgrep/internal/search/search.go`,
> `01-greenfield-gcsgrep/internal/search/search_test.go`,
> `01-greenfield-gcsgrep/cmd/gcsgrep/run.go`,
> `01-greenfield-gcsgrep/e2e/usage_test.go`,
> `01-greenfield-gcsgrep/e2e/invert_test.go`,
> `01-greenfield-gcsgrep/sdd/gcsgrep-spec.md`,
> `01-greenfield-gcsgrep/sdd/gcsgrep-cobertura-vc.md`,
> `01-greenfield-gcsgrep/sdd/gcsgrep-base-context.md`,
> `03-skills-&-agents/evidencia/demo/invert-match-spec.md` y
> `03-skills-&-agents/evidencia/demo/notas-exploracion.md`.

### INV-5 · Con `-v`, las fallas de uso, ubicación, listado y acceso son las mismas que sin `-v`

`-v` es un flag sin valor: no consume el argumento siguiente ni cambia el orden de errores
de D-08, y actúa sobre las líneas, después de listar y abrir. Cualquier falla anterior
produce con `-v` el mismo stdout, stderr y exit code que sin `-v`: errores de uso del
parser (cantidad de posicionales, `--max` inválido o sin valor, otro flag desconocido),
ubicación inválida, bucket, objeto o prefijo inexistente, tope de `--max`, credenciales
ausentes o sin permiso.

> **VC-INV-5** — Con el binario nuevo, para cada caso `./gcsgrep -v <args>` y
> `./gcsgrep <args>` producen stdout y stderr idénticos (`cmp` sale `0`) y el mismo exit
> code. Los cinco primeros son errores de uso y, además, cumplen el chequeo P con `-v`:
> - falta el patrón: `gs://$B/logs/` (`expected 2 arguments … got 1`);
> - posicional de más: `timeout gs://$B/logs/ extra`;
> - `--max` inválido: `--max 0 timeout gs://$B/logs/`;
> - `--max` sin valor: `timeout gs://$B/logs/ --max`;
> - otro flag desconocido, con errores posteriores (D-08): `-w --max 0 '' gs://$B/ extra`
>   (sale `gcsgrep: unknown flag: "-w"` con y sin `-v`);
> - bucket inexistente: `timeout gs://$B-no-existe/`;
> - objeto inexistente: `timeout gs://$B/logs/no-existe.log`;
> - prefijo sin objetos: `timeout gs://$B/no-existe/`;
> - tope de listado: `--max 5 timeout gs://$B/edge-cases/`;
> - ubicación inválida: `timeout s3://$B/`;
> - sin permiso: `timeout gs://$B/logs/` con `GOOGLE_APPLICATION_CREDENTIALS=$GCSGREP_SIN_ACCESO`;
> - sin ADC: `timeout gs://$B/logs/` en el entorno sin ADC de la base
>   (`sdd/gcsgrep-spec.md:128-129`).

### INV-6 · La precedencia de exit codes no cambia

`run.go` solo gana la línea que pasa `cfg.Invert` al `Scanner`. La asignación de
`failed` (`run.go:78-80`) y el `switch` `failed` → `2`, `matched` → `0`, si no → `1`
(`run.go:84-91`) quedan como en `22104ba`, así que un error de lectura sigue dando `2`
aunque `-v` haya impreso líneas (base FR-29a).

> **VC-INV-6** — Desde `01-greenfield-gcsgrep/`,
> `git diff -U0 22104ba "$CAMBIO" -- cmd/gcsgrep/run.go | grep -E '^[+-]' | grep -vE '^(\+\+\+|---) '`
> imprime exactamente una línea: `+	scanner.Invert = cfg.Invert` (un tab de sangría).

## Requerimientos no funcionales

### NFR-1 · Memoria acotada con `-v` en líneas enormes

Con `-v`, una línea larga que no matchea ahora se emite. Escanear una línea de 64 MiB que
no matchea, con `Invert = true`, reserva menos de 4 MiB en total (base D-10: la línea no
se carga entera).

> **VC-NFR-1** — `go test ./internal/search -run '^TestScanInvertMemoryBounded$' -v`
> sale con código `0` y `--- PASS`. El test crea el `Scanner` y compila `zzz` antes de
> medir; escanea un `io.Reader` que genera 64 MiB de `a` seguidos de `\n` sin tenerlos en
> memoria; compara `runtime.MemStats.TotalAlloc` antes y después de `Scan` y exige una
> diferencia `< 4 << 20` bytes, y que `fn` se haya llamado una vez con `truncated = true`.

### NFR-2 · `-v` no agrega una segunda pasada

La selección invertida es una comparación sobre el resultado del match, no otra
búsqueda. Sobre un input donde la mitad de las líneas matchea (ambos modos emiten la
misma cantidad de líneas), la mediana del tiempo de `Scan` con `Invert = true` es como
mucho 1,15 veces la de `Scan` sin invertir.

Input de los dos benchmarks (`BenchmarkScanPlain` y `BenchmarkScanInvert`, en
`search_test.go`), armado una vez fuera del timer: 320 000 líneas alternadas, las de
índice par `strings.Repeat("a", 94) + "MATCH"` y las impares `strings.Repeat("a", 99)`,
cada una seguida de `\n` (100 bytes por línea, ~30,5 MiB). Patrón literal
`Compile("MATCH", false, false)`, llamado directo con `b.Fatal` ante error (`mustCompile`
recibe `*testing.T` y no se toca); `fn` devuelve `nil` sin copiar nada;
un `Scanner` por benchmark, reusado en cada iteración con `bytes.NewReader(input)`. Los
dos benchmarks solo difieren en `Invert`.

> **VC-NFR-2** — Desde `01-greenfield-gcsgrep/`:
> ```sh
> go test ./internal/search -run '^$' -bench '^BenchmarkScan(Plain|Invert)$' -benchtime 20x -count 5 > bench.txt
> med() { grep -E "^BenchmarkScan$1(-[0-9]+)?[[:space:]]" bench.txt | awk '{print $3}' | sort -n | sed -n 3p; }
> awk -v p="$(med Plain)" -v i="$(med Invert)" 'BEGIN { r = i / p; printf "ratio=%.3f\n", r; exit !(r <= 1.15) }'
> ```
> `go test` sale con código `0`, `bench.txt` tiene 5 líneas de cada benchmark, y el `awk`
> final imprime `ratio=<x>` con `<x> <= 1.150` y sale con código `0`.

## Preguntas abiertas

(ninguna)
