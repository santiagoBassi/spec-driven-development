# Notas de exploración — SSH nativo en un pane de tmux

> **Fase SDD: Descubrir.** Solo se leyó el repo; no hay implementación ni spec.
> Base: `tmux/` en el commit `94796f6b1182507efac8a272fc309a79e22e58a5`.
> Todas las referencias `archivo:línea` apuntan a ese checkout y son relativas a `tmux/`.

## Qué buscamos

Un comando que abra una sesión remota en un pane **sin invocar el binario `ssh`**.
La capacidad debe compilarse **solo en Linux**, sin romper los builds de los
demás sistemas. El descubrimiento se concentró en el recorrido de `split-window`
y en las interfaces de las que depende un pane interactivo.

## El mapa en un minuto

```text
cmd.c                     registra el comando
  ↓
cmd-split-window.c        elige pane, layout y argumentos
  ↓
spawn.c                   crea pane y PTY; hace fork
  ├─ hijo                 ejecuta el comando o la shell
  └─ servidor             conecta el PTY al event loop
       ↓
window.c / input.c        teclas hacia el PTY; salida hacia la pantalla
```

Referencias del recorrido: `cmd.c:123-215`; `cmd-split-window.c:58-76,175-208`;
`spawn.c:307-372,478-590`; `window.c:1585-1638,2024-2059`.

## Hallazgos

### 1. La entrada del comando y la creación del pane son cosas distintas

`cmd.c:123-215` registra las entradas y `cmd_find` las busca por nombre
(`cmd.c:460-492`). `new-pane` y `split-window` tienen entradas separadas, pero
ambas usan `cmd_split_window_exec` (`cmd-split-window.c:39-76`). Cada entrada
declara nombre, argumentos, target y ejecutor (`tmux.h:2055-2075`).

El ejecutor resuelve el pane objetivo, reserva una celda del layout y llena
`spawn_context` antes de llamar a `spawn_pane`
(`cmd-split-window.c:79-98,175-208`; `tmux.h:2499-2532`). Luego actualiza la
selección, redibuja y dispara el hook `after-split-window`
(`cmd-split-window.c:288-319`). Ese es el comportamiento observable que sirve
de referencia para un comando nuevo.

### 2. `spawn_pane` crea un proceso real, no solo una pantalla

`spawn_pane` crea o reutiliza el `window_pane`, guarda entorno y argumentos y
calcula el tamaño inicial (`spawn.c:307-452`). `fdforkpty` crea un hijo y un PTY
conectado al pane (`spawn.c:454-490`). En el hijo, tmux limpia señales y
descriptores, instala el entorno y termina en `execvp`, `$SHELL -c` o una shell
de login, según la cantidad de argumentos (`spawn.c:539-575`).

**Consecuencia:** pasar `ssh` como `shell-command` ejecutaría un programa
externo; no cumpliría el pedido. Además, `spawn_pane` es compartido por
`split-window`, `spawn_window`, `respawn-pane`, popups y el editor interno
(`cmd-split-window.c:208`; `spawn.c:208-209,772`;
`cmd-respawn-pane.c:83`; `cmd-display-menu.c:388`). Cualquier bifurcación
para SSH debe distinguirla de los panes normales.

Tampoco alcanza con pasar host y opciones SSH como `sc.argv`: `spawn_pane`
interpreta esos argumentos como comando local, aplica `default-command` si no
hay argumentos en un pane nuevo y los registra en logs
(`spawn.c:379-390,436-445,546-575`).
La futura interfaz necesita distinguir los datos de conexión del comando que
tmux ejecuta normalmente; también debe decidir qué datos pueden aparecer en
esos logs.

### 3. El PTY ya une teclado, salida y event loop

El pane guarda `pid`, `fd`, `event` y estado del parser (`tmux.h:1361-1390`).
En el servidor, `window_pane_set_event` crea un `bufferevent` para el fd del
PTY (`window.c:1626-1638`). Las teclas y el pegado se escriben allí
(`window.c:2005-2021,2024-2059`; `input-keys.c:396-410`).

En sentido inverso, `window_pane_read_callback` recibe bytes del PTY y los
entrega al parser de pantalla, a `pipe-pane` y a los clientes de control
(`window.c:1585-1611`; `input.c:1027-1056`). El buffer también tiene offsets
que el servidor mantiene (`server-client.c:1897-1915`). Por eso conectar un
socket SSH directamente al parser no reproduce automáticamente toda la E/S
del pane.

