# gcsgrep — spec

> **Estado: borrador para revisión.** Revisión 1 (2026-09-22) aplicada; todavía no
> pasó el gate. Las decisiones de esa revisión están en `INCONSISTENCIAS.md`
> (§15–§21).
>
> Construida a partir de [`gcsgrep-base-context.md`](./gcsgrep-base-context.md), que
> ya incorpora las decisiones de [`INCONSISTENCIAS.md`](./INCONSISTENCIAS.md).
>
> Regla estructural: **cada FR y cada BR tiene un VC.** Si una línea no se puede
> verificar, no está especificada.
>
> Esta spec describe la **v1 completa**. Cómo se parte en iteraciones, y qué se
> construye en cada una, se decide en `gcsgrep-plan.md`, no acá.

## Propósito

Permitir que alguien de desarrollo u operaciones busque texto dentro del
**contenido** de objetos de Google Cloud Storage con la experiencia de `grep`, sin
bajarlos a disco, sin escribir nada en GCS y sin ampliar sus permisos.

## Alcance

### Dentro

- Un único comando: `gcsgrep [flags] [--] <patrón> <ubicación>`.
- Búsqueda línea por línea, literal por defecto o RE2 con `-E`.
- Flags `-E`, `-i`, `-n`, `-c`, `-l`, `--max` y `--concurrency`.
- Ubicaciones `gs://bucket/`, `gs://bucket/prefijo/` (recursivas) y
  `gs://bucket/objeto` (objeto puntual).
- Lectura por streaming de objetos de texto UTF-8; los objetos que no son texto se
  saltean.
- Autenticación solo por ADC (Application Default Credentials).
- Guardrail de costo por cantidad de objetos listados.
- Exit codes estilo `grep` (0 / 1 / 2), aptos para scripting.

### Fuera

Cada uno de estos es una decisión tomada, no un olvido:

- **Escribir en GCS** de cualquier forma: copiar, mover, borrar, cambiar metadata o
  permisos (ver BR-1).
- Otros proveedores (S3, Azure Blob) y otras interfaces (web, API, librería).
- Flags de `grep` que no están en la lista de arriba: `-v`, `-r`, `--include`,
  `-o`, `-w`, contexto (`-A`/`-B`/`-C`), color.
- Salida en JSON o con color.
- Autenticación con un archivo de service account pasado por flag.
- Descomprimir `.gz` al vuelo. Un objeto gzip se saltea como no-texto (BR-7).
- Codificaciones distintas de UTF-8 (Latin-1 y similares se tratan como no-texto).
- Un límite por **bytes** leídos. El guardrail solo cuenta objetos (BR-3).
- Reintentos automáticos y timeout configurable.
- **Consistencia ante objetos que cambian durante la lectura.** Se lee lo que haya al
  abrir el stream, sin fijar ni verificar la generación del objeto.
- Agrupar la salida por objeto. Con concurrencia, las líneas de objetos distintos
  pueden entrelazarse (FR-21).

### Restricciones técnicas

- Go ≥ 1.22; regex con el paquete `regexp` de la stdlib (RE2); cliente
  `cloud.google.com/go/storage`.
- Binario único compilado con `go build -o gcsgrep ./cmd/gcsgrep`. Todos los VCs
  invocan `./gcsgrep`.

## Actores

| Actor | Descripción |
|---|---|
| **Persona usuaria** | Dev u ops que ya usa `grep`, ejecuta el comando en una shell y lee la salida |
| **Script** | Ejecuta el comando y decide por el exit code, sin parsear la salida |
| **GCS** | Lista y sirve los objetos. Puede fallar, colgarse o negar acceso |
| **Credenciales ADC** | Identidad de quien invoca. Define qué puede leer la herramienta, y nada más |

## Entorno de verificación

Los VCs se ejecutan contra estos entornos. Construirlos es parte del plan, no de la
spec.

- **Bucket de fixtures `$B`.** Contiene exactamente las carpetas `logs/`, `data/` y
  `edge-cases/` de [`test-fixtures/`](./test-fixtures/README.md), subidas con
  `gcloud storage cp -r test-fixtures/logs test-fixtures/data test-fixtures/edge-cases gs://$B/`
  (sin el `README.md`, que contiene `timeout` y alteraría los resultados), más el
  prefijo `sniffing/` con estos objetos, que hay que agregar a los fixtures:

  | Objeto | Contenido | Para qué |
  |---|---|---|
  | `sniffing/utf8_split_at_512.txt` | 511 bytes ASCII que empiezan con `sniff split`, luego `ñ` (`c3 b1`) en los bytes 512–513, luego `\nsniff after\n` | Secuencia multibyte cortada por el borde de la muestra (BR-6) |
  | `sniffing/truncated_utf8_under_512.txt` | `sniff truncated\n` seguido de un único byte `c3` (menos de 512 bytes en total) | Secuencia incompleta al final de un objeto chico (BR-6) |
  | `sniffing/invalid_after_512.txt` | 600 bytes de líneas ASCII válidas, luego la línea `sniff ` + byte `ff` + ` raw\n` | UTF-8 inválido después de la muestra (BR-6) |
  | `sniffing/text_named.png` | `sniff named png\n`, subido con `Content-Type: image/png` | Ni extensión ni content-type deciden (BR-6) |
  | `sniffing/gzip_no_extension` | Un gzip real (magic bytes `1f 8b`), sin extensión | El gzip se detecta por contenido (BR-7) |
  | `sniffing/transcoded.log` | `sniff transcoded\n`, comprimido antes de subirlo con `Content-Encoding: gzip` y sin `Cache-Control: no-transform` | Decompressive transcoding de GCS (BR-7) |

- **Bucket de performance `$P`.** Dataset de NFR-1 bajo `gs://$P/nfr1/`, objetos de
  NFR-2 bajo `gs://$P/nfr2/`, con la composición que fija cada NFR, y el objeto
  `gs://$P/bw/100mb.bin` (100 000 000 bytes) para medir el ancho de banda de NFR-1.
