#!/usr/bin/env bash
# gcs-allowlist.sh — PreToolUse sobre las tools MCP del server gcs.
#
# Allowlist, no denylist: deja pasar las tres tools de lectura de objetos y veta todo lo
# demás, incluidas las que crean (`*_safe`, create_bucket), las otras de lectura (bucket,
# IAM, Insights) y cualquier tool nueva que traiga la próxima versión del server.
# Las de lectura pasan solo contra el bucket de $bucket configurado: es el caso en que el
# hook necesita mirar los argumentos, no solo el nombre.
#
# Es la tercera capa, no la garantía. Las dos de abajo no dependen de este archivo: el
# server sin --enable-destructive-tools y, la que garantiza, el rol objectViewer de la
# cuenta lectora sobre el bucket.
#
# El matcher y los nombres usan el namespace del plugin:
#   mcp__plugin_sdd-toolkit_gcs__<tool>
#
# Contrato: exit 0 deja pasar · exit 2 VETA (stderr le llega al agente).
# Requiere: jq.
set -uo pipefail

INPUT=$(cat)
if ! command -v jq >/dev/null 2>&1; then
  # Sin jq no se puede leer el evento: un guardrail que falla abierto no garantiza nada.
  echo "gcs-allowlist: falta jq, no puedo saber qué tool se pidió. Instalá jq y reintentá." >&2
  exit 2
fi

TOOL=$(jq -r '.tool_name // ""' <<<"$INPUT" 2>/dev/null) || TOOL=""
NAME=$(sed -E 's/^mcp__(plugin_sdd-toolkit_)?gcs__//' <<<"$TOOL")
PERMITIDO="${CLAUDE_PLUGIN_OPTION_BUCKET:-${GCSGREP_BUCKET:-}}"

case "$NAME" in
  list_objects|read_object_metadata|read_object_content)
    if [[ -z "$PERMITIDO" ]]; then
      echo "$TOOL bloqueada: No hay bucket configurado (CLAUDE_PLUGIN_OPTION_BUCKET ni GCSGREP_BUCKET), así que no sé contra qué bucket se puede leer. Pedile al usuario que configure el bucket del plugin." >&2
      exit 2
    fi
    BUCKET=$(jq -r '.tool_input.bucket_name // ""' <<<"$INPUT" 2>/dev/null) || BUCKET=""
    if [[ "$BUCKET" != "$PERMITIDO" ]]; then
      echo "$TOOL bloqueada: pidió el bucket '$BUCKET' y este toolkit solo lee gs://$PERMITIDO. Repetí la llamada con bucket_name=\"$PERMITIDO\"." >&2
      exit 2
    fi
    exit 0 ;;
esac

echo "${TOOL:-(tool sin nombre)} no está en la allowlist: este toolkit solo lee objetos (list_objects, read_object_metadata, read_object_content) de gs://${PERMITIDO:-\$GCSGREP_BUCKET}. No reintentes con otra tool; si hace falta escribir o borrar, decíselo al usuario para que lo haga fuera del agente." >&2
exit 2
