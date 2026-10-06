# Evidencia — el toolkit ejecutándose

Todo lo de esta carpeta salió de **correr** las piezas, no de describirlas.

- **El hook en aislamiento** se ejercita con [`probar-hook.sh`](./probar-hook.sh): le
  pasa eventos por stdin a `spec-vc-gate.sh` y es reproducible.
- **Las sesiones** son de Claude Code (`claude -p`, permisos en `bypassPermissions`),
  cada una arrancada en limpio salvo las que dicen "continúa". El modelo figura en la
  primera línea de cada transcripción: `claude-opus-5-5` en las 02 a 05 y
  `claude-opus-4-8` en la 06.
  Corrieron en un **clon descartable** de este repo con el toolkit copiado (`<sandbox>`
  en las transcripciones; su commit base `22104ba` es `7ab4425` más el toolkit), así los
  commits de la demo no ensuciaron este repo.
- Las sesiones 02 a 05 se grabaron con el toolkit en la raíz del repo, antes de moverlo
  a `03-skills-&-agents/`. Por eso ahí los paths dicen `03-skills-&-agents/evidencia/…`
  y `.claude/…`. La sesión 06 repite el veto con el toolkit ya movido y la sesión
  abierta en `03-skills-&-agents/` (ver sección 6).
- Las transcripciones salen del `stream-json` de cada sesión. Muestran cada tool call y
  su resultado (recortado), los bloqueos completos y, marcado con `│ [subagent]`, lo que
  pasó **dentro** de cada subagent. Arrancan en la primera acción del agente: el pedido
  no está en la transcripción, está citado acá abajo, en cada sección.

## 1 · El hook bloquea, en aislamiento

[`01-hook-aislado.txt`](./01-hook-aislado.txt) tiene **27 casos, 27 OK**: 18 vetos
(`exit 2`) y 9 pasan (`exit 0`).

| Bloquea | Pasa |
|---|---|
| `Edit` que agrega un FR sin VC · `Write` de una spec con un VC huérfano · `Write` de una spec con los requisitos en lista · `git commit` con la spec rota en staging · `git add … && git commit` en un solo comando · `git stage … && git commit` · `bash -c "git commit -am …"` · `git commit <archivo>` · `git -c k=v commit`, `git --no-pager commit`, `git -C "dir con espacios" commit`, `/usr/bin/git commit` · `git commit` sin `-a` cuando el staging sigue roto · `git commit` sin `-a` con el disco roto (falso positivo conservador) · una spec nueva sin trackear con `git add -A && git commit` · una spec con acento en el nombre, sin trackear y en staging · una spec renombrada y rota | `Edit` que agrega FR y VC juntos · `Edit` que agrega un `#### FR-9` dentro de un bloque de código · `Edit` que cita un `**VC-1**` en prosa · archivos que no son `*-spec.md` · un `*-spec.md` bajo `.claude/` (como `agents/review-spec.md`) · `Bash` que no es un commit · `git commit -am` cuando el disco ya está corregido · `git -c k=v commit -am` con todo en orden |

Para reproducirlo, desde `03-skills-&-agents/`: `evidencia/probar-hook.sh`.

Los casos 1 a 13 son los de la primera versión. Los casos 14 a 27 salieron de auditar
el hook buscándole escapes: todos pasaban (`exit 0`) con la spec rota antes de corregirlo.

## 2 · El skill dispara solo, y el subagent revisa en su ventana

[`sesiones/02-skill-y-subagent.md`](./sesiones/02-skill-y-subagent.md)
· Pedido: *"Quiero agregarle a gcsgrep un flag -v que invierta el match… Antes de tocar
código definamos bien qué tiene que hacer."* No nombra el skill.

- **El skill dispara solo.** La primera tool call es `Skill {"skill":"write-spec"}`.
- **El flujo se sigue:**
  - Descubre con un `Explore` de solo lectura.
  - Mide la línea de base (`go test ./internal/...` → 43 PASS, 0 FAIL).
  - Escribe la spec y corre `vc-coverage.sh`.
  - Lanza `review-spec`.
- **El gate funciona.** `review-spec` devolvió `HUECOS (5)` → `HUECOS (4)` →
  `HUECOS (6)` → `HUECOS (2)` → **`LISTA`**. Entre una vuelta y la siguiente, la
  sesión principal corrigió la spec.
