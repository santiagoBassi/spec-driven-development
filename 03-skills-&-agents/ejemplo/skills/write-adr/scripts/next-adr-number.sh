#!/usr/bin/env bash
# next-adr-number.sh — imprime el próximo número de ADR, con cuatro dígitos.
#
# Lo determinístico va en un script, no en un párrafo del skill: contar archivos
# "a ojo" es exactamente el tipo de paso que el modelo hace mal de vez en cuando.
#
# Uso: scripts/next-adr-number.sh [directorio]   (por defecto: docs/adr)
set -euo pipefail

ADR_DIR="${1:-docs/adr}"

last=0
if [ -d "$ADR_DIR" ]; then
  for f in "$ADR_DIR"/[0-9][0-9][0-9][0-9]-*.md; do
    [ -e "$f" ] || continue
    n="$(basename "$f" | cut -c1-4)"
    n=$((10#$n))
    [ "$n" -gt "$last" ] && last="$n"
  done
fi

printf '%04d\n' $((last + 1))
