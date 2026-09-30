# `ssh-pane` — base context

> **Fase SDD: Especificar (insumo de la spec).** Recoge las decisiones tomadas sobre
> las preguntas abiertas de [`notas-exploracion.md`](./notas-exploracion.md) y su
> fundamento. La spec ([`ssh-pane-spec.md`](./ssh-pane-spec.md)) cita estas decisiones
> como `D-N` y los hallazgos de las notas como "hallazgo N".
>
> Base: `tmux/` en el commit `94796f6b1182507efac8a272fc309a79e22e58a5`. Las
> referencias `archivo:línea` son relativas a `tmux/` en ese commit.

## 1. Pedido refinado

El enunciado pide un **cliente SSH nativo** en tmux: un comando nuevo que abra un
pane con una sesión remota **sin invocar el binario `ssh`**, que se compile **solo en
Linux**, sin romper los builds de macOS y BSD, y sin cambiar los comandos existentes ni
el modelo de PTY/panes.

Refinado con las decisiones de abajo:

- Un comando `ssh-pane [-dh] [-p port] [-t target-pane] [user@]host` que parte el pane
  objetivo, como `split-window`, y abre en el pane nuevo la shell de login remota.
- La conexión la hace un **hijo del servidor**, que se enlaza a `libssh` y usa el PTY
  del pane como terminal local. El servidor ve un pane normal.
- El feature existe solo si se configura con `--enable-ssh`, que solo se acepta en
  Linux. En cualquier otro build, el comando no existe.

## 2. Decisiones

Cada decisión tiene la forma **Elegido / Fundamento / Descartado**. Las que salen de
una pregunta de las notas lo indican ("Decisión pendiente N").

### D-1 · Dónde corre el cliente SSH

*Decisión pendiente 8. Hallazgos 2 a 5 y "Una integración posible".*

**Elegido:** en un **hijo con PTY, sin `exec`**. `spawn_pane` hace `fdforkpty` como
para cualquier pane (`spawn.c:478`). En el hijo, cuando el pane es SSH, en lugar de
llegar al `execvp`/`execl` (`spawn.c:550-575`) se llama a una función puente que no
retorna: conecta con `libssh`, pasa el PTY local a modo crudo y copia bytes entre el
PTY y el canal remoto hasta que termina, y sale con `_exit`.

**Fundamento:**

- Para el servidor, el pane conserva su `pid`, su PTY y su `bufferevent`
  (`window.c:1626-1638`). La entrada (`input_key_pane`, `window_pane_paste`), la salida
  (`window_pane_read_callback`), `pipe-pane` y los clientes de control funcionan igual
  que con cualquier pane.
- El resize ya funciona: el servidor aplica `TIOCSWINSZ` al PTY
  (`window.c:589-615`) y el kernel le manda `SIGWINCH` al hijo, que lo traduce a un
  cambio de tamaño del PTY remoto.
- El cierre ya funciona: cuando el hijo termina, `SIGCHLD` lleva a
  `server_child_exited` (`server.c:466-515`) y `remain-on-exit` se aplica como siempre
  (`server-fn.c:354-435`).
- DNS, conexión, handshake y autenticación ocurren en el hijo. El loop único del
  servidor (`proc.c:223-227`) nunca espera la red.
- Las credenciales (claves y agente) solo las toca el hijo. El servidor no lee ninguna
  clave ni abre el socket del agente.

**Riesgo aceptado y acotado:** el hijo hereda la memoria del servidor y el estado de
libevent. El hijo de `spawn_pane` ya limpia señales y descriptores antes del `exec`
(`spawn.c:540-544`). El puente parte de ese mismo punto: no usa el loop heredado ni
ninguna estructura del servidor salvo los datos de conexión que recibe, y termina
siempre con `_exit`. El único precedente de `fork` sin `exec` en tmux es la
daemonización, que llama a `event_reinit` (`server.c:198-199`); el puente no lo
necesita, porque no usa libevent.

**Descartado:**

