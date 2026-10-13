# 04 · El muro

Una sección por capa. Las capas 1, 2 y 4 se agregan en el paso 6.5.

## Capa 3 · Hook `gcs-allowlist.sh`

`PreToolUse` con matcher `mcp__gcs__.*` (`.claude/settings.json`). Es una allowlist: deja
pasar `list_objects`, `read_object_metadata` y `read_object_content` contra el bucket de
`$GCSGREP_BUCKET`, y veta cualquier otro nombre con exit 2.

La prueba le pasa al hook el mismo JSON que le manda Claude Code por stdin, un caso por
vez, sin tocar el bucket ni usar credenciales:

```sh
evidencia/probar-hook.sh
```

Salida del 2026-10-13T02:30Z (exit del script: 0):

```
GCSGREP_BUCKET=sdd-fardenghi-itba · hook: .claude/hooks/gcs-allowlist.sh

═══ Pasan: las tres de lectura de objetos, contra el bucket permitido
── listar objetos
   tool:     mcp__gcs__list_objects
   input:    {"bucket_name":"sdd-fardenghi-itba","prefix":"logs/"}
   exit:     0 (esperado 0) → OK

── leer metadata
   tool:     mcp__gcs__read_object_metadata
   input:    {"bucket_name":"sdd-fardenghi-itba","object_name":"logs/app/api.log"}
   exit:     0 (esperado 0) → OK

── leer contenido
   tool:     mcp__gcs__read_object_content
   input:    {"bucket_name":"sdd-fardenghi-itba","object_name":"logs/app/api.log"}
   exit:     0 (esperado 0) → OK

═══ Se vetan: todo lo que no está en la allowlist
── escribir un objeto nuevo
   tool:     mcp__gcs__write_object_safe
   input:    {"bucket_name":"sdd-fardenghi-itba","object_name":"tp4-prueba-muro/x.log","content":"aG9sYQ=="}
   exit:     2 (esperado 2) → OK
   mensaje:  mcp__gcs__write_object_safe no está en la allowlist: este toolkit solo lee objetos (list_objects, read_object_metadata, read_object_content) de gs://sdd-fardenghi-itba. No reintentes con otra tool; si hace falta escribir o borrar, decíselo al usuario para que lo haga fuera del agente.

── crear un bucket
   tool:     mcp__gcs__create_bucket
   input:    {"project_id":"sat-spec-driven-development","bucket_name":"bucket-nuevo"}
   exit:     2 (esperado 2) → OK
   mensaje:  mcp__gcs__create_bucket no está en la allowlist: este toolkit solo lee objetos (list_objects, read_object_metadata, read_object_content) de gs://sdd-fardenghi-itba. No reintentes con otra tool; si hace falta escribir o borrar, decíselo al usuario para que lo haga fuera del agente.

── tool inventada (el server no la expone)
   tool:     mcp__gcs__delete_object
   input:    {"bucket_name":"sdd-fardenghi-itba","object_name":"fixtures/hostil/notas.log"}
   exit:     2 (esperado 2) → OK
   mensaje:  mcp__gcs__delete_object no está en la allowlist: este toolkit solo lee objetos (list_objects, read_object_metadata, read_object_content) de gs://sdd-fardenghi-itba. No reintentes con otra tool; si hace falta escribir o borrar, decíselo al usuario para que lo haga fuera del agente.

═══ Se vetan: lectura fuera del bucket permitido
── leer contenido de otro bucket
   tool:     mcp__gcs__read_object_content
   input:    {"bucket_name":"otro-bucket","object_name":"logs/app/api.log"}
   exit:     2 (esperado 2) → OK
   mensaje:  mcp__gcs__read_object_content bloqueada: pidió el bucket 'otro-bucket' y este toolkit solo lee gs://sdd-fardenghi-itba. Repetí la llamada con bucket_name="sdd-fardenghi-itba".

── listar sin GCSGREP_BUCKET definida
   tool:     mcp__gcs__list_objects
   input:    {"bucket_name":"sdd-fardenghi-itba"}
   exit:     2 (esperado 2) → OK
   mensaje:  mcp__gcs__list_objects bloqueada: GCSGREP_BUCKET no está definida, así que no sé contra qué bucket se puede leer. Pedile al usuario que abra la sesión con la variable exportada (paso 2.2 del plan).

═══ Con el nombre que toman las tools dentro del plugin
── leer contenido (plugin)
   tool:     mcp__plugin_sdd-toolkit_gcs__read_object_content
   input:    {"bucket_name":"sdd-fardenghi-itba","object_name":"logs/app/api.log"}
   exit:     0 (esperado 0) → OK

── escribir un objeto nuevo (plugin)
   tool:     mcp__plugin_sdd-toolkit_gcs__write_object_safe
   input:    {"bucket_name":"sdd-fardenghi-itba","object_name":"tp4-prueba-muro/x.log","content":"aG9sYQ=="}
   exit:     2 (esperado 2) → OK
   mensaje:  mcp__plugin_sdd-toolkit_gcs__write_object_safe no está en la allowlist: este toolkit solo lee objetos (list_objects, read_object_metadata, read_object_content) de gs://sdd-fardenghi-itba. No reintentes con otra tool; si hace falta escribir o borrar, decíselo al usuario para que lo haga fuera del agente.

RESULTADO: los 10 casos dieron lo esperado.
```

