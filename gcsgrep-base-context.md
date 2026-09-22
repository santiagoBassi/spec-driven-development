# gcsgrep — requerimientos (base context refinado)

> **Estado: base context refinado.** Este documento es la evolución de
> [`gcsgrep-requirements.md`](./gcsgrep-requirements.md): mismas secciones, pero con
> las 10 preguntas abiertas resueltas, las reglas de negocio candidatas decididas
> (con fundamento) y los NFRs con umbrales concretos.
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

## Decisiones de alcance para la Iteración 1

Resumen ejecutivo de lo que se decidió. El detalle y el fundamento de cada punto
está en las secciones de abajo y en "Preguntas abiertas — resueltas".

| Dimensión | Decisión |
|---|---|
| Sabor de regex | RE2 |
| Patrón inválido | Regex RE2 mal formada → error de uso, exit 2, detectado antes de cualquier operación contra GCS |
| Autenticación | Solo ADC (Application Default Credentials); sin credenciales → error de uso/config, exit 2, detectado antes de cualquier operación contra GCS |
| Sintaxis de ubicación | Solo `gs://bucket/prefijo`, esquema obligatorio |
| Ubicación inválida | Falta el esquema `gs://`, falta el bucket, o URI malformada → error de uso, exit 2, detectado antes de cualquier operación contra GCS |
| Formato de salida | `objeto:línea:texto`, texto plano, sin color, sin JSON |
| Sintaxis de invocación | `gcsgrep [flags] [--] <patrón> <ubicación>`, posicionales en orden fijo; `--` marca fin de opciones (convención POSIX) |
| Flags de `grep` soportados | `-i`, `-n`, `-c`, `-l` (`-c` y `-l` mutuamente excluyentes; `-n` incompatible con ambos) |
| Binarios | Se saltean (sniffing de los primeros 512 bytes: NUL o UTF-8 inválido) |
| Archivos `.gz` | Se saltean en v1 |
| Exit codes | Convención `grep`: 0 match / 1 sin match / 2 error. Bucket inexistente, credenciales ausentes, fallo de listado, guardrail de `--max` disparado, o cualquier error de uso → 2; bucket/prefijo sin objetos que matcheen → 1 |
| Concurrencia | Configurable, default 4 workers, salida no ordenada entre objetos |
| Guardrail de costo | Tope por defecto de 1000 objetos **listados** (sin importar si después se clasifican como texto o no); override con `--max <N≥1>` o `--max unlimited`. Si se dispara, exit 2 |
| Detección de "es texto" | Sniffing de los primeros 512 bytes, no content-type ni extensión. Solo UTF-8 se reconoce como texto |
| Objeto que cambia durante la lectura | No se controla en v1 |
| Progreso | Siempre activo; una actualización por objeto procesado, por stderr |
| Línea que excede 1 MB | Se trunca el texto impreso; el match igual se reporta |
| Separador de línea | `\n` y `\r\n` (se recorta el `\r` final) |
| Reintentos automáticos | Ninguno, ni en listado ni en lectura de objetos |

## Requerimientos funcionales (refinados)

Siguen sin estar en formato Dado/Cuando/Entonces —eso es tarea de la spec— pero
ya no dejan comportamiento ambiguo ni mezclan varias conductas en un mismo ítem.

- **FR-a** — El usuario pasa un patrón (literal o regex RE2) y una ubicación
  `gs://bucket/prefijo`, y la herramienta busca el patrón línea por línea dentro
  del contenido de cada objeto de texto bajo esa ubicación. Una línea se delimita
  por `\n`; si está precedido por `\r` (`\r\n`), ese `\r` se recorta antes de
  matchear y de imprimir. Si el patrón no es una expresión regular RE2 válida, es
  un error de uso (exit code 2), detectado antes de cualquier operación contra
  GCS (no se intenta autenticar, listar ni leer nada).
- **FR-b** — Se puede buscar sobre todo un bucket (`gs://bucket/`) o sobre un
  prefijo dentro del bucket (`gs://bucket/prefijo`). Un prefijo sin `/` final se
  trata como filtro de nombre: matchea cualquier nombre de objeto que empiece con
  esa cadena, no solo los que están "dentro" de esa carpeta virtual. Una ubicación
  que no cumple este formato —sin el esquema `gs://`, sin nombre de bucket, o con
  sintaxis inválida— es un error de uso (exit code 2), detectado antes de
  cualquier operación contra GCS.
