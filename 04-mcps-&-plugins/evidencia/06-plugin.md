# 06 · Plugin `sdd-toolkit` 1.0.0

Capturado el 2026-10-13T17:48:31.463949+00:00, Claude Code 2.1.296. Plugin cargado con
`claude --plugin-dir <checkout>/04-mcps-&-plugins/plugin/sdd-toolkit` desde una
carpeta temporal fuera del repo: no se cargó el `.claude/` suelto del TP4.

## Validación

```console
$ claude plugin validate --strict plugin/sdd-toolkit
Validating plugin manifest: <checkout>/04-mcps-&-plugins/plugin/sdd-toolkit/.claude-plugin/plugin.json

✔ Validation passed

Validating components in: <checkout>/04-mcps-&-plugins/plugin/sdd-toolkit/skills

✔ Validation passed

Validating components in: <checkout>/04-mcps-&-plugins/plugin/sdd-toolkit/agents

✔ Validation passed

```

## Configuración de usuario

El manifest declara `project` y `bucket`, ambos no sensibles. Para esta sesión
se pasaron con `--settings <carpeta-temporal>/settings.json`:

```json
{
  "pluginConfigs": {
    "sdd-toolkit@inline": {
      "options": {
        "project": "sat-spec-driven-development",
        "bucket": "sdd-fardenghi-itba"
      }
    }
  }
}
```

No se exportó `GCSGREP_BUCKET` en el shell: el hook recibió el bucket por
`CLAUDE_PLUGIN_OPTION_BUCKET`. El server recibe `GCSGREP_BUCKET` y
`GOOGLE_CLOUD_PROJECT` desde `${user_config.*}` en `.mcp.json`; la credencial
se hereda del entorno (`GOOGLE_APPLICATION_CREDENTIALS`), sin su valor en el paquete.
Los skills usan `${user_config.bucket}` y `${user_config.project}` en su contenido.

## Descubrimiento real

Del evento `system/init` de la sesión:

```json
{
  "mcp_servers": [
    {
      "name": "plugin:sdd-toolkit:gcs",
      "status": "connected",
      "source": "plugin"
    }
  ],
  "plugins": [
    {
      "name": "sdd-toolkit",
      "source": "sdd-toolkit@inline",
      "version": "1.0.0"
    }
  ],
  "agents": [
    "sdd-toolkit:lector-fixtures",
    "sdd-toolkit:review-spec"
  ],
  "skills": [
    "sdd-toolkit:write-spec",
    "sdd-toolkit:write-vcs-con-bucket"
  ],
  "tools": [
    "mcp__plugin_sdd-toolkit_gcs__check_iam_permissions",
    "mcp__plugin_sdd-toolkit_gcs__copy_object_safe",
    "mcp__plugin_sdd-toolkit_gcs__create_bucket",
    "mcp__plugin_sdd-toolkit_gcs__execute_insights_query",
    "mcp__plugin_sdd-toolkit_gcs__get_bucket_location",
    "mcp__plugin_sdd-toolkit_gcs__get_bucket_metadata",
    "mcp__plugin_sdd-toolkit_gcs__get_metadata_table_schema",
    "mcp__plugin_sdd-toolkit_gcs__list_buckets",
    "mcp__plugin_sdd-toolkit_gcs__list_insights_configs",
    "mcp__plugin_sdd-toolkit_gcs__list_objects",
    "mcp__plugin_sdd-toolkit_gcs__read_object_content",
    "mcp__plugin_sdd-toolkit_gcs__read_object_metadata",
    "mcp__plugin_sdd-toolkit_gcs__upload_object_safe",
    "mcp__plugin_sdd-toolkit_gcs__view_iam_policy",
    "mcp__plugin_sdd-toolkit_gcs__write_object_safe"
  ]
}
```

`delete_object` no está entre las 15 tools de ese server conectado. Las búsquedas
por texto devolvieron otras tools, pero ninguna de borrado; el inventario completo
confirma la ausencia (no se infiere a partir de un server desconectado).

## Llamadas y respuestas

Se permitió expresamente el intento `write_object_safe` con `--allowedTools` para
que la prueba alcanzara el hook; el veto proviene del plugin. La sesión solo tuvo
`ToolSearch` como herramienta integrada, sin Bash. No se leyó contenido de fixtures.

`list_objects`, permitida:

```json
{
  "tool": "mcp__plugin_sdd-toolkit_gcs__list_objects",
  "input": {
    "bucket_name": "sdd-fardenghi-itba",
    "prefix": "logs/app/"
  }
}
```

```text
{
  "bucket": "sdd-fardenghi-itba",
  "prefix": "logs/app/",
  "object_count": 2,
  "objects": [
    "logs/app/api.log",
    "logs/app/worker.log"
  ],
  "next_page_token": null
}
```

`read_object_metadata`, permitida:

```json
{
  "tool": "mcp__plugin_sdd-toolkit_gcs__read_object_metadata",
  "input": {
    "bucket_name": "sdd-fardenghi-itba",
    "object_name": "logs/app/api.log"
  }
}
```

