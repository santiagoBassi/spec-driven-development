# gcsgrep — base context

> **Qué es esto.** La salida de la fase *previa* al pipeline: requerimientos
> refinados, notas de diseño y esquema de arquitectura. **No es la spec.** Es la
> materia prima que consume el paso Especificar.
>
> Parte del borrador [`gcsgrep-requirements.md`](../gcsgrep-requirements.md) y
> responde sus preguntas abiertas. Los mensajes exactos, los umbrales medibles y los
> VCs viven en [`gcsgrep-spec.md`](./gcsgrep-spec.md), no acá.

## La idea vaga con la que empezó

> "`grep`, pero sobre Google Cloud Storage."

```
gcsgrep "timeout" gs://logs/
```

Hoy, para buscar texto dentro de objetos de un bucket, hay que bajarlos con
`gsutil cp` y recién ahí correr `grep`: es lento, gasta ancho de banda y llena el
disco. Lo que sigue es lo que hubo que decidir para que esa idea fuera construible.

---

## 1 · Requerimientos refinados

**¿Qué problema resuelve?**
Buscar dentro del **contenido** de objetos remotos sin bajarlos a disco, y saber
qué objeto matcheó y en qué línea.

**¿Para quién?**
Gente de desarrollo y operaciones que usa `grep` a diario y ya tiene credenciales
de GCP configuradas. No es para usuarios finales.

**¿Quién lo ejecuta?**
Una persona desde una shell, y también scripts. Por eso los exit codes tienen que
decir lo mismo que en `grep`: el script decide por el código, no por el texto.

**¿Qué entra en la v1?**
Búsqueda línea por línea, literal o regex, sobre un bucket, un prefijo o un objeto
puntual. Flags `-E`, `-i`, `-n`, `-c` y `-l`, más `--max` (guardrail) y
`--concurrency`.

La v1 sugerida era `-i` y `-n`. Lo que se agrega tiene un porqué propio:

- **`-c`** responde "¿cuántas veces aparece en cada objeto?", la pregunta típica sobre
  un bucket de logs. Sin `-c`, un script tendría que contar a partir de la salida
  normal, y eso es frágil: el nombre del objeto ya trae `:` (`gs://…`), así que
  `cut -d: -f1 | uniq -c` corta mal. Sale del mismo lector de líneas, sin costo de
  lectura extra.
- **`-l`** responde "¿en qué objetos aparece?", y además **baja el costo**: deja de
  leer cada objeto en su primer match. Con objetos grandes, es la diferencia entre leer
  unos pocos KB y leer el objeto entero. Va en la misma línea que el guardrail.
- **`--concurrency`** configurable: ver "Concurrencia y orden de la salida" en §2.
- **Parser estricto** (flags combinados, flags repetidos, un solo error de uso): ver
  "Parser" en §2.

**¿Qué queda fuera?**
- No es un clon de `grep`: `-v`, `-r`, `--include` y contexto quedan diferidos.
- No es un gestor de GCS: no copia, no mueve, no borra, no cambia permisos.
- Solo CLI y solo GCS. S3 y Azure quedan afuera, sin cerrarles la puerta.
- Sin JSON, sin color, sin descompresión de `.gz` ni conversión de codificaciones
  distintas de UTF-8.

**Restricciones**
- **Solo lectura**, siempre.
- **Nunca más acceso que el de quien invoca.** Si esa persona no puede leer un
  bucket, `gcsgrep` tampoco.
- **No escanear un bucket enorme sin un guardrail de costo.** Leer de GCS se cobra.
- Memoria acotada: un objeto de varios GB no puede cargarse entero.

**¿Qué pasa con entradas inválidas?**
Patrón mal formado, ubicación sin `gs://`, flags incompatibles o valores fuera de
rango: todo eso es un error de uso (exit 2). Se detecta **antes** de tocar GCS, así
un typo no cuesta una llamada.

**¿Y con los estados límite?**
- Bucket o prefijo vacío: no es error, es "sin matches" (exit 1).
- Bucket inexistente, objeto puntual inexistente o sin credenciales: error (exit 2).
- Un objeto ilegible no tira abajo la corrida: se avisa, se sigue con el resto y el
  exit final es 2, para que un script sepa que el resultado puede estar incompleto.
