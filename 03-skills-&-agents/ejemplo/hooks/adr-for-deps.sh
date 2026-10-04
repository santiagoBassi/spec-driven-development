#!/usr/bin/env bash
# adr-for-deps.sh — sumar o cambiar una dependencia es una decisión.
#
# Evento: PreToolUse, matcher "Bash".
# Efecto: si el comando es un `git commit` y lo que está en staging toca un
#         manifiesto de dependencias, exige que el mismo commit agregue un ADR
#         nuevo en docs/adr/. Si no, bloquea el commit.
#
# Contrato del hook:
#   - Recibe el evento como JSON por stdin.
#   - exit 0  → deja pasar la acción.
#   - exit 2  → VETA la acción; stderr se le muestra al agente.
#   - Cualquier otro código → error no bloqueante (la acción pasa igual).
#
# Requiere: jq, git
set -uo pipefail

EVENT="$(cat)"

if ! command -v jq >/dev/null 2>&1; then
  echo "adr-for-deps: falta jq; no se puede inspeccionar el evento" >&2
  exit 1
fi

COMMAND="$(printf '%s' "$EVENT" | jq -r '.tool_input.command // ""')"

# --- ¿es un commit? -----------------------------------------------------------
if ! printf '%s' "$COMMAND" | grep -qE '(^|[;&|[:space:]])git[[:space:]]+commit([[:space:]]|$)'; then
  exit 0
fi

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(pwd)}"
cd "$PROJECT_DIR" || exit 1

# Ajustá estas dos variables a tu proyecto.
ADR_DIR="docs/adr"
DEPS_REGEX='(^|/)(pyproject\.toml|requirements[^/]*\.txt|package\.json|go\.mod|Cargo\.toml)$'

STAGED="$(git diff --cached --name-only 2>/dev/null)"
DEPS_CHANGED="$(printf '%s\n' "$STAGED" | grep -E "$DEPS_REGEX" || true)"
[ -z "$DEPS_CHANGED" ] && exit 0

NEW_ADRS="$(git diff --cached --name-only --diff-filter=A -- "$ADR_DIR" 2>/dev/null \
            | grep -E '/[0-9]{4}-[^/]*\.md$' || true)"
[ -n "$NEW_ADRS" ] && exit 0

# --- bloquear -----------------------------------------------------------------
{
  echo "COMMIT BLOQUEADO — cambia una dependencia y no hay un ADR nuevo."
  echo
  echo "Manifiestos en este commit:"
  printf '  %s\n' $DEPS_CHANGED
  echo
  echo "Sumar o cambiar una dependencia es una decisión de arquitectura."
  echo "Escribí el ADR con el skill write-adr (por qué esta y no otra),"
  echo "agregalo al staging con git add $ADR_DIR/ y volvé a commitear."
} >&2

exit 2
