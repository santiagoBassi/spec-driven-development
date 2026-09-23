# gcsgrep — decisiones

> Solo se agrega; no se reabre lo decidido sin un motivo nuevo. Cada entrada dice
> qué se decidió, por qué y qué consecuencia deja. Las de la spec y el base context
> no se repiten acá: esto es lo que se decidió **al construir**.

## Iteración 1

### D-01 · Dependencias fijadas para conservar el piso de Go 1.22

`go.mod` declara `go 1.22` y fija `cloud.google.com/go/storage v1.50.0`
(`golang.org/x/oauth2 v0.24.0`, `google.golang.org/api v0.214.0`).

`go get ...@latest` subía la directiva a `go 1.26.0`, y con `GOTOOLCHAIN=auto` el
comando descargaba solo una toolchain 1.26. La spec fija "Go ≥ 1.22" y la máquina de
desarrollo tiene 1.22.2, así que se eligió la versión más nueva que no mueve el piso.

**Consecuencia:** al subir una dependencia hay que mirar que `go.mod` siga diciendo
`go 1.22`. Se prueba con `GOTOOLCHAIN=local`, que falla en vez de descargar otra toolchain.

### D-02 · Sin reintentos desde el primer día

`client.SetRetry(storage.WithPolicy(storage.RetryNever))` en `gcs.NewClient`. El cliente
de Go reintenta por defecto, y eso contradice FR-31. Se verifica en la Iteración 3
(VC-31); apagarlo recién ahí obligaría a revisar comportamiento ya verificado.

### D-03 · Scope de solo lectura al pedir credenciales

`google.FindDefaultCredentials(ctx, storage.ScopeReadOnly)`. Con una service account, o
con una identidad impersonada, el token sale con `devstorage.read_only`, así que BR-1
queda garantizada también a nivel token y no solo por no llamar métodos de escritura.
Con un ADC `authorized_user` corriente el token es el de la persona, con los scopes con
los que hizo `gcloud auth application-default login`; ahí BR-1 depende de que el código
no escriba (no llama a ningún método de escritura).

### D-04 · ADC se verifica siempre y antes de crear el cliente

Con `STORAGE_EMULATOR_HOST` el cliente de Go no pide credenciales, y FR-35 exige
verificarlas siempre. `gcs.Credentials` corre siempre primero.

- **Cualquier** error de esa búsqueda se informa con el mensaje fijo de FR-35. Si
  `GOOGLE_APPLICATION_CREDENTIALS` apunta a un archivo roto, el mensaje dice "no
  encontradas" y no da el detalle. La spec fija el mensaje, no el diagnóstico.
- Con `STORAGE_EMULATOR_HOST` el cliente se crea **sin** `option.WithCredentials`: se
  construye sin autenticación, y las credenciales ya se comprobaron. Es la vía que usa
  el servidor de prueba de la Iteración 2.

### D-05 · Lecturas por la API JSON

`storage.WithJSONReads()`, para que el servidor de prueba de la Iteración 2 emule una
sola API (la de listado y metadata ya es JSON).

### D-06 · Listar y leer son fases separadas; el listado devuelve los nombres

`gcs.List` termina antes de que `run` lea el primer objeto y devuelve `[]string`. El
guardrail lo exige, y el total de FR-37 va a depender de lo mismo. Se corta al ver el
objeto N+1 sin pedir otra página.

**Consecuencia:** con `--max unlimited` los nombres se guardan todos en memoria (unas
decenas de bytes por objeto). NFR-2 habla del tamaño de los objetos, no de su cantidad,
y VC-42 usa 2500 objetos, pero un bucket de millones de objetos con `--max unlimited`
consumiría memoria proporcional. Es un riesgo aceptado de esta iteración.

### D-07 · Pool de workers con N = 1 y un único escritor

`workers` es una constante en `cmd/gcsgrep/run.go`; la Iteración 4 la reemplaza por el
flag. Cada registro sale con **un** `Write` bajo un mutex (`output.Printer`), así que con
N > 1 las líneas de objetos distintos se entrelazan sin partirse (FR-28). Cada worker
tiene su propio `search.Scanner` (1 MiB de buffer).

### D-08 · Reglas del parser