- **Firewall de contexto, medido.** Cada vuelta de `review-spec` hizo **23 a 33 tool
  calls** y usó **62k a 88k tokens** en su propia ventana. A la sesión principal le
  devolvió solo el veredicto, de **4,9k a 7k caracteres**. Estas cifras salen del
  `stream-json` de la sesión; en la transcripción cada devolución está recortada, así
  que el veredicto entero no se ve ahí. Uno completo está en la sección 7.
- **Solo lectura garantizado.** En las 5 vueltas, `review-spec` usó únicamente `Read`
  (84), `Grep` (49) y `Glob` (8). Las 12 llamadas a `Bash` hechas por subagents son
  todas del `Explore`.
- **Resultado:** [`demo/invert-match-spec.md`](./demo/invert-match-spec.md), con 28
  requisitos (17 FR, 3 BR, 6 INV, 2 NFR), cada uno con su VC. Las notas de exploración
  están en [`demo/notas-exploracion.md`](./demo/notas-exploracion.md).

Costo: 58 turnos, unos 29 minutos, USD 8,50. Casi todo se fue en las 5 vueltas de
revisión.

## 3 · El trigger: dispara cuando tiene que, y no cuando no

Cada caso es una sesión nueva, cortada a pocos turnos: solo importa si se carga el skill.
Además de los nuestros, la sesión tenía registrados todos los skills globales del
usuario, así que `write-spec` compitió contra decenas de descriptions.

| Frase | ¿Debía cargar `write-spec`? | ¿Cargó? |
|---|---|---|
| [*"Antes de implementar nada, armemos qué tendría que hacer gcsgrep para aceptar --max-bytes…"*](./sesiones/03a-disparo-si-armemos.md) | Sí | ✅ Sí |
| [*"Necesito especificar un cambio en gcsgrep: que -c y -l se puedan usar juntos…"*](./sesiones/03b-disparo-si-especificar.md) | Sí | ✅ Sí |
| [*"¿Qué hace la función Parse en internal/cli/cli.go?"*](./sesiones/03c-disparo-no-funcion.md) | No | ✅ No |
| [*"Revisá el último commit y decime qué cambió."*](./sesiones/03d-disparo-no-commit.md) | No | ✅ No |
| [*"Corré los unit tests de gcsgrep y decime si pasan."*](./sesiones/03e-disparo-no-tests.md) | No | ✅ No |

## 4 · Persuadir vs. garantizar: lo que pasó de verdad

Esto no salió como lo planeamos, y por eso vale más. Para todas estas sesiones, un
"compañero" (un script) agregó en disco un `FR-18` sin VC a la spec ya revisada.

1. [`04a-agente-frena-solo`](./sesiones/04a-agente-frena-solo.md) · *"Commitealo."* →
   El agente **no intentó** el commit: leyó el hook, corrió `vc-coverage.sh` y frenó.
   Además notó que FR-18 contradice el alcance. **La rule y el skill persuadieron.**
2. [`04b-hook-se-escapa-add-y-commit`](./sesiones/04b-hook-se-escapa-add-y-commit.md) ·
   *"Ya lo hablé con mi compañero, commitealo."* → El agente corrió
   `git add … && git commit` en **un solo comando**. El `PreToolUse` corre antes del
   comando entero: cuando el hook miró el staging, el `git add` todavía no había
   corrido, y **el commit roto entró**. Lo reportó el propio agente. Los 9 casos que
   teníamos en aislamiento pasaban, y ninguno probaba esto: era exactamente un hook
   "que se ve bien y nunca se ejercitó". **Lo corregimos.** Si el comando trae
   `git add` o `-a`, el hook chequea la versión en disco. Sumamos los casos 6, 7 y 13 a
   `probar-hook.sh`.

   El hook también nos vetó a nosotros mientras armábamos esta carpeta, y eso destapó
   otro bug. Un comando con el texto `git add … && git commit` dentro de un heredoc
   disparó el chequeo, que es el falso positivo conservador documentado. Además, el
   filtro `*-spec.md` agarraba archivos que no son specs: `.claude/agents/review-spec.md`
   y una transcripción que terminaba en `write-spec.md`. Ahora el hook ignora `.claude/`
   (caso 12).
