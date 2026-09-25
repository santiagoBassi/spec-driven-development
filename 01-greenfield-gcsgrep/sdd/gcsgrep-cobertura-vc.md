# gcsgrep — tabla de cobertura de VCs

> Salida del paso **Verificar**. Para cada VC dice **con qué se lo ejercita y qué se
> observó**. Un VC sin evidencia es un VC que no pasó.
>
> Estados: ✅ pasa · 🔸 implementado, VC pendiente (evidencia parcial; el resto necesita
> el servidor de prueba de la Iteración 2) · ⬜ no empezado.
>
> Detalle de la corrida, comandos y desvíos:
> [`iterations/01-busqueda-punta-a-punta.md`](./iterations/01-busqueda-punta-a-punta.md).

## Resumen (Iteración 1 cerrada)

| | |
|---|---|
| Requerimientos (FR + BR + NFR) | 65 |
| VCs definidos | 65 |
| VCs de la Iteración 1 | 18 |
| **VCs de la Iteración 1 pasando** | **18** |
| Implementados, cierran en la Iteración 2 | 4 (VC-12, 15a, 41, 42) |
| VCs sin empezar | 43 (14 de la Iteración 2, 14 de la 3, 12 de la 4 y 3 de la 5) |
| Requerimientos sin VC | 0 |

## Cobertura, uno por uno

### Iteración 1

| VC | Requerimiento | Ejercitado por | Se observa | Estado |
|---|---|---|---|---|
| VC-1 | FR-1 literal bajo un prefijo | `e2e` `TestVC01` | exit `0`, exactamente las 3 líneas `gs://$B/…:texto` | ✅ |
| VC-3 | FR-3 patrón literal | `e2e` `TestVC03` | `job_id=10.` → exit `1`; `(` sobre `logs/` → exit `1`, no `2` | ✅ |
| VC-4 | FR-4 regex con `-E` | `e2e` `TestVC04` | exit `0`, líneas `job_id=102` y `104`; con `job_id=10.` las 6 | ✅ |
| VC-5 | FR-5 regex inválida | `e2e` `TestVC05` (chequeo P) | exit `2`, stdout vacío, stderr `gcsgrep: invalid pattern: …`, 0 conexiones | ✅ |
| VC-6 | FR-6 patrón vacío | `e2e` `TestVC06` (3 casos, chequeo P) | exit `2`, stderr exactamente `gcsgrep: empty pattern`, 0 conexiones | ✅ |
| VC-7 | FR-7 `-i` | `e2e` `TestVC07` | 4 líneas con `-i`, 2 sin; `LANG=C -i 'ñandú árbol'` → `ÑANDÚ ÁRBOL`, exit `0`; sin `-i`, exit `1` | ✅ |
| VC-8 | FR-8 `-n` | `e2e` `TestVC08` | exit `0`, líneas `:4:` y `:14:`, en ese orden | ✅ |
| VC-9a | FR-9a recorte de `\r` | `e2e` `TestVC09a` | `:2:Linea 2 con timeout y terminacion Windows`, sin bytes `0x0d`; `Windows$` exit `0` | ✅ |
| VC-9b | FR-9b última línea sin `\n` | `e2e` `TestVC09b` | `-n -E 'salto final$'` sobre `data/sin_salto_final.txt` → exit `0`, stdout exactamente `…:2:Ultima linea sin salto final` + `\n` | ✅ |
| VC-14 | FR-14 ubicación inválida | `e2e` `TestVC14` (5 casos, chequeo P) | exit `2`, stderr `gcsgrep: invalid location: "<ubic>"`, 0 conexiones | ✅ |
| VC-15b | FR-15b prefijo sin objetos | `e2e` `TestVC15b` | `gs://$B/no-existe/` → exit `1`, stdout y stderr vacíos | ✅ |
| VC-21 | FR-21 `--` | `e2e` `TestVC21` | exit `0`, las 5 líneas de `postgres.log` | ✅ |
| VC-22 | FR-22 flag desconocido | `e2e` `TestVC22` (2 casos, chequeo P) | exit `2`, stderr `gcsgrep: unknown flag: "-1]"` / `"-v"`, 0 conexiones | ✅ |
| VC-23 | FR-23 dos posicionales | `e2e` `TestVC23` (3 casos, chequeo P) | exit `2`, `expected 2 arguments … got 0/1/3`, 0 conexiones | ✅ |
| VC-35 | FR-35 sin ADC | `e2e` `TestVC35` | exit `2`, stderr exacto de FR-35, 0 conexiones al listener | ✅ |
| VC-39 | BR-1 solo lectura | `e2e` `TestVC39` (`zz_vc39_test.go`) | todos los objetos de `$B` (24; 23 en las corridas del 2026-09-23) con (nombre, generación, metageneración) idénticos antes y después; `lectora` solo tiene `objects.get` y `objects.list` | ✅ |
| VC-40 | BR-2 no ampliar el acceso | `e2e` `TestVC40` | `sin-acceso`: exit `2`, `access denied: <ubicación>` (prefijo y objeto); `lectora`: exit `0` con la salida de VC-1 | ✅ |
| VC-46 | BR-8 línea de más de 1 MiB | `e2e` `TestVC46` (3 casos) | prefijo + 1 048 576 bytes + `...`, idéntico con match temprano, tardío y con regex que cruza el límite | ✅ |

