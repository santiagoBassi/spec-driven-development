# gcsgrep — plan de iteraciones

> Salida del paso **Planificar**, a partir de [`gcsgrep-spec.md`](./gcsgrep-spec.md)
> (revisada, 50 requisitos, 50 VCs, 0 huérfanos).
>
> Cada iteración es un contrato chico y verificable: termina con código andando y
> sus VCs pasando, **más todos los VCs de las iteraciones anteriores**. La siguiente
> no empieza hasta que la anterior pasa su gate.
>
> La spec describe la v1 completa. Acá se decide **en qué orden** se construye y
> **qué se posterga** en cada paso. El alcance que queda afuera de la v1 también
> vive acá, al final.

## Cómo está ordenado

Por **dependencia**, no por entusiasmo. La Iteración 1 es lo mínimo que se puede
ejercitar de punta a punta contra GCS real y cumple las tres restricciones duras del
enunciado desde el primer día: solo lectura, no ampliar el acceso y guardrail de
costo. Cada iteración siguiente agrega una capa que se apoya en la anterior.

Una segunda regla de orden: **la Iteración 1 se verifica solo contra el bucket de
fixtures, sin servidor de prueba.** El servidor aparece en la Iteración 2 y crece en
las siguientes, a medida que hacen falta cosas que GCS real no deja observar
(requests registrados, fallas inyectadas, lecturas simultáneas).

| Iteración | Entrega | Implementa | VCs que cierra |
|---|---|---|---|
| 1 | Búsqueda de punta a punta, secuencial, con guardrail | FR-1, FR-3 a FR-9, FR-12, FR-14, FR-15, FR-21 a FR-23, FR-35, BR-1 a BR-4, BR-8 | 16 |
| 2 | Servidor de prueba, sniffing y formatos `-c` / `-l` | FR-2, FR-10, FR-11, FR-17 a FR-20, BR-5 a BR-7 | 14 (10 propios + VC-12, 15, 41, 42) |
| 3 | Errores de GCS y fallos de red | FR-13, FR-16, FR-29 a FR-34, NFR-3 | 9 |
| 4 | Concurrencia, progreso y parser estricto | FR-24 a FR-28, FR-36 a FR-38 | 8 |
| 5 | NFRs y contrato de scripting | BR-9, NFR-1, NFR-2 | 3 |
| | | | **50** |

---

## Entorno y costo

Todo corre sobre una cuenta con el *Always Free* de Cloud Storage: 5 GB-mes en
`us-east1`/`us-west1`/`us-central1`, 5000 operaciones clase A, 50 000 clase B y
100 GB de egress desde Norteamérica por mes, sumados entre todos los buckets.

| Recurso | Qué es | Se prepara en |
|---|---|---|
| `$B` = `gs://sdd-fardenghi-itba` | Bucket de fixtures, `us-east1`, Standard, *public access prevention* activado. Ya tiene los 22 fixtures con la metadata de `sniffing/` correcta | Existe; en la Iteración 1 se sube `data/acentos.log` |
| `lectora` | Service account con solo `roles/storage.objectViewer` sobre `$B` | Iteración 1 |
| `sin-acceso` | Service account sin ningún rol sobre `$B` | Iteración 1 |
| Listener de conexiones | `nc -lk 127.0.0.1 <puerto>`, para el chequeo P y VC-35 | Iteración 1 |
| Servidor de prueba | Emulador local de la API JSON de GCS | Iteración 2 (crece en 3 y 4) |
| `$P` | Segundo bucket, `us-east1`, Standard, con los datasets de NFR-1 y NFR-2 | Iteración 5 |

**Consumo estimado.** Una corrida de la suite de la Iteración 1 hace unas 30
operaciones clase A (listados), unas 40 clase B (lecturas) y baja unos 3 MB: alcanza
para cientos de corridas por mes. La Iteración 5 es la única que mueve volumen:
~1,2 GB almacenados, ~1100 operaciones clase A de subida, ~3100 lecturas y ~4 GB de
egress para tres mediciones. Entra en el free tier. Se configura una alerta de
presupuesto de USD 1 antes de la Iteración 1.

---

## Estructura y memoria persistente

Esto no es una iteración: es lo que la Iteración 1 deja armado y las demás mantienen.

**Código** (según los módulos del base context):

