# gcsgrep — requerimientos (base context refinado)

> **Estado: base context refinado.** Este documento es la evolución de
> [`gcsgrep-requirements.md`](./gcsgrep-requirements.md): mismas secciones, pero con
> las 10 preguntas abiertas resueltas, las reglas de negocio candidatas decididas
> (con fundamento) y los NFRs con umbrales concretos. También incorpora las
> decisiones que resolvieron los 14 puntos de [`INCONSISTENCIAS.md`](./INCONSISTENCIAS.md).
>
> Sigue **sin ser la spec formal**. El siguiente paso del pipeline es convertir esto
> en FRs atómicos en formato Dado/Cuando/Entonces con un VC por cada FR y BR, según
> indica [`enunciado.md`](./enunciado.md). Este documento fija el *qué se decidió*;
> la spec fijará el *cómo se verifica* cada cosa formalmente.

## La idea

Queremos la experiencia de `grep`, pero apuntada a Google Cloud Storage.

Hoy, para buscar un texto dentro de objetos de un bucket, hay que bajarlos primero
—con `gsutil cp` o similar— y recién ahí correr `grep`. Es lento, gasta ancho de
banda, y llena el disco de archivos que no querés.

Queremos una herramienta de línea de comandos que busque dentro del **contenido**
de los objetos remotos, sin bajarlos a disco:

```
gcsgrep "timeout" gs://logs/
```

y que muestre qué objeto matcheó, y dónde.

## Para quién es

Gente de desarrollo y de operaciones que ya usa `grep` todos los días y ya tiene
credenciales de GCP configuradas. No es una herramienta para usuarios finales ni
para gente que no conoce la línea de comandos.

## Qué NO es

- **No es un clon completo de `grep`.** No aspiramos a soportar todos los flags.
- **No es un gestor de GCS.** No copia, no mueve, no borra, no cambia permisos.
- **Solo CLI.** No hay interfaz web, ni API, ni librería para importar (por ahora).
- **Solo GCS en la v1.** S3 y Azure Blob quedan afuera, aunque el diseño no debería
  cerrarles la puerta para siempre.

## Decisiones de alcance (v1)

Resumen ejecutivo de lo que se decidió para la **versión 1** completa. El detalle
y el fundamento de cada punto está en las secciones de abajo y en "Preguntas
abiertas — resueltas". Esta tabla no define iteraciones: cómo se parte la v1 en
iteraciones (≥ 2) se decide en `gcsgrep-plan.md`.