### 4. Resize y cierre también dependen del proceso hijo

`window_pane_resize` cambia el tamaño local y encola la actualización
(`window.c:1654-1693`). Más tarde, el servidor aplica `TIOCSWINSZ` al PTY
(`server-client.c:1834-1893`; `window.c:589-615`). Una sesión SSH necesitaría
traducir ese cambio a la terminal remota; el código actual solo actualiza el
PTY local.

Al salir el hijo, `SIGCHLD` lleva a `server_child_exited`, que registra el
estado (`server.c:435-448,466-515`). `window_pane_destroy_ready` espera ese
estado y la salida pendiente (`window.c:486-507`); `server_destroy_pane`
decide si cerrar o conservar el pane según `remain-on-exit` y emite los eventos
de salida (`server-fn.c:354-435`). Una integración que no tenga hijo tendría
que resolver explícitamente ese ciclo de vida.

### 5. El pane vacío parece un atajo, pero no sirve solo

`new-pane -E` y `split-window -E` usan `SPAWN_EMPTY`; no hacen `fork`
(`cmd-split-window.c:149-161`; `spawn.c:458-465`). El pane nace con
`fd = -1` (`window.c:1388-1405`). Con ese fd, tmux descarta teclado y pegado
y omite el procesamiento normal de resize y buffer
(`window.c:2005-2018,2048-2051`; `server-client.c:1785-1790`).
`tmux.1:4193-4205` describe `-E` para panes que reciben contenido inyectado.
Usarlo como base de SSH interactivo exigiría nuevas rutas de entrada y
mantenimiento.

### 6. Linux ya se detecta; SSH todavía no se compila por separado

`configure.ac:1006-1130` identifica Linux como `PLATFORM=linux` y define la
condición de Automake `IS_LINUX`. `Makefile.am:90-255` lista fuentes comunes,
elige `osdep-@PLATFORM@.c` y muestra otros casos de fuentes condicionales.
El patrón de detección y enlace de una biblioteca está en
`configure.ac:250-303` (libevent).

`IS_LINUX` sirve para condicionar reglas de Automake; no es por sí sola una
macro disponible en C. `cmd.c` tiene las declaraciones `extern` de las
entradas en `cmd.c:30-121` y la tabla en `cmd.c:123-216`. Si el comando SSH
solo se enlaza en Linux, ambas referencias necesitan una guarda C definida
desde `configure`, además de condicionar la fuente y la biblioteca en el
build. `configure.ac:522-527` y `Makefile.am:242-245` muestran un patrón que
combina `AC_DEFINE` y `AM_CONDITIONAL`.

No se encontraron `libssh`, `ssh_session` ni `ssh_channel` en `Makefile.am`,
`configure.ac`, `tmux.h`, `spawn.c` o `cmd.c` (búsqueda literal). Por tanto,
la biblioteca y sus guards serían nuevos. En builds no Linux no deberían
quedar dependencias de enlace ni referencias al símbolo del comando.

`compat/` aporta reemplazos de funciones del sistema: por ejemplo,
`compat/fdforkpty.c:23-33` y su detección en `configure.ac:828-839`.
`osdep-linux.c:29-99` contiene funciones específicas de Linux; Automake
selecciona ese archivo en `Makefile.am:235`. Ninguno de esos dos mecanismos
constituye por sí mismo un cliente SSH.

## Una integración posible, todavía sin decidir

Hay una vía que merece evaluación en la spec: después de `fdforkpty`, hacer
que el **hijo** ejecute código enlazado a una biblioteca SSH en lugar de
llamar a `exec` (`spawn.c:478-575`). Ese hijo podría transportar bytes entre
el PTY local y el canal remoto. Para el servidor, el pane conservaría su
`pid`, su PTY y el `bufferevent` habituales (`tmux.h:1366-1390`;
`window.c:1626-1638`).