```text
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

`write_object_safe`, vetada por el hook del plugin:

```json
{
  "tool": "mcp__plugin_sdd-toolkit_gcs__write_object_safe",
  "input": {
    "bucket_name": "sdd-fardenghi-itba",
    "object_name": "tp4-prueba-muro/plugin.txt",
    "content": "aG9sYQ==",
    "content_type": "text/plain"
  }
}
```

```text
PreToolUse:mcp__plugin_sdd-toolkit_gcs__write_object_safe hook error: ["<checkout>/04-mcps-&-plugins/plugin/sdd-toolkit/hooks/gcs-allowlist.sh"]: mcp__plugin_sdd-toolkit_gcs__write_object_safe no está en la allowlist: este toolkit solo lee objetos (list_objects, read_object_metadata, read_object_content) de gs://sdd-fardenghi-itba. No reintentes con otra tool; si hace falta escribir o borrar, decíselo al usuario para que lo haga fuera del agente.

This hook comes from the sdd-toolkit@inline plugin.
```

## Pruebas locales de portabilidad y del gate

```console
$ python3 evidencia/probar-plugin.py
PASS · list_objects · exit 0
PASS · read_object_metadata · exit 0
PASS · read_object_content · exit 0
PASS · write_object_safe · exit 2
PASS · create_bucket · exit 2
PASS · delete_object · exit 2
PASS · tool_futura · exit 2
PASS · otro bucket · exit 2
PASS · bucket ausente · exit 2
PASS · sin configuración · exit 2
PASS · userConfig prevalece sobre el shell · exit 0
PASS · Write con cobertura · exit 0
PASS · Write sin VC · exit 2
PASS · definición del agente fuera del gate de specs · exit 0
PASS · commit con spec nueva sin VC · exit 2
PASS · commit con cobertura · exit 0
PASS · staging sin VC aunque disco tenga cobertura · exit 2
PASS · metadata nativa: misma credencial, fecha normalizada y salida acotada
PASS · error de gcloud sin transmitir token ni ruta de key
PASS · metadata rechazada: objeto ajeno
PASS · metadata rechazada: precisión no soportada
PASS · sin credencial no hay consulta ni fallback a la cuenta activa
RESULTADO: 22 casos PASS; sin llamadas al bucket ni commits reales.
```

El gate de specs se probó en otro repo temporal cuya ruta contiene espacios,
sin `.claude/`, incluyendo una spec inválida en staging y válida en disco.
La garantía IAM del mismo server y la cuenta lectora está en `04-muro.md`.

## Instalación con un comando

También se instaló desde el marketplace local en el proyecto temporal.
`CLAUDE_CONFIG_DIR` apuntó a una carpeta temporal, por lo que la prueba no instaló
el plugin en la configuración habitual del usuario.

```console
$ claude plugin install sdd-toolkit --marketplace <checkout>/04-mcps-\&-plugins --scope project --config project=sat-spec-driven-development --config bucket=sdd-fardenghi-itba
Adding marketplace…
✔ Successfully added marketplace: sdd-course (declared in user settings)
Installing plugin "sdd-toolkit@sdd-course"...✔ Successfully installed plugin: sdd-toolkit@sdd-course (scope: project)
```

La instalación quedó como `sdd-toolkit@sdd-course`, con ambas opciones configuradas.

## Identidad del server

El skill incluye `scripts/identidad.py`: recibe la ruta solo por el entorno y
expone únicamente `client_email`. Evita la expansión de la ruta de la key en
un comando `jq` del agente, que la prueba no interactiva rechazó. Se verificaron
credencial ausente, service account válida, falta de identidad y JSON con tipo
inválido; ninguno expuso otros campos ni la ruta. El helper se ejecutó realmente
al cerrar una copia de la spec (`03-skill.md`, sección 9), con salida:

```text
lectora@sat-spec-driven-development.iam.gserviceaccount.com
```

La invocación real del skill también confirmó la sustitución de `userConfig`
en su contenido (proyecto y bucket configurados) y la ruta del helper dentro
del plugin aunque el checkout contiene espacios.

Se agregó un caso para las definiciones de agentes empaquetadas: `review-spec.md`
no es una spec del producto. Ambos hooks del TP4 excluyen agentes y skills
de `plugin/`; el control sigue bloqueando specs del producto sin VC.

## Procedencia nativa con la misma identidad

El plugin incorpora `procedencia.py`, que fuerza la credencial del server en
`CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE` y ejecuta exclusivamente
`gcloud storage objects describe --raw`. No usa la cuenta dueña activa, no
lee contenido y no muestra metadata libre ni el stderr que puede incluir rutas.
La fecha se normaliza a UTC con milisegundos; si hacerlo perdería precisión,
la consulta falla. Las pruebas locales cubren esas condiciones y el freno sin
credencial. La corrida real obtuvo generaciones y MD5 de los 12 fixtures
(`03-skill.md`, sección 10). La sección 11 muestra que una generation distinta
vence la spec aunque updated y MD5 coincidan.

`lector-fixtures` mantiene únicamente sus tres tools MCP. El modo `listado`
solo enumera nombres para verificar prefijos y ausencias, sin leer contenido.

## Instalación final con el helper de procedencia

Se repitió la instalación en otro proyecto y configuración temporales, con
el paquete final y sus helpers. Se comprobó que `procedencia.py` está incluido
en el plugin instalado y coincide con el del checkout.

```text
Adding marketplace…
✔ Successfully added marketplace: sdd-course (declared in user settings)
Installing plugin "sdd-toolkit@sdd-course"...✔ Successfully installed plugin: sdd-toolkit@sdd-course (scope: project)
```
