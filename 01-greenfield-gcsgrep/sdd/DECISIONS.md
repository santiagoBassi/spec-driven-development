# gcsgrep — decisiones

> Solo se agrega; no se reabre lo decidido sin un motivo nuevo. Cada entrada dice
> qué se decidió, por qué y qué consecuencia deja. Las de la spec y el base context
> no se repiten acá: esto es lo que se decidió **al construir**.

## Iteración 1

### D-01 · `go.mod`/`go.sum` recuperados del historial, no re-resueltos

Se tomaron tal cual de `7b8d597^` (el commit anterior a borrar la implementación
previa): `go 1.22`, `cloud.google.com/go/storage v1.50.0` (`golang.org/x/oauth2
v0.24.0`, `google.golang.org/api v0.214.0`). El resto del código se escribió desde
cero.

**Por qué:** esas versiones ya están validadas contra el piso de Go de la máquina de
desarrollo (1.22.2); volver a correr `go get` corre el riesgo de que
`GOTOOLCHAIN=auto` suba la directiva `go` de `go.mod` a una versión más nueva que la
instalada (ver la D-01 original, que documentó justo ese problema).

**Consecuencia:** el gate siempre corre con `GOTOOLCHAIN=local`, para que un `go get`
futuro falle en vez de descargar otra toolchain si alguien lo corre sin querer.

### D-02 · Sin reintentos desde el primer día

`client.SetRetry(storage.WithPolicy(storage.RetryNever))` en `gcs.NewClient`. El
cliente de Go reintenta por defecto, y eso contradice FR-31 (Iteración 3). Apagarlo
recién ahí obligaría a revisar comportamiento ya verificado en esta iteración.

### D-03 · Scope de solo lectura al pedir credenciales

`google.FindDefaultCredentials(ctx, storage.ScopeReadOnly)`. Con las claves de
service account de `lectora`/`sin-acceso` el token sale con `devstorage.read_only`,
así que BR-1 queda garantizada también a nivel token. Con un ADC `authorized_user`
corriente (`gcloud auth application-default login` sin impersonar) el token es el de
la persona, con los scopes con los que hizo login; ahí BR-1 depende de que el código
nunca llame a un método de escritura (que no lo hace: `gcs` solo expone `List`,
`Stat`, `Open`).

### D-04 · ADC se verifica siempre y antes de crear el cliente

`gcs.Credentials` corre primero en `run()`, siempre, aunque esté definido
`STORAGE_EMULATOR_HOST` (FR-35). Confirmado con el chequeo P (VC-35): en el entorno
sin ADC, el listener no registra ninguna conexión.

`gcs.NewClient` no pasa `option.WithCredentials` cuando `STORAGE_EMULATOR_HOST` está
definido: la librería ya arma el cliente con `option.WithoutAuthentication()` en ese
caso (visto en el código de `storage.NewClient` v1.50.0), y las credenciales ya se
comprobaron antes.

### D-05 · Lecturas por la API JSON

`storage.WithJSONReads()`, para que el servidor de prueba de la Iteración 2 emule
una sola API (la de listado y metadata ya es JSON).

### D-06 · Listar y leer son fases separadas; el listado devuelve los nombres

`gcs.List` devuelve `[]string` y termina antes de que `run` lea el primer objeto. El
guardrail lo exige (BR-3): se corta al ver el objeto N+1 sin pedir otra página
(`Query.SetAttrSelection([]string{"Name"})` para no traer más metadata de la que
hace falta).