### Con evidencia parcial: cierran en la Iteración 2

Estos requisitos están implementados, pero su VC, o una parte de él, solo se observa con el servidor
de prueba. **No** figuran como "pasa".

| VC | Requerimiento | Evidencia de la Iteración 1 | Lo que falta observar | Estado |
|---|---|---|---|---|
| VC-12 | FR-12 objeto puntual | `TestVC12Parcial` PASS: exit `0`, solo líneas de `api.log` | Que no haya request de listado ni lectura de `a.log.bak` | 🔸 |
| VC-15a | FR-15a bucket sin objetos | El código trata igual un bucket vacío que un prefijo vacío, y VC-15b pasa | Bucket existente sin objetos (`empty`) | 🔸 |
| VC-41 | BR-3 tope de objetos | `TestVC41Parcial` PASS: `--max 5` aborta con el mensaje exacto; `--max 6` → exit `0` | Tope por defecto de 1000 y que no se pida la página siguiente | 🔸 |
| VC-42 | BR-4 `--max` | `TestVC42Parcial` PASS: `0`, `-3`, `abc`, `1.5`, `+5`, ` 5` → exit `2` con el mensaje exacto; `--max` sin valor → exit `2`, `gcsgrep: flag --max requires a value` (D-20); todos con chequeo P. `TestVC42ParcialEnteroGrande` PASS: `--max 99999999999999999999` → exit `0` | `--max unlimited` leyendo 2500 objetos | 🔸 |

### Iteraciones siguientes

| Iteración | VCs | Estado |
|---|---|---|
| 2 · Servidor de prueba, sniffing, `-c`/`-l` | VC-2, 10, 11, 17a, 17b, 18a, 18b, 19, 20a, 20b, 20c, 43, 44, 45 (y cerrar 12, 15a, 41, 42) | ⬜ |
| 3 · Errores de GCS y fallos de red | VC-13a, 13b, 16a, 16b (con un solo mensaje `not found`), 29a, 29b, 30a, 30b, 31, 32, 33a, 33b, 34, 50 | ⬜ |
| 4 · Concurrencia, progreso y parser estricto | VC-24a, 24b, 25, 26a, 26b, 26c, 26d, 27, 28, 36, 37, 38 | ⬜ |
| 5 · NFRs y scripting | VC-47, 48, 49 | ⬜ |

## Evidencia que se ejecutó

Fecha: 2026-09-23. Partida: commit `840084a`. Go 1.22.2, `GOTOOLCHAIN=local`. Bucket real
`gs://sdd-fardenghi-itba`, identidades `lectora` y `sin-acceso` por claves de service account.