```
cmd/gcsgrep/        main: arma las piezas y traduce el resultado a exit code
internal/cli/       parseo de argv y validación (sin tocar GCS)
internal/location/  gs://bucket/[ruta] → modo bucket / prefijo / objeto
internal/gcs/       ADC, listado con tope, metadata y lectura por streaming
internal/sniff/     texto / no-texto con los primeros 512 bytes      (Iteración 2)
internal/search/    delimitar líneas, buscarlas completas y retener el primer MiB
                    de cada una para la salida
internal/output/    registros completos a stdout; avisos y progreso a stderr
internal/fakegcs/   servidor de prueba                                (Iteración 2)
e2e/                un test por VC: TestVC01…TestVC50, invocan ./gcsgrep
```

**Verificación.** Cada VC es un test de `e2e/` con el número del VC en el nombre,
que compila `./gcsgrep` y lo ejecuta como proceso aparte. El gate de cada iteración
es:

```bash
go build -o gcsgrep ./cmd/gcsgrep
go test ./e2e -run 'TestVC(01|03|…)$' -v     # la lista de VCs de la iteración
go test ./e2e -v                              # regresión: todo lo anterior
```

Los tests de `$B` leen el bucket y las credenciales de variables de entorno. El
chequeo P es un helper de `e2e/` que abre el listener, corre el comando sin ADC y
verifica mensaje y cero conexiones. Desde la Iteración 2, el servidor de prueba corre
dentro del proceso de test (`httptest`), así cada test lee su registro de requests e
inyecta fallas sin procesos extra.

**Memoria persistente** (ver [`docs/guia-sdd.md`](./docs/guia-sdd.md)):

| Archivo | Se crea en | Cómo evoluciona |
|---|---|---|
| `CONTEXT.md` | Iteración 1 | Se actualiza al cerrar cada iteración: estado, mapa de archivos, comandos |
| `DECISIONS.md` | Iteración 1 | Solo se agrega. Arranca con las decisiones técnicas de abajo |
| `iterations/NN-<nombre>.md` | Al cerrar cada iteración | Inmutable: qué se planeó, qué pasó, desvíos, handoff |
| `gcsgrep-cobertura-vc.md` | Iteración 1 | Una fila por VC, se completa a medida que pasan |

---

## Iteración 1 — Búsqueda de punta a punta

**Objetivo:** que alguien pueda ejecutar `gcsgrep timeout gs://bucket/prefijo/` contra
GCS real y obtener los matches correctos, con exit codes de `grep`, sin escribir
nada, sin ampliar su acceso y sin poder escanear un bucket enorme por error.

**Alcance**

- Los módulos `cli`, `location`, `gcs`, `search` y `output` con sus límites: todo lo
  que se decide con argv se decide antes de autenticar.
- Parser con `-E`, `-i`, `-n`, `--max` y `--`, flags desconocidos y conteo de
  posicionales.
- Patrón literal por defecto y RE2 con `-E`; `-i` con `(?i)` de RE2; patrón vacío
  rechazado.
- Ubicaciones de prefijo (y bucket, que es el prefijo vacío) y de objeto puntual;
  formato inválido rechazado.
- Chequeo explícito de ADC antes de crear el cliente, aunque esté definido
  `STORAGE_EMULATOR_HOST`.
- Listado paginado con tope: se corta al ver el objeto N+1 y aborta antes de leer.
- Lectura por streaming, **secuencial** (un objeto por vez), con recorte de `\r`;
  se busca en cada línea completa y se retiene solo su primer MiB para imprimir.
- Acceso denegado (`403`) en listado o metadata → `gcsgrep: access denied: <ubicación>`.
- Entorno: subir `data/acentos.log` a `$B`, crear `lectora` y `sin-acceso`, alerta de
  presupuesto.

**Fuera de alcance de esta iteración:** servidor de prueba, sniffing de binarios (los
objetos no-texto se leen como bytes), `-c`, `-l`, marcadores de carpeta, mensajes
específicos de objeto o bucket inexistente, fallos de red y timeouts, concurrencia,
progreso, flags combinados o repetidos.

**Criterios de éxito**

