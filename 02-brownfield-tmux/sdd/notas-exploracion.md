# Notas de exploración — SSH nativo en un pane de tmux

> **Fase SDD: Descubrir.** Solo lectura sobre el repo: no se editó ningún archivo
> de tmux y todavía no hay spec. El build y los tests de la línea de base se
> corrieron en una copia aparte del mismo commit.
> Base: `tmux/` en el commit `94796f6b1182507efac8a272fc309a79e22e58a5`
> (`git checkout 94796f6` antes de verificar).
> Todas las referencias `archivo:línea` apuntan a ese checkout y son relativas a `tmux/`.
> Los números de línea pueden correrse en commits posteriores; los nombres de
> archivo y de función son lo estable.

## El prompt que las produjo

<!-- TODO: reemplazar por el prompt exacto que se usó, si difiere. -->

```text
Solo explorar, sin editar nada. En tmux quiero un comando que abra un pane con
una sesión SSH remota sin invocar el binario ssh, compilado solo en Linux.
Trazá cómo un pane nuevo lanza su proceso hijo, de la tabla de comandos al
fork/exec, usando split-window como guía. Mostrame cómo aísla tmux el código
de plataforma (compat/, osdep-*.c y los chequeos de configure) y cómo se
integra un fd con el event loop. Devolveme notas con módulos tocados,
interfaces reusadas y riesgos. Cada afirmación con archivo:línea.
```

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
       ↓