- **FR-c** — La salida identifica el objeto donde apareció cada match en formato
  `gs://bucket/objeto:texto`. Con `-n`, se antepone el número de línea:
  `gs://bucket/objeto:línea:texto`. Si el texto de la línea supera el tope de
  BR-e, se imprime truncado; el match se reporta igual.
- **FR-d** — `-i` habilita búsqueda case-insensitive, igual que `grep -i`.
- **FR-e** — Si la corrida completa no encuentra ningún match en ningún objeto,
  el exit code es 1 (ver "Exit codes" más abajo), detectable por scripts sin
  tener que parsear la salida.
- **FR-f** — Si un objeto individual no se puede leer (permisos, contenido
  corrupto, o falla la lectura de ese objeto puntual —incluida una falla de red
  acotada a ese GET—), se reporta ese objeto como fallido por stderr y la corrida
  continúa con el resto. No aborta toda la ejecución. (Distinto de un fallo en la
  fase de listado: ver NFR-c.)
- **FR-g** — La herramienta muestra progreso por stderr durante toda la corrida,
  sin importar la cantidad de objetos: cada vez que termina de procesar un
  objeto (con match o sin él), emite una actualización con el conteo de objetos
  procesados sobre el total (por ejemplo, `42/1000 objetos procesados`).
- **FR-h** *(nuevo, de las decisiones de flags)* — `-c` cuenta matches por objeto
  en vez de imprimir cada línea; los objetos sin ningún match no aparecen en esa
  salida. `-l` imprime solo los nombres de los objetos que matchearon, sin
  líneas ni conteos. `-c` y `-l` son mutuamente excluyentes: pasarlos juntos es
  un error de uso (exit code 2), detectado antes de cualquier operación contra
  GCS. `-n` tampoco tiene efecto ni sobre un conteo (`-c`) ni sobre un listado
  de nombres (`-l`): combinar `-n` con `-c` o con `-l` es igualmente un error de
  uso (exit code 2), detectado antes de cualquier operación contra GCS.
- **FR-i** *(nuevo, de sintaxis de invocación)* — La invocación es
  `gcsgrep [-i] [-n] [-c | -l] [--max <N>|unlimited] [--] <patrón> <ubicación>`.
  El patrón y la ubicación son posicionales, siempre en ese orden. Se soporta
  `--` como marcador de fin de opciones (convención POSIX/GNU): todo lo que
  aparezca después de `--` se trata como posicional aunque empiece con `-`, lo
  que permite buscar un patrón literal que empiece con guion (por ejemplo,
  `gcsgrep -- -timeout gs://logs/` busca el patrón literal `-timeout`). Sin
  `--`, un argumento que empieza con `-` y no es un flag reconocido es un error
  de uso estándar de parsing (flag desconocido, exit code 2).

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
  contar lo que ya sabe por el `list`. Si el prefijo tiene más objetos que el
  tope, la herramienta aborta antes de leer nada, con **exit code 2**, con un
  mensaje explicando cuántos objetos hay y cómo levantar el límite con
  `--max`. **Fundamento:** leer objetos
  de GCS tiene costo; un guardrail por cantidad de objetos es más fácil de
  verificar de antemano (viene de un solo `list`) que uno por bytes totales, que
  requeriría sumar tamaños antes de decidir. 1000 coincide con la escala ya
  validada por NFR-a, así el tope por defecto queda dentro del rango que ya se
  sabe que rinde bien. **Excepción:** `--max <N>` con `N` entero ≥ 1, o
  `--max unlimited`, desactivan o suben el tope explícitamente. Un valor de
  `--max` igual a 0, negativo o no numérico es un error de uso (exit code 2),
  detectado antes de cualquier operación contra GCS.
- **BR-d** — Los objetos que no son de texto se saltean (con aviso) en vez de
  imprimir basura binaria por pantalla. Se decide "es texto" con una heurística
  de sniffing sobre los **primeros 512 bytes** del objeto: si aparece al menos un
  byte nulo (`0x00`) o una secuencia que no es UTF-8 válido, el objeto se
  clasifica como no-texto. No se usa content-type ni extensión de archivo.
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

- **NFR-a — Rendimiento.** Con concurrencia default (4 workers), contra un bucket
  de prueba en la misma región que el cliente, gcsgrep procesa 1000 objetos de
  menos de 1 MB cada uno en menos de 60 segundos.