| Dimensión | Decisión |
|---|---|
| Interpretación del patrón | Literal por defecto (cadena exacta, sin metacaracteres); con `-E` se interpreta como regex RE2 |
| Patrón inválido | Solo aplica con `-E`: regex RE2 mal formada → error de uso, exit 2, mensaje `gcsgrep: invalid pattern: <engine detail>`, detectado antes de cualquier operación contra GCS |
| Autenticación | Solo ADC (Application Default Credentials); sin credenciales → error de uso/config, exit 2, detectado antes de cualquier operación contra GCS |
| Sintaxis de ubicación | Esquema `gs://` obligatorio. `gs://bucket/` = todo el bucket; `gs://bucket/prefijo/` = todo lo que está bajo ese prefijo (recursivo); `gs://bucket/objeto` (sin `/` final) = un único objeto con ese nombre exacto |
| Ubicación inválida | Falta el esquema `gs://`, falta el bucket, falta la `/` después del bucket (`gs://bucket`), o URI malformada → error de uso, exit 2, detectado antes de cualquier operación contra GCS. Objeto puntual inexistente → exit 2 (detectado al consultar GCS) |
| Formato de salida | `gs://bucket/objeto:texto` por defecto; con `-n`, `gs://bucket/objeto:línea:texto`. Texto plano, sin color, sin JSON |
| Sintaxis de invocación | `gcsgrep [flags] [--] <patrón> <ubicación>`, posicionales en orden fijo; `--` marca fin de opciones (convención POSIX) |
| Flags de `grep` soportados | `-E`, `-i`, `-n`, `-c` (cuenta líneas; `objeto:<n>`, incluye `:0`), `-l` (`objeto`; corta la lectura al primer match) (`-c` y `-l` mutuamente excluyentes; `-n` incompatible con ambos) |
| Binarios | Se saltean (sniffing de los primeros 512 bytes: NUL o UTF-8 inválido; una secuencia multibyte cortada por el borde de la muestra no cuenta como inválida). Bytes inválidos después de la muestra no cambian la clasificación: se busca e imprime tal cual |
| Archivos `.gz` | Se saltean en v1 como consecuencia de BR-d (magic bytes `1f 8b`), sin regla por extensión. Objetos con `Content-Encoding: gzip` que GCS sirve descomprimidos se buscan como texto |
| Exit codes | Convención `grep`: 0 match / 1 sin match / 2 error. Bucket inexistente, objeto puntual inexistente, credenciales ausentes, fallo de listado, guardrail de `--max` disparado, al menos un objeto que falló al leerse (aunque haya habido matches), o cualquier error de uso → 2; bucket/prefijo sin objetos o sin matches → 1. Objetos salteados por no-texto no cuentan como fallo |
| Concurrencia | `--concurrency <N>`, `N` entero entre 1 y 32, default 4. Valor inválido → exit 2. Las líneas de objetos distintos pueden entrelazarse; dentro de un objeto se respeta el orden y una línea nunca se parte |
| Guardrail de costo | Tope por defecto de 1000 objetos **listados** (sin importar si después se clasifican como texto o no); override con `--max <N≥1>` o `--max unlimited`. El listado se corta al ver el objeto N+1 (no se cuenta el total). Si se dispara, exit 2. No limita bytes leídos (diferido) |
| Detección de "es texto" | Sniffing de los primeros 512 bytes, no content-type ni extensión. Solo UTF-8 se reconoce como texto |
| Objeto que cambia durante la lectura | No se controla en v1 |
| Progreso | Solo si stderr es una terminal (TTY): una única línea `X/total` reescrita con `\r` y borrada al terminar. Sin TTY no se emite progreso |
| Línea que excede 1 MB | Se trunca el texto impreso; el match igual se reporta |
| Separador de línea | `\n` y `\r\n` (se recorta el `\r` final) |
| Reintentos automáticos | Ninguno, ni en listado ni en lectura de objetos |
| Stack | Go ≥ 1.22; `regexp` de la stdlib (RE2); `cloud.google.com/go/storage`; binario compilado (`./gcsgrep`) |
| Timeout | Fijo, 30 s sin recibir datos (inactividad), en cada request de listado, consulta de objeto puntual y lectura. Lectura → objeto fallido (FR-f); listado/objeto puntual → exit 2. Sin flag en v1 |

## Requerimientos funcionales (refinados)

Siguen sin estar en formato Dado/Cuando/Entonces —eso es tarea de la spec— pero
ya no dejan comportamiento ambiguo ni mezclan varias conductas en un mismo ítem.

- **FR-a** — El usuario pasa un patrón y una ubicación (FR-b), y la herramienta
  busca el patrón línea por línea dentro del contenido de cada objeto de texto
  que abarca esa ubicación. Por defecto el patrón es **literal**: se
  busca la cadena exacta y ningún carácter tiene significado especial (`a.b`
  matchea solo los tres caracteres `a`, `.`, `b`). Con `-E`, el patrón se
  interpreta como expresión regular RE2. Una línea se delimita por `\n`; si está
  precedido por `\r` (`\r\n`), ese `\r` se recorta antes de matchear y de
  imprimir. Con `-E`, si el patrón no es una expresión regular RE2 válida, es un
  error de uso (exit code 2) con el mensaje
  `gcsgrep: invalid pattern: <engine detail>` por stderr, detectado antes de
  cualquier operación contra GCS (no se intenta autenticar, listar ni leer nada).
  Sin `-E` no existe patrón inválido: cualquier cadena es un literal válido.