proc.c / server.c         un único loop de libevent para todo el servidor
```

Referencias del recorrido: `cmd.c:123-216`; `cmd-split-window.c:58-76,175-208`;
`spawn.c:307-372,478-590`; `window.c:1585-1638,2024-2059`; `proc.c:223-227`;
`server.c:258`.

## Hallazgos

### 1. La entrada del comando y la creación del pane son cosas distintas

`cmd.c:123-216` registra las entradas y `cmd_find` las busca por nombre
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

`spawn_pane` crea o reutiliza el `window_pane`, copia los argumentos al pane,
prepara un entorno transitorio para el hijo y calcula el tamaño inicial
(`spawn.c:307-452,591`). `fdforkpty` crea un hijo y un PTY
conectado al pane (`spawn.c:454-490`). En el hijo, tmux limpia señales y
descriptores, instala el entorno y termina en `execvp`, `$SHELL -c` o una shell
de login, según la cantidad de argumentos (`spawn.c:539-575`):

```c
/* spawn.c:478 */
new_wp->pid = fdforkpty(ptm_fd, &new_wp->fd, new_wp->tty, NULL, &ws);
...
/* spawn.c:550-553 — hijo, varios argumentos */
if (new_wp->argc != 0 && new_wp->argc != 1) {
	argvp = cmd_copy_argv(new_wp->argc, new_wp->argv);
	execvp(argvp[0], argvp);
	_exit(1);
}
```

**Consecuencia:** pasar `ssh` como `shell-command` ejecutaría un programa
externo; no cumpliría el pedido. Además, `spawn_pane` es compartido por
`split-window`, `spawn_window`, `respawn-pane`, popups y el editor interno
(`cmd-split-window.c:208`; `spawn.c:209,772`;
`cmd-respawn-pane.c:83`; `cmd-display-menu.c:499`). Cualquier bifurcación
para SSH debe distinguirla de los panes normales.

Tampoco alcanza con pasar host y opciones SSH como `sc.argv`: `spawn_pane`
interpreta esos argumentos como comando local, aplica `default-command` si no
hay argumentos en un pane nuevo y los registra en logs
(`spawn.c:379-390,436-445,546-575`).
La futura interfaz necesita distinguir los datos de conexión del comando que
tmux ejecuta normalmente; también debe decidir qué datos pueden aparecer en
esos logs.

`spawn_pane` emite `pane-created` tras el `fork`, antes de saber si el hijo
logró conectar y autenticarse (`spawn.c:577-592`). El payload incluye
`pane_command`, derivado de los argumentos guardados o de la shell
(`spawn.c:74-110`). Ese evento no acredita una conexión SSH exitosa y los
datos sensibles de conexión tampoco deberían filtrarse por su payload.

### 3. El PTY ya une teclado, salida y event loop

El pane guarda `pid`, `fd`, `event` y estado del parser (`tmux.h:1361-1390`).
En el servidor, `window_pane_set_event` crea un `bufferevent` para el fd del
PTY (`window.c:1626-1638`). Las teclas y el pegado se escriben allí
(`window.c:2005-2021,2024-2059`; `input-keys.c:398-418`).

En sentido inverso, `window_pane_read_callback` recibe bytes del PTY y los
entrega al parser de pantalla, a `pipe-pane` y a los clientes de control
(`window.c:1585-1611`; `input.c:1027-1056`). El buffer también tiene offsets
que el servidor mantiene (`server-client.c:1897-1915`). Por eso conectar un
socket SSH directamente al parser no reproduce automáticamente toda la E/S
del pane.

**El loop en sí.** Todo el servidor corre en un único loop de libevent:
`server_start` llama a `proc_loop(server_proc, server_loop)` (`server.c:258`),
que itera `event_loop(EVLOOP_ONCE)` (`proc.c:223-227`). No hay hilos. Cualquier
trabajo que bloquee dentro del servidor (resolver DNS, conectar, negociar la
clave, autenticar) congela a todos los clientes y panes a la vez.

**Precedente de fd que no es PTY.** `job.c` ya integra procesos auxiliares al
mismo loop sin PTY: `job_run` (`job.c:72`) crea un `socketpair`
(`job.c:118`) y lo envuelve en un `bufferevent` (`job.c:225`). Es el patrón a
mirar si la spec evalúa alojar un socket en el servidor en vez de un pane con PTY.

### 4. Resize y cierre también dependen del proceso hijo

`window_pane_resize` cambia el tamaño local y encola la actualización
(`window.c:1654-1693`). Más tarde, el servidor aplica `TIOCSWINSZ` al PTY
(`server-client.c:1834-1893`; `window.c:589-615`). Una sesión SSH necesitaría
traducir ese cambio a la terminal remota; el código actual solo actualiza el
PTY local.

Al salir el hijo, `SIGCHLD` lleva a `server_child_signal`, que llama a
`server_child_exited` y marca `PANE_EXITED`/`PANE_STATUSREADY`
(`server.c:435-448,466-515`). `window_pane_destroy_ready` espera ese
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

`configure.ac:1062-1065,1118-1123` identifica Linux como `PLATFORM=linux` y
define la condición de Automake `IS_LINUX`:

```m4
AM_CONDITIONAL(IS_LINUX, test "x$PLATFORM" = xlinux)   # configure.ac:1122
```

`Makefile.am:90-91,235-255` lista fuentes comunes, elige `osdep-@PLATFORM@.c`
y muestra otros casos de fuentes condicionales. El patrón de detección y
enlace de una biblioteca está en `configure.ac:250-303` (libevent).

`IS_LINUX` sirve para condicionar reglas de Automake; no es por sí sola una
macro disponible en C. `cmd.c` tiene las declaraciones `extern` de las
entradas en `cmd.c:30-121` y la tabla en `cmd.c:123-216`. Si el comando SSH
solo se enlaza en Linux, ambas referencias necesitan una guarda C definida
desde `configure`, además de condicionar la fuente y la biblioteca en el
build. Hoy `cmd.c` no tiene ningún `#if`: sería la primera entrada condicional
de la tabla.

No se encontraron `libssh`, `ssh_session` ni `ssh_channel` en `Makefile.am`,
`configure.ac`, `tmux.h`, `spawn.c` o `cmd.c` (búsqueda literal). Por tanto,
la biblioteca y sus guards serían nuevos. En builds no Linux no deberían
quedar dependencias de enlace ni referencias al símbolo del comando.

