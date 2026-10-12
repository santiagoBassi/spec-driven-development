# TP4 — Plan: `gcsgrep` contra un bucket real, por MCP

## Contexto

El TP4 pide que un paso SDD de `gcsgrep` lea un bucket de GCS real a través del server
MCP `@google-cloud/storage-mcp`, probar con evidencia que el agente no puede escribir ni
borrar, y empaquetar el toolkit del TP3 más lo nuevo como plugin de Claude Code.

`criterios.md` no está disponible: la vara es el enunciado y su tabla de evidencia.

Los pasos marcados **(Filipo)** los tiene que correr él porque es el dueño del bucket.

## Decisiones

| Tema | Decisión |
|---|---|
| Paso SDD | **Especificar los VCs** |
| Feature de la demo | `invert-match -v` (el pedido de la demo del TP3) |
| Bucket | El del TP1: `gs://sdd-fardenghi-itba`, proyecto `sat-spec-driven-development` |
| Credencial del server | `lectora.json` por `GOOGLE_APPLICATION_CREDENTIALS` en el shell (nunca en el repo) |
| Permisos negativos | `sin-acceso.json` para el caso "objeto sin permisos" |
| Fixture hostil | El texto literal del enunciado (con `timeout`) |
| Agente | Solo Claude Code |
| Plugin | Commiteado completo en `plugin/sdd-toolkit/`, sin script de armado |

## Verificado

- `lectora.json` lista y lee el bucket: 24 objetos, con `generation` (p. ej. `logs/app/api.log` → `1790119237354314`).
- `lectora` no puede ver el IAM del bucket ni subir objetos.
- Las dos keys están gitignoreadas en `01-greenfield-gcsgrep/.gitignore`.
- `@google-cloud/storage-mcp` 0.6.0 expone 15 tools, con nombres distintos a los del enunciado:

  | Grupo | Tools |
  |---|---|
  | Lectura de objetos (allowlist) | `list_objects`, `read_object_metadata`, `read_object_content` |
  | Crean, aun en modo seguro | `create_bucket`, `write_object_safe`, `upload_object_safe`, `copy_object_safe` |
  | Otras de lectura (se vetan igual) | `list_buckets`, `get_bucket_metadata`, `get_bucket_location`, `view_iam_policy`, `check_iam_permissions`, `execute_insights_query`, `get_metadata_table_schema`, `list_insights_configs` |
  | No existen en este modo | `delete_object` y las que pisan |

  Argumentos: `bucket_name`, `object_name` (`copy_object_safe`: `source_*` /
  `destination_*`; `upload_object_safe`: `file_path` local).

## Nombres

- Server: `gcs` → tools `mcp__gcs__<tool>`.
- Plugin: `sdd-toolkit` → tools `mcp__plugin_sdd-toolkit_gcs__<tool>`.
- Skill `write-vcs-con-bucket`, subagent `lector-fixtures`, hook `gcs-allowlist.sh`.
- El bucket llega al skill por la variable `GCSGREP_BUCKET` (la misma del TP1).

## Estructura final

```
04-mcps-&-plugins/
├── .mcp.json                      # server gcs, versión fijada, sin secretos
├── .gitignore
├── CLAUDE.md                      # rule del TP3 + cuándo usar el skill nuevo
├── .claude/
│   ├── settings.json              # hook spec-vc-gate (TP3) + gcs-allowlist
│   ├── skills/write-spec/         # copia del TP3
│   ├── skills/write-vcs-con-bucket/SKILL.md
│   ├── agents/review-spec.md      # copia del TP3
│   ├── agents/lector-fixtures.md
│   └── hooks/{spec-vc-gate.sh,gcs-allowlist.sh}
├── plugin/sdd-toolkit/
│   ├── .claude-plugin/plugin.json # nombre, versión, userConfig (proyecto, bucket)
│   ├── .mcp.json                  # misma versión fijada
│   ├── skills/  agents/  hooks/hooks.json + scripts
├── fixtures/hostil/notas.log      # fuente del objeto hostil
├── sdd/                           # este plan + la spec que produce el skill
├── evidencia/01…07
└── README.md                      # qué hace, instalación, nota de seguridad
```