- Objeto clasificado como no-texto: se saltea con un aviso, no cuenta como fallo.
- Última línea sin `\n` final: es una línea como cualquier otra. Se busca y, si
  matchea, se imprime terminada en `\n`, como el resto de la salida.
- Línea gigante (un log de una sola línea de cientos de MB): se busca en toda la
  línea, pero se conserva para imprimir solo su primer MiB. Un match posterior a
  ese límite también se reporta, aunque el texto que matcheó no sea visible en
  la salida truncada.
- Un objeto que cambia mientras se lee: no se controla en la v1. Cada lectura es un
  único request sin reintentos, y GCS sirve una sola generación del objeto por
  request, así que lo leído nunca mezcla dos versiones. Lo único que puede pasar es
  que entre el listado y la lectura el objeto se reemplace (se busca en la versión
  nueva, un resultado válido del momento en que se leyó) o se borre (la lectura falla
  y se trata como objeto ilegible). Las alternativas no le dan nada útil a quien busca:
  fijar la generación listada convierte un reemplazo en un error, abortar tira la
  corrida entera por un caso raro, y avisar agrega un mensaje (y una comparación de
  generaciones) para algo que no cambia qué hacer con el resultado.

---

## 2 · Notas de diseño

### Interpretación del patrón

**Elegido: literal por defecto; con `-E`, regex RE2.**

Fundamento: el caso más común es buscar un string tal cual (`timeout`, un ID, una
IP), y en modo regex `a.b` o `1.2.3.4` matchean más de lo que uno cree. RE2 no tiene
backtracking catastrófico, así que un patrón no puede colgar la herramienta.

| Opción | Por qué no |
|---|---|
| Siempre regex | Obliga a escapar literales comunes y sorprende al que viene de `grep -F`. |
| PCRE / regex del lenguaje con backtracking | Tiempo exponencial con patrones patológicos. |

### Autenticación

**Elegido: solo Application Default Credentials (ADC).**

Fundamento: el público ya tiene `gcloud` configurado, y ADC respeta exactamente la
identidad de quien invoca (restricción de no ampliar acceso). Sin credenciales, error
claro antes de intentar nada.

| Opción | Por qué no |
|---|---|
| Archivo de service account por flag | Una segunda forma de identidad que verificar. ADC ya acepta una clave con `GOOGLE_APPLICATION_CREDENTIALS`. Diferido. |
| Credenciales propias de la herramienta (config o variables nuevas) | Otra fuente de identidad que puede no coincidir con la de `gcloud`. Abre la puerta a leer con más acceso que quien invoca. |
| Acceso anónimo si no hay credenciales | Un bucket público se leería sin saber con qué identidad. Y el error de "sin credenciales" quedaría escondido detrás de un `403`. |

Diferido: un flag para pasar el archivo de una service account.

### Ubicación

**Elegido: `gs://` obligatorio, y la `/` final decide el modo.**

```
gs://bucket/            todo el bucket, recursivo
gs://bucket/prefijo/    todo lo que está bajo el prefijo, recursivo
gs://bucket/objeto      un único objeto con ese nombre exacto
```

Fundamento: GCS no tiene carpetas, solo nombres con `/`. Sin una regla explícita,
`gs://logs/app` podría significar un objeto o un prefijo, y `app` también matchearía
`app-old/`. La `/` final elimina la ambigüedad.

Descartado: aceptar `bucket/prefijo` sin esquema (se confunde con una ruta local) y
`gs://bucket` sin barra (se rechaza para que la regla no tenga excepciones).

### Salida

**Elegido: el formato de `grep` con varios archivos.** `gs://bucket/objeto:texto`,
con `-n` agrega el número de línea. `-c` imprime un conteo por objeto y `-l` solo los
nombres. Texto plano.

Fundamento: quien lo usa ya sabe leerlo y ya tiene `cut`, `awk` y `sort` para
procesarlo. JSON y color son diferidos.

| Opción | Por qué no |
|---|---|
| JSON | Nadie lo pidió, y obliga a `jq` para el uso más común. Diferido. |
| Color | Ensucia la salida cuando la consume un script. |
| Número de línea siempre, sin `-n` | Rompe lo que espera quien viene de `grep`. |
| Nombre del objeto sin `gs://bucket/` | Con varias ubicaciones posibles, el nombre solo es ambiguo. La URI completa se puede pegar tal cual en otro comando. |

