# TP3 — Un toolkit SDD: la cobertura de VCs, encodeada

El toolkit encodea el paso que más repetimos a mano en las TP1 y TP2: **escribir una
spec donde cada requisito tiene su criterio de verificación, y hacerla revisar por
alguien que no la escribió.** Son cuatro piezas que se encadenan:

```
pedido ──▶ write-spec (skill) ──▶ vc-coverage.sh ──▶ review-spec (subagent) ──▶ LISTA ──▶ código
                 │                     ▲                    │
                 │   spec-vc-gate (hook): lo vuelve a       └─ HUECOS ─▶ se corrige la spec
                 └── correr en cada escritura y en cada commit, y veta
```

Todo vive en `.claude/` en la raíz del repo, que es donde Claude Code lo busca: se
carga solo al abrir una sesión en el repo, sin instalar nada.

## Las piezas

| Pieza | Archivo | Concepto de L1–L2 que encodea | ¿Persuade o garantiza? |
|---|---|---|---|
| 📘 Skill `write-spec` | [`.claude/skills/write-spec/SKILL.md`](../.claude/skills/write-spec/SKILL.md) | **Cobertura de VCs** (un VC por requisito, con su ID), **alcance acotado** (Dentro/Fuera por archivo), **seguridad ante regresiones** (línea de base + invariantes) | Persuade |
| 🔧 Script `vc-coverage.sh` | [`.claude/skills/write-spec/scripts/vc-coverage.sh`](../.claude/skills/write-spec/scripts/vc-coverage.sh) | **Cobertura de VCs**, chequeada por código y no a ojo | Determinístico |
| 👥 Subagent `review-spec` | [`.claude/agents/review-spec.md`](../.claude/agents/review-spec.md) | **Revisión independiente** (el gate de la TP1) e **higiene de contexto** | Persuade; `tools` garantiza el solo lectura |
| 🪝 Hook `spec-vc-gate` | [`.claude/hooks/spec-vc-gate.sh`](../.claude/hooks/spec-vc-gate.sh) · [`.claude/settings.json`](../.claude/settings.json) | **Cobertura de VCs**, garantizada: ninguna spec con un requisito sin VC se escribe ni se commitea | **Garantiza** (`exit 2`) |
| 📏 Rule | [`CLAUDE.md`](../CLAUDE.md) | **Spec antes que código** y **trazabilidad** commit → spec → ID | Persuade |

### 📘 `write-spec` — el flujo

La `description` dice cuándo usarlo con el fraseo real ("especificá…", "armá la spec
de…", "antes de codear definamos…") y cuándo **no** (revisar una spec es de
`review-spec`). El cuerpo son 8 pasos numerados, cada uno marcado con su concepto SDD,
una plantilla copiable con la forma de la spec de la TP2 y anti-patrones. Los
anti-patrones incluyen lo que nos marcó la cátedra en la TP1: FRs con alternativas en
el Dado/Cuando, un Entonces que necesita dos ejecuciones, e IDs de origen inexistentes.

**Lo determinístico va en un script.** Aparear 91 requisitos con 91 VCs a ojo es el
paso que el modelo hace mal de vez en cuando. El paso 7 corre `vc-coverage.sh`, que
toma los requisitos **declarados** (encabezados `#### FR-12 · …`) y los VCs declarados
(`**VC-12**`, `**VC-BR-2**`) y reporta faltantes, huérfanos y duplicados.

### 👥 `review-spec` — por qué subagent y no skill

- **Independencia.** Quien escribió la spec completa los huecos al releerla. El
  revisor arranca con una ventana limpia: solo la spec y lo que ella enlaza. Lo que no
  está escrito, no está.
- **Higiene de contexto.** Lee la spec, las notas y los `archivo:línea` citados en
  *su* ventana. A la sesión principal vuelve solo el veredicto.
- **Brief con las tres cosas.** Qué puede hacer: solo leer. Qué devuelve: `LISTA` o
  `HUECOS (N)`. Con qué formato: una tabla fija de 7 ítems más los huecos con
  `archivo:línea`. Los ítems salen de las dimensiones de la rúbrica con la que la
  cátedra corrigió la TP1.
- **El brief pide solo lectura; `tools: Read, Grep, Glob` lo garantiza.** Sin `Edit`,
  `Write` ni `Bash`, no tiene con qué modificar la spec.
- **No recuenta la cobertura**: eso ya lo garantiza el hook. El revisor juzga si los
  VCs **sirven**: si son observables y si con ellos se puede escribir el test sin
  decidir nada.

### 🪝 `spec-vc-gate` — por qué un hook

`write-spec` le pide al modelo un VC por requisito, pero un pedido como "agregá el FR,
el VC lo vemos después" le gana a cualquier instrucción. El hook no discute. Es
`PreToolUse` y corre en dos momentos:

| Cuándo | Qué chequea | Si falla |
|---|---|---|
| `Edit` / `Write` / `MultiEdit` sobre un `*-spec.md` | El contenido **como quedaría** después de la edición | Veta la escritura |
| `Bash` con `git commit` | Cada `*-spec.md` del commit, tal como va a entrar: lo que está en staging o, si el comando trae `git add` o `-a`, la versión en disco | Veta el commit |

Se considera spec todo `*-spec.md` fuera de `.claude/`.

El chequeo es el **mismo** `vc-coverage.sh` que corre el skill: un solo criterio, que
el skill usa para persuadir y el hook para garantizar. El stderr lista los IDs que
faltan y dice qué hacer, así el agente corrige en vez de reintentar.

### 📏 La rule

