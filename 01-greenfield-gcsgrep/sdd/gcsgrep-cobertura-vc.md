# gcsgrep — tabla de cobertura de VCs

> Salida del paso **Verificar**. Para cada VC dice **con qué se lo ejercita y qué se
> observó**. Un VC sin evidencia es un VC que no pasó.
>
> Estados: ✅ pasa · 🔸 implementado, VC pendiente (evidencia parcial; el resto necesita
> el servidor de prueba de la Iteración 2) · ⬜ no empezado.
>
> Se completa a medida que pasan los VCs, según el [plan](./gcsgrep-plan.md).

## Resumen

**Iteración 1 cerrada:** 18 ✅, 4 🔸 (VC-12, VC-15a, VC-41, VC-42), 48 ⬜. 70 VCs en
total (58 FR + 9 BR + 3 NFR), como fija la spec.

## Cobertura, uno por uno

| VC | Requerimiento | Ejercitado por | Se observa | Estado |
|---|---|---|---|---|
| VC-1 | FR-1 | `e2e.TestVC01` | `gcsgrep timeout gs://$B/logs/app/` → exit 0, las 3 líneas exactas de la spec | ✅ |
| VC-2 | FR-2 | — | cierra en Iteración 2 (necesita sniffing para los 6 avisos) | ⬜ |
| VC-3 | FR-3 | `e2e.TestVC03` | `job_id=10.` literal no matchea; `(` literal no rompe, exit 1 | ✅ |
| VC-4 | FR-4 | `e2e.TestVC04` | regex RE2 matchea job_id=102/104; `job_id=10.` como regex matchea las 6 líneas | ✅ |
| VC-5 | FR-5 | `e2e.TestVC05` (chequeo P) | `-E '('` → exit 2, `invalid pattern: ` + detalle del motor | ✅ |
| VC-6 | FR-6 | `e2e.TestVC06` (chequeo P) | patrón vacío (con/sin `-E`, tras `--`) → exit 2, `empty pattern` | ✅ |
| VC-7a | FR-7a | `e2e.TestVC07a` | `-i` matchea 4 líneas (vs 2 sin `-i`); `ñandú árbol`/`LANG=C` matchea `ÑANDÚ ÁRBOL` | ✅ |
| VC-7b | FR-7b | — | cierra en Iteración 2 (necesita el servidor de prueba, `fake/eszett.log`) | ⬜ |
| VC-8 | FR-8 | `e2e.TestVC08` | `-n` numera desde 1, orden `4` antes de `14` dentro del objeto | ✅ |
| VC-9a | FR-9a | `e2e.TestVC09a` | recorte de `\r`, sin bytes `0x0d` en stdout; `Windows$` matchea tras el recorte | ✅ |
| VC-9b | FR-9b | `e2e.TestVC09b` | última línea sin `\n` se busca y se imprime terminada en `0a` | ✅ |
| VC-10 | FR-10 | — | cierra en Iteración 2 (`-l` no implementado) | ⬜ |
| VC-11 | FR-11 | — | cierra en Iteración 2 (`-l` no implementado) | ⬜ |
| VC-12 | FR-12 | `e2e.TestVC12Parcial` | objeto puntual busca solo en `api.log` | 🔸 (falta: sin request de listado, sin leer `.bak`, con el servidor de prueba) |
| VC-13a | FR-13a | — | cierra en Iteración 3 | ⬜ |
| VC-13b | FR-13b | — | cierra en Iteración 3 | ⬜ |
| VC-14 | FR-14 | `e2e.TestVC14` (chequeo P) | 5 ubicaciones inválidas → exit 2, mensaje exacto con la ubicación recibida | ✅ |
| VC-15a | FR-15a | `location.TestVC15aParcial` + VC-15b | `gs://b/` parsea al mismo modo/prefijo vacío que un prefijo sin objetos | 🔸 (falta: bucket `empty` real, con el servidor de prueba) |
| VC-15b | FR-15b | `e2e.TestVC15b` | prefijo sin objetos → exit 1, stdout y stderr vacíos | ✅ |
| VC-16a | FR-16a | — | cierra en Iteración 3 | ⬜ |
| VC-16b | FR-16b | — | cierra en Iteración 3 | ⬜ |
| VC-17a | FR-17a | — | cierra en Iteración 2 | ⬜ |
| VC-17b | FR-17b | — | cierra en Iteración 2 | ⬜ |
| VC-17c | FR-17c | — | cierra en Iteración 4 | ⬜ |
| VC-18a | FR-18a | — | cierra en Iteración 2 (`-c` no implementado) | ⬜ |
| VC-18b | FR-18b | — | cierra en Iteración 2 | ⬜ |
| VC-19a | FR-19a | — | cierra en Iteración 2 (`-l` no implementado) | ⬜ |
| VC-19b | FR-19b | — | cierra en Iteración 2 | ⬜ |
| VC-19c | FR-19c | — | cierra en Iteración 2 | ⬜ |
| VC-20a | FR-20a | — | cierra en Iteración 2 | ⬜ |
| VC-20b | FR-20b | — | cierra en Iteración 2 | ⬜ |
| VC-20c | FR-20c | — | cierra en Iteración 2 | ⬜ |
| VC-21 | FR-21 | `e2e.TestVC21` | todo lo que sigue a `--` es posicional; `-1]` matchea las 5 líneas de `postgres.log` | ✅ |
| VC-22 | FR-22 | `e2e.TestVC22` (chequeo P) | flag desconocido (`-1]`, `-v`) → exit 2, `unknown flag: "..."` | ✅ |
| VC-23 | FR-23 | `e2e.TestVC23` (chequeo P) | 0, 1 y 3 posicionales → exit 2, mensaje con `<n>` correcto | ✅ |
| VC-24a | FR-24a | — | cierra en Iteración 4 | ⬜ |
| VC-24b | FR-24b | — | cierra en Iteración 4 | ⬜ |
| VC-25 | FR-25 | — | cierra en Iteración 4 | ⬜ |
| VC-26a | FR-26a | — | cierra en Iteración 4 | ⬜ |
| VC-26b | FR-26b | — | cierra en Iteración 4 | ⬜ |
| VC-26c | FR-26c | — | cierra en Iteración 4 | ⬜ |
| VC-26d | FR-26d | — | cierra en Iteración 4 | ⬜ |
| VC-27 | FR-27 | — | cierra en Iteración 4 | ⬜ |
| VC-28 | FR-28 | — | cierra en Iteración 4 (`--concurrency` no implementado) | ⬜ |
| VC-29a | FR-29a | — | cierra en Iteración 3 | ⬜ |
| VC-29b | FR-29b | — | cierra en Iteración 3 | ⬜ |
| VC-29c | FR-29c | — | cierra en Iteración 3 | ⬜ |
| VC-30a | FR-30a | — | cierra en Iteración 3 | ⬜ |
| VC-30b | FR-30b | — | cierra en Iteración 3 | ⬜ |
| VC-31 | FR-31 | — | cierra en Iteración 3 | ⬜ |
| VC-32 | FR-32 | — | cierra en Iteración 3 | ⬜ |
| VC-33a | FR-33a | — | cierra en Iteración 3 | ⬜ |
| VC-33b | FR-33b | — | cierra en Iteración 3 | ⬜ |
| VC-34 | FR-34 | — | cierra en Iteración 3 | ⬜ |
| VC-35 | FR-35 | `e2e.TestVC35` (chequeo P) | sin ADC → exit 2, mensaje fijo, 0 conexiones al listener | ✅ |
| VC-36 | FR-36 | — | cierra en Iteración 4 | ⬜ |
| VC-37 | FR-37 | — | cierra en Iteración 4 | ⬜ |
| VC-38 | FR-38 | — | cierra en Iteración 4 | ⬜ |
| VC-39 | BR-1 | `e2e.TestVC39` | foto de `$B` con `lectora` idéntica antes/después de toda la suite | ✅ |
| VC-40 | BR-2 | `e2e.TestVC40` | `sin-acceso` → `access denied` (prefijo y objeto puntual); `lectora` sigue matcheando | ✅ |
| VC-41 | BR-3 | `e2e.TestVC41Parcial` | `--max 5` sobre `edge-cases/` aborta con el mensaje exacto; `--max 6` → exit 0 | 🔸 (falta: tope por defecto de 1000 sin paginar de más, con el servidor de prueba) |
| VC-42 | BR-4 | `e2e.TestVC42Parcial` | 6 valores inválidos y `--max` sin valor → exit 2; overflow → tope que nunca se alcanza | 🔸 (falta: `--max unlimited` leyendo 2500 objetos, con el servidor de prueba) |
| VC-43 | BR-5 | — | cierra en Iteración 2 | ⬜ |
| VC-44 | BR-6 | — | cierra en Iteración 2 (sniffing no implementado) | ⬜ |
| VC-45 | BR-7 | — | cierra en Iteración 2 | ⬜ |
| VC-46 | BR-8 | `e2e.TestVC46` | línea de 1 099 869 bytes: match temprano/tardío/cruzando el límite, salida truncada a 1 048 576 bytes + `...` | ✅ |
| VC-47 | BR-9 | — | cierra en Iteración 5 (audita todos los casos de falla ya cubiertos) | ⬜ |
| VC-48 | NFR-1 | — | cierra en Iteración 5 | ⬜ |
| VC-49 | NFR-2 | — | cierra en Iteración 5 | ⬜ |
| VC-50 | NFR-3 | — | cierra en Iteración 3 | ⬜ |

