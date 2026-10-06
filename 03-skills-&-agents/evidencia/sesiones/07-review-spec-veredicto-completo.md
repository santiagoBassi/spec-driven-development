_Devolución completa de `review-spec`, sin recortar. En las transcripciones 02 a 06 el
resultado de cada subagent aparece cortado (`…[recortado]`); acá está entero: el
veredicto, la tabla de 7 ítems y los huecos. Lo único que se cambió es la ruta absoluta
del repo por `<repo>`._

## Sesión del 2026-10-05 · cwd=<repo>/03-skills-&-agents · modelo de la sesión=claude-opus-5-5

Sesión interactiva de Claude Code abierta en `03-skills-&-agents/`, con el toolkit ya
corregido después de la auditoría. La spec es la de la demo tal como quedó commiteada:
`Estado: revisada (… LISTA en la 5.ª pasada)`.

→ **Agent** subagent_type=review-spec · prompt: Revisá la spec <repo>/03-skills-&-agents/evidencia/demo/invert-match-spec.md (brownfield: flag -v en gcsgrep). Las notas de exploración que enlaza están al lado, en notas-exploracion.md. Las referencias archivo:línea de la spec son relativas a <repo>/01-greenfield-gcsgrep/. Devolvé el veredicto con el formato de tu brief.

  ← resultado (completo):