- **FR-b** — La ubicación tiene tres formas válidas, y la `/` final decide cuál:
  - `gs://bucket/` — todo el bucket, recursivamente.
  - `gs://bucket/prefijo/` — todos los objetos cuyo nombre empieza con
    `prefijo/`, recursivamente (incluye subprefijos como `prefijo/a/b.log`).
  - `gs://bucket/ruta/objeto` (sin `/` final) — **un único objeto** cuyo nombre
    es exactamente `ruta/objeto`. No se lista nada ni se buscan otros objetos
    con ese mismo comienzo. Si ese objeto no existe, la corrida termina con
    exit code 2 y el mensaje
    `gcsgrep: object not found: gs://bucket/ruta/objeto (to search under a prefix, end the location with /)`
    por stderr. Este error se detecta al consultar GCS, no antes.

  Regla de parseo: la ubicación empieza con `gs://`; el nombre del bucket es
  todo lo que sigue hasta la primera `/` y no puede estar vacío; esa `/` es
  obligatoria; lo que viene después es la ruta (vacía para el bucket completo).
  El nombre del bucket no se valida contra las reglas de nombres de GCS: un
  nombre imposible termina como bucket inexistente (exit 2). Una ubicación que
  no cumple esta regla —sin el esquema `gs://`, sin nombre de bucket, sin `/`
  después del bucket (`gs://bucket`) o con otra sintaxis inválida— es un error de
  uso (exit code 2), detectado antes de cualquier operación contra GCS.
- **FR-c** — La salida identifica el objeto donde apareció cada match en formato
  `gs://bucket/objeto:texto`. Con `-n`, se antepone el número de línea:
  `gs://bucket/objeto:línea:texto` (la primera línea del objeto es la 1). Si el texto de la línea supera el tope de
  BR-e, se imprime truncado; el match se reporta igual.
- **FR-d** — `-i` habilita búsqueda case-insensitive, igual que `grep -i`.
- **FR-e** — Si la corrida completa no encuentra ningún match en ningún objeto,
  el exit code es 1 (ver "Exit codes" más abajo), detectable por scripts sin
  tener que parsear la salida.
- **FR-f** — Si un objeto individual no se puede leer (permisos, contenido
  corrupto, o falla la lectura de ese objeto puntual —incluida una falla de red
  acotada a ese GET—), se reporta ese objeto como fallido por stderr y la corrida
  continúa con el resto. No aborta toda la ejecución. El aviso por stderr tiene
  la forma `gcsgrep: gs://bucket/objeto: read error: <detail>`. Al terminar, si
  hubo al menos un objeto fallido, el exit code es **2** aunque otros objetos
  hayan tenido matches (convención de `grep`): la salida de los objetos que sí
  se leyeron se emite completa, pero un script puede saber que el resultado
  puede estar incompleto. (Distinto de un fallo en la fase de listado: ver
  NFR-c.)
- **FR-g** — Si stderr es una terminal interactiva (TTY), la herramienta muestra
  progreso por stderr durante la fase de lectura: una única línea con el conteo
  de objetos procesados sobre el total (por ejemplo, `42/1000 objects processed`),
  que se reescribe con `\r` cada vez que termina de procesar un objeto (con
  match, sin match, salteado o fallido). Antes de escribir un aviso por stderr
  (BR-d, FR-f) se borra la línea de progreso, para que el aviso quede en una
  línea propia; al terminar la corrida, la línea de progreso se borra. Si
  stderr **no** es una TTY (redirigido a un archivo o a un pipe), no se emite
  ningún progreso: en una corrida sin avisos ni errores, stderr queda vacío. No
  hay flag para activarlo o desactivarlo. **Nota:** el total se conoce antes de
  leer el primer objeto porque el guardrail de BR-c obliga a terminar el
  listado (hasta N+1 objetos) antes de cualquier lectura; con `--max unlimited`
  se lista todo el prefijo antes de empezar a leer.