- [ ] VC-1 pasa — literal bajo un prefijo, formato `gs://bucket/objeto:texto`
- [ ] VC-3 pasa — el patrón es literal sin `-E`
- [ ] VC-4 pasa — regex RE2 con `-E`
- [ ] VC-5 pasa — regex inválida → `2`, cumple el chequeo P
- [ ] VC-6 pasa — patrón vacío → `2`, cumple el chequeo P
- [ ] VC-7 pasa — `-i`, también con `Ñ`/`Á` y `LANG=C`
- [ ] VC-8 pasa — `-n` numera desde `1`
- [ ] VC-9 pasa — recorte de `\r` en líneas `\r\n`
- [ ] VC-14 pasa — ubicaciones inválidas → `2`, cumple el chequeo P
- [ ] VC-21 pasa — todo lo que sigue a `--` es posicional
- [ ] VC-22 pasa — flag desconocido → `2`, cumple el chequeo P
- [ ] VC-23 pasa — cantidad de posicionales distinta de dos → `2`, cumple el chequeo P
- [ ] VC-35 pasa — sin ADC → `2`, ninguna conexión al listener
- [ ] VC-39 pasa — la suite de esta iteración corre con `lectora` y `$B` no cambia
- [ ] VC-40 pasa — `access denied` con `sin-acceso`, match con `lectora`
- [ ] VC-46 pasa — un match literal tardío y una regex que atraviesa el primer
  MiB se reportan con salida truncada en `...`

**Implementado, pero el VC cierra en la Iteración 2.** Estos requisitos se construyen
acá porque son parte del camino de punta a punta, pero una parte de su VC solo se
observa con el servidor de prueba. En `gcsgrep-cobertura-vc.md` figuran como
*implementado, VC pendiente* con la evidencia parcial, nunca como "pasa".

| VC | Evidencia parcial en la Iteración 1 | Lo que falta observar |
|---|---|---|
| VC-12 (FR-12) | `./gcsgrep timeout gs://$B/logs/app/api.log` → `0`, solo líneas de `api.log` | Que no haya request de listado ni lectura de `a.log.bak` |
| VC-15 (FR-15) | `gs://$B/no-existe/` → `1`, stdout y stderr vacíos | Bucket existente sin objetos (`fake/empty`) |
| VC-41 (BR-3) | `--max 5` sobre `edge-cases/` aborta con el mensaje exacto; `--max 6` → `0` | Tope por defecto de 1000 y que no se pida la página siguiente |
| VC-42 (BR-4) | Valores inválidos de `--max` → `2`, cumplen el chequeo P | `--max unlimited` leyendo 2500 objetos |

**Demostrable así:**

```bash
./gcsgrep timeout gs://$B/logs/app/                  # 3 líneas, exit 0
./gcsgrep -n -i timeout gs://$B/logs/app/api.log     # 4 líneas numeradas, exit 0
./gcsgrep -E 'job_id=\d+ queue=\w+ status=ERROR' gs://$B/logs/app/worker.log
./gcsgrep timeout gs://$B/no-existe/                 # nada, exit 1
./gcsgrep --max 5 timeout gs://$B/edge-cases/        # guardrail, exit 2
./gcsgrep timeout logs/                              # invalid location, exit 2
```

**Por qué estos y no otros**

- **El guardrail (BR-3, BR-4) se implementa acá** aunque su VC cierre en la
  Iteración 2: el enunciado prohíbe escanear un bucket enorme sin guardrail, y ya en
  esta iteración se puede listar un prefijo real. No debería existir ni una versión
  intermedia que lo permita.
- **BR-2 entra acá** por la misma razón: es una restricción dura. La herramienta la
  cumple por construcción (solo usa ADC), y VC-40 fija el mensaje.
- **BR-8 entra acá** porque buscar la línea completa reteniendo solo su primer
  MiB para la salida es parte del lector de líneas, que se escribe una sola vez.
  Sin ese límite de salida, `long_line_exceeds_1mb.log` rompe la evidencia de
  VC-41 (`--max 6` lee ese objeto) y el streaming no tendría memoria acotada.
- **VC-3 y la evidencia de VC-41 leen objetos no-texto** porque todavía no hay
  sniffing. Sus resultados no dependen de eso: `old_logs.log.gz` no contiene `(`, y
  VC-41 con `--max 6` solo mira el exit code. Se vuelven a verificar en la
  Iteración 2.

**Notas de implementación**

- **Reintentos apagados desde el día uno**
  (`client.SetRetry(storage.WithPolicy(storage.RetryNever))`). El cliente de Go
  reintenta por defecto, y eso contradice FR-31. Se verifica en la Iteración 3, pero
  apagarlo después obligaría a revisar comportamiento ya verificado.
