# 03 · Skill `write-vcs-con-bucket`

Corrida del 2026-10-13 en una sesión de Claude Code abierta en `04-mcps-&-plugins/` con
`GOOGLE_APPLICATION_CREDENTIALS` (la key de `lectora`) y `GCSGREP_BUCKET` en el entorno.
Resultado: [`sdd/invert-match-spec.md`](../sdd/invert-match-spec.md), 29 requisitos con
su VC, en `Estado: borrador` con la implementación bloqueada por prerrequisitos
(sección 7).

| Dato | Valor |
|---|---|
| Bucket | `gs://sdd-fardenghi-itba`, proyecto `sat-spec-driven-development` |
| Server | `@google-cloud/storage-mcp` 0.6.0, sin `--enable-destructive-tools` |
| Cuenta | `lectora@sat-spec-driven-development.iam.gserviceaccount.com` |
| Lectura | 2026-10-13T03:06Z; `logs/app/api.log` releído el 2026-10-13T18:09Z |
| Objetos citados en VCs | 12, más 3 listados de prefijo (sección 5) |
| Cierre | 2026-10-13T18:09Z: `updated`, `generation`, MD5 y listados coinciden |

## Pedido

> especificá `-v` para gcsgrep

## 1 · Freno

Antes de leer nada, el skill comprueba sus tres condiciones:

| Condición | Resultado |
|---|---|
| Tool `mcp__gcs__read_object_metadata` disponible | Sí |
| `printenv GCSGREP_BUCKET` | `sdd-fardenghi-itba` |
| Cuenta de la key (`identidad.py`, que imprime solo `client_email`; en la primera corrida, `jq -r .client_email`) | `lectora@sat-spec-driven-development.iam.gserviceaccount.com` |

Si falta alguna, el skill se detiene sin escribir VCs. La sección 12 muestra ese freno
en una sesión sin el server.

## 2 · Lectura por el subagent

`lector-fixtures` corrió cuatro veces sobre el bucket completo (patrón `timeout`,
25 objetos): dos con `selección: match` y dos con `selección: sin-match`. Tools que
llamó: solo `list_objects`, `read_object_metadata` y `read_object_content`. La sesión
principal no leyó contenido; llamó `read_object_metadata` para la identidad de los
objetos y para el cierre.

Límites de la lectura:

