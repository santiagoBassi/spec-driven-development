# TP4 — Specs de `gcsgrep` contra un bucket real

El toolkit alimenta el paso SDD **Especificar los VCs**: `write-vcs-con-bucket`
lee fixtures reales de GCS por MCP, a través de `lector-fixtures`, antes de escribir
la spec. Cada caso de verificación cita objeto y líneas reales; la procedencia
registra `generation`, checksum, `updated`, fecha, cuenta y versión del server.
El contenido se lee por MCP; un helper de metadata completa los campos nativos
que el server 0.6.0 omite, usando la misma credencial lectora.

La demo produjo [`sdd/invert-match-spec.md`](sdd/invert-match-spec.md) para `-v`:
29 requisitos con su VC. La spec queda en borrador y su aprobación final requiere
completar las iteraciones 2 y 3 del TP1 (sniffing y errores de GCS), registrar
la nueva base y medir sus suites. Se conservan esos VCs por decisión del equipo.
La implementación de `-v` queda fuera de este paso del TP4 y sigue bloqueada
por esos prerrequisitos y el gate de revisión.

## Instalación del plugin

Requiere Claude Code (probado con 2.1.296), Node.js con `npx`, Python 3, gcloud CLI, Bash, `jq` y Git.
Cada equipo debe usar su propio bucket sintético y una cuenta con
`roles/storage.objectViewer` solo sobre ese bucket. Las credenciales se preparan
fuera del paquete: este skill requiere `GOOGLE_APPLICATION_CREDENTIALS` en el
entorno del shell, apuntando a una key de la service account lectora. El helper
imprime únicamente `client_email`; no se incluye la key ni su ruta real.

Desde el repositorio donde se quiera usar el toolkit, instalar con un comando
(reemplazar los tres valores de ejemplo):

```sh
claude plugin install sdd-toolkit --marketplace /ruta/al/checkout/04-mcps-\&-plugins \
  --scope project \
  --config project=MI_PROYECTO_DE_PRUEBA \
  --config bucket=MI_BUCKET_SINTETICO
```

`project` y `bucket` se declaran en `userConfig`, sin secretos. Claude Code los
guarda como opciones del plugin. El server recibe los valores por su configuración
MCP; el hook recibe `CLAUDE_PLUGIN_OPTION_BUCKET`; el skill recibe el bucket en su
contenido. La cuenta lectora entra por el entorno, sin sustituirla en el manifest.

Para probar el paquete sin instalarlo, iniciar una sesión desde otra carpeta:

```sh
claude --plugin-dir /ruta/al/checkout/04-mcps-\&-plugins/plugin/sdd-toolkit
```

Configurar proyecto y bucket en `/plugin` → Configure. Con `--plugin-dir`, el ID
local de configuración es `sdd-toolkit@inline`; instalado desde el marketplace es
`sdd-toolkit@sdd-course`. Reiniciar la sesión después de configurar el server.
La sesión debe heredar la credencial lectora. Abrir desde otra carpeta evita
cargar al mismo tiempo el toolkit suelto de `.claude/` y el plugin.

## Uso y verificaciones

- `/sdd-toolkit:write-vcs-con-bucket especificá -v para gcsgrep`: lee el bucket,
  escribe VCs y procedencia, compara `generation`, checksum y `updated` al cierre, ejecuta el control de
  cobertura y pide revisión a `sdd-toolkit:review-spec`.
- `cerrá la spec de -v`: compara la metadata actual con la registrada. Si cambió,
  marca la spec como vencida y pide releer.
- `/sdd-toolkit:write-spec`: especificación de cambios que no necesitan el bucket.

El lector tiene únicamente `list_objects`, `read_object_metadata` y
`read_object_content`. El skill principal no lee contenido de objetos. `procedencia.py` ejecuta
únicamente `gcloud storage objects describe --raw`, fuerza la misma credencial
del server por comando y devuelve solo identidad, hashes y fecha. No cambia la
cuenta activa de gcloud. Si faltan
el server, la identidad o la configuración, el flujo se detiene sin inventar VCs.
No se implementa un cambio antes de obtener `LISTA` del revisor.

Desde esta carpeta:

```sh
claude plugin validate --strict plugin/sdd-toolkit
claude plugin validate --strict .
python3 evidencia/probar-plugin.py
evidencia/probar-hook.sh
```

Las pruebas locales no usan credenciales ni tocan GCS. Las sesiones reales y los
rechazos de GCS se pueden leer en `evidencia/` sin ejecutar nada.

## Nota de seguridad

1. **Server y sistema:** `@google-cloud/storage-mcp@0.6.0`, fijado tanto en la
   configuración suelta como en el plugin. Demo contra `gs://sdd-fardenghi-itba`,
   proyecto `sat-spec-driven-development`; otros equipos configuran su proyecto
   y bucket de prueba.