- **FR-h** *(nuevo, de las decisiones de flags)* — `-c` imprime, en vez de cada
  línea, una línea por objeto de texto leído con la forma
  `gs://bucket/objeto:<n>`, donde `<n>` es la cantidad de **líneas** que
  matchean (igual que `grep -c`: una línea con varias ocurrencias suma 1). Los
  objetos de texto sin ningún match aparecen con `:0` (un objeto vacío también).
  Los objetos salteados por no-texto (BR-d) o fallidos (FR-f) no aparecen en
  stdout; solo dejan su aviso en stderr. El exit code sigue a FR-e: si todos los
  conteos son 0, el exit code es 1 aunque stdout no esté vacío. `-l` imprime
  solo el nombre de cada objeto que matcheó, `gs://bucket/objeto`, uno por
  línea, sin líneas ni conteos; al encontrar el primer match en un objeto se
  imprime su nombre y se deja de leer ese objeto (se cierra su stream). Con `-l`,
  si ningún objeto matchea, stdout queda vacío y el exit code es 1. `-c` y `-l`
  son mutuamente excluyentes: pasarlos juntos es
  un error de uso (exit code 2), detectado antes de cualquier operación contra
  GCS. `-n` tampoco tiene efecto ni sobre un conteo (`-c`) ni sobre un listado
  de nombres (`-l`): combinar `-n` con `-c` o con `-l` es igualmente un error de
  uso (exit code 2), detectado antes de cualquier operación contra GCS.
- **FR-i** *(nuevo, de sintaxis de invocación)* — La invocación es
  `gcsgrep [-E] [-i] [-n] [-c | -l] [--max <N>|unlimited] [--concurrency <N>] [--] <patrón> <ubicación>`.
  El patrón y la ubicación son posicionales, siempre en ese orden. Se soporta
  `--` como marcador de fin de opciones (convención POSIX/GNU): todo lo que
  aparezca después de `--` se trata como posicional aunque empiece con `-`, lo
  que permite buscar un patrón literal que empiece con guion (por ejemplo,
  `gcsgrep -- -timeout gs://logs/` busca el patrón literal `-timeout`). Sin
  `--`, un argumento que empieza con `-` y no es un flag reconocido es un error
  de uso estándar de parsing (flag desconocido, exit code 2).
- **FR-j** *(nuevo, de concurrencia)* — `--concurrency <N>` fija cuántos objetos
  se leen en paralelo; `N` es un entero entre 1 y 32, default 4. Un valor no
  entero, menor que 1 o mayor que 32 es un error de uso (exit code 2) con el
  mensaje `gcsgrep: invalid value for --concurrency: "<value>" (integer between 1 and 32)`
  por stderr, detectado antes de cualquier operación contra GCS. La salida no
  está agrupada por objeto: las líneas de objetos distintos pueden aparecer
  entrelazadas. Se garantiza que (1) las líneas de un mismo objeto aparecen en
  el orden en que están en el objeto y (2) cada línea de salida se escribe
  completa, sin mezclarse con otra. Como cada línea lleva el nombre del objeto
  como prefijo (FR-c), el entrelazado no pierde información. **Fundamento:**
  agrupar la salida de un objeto obligaría a bufferearla o a serializar los
  workers, lo que tensiona la garantía de memoria acotada (NFR-b) o el
  rendimiento (NFR-a). El tope de 32 mantiene calculable la cota de memoria.

## Reglas de negocio (decididas)

- **BR-a** — La herramienta nunca escribe en GCS. Solo lectura, siempre.
  **Fundamento:** restricción dura del enunciado de la tarea; gcsgrep es una
  herramienta de búsqueda, no de gestión. Sin excepciones.