- Todo argumento anterior a `--` que empieza con `-` es un flag, incluido `-` solo
  (FR-22 lo dice literal). Lo que sigue a `--` es posicional siempre.
- Flags de esta iteración: `-E`, `-i`, `-n`, `--max`. `-c`, `-l` y `--concurrency`
  todavía se rechazan como `unknown flag`; también los combinados (`-in`) y `--max=5`,
  sin la pista de FR-24. Todo eso es de la Iteración 4 (y `-c`/`-l` de la 2).
- **Flags repetidos se aceptan** en esta iteración (el último valor gana). FR-25 los
  rechaza y llega en la Iteración 4.
- `--max` consume **siempre** el argumento siguiente, sea lo que sea: así `--max -3`
  informa `invalid value for --max: "-3"` y no `unknown flag`.
- `--max` sin valor no está especificado en la spec. Se usa
  `gcsgrep: flag --max requires a value`.
- Se informa el primer error en el orden: flags (en el orden de argv) → cantidad de
  posicionales → patrón vacío → regex inválida → ubicación. FR-27 no fija cuál.
- `--max` acepta solo dígitos (`+5` y ` 5` no). Un entero más grande que un `int` es un
  entero ≥ 1 válido y equivale a "sin tope práctico".

### D-09 · Cómo se compila el patrón

Siempre `regexp`: el literal es `regexp.QuoteMeta(patrón)`. Se compila primero el patrón
sin adornos y recién después con el prefijo `(?i)`, para que el detalle de una regex
inválida cite lo que escribió la persona y no `(?i)(`.

### D-10 · Línea de más de 1 MiB: siempre se busca en el stream completo

**Desvío del plan.** El plan proponía `re.Match(prefijo)` primero y `MatchReader` solo
si no daba match. Se descartó: un match sobre el prefijo retenido puede ser un **falso
positivo** con anclas o límites (`a$` sobre un prefijo que termina en `a` aunque la línea
real termine en `b`; `foo\b` sobre "…foo" que en la línea real sigue con "bar").

Toda línea que no cabe entera en el buffer se busca con `regexp.MatchReader` sobre
`retenido + resto de la línea`, y siempre se consume el resto. Las líneas normales
siguen usando `re.Match` sobre un `[]byte`.

- El buffer retenido es de `MaxLine+1` bytes (1 MiB + 1): ver ese byte extra es lo que
  distingue una línea de exactamente 1 MiB (no se trunca) de una más larga.
- **Costo:** `MatchReader` es más lento que `Match`, y solo se paga en líneas > 1 MiB.
- Los tests `TestScanLongLines/end_anchor…` y `…/word_boundary…` fallan si se vuelve a
  matchear solo el prefijo (se comprobó por mutación).

### D-11 · El `\r` final y los bordes de buffer

Se recorta **un** `\r` antes del terminador (`\r\n`) o antes del EOF. Un `\r` que cae en
el último byte de un buffer de lectura se devuelve al lector con `UnreadByte`, para
poder distinguir el `\r` del final de línea de uno que va en el medio.

### D-12 · Un error de lectura a mitad de línea no se evalúa como match

El paquete `regexp` toma cualquier error de lectura por fin de texto. Si una lectura falla
en medio de una línea larga, `$` podría matchear sobre la línea parcial. `matchLong`
revisa el error guardado y lo devuelve en vez del match.

### D-13 · Errores fuera de alcance, con el formato de la spec desde ya

Aunque su VC cierre en la Iteración 3, el código ya usa los formatos de la spec:
`gcsgrep: <ubicación>: list error: <detalle>`, `…: metadata error: <detalle>` y
`gcsgrep: gs://<bucket>/<objeto>: read error: <detalle>` (el objeto que falla se avisa, se
sigue con el resto y el exit es `2`).

**Provisional:** un objeto puntual inexistente sale como
`metadata error: storage: object doesn't exist` y no como el mensaje de FR-13; el bucket
inexistente tampoco se distingue (FR-16). La Iteración 3 los reemplaza.

### D-14 · Marcadores de carpeta

No se filtran en esta iteración (FR-17 es de la 2). Si el listado devuelve `dir/`, se lee
como un objeto de 0 bytes: no imprime nada y cuenta para el tope, que es lo que pide la
spec. `$B` no tiene marcadores (se subió con `gcloud storage cp -r`).

