# ADR-0001: Guardar los turnos en PostgreSQL

Estado: Aceptado
Fecha: 2026-03-12

## Contexto
`turnos-api` reserva turnos médicos. Dos pacientes no pueden quedarse con el mismo
turno, aunque reserven en el mismo segundo. Hoy los turnos viven en un JSON en
disco, que no soporta escrituras concurrentes. El equipo opera una sola base en
producción y no tiene experiencia con bases distribuidas.

## Opciones consideradas
1. **PostgreSQL** — a favor: transacciones y constraints `UNIQUE` resuelven la doble
   reserva en la base; el equipo ya lo opera. · en contra: un servicio más que
   mantener y migrar.
2. **MongoDB** — a favor: esquema flexible para los datos de cada especialidad. · en
   contra: la unicidad entre documentos exige transacciones multi-documento que el
   equipo no conoce; nadie lo opera hoy.
3. **Seguir con el JSON y un lock de archivo** — a favor: cero infraestructura nueva.
   · en contra: un solo proceso escribiendo; no escala a dos réplicas.

## Decisión
Usamos PostgreSQL, porque la doble reserva se resuelve con una constraint `UNIQUE
(profesional_id, inicio)` y una transacción, sin lógica de locking en la aplicación.

## Consecuencias
- La unicidad del turno la garantiza la base, no el código.
- Hace falta una herramienta de migraciones y un proceso para correrlas.
- Los datos variables por especialidad van a una columna `jsonb`, con menos
  validación que un esquema propio.
