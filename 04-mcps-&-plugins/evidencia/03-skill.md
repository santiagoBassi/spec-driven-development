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

## 8 · Cierre fallido por `updated` distinto

Pendiente: se agrega después del paso 6.7 (re-subida de `logs/app/api.log` por el dueño del
bucket) y de pedir "cerrá la spec de `-v`".