`compat/` aporta reemplazos de funciones del sistema: por ejemplo,
`compat/fdforkpty.c:23-33` y su detección en `configure.ac:828-839` (en el
build de macOS de la línea de base se enlaza `compat/fdforkpty.o`).
`osdep-linux.c:29-99` contiene funciones específicas de Linux; Automake
selecciona ese archivo en `Makefile.am:235`. Ninguno de esos dos mecanismos
constituye por sí mismo un cliente SSH.

### 7. Hay precedentes de código condicional, pero ninguno por plataforma

Tres patrones del repo sirven de molde para el límite solo-Linux:

- **Feature opcional con fuentes propias — sixel.** `configure.ac:545-552`
  declara `--enable-sixel`, hace `AC_DEFINE(ENABLE_SIXEL)` y
  `AM_CONDITIONAL(ENABLE_SIXEL, …)`; `Makefile.am:253-255` agrega las fuentes
  y `tmux.h:79,1028,1104` guarda las declaraciones con `#ifdef ENABLE_SIXEL`.
  Es el molde más cercano a "fuente nueva + declaraciones guardadas":

  ```make
  # Makefile.am:253-255
  if ENABLE_SIXEL
  dist_tmux_SOURCES += image.c image-sixel.c
  endif
  ```

- **Código específico de Linux dentro del hijo de `spawn_pane`.** Justo donde
  engancharía un hijo SSH ya hay una guarda de systemd/cgroups:

  ```c
  /* spawn.c:504-513 */
  #if defined(HAVE_SYSTEMD) && defined(ENABLE_CGROUPS)
  	if (systemd_move_to_new_cgroup(cause) < 0) { ... }
  #endif
  ```

  Su contraparte de build es `configure.ac:500-541` y `Makefile.am:242-245`
  (`AC_DEFINE` + `AM_CONDITIONAL` + `compat/systemd.c`).
- **Alternativa en el servidor.** `server.c:223-227` elige entre
  `systemd_create_socket` y `server_create_socket` con `#ifdef HAVE_SYSTEMD`.

**Lo que no existe:** ningún feature queda restringido a una sola plataforma.
sixel, systemd y cgroups dependen de `--enable-X` y de que la biblioteca
aparezca, no de `PLATFORM`. La plataforma sí cambia algunos defaults: en darwin,
utf8proc pasa a ser obligatorio salvo `--disable-utf8proc`
(`configure.ac:452-457`), y jemalloc sigue el mismo esquema
(`configure.ac:655`). Aun así, los dos se pueden compilar en cualquier
plataforma. Además, `IS_LINUX` está definido pero **no se usa** en ningún lugar
de `Makefile.am`. Una guarda que excluya un feature fuera de Linux sería nueva.

**Trampa de orden en `configure`.** `PLATFORM` se calcula al final, en el
`case "$host_os"` de `configure.ac:1008-1118`, después de todos los chequeos
de bibliotecas (libevent en 250, systemd en 500, sixel en 545). Un chequeo de
libssh condicionado a Linux tiene que ir después de la línea 1118, o hacer su
propio `case "$host_os"` como ya hacen `configure.ac:89,452,655`. Ese mismo
`case` final ya contiene lógica por plataforma que corta el build: en darwin
exige elegir `--enable/--disable-utf8proc` y `--enable/--disable-jemalloc`
(`configure.ac:1014-1056`).

### 8. Algunos formatos del pane seguirían describiendo al proceso local

`pane_current_command` consulta el proceso asociado al PTY y, si no obtiene un
nombre, usa los argumentos o la shell guardados (`format.c:953-975`;
`osdep-linux.c:29-61`). `pane_current_path` consulta el directorio de ese
proceso (`format.c:977-991`; `osdep-linux.c:63-89`). Si el hijo hace de puente
SSH, esos formatos pueden mostrar el proceso y directorio **locales** en vez
del comando o directorio remotos. La spec debe decidir qué se promete al
usuario para un pane SSH.

### 9. `libssh` u OpenSSH: contexto fuera del repo

Esto no sale del código de tmux; queda anotado para la decisión de la spec.

