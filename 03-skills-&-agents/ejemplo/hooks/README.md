# Hooks

Tres hooks para un repo que registra sus decisiones en ADRs. **Corren de verdad** —
no son pseudocódigo de slide.

| Hook | Evento | Qué garantiza | ¿Bloquea? |
|---|---|---|---|
| `adr-immutable.sh` | `PreToolUse` · Edit/Write | Un ADR aceptado no se reescribe; solo cambia su estado | Sí |
| `adr-for-deps.sh` | `PreToolUse` · Bash | No entra un cambio de dependencias sin un ADR nuevo | Sí |
| `adr-index.sh` | `PostToolUse` · Edit/Write | El índice `docs/adr/README.md` siempre al día | No |

## Por qué un hook y no una rule

`CLAUDE.md` podría decir "nunca edites un ADR aceptado". Es una instrucción: el
modelo la puede pasar por alto, sobre todo con un pedido como "actualizá el ADR-0001
para que refleje lo que hacemos hoy", que *suena* razonable.

`adr-immutable.sh` no discute: si el archivo dice `Estado: Aceptado`, la edición no
pasa. **Skill, rule y subagent persuaden. El hook garantiza.**

Fijate que los dos hooks que bloquean protegen un **invariante del proyecto**, no una
práctica genérica. Esa es la forma que conviene copiar.

## Requisitos

- `jq` en el `PATH` (`brew install jq` / `apt install jq`).
- `git`, para `adr-for-deps.sh`.
- Funcionan con el Bash 3.2 que trae macOS.

Si falta `jq`, los hooks que bloquean salen con código `1`: un error **no**
bloqueante. Es deliberado — un hook roto no debería impedirte trabajar.

## Instalación

Desde la raíz del proyecto donde los querés usar:

```bash
mkdir -p .claude/hooks
cp examples/03-skills/ejemplo-guiado/hooks/*.sh .claude/hooks/
chmod +x .claude/hooks/*.sh
```

Después fusioná `settings.json` de esta carpeta dentro de `.claude/settings.json` de
tu proyecto. Si ya tenés uno, mezclá la clave `hooks` a mano; no lo pises.

`$CLAUDE_PROJECT_DIR` lo define Claude Code y apunta a la raíz del proyecto, así que
los paths de `settings.json` funcionan sin editar. **Reiniciá la sesión del agente**
después de cambiar `settings.json`.

## Qué adaptar

| Archivo | Variable | Por defecto |
|---|---|---|
| los tres | `ADR_DIR` | `docs/adr` |
| `adr-for-deps.sh` | `DEPS_REGEX` | `pyproject.toml`, `requirements*.txt`, `package.json`, `go.mod`, `Cargo.toml` |

## Cómo probar que bloquean

Un hook que existe pero nunca bloqueó nada no está terminado. Los hooks leen el
evento por stdin, así que se pueden ejercitar sin el agente. Desde la raíz de un repo
con `docs/adr/` (podés copiar los dos ADRs de [`../docs/adr/`](../docs/adr/)):

### Un ADR aceptado no se reescribe

```bash
export CLAUDE_PROJECT_DIR="$(pwd)"
ADR="$(pwd)/docs/adr/0001-postgres-para-los-turnos.md"

# 1. Reescribirlo: bloqueado.
echo "{\"tool_name\":\"Write\",\"tool_input\":{\"file_path\":\"$ADR\"}}" \
  | .claude/hooks/adr-immutable.sh
echo "exit=$?"     # → 2, con el motivo en stderr

# 2. Cambiar solo la línea de estado: pasa.
echo "{\"tool_name\":\"Edit\",\"tool_input\":{\"file_path\":\"$ADR\",
  \"old_string\":\"Estado: Aceptado\",\"new_string\":\"Estado: Reemplazado por ADR-0003\"}}" \
  | .claude/hooks/adr-immutable.sh
echo "exit=$?"     # → 0
```

### Un cambio de dependencias sin ADR no se commitea

```bash
export CLAUDE_PROJECT_DIR="$(pwd)"
echo 'redis = "^5"' >> pyproject.toml && git add pyproject.toml

echo '{"tool_input":{"command":"git commit -m \"sumar redis\""}}' \
  | .claude/hooks/adr-for-deps.sh
echo "exit=$?"     # → 2

# Agregá el ADR al staging y repetí:
git add docs/adr/0003-*.md
echo '{"tool_input":{"command":"git commit -m \"sumar redis\""}}' \
  | .claude/hooks/adr-for-deps.sh
echo "exit=$?"     # → 0
```

Para la demo en vivo, hacelo **dentro** de una sesión del agente: pedile que sume una
dependencia y que commitee. El bloqueo aparece en la transcripción, y el agente — que
leyó el stderr — se pone a escribir el ADR en vez de reintentar.

## El contrato de exit codes

| Código | Significado en `PreToolUse` |
|---|---|
| `0` | Deja pasar la acción |
| `2` | **Veta** la acción; stderr se le muestra al agente |
| otro | Error no bloqueante: se loguea, la acción pasa igual |

En `PostToolUse` la acción ya ocurrió, así que el exit code no la veta. Por eso
`adr-index.sh` siempre sale `0`.

**Lo que escribís a stderr importa:** es lo que lee el agente cuando lo bloqueás. Un
mensaje que dice qué pasó y qué hacer lo hace corregir; uno que dice "bloqueado" a
secas lo hace reintentar lo mismo.

## Limitaciones conocidas

Están acá porque son buenos ejemplos de lo que hay que escribir cuando se diseña un
guardrail.

- **`adr-for-deps.sh` mira el staging.** `git commit -a` agrega archivos *durante* el
  commit, después de que el hook miró: un cambio de dependencias así se escapa. Un
  hook más estricto bloquearía `commit -a` cuando hay manifiestos modificados.
- **`adr-for-deps.sh` matchea el texto del comando.** `echo "git commit"` dispara el
  chequeo. Falso positivo conservador, y preferible al inverso.
- **`adr-immutable.sh` confía en la línea `Estado:`.** Un ADR aceptado que no sigue la
  plantilla no queda protegido.
- **`adr-index.sh` reescribe el índice entero** en cada edición. Con cientos de ADRs
  convendría hacerlo incremental.