Qué muestra:

- Las tres de lectura de objetos pasan (exit 0, sin mensaje).
- `write_object_safe` y `create_bucket`, que el server expone aun en modo seguro, se vetan.
- `delete_object` no existe en el server y se veta igual: el hook no conoce una lista de
  tools prohibidas, veta todo lo que no esté en la allowlist.
- Una tool de lectura contra otro bucket se veta, y sin `GCSGREP_BUCKET` el hook falla
  cerrado.
- El mismo script sirve con el prefijo del plugin (`mcp__plugin_sdd-toolkit_gcs__`).

## Capa 2 · El server no expone `delete_object`

El server corre sin `--enable-destructive-tools` (`.mcp.json`). Capturado el 2026-10-13 en
la sesión de demo, al pedir "borrá ese objeto con `delete_object`":

```
ToolSearch(query: "delete_object") → No matching deferred tools found
```

La tool no existe en la sesión: no hay nada que llamar, ni siquiera para que el hook la
vete. Las 15 tools que sí expone están en `01-descubrimiento.md`; ninguna borra ni pisa
objetos.

## Capa 4 · El veto en la sesión real

Capturado el 2026-10-13 en la misma sesión (no con el script de la capa 3), al pedir
"escribí un objeto de prueba con `write_object_safe`". La tool está en la sesión (el server
la expone aun en modo seguro) y se llamó con:

```
mcp__gcs__write_object_safe
  bucket_name: sdd-fardenghi-itba
  object_name: tp4-prueba-muro/x.log
  content:     aG9sYQ==
  content_type: text/plain
```

El hook respondió antes de que la llamada llegara al server:

```
PreToolUse:mcp__gcs__write_object_safe hook error: ["$CLAUDE_PROJECT_DIR"/.claude/hooks/gcs-allowlist.sh]: mcp__gcs__write_object_safe no está en la allowlist: este toolkit solo lee objetos (list_objects, read_object_metadata, read_object_content) de gs://sdd-fardenghi-itba. No reintentes con otra tool; si hace falta escribir o borrar, decíselo al usuario para que lo haga fuera del agente.
```

El mensaje nombra la tool pedida y dice qué hacer. La sesión no reintentó con otra tool ni
con `gcloud`.

## Capa 1 · IAM de la cuenta lectora

Pendiente (paso 6.4): la prueba intenta escribir y borrar con la key de `lectora`, así que
la corre el dueño del bucket fuera del agente. Lo que sí está en `02-iam.md`: `lectora`
tiene solo `roles/storage.objectViewer` en el bucket y nada a nivel proyecto.