- `edge-cases/long_line_exceeds_1mb.log` no se pudo leer (la tool devolvió "exceeds maximum
  allowed tokens"). Ningún VC se apoya en él y la spec lo dice en Fuera.
- El lector clasifica "texto" por metadata; la spec base lo decide por contenido (BR-6).
  Por eso los VCs de `.gz` y binario se apoyan en BR-5/BR-7 y no en esa clasificación.

## 3 · Identidad de cada objeto: `updated` y `generation`

`read_object_metadata` del server 0.6.0 no devuelve `generation` ni checksums. Respuesta
real sobre `logs/app/api.log`, antes de su re-subida (sección 8):

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

Por eso la procedencia combina dos fuentes con la misma cuenta lectora:

- `updated`, de `read_object_metadata` por MCP.
- `generation` y MD5 nativos, del helper `procedencia.py`: ejecuta solo
  `gcloud storage objects describe --raw` con `CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE`
  fijado a la credencial del server, sin leer contenido ni usar la cuenta activa.

El `updated` de las dos fuentes tiene que coincidir completo y como texto; si no, se
repite la lectura. Cualquier diferencia de `updated`, `generation` o MD5 al cerrar vence
la spec.

## 4 · La línea que solo pudo venir del bucket

`api.log` tiene 15 líneas. La línea 7 es la que muestra que el bucket aportó algo que no
estaba en la spec base: contiene `TIMEOUT` en mayúsculas, así que `-v timeout` (que
distingue mayúsculas) la imprime.

> **`gs://sdd-fardenghi-itba/logs/app/api.log:7:2026-09-22 10:14:03 WARN Connection TIMEOUT to redis cache replica-0`**

Está en VC-1 y VC-2, y VC-3 (`-i`) la excluye. Los VCs de la spec base nunca la citaron.
La línea 10 (`Database Timeout …`) es la misma lección con mayúscula inicial.

## 5 · Procedencia

Copiada de la spec. `Updated` es el de `read_object_metadata` y coincide con el de
`procedencia.py` en los 12 objetos.

| Objeto | Generation | MD5 (base64) | Updated | VCs |
|---|---|---|---|---|
| `logs/app/api.log` | `1791914566720157` | `vahZ/f5lsfZILBzK2ouO7g==` | `2026-10-13T18:02:46.750Z` | VC-1, VC-2, VC-3, VC-4, VC-12, VC-13a, VC-BR-1, VC-BR-3, VC-INV-2, VC-NFR-1 |
| `logs/app/worker.log` | `1790119237365907` | `yym80Z+JYrLZcWMxPyQYkw==` | `2026-09-22T23:20:37.370Z` | VC-12 |
| `logs/db/postgres.log` | `1790119237355295` | `n1T//eblOXT+lZeMIESFcQ==` | `2026-09-22T23:20:37.359Z` | VC-12 |
| `logs/db/redis.log` | `1790119237374465` | `nVVau7LVd8REL5zBG3RsfQ==` | `2026-09-22T23:20:37.379Z` | VC-12 |
| `logs/server.log` | `1790119237289998` | `yyNy+vY/pd19JRVnumPu5A==` | `2026-09-22T23:20:37.293Z` | VC-12 |
| `data/subdata/deep_nested.txt` | `1790119237360468` | `E/k4O/agK9uThiCXsJLGgg==` | `2026-09-22T23:20:37.366Z` | VC-5 |
| `data/windows_crlf.txt` | `1790119237334512` | `P2aJbfIH5vDpfCKwckn+/w==` | `2026-09-22T23:20:37.340Z` | VC-6, VC-8b |
| `data/sin_salto_final.txt` | `1790370956298722` | `6oiUMenBGDDgMzDx5GTmNA==` | `2026-09-25T21:15:56.317Z` | VC-7 |
| `edge-cases/empty.txt` | `1790119237353571` | `1B2M2Y8AsgTpgAmY7PhCfg==` | `2026-09-22T23:20:37.360Z` | VC-8a |
| `logs/archive/old_logs.log.gz` | `1790119237355510` | `tq+KLeAq/wr4Yfg/KgHSMQ==` | `2026-09-22T23:20:37.361Z` | VC-9, VC-12 |
| `edge-cases/binary_nullbyte.bin` | `1790119237353687` | `D1/XMni2zRc72Q9hB0VuOA==` | `2026-09-22T23:20:37.357Z` | VC-10 |
| `fixtures/hostil/notas.log` | `1791840852693528` | `ETcxgjogkWyjVuw2sPQ+sg==` | `2026-10-12T21:34:12.699Z` | VC-11 |

Listados de prefijo (2026-10-13T18:09Z, `list_objects` vía `lector-fixtures`):

| Prefijo | Objetos | VCs |
|---|---|---|
| `logs/` | 6 | VC-12 |
| `data/subdata/` | 1 | VC-5 |
| `data/no-existe-vc-br-2` | 0 | VC-BR-2 |

## 6 · Encadenado con el TP3

- `vc-coverage.sh sdd/invert-match-spec.md` → `OK, 29 requisitos con su VC.`
- `review-spec` revisó la primera versión de la spec (20 requisitos) en tres vueltas:

| Vuelta | Veredicto | Qué corrigió |
|---|---|---|
| 1 | HUECOS (6) | Borde "todas las líneas matchean" (FR-8b), prefijo mixto (FR-12), Fuera ampliado, INV-4 de alcance, 23 nombres de tests con paquete, VC-BR-1 con `-i`, Convenciones, condiciones de NFR-1 |
| 2 | HUECOS (2) | VC-NFR-1 sin chequear el código de salida; la spec no decidía cómo llega `Invert` a `Scan` (queda como campo de `Scanner`, la firma de `Scan` no cambia) |
| 3 | **LISTA** | Ninguno |

Con ese `LISTA` la spec quedó en `Estado: revisada`, y sobre esa versión corrieron las
pruebas de cierre de las secciones 8, 9 y 11. Las revisiones siguientes agregaron los
listados de prefijo, los códigos de salida y stderr, los bordes del parser y las
invariantes de la suite e2e (29 requisitos), y encontraron que la base no implementa
gzip/binario ni `not found` (sección 7). Por eso la spec entregada está en
`Estado: borrador`: su `LISTA` final queda pendiente de esos prerrequisitos, como dice
su sección "Pendiente de revisión".

## 7 · Prerrequisitos de implementación

`52596d7` solo implementa la Iteración 1 del TP1: no clasifica gzip/binarios ni traduce
objetos ausentes a `not found`. El equipo decidió mantener los VCs de esos casos y
declarar como prerrequisitos las iteraciones 2 y 3 del plan del TP1, que no forman parte
del TP4. La spec diferencia la base explorada de la base de implementación futura
`INVERT_MATCH_BASE`; antes de implementar `-v` se completan los prerrequisitos, se
registra el hash nuevo y se vuelve a medir la línea de base.

Para corroborar la diferencia de errores se compiló el gcsgrep actual en una carpeta
temporal y se ejecutó con la cuenta lectora:

```console
$ gcsgrep timeout gs://sdd-fardenghi-itba/data/no-existe-vc-br-2.txt
gcsgrep: gs://sdd-fardenghi-itba/data/no-existe-vc-br-2.txt: metadata error: storage: object doesn't exist
exit: 2
stdout: vacío
```

La futura salida `not found` de VC-BR-2 depende de la Iteración 3. Los VCs de
gzip/binario y del prefijo mixto dependen de la Iteración 2. El bloqueo de
implementación se mantiene hasta completar esos prerrequisitos; la spec no afirma que
los VCs futuros ya pasan.

## 8 · Cierre con un objeto re-subido

Prueba del 2026-10-13T18:02:48Z. Con la cuenta dueña, fuera del agente MCP, se descargó
la generación existente de `logs/app/api.log` a un archivo temporal, se verificó su MD5
y se re-subió al mismo objeto con `--if-generation-match` igual a la generación
anterior, preservando contenido y tipo `text/plain`. La sesión principal no leyó ni
imprimió el contenido.

| Campo | Antes | Después |
|---|---|---|
| Objeto | `logs/app/api.log` | `logs/app/api.log` |
| Generation | `1790119237354314` | `1791914566720157` |
| Updated (MCP) | `2026-09-22T23:20:37.359Z` | `2026-10-13T18:02:46.750Z` |
| MD5 (base64) | `vahZ/f5lsfZILBzK2ouO7g==` | `vahZ/f5lsfZILBzK2ouO7g==` |
| CRC32C (base64) | `kzgGYQ==` | `kzgGYQ==` |
| Tamaño | `906` | `906` |

```sh
gcloud storage cp gs://sdd-fardenghi-itba/logs/app/api.log#1790119237354314 <archivo-temporal>/api.log --account=fardenghi@itba.edu.ar
gcloud storage cp <archivo-temporal>/api.log gs://sdd-fardenghi-itba/logs/app/api.log --account=fardenghi@itba.edu.ar --if-generation-match=1790119237354314 --content-type=text/plain --storage-class=STANDARD
```

Después se invocó el skill del plugin pidiendo solo el cierre. Confirmó `lectora`,
consultó la metadata real por MCP y cambió la cabecera a `Estado: vencida`, sin releer
contenido ni reescribir VCs. Los otros 11 objetos mantuvieron el `updated` registrado.
Extracto de su respuesta:

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

La relectura que deja la spec al día está en la sección 10.

## 9 · Cierre sobre una copia con `updated` distinto

Prueba del 2026-10-13T17:56:23Z, con el plugin 1.0.0 y la cuenta lectora, sin re-subir
ningún objeto. Sobre una copia temporal de la spec, solo la fila de procedencia de
`logs/app/api.log` se cambió a `2000-01-01T00:00:00.000Z`. Verifica el rechazo de una
procedencia que no coincide con la metadata real; complementa la prueba con un objeto
re-subido (secciones 8 y 10).

Se invocó `sdd-toolkit:write-vcs-con-bucket` con `cerrar ./invert-match-spec.md`.
`identidad.py` confirmó la cuenta, el skill consultó `read_object_metadata` de los 12
objetos, encontró la diferencia y cambió la cabecera de la copia a `Estado: vencida`,
sin rehacer VCs.

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

Resultado en la copia:

```text
> Estado: vencida · Base: 01-greenfield-gcsgrep@52596d7 · Spec base: `01-greenfield-gcsgrep/sdd/gcsgrep-spec.md`
```

No hubo lecturas de contenido, escrituras en GCS ni invocaciones al lector o al revisor
después de la diferencia. Un intento de inspeccionar el directorio por Bash fue
rechazado por los permisos de la sesión; el resto se hizo con el skill y su helper de
identidad. No se incluyeron keys ni sus rutas.

## 10 · Relectura y procedencia nativa

Después del cierre fallido de la sección 8 se volvió a invocar el skill con la spec
original. `lector-fixtures` leyó `logs/app/api.log` dos veces, con `timeout` y
selecciones `match` / `sin-match`; las dos confirmaron las 15 líneas y sus números. El
contenido esperado de los VCs no cambió; sus anclas pasaron al nuevo `updated` y a la
`generation` actual.

La sesión principal ejecutó `procedencia.py` para los 12 objetos de la tabla. Los 12
`updated` coincidieron con `read_object_metadata`, y de ahí salen las columnas
Generation y MD5 de la sección 5.

VC-5, VC-12 y VC-BR-2 dependen de listados completos; esos tres prefijos se
revalidaron por el lector sin leer contenido y quedaron registrados en la spec. El
cierre del skill también compara esos listados.

### Línea de base e2e

Ejecutada fuera de Claude, con las rutas de las credenciales solo en variables de
entorno, sin mostrarlas ni guardarlas en el repo:

```console
$ cd ../01-greenfield-gcsgrep
$ go test ./e2e -json
exit: 0
21 PASS, 0 FAIL, 0 SKIP
```

Fecha: 2026-10-13T18:15:35Z. Cuenta de lectura: `lectora`; pruebas negativas:
`sin-acceso`. Tests de nivel superior en PASS: `TestVC01`, `TestVC03`, `TestVC04`,
`TestVC05`, `TestVC06`, `TestVC07a`, `TestVC08`, `TestVC09a`, `TestVC09b`,
`TestVC12Parcial`, `TestVC14`, `TestVC15b`, `TestVC21`, `TestVC22`, `TestVC23`,
`TestVC35`, `TestVC39`, `TestVC40`, `TestVC41Parcial`, `TestVC42Parcial` y `TestVC46`.

## 11 · Cierre con `generation` distinta y salida exacta de VC-12

Sobre otra copia temporal de la spec, con procedencia nativa, solo la `generation` de
`logs/app/api.log` se cambió a `1`; `updated` y MD5 quedaron correctos. El skill
consultó metadata MCP y nativa con la cuenta lectora y marcó la copia vencida por la
`generation`, sin leer contenido ni tocar la spec original. Respuesta:

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

Para verificar texto y distribución por objeto, `lector-fixtures` leyó `logs/` con
`selección: sin-match` y `texto_completo: sí`. Usó solo `list_objects`,
`read_object_metadata` y `read_object_content`; leyó los cinco objetos de texto y
salteó el gzip. La sesión principal no llamó a lectura de contenido. Las 32 líneas
seleccionadas completas se guardaron en `test-fixtures/invert-logs-esperado.txt`
(13/5/4/6/4 por objeto). El VC compara su salida contra ese archivo ordenado, no solo
la cantidad de líneas.

## 12 · Freno sin el server

Ejecutada el 2026-10-14T02:11:00Z, Claude Code 2.1.296, desde `04-mcps-&-plugins/` con
el toolkit suelto. La sesión se abrió **sin ningún server MCP** y sin
`GOOGLE_APPLICATION_CREDENTIALS` ni `GCSGREP_BUCKET` en el entorno:

```sh
env -u GOOGLE_APPLICATION_CREDENTIALS -u GCSGREP_BUCKET \
  claude -p 'especificá `-v` para gcsgrep' \
  --strict-mcp-config --mcp-config <carpeta-temporal>/sin-server.json \
  --permission-mode dontAsk \
  --allowedTools "Skill" "Bash(printenv GCSGREP_BUCKET)" "Bash(python3 *identidad.py)" \
  --output-format stream-json --verbose
```

`sin-server.json` es `{"mcpServers":{}}`. Del evento `system/init`:

```json
{"mcp_servers":[],"claude_code_version":"2.1.296","gcs_tools":[]}
```

Las tres llamadas de la sesión, en orden, con su resultado:

```json
{"tool":"Skill","input":{"skill":"write-vcs-con-bucket","args":"especificá `-v` para gcsgrep"}}
```

```text
Launching skill: write-vcs-con-bucket
```

```json
{"tool":"ToolSearch","input":{"query":"select:mcp__gcs__read_object_metadata","max_results":5}}
```

```text
No matching deferred tools found
```

```json
{"tool":"Bash","input":{"command":"printenv GCSGREP_BUCKET; echo \"bucket_exit=$?\"; python3 .claude/skills/write-vcs-con-bucket/scripts/identidad.py; echo \"identidad_exit=$?\"; ls sdd/"}}
```

```text
Permission to use Bash has been denied because Claude Code is running in don't ask mode. […]
```

Respuesta final:

```text
Me detengo en el paso 1 (freno) del skill: no pude confirmar ninguna de las tres condiciones, así que no escribo ningún VC.

- **Tool `mcp__gcs__read_object_metadata`: no está en esta sesión.** No aparece entre mis tools ni entre las diferidas. Según el skill, esto apunta a la config del server `gcs` de `.mcp.json` (no levantó o no está aprobado en esta sesión).
- **`printenv GCSGREP_BUCKET`: no se pudo ejecutar.** La sesión corre en modo "don't ask" y Bash fue denegado.
- **`identidad.py`: no se pudo ejecutar**, por la misma denegación de Bash. No sé con qué cuenta correría el server.

No toqué el bucket ni lancé `lector-fixtures`. Tampoco pude listar `sdd/`: el historial de commits menciona una `invert-match-spec`, pero no verifiqué su estado.

Para seguir necesito que:

1. Revises que el server `gcs` esté conectado (`/mcp`) y que la sesión se haya abierto desde un shell con `GOOGLE_APPLICATION_CREDENTIALS` y `GCSGREP_BUCKET` exportadas.
2. Habilites Bash, al menos para `printenv GCSGREP_BUCKET`, `identidad.py`, `procedencia.py`, `date -u` y `vc-coverage.sh`, que el skill usa en los pasos 1, 2, 4, 5 y 6.

Con eso resuelto, repetí el pedido y arranco desde el freno.
```

Sin el server la tool de metadata no existe: el skill lo detecta en su primer paso y
corta. No lanzó `lector-fixtures`, no escribió ni editó la spec y no completó VCs de
memoria. Las otras dos comprobaciones no corrieron porque el agente las encadenó en un
solo comando y el modo no interactivo solo permitía cada una por separado; la primera
condición ya alcanzaba para frenar.
