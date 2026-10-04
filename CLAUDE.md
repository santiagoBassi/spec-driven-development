# CLAUDE.md

## Proyecto

Entregas de Spec-Driven Development (73.31). Cada carpeta `0N-*/` es una tarea:
`01-greenfield-gcsgrep` (Go), `02-brownfield-tmux` (spec sobre tmux), `03-skills-&-agents`.

- gcsgrep — unit tests: `cd 01-greenfield-gcsgrep && go test ./internal/...`. El e2e
  (`./e2e`) necesita un bucket real: `GCSGREP_BUCKET`, `GCSGREP_LECTORA`, `GCSGREP_SIN_ACCESO`.

## SDD

- Las specs viven en `<tarea>/sdd/<feature>-spec.md`. Se escriben con el skill
  `write-spec` y se revisan con el subagent `review-spec`.
- **No escribas código de un cambio hasta que su spec tenga veredicto `LISTA` de
  `review-spec`.** Con `HUECOS`, se corrige la spec y se vuelve a revisar.
- Cada FR, BR, INV y NFR tiene un VC con su mismo ID. El hook `spec-vc-gate` lo hace
  cumplir en cada escritura de un `*-spec.md` y en cada commit.
- Cada commit referencia la spec y los IDs que toca:
  `refs: <feature>-spec FR-3, VC-3`.
- No edites las entregas ya hechas (`01-*`, `02-*`) salvo que te lo pidan.