```
Veredicto: HUECOS (3)

| # | Ítem | Resultado | Evidencia |
|---|------|-----------|-----------|
| 1 | Propósito y alcance | PASS | Propósito en una oración (spec:15-17). Tabla Dentro por archivo (spec:23-34) y Fuera concreta, no vacía (spec:38-64). Abrí todas las referencias `archivo:línea` de Dentro y Fuera contra el árbol de trabajo y coinciden: `cli.go:21-28/58-78/92-99`, `search.go:38-54/56-59/65-68/82-85/106-121/122-127`, `run.go:69/78-80/84-91/109-127`, `cli_test.go:61/93-95`, `usage_test.go:40`, `gcsgrep-spec.md:30/46/128-135/553/573-576`, `gcsgrep-cobertura-vc.md:53`, `gcsgrep-base-context.md:59`, `gcsgrep-plan.md:466`. No hay otros usos de `-v` como flag desconocido fuera de los listados. No pude confirmar que el commit `22104ba` exista (sin git). |
| 2 | Atomicidad | PASS | FR-1 a FR-17 tienen un Dado, un Cuando y un Entonces observable en una ejecución. FR-14 enumera variantes de un mismo caso (`-vn`, `-nv`, `-Ev`, `-iv`), todas con el mismo resultado; no hay "o" que esconda dos comportamientos. |
| 3 | Bordes y fallas | PASS | Patrón vacío (FR-16), regex inválida (FR-17), flag combinado (FR-14), repetido (FR-15), `-v` tras `--` (FR-13), objeto vacío (FR-6), CRLF, sin salto final, línea > 1 MiB (FR-7/8/9, BR-2), error de lectura a mitad de línea (BR-3). INV-5 cubre errores de uso, ubicación inválida, bucket/objeto/prefijo inexistente, tope de `--max`, sin permiso y sin ADC. El exit `2` por error de lectura con `-v` queda en Fuera con motivo escrito (spec:48-53) y se sostiene con INV-6. |
| 4 | Verificabilidad | FAIL | Los VCs de FR, BR y NFR dan comando, exit code y salida exacta; contrasté VC-1, 2, 3, 4, 5, 7, 8, 9, 11, VC-BR-1 y VC-BR-2 contra `test-fixtures/` y las salidas esperadas son correctas. VC-BR-3 discrimina (falla si la inversión queda antes de `search.go:122-124`). Fallan: VC-INV-1 no tiene piso (hueco 2), y los cambios de documentación de Dentro no tienen requisito ni VC y obligan a decidir (hueco 3). |
| 5 | Regresión (brownfield) | FAIL | Línea de base unitaria medida: 43 PASS (spec:71); la recontué en los `_test.go`: 23 en `cli`, 3 en `location`, 17 en `search`. Invariante de alcance (INV-4, 12 rutas = 10 de Dentro + spec + notas) y de comportamiento vecino (INV-2, INV-5, INV-6) presentes. Falla la suite e2e: línea de base "no medida" (spec:73-76) y VC-INV-3 pasa en vacío (hueco 1). |
| 6 | NFRs | PASS | NFR-1: < 4 MiB de `TotalAlloc` con una línea de 64 MiB (VC-NFR-1). NFR-2: mediana <= 1,15x con 320 000 líneas de 100 bytes, input y benchmarks definidos, script con exit code (VC-NFR-2). Ambos tienen número, carga y VC. |
| 7 | Simplicidad | PASS | Preguntas abiertas vacías (spec:500-502). Lo que excede el flag (actualizar docs de la base, `e2e/invert_test.go`, NFR-1/NFR-2) tiene su porqué escrito (hallazgo 1, spec:31, spec:465, spec:477-478). La forma larga `--invert-match` está excluida con motivo (spec:58). |

Huecos:
1. [ítem 5] <repo>/03-skills-&-agents/evidencia/demo/invert-match-spec.md:390-393 (y :73-76) — La línea de base e2e no está medida y VC-INV-3 no fija ningún piso: no exige exit code de `go test` (el pipe lo tapa) ni una cantidad mínima o lista de `--- PASS` en `base-e2e.txt`. Si `TestMain` aborta en los dos commits (faltan las variables, falta `gcloud`, `e2e/main_test.go:45-48` y `:69-73`), los dos archivos quedan vacíos, `comm -23` no imprime nada y el VC pasa sin haber corrido un solo test. Un `TestVC*` que falle en la base también queda fuera de la comparación sin que el VC lo note. Además, `base-e2e.txt` y `nuevo-e2e.txt` no figuran en la lista de archivos auxiliares de spec:88-90.
2. [ítem 4] <repo>/03-skills-&-agents/evidencia/demo/invert-match-spec.md:362-366 — VC-INV-1 tiene el mismo defecto del lado base: INV-1 dice "los 43 tests" (spec:358), pero el VC no exige que `base.txt` tenga 43 líneas ni el exit code de `go test`. Si la corrida en `/tmp/gcsgrep-base/...` no produce líneas (worktree mal armado, error de build), `comm -23` no imprime nada, `grep -c` da `0` y el VC pasa. Quien verifica tiene que decidir por su cuenta si un `base.txt` vacío o corto vale.
3. [ítem 4] <repo>/03-skills-&-agents/evidencia/demo/invert-match-spec.md:32-34 (y :26, la parte de doc de `LineFunc`/`Scan`) — Las ediciones de `sdd/gcsgrep-spec.md`, `sdd/gcsgrep-cobertura-vc.md` y `sdd/gcsgrep-base-context.md`, y el cambio de doc-comments en `search.go`, no tienen requisito ni VC: todos los VCs pueden dar verde sin tocarlas (VC-INV-4 solo acota por arriba; su único chequeo positivo es `cli.go`). Tampoco está escrito el texto resultante: "sumarlo a la lista de flags y a la sintaxis" no dice cómo queda la línea de sintaxis de `gcsgrep-spec.md:553`, y "enlazar esta spec" no dice con qué ruta ni texto en `gcsgrep-base-context.md:59`. Hay que decidirlo al implementar.
```

_Uso del subagent, según el aviso de fin de tarea: 32 tool calls, 80.653 tokens en su
ventana, 186 s. A la sesión principal volvió solo el bloque de arriba (unos 4,8k caracteres)._

## Qué muestra

- **El formato del brief se cumple**: `Veredicto`, la tabla fija de 7 ítems con PASS o
  FAIL y evidencia, y los huecos con `archivo:línea`. No propone redacción ni edita nada.
- **Solo lectura**: el propio revisor anota "sin git" cuando no puede confirmar el commit
  base. No tiene `Bash`.
- **El veredicto no es estable.** Esta misma spec había salido `LISTA` en la quinta
  pasada de la sesión 02, y una sexta pasada, en otra sesión, encuentra tres huecos
  nuevos (VCs de invariantes que pasan en vacío). La dejamos tal cual: no tocamos la
  spec para conseguir un `LISTA`. Es la misma lección de la sección 4: un revisor que es
  un modelo **persuade**, y lo único que garantiza en este toolkit es el hook.