| Opción | Por qué no |
|---|---|
| Hijo que re-ejecuta tmux (`/proc/self/exe`) en un modo interno | Imagen de proceso limpia, pero agrega un punto de entrada oculto en `tmux.c` (archivo de OpenBSD, hallazgo 10) y hay que pasar los datos de conexión por argv o por el entorno, donde quedan visibles |
| Cliente SSH en el servidor, con un socket registrado en libevent (patrón `job.c`) | El pane no tendría hijo ni PTY: habría que reescribir entrada, salida, resize y cierre (hallazgos 3 y 4). Rompe el invariante "el modelo de PTY/panes no cambia", y un error de no bloqueo congela todas las sesiones |
| Pane vacío (`-E`, `SPAWN_EMPTY`) al que se le inyecta la salida | Con `fd = -1` tmux descarta teclado, pegado y resize (hallazgo 5): habría que crear rutas nuevas para todo eso |
| Pasar `ssh` como `shell-command` | Ejecuta el binario externo: es justo lo que el enunciado excluye (hallazgo 2) |
| Sesiones remotas por *control mode* ([#5566](https://github.com/tmux/tmux/issues/5566)) | Es la dirección que prefiere upstream, pero exige un tmux en el host remoto y delega la red a `ssh`. No es un cliente SSH nativo, que es lo que pide el enunciado. Se registra como alternativa de upstream, no como camino de este cambio |

### D-2 · Biblioteca SSH

*Decisión pendiente 2. Hallazgo 9.*

**Elegido:** **`libssh` ≥ 0.10**, detectada con `PKG_CHECK_MODULES` sobre el módulo
`libssh`, como libevent (`configure.ac:250-303`).

**Fundamento:** es una biblioteca de cliente con lo que el feature necesita, sin
código propio de protocolo: verificación contra `known_hosts`
(`ssh_session_is_known_server`), autenticación por agente y por claves
(`ssh_userauth_publickey_auto`), PTY remoto con tamaño y `TERM`
(`ssh_channel_request_pty_size`) y cambio de tamaño
(`ssh_channel_change_pty_size`). La versión 0.10 es la de Ubuntu 24.04 (0.10.6, la
plataforma de la línea de base Linux) y la de Debian 12.

**Licencia:** `libssh` es LGPL-2.1 y tmux es ISC (`COPYING`). Con enlace dinámico no
cambia la licencia de tmux. El enlace estático se prohíbe (D-3).

**Descartado:**

| Opción | Por qué no |
|---|---|
| OpenSSH | No expone una biblioteca de cliente: "usar OpenSSH" es ejecutar `ssh` (hallazgo 9) |
| `libssh2` | Licencia BSD, sin problema en estático, pero más bajo nivel: sin API de alto nivel para `known_hosts` con el formato de OpenSSH ni autenticación automática. Habría más código propio en la parte más delicada |

### D-3 · Cómo se activa en el build

*Decisión pendiente 2. Hallazgos 6, 7 y 11.*

**Elegido:** **`--enable-ssh`, apagado por defecto.**

- Sin el flag, o con `--disable-ssh`, `configure` no busca `libssh` y el resumen
  imprime `libssh: off`.
- Con `--enable-ssh` en Linux y `libssh` ≥ 0.10 disponible, se compila el feature y el
  resumen imprime `libssh: <versión>`.
- `--enable-ssh` fuera de Linux corta `configure` con error.
- `--enable-ssh` en Linux sin `libssh` ≥ 0.10 corta `configure` con error.
- `--enable-ssh` junto con `--enable-static` corta `configure` con error.

El chequeo va **después** del `case "$host_os"` que calcula `PLATFORM`
(`configure.ac:1008-1118`), como el de `vlock` (`configure.ac:1133-1142`), porque antes
de esa línea `PLATFORM` está vacío (hallazgo 7, "trampa de orden"). Hace
`AC_DEFINE(ENABLE_SSH)` y `AM_CONDITIONAL(ENABLE_SSH, …)`, como sixel
(`configure.ac:545-552`).

**Fundamento:**

- Con el opt-in, el build estático oficial de `tmux-builds`
  (`--enable-static --enable-utf8proc --disable-jemalloc`, hallazgo 11) y cualquier
  build Linux existente quedan igual, aunque la máquina tenga `libssh` instalada.
- Si se pide `--enable-ssh` y no se puede cumplir, fallar en `configure` es mejor que
  producir en silencio un binario sin el comando: así lo hacen `--enable-utf8proc`
  (`configure.ac:452-457`) y los demás features opcionales.
- Prohibir `--enable-static` evita que alguien genere por accidente un binario con
  `libssh` (LGPL-2.1) y `libcrypto` estáticas, con la obligación de permitir el
  re-enlazado (hallazgo 9). Levantar esa prohibición es una decisión aparte.

**Descartado:**

| Opción | Por qué no |
|---|---|
| Detección automática en Linux | El build estático oficial empezaría a enlazar `libssh` sin que nadie lo decida (hallazgo 11) |
| Permitir `--enable-ssh --enable-static` y documentar la LGPL | Deja implícito el riesgo que las notas piden explicitar: la obligación recae en quien distribuye, sin aviso |
| Condicionar solo por `IS_LINUX`, sin flag | `IS_LINUX` es una condición de Automake, no una macro de C (hallazgo 6); igual haría falta un `AC_DEFINE`, y se volvería a la detección automática |

### D-4 · Guardas y archivos

*Decisión pendiente 10. Hallazgos 6, 7 y 10.*

**Elegido:** el grueso vive en **dos archivos nuevos, propios de portable**, que solo
se compilan bajo `if ENABLE_SSH` en `Makefile.am`:

- `cmd-ssh-pane.c`: la entrada `cmd_ssh_pane_entry` y su ejecutor.
- `ssh-pane.c`: el puente del hijo (D-1).

Los archivos compartidos con OpenBSD cambian lo mínimo, siempre bajo
`#ifdef ENABLE_SSH`:

- `cmd.c`: el `extern` y la entrada de la tabla (`cmd.c:30-216`). Es el primer `#if` de
  ese archivo (hallazgo 6).
- `tmux.h`: los datos de conexión en `struct window_pane` y en `struct spawn_context`,
  y los prototipos.
- `spawn.c`: guardar el destino en el pane y, en el hijo, llamar al puente en vez de
  hacer `exec`.
- `window.c`: liberar los datos de conexión al destruir el pane.

**Fundamento:** es el molde de sixel (fuente nueva + declaraciones guardadas, hallazgo
7). Cuanto menos código propio haya dentro de los archivos de OpenBSD, menos se expone
a que un merge lo borre en silencio, como pasó con utempter (hallazgo 10). Por eso
`cmd-respawn-pane.c`, `format.c`, `server*.c` y `window.c` (salvo la liberación) no se
tocan.

**Descartado:**

| Opción | Por qué no |
|---|---|
| Poner el puente en `osdep-linux.c` | `osdep` es para funciones puntuales por plataforma, no para features (lección de la historia de compat) |
| Poner el puente en `compat/` | `compat/` reemplaza funciones del sistema; no es lugar para un feature (hallazgo 6) |
| Un `ssh-pane` en todas las plataformas con stub | Ver D-6 |

### D-5 · Nombre del comando

*Decisión pendiente 1. Hallazgo 12.*

**Elegido:** **`ssh-pane`**, sin alias.

**Fundamento:** no vuelve ambigua ninguna de las abreviaturas que hoy funcionan
(hallazgo 12, simulación sobre los 92 comandos). El nombre dice qué crea (un pane) y
con qué (SSH).

**Descartado:**

| Opción | Por qué no |
|---|---|
| `ssh` | Tampoco rompe abreviaturas, pero en un `.tmux.conf` o en la documentación se confunde con el binario `ssh`, que es justo lo que no se ejecuta |
| `ssh-window` | El pedido es un pane; abrir una ventana nueva es otra forma, no la pedida |
| `new-ssh`, `new-ssh-pane`, `connect-pane`, `new-pane-ssh`, `split-ssh`, `split-window-ssh` | Rompen entre 1 y 10 abreviaturas existentes, solo en Linux (hallazgo 12) |
| Alias corto (p. ej. `sshp`) | Los alias exactos no generan ambigüedad, pero no hacen falta: `ssh-p` ya es una abreviatura única |

### D-6 · El comando en builds sin el feature

*Decisión pendiente 9. Hallazgo 12.*

**Elegido:** **el comando no existe.** Sin `ENABLE_SSH`, `cmd_table` no tiene la
entrada. `tmux ssh-pane …` falla con `unknown command: ssh-pane` (`cmd.c:488-489`),
`list-commands` devuelve los mismos 92 comandos de la línea de base y el binario no
tiene ningún símbolo que contenga `ssh`.

**Fundamento:** es la única opción en la que "el feature se compila afuera" se
verifica con `nm`. Además, el binario no Linux es idéntico en comandos al de la
línea de base.

**Costo aceptado:** un `.tmux.conf` compartido que use `ssh-pane` falla en esa línea
al cargarse en otra plataforma. Se puede proteger con `%if` y un formato, igual que
cualquier diferencia entre versiones de tmux.

**Descartado:**

| Opción | Por qué no |
|---|---|
| El comando existe en todas las plataformas y responde "not supported" | La entrada y su stub se compilarían en macOS y BSD (93 comandos, símbolos `ssh`): el feature ya no queda afuera del build |

### D-7 · Sintaxis y flags

*Decisión pendiente 1 y 6.*

**Elegido:** `ssh-pane [-dh] [-p port] [-t target-pane] [user@]host`.

- `-t`: el pane objetivo, resuelto como en `split-window` (`CMD_FIND_PANE`).
- `-h`: parte a la derecha. Sin `-h`, parte abajo (el default de `split-window`).
- `-d`: el pane nuevo no queda activo.
- `-p`: el puerto, un entero entre 1 y 65535 escrito solo con dígitos decimales. Por
  defecto, 22.
- `[user@]host`: exactamente un argumento. Sin `user@`, el usuario es el nombre de la
  cuenta local del proceso (`getpwuid(getuid())`), no `$USER`, igual que `ssh`.
- **No se lee `~/.ssh/config`.**
- **Sin hook propio.** No dispara `after-split-window` y no existe `after-ssh-pane`:
  la entrada no lleva `CMD_AFTERHOOK` (`cmd-queue.c:629-638`) y el ejecutor no llama a
  `cmdq_insert_hook` como hace `split-window` (`cmd-split-window.c:315`).

Los datos de conexión viajan en `spawn_context` y quedan en el pane (D-13), nunca en
`sc.argv`. Así `spawn_pane` no los interpreta como comando local, no aplica
`default-command` y no los registra como `cmd=` (hallazgo 2).

**Fundamento:**

- Con pocos flags, cada uno tiene un requisito y un VC. Los de `split-window` que no
  están (`-b`, `-f`, `-l`, `-Z`, `-c`, `-e`, `-F`/`-P`, `-T`, `-W`, `-I`, `-E`, …)
  quedan fuera de la v1.
- `-v` se omite porque es el default. Con `-v`, habría que especificar qué pasa con
  `-h -v`.
- Sin `~/.ssh/config`, el comportamiento no depende de un archivo con decenas de
  directivas (`Host`, `Match`, `ProxyJump`, `Include`…), y los VCs tampoco.
- El usuario sale de `getpwuid` porque `$USER` es una variable del entorno del pane que
  cualquiera puede cambiar con `set-environment`.

**Descartado:**

| Opción | Por qué no |
|---|---|
| Leer `~/.ssh/config` con `ssh_options_parse_config` | La spec tendría que fijar qué directivas se respetan, y cada una sería un requisito. Se puede sumar en otra iteración |
| Todos los flags de `split-window` | Varios no tienen un significado obvio para un pane remoto (`-c` directorio local, `-e` entorno, `-W` espera): mucha superficie sin pedido |
| `host:port` en el destino | Choca con los literales IPv6 (`::1`) |
| Datos de conexión en `sc.argv` | `spawn_pane` los trataría como comando local, les aplicaría `default-command` y los registraría como `cmd=` (hallazgo 2) |
| Hook `after-ssh-pane` | Cada hook `after-<comando>` es una entrada de `options-table.c` (`OPTIONS_TABLE_AFTER_HOOK`, `options-table.c:273-275,1932`); sin ella, `set-hook` lo rechaza con `invalid option`. Sumarlo toca un archivo de OpenBSD más, sin pedido concreto |
| Disparar `after-split-window` | El hook promete que corrió `split-window`; un script que lo escuche recibiría un pane que no pidió |

### D-8 · Verificación de la clave del host

*Decisión pendiente 3.*

**Elegido:** **`known_hosts` estricto, un solo archivo.** Solo se continúa si la clave
que presenta el servidor figura en `<dir-ssh>/known_hosts` para ese host y ese puerto
y coincide. `<dir-ssh>` es el de D-10. Host desconocido, archivo ausente o clave
distinta: el hijo termina con error y no escribe nada en `known_hosts`. No se consulta
`/etc/ssh/ssh_known_hosts`. Las entradas hasheadas (`HashKnownHosts yes`, el default de
Ubuntu) se reconocen. Para un puerto distinto de 22 vale la convención de OpenSSH: la
entrada es `[host]:puerto`.

**Fundamento:** con una sola fuente de verdad, bajo control de la persona, el
comando nunca acepta una clave que la persona no haya agregado. Además, el entorno de
prueba la controla con `HOME`. Es equivalente a `StrictHostKeyChecking yes` y un poco
más estricto que `ssh`, que también lee el archivo global.

**Descartado:**

| Opción | Por qué no |
|---|---|
| Preguntar en el pane (*trust on first use*) y agregar la clave | Escribe en el HOME de la persona y suma un diálogo interactivo que especificar (respuestas, eco, timeout). Hoy basta con agregar la clave con las herramientas habituales antes de conectar |
| Consultar también `/etc/ssh/ssh_known_hosts` | Una segunda fuente que la persona no controla y que el entorno de prueba tendría que vaciar |
| No verificar | Expone a *man in the middle*. Inaceptable en un cliente SSH |

### D-9 · Autenticación

*Decisión pendiente 3. Enunciado: "¿auth por claves o por agent?".*

**Elegido:** **agente primero, después las claves por defecto sin passphrase.**

1. El agente, si `SSH_AUTH_SOCK` del entorno del pane apunta a un socket que responde:
   se prueban sus identidades.
2. Si no hay agente, o ninguna identidad fue aceptada, se prueban en este orden
   `<dir-ssh>/id_ed25519`, `<dir-ssh>/id_ecdsa` y `<dir-ssh>/id_rsa`. Las que no existen
   o tienen passphrase se saltean sin pedir nada.
3. Si ninguna fue aceptada, el hijo termina con error.

Nunca se prueban contraseña ni *keyboard-interactive*, y nunca se lee nada del teclado
antes de abrir la shell remota.

**Fundamento:** cubre las dos formas habituales de autenticarse sin tipear nada. El
agente llega por el entorno del pane, que ya incluye `SSH_AUTH_SOCK` gracias a
`update-environment` (`environ.c:251-269`; `options-table.c:1207-1216`), y eso
habilita claves con passphrase sin que tmux las maneje. Sin diálogos en el pane, no hay
secretos tipeados en un PTY que el servidor también lee (salida hacia `pipe-pane` y
clientes de control, hallazgo 3).

**Descartado:**

| Opción | Por qué no |
|---|---|
| Pedir la passphrase de la clave en el pane | Agrega un diálogo, el caso de passphrase incorrecta y el manejo del eco. El agente ya lo resuelve |
| Contraseña del usuario remoto | Es lo que más diálogo y más casos de falla agrega, y la persona tipearía un secreto en un pane cuya salida puede estar en `pipe-pane` |
| Solo agente | Quien usa claves sin agente no podría usar el comando |

### D-10 · Directorio SSH local

**Elegido:** `<dir-ssh>` es `$HOME/.ssh`, con `HOME` tomado del entorno del pane
(`environ_for_session`, `environ.c:251-269`). Si `HOME` no está definido en ese
entorno, se usa el directorio de la cuenta local (`getpwuid(getuid())`).

**Fundamento:** el hijo ya tiene instalado el entorno del pane (`environ_push`,
`spawn.c:544`), y así el entorno de prueba controla `known_hosts` y las claves
arrancando el servidor con otro `HOME`. El respaldo con `getpwuid` es el mismo que usa
tmux para la shell por defecto.

**Descartado:**

| Opción | Por qué no |
|---|---|
| Siempre `getpwuid` | No se puede aislar en pruebas sin tocar `/etc/passwd` |
| Dejar la elección por defecto de `libssh` | Depende de la versión de la biblioteca: la spec no podría fijar el comportamiento |

### D-11 · La sesión remota

*Decisión pendiente 4.*

**Elegido:** **solo la shell de login remota, con PTY.** El hijo pide un PTY remoto
con el `TERM` del entorno del pane (el valor de `default-terminal`,
`environ.c:265-266`) y con el tamaño del pane, y después abre la shell. No hay comando
remoto, no se envían variables de entorno ni se reenvían agente, X11 o puertos. Cada
`SIGWINCH` que recibe el hijo se traduce a un cambio de tamaño del PTY remoto. El PTY
local queda en modo crudo: los bytes, incluidos los caracteres de control (`C-c`,
`C-z`), viajan sin transformarse.

**Fundamento:** es lo mínimo que hace útil al pane: una shell interactiva que se ve
bien y se redimensiona. Sin modo crudo, `C-c` mataría al hijo local en lugar de
interrumpir el proceso remoto (notas, "Una integración posible").

**Descartado:**

| Opción | Por qué no |
|---|---|
| Comando remoto opcional (`ssh-pane host cmd`) | Suma el parseo del comando y sus VCs sin pedido concreto |
| Reenvío de agente | Expone el agente local al host remoto; es una decisión de seguridad aparte |
| Enviar variables de entorno (`LANG`, `LC_*`) | Depende de `AcceptEnv` del servidor; agrega casos sin pedido |

### D-12 · Fallas después del `fork`

*Decisión pendiente 6. Notas, "Una integración posible".*

**Elegido:** el comando ya terminó cuando la conexión falla, así que la falla **no es
un error del comando**. El hijo escribe una línea `ssh-pane: …` en el PTY (se ve en el
pane) y sale con código `255`, como `ssh`. Después el pane sigue `remain-on-exit`:
si está en `off`, se cierra y el mensaje se pierde. Con `on`, queda muerto, con el
mensaje visible y `pane_dead_status` = `255`. Si la shell remota termina con un
código, el hijo sale con ese código.

Conectar y autenticar tienen un plazo **fijo de 30 s**, desde que empieza el hijo
hasta que la shell remota queda abierta. Si se cumple, el hijo termina con el mensaje
de timeout.

**Fundamento:** no cambia el ciclo de vida del pane: `server_child_exited` y
`server_destroy_pane` hacen lo mismo que con cualquier pane (hallazgo 4). Con el
código 255, un script distingue "falló la conexión" de "la shell remota salió con N",
igual que con `ssh`. Con un plazo fijo hay una cota verificable: sin él, un `connect`
TCP puede tardar más de dos minutos en Linux.

**Descartado:**

| Opción | Por qué no |
|---|---|
| Dejar el pane abierto ante una falla aunque `remain-on-exit` esté en `off` | Cambia `server_destroy_pane` o el estado del pane: un ciclo de vida especial para un tipo de pane |
| Mensaje en la línea de estado del cliente | El hijo no tiene canal hacia el cliente; habría que crear uno hijo→servidor |
| Esperar la conexión antes de devolver el comando | Bloquearía la cola de comandos del cliente, o haría falta un mecanismo como `-W` (`window.c:1438-1460`) |
| Plazo configurable | Una opción más sin pedido; 30 s alcanza para redes lentas y acota la espera |

### D-13 · Qué recuerda el pane

*Decisiones pendientes 5 y 7. Hallazgos 2 y 8.*

**Elegido:** el `window_pane` guarda el destino ya resuelto (usuario, host y puerto),
bajo `#ifdef ENABLE_SSH`. No guarda ningún secreto porque el servidor no tiene ninguno
(D-9). Con esos datos:

- `respawn-pane` **sin comando** sobre un pane SSH vuelve a conectar al mismo destino,
  con `-k` si el pane está vivo o sin `-k` si está muerto.
- `respawn-pane` **con comando** lo ejecuta como pane local, como hoy, y el pane deja
  de ser SSH.
- `respawn-window` sobre una ventana cuyo primer pane es SSH también reconecta:
  conserva ese pane y llama a `spawn_pane` con `SPAWN_RESPAWN`
  (`spawn.c:129-150,209`), así que pasa por la misma lógica sin tocar
  `cmd-respawn-window.c`.
- `pane-created` informa como `pane_command` la forma normalizada
  `ssh-pane -p <port> <user>@<host>`.
- `wp->argv` queda vacío y `wp->shell` tiene el valor de siempre (`default-shell`).

**Fundamento:** reconectar un pane que se cayó es el uso natural de `respawn-pane`.
Como `spawn_pane` ya usa los argumentos guardados cuando `SPAWN_RESPAWN` no trae
comando (`spawn.c:396-404`), toda la lógica queda en `spawn.c`, y
`cmd-respawn-pane.c` no cambia.

**Descartado:**

| Opción | Por qué no |
|---|---|
| Rechazar `respawn-pane` en un pane SSH | Requiere el mismo campo nuevo y quita el caso de uso más útil |
| No guardar nada | `respawn-pane` abriría una shell **local** en un pane que era remoto, y `pane-created` informaría `default-shell` |

### D-14 · Formatos que describen al proceso local

*Decisión pendiente 7. Hallazgo 8.*

**Elegido:** `pane_current_command` y `pane_current_path` **no se adaptan**. En un pane
SSH, `pane_current_command` es `tmux`, porque el hijo es un `fork` del servidor y
`osdep_get_name` lee `argv[0]` de `/proc/<pgrp>/cmdline` (`osdep-linux.c:29-61`;
`format.c:953-975`). `pane_current_path` es el directorio local del hijo. La
documentación de `ssh-pane` lo aclara.

**Fundamento:** adaptarlos obliga a tocar `format.c` y `osdep-linux.c`, código común a
todos los panes y, en el caso de `format.c`, archivo de OpenBSD. Así el cambio queda
fuera de la superficie mínima. El valor es predecible y queda fijado por un requisito.

**Descartado:**

| Opción | Por qué no |
|---|---|
| Mostrar el comando o el directorio remotos | No hay forma de saberlos sin un protocolo adicional con el host remoto |
| Mostrar `ssh-pane` en `pane_current_command` | Requiere una rama nueva en `format.c` y cambia la semántica del formato ("el proceso en primer plano del PTY") |

### D-15 · Plataformas donde se verifica

*Invariante del enunciado: "los builds no-Linux siguen compilando". Hallazgo 11.*

**Elegido:** **Linux** (Ubuntu 24.04 x86_64, la de la línea de base) con y sin
`--enable-ssh`, **macOS** y **FreeBSD**.

**Fundamento:** el enunciado nombra macOS y BSD. La CI de upstream solo cubre macOS
como no Linux, y no corre ni en PRs ni en forks (hallazgo 11), así que la comprobación
la hace el equipo. FreeBSD es el BSD que compila tmux **portable** con `configure`;
OpenBSD compila su propio árbol con su `Makefile` (hallazgo 10), que este cambio no
toca.

**Cómo se compara:** en macOS y FreeBSD la verificación es **diferencial**: en la misma
máquina y la misma corrida se construye y prueba el commit base y el commit con el
cambio, con la misma receta (`SHELL=/bin/sh`). Así la línea de base de FreeBSD, que no
se midió, no queda como un pendiente de la spec, y la de macOS, que se midió sin
`SHELL=/bin/sh` (notas, "Línea de base"), no se compara contra una receta distinta.

**Descartado:**

| Opción | Por qué no |
|---|---|
| Solo macOS | No cubre "BSD" del enunciado |
| Sumar tmux portable en OpenBSD | Otra VM que mantener, para una plataforma que en la práctica usa el árbol de OpenBSD |

### D-16 · Postura frente a upstream

*Decisión pendiente 11. Hallazgo 9.*

**Elegido:** el cambio se especifica como un **feature de tmux portable (o de un
fork)**, no como un cambio para proponer upstream. La spec lo registra y no depende de
que upstream lo acepte.

**Fundamento:** el mantenedor declaró dos veces que prefiere sesiones remotas por
control mode y que tmux no haga red ni cifrado (#1643, #5566). El `pledge` del servidor
en OpenBSD no incluye `inet` ni `dns` (`tmux.c:540-542`). Como la red la hace un hijo y
el feature no existe fuera de Linux, esa restricción no se toca.

## 3. Esquema de arquitectura

```text
tmux ssh-pane -p 2222 tester@127.0.0.1                     (cliente)
  │
  ▼
cmd.c ── cmd_table[] ──► cmd_ssh_pane_entry                (#ifdef ENABLE_SSH)
  │
  ▼
cmd-ssh-pane.c  valida -p y el destino (errores síncronos del comando)
  │             resuelve pane y celda del layout, como split-window
  │             llena spawn_context + datos de conexión (no sc.argv)
  ▼
spawn.c  spawn_pane ── guarda el destino en window_pane ── fdforkpty
  │                                                   │
  │ servidor (sin cambios)                            │ hijo
  ▼                                                   ▼
window_pane_set_event: bufferevent del PTY           ssh-pane.c (no retorna)
input / pipe-pane / control / resize (TIOCSWINSZ)      PTY local en modo crudo
SIGCHLD → server_child_exited → remain-on-exit         libssh: known_hosts, agente, claves
                                                       PTY remoto (TERM, tamaño)
                                                       SIGWINCH → cambio de tamaño remoto
                                                       bytes PTY ⇄ canal
                                                       _exit(código remoto | 255)
```

## 4. Riesgos que quedan después de decidir

| Riesgo | Mitigación en la spec |
|---|---|
| El hijo sin `exec` arrastra estado del servidor | El puente no usa el loop heredado y termina con `_exit` (D-1); INV-4 verifica que los panes normales siguen haciendo `exec` |
| Un merge de OpenBSD borra los `#ifdef` en `cmd.c`, `spawn.c`, `tmux.h` o `window.c` | Los VCs de la spec (con `--enable-ssh`) fallan si falta alguna pieza; el grueso del código vive en archivos propios de portable (D-4) |
| La CI no verifica macOS, FreeBSD ni `--enable-ssh` | INV-1 a INV-3 son verificaciones manuales obligatorias, con receta; las de macOS y FreeBSD, diferenciales (D-15) |
| Un agente "de paso" toca archivos fuera del alcance | INV-9: `git diff --name-only` contra el commit base solo puede listar los archivos de la tabla "Dentro" |
| Los tests nuevos rompen la suite en builds sin el feature | INV-6: los tests de `ssh-pane` se saltean con código `0` si el comando no existe |
| `.tmux.conf` compartido con `ssh-pane` | Costo aceptado (D-6); se documenta en `tmux.1` |
