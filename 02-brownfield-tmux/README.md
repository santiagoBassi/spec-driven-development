# TP2 — `tmux` con SSH nativo (brownfield)

Análisis y spec de un comando `ssh-pane` que abre un pane con una sesión SSH remota
**sin ejecutar el binario `ssh`**, usando `libssh`, y que existe **solo en Linux**.

**No hay implementación**: el entregable es el descubrimiento y la spec.

## Qué leer, en orden

| # | Archivo | Qué tiene |
|---|---|---|
| 1 | [`sdd/notas-exploracion.md`](./sdd/notas-exploracion.md) | Exploración de solo lectura: el camino del comando al `fork`/`exec`, PTY y event loop, compat y build, módulos tocados, interfaces reusadas, riesgos y la línea de base medida en macOS y Linux |
| 2 | [`sdd/ssh-pane-base-context.md`](./sdd/ssh-pane-base-context.md) | Pedido refinado y las decisiones D-1 a D-16, cada una con lo elegido, el fundamento y lo descartado |
| 3 | [`sdd/ssh-pane-spec.md`](./sdd/ssh-pane-spec.md) | La spec: alcance dentro y fuera por archivo, límite solo-Linux, FRs, BRs, invariantes y NFRs con sus VCs, y el orden de entrega |

## Cómo verificar las referencias

Todas las referencias `archivo:línea` son relativas a `tmux/` en el commit
`94796f6b1182507efac8a272fc309a79e22e58a5`. El checkout no se versiona en este repo:

```bash
git clone https://github.com/tmux/tmux.git
git -C tmux checkout 94796f6
```
