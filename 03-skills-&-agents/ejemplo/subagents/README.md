# Subagents

Dos subagents en el formato de Claude Code: un archivo markdown con front-matter
(`name`, `description`, `tools`) y un cuerpo, que es el brief. Para usarlos,
copialos a `.claude/agents/` de tu proyecto.

| Subagent | Qué hace | Por qué es un subagent y no un skill |
|---|---|---|
| [`adr-explorer`](./adr-explorer.md) | Busca decisiones previas que la propuesta confirma o contradice | **Firewall de contexto.** Lee `docs/adr/`, el código y la historia en *su* ventana y devuelve una lista corta. La sesión que escribe el ADR recibe la conclusión, no las lecturas. |
| [`adr-reviewer`](./adr-reviewer.md) | Decide si un ADR propuesto se acepta | **Independencia.** Quien escribió el ADR completa los huecos al releerlo. Un revisor con contexto sembrado solo con el documento no puede: lo que no está escrito, no está. |

## Las tres cosas de un brief

Abrí cualquiera de los dos y buscá:

1. **Qué puede hacer.** "Solo leer. No modifiques ningún archivo." — en la primera
   línea, no enterrado al final.
2. **Qué devuelve.** Una lista, un veredicto: algo que se puede usar sin
   reprocesarlo.
3. **Con qué formato.** Fijo. Dos corridas sobre el mismo ADR tienen que devolver la
   misma forma.

Un subagent sin esas tres cosas es un prompt largo.

## `tools`: lo que el brief no puede garantizar

El brief dice "no modifiques ningún archivo", pero eso es una instrucción: el modelo
la puede ignorar. El front-matter `tools: Read, Grep, Glob` es una allowlist: sin
`Edit`, `Write` ni `Bash`, el subagent **no tiene con qué** modificar nada.

El brief persuade; `tools` garantiza.

El trade-off: `adr-explorer` querría leer `git log` para buscar decisiones en los
commits, y para eso necesita `Bash` — que también puede escribir. Acá se eligió no
dárselo: el brief dice "si podés leerlos desde el repo", y sin `Bash` no puede.
Darle `Bash` es una decisión que, en un proyecto real, merecería su propio ADR.

## En otros agentes

El concepto es el mismo; cambia el formato. Codex define cada agente en un TOML bajo
`.codex/agents/` y restringe con `sandbox_mode = "read-only"`. Grok Build los busca
en `.grok/agents/`.
