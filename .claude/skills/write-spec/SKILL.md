---
name: write-spec
description: Usar cuando el usuario pide especificar un cambio o un feature antes de implementarlo — "especificá…", "escribí / armá la spec de…", "antes de codear definamos qué tiene que hacer…", "quiero agregar X, hagamos la spec" — sobre código existente (brownfield) o nuevo. Produce sdd/<feature>-spec.md con alcance dentro/fuera por archivo, FRs en Dado/Cuando/Entonces, BRs, invariantes, NFRs y un VC por requisito. No usarlo para revisar una spec ya escrita (eso es el subagent review-spec) ni para planificar iteraciones.
---

# write-spec

Convierte un pedido en una **spec SDD verificable**. Lo que este skill no deja que exista:
**un requisito sin su VC.** Si una línea no se puede verificar, no está especificada.

## Pasos

1. **Descubrí sin ensuciar la ventana** *(higiene de contexto)*. En brownfield, si no
   hay notas de exploración, lanzá un subagent de solo lectura (`Explore`) que devuelva
   módulos tocados con `archivo:línea`, interfaces que se reusan y riesgos. Acá llegan
   las notas, no las 40 lecturas.
2. **Medí la línea de base** *(seguridad ante regresiones)*: build y suite existente
   **antes** del cambio, con el comando exacto y el resultado (`N PASS, M FAIL`).
3. **Acotá el alcance** *(alcance acotado)*: tabla **Dentro** por archivo, con qué
   cambia y dónde (`archivo:línea`); lista **Fuera** con lo vecino que tienta tocar.
4. **Escribí los FRs atómicos**, en Dado/Cuando/Entonces: un Dado, un Cuando y un
   resultado que se observa en **una** ejecución. Si aparece un "o", partilo en
   `FR-Na` / `FR-Nb`.
5. **Escribí BRs, invariantes y NFRs.** Invariantes = lo que no tiene que cambiar: la
   suite igual que en la línea de base, solo cambian los archivos de Dentro y el
   comportamiento existente que se toca de cerca. NFRs con número y condición.
6. **Escribí un VC por requisito, con el mismo ID** *(cobertura de VCs)*: `VC-12` ↔
   `FR-12`, `VC-BR-2` ↔ `BR-2`, `VC-INV-1` ↔ `INV-1`, `VC-NFR-1` ↔ `NFR-1`. Un comando,
   el exit code y la salida literal esperada.
7. **Chequeá la cobertura con el script**, no a ojo:
   `.claude/skills/write-spec/scripts/vc-coverage.sh sdd/<feature>-spec.md`, hasta que
   dé OK. El hook `spec-vc-gate` corre lo mismo en cada escritura y cada commit: si te
   bloquea, el stderr dice qué ID falta.
8. **Pedí la revisión independiente**: lanzá el subagent `review-spec` con el path de
   la spec. Con `HUECOS`, corregí y volvé a lanzarlo. Con `LISTA`, poné
   `Estado: revisada` en la cabecera.

## Plantilla

```md
# <feature> — spec
> Estado: borrador · Base: <repo>@<commit> · Notas: [notas-exploracion.md](…)

## Propósito          <una oración: qué y para qué>
## Alcance
### Dentro            | Archivo | Cambio | Referencia (archivo:línea) |
### Fuera             - <lo que no cambia, aunque tiente>
## Línea de base      <comando> → <N PASS, M FAIL> (medida antes del cambio)
## Requerimientos funcionales
#### FR-1 · <comportamiento, con verbo>
**Dado** … **Cuando** … **Entonces** …
> **VC-1** — `<comando>` sale con código `<n>` y stdout `<literal>`.
## Reglas de negocio
### BR-1 · <regla>
> **VC-BR-1** — …
## Invariantes
### INV-1 · La suite existente da lo mismo que en la línea de base
> **VC-INV-1** — `<comando de la suite>` → mismo conjunto de PASS y FAIL.
## Requerimientos no funcionales
### NFR-1 · <métrica con número y condición>
> **VC-NFR-1** — …
## Preguntas abiertas  <vacío para pasar a revisión>
```

## Anti-patrones

- **FR sin VC, o VC "a mano"** ("funciona bien"). El hook lo bloquea; no lo pelees, escribí el VC.
- **Alternativas en el Dado o el Cuando** ("bucket o prefijo"): son dos FRs.
- **Un Entonces que necesita dos ejecuciones** para observarse: partilo.
- **Fuera de alcance vacío.** Si no escribiste qué no cambia, el alcance no está acotado.
- **Brownfield sin invariantes ni línea de base**: no hay forma de saber si rompiste algo.
- **NFR vago** ("rápido", "robusto"): sin número no hay VC.
- **IDs de origen inventados** en la trazabilidad: si citás `FR-h`, tiene que existir.