### Exit codes

**Elegido: la convención de `grep`.**

| Código | Significado |
|---|---|
| `0` | Hubo al menos un match |
| `1` | No hubo ningún match (incluye un prefijo vacío o solo objetos clasificados como no-texto) |
| `2` | Error: de uso, de credenciales, de listado, guardrail disparado, o algún objeto que falló al leerse |

Trade-off aceptado: si un objeto falla y otros matchean, el exit es 2 aunque haya
salida útil. Preferimos que un script no confunda un resultado parcial con uno
completo.

| Opción | Por qué no |
|---|---|
| Exit `0` si hubo algún match, aunque fallen objetos | Un script tomaría un resultado incompleto por completo. |
| Un código distinto por tipo de error (`3`, `4`, …) | Rompe la convención de `grep` que ya esperan los scripts. El tipo de error está en el mensaje. |
| Exit `2` para "sin matches" | "No encontré nada" es una respuesta, no un error. `grep` la distingue con `1`. |

### Parser

**Elegido: cada flag en su propio argumento, el valor en el argumento siguiente, sin
flags repetidos, y un solo error de uso por corrida.**

Fundamento: la sintaxis la define la spec, no una librería. Un parser que no acepta
variantes deja una sola forma de escribir cada invocación, y así un script no se
equivoca en silencio. `--max 5 --max 10` casi siempre es un comando armado por partes
con un error: tomar uno de los dos valores lo esconde. El mensaje de los flags
combinados indica la forma correcta, así que rechazarlos cuesta un espacio de más al
escribir. Un solo error por corrida deja stderr en una sola línea que empieza con
`gcsgrep: `, fácil de reconocer para un script.

| Opción | Por qué no |
|---|---|
| Aceptar flags combinados como `grep` (`-in`) | Hay que definir casos como `-Ec` o un valor pegado (`-m5`) solo para ahorrar un espacio. |
| Flags repetidos: gana el último | Esconde el error de un script que arma la línea de comandos por partes. |
| El paquete `flag` de Go | Acepta `-max` y `--max`, `--max=5` y `--max 5`, y no rechaza repetidos: la sintaxis quedaría definida por la librería. |
| Informar todos los errores de uso juntos | Mensajes más largos para un caso raro. Con uno alcanza para corregir y volver a probar. |

### Binarios y `.gz`

**Elegido: decidir "es texto" mirando los primeros 512 bytes del contenido.** Un
byte nulo o UTF-8 inválido → no es texto y se saltea.

Fundamento: el content-type y la extensión son metadata que quien subió el objeto
pudo poner mal o no poner. El contenido es la única fuente confiable.

Consecuencias aceptadas:
- Un gzip empieza con `1f 8b`, que no es UTF-8 válido: cae en la misma regla y se
  saltea sin una regla aparte por extensión.
- Si el objeto tiene `Content-Encoding: gzip`, GCS lo sirve ya descomprimido y se
  busca como texto. Es comportamiento de GCS y lo aceptamos.
- Un archivo en Latin-1 con bytes no válidos en UTF-8 dentro de la muestra se
  saltea; si aparecen solo después, se procesa sin convertir esos bytes.

| Opción | Por qué no |
|---|---|
| Por extensión o content-type | Poco confiable (ver fundamento). |
| Descomprimir `.gz` al vuelo | Útil, pero suma alcance. Queda para una iteración posterior. |

### Guardrail de costo

**Elegido: tope por defecto de 1000 objetos listados, con override `--max <N>` o
`--max unlimited`.** Si el listado supera el tope, se aborta antes de leer nada.

Fundamento: contar objetos es simple de explicar y de verificar, y el tope actúa
antes de gastar en lecturas. El listado se corta al ver el objeto N+1: contar el
total exacto obligaría a seguir paginando, y eso ya genera costo.

| Opción | Por qué no |
|---|---|
| Pedir confirmación interactiva | Rompe el uso desde scripts. |
| Solo un límite por bytes | Más preciso para el costo, pero más difícil de explicar. Queda diferido como complemento. |

Limitación aceptada: 1000 objetos grandes pueden ser mucho volumen. El tope acota
objetos, no bytes.

