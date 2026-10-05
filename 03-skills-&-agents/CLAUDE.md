# CLAUDE.md

## Proyecto

Toolkit SDD de la TP3 (Spec-Driven Development, 73.31). La sesión se abre en esta
carpeta (`03-skills-&-agents/`); las otras entregas están al lado:
`../01-greenfield-gcsgrep` (Go) y `../02-brownfield-tmux` (spec sobre tmux).

- gcsgrep — unit tests: `cd ../01-greenfield-gcsgrep && go test ./internal/...`. El e2e
  (`./e2e`) necesita un bucket real: `GCSGREP_BUCKET`, `GCSGREP_LECTORA`, `GCSGREP_SIN_ACCESO`.

## SDD

- Una spec es un `<feature>-spec.md`. Se escribe con el skill `write-spec` y se revisa
  con el subagent `review-spec`. La de la demo está en `evidencia/demo/`.
- **No escribas código de un cambio hasta que su spec tenga veredicto `LISTA` de
  `review-spec`.** Con `HUECOS`, se corrige la spec y se vuelve a revisar.
- Cada FR, BR, INV y NFR tiene un VC con su mismo ID. El hook `spec-vc-gate` lo hace
  cumplir en cada escritura de un `*-spec.md` y en cada commit.
- Cada commit referencia la spec y los IDs que toca:
  `refs: <feature>-spec FR-3, VC-3`.
- No edites las entregas ya hechas (`../01-*`, `../02-*`) salvo que te lo pidan.