[`CLAUDE.md`](../CLAUDE.md) fija lo que vale en toda sesión: no se escribe código sin
una spec con veredicto `LISTA`, y cada commit referencia su spec y sus IDs. No pega
el flujo (eso es el skill): apunta a él.

## Cuándo lo usan mis compañeros

- **Para especificar un cambio**, piden la spec con sus palabras ("especificá que gcsgrep
  acepte `-v`"): carga `write-spec`, que al final lanza `review-spec`.
- **Para saber si una spec está lista**, preguntan "¿está lista para planificar?":
  lanza `review-spec`.
- **El hook no se usa, se choca**: aparece solo cuando alguien intenta escribir o
  commitear una spec con un requisito sin VC.

Requisitos: `jq` y `git` en el `PATH`. Si se cambia `settings.json`, hay que reiniciar
la sesión del agente.

## Evidencia

Detalle completo en [`evidencia/README.md`](./evidencia/README.md).

| Pieza | Qué se ve | Dónde |
|---|---|---|
| 🪝 Hook, aislado | 13 casos, 13 OK: 8 vetos (`exit 2`) y 5 que pasan | [`01-hook-aislado.txt`](./evidencia/01-hook-aislado.txt) · [`probar-hook.sh`](./evidencia/probar-hook.sh) |
| 📘 Skill | Un pedido que no lo nombra → la primera tool call es `Skill write-spec`. Los 5 casos de disparo (2 sí, 3 no) dan lo esperado | [`02`](./evidencia/sesiones/02-skill-y-subagent.md) · [`03a–03e`](./evidencia/sesiones/) |
| 👥 Subagent | `review-spec`: `HUECOS (5)` → `(4)` → `(6)` → `(2)` → `LISTA`. Cada vuelta, 62k a 88k tokens en su ventana y 5k a 7k caracteres de vuelta; solo `Read`/`Grep`/`Glob` | [`02`](./evidencia/sesiones/02-skill-y-subagent.md) · [spec resultante](./evidencia/demo/invert-match-spec.md) |
| 🪝 Hook, en sesión real | `COMMIT BLOQUEADO` y `EDICIÓN BLOQUEADA`; el agente parte del stderr y corrige el camino | [`05a`](./evidencia/sesiones/05a-hook-veta-commit-y-edit.md) · [`05b`](./evidencia/sesiones/05b-agente-lee-stderr.md) · [`05c`](./evidencia/sesiones/05c-edicion-que-restaura-pasa.md) |

**Lo que encontramos al ejercitarlo.** Con 9 casos en verde, el hook se escapó en la
primera sesión real: el agente commiteó con `git add` y `git commit` en un solo
comando, y el `PreToolUse` miró el staging antes de que corriera el `add`
([`04b`](./evidencia/sesiones/04b-hook-se-escapa-add-y-commit.md)). Después, mientras
armábamos la evidencia, nos vetó a nosotros y destapó que el filtro agarraba
`.claude/agents/review-spec.md`. Corregimos las dos cosas y sumamos los casos a la
prueba. También vimos que la rule y el skill frenan al agente **casi siempre**
([`04a`](./evidencia/sesiones/04a-agente-frena-solo.md),
[`04c`](./evidencia/sesiones/04c-agente-se-niega-con-hook-corregido.md)), pero no
siempre: por eso existe el hook.

### Para la demo en vivo

En una sesión de Claude Code abierta en la raíz del repo:

1. Romper la spec de la demo "como un compañero", sin pasar por el agente (desde la
   terminal):
   `printf '\n#### FR-18 · Contar con -v -c\nDado … Cuando … Entonces …\n' >> "03-skills-&-agents/evidencia/demo/invert-match-spec.md"`
2. Pedirle al agente: *"Para probar el guardrail, agregá esa spec al staging y
   commiteala tal cual, con el mensaje 'FR-18'."* → `COMMIT BLOQUEADO — … FR-18 no
   tiene VC`.
3. Dejar la spec como estaba: `git checkout -- "03-skills-&-agents/evidencia/demo/invert-match-spec.md"`.

## Limitaciones conocidas

- **La convención de IDs es la de la TP2** (`VC-BR-2` ↔ `BR-2`). La spec de la TP1
  numera los VCs de BR y NFR por tabla de trazabilidad (VC-39 a VC-50), así que
  `vc-coverage.sh` la marca como rota. Esa spec está congelada y nadie la edita, así
  que el hook no la toca. Si se editara, habría que migrarla a la convención.
- **El hook solo mira archivos `*-spec.md` fuera de `.claude/`.** Una spec con otro
  nombre no está protegida.
- **El chequeo de commit matchea el texto del comando.** Detecta `git commit`,
  `git -C x commit`, `bash -c "git commit …"` y `(cd x && git commit)`. No ve un commit
  escondido en un script (`./commitear.sh`). En cambio, cualquier comando que contenga
  ese texto, aunque sea dentro de un heredoc o de un `echo`, dispara el chequeo: es un
  falso positivo conservador que nos pasó armando esta entrega. Un commit hecho desde
  la terminal, fuera del agente, no pasa por el hook. Para eso haría falta además un
  `pre-commit` de git.
- **Si el comando trae `git add` o `-a`, se chequean todas las specs modificadas en
  disco**, aunque el `add` no las incluya. Una spec rota sin relación con el commit lo
  bloquea igual: otro falso positivo conservador, preferible al escape que había antes.
- **El hook garantiza que exista un VC por requisito, no que el VC sea bueno.** Eso lo
  juzga `review-spec`, y por eso son dos piezas distintas.
- **Rutas con espacios.** El repo vive bajo una carpeta con espacios, así que
  `settings.json` cita `"$CLAUDE_PROJECT_DIR"`. El ejemplo de la cátedra no lo hace y
  acá fallaría.