### Concurrencia y orden de la salida

**Elegido: `--concurrency <N>` entre 1 y 32, default 4. Las líneas de objetos
distintos pueden entrelazarse.** Dentro de un objeto se respeta el orden, y una línea
nunca se parte.

Fundamento: leer de a un objeto es lento. Agrupar la salida por objeto obligaría a
bufferearla (rompe la memoria acotada) o a serializar los workers (rompe el
rendimiento). Como cada línea lleva el nombre del objeto, el entrelazado no pierde
información.

**Por qué es configurable.** El valor bueno depende de la red y de los objetos. Con
muchos objetos chicos manda la latencia, y más lecturas en paralelo rinden más. Con
una conexión lenta, más lecturas solo se reparten el mismo ancho de banda. 4 es el
default con el que se fija el umbral de rendimiento de la spec. El flag permite
subirlo en una red rápida, o bajarlo a 1 para tener la salida en el orden del listado
(así se verifica el orden de la salida) o para no saturar un enlace.

**Por qué entre 1 y 32.** Cada lectura reserva alrededor de 1 MiB para retener la línea
que imprime (ver "Línea gigante" en §1). Con 32 lecturas son unos 33 MiB, dentro de
los 100 MiB del umbral de memoria de la spec y con margen para el runtime y el cliente
HTTP. Sin techo, un typo (`--concurrency 400`) abriría cientos de conexiones y
retendría cientos de MiB.

| Opción | Por qué no |
|---|---|
| Secuencial, una lectura por vez | Con ~0,4 s de latencia por objeto, 1000 objetos tardan más de 6 minutos. El umbral de rendimiento pide menos de 180 s. |
| Concurrencia fija en 4, sin flag | No se adapta a la red: en un enlace rápido desperdicia rendimiento, y no hay forma de pedir la salida en orden. |
| Sin techo para N | Un valor alto por error dispara conexiones y memoria (ver arriba). |
| Salida agrupada por objeto | Obliga a bufferear (rompe la memoria acotada) o a serializar (rompe el rendimiento). |

### Progreso

**Elegido: una línea `X/total` en stderr, solo si stderr es una terminal.**

Fundamento: con muchos objetos, si no se ve avance, parece colgado. Pero en un
pipe o un archivo el progreso es ruido, y ensucia la stderr que miran los scripts.
El total se conoce porque el guardrail obliga a terminar el listado antes de leer.

| Opción | Por qué no |
|---|---|
| Progreso siempre en stderr | Ensucia la stderr que leen los scripts, donde un aviso tiene que ser una línea de `gcsgrep: `. |
| Un flag `--progress` | Una opción más para algo que se decide solo: si hay una persona mirando la terminal. |
| Porcentaje por bytes | Un objeto salteado, o cortado en el primer match con `-l`, no se lee entero, y la barra mentiría. Contar objetos es exacto. |
| Una línea nueva por objeto | Con 1000 objetos llena la terminal y tapa los avisos. |

### Fallos de red

**Elegido: sin reintentos; timeout fijo por inactividad de 30 s.**

Fundamento: los reintentos agregan comportamiento difícil de verificar. El timeout
cuenta desde el último byte recibido, así una lectura lenta pero continua de un
objeto grande no se corta. Si falla el listado, la corrida aborta; si falla la
lectura de un objeto, se trata como objeto ilegible y se sigue.

Si la lectura de un objeto se corta a mitad, las líneas que ya se imprimieron quedan,
y el objeto se informa como fallido. Retirar lo impreso obligaría a bufferear cada
objeto entero, y el exit 2 ya le avisa al script que el resultado está incompleto.

| Opción | Por qué no |
|---|---|
| Reintentos con backoff | Son difíciles de verificar, pueden duplicar el costo de lectura, y un reintento a mitad de un objeto puede repetir líneas ya impresas. Diferido. |
| Timeout total por request | Corta la lectura lenta pero sana de un objeto grande. |
| Sin timeout | Un GCS colgado cuelga la herramienta para siempre. |
| Abortar toda la corrida si falla un objeto | El borrador pide que un objeto ilegible no tire abajo la corrida. |

Diferido: reintentos con backoff y un timeout configurable.

### Stack

