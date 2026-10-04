#!/usr/bin/env bash
# probar-hook.sh — ejercita spec-vc-gate.sh sin el agente, pasándole el evento por stdin
# (es lo que hace Claude Code). Arma un repo descartable en un directorio temporal,
# así que no toca este repo. Salida esperada: la de 01-hook-aislado.txt.
#
# Uso, desde la raíz del repo:  03-skills-\&-agents/evidencia/probar-hook.sh
set -uo pipefail

REPO="$(git rev-parse --show-toplevel)"
SB="$(mktemp -d)"; trap 'rm -rf "$SB"' EXIT
mkdir -p "$SB/sdd" && cp -R "$REPO/.claude" "$SB/" && cd "$SB" && git init -q

cat > sdd/demo-spec.md <<'EOF'
# demo — spec

#### FR-1 · Buscar un literal
> **VC-1** — `./demo x gs://b/` sale con código 0.

#### FR-2 · Informar una corrida sin matches
> **VC-2** — `./demo zzz gs://b/` sale con código 1.

### BR-1 · No escribe en el bucket
> **VC-BR-1** — el servidor de prueba no recibe requests de escritura.
EOF
git add -A && git -c user.name=t -c user.email=t@t commit -qm base

export CLAUDE_PROJECT_DIR="$SB"
F="$SB/sdd/demo-spec.md"
caso() {  # caso <número> <descripción> <exit esperado> <evento JSON>
  printf '\n=== Caso %s · %s (esperado: exit %s)\n' "$1" "$2" "$3"
  printf '%s' "$4" | .claude/hooks/spec-vc-gate.sh
  local got=$?
  printf -- '--> exit=%s %s\n' "$got" "$([ "$got" = "$3" ] && echo OK || echo 'NO COINCIDE')"
}
edit() { jq -nc --arg f "$F" --arg o "$1" --arg n "$2" '{tool_name:"Edit",tool_input:{file_path:$f,old_string:$o,new_string:$n}}'; }
bash_ev() { jq -nc --arg c "$1" '{tool_name:"Bash",tool_input:{command:$c}}'; }

caso 1 "Edit que agrega FR-3 sin su VC" 2 \
  "$(edit '### BR-1' $'#### FR-3 · Invertir el match con -v\nDado … Cuando … Entonces …\n\n### BR-1')"
caso 2 "Edit que agrega FR-3 con VC-3" 0 \
  "$(edit '### BR-1' $'#### FR-3 · Invertir el match con -v\n> **VC-3** — sale con código 0.\n\n### BR-1')"
caso 3 "Write de una spec nueva con un VC que no verifica nada" 2 \
  "$(jq -nc --arg f "$SB/sdd/nueva-spec.md" '{tool_name:"Write",tool_input:{file_path:$f,content:"#### FR-1 · a\n> **VC-1** — x\n> **VC-INV-1** — y\n"}}')"
caso 4 "Write de un archivo que no es *-spec.md" 0 \
  "$(jq -nc --arg f "$SB/notas.md" '{tool_name:"Write",tool_input:{file_path:$f,content:"#### FR-9 · sin VC"}}')"
caso 5 "Bash que no es un commit" 0 "$(bash_ev 'git status')"

# La spec queda rota en disco, sin agregar al staging.
printf '\n#### FR-3 · Invertir el match con -v\nDado … Cuando … Entonces …\n' >> sdd/demo-spec.md
caso 6 "git add y git commit en un solo comando (el escape que encontró la sesión s3)" 2 \
  "$(bash_ev 'git add sdd/demo-spec.md && git commit -m "feat: -v"')"
caso 7 "git commit -am dentro de bash -c" 2 "$(bash_ev 'bash -c "git commit -am feat"')"

git add sdd/demo-spec.md
caso 8 "git commit con la spec rota en staging" 2 "$(bash_ev 'git commit -m "feat: -v"')"

printf '> **VC-3** — sale con código 0.\n' >> sdd/demo-spec.md
caso 9 "git commit -am: staging roto, disco corregido (entra lo de disco)" 0 "$(bash_ev 'git commit -am "feat: -v"')"
caso 10 "git commit sin -a: staging sigue roto" 2 "$(bash_ev 'git commit -m "feat: -v"')"

git add sdd/demo-spec.md
caso 11 "git commit con la spec corregida en staging" 0 "$(bash_ev 'git commit -m "feat: -v"')"

caso 12 "Write de un *-spec.md bajo .claude/ (el subagent review-spec.md no es una spec)" 0 \
  "$(jq -nc --arg f "$SB/.claude/agents/review-spec.md" '{tool_name:"Write",tool_input:{file_path:$f,content:"> **VC-5** — ejemplo"}}')"

printf '#### FR-1 · a\n' > sdd/otra-spec.md
caso 13 "spec nueva sin trackear, rota, con git add -A && git commit" 2 "$(bash_ev 'git add -A && git commit -m otra')"
