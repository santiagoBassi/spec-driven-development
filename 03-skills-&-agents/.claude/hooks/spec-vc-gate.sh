#!/usr/bin/env bash
# spec-vc-gate.sh — no entra una spec con un requisito sin VC.
#
# Evento: PreToolUse.
#   - matcher "Edit|Write|MultiEdit": si el archivo es un *-spec.md, arma el contenido
#     como quedaría después de la edición y lo chequea. Bloquea antes de escribir.
#   - matcher "Bash": si el comando es un `git commit`, chequea cada *-spec.md que
#     puede entrar en el commit: la versión en disco de toda spec modificada o nueva,
#     y la versión en staging. Bloquea el commit.
# El chequeo es .claude/skills/write-spec/scripts/vc-coverage.sh: el mismo script que
# el skill corre para chequearse. El skill persuade; este hook garantiza.
#
# Contrato: exit 0 deja pasar · exit 2 VETA (stderr le llega al agente) ·
#           otro código = error no bloqueante.
# Requiere: jq, git.
set -uo pipefail

EVENT="$(cat)"
if ! command -v jq >/dev/null 2>&1; then
  # Sin jq no se puede leer el evento. Si puede ser una spec o un commit, se veta:
  # un guardrail que falla abierto no garantiza nada.
  case "$EVENT" in
    *-spec.md*|*commit*)
      echo "spec-vc-gate: falta jq, no puedo chequear la cobertura de VCs. Instalá jq y reintentá." >&2
      exit 2 ;;
  esac
  exit 0
fi

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
# Lee rutas separadas por NUL (git -z: sin comillas ni escapes, aunque tengan acentos).
specs() { tr '\0' '\n' | while IFS= read -r f; do is_spec "$f" && printf '%s\n' "$f"; done; }

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
    block "EDICIÓN BLOQUEADA — así como quedaría, la spec no pasa la cobertura de VCs." \
          "Agregá el VC que falta (o sacá el que sobra) en la misma edición y volvé a escribir."
    ;;
  Bash)
    CMD="$(printf '%s' "$EVENT" | jq -r '.tool_input.command // ""')"
    # `git commit`, con opciones globales en el medio (`git -c k=v commit`,
    # `git -C "un dir" commit`, `git --no-pager commit`) o con ruta (`/usr/bin/git`).
    Q="\"'"
    OPT="[[:space:]]+-[^[:space:]]*([[:space:]]+(\"[^\"]*\"|'[^']*'|[^-[:space:]][^[:space:]]*))?"
    printf '%s' "$CMD" | grep -qE "(^|[;&|[:space:]$Q(\`{/])git($OPT)*[[:space:]]+commit([[:space:]$Q;)]|\$)" || exit 0
    # Git da las rutas relativas a la raíz del repo, aunque la sesión esté abierta en
    # una subcarpeta (03-skills-&-agents/): se trabaja desde la raíz.
    cd "$PROJECT_DIR" && cd "$(git rev-parse --show-toplevel)" || exit 1
    # El hook corre ANTES del comando entero, y no sabe qué va a entrar: con
    # `git add x && git commit`, `git commit -a` o `git commit <archivo>` entra lo que
    # está en disco, no lo que está en staging. Por eso se chequean SIEMPRE las dos
    # versiones de toda spec modificada o nueva: la de disco y la de staging. Un
    # `git commit -am` con el staging todavía roto se veta aunque el disco esté sano
    # (falso positivo conservador: hay que hacer `git add` antes).
    DISK="$( { git diff -z --name-only --diff-filter=ACMR; git ls-files -z --others --exclude-standard; } \
             | specs | sort -u)"
    STAGED="$(git diff -z --cached --name-only --diff-filter=ACMR | specs)"
    FAILED=0
    while IFS= read -r f; do
      [ -n "$f" ] || continue
      "$CHECK" "$f" 2>>"$ERR" || FAILED=1
    done <<EOF_DISK
$DISK
EOF_DISK
    while IFS= read -r f; do
      [ -n "$f" ] || continue
      git show ":$f" | "$CHECK" - "$f (en staging)" 2>>"$ERR" || FAILED=1
    done <<EOF_STAGED
$STAGED
EOF_STAGED
    [ "$FAILED" = 0 ] && exit 0
    block "COMMIT BLOQUEADO — una spec modificada no pasa la cobertura de VCs." \
          "Corregí la spec (skill write-spec), hacé git add y volvé a commitear."
    ;;
esac
exit 0