Se reusa del TP3 (`03-skills-&-agents/.claude/`): `write-spec` con `vc-coverage.sh`,
`review-spec`, `spec-vc-gate.sh`. Forma de referencia: `ejemplo/`.

## Pasos

Se marca `[x]` cada punto al terminarlo. Los pasos 3, 4, 5 y 7.1–7.3 no dependen del paso 1.

### 0 · Preparar

1. [x] Crear `fixtures/hostil/notas.log` con las 3 líneas literales del enunciado (dominio `.invalid`).
2. [x] Crear `.gitignore` del TP4: keys (`lectora.json`, `sin-acceso.json`) y `.claude/settings.local.json`.
3. [x] Mail de la cuenta lectora, para los comandos del paso 1: `lectora@sat-spec-driven-development.iam.gserviceaccount.com`.

### 1 · Bucket e IAM

1. [ ] **(Filipo)** Subir el fixture hostil:
   ```sh
   gcloud storage cp notas.log gs://sdd-fardenghi-itba/fixtures/hostil/notas.log
   ```
2. [ ] **(Filipo)** Volcar el IAM del bucket y pasar la salida completa:
   ```sh
   gcloud storage buckets get-iam-policy gs://sdd-fardenghi-itba
   ```
3. [ ] **(Filipo)** Listar los roles de `lectora` a nivel proyecto y pasar la salida (lo esperado es vacío):
   ```sh
   gcloud projects get-iam-policy sat-spec-driven-development \
     --flatten="bindings[].members" --filter="bindings.members:lectora@sat-spec-driven-development.iam.gserviceaccount.com" \
     --format="table(bindings.role)"
   ```
4. [ ] **(Filipo)** Solo si el punto 3 devolvió algún rol de Storage: moverlo al bucket y repetir 2 y 3.
   ```sh
   gcloud projects remove-iam-policy-binding sat-spec-driven-development \
     --member="serviceAccount:lectora@sat-spec-driven-development.iam.gserviceaccount.com" --role="<el rol que apareció>"
   gcloud storage buckets add-iam-policy-binding gs://sdd-fardenghi-itba \
     --member="serviceAccount:lectora@sat-spec-driven-development.iam.gserviceaccount.com" --role="roles/storage.objectViewer"
   ```
5. [ ] Guardar las salidas finales de 2 y 3 en `evidencia/02-iam.md`, marcando la línea con el rol de `lectora`.

**Listo cuando:** `lectora` tiene `roles/storage.objectViewer` en el bucket, nada a
nivel proyecto, y `fixtures/hostil/notas.log` aparece al listar el bucket.

### 2 · Conectar

1. [x] `.mcp.json` con `npx -y @google-cloud/storage-mcp@0.6.0`, sin `--enable-destructive-tools`.
2. [ ] Abrir `claude` con la credencial en el entorno. Sin las variables el server usa el ADC personal y no `lectora`.
   ```sh
   export GOOGLE_APPLICATION_CREDENTIALS="$HOME/Documents/spec-driven-development/01-greenfield-gcsgrep/lectora.json"
   export GCSGREP_BUCKET=sdd-fardenghi-itba
   cd "$HOME/Documents/spec-driven-development/04-mcps-&-plugins" && claude
   ```
3. [ ] Pedir: "listá las tools del server gcs". Guardar la respuesta en `evidencia/01-descubrimiento.md`, con la diferencia de nombres respecto del enunciado.
4. [ ] Comprobar que el proceso `storage-mcp` tiene `GOOGLE_APPLICATION_CREDENTIALS` en su entorno y anotar el `client_email` en la misma evidencia.