- **BR-b** — La herramienta no amplía el acceso más allá de las credenciales de
  quien la invoca (vía ADC). Si el usuario no puede leer un bucket, `gcsgrep`
  tampoco. **Fundamento:** restricción dura del enunciado; evita que la
  herramienta se convierta en un vector de escalamiento de privilegios.
- **BR-c** — Hay un tope por defecto de **1000 objetos** a la cantidad que se
  escanean en una corrida. El conteo contra el tope es sobre **todos los
  objetos que devuelve el listado** del bucket/prefijo, sin importar si luego
  se clasifican como texto o no-texto (BR-d): esa clasificación requiere leer
  cada objeto, y el guardrail actúa **antes de leer nada**, así que solo puede
  contar lo que ya sabe por el `list`. El listado se detiene en cuanto aparece
  el objeto número N+1 (siendo N el tope vigente): no se sigue paginando para
  obtener el total. En ese caso la herramienta aborta antes de leer nada, con
  **exit code 2** y el mensaje
  `gcsgrep: more than <N> objects under <location>; use --max <N> or --max unlimited`
  por stderr. **Fundamento:** leer objetos de GCS tiene costo; un guardrail por
  cantidad de objetos es simple de explicar y de verificar. Cortar en N+1 evita
  que el propio guardrail genere costo paginando un listado que ya se sabe
  excedido. 1000 coincide con la escala ya validada por NFR-a, así el tope por
  defecto queda dentro del rango que ya se sabe que rinde bien.
  Cuando la ubicación es un objeto puntual (sin `/` final, FR-b) no hay listado:
  se busca en un solo objeto y el tope nunca se dispara.
  **Limitación conocida:** el tope acota la cantidad de objetos, no los bytes
  leídos; 1000 objetos grandes pueden implicar un volumen de lectura alto. Un
  límite por bytes (el `list` ya devuelve el tamaño de cada objeto, así que
  sería barato de calcular) queda diferido a una iteración posterior. **Excepción:** `--max <N>` con `N` entero ≥ 1, o
  `--max unlimited`, desactivan o suben el tope explícitamente. Un valor de
  `--max` igual a 0, negativo o no numérico es un error de uso (exit code 2),
  detectado antes de cualquier operación contra GCS.
- **BR-d** — Los objetos que no son de texto se saltean en vez de imprimir
  basura binaria por pantalla, con el aviso
  `gcsgrep: gs://bucket/objeto: not a text file, skipped` por stderr. Saltear un
  objeto no es un fallo: no afecta el exit code (si todos los objetos son
  no-texto, no hay matches y el exit code es 1). Se decide "es texto" con una heurística
  de sniffing sobre los **primeros 512 bytes** del objeto: si aparece al menos un
  byte nulo (`0x00`) o una secuencia que no es UTF-8 válido, el objeto se
  clasifica como no-texto. Si la muestra termina en medio de un carácter
  multibyte (los últimos 1 a 3 bytes son el prefijo válido de una secuencia
  UTF-8 que continúa después del byte 512), esos bytes no cuentan como
  inválidos. Si el objeto entero mide 512 bytes o menos, se analiza completo y
  una secuencia incompleta al final sí es inválida. Un objeto vacío es texto
  (sin líneas, sin matches). La clasificación se decide una sola vez con la
  muestra: bytes UTF-8 inválidos que aparezcan después no la cambian, no generan
  aviso, y las líneas que los contienen se buscan e imprimen tal cual, byte a
  byte. No se usa content-type ni extensión de archivo.
  **Consecuencia para `.gz`:** un objeto gzip empieza con los magic bytes
  `1f 8b`, y `0x8b` no es un byte inicial UTF-8 válido, así que cae en esta
  misma regla como no-texto; no hay una regla aparte basada en el nombre.
  **Excepción de GCS:** si el objeto tiene metadata `Content-Encoding: gzip`,
  GCS aplica *decompressive transcoding* y entrega el contenido ya
  descomprimido; gcsgrep lo sniffea y busca como cualquier otro objeto. Se
  acepta como comportamiento de GCS.
  **Fundamento:** el content-type de un objeto en GCS es metadata que el
  uploader pudo haber seteado mal o no haber seteado; sniffear el contenido real
  es la única fuente confiable. **Limitación conocida:** esta heurística solo
  reconoce UTF-8 (ASCII incluido, como subconjunto válido) como texto. Un objeto
  en otra codificación (por ejemplo, Latin-1/ISO-8859-1) es legible para un
  humano pero se clasifica igual como no-texto y se saltea. Ampliar el soporte
  de encodings queda para una iteración posterior.