- **Servidor de prueba.** Un servidor HTTP local que emula el subconjunto de la API
  JSON de GCS que usa la herramienta (listar, consultar metadata y leer contenido),
  al que se apunta con `STORAGE_EMULATOR_HOST`. Expone el bucket `fake`, registra
  cada request recibido y permite inyectar fallas por request: responder `500`,
  aceptar la conexión y no responder nunca, o enviar el cuerpo en tramos espaciados.
- **Entorno sin ADC.** `env -u GOOGLE_APPLICATION_CREDENTIALS HOME=$(mktemp -d)
  CLOUDSDK_CONFIG=$(mktemp -d)`, en una máquina fuera de GCP (sin metadata server).
- **Chequeo P (precedencia).** Donde un VC dice "cumple el chequeo P", el comando se
  repite en el entorno sin ADC con `STORAGE_EMULATOR_HOST` apuntando al servidor de
  prueba, y se verifica que (1) el mensaje de stderr es el del error de uso
  esperado, no el de credenciales ausentes, y (2) el servidor de prueba registra
  **cero** requests. Así se observa que el error se detecta antes de autenticar y de
  cualquier operación contra GCS.

Convenciones: "stdout exacto" compara el conjunto de líneas sin importar el orden
entre objetos distintos (ver FR-21). En los mensajes, `<...>` es un valor variable.

---

## Requerimientos funcionales

### Búsqueda

### FR-1 · Buscar un literal bajo un prefijo

**Dado** un prefijo con objetos de texto,
**Cuando** la persona ejecuta `./gcsgrep <patrón> gs://<bucket>/<prefijo>/`,
**Entonces** el sistema imprime por stdout cada línea que contiene el patrón, con la
forma `gs://<bucket>/<objeto>:<texto>`, y sale con código `0`.

> **VC-1** — `./gcsgrep timeout gs://$B/logs/app/` sale con código `0` y su stdout
> es exactamente estas tres líneas (en cualquier orden entre objetos):
> `gs://$B/logs/app/api.log:2026-09-22 10:05:22 ERROR Request timeout after 3000ms uri=/api/v1/checkout`,
> `gs://$B/logs/app/api.log:2026-09-22 10:50:23 ERROR Gateway timeout=504 from upstream payment service`
> y `gs://$B/logs/app/worker.log:job_id=102 queue=default status=ERROR reason=timeout_exceeded retries=3`.

### FR-2 · Informar una corrida sin matches

**Dado** una ubicación cuyos objetos de texto no contienen el patrón,
**Cuando** la persona ejecuta la búsqueda,
**Entonces** stdout queda vacío y el sistema sale con código `1`, para que un script
lo detecte sin parsear la salida.

> **VC-2** — `./gcsgrep palabra_inexistente_xyz gs://$B/` sale con código `1` y
> stdout tiene 0 bytes. En stderr solo aparecen avisos de objetos salteados (BR-5).

### FR-3 · Tratar el patrón como literal por defecto

**Dado** un patrón con caracteres que en una regex serían especiales,
**Cuando** la persona lo busca sin `-E`,
**Entonces** el sistema busca la cadena exacta: ningún carácter tiene significado
especial y ningún patrón no vacío es inválido (el patrón vacío es FR-31).

> **VC-3** — `./gcsgrep 'job_id=10.' gs://$B/logs/app/worker.log` sale con código
> `1` y stdout vacío, porque la cadena `job_id=10.` no aparece literalmente.
> `./gcsgrep '(' gs://$B/logs/` sale con código `1`, no `2`.

### FR-4 · Buscar con una regex RE2 usando `-E`

**Dado** un patrón que es una regex RE2 válida,
**Cuando** la persona ejecuta `./gcsgrep -E <patrón> <ubicación>`,
**Entonces** el sistema imprime las líneas que matchean la regex y sale con código
`0`.

> **VC-4** — `./gcsgrep -E 'job_id=\d+ queue=\w+ status=ERROR' gs://$B/logs/app/worker.log`
> sale con código `0` e imprime exactamente las líneas de `job_id=102` y
> `job_id=104`. `./gcsgrep -E 'job_id=10.' gs://$B/logs/app/worker.log` imprime las 6
> líneas del objeto.

### FR-5 · Rechazar una regex inválida

**Dado** un patrón que no es una regex RE2 válida,
**Cuando** la persona ejecuta la búsqueda con `-E`,
**Entonces** el sistema sale con código `2`, escribe
`gcsgrep: invalid pattern: <detalle del motor>` por stderr y no opera contra GCS.

> **VC-5** — `./gcsgrep -E '(' gs://$B/logs/` sale con código `2`, stdout vacío, y la
> primera línea de stderr empieza con `gcsgrep: invalid pattern: `. Cumple el
> chequeo P.

### FR-6 · Ignorar mayúsculas con `-i`

**Dado** un objeto con el patrón escrito con distintas combinaciones de mayúsculas,
**Cuando** la persona ejecuta la búsqueda con `-i`,
**Entonces** el sistema matchea sin distinguir mayúsculas de minúsculas, también en
letras no ASCII (`ñ`/`Ñ`, `á`/`Á`), con el mismo resultado en cualquier máquina: no
depende del locale ni de variables como `LANG`. Se usa el case folding simple de
Unicode de RE2 (`(?i)`), así que las equivalencias de más de un carácter (`ß`/`SS`) no
matchean.

> **VC-6** — `./gcsgrep -i timeout gs://$B/logs/app/api.log` sale con código `0` e
> imprime 4 líneas (las que contienen `timeout`, `TIMEOUT` y `Timeout`). Sin `-i`, el
> mismo comando imprime 2. Contra el servidor de prueba, con `fake/acentos.log` cuya
> única línea es `ÑANDÚ ÁRBOL`, `LANG=C ./gcsgrep -i 'ñandú árbol' gs://fake/acentos.log`
> sale con código `0` e imprime esa línea; sin `-i`, sale con código `1`.

### FR-7 · Numerar líneas con `-n`

**Dado** un objeto con matches,
**Cuando** la persona ejecuta la búsqueda con `-n`,
**Entonces** cada línea sale con la forma `gs://<bucket>/<objeto>:<línea>:<texto>`,
donde la primera línea del objeto es la `1`.