2. **Modo y tools:** se ejecuta sin `--enable-destructive-tools`. Expone 15 tools:
   las tres lecturas de objetos; `create_bucket`, `write_object_safe`,
   `upload_object_safe`, `copy_object_safe`; y `list_buckets`, `get_bucket_metadata`,
   `get_bucket_location`, `view_iam_policy`, `check_iam_permissions`,
   `execute_insights_query`, `get_metadata_table_schema`, `list_insights_configs`.
   No expone `delete_object`. El modo todavía permite crear: por eso se necesitan
   el hook y la cuenta lectora.
3. **Credencial:** `lectora@sat-spec-driven-development.iam.gserviceaccount.com`
   tiene `roles/storage.objectViewer` sobre el bucket y ningún rol en el proyecto
   ([IAM](evidencia/02-iam.md)). La demo usa una key local, ignorada por Git y
   referenciada solo por el entorno. Se mantuvo esa identidad para reproducir las
   entregas previas; una key de larga vida requiere custodia y rotación. Para otro
   equipo conviene migrar a ADC con impersonación de su cuenta lectora: esa opción
   requiere adaptar el helper de identidad, que hoy exige la key de service account.
4. **Datos:** 25 objetos sintéticos: logs, texto con bordes, gzip, binarios,
   prefijos anidados y un fixture hostil que apunta a un dominio `.invalid`.
   No se usan datos de producción.
5. **Peor caso y capas:** primero el hook veta toda tool fuera de la allowlist y
   toda lectura de otro bucket. Si se omite ese hook, el modo del server todavía
   no ofrece borrado. Si el server intenta crear o borrar, GCS rechaza la operación
   con la cuenta lectora. Se observaron el veto real y los `403` de creación y
   borrado; el borrado agregó una precondición que protege el fixture existente.
   [Muro](evidencia/04-muro.md), [prueba con el plugin](evidencia/06-plugin.md).
6. **Trifecta:** las tres patas son un dato privado (la credencial local y los
   archivos de la máquina), texto ajeno (el contenido de los objetos) y un canal
   hacia afuera (Bash con red). La sesión principal tiene la primera y la tercera.
   Se cortó el canal hacia afuera en el lugar donde entra el texto ajeno: el
   contenido lo lee el subagente, que no tiene Bash, archivos locales ni red, solo
   las tres lecturas MCP. Devuelve aparte las primeras 40 letras de las líneas que
   parecen instrucciones, sin transmitir el comando completo. La demo registró
   esas líneas como datos y ningún intento de ejecutarlas
   ([fixture hostil](evidencia/05-fixture-hostil.md)).
   **Riesgo residual:** el corte tiene dos límites. El hook permite
   `read_object_content` a cualquier agente: que la sesión principal no la llame
   depende de las instrucciones de `CLAUDE.md` y del skill, no de un veto. Y qué
   línea "parece una instrucción" lo decide el modelo del lector: una que no
   marque llega completa a la sesión principal cuando se pide `texto_completo`.
   En ese caso el daño no alcanza al bucket (la cuenta solo lee), pero sí podría
   alcanzar a lo que Bash tenga a mano en la máquina.
7. **Contexto:** `/context` contabilizó **3.469 tokens** en los esquemas de las 15
   tools GCS; los dos agentes sumaron 351 y las descripciones de los dos skills
   aproximadamente 410. Los esquemas pueden quedar diferidos: ese costo no es
   igual al costo activo de cada turno. La medición incluye otros conectores y
   separa el aporte GCS en [07-contexto](evidencia/07-contexto.md). Habilitar el
   plugin en proyectos que necesiten el bucket, no globalmente.

## Estado de entrega

Conexión, IAM, skill, lector, muro, fixture hostil, plugin y medición de contexto
tienen evidencia. Se probó el vencimiento de la spec al renovar `logs/app/api.log`
con idéntico contenido y volver a comparar su metadata. La relectura y revisión
quedan registradas en [03-skill](evidencia/03-skill.md).

La procedencia nativa se obtiene con la cuenta lectora, además de la metadata MCP.
El helper de metadata complementa el MCP: los fixtures y las líneas de los VCs
siguen entrando exclusivamente por el subagente. La consulta usa el modo de
[metadata de Objects.get](https://docs.cloud.google.com/storage/docs/json_api/v1/objects/get)
y los [campos generation y hashes de GCS](https://docs.cloud.google.com/storage/docs/json_api/v1/objects).
No se pide contenido por la CLI ni se hace fallback a la cuenta dueña.

El [plan](sdd/tp4-plan.md) registra la revisión y el commit final.

Referencias del formato de empaquetado:
[manifest y userConfig](https://code.claude.com/docs/en/plugins-reference).
 