- **OpenSSH no expone una biblioteca de cliente.** "Usar OpenSSH" en la
  práctica es ejecutar el binario `ssh`, que es justo lo que el pedido excluye.
- **`libssh`** (LGPL-2.1) y **`libssh2`** (BSD) son bibliotecas de cliente.
  tmux usa la licencia ISC (`COPYING`); enlazar dinámicamente una biblioteca
  LGPL no cambia eso, pero la spec debería nombrar la licencia de la elegida.
- Cualquiera de las dos tendría que detectarse como libevent, con
  `PKG_CHECK_MODULES` (`configure.ac:250-303`).
- Si la biblioteca corre en el servidor, necesita un modo no bloqueante y un
  fd que se pueda registrar en libevent (hallazgo 3). Si corre en un hijo, esa
  restricción desaparece, pero aparece la del riesgo "hijo sin `exec`".

## Una integración posible, todavía sin decidir

Hay una vía que merece evaluación en la spec: después de `fdforkpty`, hacer
que el **hijo** ejecute código enlazado a una biblioteca SSH en lugar de
llamar a `exec` (`spawn.c:478-575`). Ese hijo podría transportar bytes entre
el PTY local y el canal remoto. Para el servidor, el pane conservaría su
`pid`, su PTY y el `bufferevent` habituales (`tmux.h:1366-1390`;
`window.c:1626-1638`). El hijo también hereda el entorno de la sesión,
incluido `SSH_AUTH_SOCK` si `update-environment` lo trae
(`environ.c:251-269`; `options-table.c:1207-1216`), lo que haría posible la
autenticación por agente.

Es una hipótesis, no una solución comprobada. Faltaría definir autenticación,
verificación del host, conexión, PTY remoto, propagación de resize, señales,
errores y cierre. Si el hijo funciona como puente de bytes, también tendría
que evitar que el modo de línea del PTY local altere la entrada y transportar
datos en ambos sentidos sin bloquear la interacción. El servidor vuelve de
`spawn_pane` tras el `fork` (`spawn.c:492-501,577-613`), y el ejecutor del
comando continúa con selección, hook y retorno (`cmd-split-window.c:288-332`):
un fallo posterior de conexión o autenticación no sería automáticamente un
error síncrono del comando. La spec debe definir cómo se informa, qué ocurre
con el pane y qué estado devuelve una espera como `-W`
(`cmd-split-window.c:323-330`; `window.c:1438-1460`). Alojar SSH en el servidor
es otra posibilidad, con cambios más profundos en las rutas de E/S y salida
descritas en los hallazgos 3 y 4, y con el precedente de `job.c`.

## Módulos tocados

| Área | Archivos | Por qué importan |
|---|---|---|
| Comando y layout | `cmd.c:30-216`; `cmd-split-window.c:39-76,175-208` | Registro y preparación del pane. |
| Spawn y estado | `spawn.c:374-613`; `tmux.h:1361-1390,2499-2532` | PTY, hijo y estado. |
| E/S y cierre | `window.c:1585-1638,2024-2059`; `server.c:466-515`; `server-fn.c:354-435` | Entrada, salida y ciclo de vida. |
| Límite Linux | `configure.ac:250-303,545-552,1008-1123`; `Makefile.am:235-255` | Detección, compilación y enlace condicional. |
| Documentación y tests | `tmux.1:4086-4105`; `regress/pane-ops.sh`; `regress/hooks-notify.sh:281-302,380-394` | Interfaz, panes y eventos. |

## Interfaces reusadas

