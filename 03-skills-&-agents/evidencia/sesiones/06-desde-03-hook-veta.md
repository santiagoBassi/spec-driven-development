_Registrados en la sesión (evento init): skill `write-spec` y subagent `review-spec`, con la sesión abierta en `03-skills-&-agents/`._

## Sesión nueva · cwd=<sandbox>/03-skills-&-agents · modelo=claude-opus-4-8

**Agente:** Dale, lo corro tal cual.

→ **Bash** `cd "<sandbox>/03-skills-&-agents" && git add evidencia/demo/invert-match-spec.md && git commit -m 'FR-18, -v con -c (refs: invert-match-spec FR …[recortado]`

  ⛔ **resultado (error):**
```
PreToolUse:Bash hook error: ["$CLAUDE_PROJECT_DIR"/.claude/hooks/spec-vc-gate.sh]: COMMIT BLOQUEADO — una spec del commit tiene requisitos sin VC.
vc-coverage: 03-skills-&-agents/evidencia/demo/invert-match-spec.md — la cobertura de VCs está rota.
  - FR-18 no tiene VC: falta un '**VC-18**' que lo verifique.

Regla de la spec: cada FR, BR, INV y NFR tiene un VC con su mismo ID, y cada VC
corresponde a un requisito declarado (VC-12 ↔ FR-12, VC-BR-2 ↔ BR-2).
Corregí la spec (skill write-spec), hacé git add y volvé a commitear.

```

**Agente:** El guardrail saltó. Te resumo qué pasó.

## Qué pasó

El commit **quedó bloqueado**. No lo bloqueó git, sino el hook `spec-vc-gate` del toolkit, que corre como `PreToolUse` antes de que se ejecute el comando. Como es un hook *pre*, frenó toda la línea de una (ni el `git add` ni el `git commit` llegaron a correr).

El motivo que reportó:

- **Spec:** `evidencia/demo/invert-match-spec.md`
- **Falla:** cobertura de VCs rota — `FR-18` no tiene un `VC-18` que lo verifique.
- **Regla violada:** cada FR/BR/INV/NFR tiene que tener un VC con su mismo ID (y a la inversa, cada VC apunta a un requisito declarado).

Es decir: el commit declaraba tocar `FR-18` (`refs: invert-match-spec FR-18`), pero la spec tiene el `FR-18` sin su `VC-18` correspondiente. El gate lo detectó y abortó.

## Confirmación de que nada quedó a medias

→ **Bash** `cd "<sandbox>/03-skills-&-agents" && git status --short && echo "---" && git log -1 --oneline`

  ← resultado:  M evidencia/demo/invert-match-spec.md ⏎ --- ⏎ e28dd14 toolkit movido a 03 (sandbox)

**Agente:** Confirmado, todo limpio:

- **Último commit:** sigue siendo `e28dd14 toolkit movido a 03 (sandbox)` — no se creó ningún commit nuevo.
- **Working tree:** `invert-match-spec.md` aparece como ` M` (modificado, **sin stagear**). O sea el `git add` tampoco llegó a ejecutarse — el hook `PreToolUse` frenó el comando entero antes de que arrancara.

## Veredicto

El guardrail funcionó exactamente como debía: detectó que el commit referenciaba `FR-18` sin su `VC-18` en la spec y abortó sin dejar nada a medias. 

Si querés que la demo siga, el camino que marca el propio toolkit es: corregir la spec con `write-spec` (agregar el `VC-18` que verifique `FR-18`), pasar `review-spec` para el veredicto `LISTA`, y recién ahí `git add` + commit. Decime si lo encaramos.


---
Fin: success · turnos=3 · costo USD=0.2733645 · duración=27s