### D-15 · Suite `e2e/`

- `TestMain` compila `./gcsgrep` y toma la foto de `$B` para VC-39. Los tests ejecutan
  el binario como proceso aparte, con un entorno limpio (sin `GOOGLE_APPLICATION_CREDENTIALS`,
  `STORAGE_EMULATOR_HOST` ni `LANG`).
- Una variable de entorno que falta **hace fallar** el test; nunca lo saltea.
- **Chequeo P** (`requireUsageError`): cada caso corre dos veces, con las credenciales de
  `lectora` y sin ADC con el endpoint apuntando a un listener local; se verifica el
  mensaje de uso, el exit `2`, stdout vacío y **cero** conexiones. Se comprobó que el
  listener registra una conexión cuando la herramienta sí intenta conectarse.
- **VC-39** vive en `e2e/zz_vc39_test.go` (los archivos corren en orden alfabético, así
  corre último) y compara la foto de `$B` de `TestMain` con una tomada al final. Solo
  tiene sentido en la corrida completa de `go test ./e2e`. Usa `gcloud storage objects
  list … --access-token-file` con un token de `lectora`, tal como pide la spec: **la
  suite necesita `gcloud` en el PATH**. Registra cuántos objetos vio (`-v`).
- Los tests de VC-21 y VC-46 leen las líneas esperadas de `test-fixtures/` (los mismos
  bytes que hay en `$B`); VC-46 además comprueba que el fixture tenga `timeout=late` en
  el byte 1 099 838, para detectar que los fixtures se desviaron de la spec.
- Las pruebas de VCs que cierran en la Iteración 2 se llaman `TestVC12Parcial`,
  `TestVC15Parcial`, `TestVC41Parcial` y `TestVC42Parcial`, para que no se lean como "pasa".

### D-16 · Las identidades de prueba son claves JSON de service account

`lectora` y `sin-acceso` se usan como ADC con `GOOGLE_APPLICATION_CREDENTIALS` apuntando a
sus claves (`lectora.json` y `sin-acceso.json`, en la raíz del repo). El plan admitía esta vía
o la impersonación; quien administra el proyecto entregó claves.

Se había escrito un script de impersonación (`tools/e2e-creds.sh`); nunca se probó contra
GCS real y se eliminó en vez de dejar código sin verificar.

**Consecuencias:**
- Son secretos de larga vida. Están en `.gitignore` (`/lectora.json`, `/sin-acceso.json`), con
  permisos `0600`. **Nunca se versionan**; conviene rotarlas o borrarlas al terminar el ejercicio.
- Su contenido apareció en la conversación de trabajo con el agente, así que hay que
  considerarlas expuestas más allá de la máquina local.
- El scope de solo lectura (D-03) aplica por construcción: `gcs.Credentials` lo pasa a
  `FindDefaultCredentials`, y con una clave de service account el token se pide con él. No se
  inspeccionó el token que usa la herramienta; lo que sí se comprobó es que la identidad no tiene
  permisos de escritura (abajo).
- Los permisos efectivos se comprobaron con `testIamPermissions` (que no escribe): `lectora` tiene
  solo `storage.objects.get` y `storage.objects.list`; `sin-acceso`, ninguno.

## Entre la Iteración 1 y la 2 (clasificación de lo aprendido)

Se clasificó lo aprendido al cerrar la Iteración 1: lo que cambia el *cómo* va al plan, lo que
cambia el *qué* vuelve a la spec, que se revisó de nuevo (sigue en 50 requisitos y 50 VCs).

### D-17 · Un recurso inexistente se informa con un solo mensaje: `not found`

**Decisión (cambia la spec, FR-13 y FR-16):** `gcsgrep: not found: <ubicación>`, con la
ubicación tal como se recibió, igual que `access denied: <ubicación>`. Ya no se distingue si
falta el objeto o el bucket. Los dos requisitos se reparten por **tipo de ubicación**, no por qué
recurso falta: FR-13 para objeto puntual (falte el objeto o su bucket), FR-16 para bucket o
prefijo. Así se conservan los números y la regla de un VC por requisito.