| Interfaz | Dónde | Contrato que hay que respetar |
|---|---|---|
| `struct cmd_entry` + `cmd_table[]` | `tmux.h:2055-2075`; `cmd.c:30-216` | Un comando es un `extern` más un puntero en la tabla; `cmd_find` lo resuelve por nombre (`cmd.c:460-492`). |
| `struct spawn_context` + `spawn_pane()` | `tmux.h:2499-2532`; `spawn.c:243` | Devuelve el pane o `NULL` con `cause`. Tiene cinco callers: no puede cambiar para ellos. |
| `fdforkpty()` | `spawn.c:478`; `compat/fdforkpty.c:23-33` | Crea hijo y PTY; en macOS lo aporta `compat/`. |
| `environ_for_session()` | `environ.c:251-269` | Entorno del hijo; incluye `update-environment`. |
| `window_pane_set_event()` | `window.c:1626-1638` | `bufferevent` sobre `wp->fd` e `input_init`. |
| `input_key_pane()` / `window_pane_paste()` | `input-keys.c:398-418`; `window.c:2006-2021` | Teclas y pegado se escriben en el `bufferevent` del pane. |
| `window_pane_send_resize()` | `window.c:589-615` | Resize = `TIOCSWINSZ` sobre el PTY local. |
| `server_child_signal()` / `server_child_exited()` | `server.c:466-515` | `PANE_EXITED` y `PANE_STATUSREADY` gobiernan el cierre. |
| `server_destroy_pane()` | `server-fn.c:354-435` | `remain-on-exit` y eventos de salida. |
| `proc_loop()` | `proc.c:223-227`; `server.c:258` | Un solo loop de libevent, sin hilos. |
| `job_run()` | `job.c:72,118,225` | Precedente de fd auxiliar (`socketpair`) en el loop. |

## Riesgos que la spec no debería dejar implícitos

- **Afectar panes normales:** `spawn_pane` sirve a varios comandos y un pane
  interactivo depende de su PTY y del estado de salida (hallazgos 2 a 5).
- **Romper builds no Linux:** las fuentes y la tabla de comandos son comunes;
  hacen falta guardas coherentes de compilación y enlace (hallazgos 6 y 7).
- **Orden de `configure`:** `PLATFORM` no existe todavía cuando corren los
  chequeos de bibliotecas; una guarda mal ubicada evalúa vacío y se comporta
  igual en todas las plataformas (hallazgo 7).
- **Bloquear el servidor:** con un solo loop y sin hilos, una conexión o una
  autenticación bloqueante en el servidor congela todas las sesiones
  (hallazgo 3).
- **Hijo sin `exec`:** hoy el hijo de `spawn_pane` siempre termina en `exec` o
  `_exit` (`spawn.c:550-575`). Un hijo de larga vida heredaría la memoria y el
  estado de libevent del servidor. El único precedente de `fork` sin `exec` es
  la daemonización, que llama a `event_reinit` (`server.c:198-199`).
- **Filtrar datos de autenticación:** además de logs, `pane-created` expone
  `pane_command` si los datos terminan en los argumentos del pane
  (`spawn.c:74-110,436-445`). El entorno puede heredar
  `SSH_AUTH_SOCK`/`SSH_ASKPASS` según `update-environment` (`environ.c:251-269`;
  `options-table.c:1207-1216`), pero eso no establece una política SSH.
- **Un runner que no corre nada:** `regress/Makefile:1` usa `TESTS!= echo *.sh`,
  sintaxis que GNU make 3.81 (el de macOS) no entiende. `make -C regress` expande
  la lista vacía y termina sin error y sin correr ningún test. Verde no significa
  correcto (ver línea de base).

## Decisiones pendientes para la spec

1. Nombre, sintaxis y target del comando: ¿split, pane flotante o ambos?
2. Biblioteca SSH (hallazgo 9), `--enable-ssh` explícito o detección
   automática, y comportamiento de `configure` si falta en Linux o se pide
   fuera de Linux.
3. Verificación del host y autenticación: claves, agente, contraseña y
   mensajes interactivos.
4. Shell o comando remoto; PTY remoto, `TERM` y resize.
5. Qué ocurre al desconectar, matar o respawnear el pane, y si el comando
   ofrece una espera como `-W` o respeta `remain-on-exit`
   (`cmd-respawn-pane.c:33-99`;
   `cmd-split-window.c:323-330`; `server-fn.c:220-235,354-435`).
6. Cómo se separan destino/opciones SSH de `shell-command`, qué se registra en
   logs y qué ve el usuario ante un fallo de conexión posterior al `fork`.
