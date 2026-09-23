# Spec-Driven Development (SDD)

> **Primero definís qué debe cumplir el software y cómo vas a demostrarlo; después escribís código.**

SDD (*Spec-Driven Development*) es una forma de trabajar en la que la fuente de
verdad no es el chat ni «lo que el agente entendió», sino una **especificación
escrita y verificable**.

La meta es evitar que un agente entregue algo que «anda», pero que no era realmente
lo pedido.

## Índice

1. [El problema que resuelve SDD](#el-problema-que-resuelve-sdd)
2. [El recorrido completo](#el-recorrido-completo)
3. [Antes del pipeline: base context](#antes-del-pipeline-base-context)
4. [El pipeline SDD, paso a paso](#el-pipeline-sdd-paso-a-paso)
5. [Cómo aplicarlo a `gcsgrep`](#cómo-aplicarlo-a-gcsgrep)
6. [¿Se puede modificar el plan entre iteraciones?](#se-puede-modificar-el-plan-entre-iteraciones)

---

## El problema que resuelve SDD

Cuando se le pide algo a un agente de manera informal suelen aparecer tres fallas:

| Falla | Qué pasa | Ejemplo |
|---|---|---|
| **Ambigüedad** | El agente completa decisiones que nadie tomó. | «Haceme un login»: ¿OAuth?, ¿recuperación de contraseña?, ¿cuánto dura la sesión? |
| **Drift** | En un chat largo, las decisiones anteriores se pierden o se reinterpretan. | Una restricción acordada al comienzo desaparece después de muchas vueltas. |
| **Salida no verificable** | Se usan cualidades que no se pueden comprobar objetivamente. | «Que sea rápido» o «que maneje bien los errores». |

SDD combate esto usando documentos como **memoria persistente** y como **contrato**.

## El recorrido completo

```text
Idea vaga
  ↓
Base context
  ↓
Spec ↔ Revisión
  ↓
Plan por iteraciones
  ↓
Implementar una iteración
  ↓
Verificar contra los VCs
  ↓
Freeze / cápsula de contexto
```

---

## Antes del pipeline: base context

El *base context* todavía **no es la spec**. Es la etapa para pensar, explorar
opciones y tomar decisiones antes de comprometerse con requisitos formales.

Acá se permite divergir: hacer preguntas, comparar alternativas y cambiar de
opinión. El agente puede ayudar a pensar, pero las decisiones son del equipo.

### 1. Requerimientos refinados

- ¿Qué problema resuelve?
- ¿Para quién?
- ¿Qué entra y qué queda fuera?
- ¿Qué restricciones existen?
- ¿Qué pasa ante entradas inválidas o estados límite?

### 2. Diseño

- ¿Qué opciones hay?
- ¿Cuáles se descartan y por qué?
- ¿Qué *trade-offs* se aceptan?

### 3. Arquitectura

- Componentes principales.
- Flujo de datos.
- Límites entre componentes.
- Riesgos conocidos que se decide aceptar.

> En `gcsgrep`, por ejemplo, las decisiones sobre regex, credenciales, binarios,
> archivos `.gz` y límites de costo pertenecen primero al *base context*. No
> conviene dejarlas para «cuando programemos».

---

## El pipeline SDD, paso a paso

### 1. Especificar

**Objetivo:** transformar el *base context* en un contrato construible y comprobable.

Una spec incluye:

| Elemento | Pregunta que responde |
|---|---|
| **Propósito** | ¿Por qué existe el producto? |
| **Alcance** | ¿Qué entra y qué queda explícitamente fuera? |
| **Actores** | ¿Quién interactúa con el sistema? |
| **FR** (*Functional Requirements*) | ¿Qué debe hacer? |
| **BR** (*Business Rules*) | ¿Qué políticas o restricciones aplican? |
| **NFR** (*Non-Functional Requirements*) | ¿Con qué cualidad debe hacerlo? |
| **VC** (*Verification Criteria*) | ¿Cómo se demuestra cada requisito? |

#### FR — Requisito funcional

Describe un comportamiento observable. Conviene escribirlo así:

> **Dado** `[estado inicial]`,  
> **Cuando** `[acción]`,  
> **Entonces** `[resultado observable]`.

**Ejemplo abstracto**

> Dado un objeto de texto que contiene una coincidencia,  
> cuando se ejecuta una búsqueda válida,  
> entonces se imprime la URI del objeto, la línea y su número.

Un FR debe ser **atómico**: no mezclar varios comportamientos grandes en una misma
línea.

#### BR — Regla de negocio

Una BR es una política del sistema, con su fundamento y sus excepciones.

> **BR:** La herramienta nunca escribe objetos ni modifica permisos en GCS.  
> **Fundamento:** debe respetar el alcance de una herramienta de lectura.  
> **Excepciones:** ninguna.

#### NFR — Requisito no funcional

No expresa qué hace el producto, sino **con qué cualidad** lo hace: rendimiento,
memoria, seguridad, disponibilidad, etc.

Un NFR debe incluir:

```text
métrica + umbral + condición de carga
```

| No verificable | Verificable |
|---|---|
| «La herramienta debe ser rápida». | «Con objetos de hasta cierto tamaño, el proceso no supera cierto uso de memoria». |

El número exacto se decide **antes** de implementar: no se mide después para adaptar
el requisito a lo que dio.

#### VC — Criterio de verificación

Un VC es la prueba observable de un requisito. No dice «hay que probarlo»; dice qué
se ejecuta y qué debe verse.

> **VC:** Dado un objeto de prueba con dos *matches* conocidos, el comando:
>
> - sale con código `0`;
> - imprime dos líneas;
> - cada línea contiene la URI y el número de línea esperados.

La propiedad importante es una trazabilidad uno a uno:

```text
Cada FR, BR y NFR tiene un VC.
Cada VC corresponde a un requisito.
```

Si no se puede comprobar un requisito, todavía no está bien especificado.

---

### 2. Revisar

La revisión es un **gate**: decide si la spec está lista para planificar.

No se debería planificar si quedan:

- Preguntas abiertas o `TBD`.
- Términos subjetivos como «bien», «rápido» o «elegante».
- Requisitos sin VC.
- Solo caminos felices.
- NFR sin métrica y umbral.
- Alcance ambiguo.

```text
Spec con dudas
  ↓
Revisión falla
  ↓
Volvés a especificar
  ↓
Revisión pasa
  ↓
Recién planificás
```

Si la revisión encuentra huecos, se vuelve a **Especificar**. No se los manda al
plan ni se los deja para implementación. Es el único paso formal que vuelve hacia
atrás.

---

### 3. Planificar

**Objetivo:** partir una spec aprobada en iteraciones pequeñas, funcionales y
verificables.

Un plan no es una lista de archivos para crear. Cada iteración entrega un incremento
útil que se puede ejecutar de punta a punta.

| Big bang | Iterativo |
|---|---|
| Mucho código → un chequeo final → el error aparece tarde. | Incremento chico → verificación → siguiente incremento. |

Un buen plan:

- Ordena por dependencias, no por entusiasmo.
- Tiene una Iteración 1 que ya funciona de punta a punta.
- Define el alcance y las exclusiones de cada iteración.
- Indica qué VCs deben pasar.
- No inicia la siguiente iteración hasta que la anterior pasa su *gate*.
- Deja documentado el alcance diferido.

Por ejemplo, no hace falta soportar todos los flags de `grep` en la primera
iteración. El plan documenta qué se posterga y por qué.

#### Memoria persistente entre iteraciones

| Archivo | Rol | Cómo evoluciona |
|---|---|---|
| `CONTEXT.md` | Estado actual, mapa de archivos, interfaces y comandos útiles. | Se actualiza. |
| `DECISIONS.md` | Decisiones no obvias, alternativas descartadas y consecuencias. | Solo se agrega; no se reabre lo decidido sin motivo. |
| `iterations/NN-*.md` | Registro de comandos, resultados, desvíos y *handoff* de cada iteración. | Es inmutable una vez terminada la iteración. |

Así, una persona o un agente nuevo puede retomar el proyecto sin depender de un chat
viejo.

---

### 4. Implementar

Se implementa **una iteración por vez**, respetando la spec y el plan.

El ciclo de un agente de código es:

```text
Leer → Planificar el próximo paso → Actuar → Observar → repetir
```

| Etapa | Qué implica |
|---|---|
| **Leer** | Pedido, spec, plan, reglas del proyecto, código existente y salidas previas. |
| **Planificar** | Decidir el siguiente paso concreto; no imaginar toda la solución de una vez. |
| **Actuar** | Editar, ejecutar comandos o pruebas. |
| **Observar** | Leer fallas, tests, logs y resultados para corregir. |

El único paso que cambia el mundo es **Actuar**; por eso importa tanto la calidad de
lo que se leyó antes.

Al cerrar una iteración no basta con que el código compile. Deben pasar sus criterios
de éxito, actualizarse los artefactos de contexto y quedar el repositorio en un
estado sano.

---

### 5. Verificar

**Objetivo:** reunir evidencia de que se cumple cada VC.

No alcanza con decir «lo probé manualmente». Una cobertura útil traza cada VC a un
requisito y a evidencia concreta:

| VC | Requisito | Cómo se ejercita | Qué se observa | Estado |
|---|---|---|---|---|
| `VC-1` | `FR-1` | Comando o test concreto. | Exit code, salida, archivo o métrica. | Pasa / falla |

La evidencia debe ser reproducible:

- Comando o test nombrado.
- Datos de prueba conocidos.
- Resultado observable.
- Medición concreta cuando aplica.

Además de los caminos felices, se deben cubrir:

- Errores.
- Casos límite.
- Invariantes, por ejemplo: «una falla no modifica datos».
- NFR, como rendimiento o memoria.

Verificar no es solamente correr tests: es poder rastrear cada requisito hasta una
evidencia.

---

### 6. Freeze

Al final se destila el aprendizaje y el estado útil en una **cápsula de contexto**.

Debe dejar claro:

- Qué se construyó.
- Qué decisiones quedaron tomadas.
- Qué limitaciones siguen existiendo.
- Qué se verificó.
- Cómo retomar o extender el trabajo después.

Esto evita que conocimiento importante quede perdido en conversaciones, en la
memoria de una persona o en el contexto temporal de un agente.

---

## Cómo aplicarlo a `gcsgrep`

El recorrido recomendado es:

1. Refinar el borrador y responder las preguntas abiertas.
2. Escribir la spec con FR, BR, NFR y VC.
3. Revisarla hasta que no queden ambigüedades.
4. Crear el plan de iteraciones.
5. Implementar solamente la Iteración 1.
6. Armar evidencia de cobertura para cada VC de esa iteración.

> **La spec define qué debe cumplir; los VC demuestran que lo cumple; el plan
> controla cuánto se construye por vez.**

---

## ¿Se puede modificar el plan entre iteraciones?

**Sí.** El plan puede —y debería— ajustarse entre iteraciones cuando aparece
información nueva. No es un documento congelado como la evidencia de una iteración
ya terminada.

La distinción importante es **qué cambió**:

| Tipo de cambio | Ejemplo | Qué hacer |
|---|---|---|
| Cambió el **cómo** o el orden técnico. | Hace falta un módulo compartido antes de implementar concurrencia. | Actualizar las iteraciones futuras del plan. |
| Cambió el **qué** del producto. | No estaba definido el exit code ante un objeto ilegible, o ahora se quiere soportar `.gz`. | Volver a la spec, modificar o resolver el requisito, revisarla y recién entonces actualizar el plan. |

```text
Termina Iteración 1
  ↓
¿Sus VCs pasaron?
  ├─ No → Corregir dentro de la Iteración 1
  └─ Sí → Registrar lo aprendido
             ↓
       ¿Afecta solo el plan técnico?
         ├─ Sí → Actualizar el plan futuro
         └─ No, afecta requisitos o VCs
                  ↓
                Volver a Spec → Review → actualizar el plan
```

No se debería modificar retroactivamente el registro de una iteración terminada:
ahí conviene anotar qué se planeó, qué ocurrió y qué impacto tiene. En cambio,
`CONTEXT.md`, `DECISIONS.md` y las iteraciones futuras sí deben reflejar el estado
nuevo.

> **Regla simple:** si el cambio podría hacer que alguien diga «entonces ya no
> estamos construyendo lo que prometía la spec», no es un ajuste de plan; es una
> revisión de spec.
