#!/usr/bin/env bash
# adr-immutable.sh — un ADR aceptado no se reescribe.
#
# Evento: PreToolUse, matcher "Edit|Write".
# Efecto: bloquea cualquier edición de un ADR cuyo estado es "Aceptado", salvo el
#         único cambio permitido: reemplazar la línea "Estado: …" (lo que hace
#         supersede-adr para marcarlo como reemplazado).
#
# Contrato del hook:
#   - Recibe el evento como JSON por stdin.
#   - exit 0  → deja pasar la acción.
#   - exit 2  → VETA la acción; stderr se le muestra al agente.
#   - Cualquier otro código → error no bloqueante (la acción pasa igual).
#
# Requiere: jq
set -uo pipefail

EVENT="$(cat)"

if ! command -v jq >/dev/null 2>&1; then
  echo "adr-immutable: falta jq; no se puede inspeccionar el evento" >&2
  exit 1
fi

FILE_PATH="$(printf '%s' "$EVENT" | jq -r '.tool_input.file_path // ""')"
[ -z "$FILE_PATH" ] && exit 0

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(pwd)}"
ADR_DIR="docs/adr"   # Ajustalo a tu proyecto.

REL_PATH="${FILE_PATH#"$PROJECT_DIR"/}"

# --- ¿es un ADR existente? ----------------------------------------------------
case "$REL_PATH" in
  "$ADR_DIR"/README.md) exit 0 ;;   # el índice lo regenera adr-index.sh
  "$ADR_DIR"/*.md) ;;
  *) exit 0 ;;
esac

ABS_PATH="$PROJECT_DIR/$REL_PATH"
[ -f "$ABS_PATH" ] || exit 0          # ADR nuevo: se puede escribir

# --- ¿está aceptado? ----------------------------------------------------------
grep -qE '^Estado:[[:space:]]*Aceptado' "$ABS_PATH" || exit 0

# --- la única excepción: cambiar la línea de estado ---------------------------
TOOL="$(printf '%s' "$EVENT" | jq -r '.tool_name // ""')"
if [ "$TOOL" = "Edit" ]; then
  OLD="$(printf '%s' "$EVENT" | jq -r '.tool_input.old_string // ""')"
  NEW="$(printf '%s' "$EVENT" | jq -r '.tool_input.new_string // ""')"
  one_line() { [ "$(printf '%s' "$1" | wc -l | tr -d ' ')" = "0" ]; }
  if one_line "$OLD" && one_line "$NEW" \
     && printf '%s' "$OLD" | grep -qE '^Estado:' \
     && printf '%s' "$NEW" | grep -qE '^Estado:'; then
    exit 0
  fi
fi

# --- bloquear -----------------------------------------------------------------
{
  echo "EDICIÓN BLOQUEADA — $REL_PATH es un ADR aceptado, y un ADR aceptado no se reescribe."
  echo
  echo "Si la decisión cambió, usá el skill supersede-adr:"
  echo "  1. escribí un ADR nuevo que diga 'Reemplaza a' este;"
  echo "  2. en este archivo, cambiá SOLO la línea 'Estado:' a"
  echo "     'Estado: Reemplazado por ADR-NNNN'. Ese cambio sí pasa."
} >&2

exit 2