7. Qué significan `pane-created`, `pane_current_command` y
   `pane_current_path` para un pane SSH (hallazgos 2 y 8).
8. Dónde corre la biblioteca: hijo con PTY o servidor (hallazgos 3 y 9).

## Lo que NO hace falta entender

- **Render y terminal del cliente:** `tty.c`, `tty-*.c`, `screen-write.c`,
  `screen-redraw.c`, `grid*.c`. El pane SSH entrega bytes al mismo parser
  (`input.c`); lo que pasa después no cambia.
- **Layout:** `layout*.c`. Se usa como caja negra vía
  `layout_get_tiled_cell` / `layout_get_floating_cell`
  (`cmd-split-window.c:175-179`).
- **Modos y menús:** `window-copy.c`, `window-tree.c`, `mode-tree.c`, `menu.c`.
- **Control mode y cliente:** `control.c`, `control-notify.c`, `client.c`.
  Consumen la salida del pane sin saber de dónde viene.
- **Parser de comandos:** `cmd-parse.y`. Un comando nuevo solo necesita su
  `cmd_entry`.
- **El resto de `compat/`:** salvo `fdforkpty.c` y `systemd.c`, son reemplazos
  de libc que no se tocan.
- **Sixel (`image*.c`):** solo interesa como molde de build (hallazgo 7).

Acotar también es decidir qué no leer.

## Línea de base

El checkout no trae `configure` generado ni binario. Para construir desde Git,
`README:32-39` indica `sh autogen.sh`, `./configure` y `make` (requiere
autoconf, automake y pkg-config).

### macOS (no Linux) — medida

Medida el 2026-09-29 en macOS 26 arm64, sobre una copia del commit base,
con libevent 2.1.12, ncurses del sistema y utf8proc 2.11.3.

```bash
sh autogen.sh
./configure --disable-jemalloc   # sin el flag, configure corta en darwin
make
./tmux -V                        # tmux next-3.9
```

- **Build:** compila y enlaza, con 0 warnings. Enlaza `osdep-darwin.o` y
  `compat/fdforkpty.o`. Esto es lo que el invariante "no-Linux sigue
  compilando" tiene que preservar.
- **Tests más cercanos**, corridos uno por uno como lo hace el runner
  (`env -i LC_CTYPE=C.UTF-8 MallocNanoZone=0 sh -x <test>`, desde `regress/`):
  `pane-ops.sh`, `hooks-notify.sh`, `respawn-pane-control-lag.sh` y
  `kill-session-process-exit.sh` → **PASS**.
- **Suite completa (`regress/*.sh`, 164 tests, ~18 min):** **163 PASS, 1 FAIL.**
  Falla `screen-redraw-menus.sh`, en la escena `menu-over-split`: la captura
  difiere de `regress/screen-redraw-results/menu-over-split.result`. Falla igual
  en tres corridas y sobre el código sin tocar, así que es **preexistente**. Es
  render de menús, fuera de la superficie del cambio. Queda registrada para que
  no se le atribuya después al comando SSH; la causa no se investigó.
- **`make -C regress` no sirve en macOS:** con GNU make 3.81 no corre ningún
  test (riesgo "runner que no corre nada"). Hay que usar el bucle manual de
  arriba o un make que entienda `!=` (GNU make ≥ 4.0 o bmake).

### Linux — pendiente

No medida todavía: el daemon de Docker no estaba disponible en la máquina de
exploración. Hay que repetir los mismos pasos en una distro con `libevent-dev`,
`libncurses-dev`, `bison`, autotools y `pkg-config`, y registrar build y suite
**antes** de tocar nada. Es el lado donde se enlazará el feature.

No se identificó una prueba de cliente SSH nativo en `regress/`; la spec
debería prever una prueba con servidor SSH controlado en Linux y una
comprobación de que el build no Linux excluye la función y sus dependencias
(por ejemplo, que `nm tmux` no tenga símbolos de la biblioteca SSH ni del
comando en macOS).