> **VC-7** — `./gcsgrep -n timeout gs://$B/logs/app/api.log` sale con código `0` y su
> stdout es exactamente
> `gs://$B/logs/app/api.log:4:2026-09-22 10:05:22 ERROR Request timeout after 3000ms uri=/api/v1/checkout`
> y
> `gs://$B/logs/app/api.log:14:2026-09-22 10:50:23 ERROR Gateway timeout=504 from upstream payment service`,
> en ese orden.

### FR-8 · Recortar el `\r` de las líneas `\r\n`

**Dado** un objeto con líneas terminadas en `\r\n`,
**Cuando** la persona busca en él,
**Entonces** el sistema recorta el `\r` final de cada línea antes de matchear y antes
de imprimir.

> **VC-8** — `./gcsgrep -n timeout gs://$B/data/windows_crlf.txt` imprime exactamente
> `gs://$B/data/windows_crlf.txt:2:Linea 2 con timeout y terminacion Windows` y stdout
> no contiene ningún byte `0x0d`. `./gcsgrep -E 'Windows$' gs://$B/data/windows_crlf.txt`
> sale con código `0`.

### Ubicación

### FR-9 · Buscar en todo un bucket, recursivamente

**Dado** un bucket con objetos anidados en varios niveles,
**Cuando** la persona usa la ubicación `gs://<bucket>/`,
**Entonces** el sistema busca en todos los objetos del bucket, a cualquier
profundidad.

> **VC-9** — `./gcsgrep -l timeout gs://$B/` sale con código `0` y su stdout es
> exactamente estos 9 nombres: `logs/app/api.log`, `logs/app/worker.log`,
> `logs/db/postgres.log`, `data/users.csv`, `data/config.json`,
> `data/windows_crlf.txt`, `data/subdata/deep_nested.txt`,
> `edge-cases/small_under_512.txt` y `edge-cases/long_line_exceeds_1mb.log`, cada uno
> con el prefijo `gs://$B/`.

### FR-10 · Buscar bajo un prefijo, recursivamente

**Dado** un bucket con objetos dentro y fuera de un prefijo,
**Cuando** la persona usa la ubicación `gs://<bucket>/<prefijo>/`,
**Entonces** el sistema busca solo en los objetos cuyo nombre empieza con
`<prefijo>/`, incluidos los de subprefijos.

> **VC-10** — `./gcsgrep -l timeout gs://$B/data/` sale con código `0` y su stdout es
> exactamente `gs://$B/data/users.csv`, `gs://$B/data/config.json`,
> `gs://$B/data/windows_crlf.txt` y `gs://$B/data/subdata/deep_nested.txt`. Ningún
> objeto fuera de `data/` aparece.

### FR-11 · Buscar en un único objeto

**Dado** un objeto existente,
**Cuando** la persona usa su nombre exacto sin `/` final, `gs://<bucket>/<objeto>`,
**Entonces** el sistema busca solo en ese objeto, sin listar el bucket.

> **VC-11** — `./gcsgrep timeout gs://$B/logs/app/api.log` sale con código `0` e
> imprime solo líneas de `api.log`. Contra el servidor de prueba, con los objetos
> `fake/a.log` y `fake/a.log.bak`, `./gcsgrep timeout gs://fake/a.log` no genera
> ningún request de listado y no lee `a.log.bak`.

### FR-12 · Rechazar un objeto puntual inexistente

**Dado** que no existe un objeto con el nombre exacto indicado,
**Cuando** la persona usa una ubicación sin `/` final,
**Entonces** el sistema sale con código `2` y escribe por stderr
`gcsgrep: object not found: gs://<bucket>/<ruta> (to search under a prefix, end the location with /)`.
No busca en otros objetos que empiecen igual.

> **VC-12** — `./gcsgrep timeout gs://$B/logs/app` (existe `logs/app/` como prefijo,
> no como objeto) sale con código `2`, stdout vacío, y stderr es exactamente
> `gcsgrep: object not found: gs://$B/logs/app (to search under a prefix, end the location with /)`.

### FR-13 · Rechazar una ubicación con formato inválido

**Dado** una ubicación sin el esquema `gs://`, sin nombre de bucket o sin la `/` que
sigue al bucket,
**Cuando** la persona ejecuta la búsqueda,
**Entonces** el sistema sale con código `2`, escribe
`gcsgrep: invalid location: "<ubicación>"` por stderr y no opera contra GCS.

> **VC-13** — Para cada una de `logs/`, `s3://$B/`, `gs://`, `gs:///logs/` y
> `gs://$B`, `./gcsgrep timeout <ubicación>` sale con código `2`, stdout vacío, y la
> primera línea de stderr empieza con `gcsgrep: invalid location: `. Cada caso cumple
> el chequeo P.

### FR-14 · Aceptar un bucket o prefijo sin objetos

**Dado** un bucket existente o un prefijo que no contiene ningún objeto,
**Cuando** la persona busca en él,
**Entonces** el sistema sale con código `1` con stdout vacío. No es un error.

> **VC-14** — `./gcsgrep timeout gs://$B/no-existe/ 2>err.txt` sale con código `1`,
> stdout vacío y `err.txt` de 0 bytes.

### FR-15 · Rechazar un bucket inexistente

**Dado** un bucket que no existe,
**Cuando** la persona busca en él,
**Entonces** el sistema sale con código `2` y lo informa por stderr, distinto de un
bucket vacío (FR-14).

> **VC-15** — `./gcsgrep timeout gs://gcsgrep-bucket-que-no-existe-7f3a/` sale con
> código `2`, stdout vacío, y la primera línea de stderr empieza con `gcsgrep: ` y
> contiene `gcsgrep-bucket-que-no-existe-7f3a`.

### FR-33 · Ignorar los marcadores de carpeta

