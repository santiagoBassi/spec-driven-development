---
name: supersede-adr
description: Usar cuando una decisión que ya tiene un ADR aceptado cambia o se revierte — "ya no usamos X", "cambiemos la decisión del ADR-0007", "esto contradice lo que habíamos decidido". Crea un ADR nuevo que reemplaza al anterior y marca el viejo como reemplazado, sin reescribirlo.
---

# supersede-adr

Las decisiones cambian. Los ADR no. Cuando una decisión aceptada deja de valer, se
escribe **un ADR nuevo** que la reemplaza, y el viejo queda intacto salvo por una
línea: su estado.

Así la historia se puede leer: por qué se eligió X en marzo, y por qué en
septiembre se pasó a Y. Si se edita el ADR viejo, esa historia desaparece.

## Pasos

1. **Identificá el ADR que cambia.** Si no sabés cuál es, lanzá el subagent
   `adr-explorer` con la decisión nueva.

2. **Escribí el ADR nuevo con `write-adr`**, con dos agregados:
   - En **Contexto**, la primera línea: `Reemplaza a ADR-NNNN.` y qué cambió desde
     entonces — un dato nuevo, una restricción que apareció, un supuesto que resultó
     falso.
   - En **Opciones consideradas**, la decisión vieja figura como una opción más,
     con su en contra *actual*.

3. **Cambiá solo la línea de estado del ADR viejo**:

   ```md
   Estado: Reemplazado por ADR-MMMM
   ```

   Nada más. Ni el contexto, ni las consecuencias, ni "una aclaración". El hook
   `adr-immutable` deja pasar exactamente ese cambio y bloquea cualquier otro.

## Anti-patrones

- **Editar el ADR viejo "para que quede al día".** El hook lo va a bloquear, y con
  razón.
- **Borrar el ADR viejo.** Un ADR reemplazado sigue siendo historia.
- **ADR nuevo sin `Reemplaza a`.** Quedan dos decisiones aceptadas que se
  contradicen, y nadie sabe cuál vale.
- **Reemplazar sin decir qué cambió.** Si el contexto es el mismo que el del ADR
  viejo, ¿por qué la decisión es distinta?
