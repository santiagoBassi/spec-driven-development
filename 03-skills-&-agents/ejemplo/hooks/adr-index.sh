#!/usr/bin/env bash
# adr-index.sh — mantiene al día el índice de ADRs.
#
# Evento: PostToolUse, matcher "Edit|Write".
# Efecto: cuando se escribe un ADR en docs/adr/, regenera docs/adr/README.md con la
#         tabla de número, título y estado. No bloquea nada: la acción ya ocurrió.
#
# Es el ejemplo de un hook que REACCIONA en vez de vetar. Siempre sale 0.
#
# Requiere: jq
set -uo pipefail

EVENT="$(cat)"
command -v jq >/dev/null 2>&1 || exit 0

FILE_PATH="$(printf '%s' "$EVENT" | jq -r '.tool_input.file_path // ""')"
[ -z "$FILE_PATH" ] && exit 0

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(pwd)}"
ADR_DIR="docs/adr"   # Ajustalo a tu proyecto.

REL_PATH="${FILE_PATH#"$PROJECT_DIR"/}"
case "$REL_PATH" in
  "$ADR_DIR"/README.md) exit 0 ;;
  "$ADR_DIR"/[0-9][0-9][0-9][0-9]-*.md) ;;
  *) exit 0 ;;
esac

INDEX="$PROJECT_DIR/$ADR_DIR/README.md"
{
  echo "# Decisiones de arquitectura"
  echo
  echo "<!-- Generado por .claude/hooks/adr-index.sh. No editar a mano. -->"
  echo
  echo "| ADR | Decisión | Estado |"
  echo "|---|---|---|"
  for f in "$PROJECT_DIR/$ADR_DIR"/[0-9][0-9][0-9][0-9]-*.md; do
    [ -e "$f" ] || continue
    name="$(basename "$f")"
    title="$(grep -m1 -E '^# ' "$f" | sed -E 's/^# (ADR-[0-9]+: )?//')"
    estado="$(grep -m1 -E '^Estado:' "$f" | sed -E 's/^Estado:[[:space:]]*//')"
    echo "| [${name%%-*}](./$name) | ${title:-?} | ${estado:-?} |"
  done
} > "$INDEX"

exit 0