## Evidencia que se ejecutó

**2026-09-27**, commit de partida `4c90e25a4b499a36313e128424fe112b9544c3d7`, Go
1.22.2 linux/amd64, `cloud.google.com/go/storage v1.50.0`, `$B = sdd-fardenghi-itba`.

```
$ go build -o gcsgrep ./cmd/gcsgrep && go vet ./...
(sin salida, exit 0)

$ go test ./internal/...
ok    gcsgrep/internal/cli        0.004s
ok    gcsgrep/internal/location   0.004s
ok    gcsgrep/internal/search     0.314s
?     gcsgrep/internal/gcs        [no test files]
?     gcsgrep/internal/output     [no test files]

$ GCSGREP_BUCKET=sdd-fardenghi-itba GCSGREP_LECTORA=$PWD/lectora.json \
  GCSGREP_SIN_ACCESO=$PWD/sin-acceso.json go test ./e2e -v -count=1
--- PASS: TestVC15b (0.40s)
--- PASS: TestVC35 (0.18s)
--- PASS: TestVC40 (2.60s)
--- PASS: TestVC12Parcial (1.19s)
--- PASS: TestVC41Parcial (4.39s)
--- PASS: TestVC42Parcial (2.50s)
--- PASS: TestVC01 (1.20s)
--- PASS: TestVC03 (3.56s)
--- PASS: TestVC04 (2.16s)
--- PASS: TestVC07a (4.07s)
--- PASS: TestVC08 (1.01s)
--- PASS: TestVC09a (2.06s)
--- PASS: TestVC09b (0.55s)
--- PASS: TestVC21 (1.00s)
--- PASS: TestVC46 (6.05s)
--- PASS: TestVC05 (0.19s)
--- PASS: TestVC06 (0.56s)
--- PASS: TestVC14 (0.93s)
--- PASS: TestVC22 (0.37s)
--- PASS: TestVC23 (0.56s)
--- PASS: TestVC39 (2.42s)
PASS
ok    gcsgrep/e2e    42.114s
```

Corrida completa (build + vet + unitarios + e2e) repetida dos veces, mismo
resultado las dos veces. Detalle de cada VC, deviaciones y dificultades en
[`iterations/01-busqueda-punta-a-punta.md`](./iterations/01-busqueda-punta-a-punta.md).
