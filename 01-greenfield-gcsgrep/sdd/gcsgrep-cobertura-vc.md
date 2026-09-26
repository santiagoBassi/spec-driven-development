# gcsgrep — tabla de cobertura de VCs

> Salida del paso **Verificar**. Para cada VC dice **con qué se lo ejercita y qué se
> observó**. Un VC sin evidencia es un VC que no pasó.
>
> Estados: ✅ pasa · 🔸 implementado, VC pendiente (evidencia parcial; el resto necesita
> el servidor de prueba de la Iteración 2) · ⬜ no empezado.
>
> Se completa a medida que pasan los VCs, según el [plan](./gcsgrep-plan.md).

## Resumen

Sin iteraciones cerradas: todavía no hay evidencia. Los conteos se completan al cerrar la
Iteración 1.

## Cobertura, uno por uno

Una fila por VC, con este formato en cada iteración:

| VC | Requerimiento | Ejercitado por | Se observa | Estado |
|---|---|---|---|---|

## Evidencia que se ejecutó

Comandos y salida de cada corrida, con fecha, commit de partida y toolchain. Sin corridas
todavía.