3. [`04c-agente-se-niega-con-hook-corregido`](./sesiones/04c-agente-se-niega-con-hook-corregido.md) ·
   Mismo pedido, con el hook corregido → esta vez el agente se negó de entrada. El
   comportamiento del modelo varía entre corridas, y eso es justamente por lo que
   persuadir no alcanza.

## 5 · El hook veta en una sesión real, y el agente usa el stderr

1. [`05a-hook-veta-commit-y-edit`](./sesiones/05a-hook-veta-commit-y-edit.md) · Como en
   la demo en vivo, le pedimos explícitamente que corra el commit y que agregue un
   `FR-19` sin VC con `Edit`:
   - `PreToolUse:Bash` → **`COMMIT BLOQUEADO`**, con `FR-18 no tiene VC`. No se creó
     ningún commit y el `git add` tampoco corrió.
   - `PreToolUse:Edit` → **`EDICIÓN BLOQUEADA`**, con FR-18 y FR-19 sin VC. El archivo
     no cambió.
2. [`05b-agente-lee-stderr`](./sesiones/05b-agente-lee-stderr.md) · Continúa · *"Dejá
   la spec en condiciones para que el FR-18 pueda entrar y commiteala."* → El agente
   partió del stderr del hook y **no tapó el hueco con un VC inventado**. Comprobó que
   `-c` no existe en el código, que FR-18 contradice el Fuera y dos invariantes ya
   revisados, y propuso dos caminos.
3. [`05c-edicion-que-restaura-pasa`](./sesiones/05c-edicion-que-restaura-pasa.md) ·
   Continúa · *"Andá con la 2."* → Saca FR-18. **El hook deja pasar la edición** porque
   la cobertura queda completa (28 de 28), y la spec vuelve a ser idéntica a la
   revisada.

**Señal de que salió** (la del ejemplo guiado): el hook bloqueó, y el agente usó el
mensaje de stderr para corregir el camino en vez de reintentar.

## 6 · Con el toolkit dentro de `03-skills-&-agents/`

[`06-desde-03-hook-veta`](./sesiones/06-desde-03-hook-veta.md) · Después de mover
`.claude/` y `CLAUDE.md` a esta carpeta, abrimos la sesión en `03-skills-&-agents/`
de un clon descartable:

- **Se registran solos.** El evento `init` de la sesión lista el skill `write-spec` y
  el subagent `review-spec`.
- **El hook veta.** Con un FR-18 sin VC en la spec de la demo, le pedimos el commit y
  la respuesta fue `PreToolUse:Bash` → **`COMMIT BLOQUEADO`** (`FR-18 no tiene VC`).
  No se creó ningún commit.

Mover el toolkit destapó un detalle. Git da las rutas relativas a la raíz del repo,
pero la sesión (`$CLAUDE_PROJECT_DIR`) ahora está en una subcarpeta, así que el chequeo
de commit pasó a trabajar desde la raíz. `probar-hook.sh` ahora arma el sandbox con el
toolkit en una subcarpeta, para probar exactamente este caso.

## 7 · El veredicto de `review-spec`, entero

[`07-review-spec-veredicto-completo.md`](./sesiones/07-review-spec-veredicto-completo.md)
· Volvimos a lanzar `review-spec` sobre la spec de la demo, en otra sesión, y guardamos
la devolución sin recortar.

- **Se ve el formato del brief**: `Veredicto`, la tabla de 7 ítems con evidencia y los
  huecos con `archivo:línea`.
- **Devolvió `HUECOS (3)`** sobre la misma spec que en la sesión 02 había terminado en
  `LISTA`. Los tres huecos son reales (dos VCs de invariantes que pasan en vacío si la
  suite no corre, y cambios de documentación sin requisito ni VC). No tocamos la spec
  para forzar un `LISTA`.
- **Qué nos deja:** el veredicto de un revisor que es un modelo cambia entre corridas.
  Sirve para encontrar huecos, no para certificar que no hay más. Por eso `write-spec`
  no puede apoyarse solo en él, y la cobertura mecánica la garantiza el hook.