**Dado** un listado que incluye objetos cuyo nombre termina en `/` (los "marcadores
de carpeta" que crea, por ejemplo, la consola de GCS),
**Cuando** la herramienta procesa la ubicación,
**Entonces** esos objetos no se leen, no aparecen en stdout con ningún flag, no
generan aviso y no cuentan en el total del progreso (FR-27). Sí cuentan para el tope
de BR-3, porque el listado los devuelve.

> **VC-45** — Servidor de prueba con `fake/dir/` (0 bytes) y `fake/dir/a.log`
> (`timeout a`). `./gcsgrep -c timeout gs://fake/ 2>err.txt` sale con código `0`,
> su stdout es exactamente `gs://fake/dir/a.log:1`, `err.txt` tiene 0 bytes, y el
> servidor no registra ninguna lectura de contenido de `dir/`.
> `./gcsgrep --max 1 timeout gs://fake/` sale con código `2` con el mensaje de BR-3
> para `1`.

### Formatos de salida

### FR-16 · Contar líneas por objeto con `-c`

**Dado** una ubicación con objetos de texto con y sin matches,
**Cuando** la persona ejecuta la búsqueda con `-c`,
**Entonces** el sistema imprime una línea `gs://<bucket>/<objeto>:<n>` por cada
objeto de texto leído, donde `<n>` es la cantidad de **líneas** que matchean
(incluido `0`). Los objetos salteados o fallidos no aparecen en stdout. Si todos los
conteos son `0`, sale con código `1`.

> **VC-16** — `./gcsgrep -c timeout gs://$B/logs/` sale con código `0` y su stdout es
> exactamente `gs://$B/logs/server.log:0`, `gs://$B/logs/app/api.log:2`,
> `gs://$B/logs/app/worker.log:1`, `gs://$B/logs/db/postgres.log:1` y
> `gs://$B/logs/db/redis.log:0`; `logs/archive/old_logs.log.gz` no aparece.
> `./gcsgrep -c palabra_inexistente_xyz gs://$B/logs/db/` imprime
> `gs://$B/logs/db/postgres.log:0` y `gs://$B/logs/db/redis.log:0` y sale con código
> `1`.

### FR-17 · Listar objetos que matchean con `-l`

**Dado** una ubicación con objetos que matchean,
**Cuando** la persona ejecuta la búsqueda con `-l`,
**Entonces** el sistema imprime `gs://<bucket>/<objeto>` una vez por cada objeto con
al menos un match, y deja de leer cada objeto en cuanto encuentra su primer match.

> **VC-17** — `./gcsgrep -l timeout gs://$B/logs/db/` imprime exactamente
> `gs://$B/logs/db/postgres.log` y sale con código `0`;
> `./gcsgrep -l palabra_inexistente_xyz gs://$B/logs/db/` sale con código `1` y stdout
> vacío. Contra el servidor de prueba, con `fake/big.log` de 100 MB cuya primera
> línea contiene `timeout`, `./gcsgrep -l timeout gs://fake/big.log` sale con código
> `0` y el servidor registra que el cliente cerró la conexión antes de recibir el
> cuerpo completo.

### FR-18 · Rechazar combinaciones de flags sin sentido

**Dado** que `-c` y `-l` son excluyentes, y que `-n` no aplica a ninguno de los dos,
**Cuando** la persona combina `-c` con `-l`, o `-n` con `-c` o con `-l`,
**Entonces** el sistema sale con código `2` sin operar contra GCS.

> **VC-18** — Para cada una de `-c -l`, `-n -c` y `-n -l`, `./gcsgrep <flags> timeout gs://$B/logs/`
> sale con código `2`, stdout vacío, y la primera línea de stderr empieza con
> `gcsgrep: ` y nombra los dos flags en conflicto. Cada caso cumple el chequeo P.

### Invocación y concurrencia

### FR-19 · Parsear la invocación y el marcador `--`

**Dado** la sintaxis `gcsgrep [-E] [-i] [-n] [-c | -l] [--max <N>|unlimited] [--concurrency <N>] [--] <patrón> <ubicación>`,
**Cuando** la persona pasa `--`,
**Entonces** todo lo que sigue se trata como posicional aunque empiece con `-`. Sin
`--`, un argumento que empieza con `-` y no es un flag reconocido, o una cantidad de
posicionales distinta de dos, es un error de uso (código `2`).

> **VC-19** — `./gcsgrep -- -1] gs://$B/logs/db/postgres.log` sale con código `0` e
> imprime las 5 líneas del objeto. `./gcsgrep -1] gs://$B/logs/db/postgres.log`,
> `./gcsgrep timeout` y `./gcsgrep timeout gs://$B/logs/ extra` salen con código `2`,
> stdout vacío y la primera línea de stderr empieza con `gcsgrep: `. Los tres casos
> de error cumplen el chequeo P.

### FR-20 · Validar `--concurrency`

**Dado** que `--concurrency` acepta un entero entre 1 y 32 (default 4),
**Cuando** la persona pasa un valor no entero, menor que 1 o mayor que 32,
**Entonces** el sistema sale con código `2` y escribe
`gcsgrep: invalid value for --concurrency: "<valor>" (integer between 1 and 32)` por
stderr, sin operar contra GCS.

> **VC-20** — Para cada uno de `0`, `33`, `-1`, `abc` y `1.5`,
> `./gcsgrep --concurrency <valor> timeout gs://$B/logs/` sale con código `2`, stdout
> vacío, y stderr es exactamente el mensaje de arriba con ese valor. Cada caso cumple
> el chequeo P. Con `1` y con `32`, el mismo comando sale con código `0`.

### FR-29 · Aceptar cada flag solo en su forma separada

**Dado** que cada flag se pasa como un argumento propio (`-i -n`, no `-in`) y que los
flags con valor lo reciben en el argumento siguiente (`--max 5`, no `--max=5`),
**Cuando** la persona combina flags cortos en un solo argumento (`-in`, `-ic`, `-Ei`)
o pega el valor con `=` (`--max=5`, `--concurrency=8`),
**Entonces** el sistema sale con código `2` y escribe
`gcsgrep: unknown flag: "<argumento>" (pass each flag separately, values after a space)`
por stderr, sin operar contra GCS.

> **VC-41** — Para cada uno de `-in`, `-ic`, `-Ei`, `--max=5` y `--concurrency=8`,
> `./gcsgrep <argumento> timeout gs://$B/logs/` sale con código `2`, stdout vacío, y
> stderr es exactamente el mensaje de arriba con ese argumento. Cada caso cumple el
> chequeo P. `./gcsgrep -i -n timeout gs://$B/logs/app/api.log` sale con código `0`.

### FR-30 · Rechazar un flag repetido

**Dado** cualquier flag de la invocación, con o sin valor,
**Cuando** la persona lo pasa más de una vez,
**Entonces** el sistema sale con código `2` y escribe
`gcsgrep: flag specified more than once: <flag>` por stderr, sin operar contra GCS,
aunque los valores coincidan.

> **VC-42** — Para cada una de `-i -i`, `--max 5 --max 10`, `--max 5 --max 5` y
> `--concurrency 2 --concurrency 8`, `./gcsgrep <flags> timeout gs://$B/logs/` sale
> con código `2`, stdout vacío, y stderr es exactamente el mensaje de arriba con el
> flag repetido (`-i`, `--max`, `--max`, `--concurrency`). Cada caso cumple el
> chequeo P.

### FR-31 · Rechazar un patrón vacío

**Dado** un patrón de longitud cero, con o sin `-E`,
**Cuando** la persona ejecuta la búsqueda,
**Entonces** el sistema sale con código `2` y escribe `gcsgrep: empty pattern` por
stderr, sin operar contra GCS.

> **VC-43** — `./gcsgrep '' gs://$B/logs/`, `./gcsgrep -E '' gs://$B/logs/` y
> `./gcsgrep -- '' gs://$B/logs/` salen con código `2`, stdout vacío, y stderr es
> exactamente `gcsgrep: empty pattern`. Cada caso cumple el chequeo P.

### FR-21 · Mantener cada línea entera y el orden dentro de cada objeto

**Dado** varios objetos leídos en paralelo,
**Cuando** la corrida produce matches en más de un objeto,
**Entonces** las líneas de objetos distintos pueden salir entrelazadas, pero cada
línea de salida se escribe completa y las de un mismo objeto salen en el orden en
que están en el objeto.

> **VC-21** — Se ejecuta 10 veces `./gcsgrep -n --concurrency 32 timeout gs://$B/`.
> En cada corrida: (1) el multiconjunto de líneas de stdout es idéntico al de
> `./gcsgrep -n --concurrency 1 timeout gs://$B/`, y (2) para cada objeto, los números
> de línea aparecen en orden estrictamente creciente.

### Fallos

### FR-22 · Seguir cuando un objeto no se puede leer

**Dado** una corrida en la que un objeto listado falla al leerse,
**Cuando** la herramienta procesa la ubicación,
**Entonces** escribe `gcsgrep: gs://<bucket>/<objeto>: read error: <detalle>` por
stderr, sigue con el resto de los objetos, emite completa la salida de los demás y
sale con código `2` aunque haya habido matches.

> **VC-22** — Servidor de prueba con `fake/a.log` (`timeout a`), `fake/b.log` (la
> lectura responde `500`) y `fake/c.log` (`timeout c`).
> `./gcsgrep timeout gs://fake/` imprime exactamente `gs://fake/a.log:timeout a` y
> `gs://fake/c.log:timeout c`, stderr contiene una línea que empieza con
> `gcsgrep: gs://fake/b.log: read error: `, y sale con código `2`.

### FR-23 · Abortar si falla el listado o la consulta del objeto puntual

**Dado** que GCS falla al listar la ubicación, o al consultar el objeto puntual,
**Cuando** la herramienta intenta obtener los objetos a leer,
**Entonces** aborta la corrida con código `2` sin leer ningún objeto.

> **VC-23** — Con el servidor de prueba respondiendo `500` al listado,
> `./gcsgrep timeout gs://fake/` sale con código `2`, stdout vacío, stderr empieza con
> `gcsgrep: `, y el servidor no registra ninguna lectura de contenido. Con el
> servidor respondiendo `500` a la consulta de metadata de `fake/a.log`,
> `./gcsgrep timeout gs://fake/a.log` sale con código `2`.

### FR-24 · No reintentar

**Dado** un request contra GCS que falla,
**Cuando** la herramienta lo procesa,
**Entonces** no lo reintenta: la falla se resuelve según FR-22 o FR-23 en el primer
intento.

> **VC-24** — En los escenarios de VC-22 y VC-23, el servidor de prueba registra
> exactamente **un** request al recurso que falla (la lectura de `b.log`, el listado o
> la metadata de `a.log`, según el caso).

### FR-25 · Cortar por inactividad a los 30 s

**Dado** un request contra GCS que pasa 30 segundos sin recibir datos,
**Cuando** se cumple ese plazo,
**Entonces** el request se corta. En una lectura, el objeto se reporta como fallido
con `gcsgrep: gs://<bucket>/<objeto>: read error: timeout after 30s without data` y
la corrida sigue (FR-22). En el listado o en la consulta del objeto puntual, la
corrida aborta con código `2`. El plazo cuenta desde el último byte recibido, así
que una lectura lenta pero continua no se corta.

> **VC-25** — Servidor de prueba: (1) con la lectura de `fake/b.log` colgada,
> `./gcsgrep timeout gs://fake/` imprime los matches de `a.log` y `c.log`, stderr
> contiene exactamente
> `gcsgrep: gs://fake/b.log: read error: timeout after 30s without data`, y sale con
> código `2`; (2) con el listado colgado, sale con código `2` y stdout vacío; (3) con
> `fake/slow.log` enviado en tramos cada 10 s durante 60 s, cuya última línea
> contiene `timeout`, `./gcsgrep timeout gs://fake/slow.log` sale con código `0` e
> imprime esa línea.

### FR-26 · Rechazar la corrida sin credenciales

**Dado** que no hay Application Default Credentials disponibles,
**Cuando** la persona ejecuta una búsqueda con argumentos válidos,
**Entonces** el sistema sale con código `2` con un mensaje por stderr, antes de
cualquier operación contra GCS.

> **VC-26** — En el entorno sin ADC, con `STORAGE_EMULATOR_HOST` apuntando al
> servidor de prueba, `./gcsgrep timeout gs://fake/` sale con código `2`, stdout
> vacío, la primera línea de stderr empieza con `gcsgrep: ` y contiene
> `credentials`, y el servidor registra cero requests.

### FR-32 · Terminar en silencio si se cierra stdout

**Dado** que quien lee stdout lo cierra antes de que termine la corrida (por ejemplo,
`gcsgrep … | head -1`),
**Cuando** la herramienta intenta escribir la siguiente línea,
**Entonces** termina de inmediato como `grep`: el proceso muere por `SIGPIPE` (la
shell ve el código `141`), no escribe nada en stderr y no inicia la lectura de más
objetos.

> **VC-44** — Servidor de prueba con 2500 objetos bajo `fake/many/`, cada uno con
> una línea que contiene `timeout`. En bash,
> `./gcsgrep --max unlimited timeout gs://fake/many/ 2>err.txt | head -1` imprime una
> sola línea, `${PIPESTATUS[0]}` es `141`, `err.txt` tiene 0 bytes, y el servidor
> registra lecturas de contenido de menos de 100 objetos.

### Progreso

### FR-27 · Mostrar progreso cuando stderr es una terminal

**Dado** que stderr es una TTY,
**Cuando** la herramienta está leyendo objetos,
**Entonces** muestra por stderr una única línea `<procesados>/<total> objects processed`
que reescribe con `\r` cada vez que termina un objeto (con match, sin match,
salteado o fallido). Antes de cada aviso por stderr, y al terminar la corrida,
borra esa línea con `\r` seguido de `ESC[K`.

> **VC-27** — `script -q out.txt ./gcsgrep timeout gs://$B/logs/` (stdout y stderr en
> una pseudo-terminal): `out.txt` contiene `1/6 objects processed` y
> `6/6 objects processed`, el aviso de `old_logs.log.gz` empieza al comienzo de una
> línea (precedido por `\r` + `ESC[K`), y la última secuencia de stderr antes del fin
> es `\r` + `ESC[K`.

### FR-28 · No emitir progreso fuera de una terminal

**Dado** que stderr está redirigido a un archivo o a un pipe,
**Cuando** la herramienta corre,
**Entonces** no emite ningún progreso: en una corrida sin avisos ni errores, stderr
queda vacío.

> **VC-28** — `./gcsgrep timeout gs://$B/logs/app/ 2>err.txt` sale con código `0` y
> `err.txt` tiene 0 bytes.

---

## Reglas de negocio

### BR-1 · Solo lectura

La herramienta nunca escribe en GCS: no crea, modifica ni borra objetos, metadata ni
permisos.

*Fundamento:* restricción dura del enunciado. gcsgrep busca; no gestiona.
*Excepciones:* ninguna.

> **VC-29** — Toda la suite de VCs contra `$B` se ejecuta con una identidad que solo
> tiene `roles/storage.objectViewer` sobre `$B`, y pasa. La salida de
> `gcloud storage objects list "gs://$B/**" --format="value(name,generation,metageneration)"`
> es idéntica antes y después de la suite.

### BR-2 · No ampliar el acceso de quien invoca

La herramienta usa solo las credenciales ADC de quien la invoca. Si esa identidad no
puede leer un bucket o un objeto, gcsgrep tampoco.

*Fundamento:* restricción dura del enunciado. Evita que la herramienta sea un vector
de escalamiento de privilegios.
*Excepciones:* ninguna.

> **VC-30** — Con la identidad de VC-29 (solo lectura sobre `$B`) y un bucket `$B2`
> sin permisos para ella, `./gcsgrep timeout gs://$B2/` sale con código `2` y stdout
> vacío. Con una identidad que sí puede leer `$B2`, el mismo comando sale con código
> `0` o `1`.

### BR-3 · Tope de objetos listados

Por defecto, una corrida escanea como máximo **1000 objetos**. El conteo incluye
**todo** objeto que devuelve el listado, sea texto o no. El listado se detiene en
cuanto aparece el objeto N+1 (N = tope vigente), sin seguir paginando, y la corrida
aborta **antes de leer ningún objeto** con código `2` y el mensaje
`gcsgrep: more than <N> objects under <ubicación>; use --max <N> or --max unlimited`,
donde el primer `<N>` es el tope vigente y el segundo es texto literal.

*Fundamento:* leer objetos cuesta dinero. Un tope por cantidad es simple de explicar
y de verificar, y cortar en N+1 evita que el propio guardrail pague por paginar un
listado que ya se sabe excedido. 1000 coincide con la escala validada en NFR-1.
*Excepciones:* una ubicación de objeto puntual no lista nada, así que el tope nunca
se dispara. `--max` cambia el tope (BR-4).
*Limitación conocida:* el tope no acota bytes; 1000 objetos grandes pueden implicar
mucha lectura.

> **VC-31** — `./gcsgrep --max 5 timeout gs://$B/edge-cases/` (6 objetos, 3 de ellos
> no-texto) sale con código `2`, stdout vacío, y stderr es exactamente
> `gcsgrep: more than 5 objects under gs://$B/edge-cases/; use --max <N> or --max unlimited`,
> sin avisos de objetos salteados. Con `--max 6`, el mismo comando sale con código
> `0`. Contra el servidor de prueba con 2500 objetos bajo `fake/many/`,
> `./gcsgrep timeout gs://fake/many/` sale con código `2` con el mensaje para
> `1000`, el servidor no registra ninguna lectura de contenido, y no recibe ningún
> request de página posterior a la que contiene el objeto 1001.

### BR-4 · Cambiar el tope con `--max`

`--max <N>` con N entero ≥ 1 fija el tope en N, y `--max unlimited` lo desactiva.
Cualquier otro valor es un error de uso: código `2` y el mensaje
`gcsgrep: invalid value for --max: "<valor>" (integer >= 1 or unlimited)`, sin operar
contra GCS.

*Fundamento:* el tope protege del error, no del uso deliberado. Subirlo tiene que
ser una decisión explícita en la línea de comandos.
*Excepciones:* ninguna.

> **VC-32** — Para cada uno de `0`, `-3`, `abc` y `1.5`,
> `./gcsgrep --max <valor> timeout gs://$B/logs/` sale con código `2`, stdout vacío, y
> stderr es exactamente el mensaje de arriba con ese valor; cada caso cumple el
> chequeo P. Contra el servidor de prueba con 2500 objetos,
> `./gcsgrep --max unlimited timeout gs://fake/many/` no sale con código `2` y el
> servidor registra la lectura de los 2500 objetos.

### BR-5 · Los objetos que no son texto se saltean sin contar como fallo

Un objeto clasificado como no-texto (BR-6) no se busca: se escribe
`gcsgrep: gs://<bucket>/<objeto>: not a text file, skipped` por stderr y la corrida
sigue. Saltear no es un fallo y no afecta el exit code.

*Fundamento:* imprimir basura binaria en la terminal no le sirve a nadie, y un
bucket mixto es el caso normal, no un error.
*Excepciones:* ninguna.

> **VC-33** — `./gcsgrep -c timeout gs://$B/edge-cases/` sale con código `0`; su
> stdout es exactamente `gs://$B/edge-cases/empty.txt:0`,
> `gs://$B/edge-cases/small_under_512.txt:1` y
> `gs://$B/edge-cases/long_line_exceeds_1mb.log:3`; y stderr contiene exactamente
> tres avisos `not a text file, skipped`, para `binary_nullbyte.bin`,
> `sample_image.png` y `non_utf8_latin1.txt`. `./gcsgrep timeout gs://$B/edge-cases/sample_image.png`
> sale con código `1`, no `2`.

### BR-6 · "Es texto" se decide por los primeros 512 bytes

Un objeto es no-texto si sus primeros 512 bytes contienen un byte `0x00` o una
secuencia que no es UTF-8 válido. Si la muestra termina en medio de un carácter
multibyte que sigue después del byte 512, esos bytes finales no cuentan como
inválidos; si el objeto mide 512 bytes o menos, se analiza entero y una secuencia
incompleta al final sí es inválida. Un objeto vacío es texto. La clasificación se
decide una sola vez: bytes inválidos posteriores a la muestra no la cambian, no
generan aviso, y sus líneas se buscan e imprimen byte a byte. No se usan ni la
extensión ni el content-type.

*Fundamento:* el content-type es metadata que el uploader pudo poner mal o no poner;
el contenido es la única fuente confiable.
*Excepciones:* ninguna.
*Limitación conocida:* solo UTF-8 (ASCII incluido) cuenta como texto.

> **VC-34** — `./gcsgrep sniff gs://$B/sniffing/` imprime líneas de
> `utf8_split_at_512.txt` (incluida `sniff after`), `invalid_after_512.txt` y
> `text_named.png`, y escribe el aviso `not a text file, skipped` para
> `truncated_utf8_under_512.txt`. La línea impresa de `invalid_after_512.txt` es
> `gs://$B/sniffing/invalid_after_512.txt:sniff ` + byte `ff` + ` raw`, verificado con
> `od -An -tx1`. Ninguno de esos tres objetos de texto genera aviso.

### BR-7 · El gzip se detecta por contenido

Un objeto gzip empieza con los bytes `1f 8b`; `0x8b` no inicia ninguna secuencia
UTF-8 válida, así que el objeto es no-texto por BR-6 y se saltea por BR-5. No hay una
regla basada en el nombre. Un objeto con `Content-Encoding: gzip` se busca como
texto si GCS lo sirve descomprimido (*decompressive transcoding*). Para ello no
debe tener `Cache-Control: no-transform` y la lectura no debe solicitar
`Accept-Encoding: gzip`.

*Fundamento:* descomprimir al vuelo queda fuera de la v1, y una regla por extensión
contradiría BR-6.
*Excepción decidida:* los objetos servidos descomprimidos por GCS se buscan.

> **VC-35** — `./gcsgrep timeout gs://$B/logs/archive/` sale con código `1`, stdout
> vacío, y stderr contiene
> `gcsgrep: gs://$B/logs/archive/old_logs.log.gz: not a text file, skipped`.
> `./gcsgrep sniff gs://$B/sniffing/` escribe el mismo aviso para
> `gzip_no_extension` e imprime `gs://$B/sniffing/transcoded.log:sniff transcoded`.

### BR-8 · Línea de más de 1 MB

Si una línea supera **1 MB (1 048 576 bytes)**, solo se busca en su primer MB; el
resto no se analiza. Si hay match en esa porción, se imprime esa porción seguida de
`...`.

*Fundamento:* para imprimir una línea hay que tenerla entera en un buffer. Sin tope,
una línea de cientos de MB rompería la memoria acotada de NFR-2. Truncar conserva la
señal del match, que es más útil que descartar el objeto.
*Excepciones:* ninguna.

> **VC-36** — `./gcsgrep timeout=early gs://$B/edge-cases/long_line_exceeds_1mb.log`
> sale con código `0` e imprime una sola línea: el prefijo
> `gs://$B/edge-cases/long_line_exceeds_1mb.log:`, luego exactamente los primeros
> 1 048 576 bytes de la línea 1 y luego `...`.
> `./gcsgrep timeout=late gs://$B/edge-cases/long_line_exceeds_1mb.log` (el texto está
> en el byte 1 099 837 de la línea) sale con código `1`.

### BR-9 · Salida apta para scripting

stdout contiene solo resultados; todo error y aviso va a stderr. La primera línea de
stderr de cualquier error empieza con `gcsgrep: `. Ningún caso de error imprime un
panic ni un stack trace.

*Fundamento:* un script decide por el exit code y consume stdout como datos (actor
**Script**). Un aviso mezclado en stdout, o un stack trace en stderr, rompe ese
contrato y expone detalles internos que no le sirven a quien invoca.
*Excepciones:* ninguna. La línea de progreso (FR-27) también va a stderr.

> **VC-40** — Para cada caso de falla cubierto (VC-5, VC-12, VC-13, VC-15, VC-18,
> VC-19, VC-20, VC-23, VC-26, VC-31, VC-32, VC-41, VC-42 y VC-43), stdout está vacío,
> la primera línea de stderr empieza con `gcsgrep: `, y stderr no contiene `panic:`,
> `goroutine ` ni `.go:`. En VC-22 y VC-25, stderr cumple la misma condición.

---

## Requerimientos no funcionales

### NFR-1 · Rendimiento con 1000 objetos

Con `--concurrency 4` (default), gcsgrep procesa 1000 objetos de texto en **menos de
120 segundos**.

Condiciones: desde la laptop de desarrollo contra un bucket en la región de GCS más
cercana, con un ancho de banda de bajada medido de **al menos 50 Mbps**. El ancho de
banda se mide inmediatamente antes de las corridas, bajando `gs://$P/bw/100mb.bin`
con `gcloud storage cp` a `/dev/null` (100 000 000 bytes × 8 / segundos). Si da menos
de 50 Mbps, la medición no es válida: no pasa ni falla, se repite en otra red.

Dataset (unidades decimales): exactamente 1000 objetos UTF-8 de **100 000 bytes**
(100 MB en total), cada uno con 1000 líneas de exactamente 100 bytes (99 caracteres
ASCII más `\n`), sin binarios. Las líneas cuyo número es múltiplo de 100 contienen el
patrón literal y ninguna otra lo contiene: **exactamente el 1 %** de las líneas (10
por objeto). Se mide el tiempo de pared desde la invocación hasta el exit, listado
incluido.

> **VC-37** — Con un ancho de banda medido ≥ 50 Mbps, tres corridas de
> `./gcsgrep <patrón> gs://$P/nfr1/ > /dev/null` tienen una mediana de tiempo de
> pared menor a 120 s. El reporte registra el ancho de banda medido y los tres
> tiempos.

### NFR-2 · Memoria acotada con objetos grandes

La memoria no crece con el tamaño de los objetos: se leen por streaming y ningún
objeto se carga entero en memoria. Con `--concurrency 4` contra un objeto UTF-8 de
**1 GB (1 000 000 000 bytes)**, stdout a `/dev/null`, el pico de RSS es **≤ 100 MB**
y no supera en más de **20 MB** al pico medido con un objeto de **10 MB
(10 000 000 bytes)** de la misma composición.

Composición de ambos objetos: líneas de exactamente 100 bytes (99 caracteres ASCII
más `\n`); las líneas cuyo número es múltiplo de 100 contienen el patrón literal y
ninguna otra lo contiene (**exactamente el 1 %**).

> **VC-38** — Con `/usr/bin/time -l` (macOS) o `/usr/bin/time -v` (Linux),
> `./gcsgrep <patrón> gs://$P/nfr2/big-1g.log > /dev/null` reporta un *maximum
> resident set size* ≤ 100 MB, y la diferencia con la misma medición sobre
> `gs://$P/nfr2/big-10m.log` es ≤ 20 MB.

### NFR-3 · Cota de tiempo ante un GCS colgado

Contra un servidor que acepta la conexión y nunca responde, la corrida termina en
**≤ 35 segundos** desde que se inicia el request colgado, con el exit code y el
mensaje que corresponden a la fase (FR-25).

> **VC-39** — En los escenarios (1) y (2) de VC-25, el tiempo entre el request
> colgado registrado por el servidor de prueba y el exit de `./gcsgrep` es ≤ 35 s.

---

## Tabla de trazabilidad

| Requerimiento | Origen en el base context | VC | Camino |
|---|---|---|---|
| FR-1 | FR-a, FR-c | VC-1 | feliz |
| FR-2 | FR-e | VC-2 | borde (sin matches) |
| FR-3 | FR-a | VC-3 | borde |
| FR-4 | FR-a | VC-4 | feliz |
| FR-5 | FR-a | VC-5 | falla |
| FR-6 | FR-d | VC-6 | feliz |
| FR-7 | FR-c | VC-7 | feliz |
| FR-8 | FR-a | VC-8 | borde (`\r\n`) |
| FR-9 | FR-b | VC-9 | feliz |
| FR-10 | FR-b | VC-10 | feliz |
| FR-11 | FR-b | VC-11 | feliz |
| FR-12 | FR-b | VC-12 | falla |
| FR-13 | FR-b | VC-13 | falla |
| FR-14 | Pregunta 8 | VC-14 | borde (vacío) |
| FR-15 | Pregunta 8 | VC-15 | falla |
| FR-16 | FR-h | VC-16 | feliz + borde (todo 0) |
| FR-17 | FR-h | VC-17 | feliz |
| FR-18 | FR-h | VC-18 | falla |
| FR-19 | FR-i | VC-19 | borde + falla |
| FR-20 | FR-j | VC-20 | borde (límites) |
| FR-21 | FR-j | VC-21 | invariante |
| FR-22 | FR-f | VC-22 | falla parcial |
| FR-23 | NFR-c | VC-23 | falla |
| FR-24 | NFR-c | VC-24 | invariante |
| FR-25 | NFR-c | VC-25 | falla |
| FR-26 | Pregunta 2 | VC-26 | falla |
| FR-27 | FR-g | VC-27 | feliz |
| FR-28 | FR-g | VC-28 | borde (sin TTY) |
| FR-29 | FR-i (revisión) | VC-41 | falla |
| FR-30 | FR-i (revisión) | VC-42 | falla |
| FR-31 | FR-i (revisión) | VC-43 | falla |
| FR-32 | BR-9 (revisión) | VC-44 | borde (pipe cerrado) |
| FR-33 | FR-b (revisión) | VC-45 | borde (marcador de carpeta) |
| BR-1 | BR-a | VC-29 | invariante |
| BR-2 | BR-b | VC-30 | falla (acceso) |
| BR-3 | BR-c | VC-31 | borde (límite) |
| BR-4 | BR-c | VC-32 | borde + falla |
| BR-5 | BR-d | VC-33 | borde (mixto) |
| BR-6 | BR-d | VC-34 | borde (muestra) |
| BR-7 | BR-d | VC-35 | borde (gzip) |
| BR-8 | BR-e | VC-36 | borde (límite) |
| BR-9 | — | VC-40 | invariante |
| NFR-1 | NFR-a | VC-37 | medición |
| NFR-2 | NFR-b | VC-38 | medición |
| NFR-3 | NFR-c | VC-39 | medición |

**45 requerimientos (33 FR, 9 BR, 3 NFR), 45 VCs, 0 huérfanos.**

## Preguntas abiertas

Ninguna.

## Qué sigue

El plan de iteraciones va en `gcsgrep-plan.md`.