- **NFR-b — Memoria con objetos grandes.** El uso de memoria del proceso no crece
  con el tamaño de los objetos: cada objeto se lee por streaming (en chunks), y en
  ningún momento se carga un objeto completo en memoria. Verificable corriendo
  contra un objeto de prueba de al menos 1 GB y observando que el RSS del proceso
  se mantiene acotado independientemente del tamaño del objeto. Esta garantía es
  por objeto, no por línea: cada línea se acumula en un buffer transitorio para
  poder matchearla e imprimirla completa, y ese buffer se libera al pasar a la
  línea siguiente. El buffer de línea tiene un tope de 1 MB (BR-e) para que una
  línea individual de tamaño desproporcionado no rompa esta garantía.
- **NFR-c — Comportamiento ante fallos de red.** Se distingue la fase de listado
  de la fase de lectura:
  - Un fallo de red o de la API al **listar** los objetos del bucket/prefijo
    aborta toda la corrida con exit code 2, porque sin poder listar no hay nada
    que hacer.
  - Un fallo de red al **leer** un objeto individual ya listado se trata como
    FR-f: ese objeto se reporta como fallido y la corrida continúa con el resto.
    No hay reintentos automáticos en la v1 — un fallo puntual de lectura pasa
    directo a FR-f en el primer intento.
  - Ninguna de las dos fases reintenta automáticamente: tanto un fallo de
    listado como un fallo de lectura de un objeto puntual se resuelven en el
    primer intento, sin backoff ni reintentos implícitos.

## Preguntas abiertas — resueltas

1. **¿Qué sabor de expresiones regulares?** → RE2. Sintaxis moderna, sin
   backtracking catastrófico, segura incluso con patrones no confiables.
2. **¿Cómo se autentica?** → Solo Application Default Credentials (ADC) en v1.
   Sin credenciales, la herramienta falla con un error claro antes de intentar
   nada, con exit code 2. Un flag para archivo de service account explícito
   queda para una iteración posterior.
3. **¿Cómo se escribe la ubicación?** → Solo `gs://bucket/prefijo`, esquema
   obligatorio. Un prefijo sin `/` final es un filtro de prefijo de nombre, no
   una carpeta (ver FR-b). Una ubicación con formato inválido es un error de
   uso, exit code 2, detectado antes de cualquier operación contra GCS.
4. **¿Qué flags de `grep` en la v1?** → `-i`, `-n`, `-c`, `-l`. `-c` y `-l` son
   mutuamente excluyentes entre sí, y `-n` es incompatible con ambos (error de
   uso, exit 2, si se combinan). `-v`, `-r`, `--include` quedan diferidos.
5. **¿Qué se hace con binarios y `.gz`?** → Los binarios se saltean (sniffing de
   los primeros 512 bytes: byte nulo o secuencia no-UTF8 válida, BR-d). Solo se
   reconoce UTF-8 como texto; otras codificaciones (por ejemplo, Latin-1) se
   tratan como no-texto en v1. Los `.gz` también se saltean en v1; descomprimir
   al vuelo queda para una iteración posterior.
6. **¿Qué guardrails de costo?** → Tope por defecto de 1000 objetos a escanear,
   con override explícito vía `--max <N≥1>` o `--max unlimited` (BR-c). Un
   `--max` igual a 0, negativo o no numérico es un error de uso (exit 2).
7. **¿Cómo es la salida?** → `gs://bucket/objeto:línea:texto` en texto plano, sin
   color, sin JSON en v1 (FR-c). Las líneas se delimitan por `\n` (también se
   reconoce `\r\n`, recortando el `\r`). Una línea que supera 1 MB se trunca en
   la salida, pero el match se reporta igual (BR-e).
8. **¿Qué exit codes?** → Convención de `grep`: 0 si hubo match, 1 si no hubo
   match, 2 si hubo error de corrida o de uso: fallo de listado, credenciales
   ausentes, bucket inexistente, ubicación con formato inválido (FR-b), patrón
   regex inválido (FR-a), guardrail de `--max` disparado (BR-c), combinación
   inválida de flags (`-c`+`-l` o `-n` con cualquiera de los dos, FR-h),
   `--max` mal formado (BR-c), o un flag desconocido sin usar `--` (FR-i).
   Fallos de objetos individuales no cuentan como error de corrida (ver FR-f).
   Un bucket o prefijo que existe pero no tiene ningún objeto que matchee el
   filtro de FR-b es una corrida válida sin resultados (exit 1), distinto de un
   bucket inexistente o inaccesible (exit 2).
9. **¿Concurrencia?** → Configurable vía flag, default 4 workers. La salida no
   está ordenada entre objetos distintos (se emite a medida que cada uno termina),
   pero las líneas de un mismo objeto siempre salen juntas y en orden.
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