**Prefijo con barra final para el listado.** `location.Location.ListPrefix()`
agrega la barra final al prefijo antes de pedírselo a GCS (`Path + "/"`), aunque
`Path` se guarda sin ella. Sin la barra, `gcs.List(bucket, "data", …)` matchearía
también un hipotético `database/…` por ser un prefijo de string, no de ruta; `$B` no
tiene ese choque hoy, pero es la lectura correcta de FR-11 ("objetos cuyo nombre
empieza con `<prefijo>/`") y evita que un fixture futuro lo exponga en silencio.

**Consecuencia aceptada:** con `--max unlimited` los nombres se guardan todos en
memoria (unas decenas de bytes por objeto). Un bucket de millones de objetos con
`--max unlimited` consumiría memoria proporcional a la cantidad de objetos, no a su
tamaño. NFR-2 (Iteración 5) mide memoria contra el tamaño de los objetos, no contra
su cantidad; queda como riesgo conocido, igual que en la implementación anterior.

### D-07 · Pool de workers con N = 1 y un único escritor

El bucle de `run()` procesa los objetos listados de a uno. `output.Printer` ya
escribe bajo mutex (un `Write` por registro) aunque con `N = 1` no hace falta para
la integridad de las líneas: se escribe así ahora para no reabrir este archivo
cuando la Iteración 4 suba `N`.

### D-08 · Reglas del parser (`internal/cli`)

- Todo argumento anterior a `--` que empieza con `-` es un flag, incluido `-` solo.
  Lo que sigue a `--` es posicional siempre (FR-21).
- Flags de esta iteración: `-E`, `-i`, `-n`, `--max`. `-c`, `-l` y `--concurrency`
  todavía se rechazan como `unknown flag` (son de las Iteraciones 2 y 4); los
  combinados (`-in`) y `--max=5` también, sin la pista de FR-24 (Iteración 4).
- **Flags repetidos se aceptan** en esta iteración: gana el último (`-i -i` no
  rompe nada porque son booleanos idempotentes; `--max 5 --max 10` se queda con
  `10`). FR-25 los rechaza y llega en la Iteración 4.
- `--max` consume **siempre** el argumento siguiente, sea lo que sea: `--max -3`
  informa `invalid value for --max: "-3"`, no `unknown flag: "-3"`.
- Orden de un único error de uso (FR-27): flags, en el orden de argv → cantidad de
  posicionales → patrón vacío. El patrón inválido como regex (`search.Compile`) y la
  ubicación inválida (`location.Parse`) se comprueban después, en ese orden, fuera de
  `cli.Parse`.
- `--max` acepta solo dígitos decimales (`+5` y ` 5` no válidos, dígito a dígito
  antes de parsear). Un entero que desborda `uint64` (`strconv.ParseUint` con
  `ErrRange`) es un tope válido que nunca se alcanza (`Unlimited = math.MaxInt64`),
  igual que uno que cabe en `uint64` pero no en `int64`.

### D-09 · Cómo se compila el patrón (`internal/search.Compile`)

Siempre `regexp` de la stdlib: el literal se arma con `regexp.QuoteMeta`. Se
compila primero el patrón sin `(?i)` y recién después con el prefijo si hay `-i`,
para que el detalle de una regex inválida (FR-5) cite lo que la persona escribió, no
`(?i)(`.

### D-10 · Línea de más de 1 MiB: streaming con un `io.RuneReader` propio, no `bufio.NewReader`

El buffer retenido es de `MaxLine+1` bytes (1 MiB + 1): ver ese byte extra distingue
una línea de exactamente 1 MiB (no se trunca) de una más larga, sin necesitar una
lectura de más para decidirlo hasta que hace falta.

Toda línea que no entra en ese buffer se busca con `regexp.MatchReader` sobre un
`runeReader` que entrega primero lo retenido y después el resto de la línea, tomado
del stream a demanda, decodificando UTF-8 un rune a la vez. **Se descartó envolver
esa fuente con `bufio.NewReader`** (que sí implementa `io.RuneReader`): su buffer
interno de lectura adelantada (4 KiB por defecto) consume del stream subyacente más
bytes de los que `MatchReader` termina usando, y esos bytes quedarían atrapados
adentro de ese buffer, invisibles para el drenado posterior. Como la spec exige
"siempre se consume el resto" de la línea después de decidir el match (para que la
siguiente línea empiece en el lugar correcto), ese buffer adelantado rompería el
conteo de líneas siguientes. El `runeReader` propio decodifica exactamente los bytes
que un rune necesita (usando `utf8.FullRune` para saber cuándo tiene una secuencia
completa) y no lee de más; el drenado posterior se hace llamando al mismo
`readByte()` del `runeReader` (no directamente a la fuente subyacente), así ningún
byte que haya quedado en su cola de "leído pero no consumido por un rune" se pierde.

Se comprobó con `TestScanEndAnchorPastLimit` (un test adversarial: línea de
`MaxLine` bytes 'a' + un byte final 'b', que dispara falsos positivos con `a$` si el
código llega a matchear solo sobre el prefijo retenido) y
`TestScanWordBoundaryAcrossLimit` (`\bword\b` con la palabra a caballo del límite).

### D-11 · El `\r` final, byte a byte, sin `bufio.Reader.UnreadByte`

`lineState.nextByte()` decide `\r\n` con una máquina de dos estados: al leer un
`\r`, se espía el byte siguiente; si es `\n`, la línea terminó sin `\r`; si no, el
`\r` es literal y el byte espiado se guarda en un único `pushback` de 1 byte para la
próxima llamada. Un `\r` como último byte del stream (sin nada después) es literal
también: no hay `\n` que lo convierta en fin de línea. Cubierto por
`TestScanCRNotFollowedByLF` y `TestScanTrailingLoneCR`.

### D-12 · Un error de lectura no se evalúa como si fuera match

`lineState.err` guarda cualquier error de lectura que no sea `io.EOF`. `Scan`
revisa ese campo después de terminar de leer la línea (por streaming o no) y
**antes** de invocar la función de callback: un corte a mitad de línea nunca llega a
imprimirse ni a contar como match, aunque el contenido recibido hasta ahí lo
tuviera. No hay ningún VC de la Iteración 1 que ejercite esto contra GCS real (las
fallas de red son de la Iteración 3), pero el mecanismo ya queda listo para FR-29b
y quedó cubierto con dos unitarios (`TestScanReadErrorMidLine`,
`TestScanReadErrorDuringLongLine`).

### D-13 · Errores fuera de alcance, con mensajes provisionales

- Un objeto puntual inexistente, o un bucket inexistente en cualquier ubicación,
  sale como `metadata error: storage: object doesn't exist` o
  `list error: storage: bucket doesn't exist` (los mensajes que ya trae el cliente
  de Go), no como `not found: <ubicación>` (FR-13a/FR-13b/FR-16a/FR-16b, Iteración
  3).
- `access denied: <ubicación>` (BR-2) sí está implementado ya, porque no depende de
  distinguir qué recurso falta: alcanza con el código HTTP 403.

### D-14 · Marcadores de carpeta

No se filtran en esta iteración (FR-17a es de la Iteración 2). `$B` no tiene
marcadores (los fixtures se subieron con `gcloud storage cp -r`, que no los crea),
así que ningún VC de esta iteración los ejercita.

### D-15 · Suite `e2e/`

- `TestMain` compila `../cmd/gcsgrep` a un binario temporal (con `GOTOOLCHAIN=local`
  para el subproceso de build) y toma la foto de VC-39 con `gcloud storage objects
  list … --format=value(name,generation,metageneration)`, usando
  `CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE` apuntando a la clave de `lectora` y un
  `CLOUDSDK_CONFIG` temporal (para no depender de que quien corre la suite tenga
  `gcloud` logueado). Si falta alguna de las tres variables de entorno
  (`GCSGREP_BUCKET`, `GCSGREP_LECTORA`, `GCSGREP_SIN_ACCESO`), `TestMain` falla con
  código 1: nunca saltea un test en silencio.
- Cada invocación (`runTool`) arma un entorno desde cero: solo `PATH`, un `HOME`
  temporal, y `GOOGLE_APPLICATION_CREDENTIALS` apuntando a la clave que corresponda
  (o, para el caso sin ADC, `HOME` y `CLOUDSDK_CONFIG` temporales sin esa
  variable). Nunca hereda `LANG` ni `STORAGE_EMULATOR_HOST` salvo que el test los
  agregue explícitamente.
- **Chequeo P** (`checkUsage`, con `requireUsageError`/`requireUsageErrorPrefix`
  arriba): cada caso corre con `lectora` y otra vez sin ADC, con
  `STORAGE_EMULATOR_HOST` apuntando a un listener TCP propio del test (`net.Listen`
  en `127.0.0.1:0`, en proceso, sin depender de `nc`). Se verifica el mensaje de
  uso, exit `2`, stdout vacío y **cero** conexiones aceptadas.
- **Autotest del listener** (hecho a mano durante el desarrollo, no como test
  automatizado): con `lectora` y `STORAGE_EMULATOR_HOST` apuntando al mismo tipo de
  listener, se confirmó que sí registra una conexión (593 bytes del request HTTP)
  cuando la herramienta intenta hablar con GCS. Da confianza en que "cero
  conexiones" en el chequeo P es una señal real, no un listener que nunca iba a
  recibir nada.
- `TestVC39` vive en `e2e/zz_vc39_test.go` (los archivos de un paquete se procesan
  en orden alfabético, así corre último) y compara la foto de `TestMain` con una
  nueva al final. Solo tiene sentido en la corrida completa de `go test ./e2e`.
- Las líneas esperadas de VC-4, VC-21 y VC-46 se leen de `test-fixtures/` en vez de
  transcribirse a mano, y VC-46 además comprueba que el fixture tenga
  `timeout=late` en el byte 1 099 838 de la línea 1: si el fixture se desvía de la
  spec, el test lo dice explícitamente en vez de fallar de forma confusa más abajo.
- Los VCs que esta iteración deja con evidencia parcial se llaman
  `TestVC12Parcial`, `TestVC41Parcial` y `TestVC42Parcial` (más
  `TestVC15aParcial`, unitario de `internal/location`), para que no se lean como
  "pasa" en un `go test -v`.

### D-16 · Las identidades de prueba son claves JSON de service account

`lectora` y `sin-acceso` se usan como ADC con `GOOGLE_APPLICATION_CREDENTIALS`
apuntando a sus claves (`lectora.json` y `sin-acceso.json`, en la raíz del repo,
`.gitignore` con permisos `0600`, nunca se versionan). El plan admitía esta vía o
impersonación; ya estaban creadas de una preparación anterior del entorno, y se
confirmó con `gcloud storage ls`/`objects list` que `lectora` lista `$B` y
`sin-acceso` recibe `403` antes de escribir una sola línea de Go.

### D-17 · Identificadores de test estables

Cada test de `e2e/` lleva el número del VC que ejercita en el nombre
(`TestVC01`, `TestVC07a`, `TestVC09b`, …), con el sufijo del VC cuando lo tiene. Un
VC con evidencia parcial en esta iteración lleva el sufijo `Parcial`. Así el mapeo
entre `gcsgrep-cobertura-vc.md` y el código es directo, sin tener que leer el
cuerpo del test para saber qué cubre.
