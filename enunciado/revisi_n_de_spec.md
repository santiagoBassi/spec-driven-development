#### Revisión de spec: gcsgrep — spec (sdd/gcsgrep-spec.md)

* Equipo: grupo2 - Hash: `8b63b8de7f402f5a0419039d4a89d3de28a2311a`
* Criterio: correccion-de-specs v1.1 · Corrector/a: Claude (agente corrector, a pedido de la cátedra) · Fecha: 2026-09-24
* Base context: `sdd/gcsgrep-base-context.md` (enlazado en `sdd/gcsgrep-spec.md:20`)
* M1 FR: 38 BR: 9 NFR: 3 VC: 50

#### Resumen

Es una spec muy rigurosa. Los 38 FRs están en Dado/Cuando/Entonces, hay un VC por requisito con exit code y texto literal, los cuatro caminos de falla obligatorios están cubiertos, los tres NFRs tienen número, condición de carga y un VC que los mide, y las 10 preguntas del borrador están decididas. M3, M4 y M6 no dan hits reales, y M5 no cae dentro de ningún FR. El único bloqueo es la **atomicidad**: al menos nueve FRs ponen alternativas en el Dado o el Cuando ("bucket o prefijo", "-c con -l, o -n con…"), o tienen en el Entonces resultados que no se observan en una sola ejecución. Por lo demás hay huecos chicos: dos bordes sin comportamiento definido (última línea sin \n, corte de red a mitad de una lectura), orígenes en la tabla de trazabilidad que no existen en el borrador, una pregunta (la 10) sin fundamento, y tres capacidades por encima de la v1 sugerida (-c, -l, --concurrency) sin un porqué explícito.

#### Hallazgos por dimensión

#### 1. Propósito y alcance — PASS

Sin hallazgos. El propósito entra en una oración y dice por qué existe la herramienta (`sdd/gcsgrep-spec.md:31-33`), el fuera de alcance tiene más de diez ítems concretos (`sdd/gcsgrep-spec.md:54-69`), los actores incluyen Script, GCS y Credenciales ADC (`sdd/gcsgrep-spec.md:80-85`), M6 da vacío, cada flag de la v1 tiene su FR o BR y el base context está enlazado.

#### 2. Completitud y consistencia — FAIL

* **Issue (2.3)** · Hay 9 FRs no atómicos según B6 (con 3 o más ya es Issue):
  * Alternativas en Dado/Cuando:
    * FR-13 (`sdd/gcsgrep-spec.md:306-307`): "no hay un objeto con ese nombre exacto, o el bucket no existe".
    * FR-15 (`sdd/gcsgrep-spec.md:337`): "Dado un bucket existente o un prefijo que no contiene ningún objeto".
    * FR-16 (`sdd/gcsgrep-spec.md:347`): "una ubicación de bucket o de prefijo cuyo bucket no existe".
    * FR-20 (`sdd/gcsgrep-spec.md:416`): "combina -c con -l, o -n con -c o con -l".
    * FR-24 (`sdd/gcsgrep-spec.md:473-474`): "combina flags cortos en un solo argumento … o pega el valor con =".
    * FR-30 (`sdd/gcsgrep-spec.md:568-569`): "GCS falla al listar la ubicación, o al consultar la metadata del objeto puntual".
    * FR-33 (`sdd/gcsgrep-spec.md:611-612`): "una página del listado, o la consulta del objeto puntual".
  * Alternativas en el Cuando y además un Entonces que no se observa en una sola ejecución:
    * FR-26 (`sdd/gcsgrep-spec.md:502-510`): "con `--concurrency <N>`, o sin el flag". El Entonces suma el tope de lecturas, el error por valor fuera de rango y el error por flag sin valor.
  * Entonces que no se observa en una sola ejecución:
    * FR-2 (`sdd/gcsgrep-spec.md:166-169`): el Cuando es "sin -c", pero el Entonces agrega "Con -c el exit también es 1, pero stdout lista los conteos en 0".
* **Warning (2.5)** · `sdd/gcsgrep-spec.md:962-967, 972, 990`. La columna *Origen* cita FR-h (FR-18 a FR-20), FR-i (FR-21 a FR-23), FR-j (FR-28) y BR-e (BR-8), que no existen en el borrador (`gcsgrep-requirements.md:50-77` solo define FR-a a FR-g y BR-a a BR-d), ni en el base context. Son referencias a IDs inexistentes. Ningún FR ni BR queda sin VC por esto, así que es Warning.

#### 3. Casos borde y verificabilidad — WARN

