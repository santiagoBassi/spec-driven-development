#!/usr/bin/env bash
# probar-hook.sh — ejercita spec-vc-gate.sh sin el agente, pasándole el evento por stdin
# (es lo que hace Claude Code). Arma un repo descartable en un directorio temporal,
# así que no toca este repo. Salida esperada: la de 01-hook-aislado.txt.
#
# Como en este repo, el toolkit vive en una SUBCARPETA del repo (la sesión se abre ahí
# y CLAUDE_PROJECT_DIR apunta a ella): <repo>/toolkit/.claude, <repo>/toolkit/sdd.
#
# Uso, desde 03-skills-&-agents/:  evidencia/probar-hook.sh
set -uo pipefail

TOOLKIT="$(cd "$(dirname "$0")/.." && pwd)"
REPO_SB="$(mktemp -d)"; trap 'rm -rf "$REPO_SB"' EXIT
SB="$REPO_SB/toolkit"
mkdir -p "$SB/sdd" && cp -R "$TOOLKIT/.claude" "$SB/" && git -C "$REPO_SB" init -q && cd "$SB"

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

# --- Casos 14 a 27: los escapes que encontró la auditoría del toolkit. ---
rm sdd/otra-spec.md
git -c user.name=t -c user.email=t@t commit -qm "feat: -v"   # staging y disco limpios

caso 14 "Write de una spec con los requisitos en lista (ninguno declarado como encabezado)" 2 \
  "$(jq -nc --arg f "$SB/sdd/lista-spec.md" '{tool_name:"Write",tool_input:{file_path:$f,content:"- **FR-1**: buscar\n- **FR-2**: informar\n"}}')"
caso 15 "Edit que agrega un bloque de código con un '#### FR-9' de ejemplo" 0 \
  "$(edit '### BR-1' $'```md\n#### FR-9 · ejemplo de formato\n```\n\n### BR-1')"
caso 16 "Edit que cita **VC-1** en prosa (una mención no es una declaración)" 0 \
  "$(edit '### BR-1' $'Nota: **VC-1** y **VC-2** comparten fixture.\n\n### BR-1')"

# La spec queda rota en disco, sin agregar al staging.
printf '\n#### FR-4 · Contar con -c\nDado … Cuando … Entonces …\n' >> sdd/demo-spec.md
caso 17 "git commit con pathspec (entra la versión en disco)" 2 "$(bash_ev 'git commit -m "feat: -c" sdd/demo-spec.md')"
caso 18 "git -c k=v commit -am" 2 "$(bash_ev 'git -c user.name=x commit -am "feat: -c"')"
caso 19 "git --no-pager commit -am" 2 "$(bash_ev 'git --no-pager commit -am "feat: -c"')"
caso 20 "git -C con un directorio con espacios, entre comillas" 2 "$(bash_ev 'git -C "/tmp/un dir con espacios" commit -am "feat: -c"')"
caso 21 "/usr/bin/git commit -am" 2 "$(bash_ev '/usr/bin/git commit -am "feat: -c"')"
caso 22 "git stage y git commit en un solo comando" 2 "$(bash_ev 'git stage sdd/demo-spec.md && git commit -m "feat: -c"')"
caso 23 "git commit sin -a: disco roto, staging sano (falso positivo conservador)" 2 "$(bash_ev 'git commit -m "otra cosa"')"

git checkout -- sdd/demo-spec.md
caso 24 "git -c k=v commit -am con todo en orden" 0 "$(bash_ev 'git -c user.name=x commit -am "feat: -c"')"

printf '#### FR-1 · a\n' > sdd/autenticación-spec.md
caso 25 "spec nueva con acento en el nombre, rota, con git add -A && git commit" 2 "$(bash_ev 'git add -A && git commit -m auth')"
git add -A
caso 26 "la misma spec, ya en staging" 2 "$(bash_ev 'git commit -m auth')"
git rm -q --cached sdd/autenticación-spec.md && rm sdd/autenticación-spec.md

git mv sdd/demo-spec.md sdd/renombrada-spec.md
printf '\n#### FR-4 · Contar con -c\nDado … Cuando … Entonces …\n' >> sdd/renombrada-spec.md
git add -A
caso 27 "spec renombrada y rota, en staging (git la ve como rename)" 2 "$(bash_ev 'git commit -m rename')"