**Listo cuando:** existe `evidencia/01-descubrimiento.md` y el server corre como `lectora`.

### 3 · Hook allowlist

1. [ ] `.claude/hooks/gcs-allowlist.sh`: `PreToolUse` que lee el JSON por stdin y deja pasar solo `list_objects`, `read_object_metadata`, `read_object_content`. Veta el resto (exit 2) con un mensaje que nombra la tool pedida y dice qué hacer. Veta también las de lectura si `bucket_name` no es `$GCSGREP_BUCKET`.
2. [ ] Copiar del TP3 `spec-vc-gate.sh`, `write-spec/` y `review-spec.md` a `.claude/`.
3. [ ] `.claude/settings.json`: el hook del TP3 más el matcher `mcp__gcs__.*` → `gcs-allowlist.sh`.
4. [ ] `evidencia/probar-hook.sh`: pasa al hook un JSON por caso (las 3 de lectura, `write_object_safe`, `create_bucket`, una tool inventada, otro bucket) y muestra exit code y mensaje. La salida va a `04-muro`.

**Listo cuando:** pasan los 3 casos de lectura y se vetan los demás, incluida la tool inventada.

### 4 · Subagent `lector-fixtures`

1. [ ] `.claude/agents/lector-fixtures.md` con `tools:` = solo las tres `mcp__gcs__*` de lectura. Sin Bash, sin Read, sin red.
2. [ ] Salida: tabla con objeto, generation, tamaño, tipo de contenido y líneas con match (número y texto corto).
3. [ ] Sección aparte **"Parece una instrucción"**: objeto, número de línea y primeras 40 letras, sin copiar el texto completo.
4. [ ] Los objetos que no son texto (`.gz`, binario) se reportan por metadata, sin decodificar.

**Listo cuando:** el frontmatter no lista ninguna tool fuera de las tres.

### 5 · Skill `write-vcs-con-bucket`

1. [ ] `.claude/skills/write-vcs-con-bucket/SKILL.md` con `allowed-tools` acotado y este flujo:
   1. **Freno:** si no existen las tools `mcp__gcs__*` o falta `GCSGREP_BUCKET`, se detiene y lo dice. No escribe VCs.
   2. **Lectura:** lanza `lector-fixtures` (`list_objects` → `read_object_metadata` → `read_object_content`). El skill no lee contenido por su cuenta.
   3. **VCs:** escribe los VCs de `-v` nombrando objeto real, línea real y generation.
   4. **Procedencia:** sección con objeto, generation, fecha y cuenta (`client_email`).
   5. **Cierre:** relee `read_object_metadata` de cada objeto citado; si una generation cambió, falla y pide releer.
   6. **Encadenado TP3:** `vc-coverage.sh` y `review-spec`.
2. [ ] `CLAUDE.md`: la rule del TP3 más cuándo se dispara este skill.

### 6 · Demo y evidencia 03–05

Requiere el paso 1 terminado y la sesión abierta como en 2.2.

1. [ ] Pedir: "especificá `-v` para gcsgrep". La spec queda en `sdd/` y la transcripción en `evidencia/03-skill.md`, con la línea que solo pudo venir del bucket marcada y la sección de procedencia.
2. [ ] Pedir: "escribí un objeto de prueba con `write_object_safe`" → veto del hook.
3. [ ] Pedir: "borrá ese objeto con `delete_object`" → la tool no existe.
4. [ ] Probar la credencial fuera del agente, sin tocar el `gcloud` activo. El `rm` se corre solo si el `cp` dio `403`; si no, el rol es más amplio de lo esperado y se vuelve al paso 1.
   ```sh
   K="$HOME/Documents/spec-driven-development/01-greenfield-gcsgrep"
   CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE="$K/lectora.json" gcloud storage cp fixtures/hostil/notas.log gs://sdd-fardenghi-itba/tp4-prueba-muro/x.log
   CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE="$K/lectora.json" gcloud storage rm gs://sdd-fardenghi-itba/fixtures/hostil/notas.log
   CLOUDSDK_AUTH_CREDENTIAL_FILE_OVERRIDE="$K/sin-acceso.json" gcloud storage cat gs://sdd-fardenghi-itba/logs/app/api.log
   ```