* 3.5 · FR-1: se pudo escribir el test sin decidir nada. Los datos salen de `test-fixtures/logs/app/` subidos a `$B` (`sdd/gcsgrep-spec.md:100-103`), el comando y las tres líneas exactas de stdout, en cualquier orden, salen de VC-1 (`sdd/gcsgrep-spec.md:157-161`), el exit es 0 y stderr queda vacío fuera de una TTY por FR-38 (`sdd/gcsgrep-spec.md:684-685`). FR-29 (objeto ilegible): se pudo escribir el test sin decidir nada. La spec define el servidor de prueba con falla 500 inyectada y ADC válidas (`sdd/gcsgrep-spec.md:119-127`), y fija stdout exacto, el prefijo de stderr `gcsgrep: gs://fake/b.log: read error:  ` y el exit 2 (`sdd/gcsgrep-spec.md:560-564`). El caso de una falla a mitad de lectura queda abierto (ver 3.4).
* **Warning (3.4)** · Última línea sin \n: ningún FR ni BR define si una última línea sin terminador se busca y se imprime, ni cómo sale, y no hay VC que la ejercite. Todos los fixtures de texto terminan en `0a`. Tendría que estar junto a FR-9 (`sdd/gcsgrep-spec.md:253-258`).
* **Warning (3.4)** · Corte de red a mitad de una lectura: FR-29 (`sdd/gcsgrep-spec.md:554-558`) dice "emite completa la salida de los demás", pero no dice qué pasa con las líneas del objeto que falla que ya se imprimieron antes del corte (¿quedan en stdout o no?). Los VCs fallan al empezar la lectura: VC-29 responde "500" (`sdd/gcsgrep-spec.md:561`) y en VC-32 "la lectura queda colgada" (`sdd/gcsgrep-spec.md:603`). Ninguno corta el cuerpo después de un match.

#### 4. Requerimientos no funcionales — PASS

Sin hallazgos. NFR-1 (`sdd/gcsgrep-spec.md:885-899`), NFR-2 (`sdd/gcsgrep-spec.md:914-922`) y NFR-3 (`sdd/gcsgrep-spec.md:932-934`) tienen métrica, número y condición de carga. El umbral de memoria ($\le 100\text{ MiB}$) queda por debajo del objeto de 1 GB, y la política de red es explícita: 0 reintentos (FR-31), 30 s de inactividad (FR-32/33) y exit 2. VC-48, VC-49 y VC-50 repiten umbral y condición y dicen con qué se miden. Ningún umbral está marcado como provisorio.

#### 5. Tecnología y fundamento — WARN

* **Warning (5.3)** · Pregunta 10 (objeto que cambia durante la lectura). Se decide en `sdd/gcsgrep-spec.md:66-67`: "Se lee lo que haya al abrir el stream, sin fijar ni verificar la generación del objeto". En ningún lado se dice por qué esa opción y no otra (fijar la generación, abortar o avisar): `sdd/gcsgrep-base-context.md:73` dice "no se controla en la v1", `sdd/gcsgrep-base-context.md:275-276` lo lista como riesgo aceptado y `sdd/gcsgrep-plan.md:433` dice "Riesgo conocido, aceptado". La decisión se registra pero sin fundamento. A la pregunta 4 también le falta el porqué de incluir -c, -l y --concurrency. Eso se registra en 6.1.
* **Warning (5.6)** · `sdd/DECISIONS.md:86` — "`--max` acepta solo dígitos (+5 y 5 no)". Es comportamiento observable que la spec no fija: BR-4 dice "`--max <N>` con N entero $\ge 1$" (`sdd/gcsgrep-spec.md:755`), y +5 se puede leer como entero $\ge 1$. El mismo hueco aplica a `--concurrency` ("N debe ser un entero entre 1 y 32", `sdd/gcsgrep-spec.md:504`). Es un hueco del mismo tipo que el que D-18 ya cerró volviendo a la spec.

#### 6. Simplicidad — WARN

* **Warning (6.1)** · -c (FR-18, `sdd/gcsgrep-spec.md:381-388`): está por encima de la v1 sugerida y no tiene fundamento. `sdd/gcsgrep-base-context.md:39-42` y `121-123` lo listan y describen su formato, pero no dicen por qué entra en la v1 y no va al plan.
* **Warning (6.1)** · -l (FR-19, `sdd/gcsgrep-spec.md:398-403`): mismo caso que -c.
* **Warning (6.1)** · `--concurrency` configurable (FR-26, `sdd/gcsgrep-spec.md:499-510`). La concurrencia está fundamentada (`sdd/gcsgrep-base-context.md:186-189`: "leer de a un objeto es lento"), pero que sea configurable por flag, entre 1 y 32, no tiene un porqué.
* **Suggestion (6.3)** · FR-24, FR-25 y FR-27 (`sdd/gcsgrep-spec.md:470-535`) endurecen el parser: rechazan flags combinados, flags repetidos y reportan un solo error. Se podrían diferir sin afectar el propósito de la herramienta.
* **Suggestion (6.4)** · La misma regla está escrita dos veces: el exit 1 con -c y todos los conteos en 0 aparece en FR-2 (`sdd/gcsgrep-spec.md:168-169`) y en FR-18 (`sdd/gcsgrep-spec.md:387-388`).

#### Veredicto general

**NEEDS WORK**

