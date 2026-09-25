# Fixtures de prueba para gcsgrep

Este directorio contiene archivos de prueba diseñados para ejercitar todos los casos de uso,
reglas de negocio y casos borde de `gcsgrep`.

## Estructura de archivos y qué prueba cada uno

| Ruta relativa | Propósito / Caso de prueba |
|---|---|
| `logs/server.log` | Archivo de texto en la raíz de un prefijo virtual |
| `logs/app/api.log` | Múltiples matches (`timeout`, `TIMEOUT`, `Timeout`), números de línea (`-n`), conteo (`-c`), case-insensitive (`-i`) |
| `logs/app/worker.log` | Formato clave-valor estructurado para probar regex RE2 (`job_id=\d+ status=ERROR`) |
| `logs/db/postgres.log` | Texto plano con errores de BD y `statement timeout` |
| `logs/db/redis.log` | Archivo limpio **sin matches**, para probar exit code `1` y exclusión en `-c` y `-l` |
| `logs/archive/old_logs.log.gz` | Archivo comprimido `.gz` real, para verificar que se saltea como no-texto (BR-7) |
| `data/users.csv` | Archivo tabular CSV con matches en campos específicos |
| `data/config.json` | Archivo JSON multilínea con claves de configuración (`timeout_seconds`) |
| `data/windows_crlf.txt` | Texto con saltos de línea Windows (`\r\n`) para verificar recorte de `\r` (FR-9a) |
| `data/subdata/deep_nested.txt` | Archivo anidado en subdirectorio profundo para probar búsqueda recursiva |
| `data/acentos.log` | Única línea `ÑANDÚ ÁRBOL`, para probar `-i` con letras no ASCII (FR-7) |
| `data/sin_salto_final.txt` | Dos líneas; la última no termina en `\n` (el archivo no termina en `0a`), para probar que se busca igual (FR-9b). No contiene `timeout` |
| `edge-cases/empty.txt` | Archivo de 0 bytes (no debe romper el procesamiento ni conteos) |
| `edge-cases/binary_nullbyte.bin` | Archivo binario con bytes nulos (`0x00`) para verificar el sniffing (BR-6) |
| `edge-cases/sample_image.png` | Archivo PNG binario real para verificar omisión con aviso |
| `edge-cases/non_utf8_latin1.txt` | Archivo con caracteres ISO-8859-1 (no UTF-8), se clasifica como no-texto (BR-6) |
| `edge-cases/small_under_512.txt` | Archivo de texto chico (< 512 bytes) para probar sniffing sin desborde |
| `edge-cases/long_line_exceeds_1mb.log` | Archivo con una línea de > 1 MB para probar BR-8 (truncamiento y buffer acotado) |
| `sniffing/utf8_split_at_512.txt` | Un carácter UTF-8 cruza el byte 512 de la muestra |
| `sniffing/truncated_utf8_under_512.txt` | Secuencia UTF-8 incompleta al final de un objeto chico |
| `sniffing/invalid_after_512.txt` | Byte inválido después de la muestra; debe imprimirse tal cual |
| `sniffing/text_named.png` | Texto con extensión `.png` y `Content-Type: image/png` |
| `sniffing/gzip_no_extension` | Gzip real sin extensión ni `Content-Encoding: gzip` |
| `sniffing/transcoded.log` | Texto que se sube comprimido, con `Content-Encoding: gzip` y sin `Cache-Control: no-transform` |

## Cómo subir estos archivos a Google Cloud Storage

Suponiendo que tenés un bucket creado llamado `mi-bucket-pruebas`:

```bash
# Subir únicamente las carpetas de fixtures; README.md contiene "timeout" y
# alteraría los resultados exactos de los VCs sobre la raíz del bucket.
gcloud storage cp -r test-fixtures/logs test-fixtures/data test-fixtures/edge-cases gs://mi-bucket-pruebas/
gcloud storage cp -r test-fixtures/sniffing gs://mi-bucket-pruebas/

# Corregir la metadata del objeto que prueba decompressive transcoding.
# No usar -Z: agrega Cache-Control: no-transform.
gzip -c test-fixtures/sniffing/transcoded.log > /tmp/gcsgrep-transcoded.log.gz
gcloud storage cp /tmp/gcsgrep-transcoded.log.gz gs://mi-bucket-pruebas/sniffing/transcoded.log --content-encoding=gzip --content-type=text/plain --cache-control=no-cache
gcloud storage objects describe gs://mi-bucket-pruebas/sniffing/transcoded.log --format='yaml(contentEncoding,cacheControl,contentType)'
```

## Ejemplos de comandos para probar gcsgrep contra el bucket

```bash
# 1. Búsqueda básica en todo el bucket:
gcsgrep "timeout" gs://mi-bucket-pruebas/

# 2. Búsqueda en prefijo específico con números de línea:
gcsgrep -n "timeout" gs://mi-bucket-pruebas/logs/

# 3. Búsqueda case-insensitive:
gcsgrep -i "timeout" gs://mi-bucket-pruebas/logs/app/

# 4. Solo listar archivos que matchean (-l):
gcsgrep -l "timeout" gs://mi-bucket-pruebas/

# 5. Conteo de matches por archivo (-c):
gcsgrep -c "timeout" gs://mi-bucket-pruebas/

# 6. Búsqueda con regex RE2:
gcsgrep -E 'job_id=\d+ queue=\w+ status=ERROR' gs://mi-bucket-pruebas/logs/

# 7. Búsqueda sin matches (debe retornar exit code 1):
gcsgrep "palabra_inexistente_xyz" gs://mi-bucket-pruebas/

# 8. Guardrail de costo (--max 5 en bucket con >5 objetos -> debe abortar con exit code 2):
gcsgrep --max 5 "timeout" gs://mi-bucket-pruebas/
```