- **BR-e** — Si una sola línea de un objeto supera **1 MB** de longitud, la
  búsqueda del patrón se limita a ese primer 1 MB; el resto de la línea no se
  analiza. Si hay match dentro de esa porción, se reporta como tal, y el texto
  impreso se trunca a esos mismos 1 MB con un indicador de truncamiento (`...`)
  al final. **Fundamento:** imprimir una línea requiere tenerla completa en un
  buffer transitorio (ver NFR-b); sin un tope, una línea patológicamente larga
  (por ejemplo, un log de una sola línea de cientos de MB) rompería la garantía
  de memoria acotada. Truncar en vez de descartar el objeto entero conserva la
  señal de que hubo un match, que es más útil que un fallo silencioso.

## Requerimientos no funcionales

- **NFR-a — Rendimiento.** Con concurrencia default (`--concurrency 4`),
  gcsgrep procesa 1000 objetos de texto en menos de **120 segundos**.
  **Condiciones de medición:**
  - **Cliente:** la laptop de desarrollo, contra un bucket de prueba en la región
    de GCS más cercana. Se registra el ancho de banda de bajada medido al
    momento de la prueba junto con el resultado.
  - **Dataset:** exactamente 1000 objetos de texto UTF-8 de 100 KB cada uno
    (unos 100 MB en total), con líneas de unos 100 bytes y sin objetos binarios.
    Al ser 1000, entra en el tope por defecto de BR-c sin usar `--max`.
  - **Corrida:** patrón literal que matchea alrededor del 1 % de las líneas,
    ubicación `gs://bucket/prefijo/` que abarca exactamente esos 1000 objetos,
    stdout redirigido a `/dev/null`.
  - **Qué se mide:** tiempo de pared desde la invocación hasta el exit (listado
    incluido). Pasa si la **mediana de 3 corridas** es menor a 120 s.
- **NFR-b — Memoria con objetos grandes.** El uso de memoria del proceso no crece
  con el tamaño de los objetos: cada objeto se lee por streaming (en chunks), y en
  ningún momento se carga un objeto completo en memoria. **Umbral:** corriendo
  con `--concurrency 4` contra un objeto de texto UTF-8 de 1 GB (líneas de unos
  100 bytes, con un patrón literal que matchea alrededor del 1 % de las líneas,
  y stdout redirigido a `/dev/null`), (1) el pico de RSS del proceso es
  **≤ 100 MB**, y (2) ese pico no supera en más de **20 MB** al pico medido con
  un objeto de 10 MB de la misma composición. El pico de RSS se toma del campo
  *maximum resident set size* de `/usr/bin/time -v` (Linux) o `/usr/bin/time -l`
  (macOS). Esta garantía es
  por objeto, no por línea: cada línea se acumula en un buffer transitorio para
  poder matchearla e imprimirla completa, y ese buffer se libera al pasar a la
  línea siguiente. El buffer de línea tiene un tope de 1 MB (BR-e) para que una
  línea individual de tamaño desproporcionado no rompa esta garantía.