- **Scope de solo lectura** (`devstorage.read_only`) al pedir las credenciales: BR-1
  queda garantizada también a nivel token, no solo por no llamar métodos de escritura.
- **ADC explícito.** Con `STORAGE_EMULATOR_HOST` el cliente de Go no pide
  credenciales. FR-35 exige verificarlas siempre, así que se llama a
  `google.FindDefaultCredentials` antes de crear el cliente.
- **Lecturas por la API JSON** (`storage.WithJSONReads()`), para que el servidor de
  prueba de la Iteración 2 emule una sola API.
- **Listar y leer son fases separadas** aunque todavía no haya progreso: el
  guardrail lo exige, y el total de FR-37 va a depender de lo mismo.
- **Pool de workers con N = 1.** La Iteración 4 solo cambia N y agrega el flag. Cada
  línea se escribe con una sola escritura desde un único escritor.
- **Identidades de prueba.** `lectora` y `sin-acceso` se usan como ADC con
  `gcloud auth application-default login --impersonate-service-account=…` (requiere
  `roles/iam.serviceAccountTokenCreator` sobre cada una) o con una clave JSON en
  `GOOGLE_APPLICATION_CREDENTIALS`. Las claves no se versionan.

---

## Iteración 2 — Servidor de prueba, sniffing y formatos de salida

**Objetivo:** que la herramienta sirva sobre un bucket real, donde hay binarios,
gzip y carpetas, que tenga los dos formatos de resumen de `grep`, y cerrar los VCs
que la Iteración 1 dejó con evidencia parcial.

**Alcance**

- Servidor de prueba: listar (con paginación y prefijo), metadata y contenido;
  buckets `fake` y `empty`; registro de requests; detección de que el cliente cerró
  la conexión antes de terminar el cuerpo.
- Módulo `sniff`: clasificación por los primeros 512 bytes, con los casos de borde de
  BR-6 (carácter cortado por el borde, secuencia incompleta en objetos chicos, bytes
  inválidos después de la muestra).
- Aviso `not a text file, skipped` sin afectar el exit code.
- `-c` (conteo por objeto, incluido `0`) y `-l` (con corte de la lectura en el primer
  match), y el rechazo de `-c`/`-l`/`-n` combinados.
- Ubicación de bucket completo y de prefijo verificadas con objetos anidados.
- Marcadores de carpeta: cuentan para el tope, no se leen ni se informan.

**Criterios de éxito**

- [ ] VC-2 pasa — sin matches en todo `$B` → `1`, exactamente 6 avisos
- [ ] VC-10 pasa — `-l` sobre todo el bucket, 9 objetos
- [ ] VC-11 pasa — `-l` bajo `data/`, nada de afuera
- [ ] VC-17 pasa — marcadores de carpeta ignorados, pero cuentan para `--max`
- [ ] VC-18 pasa — `-c` con conteos en `0` y todo en `0` → `1`
- [ ] VC-19 pasa — `-l` corta la lectura en el primer match
- [ ] VC-20 pasa — `-c -l`, `-n -c`, `-n -l` → `2`, cumple el chequeo P
- [ ] VC-43 pasa — no-texto salteado sin contar como fallo
- [ ] VC-44 pasa — bordes de la muestra de 512 bytes
- [ ] VC-45 pasa — gzip salteado por contenido; `transcoded.log` se busca como texto
- [ ] VC-12 pasa — objeto puntual sin request de listado *(implementado en la 1)*
- [ ] VC-15 pasa — también con el bucket `empty` *(implementado en la 1)*
- [ ] VC-41 pasa — tope de 1000 sin paginar de más *(implementado en la 1)*
- [ ] VC-42 pasa — `--max unlimited` sobre 2500 objetos *(implementado en la 1)*
- [ ] **VC-3 sigue pasando**, ahora con los no-texto salteados

**Riesgo a resolver primero: transcoding (VC-45).** BR-7 exige que la lectura no
pida `Accept-Encoding: gzip`. Hay que confirmar contra `$B` qué headers manda el
cliente de Go con lecturas JSON y si el transporte HTTP descomprime por su cuenta.
Si `transcoded.log` llega comprimido, la corrección es de configuración del cliente.
Si no se puede lograr, cambia el *qué*: se vuelve a la spec, no se ajusta el VC.

---

## Iteración 3 — Errores de GCS y fallos de red

**Objetivo:** que ningún fallo de GCS cuelgue la herramienta ni se confunda con un
resultado completo.

