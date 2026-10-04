---
name: write-adr
description: Usar cuando la persona toma o acaba de tomar una decisión de arquitectura o de diseño no obvia y quiere dejarla registrada — "elegimos X sobre Y", "documentá por qué", "anotá esta decisión", "escribí un ADR". Produce un ADR numerado en docs/adr/ con contexto, opciones consideradas, decisión y consecuencias.
---

# write-adr

Convierte una decisión que hoy vive en una conversación en un **ADR** (Architecture
Decision Record): un archivo corto, numerado e inmutable que dice qué se decidió,
qué se descartó y por qué.

**La regla que este skill hace imposible violar:** no hay ADR con una sola opción. Si
no se consideró ninguna alternativa, no hubo decisión — hubo un default, y eso no se
registra como ADR.

## Cuándo NO usarlo

- La "decisión" es una convención que vale siempre ("los errores van a stderr"). Eso
  es una rule: va en `CLAUDE.md` / `AGENTS.md`.
- Cambia una decisión que ya tiene ADR aceptado. Para eso está `supersede-adr`: un
  ADR aceptado no se edita.

## Pasos

1. **Buscá decisiones previas.** Antes de escribir, lanzá el subagent `adr-explorer`
   con la decisión propuesta. Si devuelve un ADR aceptado que la contradice, frená:
   esto es un `supersede-adr`, no un ADR nuevo.

2. **Obtené el número.** Corré `scripts/next-adr-number.sh` desde la raíz del repo.
   No lo calcules a mano: dos personas contando archivos llegan a números distintos.

3. **Escribí el contexto** sin la decisión: qué problema hay, qué restricciones,
   qué fuerza la decisión *ahora*.

4. **Listá al menos dos opciones**, cada una con a favor y en contra. La elegida
   también tiene contras.

5. **Escribí la decisión** en una oración que empiece con un verbo: "Usamos…",
   "No exponemos…".

6. **Escribí las consecuencias**, incluidas las negativas y las que obligan a otros
   equipos o a código futuro.

7. **Guardalo como `Estado: Propuesto`.** Pasa a `Aceptado` cuando lo aprueba
   alguien que no lo escribió — idealmente con el subagent `adr-reviewer`.

## Plantilla

Archivo: `docs/adr/NNNN-titulo-en-kebab-case.md`

```md
# ADR-NNNN: <la decisión, en pocas palabras>

Estado: Propuesto
Fecha: AAAA-MM-DD

## Contexto
<qué problema, qué restricciones, por qué ahora>

## Opciones consideradas
1. **<opción A>** — a favor: … · en contra: …
2. **<opción B>** — a favor: … · en contra: …

## Decisión
<Usamos / No usamos …>, porque <el motivo que inclinó la balanza>.

## Consecuencias
- <lo que se vuelve más fácil>
- <lo que se vuelve más difícil, o lo que ahora no se puede hacer>
```

## Anti-patrones

- **Una sola opción.** No es un ADR, es un anuncio.
- **Consecuencias solo positivas.** Toda decisión cuesta algo; si no lo escribiste,
  no lo pensaste.
- **Contexto que ya contiene la decisión** ("Como vamos a usar Postgres…"). El
  contexto tiene que poder leerse sin saber qué se eligió.
- **Editar un ADR aceptado para "actualizarlo".** Se pierde la historia. Usá
  `supersede-adr`.
- **Numerar a mano.** Para eso está el script.
