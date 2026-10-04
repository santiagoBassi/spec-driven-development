# ADR-0002: Enviar las notificaciones por una cola

Estado: Propuesto
Fecha: 2026-09-20

## Contexto
Al reservar un turno se manda un email y un SMS. Hoy se mandan dentro del request de
reserva: si el proveedor de SMS tarda 8 segundos, la reserva tarda 8 segundos, y si
falla, el paciente ve un error aunque el turno quedó guardado.

## Opciones consideradas
1. **Cola (Redis + worker)** — a favor: la reserva responde sin esperar al
   proveedor; los envíos fallidos se reintentan. · en contra: un componente más
   (Redis) y un worker que monitorear.
2. **Enviar en un thread aparte, en el mismo proceso** — a favor: sin
   infraestructura nueva. · en contra: si el proceso se reinicia, se pierden los
   envíos pendientes; sin reintentos.
3. **Dejarlo como está** — a favor: nada que cambiar. · en contra: el problema sigue.

## Decisión
Usamos una cola en Redis con un worker separado, porque es la única opción que
sobrevive a un reinicio y permite reintentar.

## Consecuencias
- La reserva deja de depender de la latencia de los proveedores.
- Una notificación puede llegar con segundos de demora.
- Suma Redis como dependencia de infraestructura, y `redis` como dependencia de
  Python: este ADR es el que habilita ese cambio en `pyproject.toml`.
