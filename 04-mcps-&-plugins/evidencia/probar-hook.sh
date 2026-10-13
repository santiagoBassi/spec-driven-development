#!/usr/bin/env bash
# probar-hook.sh — le pasa a gcs-allowlist.sh un evento PreToolUse por caso y muestra
# el exit code y el mensaje. No toca el bucket ni necesita credenciales: prueba el hook
# solo, con el mismo JSON que le manda Claude Code por stdin.
#
# Uso: evidencia/probar-hook.sh     (desde 04-mcps-&-plugins/)
# Termina con exit 1 si algún caso no dio lo esperado.
set -uo pipefail

cd "$(dirname "$0")/.."
HOOK=.claude/hooks/gcs-allowlist.sh
BUCKET=sdd-fardenghi-itba
FALLOS=0

# caso <descripción> <exit esperado> <GCSGREP_BUCKET> <tool_name> <tool_input JSON>
caso() {
  local desc="$1" esperado="$2" bucket="$3" tool="$4" input="$5" salida codigo
  salida=$(jq -cn --arg t "$tool" --argjson i "$input" \
             '{hook_event_name: "PreToolUse", tool_name: $t, tool_input: $i}' \
           | GCSGREP_BUCKET="$bucket" "$HOOK" 2>&1)
  codigo=$?
  local veredicto="OK"
  [[ "$codigo" == "$esperado" ]] || { veredicto="FALLÓ"; FALLOS=$((FALLOS + 1)); }
  echo "── $desc"
  echo "   tool:     $tool"
  echo "   input:    $input"
  echo "   exit:     $codigo (esperado $esperado) → $veredicto"
  [[ -n "$salida" ]] && echo "   mensaje:  $salida"
  echo
}

echo "GCSGREP_BUCKET=$BUCKET · hook: $HOOK"
echo

echo "═══ Pasan: las tres de lectura de objetos, contra el bucket permitido"
caso "listar objetos" 0 "$BUCKET" mcp__gcs__list_objects \
  "{\"bucket_name\":\"$BUCKET\",\"prefix\":\"logs/\"}"
caso "leer metadata" 0 "$BUCKET" mcp__gcs__read_object_metadata \
  "{\"bucket_name\":\"$BUCKET\",\"object_name\":\"logs/app/api.log\"}"
caso "leer contenido" 0 "$BUCKET" mcp__gcs__read_object_content \
  "{\"bucket_name\":\"$BUCKET\",\"object_name\":\"logs/app/api.log\"}"

echo "═══ Se vetan: todo lo que no está en la allowlist"
caso "escribir un objeto nuevo" 2 "$BUCKET" mcp__gcs__write_object_safe \
  "{\"bucket_name\":\"$BUCKET\",\"object_name\":\"tp4-prueba-muro/x.log\",\"content\":\"aG9sYQ==\"}"
caso "crear un bucket" 2 "$BUCKET" mcp__gcs__create_bucket \
  "{\"project_id\":\"sat-spec-driven-development\",\"bucket_name\":\"bucket-nuevo\"}"
caso "tool inventada (el server no la expone)" 2 "$BUCKET" mcp__gcs__delete_object \
  "{\"bucket_name\":\"$BUCKET\",\"object_name\":\"fixtures/hostil/notas.log\"}"

echo "═══ Se vetan: lectura fuera del bucket permitido"
caso "leer contenido de otro bucket" 2 "$BUCKET" mcp__gcs__read_object_content \
  "{\"bucket_name\":\"otro-bucket\",\"object_name\":\"logs/app/api.log\"}"
caso "listar sin GCSGREP_BUCKET definida" 2 "" mcp__gcs__list_objects \
  "{\"bucket_name\":\"$BUCKET\"}"

echo "═══ Con el nombre que toman las tools dentro del plugin"
caso "leer contenido (plugin)" 0 "$BUCKET" mcp__plugin_sdd-toolkit_gcs__read_object_content \
  "{\"bucket_name\":\"$BUCKET\",\"object_name\":\"logs/app/api.log\"}"
caso "escribir un objeto nuevo (plugin)" 2 "$BUCKET" mcp__plugin_sdd-toolkit_gcs__write_object_safe \
  "{\"bucket_name\":\"$BUCKET\",\"object_name\":\"tp4-prueba-muro/x.log\",\"content\":\"aG9sYQ==\"}"

if [[ "$FALLOS" -gt 0 ]]; then
  echo "RESULTADO: $FALLOS caso(s) no dieron lo esperado."
  exit 1
fi
echo "RESULTADO: los 10 casos dieron lo esperado."
