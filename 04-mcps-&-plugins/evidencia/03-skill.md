# 03 · Skill `write-vcs-con-bucket`

Corrida del 2026-10-13 en una sesión de Claude Code abierta en `04-mcps-&-plugins/` con
`GOOGLE_APPLICATION_CREDENTIALS` (la key de `lectora`) y `GCSGREP_BUCKET` en el entorno.
Resultado: [`sdd/invert-match-spec.md`](../sdd/invert-match-spec.md), `Estado: revisada`.

## Pedido

> especificá `-v` para gcsgrep

## 1 · Freno

Las tres condiciones del skill, antes de leer nada:

| Condición | Resultado |
|---|---|
| Tool `mcp__gcs__read_object_metadata` disponible | Sí |
| `printenv GCSGREP_BUCKET` | `sdd-fardenghi-itba` |
| `jq -r .client_email "$GOOGLE_APPLICATION_CREDENTIALS"` | `lectora@sat-spec-driven-development.iam.gserviceaccount.com` |

## 2 · Hallazgo: el server no devuelve `generation`

El plan y el skill daban por hecho que `read_object_metadata` devolvía `generation`. Los
dos lectores (25 objetos) y una llamada directa de la sesión sobre `logs/app/api.log`
coincidieron:

```json
{
  "bucket": "sdd-fardenghi-itba",
  "object": "logs/app/api.log",
  "size": 906,
  "content_type": "text/plain",
  "time_created": "2026-09-22T23:20:37.359Z",
  "updated": "2026-09-22T23:20:37.359Z",
  "storage_class": "STANDARD"
}
```

El skill se detuvo en lugar de inventar la columna. Decisión del usuario: usar `updated`
como identidad del objeto. Cambia con cada re-subida, y también con un cambio de
metadata: puede dar un falso "vencida", nunca un falso "al día". Se ajustaron
`SKILL.md`, `lector-fixtures.md` y `sdd/tp4-plan.md`; no se usó `gcloud` para
reconstruir la generation.

## 3 · Lectura por el subagent

`lector-fixtures` corrió cuatro veces (patrón `timeout`, bucket completo, 25 objetos): dos
con `selección: match` y dos con `selección: sin-match`, las dos últimas ya con la columna
`updated`. Tools que llamó: solo `list_objects`, `read_object_metadata` y
`read_object_content`. La sesión principal no leyó contenido; llamó
`read_object_metadata` para el hallazgo de arriba y para el cierre.

Límites de la lectura:

- `edge-cases/long_line_exceeds_1mb.log` no se pudo leer (la tool devolvió "exceeds maximum
  allowed tokens"). Ningún VC se apoya en él y la spec lo dice en Fuera.
- El lector clasifica "texto" por metadata; la spec base lo decide por contenido (BR-6).
  Por eso los VCs de `.gz` y binario se apoyan en BR-5/BR-7 y no en esa clasificación.

## 4 · La línea que solo pudo venir del bucket

`api.log` tiene 15 líneas. La línea 7 es la que muestra que el bucket aportó algo que no
estaba en la spec base: contiene `TIMEOUT` en mayúsculas, así que `-v timeout` (que
distingue mayúsculas) la imprime.

> **`gs://sdd-fardenghi-itba/logs/app/api.log:7:2026-09-22 10:14:03 WARN Connection TIMEOUT to redis cache replica-0`**

Está en VC-1 y VC-2, y VC-3 (`-i`) la excluye. Los VCs de la spec base nunca la citaron.
La línea 10 (`Database Timeout …`) es la misma lección con mayúscula inicial.

## 5 · Procedencia (de la spec)

Leídos el 2026-10-13T03:06Z, server `@google-cloud/storage-mcp` 0.6.0 sin
`--enable-destructive-tools`, cuenta `lectora@sat-spec-driven-development.iam.gserviceaccount.com`.

| Objeto | Updated | VCs |
|---|---|---|
| `logs/app/api.log` | `2026-09-22T23:20:37.359Z` | VC-1, VC-2, VC-3, VC-4, VC-12, VC-BR-1, VC-INV-2, VC-NFR-1 |
| `logs/app/worker.log` | `2026-09-22T23:20:37.370Z` | VC-12 |
| `logs/db/postgres.log` | `2026-09-22T23:20:37.359Z` | VC-12 |
| `logs/db/redis.log` | `2026-09-22T23:20:37.379Z` | VC-12 |
| `logs/server.log` | `2026-09-22T23:20:37.293Z` | VC-12 |
| `data/subdata/deep_nested.txt` | `2026-09-22T23:20:37.366Z` | VC-5 |
| `data/windows_crlf.txt` | `2026-09-22T23:20:37.340Z` | VC-6, VC-8b |
| `data/sin_salto_final.txt` | `2026-09-25T21:15:56.317Z` | VC-7 |
| `edge-cases/empty.txt` | `2026-09-22T23:20:37.360Z` | VC-8a |
| `logs/archive/old_logs.log.gz` | `2026-09-22T23:20:37.361Z` | VC-9, VC-12 |
| `edge-cases/binary_nullbyte.bin` | `2026-09-22T23:20:37.357Z` | VC-10 |
| `fixtures/hostil/notas.log` | `2026-10-12T21:34:12.699Z` | VC-11 |

## 6 · Cierre

`read_object_metadata` de los 12 objetos: todos los `updated` coinciden con lo anotado.
La procedencia vale.

## 7 · Encadenado con el TP3

- `vc-coverage.sh sdd/invert-match-spec.md` → `OK, 20 requisitos con su VC.`
- `review-spec`, tres vueltas:

| Vuelta | Veredicto | Qué corrigió |
|---|---|---|
| 1 | HUECOS (6) | Borde "todas las líneas matchean" (FR-8b), prefijo mixto (FR-12), Fuera ampliado, INV-4 de alcance, 23 nombres de tests con paquete, VC-BR-1 con `-i`, Convenciones, condiciones de NFR-1 |
| 2 | HUECOS (2) | VC-NFR-1 sin chequear el código de salida; la spec no decidía cómo llega `Invert` a `Scan` (queda como campo de `Scanner`, la firma de `Scan` no cambia) |
| 3 | **LISTA** | Ninguno |

## 8 · Prueba real: re-subida y cierre fallido

Ejecutada el 2026-10-13T18:02:48.587903+00:00, después de renovar el login de la cuenta dueña
`fardenghi@itba.edu.ar`. Se descargó la generación existente a un archivo temporal,
se verificó su MD5 y se re-subió al mismo objeto con `--if-generation-match`
igual a la generación anterior. Se preservaron contenido y tipo `text/plain`.
No se leyó ni imprimió el contenido en la sesión principal.

| Campo | Antes | Después |
|---|---|---|
| Objeto | `logs/app/api.log` | `logs/app/api.log` |
| Generation | `1790119237354314` | `1791914566720157` |
| Updated (MCP) | `2026-09-22T23:20:37.359Z` | `2026-10-13T18:02:46.750Z` |
| MD5 (base64) | `vahZ/f5lsfZILBzK2ouO7g==` | `vahZ/f5lsfZILBzK2ouO7g==` |
| CRC32C (base64) | `kzgGYQ==` | `kzgGYQ==` |
| Tamaño | `906` | `906` |

Comandos de la renovación (cuenta dueña, fuera del agente MCP):

```sh
gcloud storage cp gs://sdd-fardenghi-itba/logs/app/api.log#1790119237354314 <archivo-temporal>/api.log --account=fardenghi@itba.edu.ar
gcloud storage cp <archivo-temporal>/api.log gs://sdd-fardenghi-itba/logs/app/api.log --account=fardenghi@itba.edu.ar --if-generation-match=1790119237354314 --content-type=text/plain --storage-class=STANDARD
```

Después se invocó el skill del plugin sobre la spec original, pidiendo solamente
el cierre. Confirmó `lectora`, consultó la metadata real por MCP y editó la
cabecera a `Estado: vencida`; no releyó contenido ni reescribió VCs en esa corrida.
Los otros 11 objetos mantuvieron el updated registrado.

Extracto de la respuesta final de esa corrida:

```text
No se pudo cerrar la spec: `logs/app/api.log` tiene otro `updated`, así que la marqué como `Estado: vencida` y me detuve.

**Precondiciones**
- La tool `read_object_metadata` está disponible.
- El proyecto es `sat-spec-driven-development` y el bucket `sdd-fardenghi-itba`.
- `identidad.py` devolvió la cuenta `lectora@sat-spec-driven-development.iam.gserviceaccount.com`.

**Comparación con la metadata real (MCP)**

| Objeto | `updated` anotado | `updated` actual | Resultado |
|---|---|---|---|
| `logs/app/api.log` | `2026-09-22T23:20:37.359Z` | `2026-10-13T18:02:46.750Z` | **no coincide** |
| Los otros 11 objetos de la tabla de procedencia | — | — | coinciden |

El tamaño sigue siendo 906 bytes y `time_created` es igual al nuevo `updated`, lo que cuadra con una re-subida. Igual, un `updated` distinto cuenta como otro objeto hasta que se relea, aunque el contenido sea el mismo.

**Cambios en `invert-match-spec.md`** (el único archivo que edité)
- La cabecera pasó de `Estado: revisada` a `Estado: vencida`.
- En la sección de procedencia agregué una nota del cierre fallido, con los dos `updated` y la cuenta que leyó.

Tal como pediste, no leí contenido, no reescribí VCs, no corrí `vc-coverage.sh` ni la revisión, y no lancé una relectura. Para dejar la spec al día hay que releer `logs/app/api.log` con `lector-fixtures` (normal y con `selección: sin-match`). Después hay que rehacer los VCs que dependen de ese objeto (VC-1 a VC-4, VC-12, VC-BR-1, VC-INV-2 y VC-NFR-1) y su fila de procedencia.
```

La relectura que restaura la spec se registra en la sección 10.

## 9 · Prueba adicional de procedencia vencida (copia local)

Ejecutada el 2026-10-13T17:56:23.458118+00:00, con el plugin 1.0.0
y la misma cuenta lectora. La cuenta dueña pidió reautenticación, por lo que
**no se re-subió ningún objeto**. Se probó el cierre sobre una copia temporal de
la spec: solo la fila de procedencia de `logs/app/api.log` se cambió a
`2000-01-01T00:00:00.000Z`. En esa corrida la spec original del repo se conservó revisada y al día.

Esta prueba verifica el rechazo de una procedencia distinta de la metadata real;
es complementaria a la prueba real de cambiar un objeto y releerlo (secciones 8 y 10).

Se invocó el skill `sdd-toolkit:write-vcs-con-bucket` con
`cerrar ./invert-match-spec.md`. El helper `identidad.py` confirmó la cuenta,
el skill consultó `read_object_metadata` de los 12 objetos, encontró el mismatch
y cambió la cabecera de la copia a `Estado: vencida`, sin rehacer los VCs.

### Extractos de las llamadas reales

```json
{
  "tool": "Skill",
  "input": {
    "skill": "sdd-toolkit:write-vcs-con-bucket",
    "args": "cerrar ./invert-match-spec.md"
  }
}
```

```json
{
  "tool": "Bash",
  "input": {
    "command": "python3 \"<checkout>/04-mcps-&-plugins/plugin/sdd-toolkit/skills/write-vcs-con-bucket/scripts/identidad.py\"",
    "description": "Print the service account email in use"
  }
}
```

```json
{
  "tool": "mcp__plugin_sdd-toolkit_gcs__read_object_metadata",
  "input": {
    "bucket_name": "sdd-fardenghi-itba",
    "object_name": "logs/app/api.log"
  }
}
```

```json
{
  "bucket": "sdd-fardenghi-itba",
  "object": "logs/app/api.log",
  "size": 906,
  "content_type": "text/plain",
  "time_created": "2026-09-22T23:20:37.359Z",
  "updated": "2026-09-22T23:20:37.359Z",
  "storage_class": "STANDARD"
}
```

```json
{
  "tool": "Edit",
  "input": {
    "replace_all": false,
    "file_path": "<carpeta-temporal>/invert-match-spec.md",
    "old_string": "> Estado: revisada · Base:",
    "new_string": "> Estado: vencida · Base:"
  }
}
```

```json
{
  "tool": "Edit",
  "input": {
    "replace_all": false,
    "file_path": "<carpeta-temporal>/invert-match-spec.md",
    "old_string": "Cierre del 2026-10-13: se releyó `read_object_metadata` de los 12 objetos de la tabla y todos los `updated` coinciden con lo anotado.",
    "new_string": "Cierre del 2026-10-13 (revalidación): **falla**. `logs/app/api.log` tiene anotado `2000-01-01T00:00:00.000Z` y `read_object_metadata` devuelve `2026-09-22T23:20:37.359Z`; los otros 11 objetos coinciden. Hay que releer ese objeto y rehacer sus VCs (VC-1, VC-2, VC-3, VC-4, VC-12, VC-BR-1, VC-INV-2, VC-NFR-1)."
  }
}
```

Resultado observado en la copia:

```text
> Estado: vencida · Base: 01-greenfield-gcsgrep@52596d7 · Spec base: `01-greenfield-gcsgrep/sdd/gcsgrep-spec.md`
```

No hubo lecturas de contenido, escrituras en GCS ni invocaciones al lector o
al revisor después del mismatch. Un intento inicial de inspección del directorio
por Bash fue rechazado por los permisos de la sesión; luego se usaron el skill
y su helper de identidad. No se incluyeron keys ni sus rutas en esta evidencia.

## 10 · Relectura real y procedencia nativa

Después del cierre fallido de la sección 8, se volvió a invocar el skill con la
spec original. `lector-fixtures` leyó `logs/app/api.log` dos veces, con
`timeout` y selecciones `match` / `sin-match`; ambas confirmaron las 15 líneas
y sus números. El contenido esperado se conservó y las anclas se actualizaron
al nuevo updated y a la generation actual.

La sesión principal ejecutó `procedencia.py` para los 12 objetos de la tabla:
`gcloud storage objects describe --raw`, exclusivamente metadata, con
`CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE` fijado a la credencial lectora del MCP.
El helper descarta metadata libre, muestra solo identidad y hashes nativos y
normaliza el formato de fecha de gcloud al del MCP sin perder precisión.
Los 12 updated coincidieron con `read_object_metadata`.

La revisión detectó dependencias de listados completos en VC-5, VC-12 y VC-BR-2;
se revalidaron esos tres prefijos por el lector sin leer su contenido y se
registraron en la spec. El skill ahora comprueba también esos listados al cerrar.
Se precisaron códigos de salida, stderr, bordes del parser e invariantes de la
suite e2e durante la revisión; no se implementó `-v`.

### Línea de base e2e medida

Ejecutada fuera de Claude, con las rutas de las credenciales solo en variables
de entorno, sin mostrarlas ni almacenarlas en el repo:

```console
$ cd ../01-greenfield-gcsgrep
$ go test ./e2e -json
exit: 0
21 PASS, 0 FAIL, 0 SKIP
```

Fecha: 2026-10-13T18:15:35.450225+00:00. Cuenta de lectura: `lectora`; pruebas negativas:
`sin-acceso`. Tests de nivel superior en PASS:

- `TestVC01`
- `TestVC03`
- `TestVC04`
- `TestVC05`
- `TestVC06`
- `TestVC07a`
- `TestVC08`
- `TestVC09a`
- `TestVC09b`
- `TestVC12Parcial`
- `TestVC14`
- `TestVC15b`
- `TestVC21`
- `TestVC22`
- `TestVC23`
- `TestVC35`
- `TestVC39`
- `TestVC40`
- `TestVC41Parcial`
- `TestVC42Parcial`
- `TestVC46`

## 11 · Generation distinta aunque updated y checksum coincidan

Se hizo otra prueba sobre una copia temporal de la spec con procedencia nativa:
solo la generation de `logs/app/api.log` en la tabla se cambió a `1`. El updated
y MD5 se dejaron correctos. El skill consultó metadata MCP y nativa con la
cuenta lectora y marcó la copia vencida por la generation, sin leer contenido
ni tocar la spec original. Respuesta observada:

```text
La spec no pasó el cierre. Marqué `./invert-match-spec.md` como **`Estado: vencida`** y me detuve ahí.

- **Cuenta:** `identidad.py` devolvió `lectora@sat-spec-driven-development.iam.gserviceaccount.com`.
- **Objeto:** `logs/app/api.log`, con bucket `sdd-fardenghi-itba`.
- **Campo que no coincide: `generation`**
  - En la tabla de procedencia figura `1`.
  - `procedencia.py` devolvió `1791914566720157`.
- **Lo que sí coincide:**
  - `updated` es `2026-10-13T18:02:46.750Z` en lo anotado, en la metadata MCP y en `procedencia.py`.
  - El MD5 nativo es `vahZ/f5lsfZILBzK2ouO7g==` y coincide con el anotado.

En el cuerpo de los VCs, la generation que se cita sigue siendo `1791914566720157`. La tabla de procedencia es lo único que quedó con la vieja.

Lo único que cambié en la copia fue la línea de estado de la cabecera. No revisé otros objetos, no releí contenido, no listé prefijos, no rehice VCs y no toqué el repo original. Para volver a cerrar la spec hay que releer `api.log` con el lector y actualizar su fila de procedencia (paso 2 del skill).
```

### Salida exacta de VC-12

Para verificar texto y distribución por objeto, se volvió a leer `logs/` con
`lector-fixtures`, `selección: sin-match` y `texto_completo: sí`. El lector usó
solo `list_objects`, `read_object_metadata` y `read_object_content`; leyó los
cinco objetos de texto y salteó el gzip. La sesión principal no llamó a lectura
de contenido. Las 32 líneas seleccionadas completas se guardaron en
`test-fixtures/invert-logs-esperado.txt` (13/5/4/6/4 por objeto). El VC compara
su salida contra ese fixture ordenado, no solo la cantidad de líneas.

## 12 · Prerrequisitos conservados por decisión del equipo

La revisión de la base encontró que `52596d7` solo implementa la Iteración 1:
no clasifica gzip/binarios ni traduce objetos ausentes a `not found`. El equipo
decidió mantener los VCs de esos casos y declarar como prerrequisitos las
iteraciones 2 y 3 del plan del TP1. No se implementaron esas iteraciones en el
TP4. La spec diferencia base explorada de la base de implementación futura
`INVERT_MATCH_BASE`; antes de implementar `-v` se completan los prerrequisitos,
se registra el hash nuevo y se vuelve a medir la línea de base.

Para corroborar la diferencia de errores se compiló el gcsgrep actual en una
carpeta temporal y se ejecutó con la cuenta lectora:

```console
$ gcsgrep timeout gs://sdd-fardenghi-itba/data/no-existe-vc-br-2.txt
gcsgrep: gs://sdd-fardenghi-itba/data/no-existe-vc-br-2.txt: metadata error: storage: object doesn't exist
exit: 2
stdout: vacío
```

La futura salida `not found` de VC-BR-2 depende expresamente de la Iteración 3.
Los VCs de gzip/binario y del prefijo mixto dependen de la Iteración 2.
El veredicto de la spec habilita planificar y conserva el bloqueo de implementación
hasta completar esos prerrequisitos; no afirma que los VCs futuros ya pasan.