Es una hipótesis, no una solución comprobada. Faltaría definir autenticación,
verificación del host, conexión, PTY remoto, propagación de resize, señales,
errores y cierre. Si el hijo funciona como puente de bytes, también tendría
que evitar que el modo de línea del PTY local altere la entrada y transportar
datos en ambos sentidos sin bloquear la interacción. El servidor vuelve de
`spawn_pane` tras el `fork` (`spawn.c:492-501,577-613`), y el ejecutor del
comando continúa con selección, hook y retorno (`cmd-split-window.c:288-332`):
un fallo posterior de conexión o autenticación no sería automáticamente un
error síncrono del comando. La spec debe definir cómo se informa y qué ocurre
con el pane en ese caso. Alojar SSH en el servidor es otra posibilidad, con
cambios más profundos en las rutas de E/S y salida descritas en los hallazgos
3 y 4.

## Módulos a tener presentes en la spec

| Área | Archivos | Por qué importan |
|---|---|---|
| Comando y layout | `cmd.c:123-215`; `cmd-split-window.c:39-76,175-208` | Entrada del comando y preparación del pane. |
| Spawn y estado | `spawn.c:243-613`; `tmux.h:1305-1390,2499-2532` | PTY, hijo, argumentos y estado del pane. |
| E/S y ciclo de vida | `window.c:1585-1638,2024-2059`; `server.c:466-515`; `server-fn.c:354-435` | Teclas, salida, cierre y estado. |
| Límite Linux | `configure.ac:1006-1130`; `Makefile.am:90-255`; `cmd.c:123-215` | Compilación, enlace y registro condicional. |
| Documentación y tests | `tmux.1:4086-4105`; `regress/pane-ops.sh:3-25` | Nueva interfaz y regresión de panes. |

## Riesgos que la spec no debería dejar implícitos

- **Confundir nativo con ejecutar un comando:** las rutas actuales terminan
  en `execvp` o `execl` (`spawn.c:546-575`).
- **Cambiar el spawn de todos los panes:** el mismo `spawn_pane` tiene los
  callers enumerados en el hallazgo 2.
- **Perder entrada, salida o estado de cierre:** `fd == -1` bloquea entrada
  (`window.c:2048-2051`); la salida usa buffers y offsets
  (`window.c:1585-1611`; `server-client.c:1897-1915`); el cierre espera
  `PANE_STATUSREADY` (`server-fn.c:382-435`).
- **Romper otros builds:** las fuentes y la tabla de comandos son comunes
  (`Makefile.am:90-235`; `cmd.c:123-215`).
- **Dejar seguridad SSH sin definir:** el hijo recibe el entorno de sesión
  (`environ.c:251-269`; `spawn.c:407-411`), y el default de
  `update-environment` incluye `SSH_AUTH_SOCK`/`SSH_ASKPASS`
  (`options-table.c:1207-1216`), pero no hay una política SSH existente
  identificada en este recorrido.

## Decisiones pendientes para la spec

1. Nombre, sintaxis y target del comando: ¿split, pane flotante o ambos?
2. Biblioteca SSH y comportamiento de `configure` si falta en Linux.
3. Verificación del host y autenticación: claves, agente, contraseña y
   mensajes interactivos.
4. Shell o comando remoto; PTY remoto, `TERM` y resize.
5. Qué ocurre al desconectar, matar o respawnear el pane, y con
   `remain-on-exit` o la espera `-W` (`cmd-respawn-pane.c:33-99`;
   `cmd-split-window.c:323-330`; `server-fn.c:220-235,354-435`).
6. Cómo se separan destino/opciones SSH de `shell-command`, qué se registra en
   logs y qué ve el usuario ante un fallo de conexión posterior al `fork`.

## Estado de la línea de base

Se inspeccionó el código y se consultó CodeGraph; **no se compiló ni se
ejecutaron tests**. El checkout no tiene `configure` generado ni binario
`tmux`. Para construir desde Git, `README:32-39` indica `sh autogen.sh`,
`./configure` y `make`.

Antes de cerrar la spec, falta medir una línea de base de build y regresión
para poder comparar el cambio futuro. Los tests relevantes incluyen
`regress/pane-ops.sh:3-25`, `regress/new-window-command.sh`,
`regress/new-session-command.sh`,
`regress/respawn-pane-control-lag.sh:3-12` y
`regress/kill-session-process-exit.sh:3-20`. El runner está en
`regress/Makefile:1-44`. Tanto el build como esos tests siguen **sin medir**.