Falla solo la dimensión 2, por atomicidad: hay nueve FRs con alternativas en Dado/Cuando o con un Entonces que necesita más de una ejecución. Dos agentes podrían partir o verificar esos FRs de forma distinta. La dimensión 3 no falla (hay VCs para todo), así que alcanza con partir esos FRs (manteniendo la regla de un VC por requisito) y volver a revisar antes de seguir planificando.

#### Estado en la consigna

| VE | Issues | Warnings | Estado |
| :--- | :--- | :--- | :--- |
| VE-1 | 1 | 4 | Parcial |
| VE-2 | 0 | 2 | Se cumple |
| VE-3 | 0 | 2 | Se cumple |

#### Acciones (priorizadas)

1. **[MUST]** Partir en FRs atómicos, cada uno con su VC: FR-2 (sacar el caso -c del Entonces), FR-13, FR-15, FR-16, FR-20, FR-24, FR-26, FR-30 y FR-33 (`sdd/gcsgrep-spec.md:163-622`), uno por alternativa del Dado/Cuando.
2. **[SHOULD]** Corregir la columna *Origen* de la tabla de trazabilidad (`sdd/gcsgrep-spec.md:962-990`): reemplazar FR-h, FR-i, FR-j y BR-e por el origen real (una pregunta, el gate de revisión) o definir esos IDs en el base context.
3. **[SHOULD]** Agregar un FR (o extender FR-9, `sdd/gcsgrep-spec.md:253`) que defina cómo se busca e imprime una última línea sin \n, con un fixture que termine sin 0a y su VC.
4. **[SHOULD]** Definir en FR-29 (`sdd/gcsgrep-spec.md:554`) qué pasa con las líneas ya emitidas de un objeto cuya lectura se corta a mitad, y agregar un VC con el servidor de prueba que corte el cuerpo después de un match.
5. **[SHOULD]** Agregar el fundamento de la pregunta 10 (`sdd/gcsgrep-spec.md:66` o `sdd/gcsgrep-base-context.md:73`): por qué se lee sin fijar la generación, y no se aborta ni se avisa.
6. **[SHOULD]** Fijar en BR-4 (`sdd/gcsgrep-spec.md:755`) y en FR-26 (`sdd/gcsgrep-spec.md:504`) que N se escribe solo con dígitos (+5 es inválido), como ya decidió D-08, y agregar el caso a VC-42 y VC-26.
7. **[SHOULD]** Fundamentar en el base context (`sdd/gcsgrep-base-context.md:39-42`) por qué -c entra en la v1, o moverlo al plan.
8. **[SHOULD]** Fundamentar en el base context (`sdd/gcsgrep-base-context.md:39-42`) por qué -l entra en la v1, o moverlo al plan.
9. **[SHOULD]** Fundamentar en `sdd/gcsgrep-base-context.md:180-189` por qué la concurrencia es configurable por flag (entre 1 y 32), o dejarla fija en 4 y mover `--concurrency` al plan.
10. **[COULD]** Evaluar diferir al plan FR-24, FR-25 y FR-27 (`sdd/gcsgrep-spec.md:470-535`), que endurecen el parser.
11. **[COULD]** Dejar la regla del exit con -c en un solo lugar (FR-18, `sdd/gcsgrep-spec.md:387`) y sacarla de FR-2 (`sdd/gcsgrep-spec.md:168-169`).

---

#### Diseño en el base context · Grupo 2

**Archivo revisado:** `sdd/gcsgrep-base-context.md` (la spec lo enlaza en `sdd/gcsgrep-spec.md:20`)

**Veredicto: tiene diseño, completo y con título propio.** Es la entrega que más se acerca a lo que se pide.

#### Qué hay

* El documento tiene las tres facetas separadas: **§1 Requerimientos refinados** (:25), **§2 Notas de diseño** (:77) y **§3 Esquema de arquitectura** (:221).
* §2 cubre 11 temas: interpretación del patrón, autenticación, ubicación, salida, exit codes, binarios y .gz, guardrail de costo, concurrencia, progreso, fallos de red y stack. Cada uno tiene el formato **Elegido / Fundamento**.
* **Opciones descartadas con motivo:** hay tablas Opción | Por qué no para el patrón (:87), binarios y .gz (:158) y el guardrail (:172). En ubicación el descarte está en prosa (:116).
* **Trade-offs explícitos:** exit 2 con un resultado parcial (:138), el tope por objetos y no por bytes, y el entrelazado de la salida con concurrencia.
* **Lo diferido queda anotado** (key file, reintentos, límite por bytes), así que no se pierde.

#### Qué se puede mejorar

* Auth, salida, exit codes, concurrencia, progreso, fallos de red y stack tienen fundamento, pero no una alternativa descartada escrita. Por ejemplo, para la concurrencia no se anota "salida agrupada por objeto" como descartada, aunque el fundamento la discute. Una tabla Opción | Por qué no en esos temas deja el diseño parejo.