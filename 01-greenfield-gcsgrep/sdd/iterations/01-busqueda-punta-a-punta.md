# Iteración 1 — Búsqueda de punta a punta

> Registro inmutable. Lo aprendido para las siguientes iteraciones vive en
> `CONTEXT.md` y `DECISIONS.md`, no acá.

**SHA de partida:** `4c90e25a4b499a36313e128424fe112b9544c3d7` ("Mark spec as
reviewed and fix stale requirement count in the plan"). La implementación anterior
ya se había borrado en `7b8d597` para reiniciar desde la spec; esta iteración
escribió el código desde cero (solo se recuperó `go.mod`/`go.sum` del historial,
ver D-01).

**Toolchain:** Go 1.22.2 linux/amd64 (`GOTOOLCHAIN=local`), `cloud.google.com/go/storage
v1.50.0`, `gcloud` 401.0.0+ en el PATH. Bucket de fixtures `$B = sdd-fardenghi-itba`
(`us-east1`), identidades `lectora`/`sin-acceso` como claves JSON.

## Gate

```bash
go build -o gcsgrep ./cmd/gcsgrep && go vet ./...
go test ./internal/...
GCSGREP_BUCKET=sdd-fardenghi-itba GCSGREP_LECTORA=$PWD/lectora.json \
  GCSGREP_SIN_ACCESO=$PWD/sin-acceso.json go test ./e2e -v -count=1
```

Corrido dos veces completo (build limpio + los tres comandos), ambas verdes:
`go test ./e2e` tardó 42.1s y 42.3s. `go test ./internal/...` es prácticamente
instantáneo salvo `internal/search` (~0.3s, por los tests con líneas de más de 1
MiB).

## VCs — resultado

Los 18 criterios de éxito de la Iteración 1 pasan. Los 4 que el plan marca con
evidencia parcial quedan así, no como "pasa".

| VC | Test / evidencia | Resultado |
|---|---|---|
| VC-1 | `e2e.TestVC01` | ✅ PASS |
| VC-3 | `e2e.TestVC03` | ✅ PASS |
| VC-4 | `e2e.TestVC04` | ✅ PASS |
| VC-5 | `e2e.TestVC05` (chequeo P) | ✅ PASS |
| VC-6 | `e2e.TestVC06` (chequeo P) | ✅ PASS |
| VC-7a | `e2e.TestVC07a` | ✅ PASS |
| VC-8 | `e2e.TestVC08` | ✅ PASS |
| VC-9a | `e2e.TestVC09a` | ✅ PASS |
| VC-9b | `e2e.TestVC09b` | ✅ PASS |
| VC-14 | `e2e.TestVC14` (chequeo P) | ✅ PASS |
| VC-15b | `e2e.TestVC15b` | ✅ PASS |
| VC-21 | `e2e.TestVC21` | ✅ PASS |
| VC-22 | `e2e.TestVC22` (chequeo P) | ✅ PASS |
| VC-23 | `e2e.TestVC23` (chequeo P) | ✅ PASS |
| VC-35 | `e2e.TestVC35` (chequeo P) | ✅ PASS |
| VC-39 | `e2e.TestVC39` | ✅ PASS |
| VC-40 | `e2e.TestVC40` | ✅ PASS |
| VC-46 | `e2e.TestVC46` | ✅ PASS |
| VC-12 | `e2e.TestVC12Parcial` | 🔸 implementado, VC pendiente |
| VC-15a | `location.TestVC15aParcial` + VC-15b | 🔸 implementado, VC pendiente |
| VC-41 | `e2e.TestVC41Parcial` | 🔸 implementado, VC pendiente |
| VC-42 | `e2e.TestVC42Parcial` | 🔸 implementado, VC pendiente |

El detalle de qué falta para cada parcial está en `gcsgrep-cobertura-vc.md` y en
`CONTEXT.md` ("Qué falta para que los 4 parciales cierren").

## Demostrable así (verificado a mano, `ADC = lectora`)

```
$ ./gcsgrep timeout gs://$B/logs/app/
gs://.../logs/app/api.log:2026-09-22 10:05:22 ERROR Request timeout after 3000ms uri=/api/v1/checkout
gs://.../logs/app/api.log:2026-09-22 10:50:23 ERROR Gateway timeout=504 from upstream payment service
gs://.../logs/app/worker.log:job_id=102 queue=default status=ERROR reason=timeout_exceeded retries=3
exit=0

$ ./gcsgrep -n -i timeout gs://$B/logs/app/api.log
(4 líneas numeradas)                                    exit=0

$ ./gcsgrep -E 'job_id=\d+ queue=\w+ status=ERROR' gs://$B/logs/app/worker.log
(2 líneas: job_id=102 y job_id=104)                     exit=0

$ ./gcsgrep timeout gs://$B/no-existe/
(nada)                                                  exit=1

$ ./gcsgrep --max 5 timeout gs://$B/edge-cases/
gcsgrep: more than 5 objects under gs://.../edge-cases/; use --max <N> or --max unlimited
exit=2

$ ./gcsgrep timeout logs/
gcsgrep: invalid location: "logs/"
exit=2
```

Los seis coinciden exactamente con lo que pide el plan.

## Desvíos del plan

Ninguno en el *qué*. Un ajuste de implementación no anticipado en el detalle del
plan:

- **`location.Location.ListPrefix()`**: el prefijo que se le pasa al listado de GCS
  lleva la barra final (`Path + "/"`), aunque `Location.Path` se guarda sin ella.
  Sin esto, listar `gs://$B/data/` con el prefijo crudo `"data"` también
  matchearía un hipotético `database/…` (prefijo de *string*, no de ruta). `$B` no
  tiene ese choque hoy así que ningún VC lo habría detectado, pero es la lectura
  correcta de FR-11 y evita un bug silencioso. Detalle en `DECISIONS.md` D-06.

## Dificultades

- **Falso positivo de anclas/`\b` en líneas largas (BR-8).** El primer diseño para
  buscar en una línea de más de 1 MiB consideró matchear primero contra el prefijo
  retenido y solo si no matcheaba, seguir por streaming. Se descartó antes de
  escribir código de producción: un patrón con `$` o `\b` puede dar un falso
  positivo contra el prefijo aunque la línea real termine distinto. Se implementó
  siempre por streaming para líneas largas, y se agregó un test adversarial
  (`TestScanEndAnchorPastLimit`) que efectivamente falla si se vuelve a intentar
  ese atajo.
- **`bufio.NewReader` como `io.RuneReader` pierde bytes.** Envolver la fuente de
  streaming con `bufio.NewReader(...)` para obtener un `io.RuneReader` parecía la
  forma obvia de usar `regexp.MatchReader`, pero su buffer de lectura adelantada
  (4 KiB) consume del stream más de lo que `MatchReader` termina necesitando, y esos
  bytes quedan atrapados adentro, invisibles para el drenado posterior que la spec
  exige ("siempre se consume el resto"). Se detectó por inspección antes de que
  causara una falla real, y se resolvió con un `io.RuneReader` propio que decodifica
  exactamente los bytes que necesita cada rune (D-10).
- Nada más se atascó: el resto del desarrollo fue directo a partir de la spec ya
  revisada (sin preguntas abiertas) y el plan (con las notas de implementación ya
  resueltas de antemano).

## Handoff a la Iteración 2

- Servidor de prueba (`internal/fakegcs`) y los 4 VCs con evidencia parcial: sus
  huecos exactos están en `CONTEXT.md`.
- **BR-7 / transcoding**: la Iteración 2 tiene que confirmar que el cliente no pida
  `Accept-Encoding: gzip` (la nota "A resolver primero" del plan). No se tocó en
  esta iteración: no hay lectura de gzip todavía (los objetos no-texto se leen como
  bytes, no hay sniffing).
- Módulo `internal/sniff` no existe todavía; `-c`/`-l` tampoco (rechazados como
  `unknown flag`).
- `gcsgrep-cobertura-vc.md` queda con 18 ✅, 4 🔸 y 48 ⬜.
