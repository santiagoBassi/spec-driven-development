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