**Alcance**

- Distinción entre objeto inexistente (FR-13) y bucket inexistente (FR-16).
- Error de lectura de un objeto: aviso, se sigue con el resto, exit `2`.
- Error de listado o de metadata: aborta sin leer.
- Timeout por inactividad de 30 s en listado, metadata y lectura, contado desde el
  último byte recibido.
- El servidor de prueba agrega la inyección de fallas por request: `500`, conexión
  colgada y cuerpo en tramos espaciados.

**Criterios de éxito**

- [ ] VC-13 pasa — objeto inexistente con la pista de la `/` final
- [ ] VC-16 pasa — `bucket not found` para ubicación de bucket y de objeto
- [ ] VC-29 pasa — un objeto que falla no frena al resto, exit `2`
- [ ] VC-30 pasa — `list error` y `metadata error` abortan sin leer
- [ ] VC-31 pasa — exactamente un request al recurso que falla
- [ ] VC-32 pasa — lectura colgada → `timeout after 30s without data`
- [ ] VC-33 pasa — listado o metadata colgados → abortan con el mismo detalle
- [ ] VC-34 pasa — una lectura lenta pero continua no se corta
- [ ] VC-50 pasa — ≤ 35 s desde el request colgado hasta el exit

**Notas de implementación**

- **El timeout va en el transporte HTTP**, no en cada llamada:
  `ResponseHeaderTimeout` de 30 s y un wrapper del cuerpo que reinicia un timer con
  cada lectura. Así cubre listado, metadata y contenido de la misma forma, y un
  deadline total no cortaría una lectura lenta pero continua (FR-34).
- **Bucket vs. objeto inexistente sin `storage.buckets.get`.** `lectora` solo tiene
  `objectViewer`, así que no puede consultar el bucket. El `404` de metadata se
  distingue por el cuerpo del error de GCS (`No such object` vs. bucket
  inexistente). Se valida contra GCS real al empezar la iteración, y el servidor de
  prueba replica ese cuerpo. Si GCS no los distingue, la alternativa es un listado
  con `maxResults=1` solo en el camino de error. La spec no lo prohíbe, porque FR-12
  habla del caso exitoso. Se registra en `DECISIONS.md`.
- **VC-32, VC-33 y VC-34 tardan entre 30 y 60 s cada uno.** Corren en paralelo con
  `t.Parallel()` para que el gate no pase de unos pocos minutos.

---

## Iteración 4 — Concurrencia, progreso y parser estricto

**Objetivo:** que la herramienta sea rápida y usable en buckets con muchos objetos,
y que el parser no acepte nada fuera de la sintaxis de la spec.

**Alcance**

- `--concurrency <N>` (1 a 32, default 4); el pool pasa de 1 a N workers.
- Salida entrelazada entre objetos, con líneas enteras y orden dentro de cada objeto.
- Progreso `X/total objects processed` solo si stderr es una TTY, borrado antes de
  cada aviso y al final.
- Muerte silenciosa por `SIGPIPE` cuando se cierra stdout, sin arrancar más lecturas.
- Parser estricto: flags combinados o con `=`, flags repetidos, un solo error de uso
  reportado.
- El servidor de prueba agrega el contador de lecturas simultáneas.

**Criterios de éxito**

- [ ] VC-24 pasa — `-in`, `--max=5`, etc. → `2`, cumple el chequeo P
- [ ] VC-25 pasa — flag repetido → `2`, cumple el chequeo P
- [ ] VC-26 pasa — máximo de lecturas simultáneas exactamente `4`, `1` y `8`
- [ ] VC-27 pasa — varios errores de uso → una sola línea
- [ ] VC-28 pasa — 10 corridas con `--concurrency 32` idénticas a la secuencial
- [ ] VC-36 pasa — `| head -1` → `141`, stderr vacío, menos de 100 lecturas
- [ ] VC-37 pasa — progreso en TTY (Linux, `script` de util-linux)
- [ ] VC-38 pasa — sin TTY no hay progreso
- [ ] **Todos los VCs de las iteraciones 1 a 3 siguen pasando**, ahora con
  concurrencia 4 por defecto

**Nota de regresión:** el default cambia de 1 a 4 workers. Los VCs de salida ya
comparan líneas sin importar el orden entre objetos (convención de la spec), pero
recién ahora eso se ejercita de verdad. VC-19 (corte de `-l`), VC-29 y VC-32 (fallo
de un objeto con otros en paralelo) son los más sensibles.