**Elegido: Go ≥ 1.22, `regexp` de la stdlib y `cloud.google.com/go/storage`, en un
binario único.**

Fundamento: `regexp` es RE2 nativo, sin bindings. El cliente oficial trae ADC,
listado paginado y lectura por streaming. Un binario Go tiene un consumo base de
memoria bajo, compatible con la garantía de memoria acotada.

| Opción | Por qué no |
|---|---|
| Envolver `gcloud storage cat` con `grep` | Un proceso por objeto, sin control de la concurrencia, de los timeouts ni del exit code por objeto. Depende de que `gcloud` esté instalado. |
| Python con `google-cloud-storage` | Necesita intérprete y dependencias en la máquina, y su `re` hace backtracking (tiempo exponencial con patrones patológicos). |
| JVM (Java o Kotlin) | Arranque lento y consumo base de memoria alto para un CLI que se invoca muchas veces. |

---

## 3 · Esquema de arquitectura

### Módulos

```
cli       → parsea argv, valida flags/patrón/ubicación, traduce errores a exit codes
location  → interpreta gs://bucket/[ruta] y decide el modo (bucket, prefijo, objeto)
gcs       → autenticación ADC, listar con tope, consultar y leer objetos en streaming
sniff     → clasifica texto / no-texto con los primeros 512 bytes
search    → delimita líneas en el stream, busca en cada una completa y retiene
            como máximo su primer MiB para la salida
output    → escribe registros completos a stdout, avisos y progreso a stderr
```

Dependencias: `cli` orquesta; `search` y `sniff` no saben que existe GCS (reciben
un stream de bytes), y `gcs` no sabe qué es un match. Eso permite probar la búsqueda
sin red y la parte de GCS contra un emulador.

### Flujo de datos

```
argv → cli (valida, sin tocar GCS)
         ↓
       location → gcs: ADC → listar hasta N+1 (o consultar el objeto puntual)
         ↓                        ↓ supera el tope → exit 2
       pool de N workers, un objeto por vez cada uno:
         gcs (stream) → sniff → search → output
         ↓
       cli: resume matches / fallos → exit code
```

### Límites entre componentes

- **Validar vs. operar.** Todo lo que se puede decidir con argv se decide antes de
  autenticar. La primera llamada a GCS pasa recién cuando la invocación es válida.
- **Listar vs. leer.** Son fases separadas: el guardrail y el total del progreso
  dependen de que el listado termine antes de la primera lectura.
- **Streaming.** Ningún módulo tiene un objeto completo en memoria. Cada línea se
  busca completa sin retenerla entera; para la salida se conserva como máximo
  su primer MiB.

### Actores

| Actor | Interacción |
|---|---|
| Persona usuaria | Ejecuta el comando y lee la salida |
| Script | Ejecuta el comando y decide por el **exit code** |
| GCS | Lista y sirve objetos; puede fallar, colgarse o negar acceso |
| Credenciales ADC | Definen qué puede leer la herramienta, y nada más |

### Riesgos conocidos, aceptados

- **Costo por bytes.** El guardrail cuenta objetos; unos pocos objetos enormes siguen
  siendo caros. Decisión consciente; el límite por bytes queda diferido.
- **Objetos que cambian durante la lectura.** Se lee lo que haya al abrir el stream,
  sin fijar la generación. Una lectura nunca mezcla dos versiones; el porqué está en
  §1, "¿Y con los estados límite?".
- **Encodings.** Solo se valida como UTF-8 la muestra inicial de hasta 512 bytes.
  Los bytes inválidos posteriores no cambian la clasificación y se imprimen sin
  conversión.
- **Transcoding de GCS.** Un objeto con `Content-Encoding: gzip` se busca
  descomprimido, aunque un `.gz` normal se saltee.

Quedan anotados acá para que sean decisiones y no olvidos, y aparecen como "Fuera"
en la spec.

---

## Qué sigue

Este documento alimenta el paso **Especificar**. El resultado está en
[`gcsgrep-spec.md`](./gcsgrep-spec.md).

Acá hay prosa, opciones y fundamentos; allá hay requerimientos atómicos en
Dado/Cuando/Entonces, mensajes exactos, umbrales y un VC por requisito. El base
context explica *por qué*; la spec define *qué* y *cómo se verifica*.