**Por qué:** medido contra GCS real, el cliente de Go convierte todo `404` de la consulta de
metadata en `object doesn't exist` y descarta el cuerpo del error, así que un objeto
inexistente y un bucket inexistente con ubicación de objeto son indistinguibles.
`lectora` tampoco puede consultar el bucket (no tiene `storage.buckets.get`).

**Alternativas descartadas:**
- Un listado de 1 resultado solo en el camino de error: distingue, pero roza la letra de FR-12
  ("sin listar el bucket") y suma un request.
- Un transporte HTTP propio que conserve el cuerpo del `404`: cumple FR-12, pero es mucho más
  código y toca cómo se autentica.
- Aclarar FR-12 ("en el caso exitoso") para permitir el listado: una revisión de spec para
  distinguir algo que no se necesita.

**Consecuencias:**
- Se pierde la pista de la `/` final: `gcsgrep timeout gs://b/logs/app` (sin barra) ahora dice
  solo `not found: gs://b/logs/app`. Se puede volver a agregar con otra revisión de spec.
- Un prefijo sin objetos en un bucket que existe sigue siendo `1` (FR-15): eso no es un recurso
  inexistente.
- Un `404` al leer un objeto ya listado (borrado entre el listado y la lectura) es un fallo de
  lectura (FR-29), no `not found`.
- **El código de la Iteración 1 no cambia.** Sigue con los mensajes provisionales de D-13
  (`list error: storage: bucket doesn't exist`, `metadata error: storage: object doesn't
  exist`); la Iteración 3 los reemplaza. Se implementa mapeando `ErrObjectNotExist` y
  `ErrBucketNotExist`, sin requests extra.

### D-18 · Un flag que exige valor y llega sin él es un error de uso

**Decisión (cambia la spec, BR-4 y FR-26):** el valor de `--max` y de `--concurrency` es siempre el
argumento que les sigue, sea cual sea (`--max -3` informa el valor `-3`). Si el flag es el último
argumento: código `2` y `gcsgrep: flag <flag> requires a value`, sin operar contra GCS.

**Por qué:** la Iteración 1 lo resolvió por su cuenta (D-08) sin que la spec dijera nada: un hueco
sin VC, del tipo "lo decide el agente por vos". Se cerró extendiendo los dos requisitos y sus VCs
(un caso más en VC-42 y en VC-26), sin requisito nuevo, así que la cuenta no cambia.

**Alternativas descartadas:** un requisito nuevo (FR-39 y VC-51), que cambia los totales en la
spec, el plan y la cobertura; y dejarlo como decisión de implementación, que deja la spec callada.

**Consecuencias:** el código ya cumple para `--max`. El caso de `--max` se agrega a
`TestVC42Parcial` cuando VC-42 cierre en la Iteración 2; el de `--concurrency` llega con el flag,
en la Iteración 4.

### D-19 · El cliente no debe pedir gzip: se ajusta el cliente, no BR-7

**Medido** (servidor local que registra los headers; objeto real con `contentEncoding: gzip` y
`cacheControl: no-cache`): `transcoded.log` ya llega como texto, pero el request envía
`Accept-Encoding: gzip` en metadata y en lectura. No lo pide la librería (solo con
`ReadCompressed`): lo agrega el transporte HTTP de Go, que descomprime por su cuenta. BR-7 exige
que la lectura no lo pida.

**Decisión:** se mantiene BR-7 tal como está y en la Iteración 2 se ajusta el cliente para que no
pida gzip, de modo que sea GCS quien decida si descomprime. Se verifica con el servidor de prueba,
que registra los headers, y con VC-45 contra `$B`, que tiene que seguir pasando.

**Alternativa descartada:** relajar BR-7 y aceptar que el transporte descomprima. Es más barato,
pero cambia la spec para acomodar un detalle de la librería, y un objeto con `no-transform` se
buscaría como texto contra lo que dice BR-7 (ningún fixture lo cubre).

**Consecuencia aceptada:** para un objeto con `Content-Encoding: gzip`, GCS descomprime del lado
del servidor y viajan más bytes que con gzip en el cable. Es raro en el uso previsto, y el
guardrail de costo cuenta objetos, no bytes.