**Nota de plataforma:** VC-37 exige Linux. En la laptop (macOS) se corre dentro de
un contenedor `golang` con `util-linux`. El resto de la suite es portable.

---

## Iteración 5 — NFRs y contrato de scripting

**Objetivo:** medir contra los umbrales ya escritos en la spec. No se ajusta ningún
umbral para que pase.

**Alcance**

- Bucket `$P` en `us-east1` con los datasets exactos de NFR-1 y NFR-2 y el objeto
  `bw/100mb.bin`, generados y subidos por un script versionado (`tools/gendata`).
  Es un bucket aparte para que sus 1000+ objetos no disparen el guardrail en los VCs
  que buscan en la raíz de `$B`.
- Script de medición de NFR-1: ancho de banda, tres corridas, mediana. Si el ancho
  de banda da menos de 50 Mbps, la corrida no es válida y se repite en otra red.
- Medición de pico de RSS de NFR-2 con `/usr/bin/time`.
- Auditoría de BR-9 sobre todos los casos de falla ya cubiertos.

**Criterios de éxito**

- [ ] VC-47 pasa — stdout vacío, `gcsgrep: ` al inicio y sin `panic:`, `goroutine `
  ni `.go:` en todos los casos de falla
- [ ] VC-48 pasa — mediana < 180 s con 1000 objetos de `$P` y ≥ 50 Mbps medidos
- [ ] VC-49 pasa — pico ≤ 100 MiB con 1 GB y ≤ 20 MiB más que con 10 MB
- [ ] **Los 50 VCs pasan en una misma corrida**, y `gcsgrep-cobertura-vc.md`
  queda completo

**Si VC-48 no pasa:** el umbral ya contempla la peor latencia medida a `us-east1`
(0,59 s por objeto → ~150 s). Si igual no pasa, el candidato es la herramienta, no la
red: conexiones que no se reusan, o listado y lectura que no se solapan con la
latencia. La corrección va acá, contra el umbral fijo.

**Si VC-49 no pasa:** revisar que el lector de líneas reuse su buffer y que el
tamaño de lectura del cliente no escale con el objeto.

---

## Lo que quedó afuera del plan entero

Esto no es "todavía no lo hicimos". Es alcance que la spec deja **fuera de la v1** y
que vive acá para que no reaparezca en cada conversación. Si alguna vez entra, entra
por una revisión de spec, no por el plan.

| Idea | Decisión |
|---|---|
| Descomprimir `.gz` al vuelo | Diferido post-v1. Hoy se saltea como no-texto (BR-7) |
| Límite por bytes leídos | Diferido post-v1, como complemento del tope por objetos |
| Service account por flag | Diferido post-v1. En la v1, solo ADC |
| Reintentos con backoff y timeout configurable | Diferido post-v1 |
| `-v`, `-w`, `-o`, contexto (`-A`/`-B`/`-C`), `--include` | Diferido post-v1 |
| Salida JSON | Diferido post-v1 |
| Agrupar la salida por objeto | Descartado: rompe memoria acotada o concurrencia |
| Color | Descartado: ensucia la salida para scripts (BR-9) |
| `-r` | Descartado: la recursión ya la define la `/` final |
| Pedir confirmación antes de una corrida cara | Descartado: rompe el uso desde scripts |
| Convertir codificaciones distintas de UTF-8 | Fuera de v1; BR-6 clasifica por la muestra inicial de hasta 512 bytes |
| S3, Azure, interfaz web, API o librería | Descartado en v1 |
| Consistencia ante objetos que cambian durante la lectura | Riesgo conocido, aceptado |

## Cuándo este plan se modifica y cuándo se vuelve a la spec

- **Se actualiza el plan** si cambia el *cómo* o el orden: una pieza compartida que
  conviene adelantar, un VC que resulta depender de algo de otra iteración.
- **Se vuelve a la spec** si cambia el *qué*. Los dos candidatos identificados son el
  transcoding de GCS (Iteración 2) y la distinción bucket/objeto inexistente
  (Iteración 3).
- **No se reescribe** el registro de una iteración terminada. Lo aprendido va a
  `CONTEXT.md`, `DECISIONS.md` y a las iteraciones futuras de este plan.

## Qué sigue

Implementar **solo la Iteración 1** y registrar su evidencia en
`gcsgrep-cobertura-vc.md`.