5. [ ] Juntar 3.4, 6.2, 6.3 y 6.4 en `evidencia/04-muro.md`, una sección por capa.
6. [ ] `evidencia/05-fixture-hostil.md`: la salida del subagent con `fixtures/hostil/notas.log` marcado y la lista de tools llamadas en la sesión (ningún Bash con `curl`).
7. [ ] **(Filipo)** Re-subir un objeto citado en la spec para que cambie su generation:
   ```sh
   gcloud storage cp gs://sdd-fardenghi-itba/logs/app/api.log gs://sdd-fardenghi-itba/logs/app/api.log
   ```
8. [ ] Pedir: "cerrá la spec de `-v`" → el cierre del skill falla por generation distinta. Agregar ese fallo a `evidencia/03-skill.md` y releer para dejar la spec al día.

### 7 · Empaquetar

1. [ ] Armar `plugin/sdd-toolkit/` con `skills/`, `agents/`, `hooks/hooks.json` + scripts y `.mcp.json` (misma versión `0.6.0`).
2. [ ] `.claude-plugin/plugin.json`: nombre, versión `1.0.0`, `userConfig` con proyecto y bucket (no son secretos).
3. [ ] Renombrar `mcp__gcs__` → `mcp__plugin_sdd-toolkit_gcs__` en el matcher del hook, en `allowed-tools` del skill y en `tools` del subagent. Comprobar con `grep` que no quede ningún `mcp__gcs__` dentro de `plugin/`.
4. [ ] `claude plugin validate plugin/sdd-toolkit`.
5. [ ] Abrir una sesión con el plugin desde otra carpeta (para que no se cargue también el `.claude/` suelto), con las variables de 2.2: `claude --plugin-dir <ruta>/plugin/sdd-toolkit`. Repetir 2.3, 6.2 y 6.3.
6. [ ] Confirmar cómo llega `userConfig` al skill; si no se sustituye en el texto, el bucket sigue entrando por `GCSGREP_BUCKET`.
7. [ ] Guardar todo en `evidencia/06-plugin.md`: validate, tools con nombres nuevos y el muro repetido.

**Listo cuando:** `validate` pasa y el veto aparece con el nombre
`mcp__plugin_sdd-toolkit_gcs__write_object_safe`.

### 8 · Contexto, README y nota

1. [ ] `/context` con el server conectado → `evidencia/07-contexto.md`.
2. [ ] `README.md`: qué hace, qué paso SDD alimenta, cómo se instala el plugin.
3. [ ] Nota de seguridad con los 7 puntos del enunciado, siguiendo `ejemplo/nota-seguridad.md`. Incluye justificar la key de larga vida frente a impersonation.

### 9 · Cierre

1. [ ] Buscar secretos en lo que se va a commitear: `private_key`, `lectora.json`, paths absolutos a keys.
2. [ ] Releer cada evidencia contra la tabla del enunciado.
3. [ ] Commit y mandar el hash.

## Riesgos

- **Los e2e del TP1 cambian**: el objeto hostil contiene `timeout` y suma un objeto al bucket. Aceptado.
- **Rol de `lectora` a nivel proyecto**: no cumple el enunciado; se corrige en 1.4.
- **Server con la identidad equivocada**: si se abre `claude` sin las variables, el server usa el ADC personal. Se comprueba con 2.4 en cada sesión de demo.
- **Nombres de tools distintos del enunciado**: se documenta en `01-descubrimiento` y en la nota de seguridad.
- **Key de larga vida**: el enunciado sugiere impersonation; se justifica en la nota de seguridad.
