# 01 · Descubrimiento de tools

Capturado el 2026-10-12T23:48Z en una sesión de Claude Code 2.1.296 abierta en
`04-mcps-&-plugins/`, con el server `gcs` de `.mcp.json`:

```json
{ "command": "npx", "args": ["-y", "@google-cloud/storage-mcp@0.6.0"] }
```

Versión en ejecución: `@google-cloud/storage-mcp` **0.6.0**, sin `--enable-destructive-tools`.

## Pedido

> listá las tools del server gcs

## Respuesta del agente

El server `gcs` expone **15 tools**. Claude Code las publica como `mcp__gcs__<tool>`.

### Lectura de objetos (las únicas que usa el skill)

| Tool | Qué hace | Argumentos |
|---|---|---|
| `mcp__gcs__list_objects` | Lista los nombres de los objetos de un bucket | `bucket_name`\*, `prefix`, `delimiter`, `max_results`, `page_token`, `versions` |
| `mcp__gcs__read_object_metadata` | Lee la metadata de un objeto | `bucket_name`\*, `object_name`\* |
| `mcp__gcs__read_object_content` | Lee el contenido de un objeto | `bucket_name`\*, `object_name`\* |

### Crean, aun en modo seguro

| Tool | Qué hace | Argumentos |
|---|---|---|
| `mcp__gcs__create_bucket` | Crea un bucket | `project_id`\*, `bucket_name`\*, `location`, `storage_class`, `labels`, `versioning_enabled`, `requester_pays` |
| `mcp__gcs__write_object_safe` | Escribe un objeto nuevo; falla si ya existe | `bucket_name`\*, `object_name`\*, `content`\* (base64), `content_type` |
| `mcp__gcs__upload_object_safe` | Sube un archivo local; falla si el objeto ya existe | `bucket_name`\*, `file_path`\*, `object_name`, `content_type` |
| `mcp__gcs__copy_object_safe` | Copia un objeto; falla si el destino ya existe | `source_bucket_name`\*, `source_object_name`\*, `destination_bucket_name`\*, `destination_object_name`\* |

### Otras de lectura (bucket, IAM, Storage Insights)

| Tool | Qué hace | Argumentos |
|---|---|---|
| `mcp__gcs__list_buckets` | Lista los buckets del proyecto | `project_id` |
| `mcp__gcs__get_bucket_metadata` | Metadata detallada de un bucket | `bucket_name`\* |
| `mcp__gcs__get_bucket_location` | Ubicación y storage class de un bucket | `bucket_name`\* |
| `mcp__gcs__view_iam_policy` | Política IAM de un bucket | `bucket_name`\* |
| `mcp__gcs__check_iam_permissions` | Prueba permisos IAM sobre un bucket | `bucket_name`\*, `permissions`\* |
| `mcp__gcs__list_insights_configs` | Lista las configuraciones de Storage Insights | `projectId` |
| `mcp__gcs__get_metadata_table_schema` | Esquema de la tabla de BigQuery de un dataset de Insights | `datasetConfigName`\*, `datasetConfigLocation`\*, `projectId` |
| `mcp__gcs__execute_insights_query` | Corre una query SQL de BigQuery contra un dataset de Insights | `config`\*, `query`\*, `jobTimeoutMs` |

\* obligatorio.

### No existen en este modo

No hay ninguna tool que borre o pise: no existe `delete_object` ni variantes de escritura
sin el sufijo `_safe`.

## Identidad del server

Comprobado el 2026-10-12T23:50Z, en la sesión abierta con las variables del paso 2.2.

Entorno del proceso del server (`/proc/<pid>/environ`, filtrado a las dos variables):

```
--- pid 101311 (ppid 101310) started lun oct 12 20:50:08 2026
node /home/santiago/.npm/_npx/fc16a9b346a70fd6/node_modules/.bin/storage-mcp
cwd: /home/santiago/Documents/spec-driven-development/04-mcps-&-plugins
GOOGLE_APPLICATION_CREDENTIALS=/home/santiago/Documents/spec-driven-development/01-greenfield-gcsgrep/lectora.json
GCSGREP_BUCKET=sdd-fardenghi-itba
```

La variable apunta a la key; quién es lo dice el propio server. `mcp__gcs__view_iam_policy`
sobre el bucket devuelve un rechazo que nombra la cuenta con la que se autenticó:

```json
{"success":false,"error":"Error viewing IAM policy: lectora@sat-spec-driven-development.iam.gserviceaccount.com does not have storage.buckets.getIamPolicy access to the Google Cloud Storage bucket. Permission 'storage.buckets.getIamPolicy' denied on resource '//storage.googleapis.com/projects/_/buckets/sdd-fardenghi-itba' (or it may not exist).","error_type":"Unknown"}
```

**`client_email`: `lectora@sat-spec-driven-development.iam.gserviceaccount.com`**, no el
ADC personal. Con esa misma identidad, `mcp__gcs__list_objects` con prefijo
`fixtures/hostil/` devuelve el objeto subido en el paso 1:

```json
{ "bucket": "sdd-fardenghi-itba", "prefix": "fixtures/hostil/", "object_count": 1, "objects": ["fixtures/hostil/notas.log"], "next_page_token": null }
```

## Diferencia con el enunciado

El enunciado nombra las tools que crean con el sufijo `_new`. La versión 0.6.0 las
expone con el sufijo `_safe`:

| Enunciado | Server 0.6.0 |
|---|---|
| `write_object_new` | `write_object_safe` |
| `upload_object_new` | `upload_object_safe` |
| `copy_object_new` | `copy_object_safe` |
| `create_bucket` | `create_bucket` (igual) |
| `list_objects`, `read_object_metadata`, `read_object_content` | iguales |
| `delete_object` | no existe (igual que en el enunciado) |

Consecuencias para el resto del TP:

- La prueba del hook ("pedile un `write_object_new`") se hace con `write_object_safe`.
- El hook es una allowlist, no una lista de vetos: deja pasar las tres de lectura de
  objetos y veta cualquier otro nombre, incluidos los ocho de lectura que el enunciado
  no menciona y cualquier tool que aparezca en una versión futura.