- **NFR-c — Comportamiento ante fallos de red.** Se distingue la fase de listado
  de la fase de lectura:
  - Un fallo de red o de la API al **listar** los objetos del bucket/prefijo
    aborta toda la corrida con exit code 2, porque sin poder listar no hay nada
    que hacer. Con una ubicación de objeto puntual (FR-b), el equivalente es la
    consulta inicial de ese objeto: si falla (red, API, permisos o inexistencia),
    la corrida aborta con exit code 2.
  - Un fallo de red al **leer** un objeto individual ya listado se trata como
    FR-f: ese objeto se reporta como fallido y la corrida continúa con el resto.
    No hay reintentos automáticos en la v1 — un fallo puntual de lectura pasa
    directo a FR-f en el primer intento.
  - Ninguna de las dos fases reintenta automáticamente: tanto un fallo de
    listado como un fallo de lectura de un objeto puntual se resuelven en el
    primer intento, sin backoff ni reintentos implícitos.
  - **Timeout por inactividad:** toda request contra GCS (cada página del
    listado, la consulta de un objeto puntual y la lectura de cada objeto) se
    corta si pasan **30 segundos sin recibir datos**. El plazo se cuenta desde
    el último byte recibido, no desde el inicio de la request, así que una
    lectura lenta pero continua de un objeto grande no se corta. Si vence
    durante una lectura, se trata como FR-f con el aviso
    `gcsgrep: gs://bucket/objeto: read error: timeout after 30s without data`
    (la corrida continúa y termina con exit 2). Si vence durante el listado o la
    consulta del objeto puntual, la corrida aborta con exit code 2. El valor es
    fijo en v1; un flag para configurarlo queda diferido.
  - **Umbral verificable:** contra un servidor de prueba que acepta la conexión
    pero nunca responde, la corrida termina en **≤ 35 segundos** desde que se
    inicia la request colgada, con el exit code y el mensaje correspondientes a
    la fase (lectura o listado).

## Stack de implementación

- **Lenguaje:** Go, versión mínima 1.22.
- **Regex:** paquete `regexp` de la biblioteca estándar, que implementa RE2
  (sintaxis y garantía de tiempo lineal) sin dependencias externas. El modo
  literal por defecto no usa el motor de regex (búsqueda de subcadena), salvo
  combinado con `-i`, donde puede usarse `regexp` sobre el literal escapado.
- **Cliente de GCS:** `cloud.google.com/go/storage`, que obtiene ADC de forma
  nativa, lista con paginación y lee objetos en streaming.
- **Distribución e invocación:** binario único compilado con
  `go build -o gcsgrep ./cmd/gcsgrep`. La spec y los VCs se refieren a la
  invocación `./gcsgrep [flags] [--] <patrón> <ubicación>`.
- **Fundamento:** RE2 nativo evita bindings con otras garantías y mensajes de
  error; un binario Go con este cliente tiene un RSS base compatible con el
  umbral de NFR-b (≤ 100 MB).

## Preguntas abiertas — resueltas

1. **¿Qué sabor de expresiones regulares?** → El patrón es literal por
   defecto; con `-E` se interpreta como RE2. RE2: sintaxis moderna, sin
   backtracking catastrófico, segura incluso con patrones no confiables. Un
   patrón RE2 inválido (solo posible con `-E`) es error de uso, exit 2, con el
   mensaje `gcsgrep: invalid pattern: <engine detail>`.
2. **¿Cómo se autentica?** → Solo Application Default Credentials (ADC) en v1.
   Sin credenciales, la herramienta falla con un error claro antes de intentar
   nada, con exit code 2. Un flag para archivo de service account explícito
   queda para una iteración posterior.
3. **¿Cómo se escribe la ubicación?** → Esquema `gs://` obligatorio y `/`
   obligatoria después del bucket. Con `/` final (`gs://bucket/` o
   `gs://bucket/prefijo/`) se busca recursivamente bajo ese prefijo; sin `/`
   final (`gs://bucket/objeto`) se busca en un único objeto con ese nombre
   exacto, y si no existe es exit 2 (ver FR-b). `gs://bucket` sin barra, o
   cualquier otro formato inválido, es error de uso, exit code 2, detectado
   antes de cualquier operación contra GCS.
