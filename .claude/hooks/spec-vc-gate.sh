#!/usr/bin/env bash
# spec-vc-gate.sh — no entra una spec con un requisito sin VC.
#
# Evento: PreToolUse.
#   - matcher "Edit|Write": si el archivo es un *-spec.md, arma el contenido como
#     quedaría después de la edición y lo chequea. Bloquea antes de escribir.
#   - matcher "Bash": si el comando es un `git commit`, chequea cada *-spec.md que
#     entra en el commit (lo que está en staging). Bloquea el commit.
# El chequeo es .claude/skills/write-spec/scripts/vc-coverage.sh: el mismo script que
# el skill corre para chequearse. El skill persuade; este hook garantiza.
#
# Contrato: exit 0 deja pasar · exit 2 VETA (stderr le llega al agente) ·
#           otro código = error no bloqueante.
# Requiere: jq, git.
set -uo pipefail

EVENT="$(cat)"
command -v jq >/dev/null 2>&1 || { echo "spec-vc-gate: falta jq" >&2; exit 1; }

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-$(pwd)}"
CHECK="$PROJECT_DIR/.claude/skills/write-spec/scripts/vc-coverage.sh"
TOOL="$(printf '%s' "$EVENT" | jq -r '.tool_name // ""')"

block() {
  {
    echo "$1"
    cat "$ERR"
    echo
    echo "Regla de la spec: cada FR, BR, INV y NFR tiene un VC con su mismo ID, y cada VC"
    echo "corresponde a un requisito declarado (VC-12 ↔ FR-12, VC-BR-2 ↔ BR-2)."
    echo "$2"
  } >&2
  exit 2
}
ERR="$(mktemp)"; trap 'rm -f "$ERR"' EXIT

# Una spec es un *-spec.md fuera de .claude/ (ahí viven write-spec, review-spec.md, …).
is_spec() { case "$1" in .claude/*|*/.claude/*) return 1 ;; *-spec.md) return 0 ;; *) return 1 ;; esac; }
specs() { while IFS= read -r f; do is_spec "$f" && printf '%s\n' "$f"; done; }

case "$TOOL" in
  Edit|Write|MultiEdit)
    FILE="$(printf '%s' "$EVENT" | jq -r '.tool_input.file_path // ""')"
    is_spec "${FILE#"$PROJECT_DIR"/}" || exit 0
    # El contenido después de la edición: Write lo trae entero; Edit/MultiEdit se
    # aplican sobre el archivo actual (split/join evita índices, que con UTF-8 mienten).
    AFTER="$(printf '%s' "$EVENT" | jq -r --rawfile cur "${FILE}" '
      def apply($s; $e):
        ($s | split($e.old_string)) as $p
        | if ($p | length) < 2 then $s
          elif ($e.replace_all // false) then $p | join($e.new_string)
          else $p[0] + $e.new_string + ($p[1:] | join($e.old_string)) end;
      .tool_input as $t
      | if .tool_name == "Write" then $t.content
        elif .tool_name == "Edit" then apply($cur; $t)
        else reduce $t.edits[] as $e ($cur; apply(.; $e)) end' 2>/dev/null)" \
      || AFTER="$(printf '%s' "$EVENT" | jq -r '.tool_input.content // ""')"   # Write de un archivo nuevo
    printf '%s\n' "$AFTER" | "$CHECK" - "${FILE#"$PROJECT_DIR"/}" 2>"$ERR" && exit 0
    block "EDICIÓN BLOQUEADA — esta escritura deja la spec con requisitos sin VC." \
          "Agregá el VC que falta (o sacá el que sobra) en la misma edición y volvé a escribir."
    ;;
  Bash)
    CMD="$(printf '%s' "$EVENT" | jq -r '.tool_input.command // ""')"
    printf '%s' "$CMD" | grep -qE '(^|[;&|[:space:]"'"'"'(`])git([[:space:]]+-C[[:space:]]+[^[:space:]]+)?[[:space:]]+commit([[:space:]"'"'"';)]|$)' || exit 0
    cd "$PROJECT_DIR" || exit 1
    # El hook corre ANTES del comando entero: en `git add x && git commit` o en
    # `git commit -a`, lo que va a entrar todavía está en disco, no en staging.
    # En esos casos se chequea la versión en disco de toda spec modificada o nueva.
    DISK=""
    if printf '%s' "$CMD" | grep -qE 'git[[:space:]]+add|[[:space:]](-[a-zA-Z]*a[a-zA-Z]*|--all)([[:space:]]|$)'; then
      DISK="$( { git diff --name-only --diff-filter=AM; git ls-files --others --exclude-standard; } \
               | specs | sort -u)"
    fi
    STAGED="$(git diff --cached --name-only --diff-filter=AM | specs)"
    FAILED=0
    for f in $DISK; do
      "$CHECK" "$f" 2>>"$ERR" || FAILED=1
    done
    # Lo que ya está en staging entra tal como está ahí (no como está en disco).
    for f in $STAGED; do
      printf '%s\n' "$DISK" | grep -qxF -- "$f" && continue
      git show ":$f" | "$CHECK" - "$f" 2>>"$ERR" || FAILED=1
    done
    [ "$FAILED" = 0 ] && exit 0
    block "COMMIT BLOQUEADO — una spec del commit tiene requisitos sin VC." \
          "Corregí la spec (skill write-spec), hacé git add y volvé a commitear."
    ;;
esac
exit 0