```bash
$ go vet ./...                       # sin hallazgos; gofmt -l . vacío
$ go test ./internal/... -count=1
ok  gcsgrep/internal/cli
ok  gcsgrep/internal/location
ok  gcsgrep/internal/search          # bordes de línea: 1 MiB, \r en el borde de buffer, anclas

$ go build -o gcsgrep ./cmd/gcsgrep
$ go test ./e2e -count=1 -v
--- PASS: TestVC01, 03, 04, 05, 06, 07, 08, 09, 14, 21, 22, 23, 35, 39, 40, 46
--- PASS: TestVC12Parcial, TestVC15Parcial, TestVC41Parcial, TestVC42Parcial
    zz_vc39_test.go:30: 23 objects in the bucket, identical (name, generation, metageneration)
                        before and after the suite
PASS
ok  gcsgrep/e2e  48.493s
```

**Corrida posterior al cierre** (2026-09-23, partida: commit `5da1857` más el caso de D-20). En otra
máquina: macOS 26.6.2 (arm64), Go 1.27.1, `GOTOOLCHAIN=local`; mismo bucket y mismas identidades.
Reproduce el resultado del cierre y agrega la evidencia de `--max` sin valor para VC-42 (D-20).

```bash
$ go vet ./...                       # sin hallazgos; gofmt -l . vacío
$ go test ./internal/... -count=1    # ok en cli, location y search
$ go build -o gcsgrep ./cmd/gcsgrep
$ go test ./e2e -count=1 -v
--- PASS: TestVC01, 03, 04, 05, 06, 07, 08, 09, 14, 21, 22, 23, 35, 39, 40, 46
--- PASS: TestVC12Parcial, TestVC15Parcial, TestVC41Parcial, TestVC42Parcial
    --- PASS: TestVC42Parcial/without_value
    zz_vc39_test.go:30: 23 objects in the bucket, identical (name, generation, metageneration)
                        before and after the suite
PASS
ok  gcsgrep/e2e  39.884s
```

En esas dos corridas, los tests de VC-9a y VC-15b se llamaban `TestVC09` y `TestVC15Parcial`; las
aserciones son las mismas.

**Corrida del 2026-09-25** (macOS 26.6.2 arm64, Go 1.27.1, `GOTOOLCHAIN=local`; mismo bucket y mismas
identidades). `$B` ya tiene `data/sin_salto_final.txt` (24 objetos). Agrega VC-9b, y los casos `+5`,
` 5` y el entero más grande que un `int` de VC-42.

```bash
$ go vet ./...                       # sin hallazgos; gofmt -l . vacío
$ go test ./internal/... -count=1    # ok en cli, location y search
$ go test ./e2e -count=1 -v
--- PASS: TestVC01, 03, 04, 05, 06, 07, 08, 09a, 09b, 14, 15b, 21, 22, 23, 35, 39, 40, 46
--- PASS: TestVC12Parcial, TestVC41Parcial, TestVC42Parcial, TestVC42ParcialEnteroGrande
    --- PASS: TestVC42Parcial/0, -3, abc, 1.5, +5, _5, without_value
    zz_vc39_test.go:30: 24 objects in the bucket, identical (name, generation, metageneration)
                        before and after the suite
PASS
ok  gcsgrep/e2e  42.868s
```

Comprobaciones adicionales, que respaldan cómo se lee la tabla:

- **El chequeo P observa algo.** Se comprobó aparte que el listener registra una conexión cuando la
  herramienta intenta conectarse a un endpoint (ADC ficticio, invocación válida); por eso "0 conexiones"
  es una observación real y no un listener que no escucha.
- **VC-39 demuestra algo.** `testIamPermissions` (no escribe) sobre `$B`: `lectora` →
  `storage.objects.get`, `storage.objects.list`; sin `create`, `update`, `delete` ni `buckets.get`.
  `sin-acceso` → ninguno.
- **Los tests de línea larga detectan el error que evitan.** Una mutación (matchear solo el prefijo
  retenido) hace fallar los tests de `$` y `\b` (`DECISIONS.md` D-10).

## Qué mirar en esta tabla

1. Cada fila tiene un ejercitador nombrado, y la columna "Se observa" es observable.
2. Los cuatro VCs implementados que dependen del servidor de prueba no figuran como "pasa" hasta la
   Iteración 2.
3. De los 18 VCs de la Iteración 1, 13 ejercitan fallas o bordes (VC-3, 5, 6, 9a, 9b, 14, 15b, 21, 22,
   23, 35, 40, 46); no es una tabla de caminos felices.
