#!/usr/bin/env bash
# vc-coverage.sh — cada requisito tiene su VC, y cada VC corresponde a un requisito.
#
# Lo determinístico va en un script: aparear 90 IDs "a ojo" es justo el paso que el
# modelo hace mal de vez en cuando. Lo usan el skill write-spec (para chequearse antes
# de entregar) y el hook spec-vc-gate.sh (para garantizarlo).
#
# Convención (la de 02-brownfield-tmux/sdd/ssh-pane-spec.md):
#   - Un requisito se declara como encabezado:  "#### FR-12 · …", "### BR-3a · …",
#     "### INV-1 · …", "### NFR-2 · …".
#   - Su VC se declara en negrita al comienzo de una línea (después de ">", "-" o
#     espacios), con el mismo ID: "> **VC-12** — …", "**VC-BR-3a**", "**VC-INV-1**",
#     "**VC-NFR-2**". Un VC de FR no lleva el prefijo FR-.
#   Las menciones sueltas en el texto ("ver FR-12", "reemplaza a **VC-3**") no cuentan:
#   solo las declaraciones. Lo que está dentro de un bloque de código (```) tampoco.
#   Una spec sin ningún requisito declarado así no pasa: no hay nada que garantizar.
#
# Uso:  vc-coverage.sh <spec.md>
#       vc-coverage.sh - <nombre>     (lee la spec por stdin; el nombre es para el reporte)
# Sale 0 si la cobertura es completa, 1 si no (el reporte va a stderr).
set -uo pipefail

if [ "${1:-}" = "-" ]; then
  NAME="${2:-stdin}"; SPEC="$(cat)"
else
  NAME="${1:?uso: vc-coverage.sh <spec.md> | vc-coverage.sh - <nombre>}"; SPEC="$(cat "$1")" || exit 1
fi

# Fuera los bloques de código: un "#### FR-2" de ejemplo no es un requisito.
SPEC="$(printf '%s\n' "$SPEC" | awk '/^[[:space:]]*(```|~~~)/ { f = !f; next } !f')"

# FR-12 / BR-3a / INV-1 / NFR-2, tal como aparecen en los encabezados.
REQS="$(printf '%s\n' "$SPEC" \
  | grep -E '^#{2,5}[[:space:]]+(FR|BR|INV|NFR)-[0-9]+[A-Za-z]?([^0-9A-Za-z]|$)' \
  | sed -E 's/^#{2,5}[[:space:]]+((FR|BR|INV|NFR)-[0-9]+[A-Za-z]?).*/\1/' | sort)"

if [ -z "$REQS" ]; then
  {
    echo "vc-coverage: $NAME — no declara ningún requisito."
    echo "  - Cada requisito va como encabezado: '#### FR-1 · …', '### BR-1 · …', '### INV-1 · …', '### NFR-1 · …'."
    echo "    En lista, en tabla o con el ID en negrita no se reconoce."
  } >&2
  exit 1
fi

# "> **VC-12** — …" → FR-12 · "> **VC-BR-3a** — …" → BR-3a: el ID del requisito que
# dice verificar. Solo donde se declara: al comienzo de la línea.
VCS="$(printf '%s\n' "$SPEC" \
  | sed -nE 's/^[[:space:]>*-]*\*\*VC-(((BR|INV|NFR)-)?[0-9]+[A-Za-z]?)\*\*.*/\1/p' \
  | sed -E 's/^([0-9])/FR-\1/' | sort)"

MISSING="$(comm -23 <(printf '%s\n' "$REQS" | sort -u) <(printf '%s\n' "$VCS" | sort -u) | grep . || true)"
ORPHAN="$(comm -13 <(printf '%s\n' "$REQS" | sort -u) <(printf '%s\n' "$VCS" | sort -u) | grep . || true)"
DUP_REQ="$(printf '%s\n' "$REQS" | grep . | uniq -d || true)"
DUP_VC="$(printf '%s\n' "$VCS" | grep . | uniq -d || true)"

if [ -z "$MISSING$ORPHAN$DUP_REQ$DUP_VC" ]; then
  echo "vc-coverage: $NAME — OK, $(printf '%s\n' "$REQS" | grep -c .) requisitos con su VC." >&2
  exit 0
fi

vc_of() { case "$1" in FR-*) echo "VC-${1#FR-}" ;; *) echo "VC-$1" ;; esac; }
{
  echo "vc-coverage: $NAME — la cobertura de VCs está rota."
  for id in $MISSING; do echo "  - $id no tiene VC: falta un '**$(vc_of "$id")**' que lo verifique."; done
  for id in $ORPHAN;  do echo "  - $(vc_of "$id") no verifica ningún requisito: no hay un encabezado '$id'."; done
  for id in $DUP_REQ; do echo "  - $id está declarado más de una vez."; done
  for id in $DUP_VC;  do echo "  - $(vc_of "$id") está declarado más de una vez."; done
} >&2
exit 1
