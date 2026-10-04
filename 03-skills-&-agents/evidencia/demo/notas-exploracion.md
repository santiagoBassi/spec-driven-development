# Notas de exploración — `-v` en gcsgrep

> Generadas por un subagent `Explore` (solo lectura) para la spec
> [`invert-match-spec.md`](./invert-match-spec.md). Base: `01-greenfield-gcsgrep/` en el
> commit `22104ba`. Toda referencia `archivo:línea` es relativa a `01-greenfield-gcsgrep/`.
> Se citan desde la spec como "hallazgo N".

## Hallazgo 1 · `-v` está excluido de forma explícita en la base

- `sdd/gcsgrep-spec.md:46` lo pone fuera de alcance ("Flags de `grep` que no están en la
  lista de arriba: `-v`, `-r`, …"); `sdd/gcsgrep-plan.md:466` lo difiere post-v1.
- Es el ejemplo de flag desconocido en VC-22 (`sdd/gcsgrep-spec.md:573-576`),
  `e2e/usage_test.go:40`, `internal/cli/cli_test.go:61` y `internal/cli/cli_test.go:93`
  (`TestParseErrorOrder`). Al reconocer `-v`, esos cuatro lugares tienen que pasar a otro
  flag desconocido (`-w`, que sigue fuera de alcance en `sdd/gcsgrep-spec.md:46`).

## Hallazgo 2 · Parseo de flags (`internal/cli/cli.go`)

- `Config` (`:21-28`): `Extended` (-E), `IgnoreCase` (-i), `LineNumber` (-n), `Max`,
  `Pattern`, `Location`.
- Loop manual con `switch a` y coincidencia exacta (`:58-78`). `--` vuelve posicional lo
  que sigue (`:53-56`). Los flags combinados (`-in`) caen en `unknown flag: %q` (`:77`).
  Los repetidos se aceptan y gana el último (D-08, `sdd/DECISIONS.md:90-92`).
- Orden de errores (D-08): flags en orden de argv → cantidad de posicionales → patrón
  vacío; después `search.Compile` y `location.Parse`. Todo antes de tocar GCS
  (`cmd/gcsgrep/run.go:24-36`).
- `TestParseValid` compara el `Config` entero con `!=` (`cli_test.go:43`).

## Hallazgo 3 · Selección de líneas (`internal/search/search.go`)

- `Compile` (`:38-54`): literal con `QuoteMeta` sin `-E`; RE2 con `-E`; `(?i)` con `-i`.
- `Scanner.Scan` (`:86-137`): la decisión de emitir es `if matched {` en `:127`.
  `lineNo` se incrementa en todas las líneas (`:126`), así que `-n` numera bien aunque se
  emitan las no-matcheantes.
- `matched` sale de `re.Match` (`:108`) o, para líneas de más de 1 MiB, de
  `re.MatchReader` en streaming sobre la línea entera más el drenado del resto (`:114-120`,
  D-10).
- Objeto vacío: `return` sin llamar a `fn` (`:99-102`). `"abc\n"` es una línea (no hay
  línea fantasma); `"\n\n"` son dos líneas vacías.
- CRLF: el `\r` se recorta antes de matchear e imprimir (D-11).
- Error de lectura a mitad de línea: `return ls.err` en `:122-124`, **antes** de `:127`;
  la línea cortada nunca se evalúa (D-12).
- Contrato documentado: `LineFunc` "una vez por línea que matchea" (`:56-59`), `Scan`
  (`:82-85`).

## Hallazgo 4 · Salida y exit codes (`cmd/gcsgrep/run.go`)

- `formatLine` (`:109-127`): `gs://<bucket>/<obj>:[<n>:]<texto>[...]\n`.
- `matched = true` dentro del callback (`:75`), es decir, cuando se emitió al menos una
  línea. Exit: `failed` → 2, `matched` → 0, si no → 1 (`:84-91`).
- `Scanner` se crea una vez por corrida (`:69`) y se reusa entre objetos.

## Hallazgo 5 · Binarios

El sniffing de BR-5/BR-6 de la base no está implementado (Iteración 1 cerrada,
`sdd/CONTEXT.md:8-15,36-37`): hoy un binario se busca como texto, con o sin flags.

## Hallazgo 6 · Tests

- `search` recibe un `io.Reader`; se testea con `strings.NewReader`. Helpers en
  `internal/search/search_test.go`: `scanAll` (`:84-95`), `mustCompile` (`:276`),
  `assertLines` (`:285`), `errAfterReader` (`:261-274`).
- `cmd/gcsgrep` no tiene tests unitarios: `run`, `formatLine` y los exit codes se cubren
  en `e2e/`, que exige bucket real y credenciales (`e2e/main_test.go:37-80`, D-15).
- Los flags `-c`, `-l`, `--concurrency` y la pista de FR-24a/FR-25 son de iteraciones
  futuras: no existen en `22104ba`.

## Riesgos para `-v`

1. Binarios: sin sniff, `-v` imprime casi todo un binario (hallazgo 5).
2. Lectura parcial: la inversión tiene que quedar después de `:122-124` o se imprime la
   línea cortada.
3. Líneas > 1 MiB: el match decide sobre la línea entera, no sobre lo impreso.
4. Exit 1 cuando todas las líneas matchean o no hay líneas.
5. Cambiar `Scan` de firma toca `scanAll` y tres llamadas directas en `search_test.go`.