4. **¿Qué flags de `grep` en la v1?** → `-E`, `-i`, `-n`, `-c`, `-l`. `-c` y `-l` son
   mutuamente excluyentes entre sí, y `-n` es incompatible con ambos (error de
   uso, exit 2, si se combinan). `-v`, `-r`, `--include` quedan diferidos.
5. **¿Qué se hace con binarios y `.gz`?** → Los binarios se saltean (sniffing de
   los primeros 512 bytes: byte nulo o secuencia no-UTF8 válida, BR-d). Solo se
   reconoce UTF-8 como texto; otras codificaciones (por ejemplo, Latin-1) se
   tratan como no-texto en v1. Los `.gz` también se saltean en v1, como
   consecuencia del mismo sniffing (magic bytes `1f 8b`), no por la extensión;
   descomprimir al vuelo queda para una iteración posterior. Excepción: los
   objetos con `Content-Encoding: gzip` los entrega GCS ya descomprimidos y se
   buscan como texto (BR-d).
6. **¿Qué guardrails de costo?** → Tope por defecto de 1000 objetos a escanear,
   con override explícito vía `--max <N≥1>` o `--max unlimited` (BR-c). Un
   `--max` igual a 0, negativo o no numérico es un error de uso (exit 2).
7. **¿Cómo es la salida?** → `gs://bucket/objeto:texto` por defecto, y
   `gs://bucket/objeto:línea:texto` con `-n`; texto plano, sin color, sin JSON
   en v1 (FR-c). Las líneas se delimitan por `\n` (también se
   reconoce `\r\n`, recortando el `\r`). Una línea que supera 1 MB se trunca en
   la salida, pero el match se reporta igual (BR-e).
8. **¿Qué exit codes?** → Convención de `grep`: 0 si hubo match, 1 si no hubo
   match, 2 si hubo error de corrida o de uso: fallo de listado, credenciales
   ausentes, bucket inexistente, ubicación con formato inválido (FR-b), patrón
   regex inválido con `-E` (FR-a), guardrail de `--max` disparado (BR-c), combinación
   inválida de flags (`-c`+`-l` o `-n` con cualquiera de los dos, FR-h),
   `--max` mal formado (BR-c), `--concurrency` fuera de rango (FR-j), o un flag desconocido sin usar `--` (FR-i).
   Un fallo de lectura de un objeto individual no aborta la corrida, pero
   hace que el exit code final sea 2 aunque haya habido matches (FR-f). Un
   objeto salteado por no-texto no es un fallo (BR-d).
   Un bucket o prefijo (`.../`) que existe pero no contiene ningún objeto es una
   corrida válida sin resultados (exit 1), distinto de un bucket inexistente o
   inaccesible (exit 2). En cambio, un objeto puntual (ubicación sin `/` final)
   que no existe es exit 2 (FR-b).
9. **¿Concurrencia?** → `--concurrency <N>` con `N` entre 1 y 32, default 4
   (FR-j). Las líneas de objetos distintos pueden salir entrelazadas; dentro de
   un objeto se respeta el orden, y ninguna línea se parte.
10. **¿Qué pasa si un objeto cambia mientras se lee?** → No se controla en v1: se
    lee lo que haya en el momento de abrir el stream, sin fijar ni verificar la
    generación del objeto.

## Cómo seguir

Este documento ya no tiene preguntas abiertas. El camino que sigue:

1. ~~Refinar este documento con el agente hasta que no queden preguntas
   abiertas.~~ ✅ Hecho — este documento.
2. Convertirlo en una spec: propósito, alcance, actores, FRs atómicos en
   Dado/Cuando/Entonces, BRs con fundamento, NFRs con umbral, y **un VC por cada
   FR y BR**.
3. Pasar la spec por una revisión que la habilite.
4. Partirla en un plan de iteraciones (`gcsgrep-plan.md`, ≥ 2 iteraciones).
5. Implementar y verificar la Iteración 1.

La consigna completa está en [`enunciado.md`](./enunciado.md).
