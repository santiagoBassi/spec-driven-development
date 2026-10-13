# 07 · Costo de contexto

Capturado el 2026-10-13T18:30:00.557784+00:00, Claude Code
2.1.296, con el plugin 1.0.0 y el server conectado usando la credencial lectora.
Sesión en una carpeta temporal fuera del repo, con las opciones de `06-plugin.md`.

Comando ejecutado:

```sh
claude --plugin-dir <checkout>/04-mcps-\&-plugins/plugin/sdd-toolkit \
  --setting-sources "" --settings <carpeta-temporal>/settings.json \
  --permission-mode dontAsk -p /context
```

## Salida de `/context` (extracto)

Los totales generales incluyen conectores de claude.ai. Se conservan sus totales
y las filas GCS para separar el aporte del toolkit.

## Context Usage

**Model:** claude-opus-5-5
**Tokens:** 12.8k / 1m (1%)

### Estimated usage by category

| Category | Tokens | Percentage |
|----------|--------|------------|
| System prompt | 2.3k | 0.2% |
| System tools | 5.5k | 0.5% |
| MCP tools | 622 | 0.1% |
| MCP server instructions | 1.4k | 0.1% |
| MCP tools (deferred) | 168.9k | 16.9% |
| System tools (deferred) | 16.6k | 1.7% |
| Custom agents | 351 | 0.0% |
| Skills | 2.5k | 0.3% |
| Messages | 10 | 0.0% |
| Free space | 954.3k | 95.4% |
| Autocompact buffer | 33k | 3.3% |

### MCP Tools — filas GCS

| Tool | Server | Tokens |
|------|--------|--------|
| mcp__plugin_sdd-toolkit_gcs__check_iam_permissions | plugin_sdd-toolkit_gcs | 179 |
| mcp__plugin_sdd-toolkit_gcs__copy_object_safe | plugin_sdd-toolkit_gcs | 299 |
| mcp__plugin_sdd-toolkit_gcs__create_bucket | plugin_sdd-toolkit_gcs | 374 |
| mcp__plugin_sdd-toolkit_gcs__execute_insights_query | plugin_sdd-toolkit_gcs | 255 |
| mcp__plugin_sdd-toolkit_gcs__get_bucket_location | plugin_sdd-toolkit_gcs | 137 |
| mcp__plugin_sdd-toolkit_gcs__get_bucket_metadata | plugin_sdd-toolkit_gcs | 135 |
| mcp__plugin_sdd-toolkit_gcs__get_metadata_table_schema | plugin_sdd-toolkit_gcs | 293 |
| mcp__plugin_sdd-toolkit_gcs__list_buckets | plugin_sdd-toolkit_gcs | 125 |
| mcp__plugin_sdd-toolkit_gcs__list_insights_configs | plugin_sdd-toolkit_gcs | 157 |
| mcp__plugin_sdd-toolkit_gcs__list_objects | plugin_sdd-toolkit_gcs | 522 |
| mcp__plugin_sdd-toolkit_gcs__read_object_content | plugin_sdd-toolkit_gcs | 168 |
| mcp__plugin_sdd-toolkit_gcs__read_object_metadata | plugin_sdd-toolkit_gcs | 168 |
| mcp__plugin_sdd-toolkit_gcs__upload_object_safe | plugin_sdd-toolkit_gcs | 268 |
| mcp__plugin_sdd-toolkit_gcs__view_iam_policy | plugin_sdd-toolkit_gcs | 132 |
| mcp__plugin_sdd-toolkit_gcs__write_object_safe | plugin_sdd-toolkit_gcs | 257 |

### Custom Agents

| Agent Type | Source | Tokens |
|------------|--------|--------|
| sdd-toolkit:lector-fixtures | Plugin | 200 |
| sdd-toolkit:review-spec | Plugin | 151 |

### Skills — filas del plugin

| Skill | Source | Tokens |
|-------|--------|--------|
| sdd-toolkit:write-spec | Plugin (sdd-toolkit) | ~180 |
| sdd-toolkit:write-vcs-con-bucket | Plugin (sdd-toolkit) | ~230 |

## Interpretación

Los esquemas de las 15 tools GCS suman **3.469 tokens**. Los agentes suman 351
y las descripciones de los skills aproximadamente 410. No incluye sus cuerpos
cargados ni las respuestas de lectura de los objetos. Los helpers Python no
agregan tools MCP: se invocan por Bash como scripts del skill.
Las tools diferidas no equivalen a tokens activos en cada turno; los totales
generales incluyen otros conectores y no son el incremento del server GCS.
Habilitar el plugin por proyecto o con `--plugin-dir` cuando se necesita el bucket.
