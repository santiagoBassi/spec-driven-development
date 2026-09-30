# `ssh-pane` — spec

> **Estado: borrador, pendiente de revisión.** Sin preguntas abiertas. La línea de base
> de FreeBSD todavía no está medida: medirla es un requisito previo del plan (INV-2).
>
> Construida a partir de [`notas-exploracion.md`](./notas-exploracion.md) (se citan como
> "hallazgo N") y de [`ssh-pane-base-context.md`](./ssh-pane-base-context.md), donde
> están el fundamento y las opciones descartadas de cada decisión (se citan como `D-N`).
>
> Base: `tmux/` en el commit `94796f6b1182507efac8a272fc309a79e22e58a5`. Toda
> referencia `archivo:línea` es relativa a `tmux/` en ese commit.
>
> Regla estructural: **cada FR, BR, INV y NFR tiene un VC, y cada VC corresponde a un
> único requisito.** El VC lleva el identificador de su requisito: VC-13 verifica FR-13,
> VC-BR-2 verifica BR-2, VC-INV-3 verifica INV-3 y VC-NFR-1 verifica NFR-1. Los
> requisitos hermanos, que comparten tema, llevan el mismo número con un sufijo (`a`,
> `b`, …). Si una línea no se puede verificar, no está especificada.
>
> Esta spec describe la **v1 completa**. Cómo se parte en iteraciones se decide en el
> plan, no acá.

## Propósito

Permitir que alguien que trabaja en tmux sobre Linux abra una shell remota en un pane
nuevo **sin que tmux ejecute el binario `ssh`**: la conexión la hace tmux con `libssh`,
sin cambiar el comportamiento de los comandos ni de los panes existentes y sin que el
feature exista en los builds que no son Linux.

## Alcance

### Dentro

| Archivo | Cambio | Dueño (hallazgo 10) |
|---|---|---|
| `configure.ac` | `--enable-ssh`: chequeo después del `case "$host_os"` (`configure.ac:1008-1118`), `PKG_CHECK_MODULES` de `libssh` ≥ 0.10, `AC_DEFINE(ENABLE_SSH)`, `AM_CONDITIONAL(ENABLE_SSH)` y la línea `libssh:` en el resumen (`configure.ac:1144-1167`) | portable |
| `Makefile.am` | `if ENABLE_SSH` agrega `cmd-ssh-pane.c` y `ssh-pane.c` a las fuentes (patrón de `Makefile.am:253-255`) | portable |
| `cmd-ssh-pane.c` (nuevo) | `cmd_ssh_pane_entry` y su ejecutor: valida argumentos, resuelve el pane y la celda del layout como `cmd_split_window_exec` (`cmd-split-window.c:79-208`) y llama a `spawn_pane` | portable |
| `ssh-pane.c` (nuevo) | El puente del hijo (D-1): `libssh`, PTY local en modo crudo, copia de bytes, `SIGWINCH`, mensajes de falla, `_exit` | portable |
| `cmd.c` | `extern` y entrada de la tabla bajo `#ifdef ENABLE_SSH` (`cmd.c:30-216`) | OpenBSD |
| `tmux.h` | Datos de conexión en `struct window_pane` (`tmux.h:1361-1390`) y en `struct spawn_context` (`tmux.h:2499-2532`), y prototipos, bajo `#ifdef ENABLE_SSH` | OpenBSD |
| `spawn.c` | Bajo `#ifdef ENABLE_SSH`: guardar el destino en el pane, conservarlo o borrarlo en un respawn (`spawn.c:396-404`), informar `pane_command` en `pane-created` (`spawn.c:74-110`) y, en el hijo, llamar al puente en lugar de hacer `exec` (`spawn.c:546-575`) | OpenBSD |
| `window.c` | Liberar los datos de conexión al destruir el pane, bajo `#ifdef ENABLE_SSH` | OpenBSD |
| `tmux.1` | Entrada de `ssh-pane` junto a `split-window` (`tmux.1:4087`), que aclare que existe solo en Linux con `--enable-ssh` y qué muestran `pane_current_command` y `pane_current_path` (D-14) | OpenBSD |
| `regress/ssh-pane-*.sh` (nuevos) | Pruebas automatizadas de los VCs que corren en el host Linux de prueba; se saltean si el comando no existe (INV-6) | portable |

### Fuera

Por path. **Estos archivos no se modifican:**

- **`cmd-split-window.c`, `cmd-respawn-pane.c`, `cmd-new-window.c`**: los comandos
  existentes no cambian. La reconexión con `respawn-pane` se resuelve en `spawn.c`
  (D-13).
- **`server.c`, `server-fn.c`, `server-client.c`, `proc.c`, `job.c`**: el loop, el
  ciclo de vida del pane y el resize siguen como están (D-1).
- **`window.c`, salvo la liberación de los datos de conexión**: entrada, salida,
  `bufferevent` y resize de los panes no cambian (`window.c:589-615,1585-1638,2005-2059`).
- **`input.c`, `input-keys.c`**: el parser y la codificación de teclas no cambian.
- **`format.c`, `osdep-*.c`**: `pane_current_command` y `pane_current_path` no se
  adaptan (D-14).
- **`compat/`**: no aporta nada al feature (D-4).
- **`tmux.c`**: sin punto de entrada nuevo ni cambios de `pledge` (D-1, D-16).
- **`options-table.c`**: el feature no agrega opciones.
- **`.github/workflows/`, `.travis.yml`**: la CI de upstream no se modifica (INV-1 a
  INV-3 se verifican a mano).

Por funcionalidad. Cada ítem es una decisión tomada, no un olvido:

- Ejecutar `ssh` o cualquier otro programa para conectar (FR-7).
- `ssh-pane` en builds sin `--enable-ssh` o fuera de Linux: el comando no existe (D-6).
- `--enable-ssh` con `--enable-static` (D-3).
- Leer `~/.ssh/config` o `/etc/ssh/ssh_config` (D-7).
- Flags de `split-window` que no son `-d`, `-h` y `-t`: `-b`, `-f`, `-l`, `-Z`, `-c`,
  `-e`, `-F`, `-P`, `-T`, `-W`, `-I`, `-E` y los de panes flotantes. Tampoco `-v`, que
  es el default (D-7).
- Preguntar si se confía en un host desconocido o escribir `known_hosts` (D-8).
- Consultar `/etc/ssh/ssh_known_hosts` (D-8).
- Autenticarse con contraseña, *keyboard-interactive* o pidiendo la passphrase de una
  clave (D-9).
- Claves distintas de `id_ed25519`, `id_ecdsa` e `id_rsa`, o elegidas por flag (D-9).
- Comando remoto, variables de entorno hacia el remoto, reenvío de agente, X11 o
  puertos (D-11).
- Plazo de conexión configurable (D-12).
- Mostrar el comando o el directorio remotos en los formatos (D-14).
- Proponer el cambio a upstream (D-16).

### Límite solo-Linux

El límite se define en tres capas. Cada una tiene su verificación:

| Capa | Regla | Verificación |
|---|---|---|
| `configure` | `--enable-ssh` se acepta solo si `PLATFORM` es `linux`; en otro caso, error (D-3) | FR-2 |
| Automake | `cmd-ssh-pane.c` y `ssh-pane.c` se compilan y `libssh` se enlaza solo bajo `ENABLE_SSH` | FR-1, INV-1 a INV-3 |
| C | Toda referencia a `cmd_ssh_pane_entry`, a los datos de conexión y al puente en archivos compartidos va bajo `#ifdef ENABLE_SSH` | INV-1 a INV-3 (`nm` sin símbolos `ssh`, 92 comandos) |

"Build con el feature" significa Linux configurado con `--enable-ssh`. "Build sin el
feature" es cualquier otro: Linux sin el flag, macOS o FreeBSD.

### Restricciones técnicas

- `libssh` ≥ 0.10, detectada con `pkg-config` (módulo `libssh`) (D-2).
- C y Autotools, como el resto de tmux portable. Ningún cambio exige otra versión de
  Autoconf, Automake o libevent que la de la línea de base.
- Todos los VCs invocan el binario `./tmux` construido en la raíz de `tmux/`.

## Actores

| Actor | Descripción |
|---|---|
| **Persona usuaria** | Usa tmux en Linux; ejecuta `ssh-pane` desde un pane, desde `.tmux.conf` o desde una shell con `tmux ssh-pane …` |
| **Servidor tmux** | Resuelve el pane, crea el PTY y el hijo, y trata al pane como a cualquier otro. No hace red ni toca credenciales (BR-1) |
| **Hijo SSH** | Proceso hijo del servidor, sin `exec`, que conecta con `libssh` y hace de puente entre el PTY y el canal remoto |
| **Servidor SSH remoto** | Autentica, sirve la shell de login y puede cortar la conexión |
| **Agente SSH** | Opcional. Ofrece identidades por `SSH_AUTH_SOCK` |
| **Quien construye** | Configura y compila tmux en Linux, macOS o FreeBSD, con o sin `--enable-ssh` |

---

## Línea de base de regresión

Medida **antes** de tocar una línea, sobre el commit base. El detalle está en las notas
("Línea de base").

| Plataforma | Build | Suite `regress/*.sh` | Otras referencias |
|---|---|---|---|
| **Linux** (Ubuntu 24.04.5 x86_64) | `./configure --enable-utf8proc --enable-asan`: 0 warnings | **164 PASS, 0 FAIL** con `SHELL=/bin/sh` | `nm tmux` sin símbolos que contengan `ssh`; `list-commands` = 92 líneas |
| **Linux estático** (flags de `tmux-builds`, con glibc) | `./configure --enable-static --enable-utf8proc --disable-jemalloc`: compila, 7 warnings del enlazador de glibc | — | `nm tmux` sin símbolos `ssh` |
| **macOS** 26 arm64 | `./configure --disable-jemalloc`: 0 warnings | **163 PASS, 1 FAIL** (`screen-redraw-menus.sh`, preexistente) | — |
| **FreeBSD** | **Pendiente de medir** (prerrequisito del plan) | Pendiente | Pendiente |

La receta de la suite es la de las notas: desde `regress/`,
`env -i LC_CTYPE=C.UTF-8 MallocNanoZone=0 SHELL=/bin/sh sh -x <test>` para cada `*.sh`.
En macOS no se usa `make -C regress` (hallazgo "runner que no corre nada").

Al cerrar el cambio, cada fila tiene que dar lo mismo, más los tests nuevos. Cualquier
diferencia es una regresión, no un efecto colateral aceptable.

## Entorno de verificación

Los VCs corren contra estos entornos. Construirlos es parte del plan, no de la spec.

- **Host Linux de prueba `L`.** Contenedor o VM con Ubuntu 24.04 x86_64, los paquetes
  de la CI de upstream (`.github/workflows/regress.yml:45-57`) y además `libssh-dev`
  (0.10.6), `openssh-server` (se usa solo `sshd`), `openssh-client` (se usan solo
  `ssh-keygen`, `ssh-agent` y `ssh-add`), `strace`, `iproute2` (`ss`) y
  `netcat-openbsd`. tmux se construye con
  `sh autogen.sh && ./configure --enable-utf8proc --enable-asan --enable-ssh && make`,
  salvo en los VCs de build. La cuenta local `tester` tiene `/bin/sh` como shell de
  login y la contraseña `vc-password`.
- **Claves de prueba.** Pares generados una vez con `ssh-keygen`:
  `K_ed` (ed25519), `K_ec` (ecdsa) y `K_rsa` (rsa), sin passphrase; `K_pass` (ed25519,
  passphrase `vc-pass`); `K_agent` (ed25519, sin passphrase, solo se carga en el
  agente); y `K_unauth` (ed25519, sin passphrase). `/home/tester/.ssh/authorized_keys`
  autoriza todas menos `K_unauth`.
- **sshd de prueba.** `sshd -D -f <cfg> -E /var/log/sshd-vc.log`, corriendo como root
  en `L`, con `ListenAddress 127.0.0.1:22` y `ListenAddress 127.0.0.1:2222`,
  `HostKey` = la clave ed25519 fija `HK_A`, `PubkeyAuthentication yes`,
  `PasswordAuthentication yes`, `KbdInteractiveAuthentication yes`, `AcceptEnv *`,
  `AllowAgentForwarding yes` y `LogLevel VERBOSE` (registra la huella de la clave
  aceptada). `HK_B` es otra clave ed25519 que el servidor **no** usa.
- **Otros puertos.** Listener mudo en `127.0.0.1:2223` (`nc -lk 127.0.0.1 2223`: acepta
  TCP y no envía nada). Nada escucha en `127.0.0.1:2224`.
- **HOME de prueba `$T`.** Un directorio temporal nuevo por VC. La configuración
  estándar de `$T/.ssh/` es: `known_hosts` con las líneas `[127.0.0.1]:2222 <HK_A>` y
  `127.0.0.1 <HK_A>`, e `id_ed25519` = `K_ed` (con su `.pub`), permisos `0600`. Cada VC
  indica lo que cambia. `/etc/ssh/ssh_known_hosts` no existe, salvo en VC-30.
- **`ssh` trampa.** `$T/bin/ssh` es un script que crea `$T/ssh-invoked` y sale con `1`.
- **Servidor tmux de prueba.** `tm` abrevia
  `env -i HOME=$T PATH=$T/bin:/usr/bin:/bin SHELL=/bin/sh LC_CTYPE=C.UTF-8 TERM=screen ./tmux -L vc -f /dev/null`.
  Cada VC arranca un servidor nuevo con `tm new -d -x 120 -y 40` (salvo que diga otra
  cosa). El pane inicial es `%0` y el siguiente que se crea es `%1`.
- **Convenciones de los VCs.**
  - *Remoto `<cmd>`*: `tm send-keys -t %1 '<cmd>' Enter`.
  - *Esperar `<texto>`*: leer `tm capture-pane -p -t %1` cada 0,2 s hasta que contenga
    `<texto>`, hasta 10 s. Si no aparece, el VC falla.
  - *Prueba de remoto*: remoto `echo "R=$SSH_CONNECTION"` y esperar `R=127.0.0.1 `.
    `SSH_CONNECTION` solo existe en una shell abierta por `sshd`: el entorno de `tm` no
    la tiene.
  - *Con remain-on-exit*: se ejecuta `tm set -g remain-on-exit on` antes de
    `ssh-pane`. *Estado del pane*: `tm display -p -t %1 '#{pane_dead} #{pane_dead_status}'`.
  - *Error del comando*: `tm ssh-pane …` sale con código `1`, su stderr es exactamente
    el mensaje indicado, y después `tm list-panes | wc -l` sigue siendo `1`.
  - *Huella de `K_x`*: `ssh-keygen -lf K_x.pub | cut -d' ' -f2`.
- **Hosts no Linux.** macOS 26 arm64 con las dependencias de la línea de base, y una VM
  FreeBSD (versión a fijar al medir su línea de base), ambos con `libssh` ≥ 0.10
  instalada (Homebrew y `pkg`), para mostrar que la plataforma, y no la ausencia de la
  biblioteca, deja el feature afuera.

---

## Requerimientos funcionales

### Build y límite de plataforma

#### FR-1 · Compilar el feature con `--enable-ssh` en Linux

**Dado** un host Linux con `libssh` ≥ 0.10 instalada,
**Cuando** quien construye ejecuta `./configure --enable-ssh` y `make`,
**Entonces** `configure` termina con código `0`, su resumen incluye la línea
`libssh: <versión>`, el binario enlaza `libssh` y `ssh-pane` figura entre los comandos.

> **VC-1** — En `L`, `./configure --enable-utf8proc --enable-asan --enable-ssh` sale
> con código `0` y su salida contiene la línea `configure: libssh: 0.10.6`; `make`
> termina sin warnings; `ldd ./tmux | grep -c 'libssh\.so'` es `1`;
> `./tmux -L vc -f /dev/null list-commands | wc -l` es `93`; y
> `./tmux -L vc -f /dev/null list-commands -F '#{command_list_name}' | grep -cx ssh-pane`
> es `1`.

#### FR-2 · Rechazar `--enable-ssh` fuera de Linux

**Dado** un host que no es Linux, con `libssh` ≥ 0.10 instalada,
**Cuando** quien construye ejecuta `configure` con `--enable-ssh`,
**Entonces** `configure` termina con un código distinto de `0` y su última línea de
error es `configure: error: --enable-ssh is only supported on Linux`.

> **VC-2** — En macOS, `./configure --disable-jemalloc --enable-ssh` (los flags de la
> línea de base más `--enable-ssh`), y en FreeBSD, `./configure --enable-ssh`, salen
> con código distinto de `0`, y stderr contiene exactamente la línea
> `configure: error: --enable-ssh is only supported on Linux`. No se genera ningún
> `Makefile`.

#### FR-3a · Rechazar `--enable-ssh` sin `libssh` en Linux

**Dado** un host Linux en el que `pkg-config` no encuentra el módulo `libssh`,
**Cuando** quien construye ejecuta `./configure --enable-ssh`,
**Entonces** `configure` termina con un código distinto de `0` y su última línea de
error es `configure: error: libssh >= 0.10 not found`.

> **VC-3a** — En `L`, después de `apt-get remove -y libssh-dev`,
> `./configure --enable-utf8proc --enable-ssh` sale con código distinto de `0` y stderr
> contiene exactamente la línea `configure: error: libssh >= 0.10 not found`.

#### FR-3b · Rechazar `--enable-ssh` con una `libssh` anterior a 0.10

**Dado** un host Linux en el que `pkg-config` informa el módulo `libssh` con una
versión menor que 0.10,
**Cuando** quien construye ejecuta `./configure --enable-ssh`,
**Entonces** `configure` termina con un código distinto de `0` con el mismo mensaje de
FR-3a.

> **VC-3b** — En `L`, sin `libssh-dev` y con `PKG_CONFIG_PATH` apuntando a un directorio
> que solo contiene un `libssh.pc` con `Version: 0.9.8`,
> `./configure --enable-utf8proc --enable-ssh` sale con código distinto de `0` y stderr
> contiene exactamente la línea `configure: error: libssh >= 0.10 not found`.

#### FR-4 · Rechazar `--enable-ssh` junto con `--enable-static`

**Dado** un host Linux con `libssh` ≥ 0.10 instalada,
**Cuando** quien construye ejecuta `configure` con `--enable-ssh` y `--enable-static`,
**Entonces** `configure` termina con un código distinto de `0` y su última línea de
error es `configure: error: --enable-ssh cannot be used with --enable-static`.

> **VC-4** — En `L`,
> `./configure --enable-static --enable-utf8proc --disable-jemalloc --enable-ssh` sale
> con código distinto de `0` y stderr contiene exactamente la línea
> `configure: error: --enable-ssh cannot be used with --enable-static`.

#### FR-5 · Tratar `--disable-ssh` como la ausencia del flag

**Dado** un host Linux con `libssh` ≥ 0.10 instalada,
**Cuando** quien construye ejecuta `configure` con `--disable-ssh`,
**Entonces** el resultado es el de no pasar el flag: el resumen dice `libssh: off` y el
binario no tiene el comando.

> **VC-5** — En `L`, `./configure --enable-utf8proc --enable-asan --disable-ssh` sale
> con código `0` y su salida contiene la línea `configure: libssh: off`; después de
> `make`, se cumplen las mismas comprobaciones de VC-INV-3.

### Invocación

Sintaxis: `ssh-pane [-dh] [-p port] [-t target-pane] [user@]host`. Todo lo que sigue
supone un build con el feature.

#### FR-6 · Abrir una shell remota en un pane nuevo

**Dado** un pane existente y un servidor SSH cuya clave está en `known_hosts` y que
acepta una identidad disponible,
**Cuando** la persona ejecuta `ssh-pane -t <pane> -p <port> <user>@<host>`,
**Entonces** el comando termina sin error, se crea un pane nuevo y en él queda abierta
la shell de login remota de `<user>`.

> **VC-6** — `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1` sale con código `0`, con
> stdout y stderr vacíos; `tm list-panes | wc -l` es `2`; la prueba de remoto en `%1`
> muestra una línea que empieza con `R=127.0.0.1 ` y termina con ` 127.0.0.1 2222`.

#### FR-7 · Conectar sin ejecutar ningún programa

**Dado** un pane SSH con la shell remota abierta,
**Cuando** se inspecciona el proceso del pane,
**Entonces** ese proceso es el binario de tmux (no hizo `exec`), no tiene hijos, y el
`ssh` del `PATH` no se ejecutó.

> **VC-7** — Con el escenario de VC-6 y `pid=$(tm display -p -t %1 '#{pane_pid}')`:
> `readlink /proc/$pid/exe` es igual a `readlink -f ./tmux`, `ps -o pid= --ppid $pid`
> no imprime nada, y `$T/ssh-invoked` no existe.

#### FR-8 · Partir hacia abajo por defecto

**Dado** un pane que ocupa toda la ventana,
**Cuando** la persona ejecuta `ssh-pane` sin `-h`,
**Entonces** el pane nuevo queda debajo del pane objetivo, con su mismo ancho.

> **VC-8** — `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; después,
> `tm display -p -t %1 '#{pane_left} #{pane_width} #{pane_top}'` imprime `0 120 <n>`
> con `<n>` mayor que `0`.

#### FR-9 · Partir hacia la derecha con `-h`

**Dado** un pane que ocupa toda la ventana,
**Cuando** la persona ejecuta `ssh-pane -h`,
**Entonces** el pane nuevo queda a la derecha del pane objetivo, con su misma altura.

> **VC-9** — `tm ssh-pane -h -t %0 -p 2222 tester@127.0.0.1`; después,
> `tm display -p -t %1 '#{pane_top} #{pane_height} #{pane_left}'` imprime `0 40 <n>`
> con `<n>` mayor que `0`.

#### FR-10 · Activar el pane nuevo

**Dado** una ventana con un pane activo,
**Cuando** la persona ejecuta `ssh-pane` sin `-d`,
**Entonces** el pane nuevo queda como pane activo de la ventana.

> **VC-10** — `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; después,
> `tm list-panes -F '#{pane_id} #{pane_active}'` imprime exactamente `%0 0` y `%1 1`.

#### FR-11 · No activar el pane nuevo con `-d`

**Dado** una ventana con un pane activo,
**Cuando** la persona ejecuta `ssh-pane -d`,
**Entonces** el pane activo no cambia.

> **VC-11** — `tm ssh-pane -d -t %0 -p 2222 tester@127.0.0.1`; después,
> `tm list-panes -F '#{pane_id} #{pane_active}'` imprime exactamente `%0 1` y `%1 0`.

#### FR-12 · Usar el pane actual si no hay `-t`

**Dado** una ventana con dos panes, uno de ellos activo,
**Cuando** la persona ejecuta `ssh-pane` sin `-t`,
**Entonces** el objetivo es el pane actual, resuelto como en `split-window`
(`CMD_FIND_PANE`).

> **VC-12** — `tm split-window -h -t %0` (crea `%1`, a la derecha); `tm select-pane -t %0`;
> `tm ssh-pane -p 2222 tester@127.0.0.1` (crea `%2`). Después,
> `tm display -p -t %2 '#{pane_left}'` imprime `0` y
> `tm display -p -t %0 '#{pane_height}'` es menor que `40`: se partió `%0`, no `%1`.

#### FR-13 · Usar la cuenta local si no hay `user@`

**Dado** un entorno del pane en el que `USER` tiene otro valor,
**Cuando** la persona ejecuta `ssh-pane` con un destino sin `user@`,
**Entonces** el usuario remoto es el nombre de la cuenta local del proceso
(`getpwuid(getuid())`), no el valor de `USER`.

> **VC-13** — `tm set-environment -g USER nobody`;
> `tm ssh-pane -t %0 -p 2222 127.0.0.1`; remoto `whoami`; esperar `tester`.
> `/var/log/sshd-vc.log` contiene `Accepted publickey for tester`.

#### FR-14 · Usar el puerto 22 si no hay `-p`

**Dado** un servidor SSH que escucha en el puerto 22,
**Cuando** la persona ejecuta `ssh-pane` sin `-p`,
**Entonces** la conexión va al puerto 22.

> **VC-14** — `tm ssh-pane -t %0 tester@127.0.0.1`; la prueba de remoto muestra una
> línea que termina con ` 127.0.0.1 22`.

#### FR-15 · Rechazar un puerto inválido

**Dado** una invocación que en todo lo demás es válida,
**Cuando** el valor de `-p` no es un entero entre 1 y 65535 escrito solo con dígitos
decimales,
**Entonces** el comando falla con `invalid port: "<valor>"`, sin crear ningún pane ni
iniciar ninguna conexión. Un signo, un espacio o un punto invalidan el valor.

> **VC-15** — Para cada uno de `0`, `65536`, `-1`, `+22`, ` 22` (con un espacio
> adelante), `22.0`, `abc` y la cadena vacía, `tm ssh-pane -t %0 -p <valor> tester@127.0.0.1`
> es un error del comando con el mensaje `invalid port: "<valor>"`, y
> `/var/log/sshd-vc.log` no registra ninguna conexión nueva.

#### FR-16 · Rechazar `-p` sin valor

**Dado** una invocación en la que `-p` es el último argumento,
**Cuando** la persona la ejecuta,
**Entonces** el comando falla con el mensaje del parser de argumentos de tmux,
`command ssh-pane: -p expects an argument` (`arguments.c:180`, `cmd.c:534`).

> **VC-16** — `tm ssh-pane -p` es un error del comando con el mensaje
> `command ssh-pane: -p expects an argument`.

#### FR-17 · Rechazar un destino con formato inválido

**Dado** una invocación que en todo lo demás es válida,
**Cuando** el destino no tiene la forma `[user@]host`, con `user` y `host` no vacíos y a
lo sumo un `@`,
**Entonces** el comando falla con `invalid destination: "<destino>"`, sin crear ningún
pane ni iniciar ninguna conexión.

> **VC-17** — Para cada uno de `@127.0.0.1`, `tester@`, `a@b@127.0.0.1` y la cadena
> vacía, `tm ssh-pane -t %0 -p 2222 <destino>` es un error del comando con el mensaje
> `invalid destination: "<destino>"`, y `/var/log/sshd-vc.log` no registra ninguna
> conexión nueva.

#### FR-18a · Rechazar la invocación sin destino

**Dado** una invocación sin argumentos posicionales,
**Cuando** la persona la ejecuta,
**Entonces** el comando falla con `command ssh-pane: too few arguments (need at least 1)`
(`arguments.c:332-334`).

> **VC-18a** — `tm ssh-pane -t %0` es un error del comando con el mensaje
> `command ssh-pane: too few arguments (need at least 1)`.

#### FR-18b · Rechazar más de un destino

**Dado** una invocación con dos o más argumentos posicionales,
**Cuando** la persona la ejecuta,
**Entonces** el comando falla con `command ssh-pane: too many arguments (need at most 1)`
(`arguments.c:338-341`).

> **VC-18b** — `tm ssh-pane -t %0 tester@127.0.0.1 extra` es un error del comando con el
> mensaje `command ssh-pane: too many arguments (need at most 1)`.

#### FR-19 · Rechazar un flag que `ssh-pane` no tiene

**Dado** una invocación con un flag que no es `-d`, `-h`, `-p` ni `-t`,
**Cuando** la persona la ejecuta,
**Entonces** el comando falla con `command ssh-pane: unknown flag -<letra>`
(`arguments.c:240`).

> **VC-19** — `tm ssh-pane -v -t %0 tester@127.0.0.1` y
> `tm ssh-pane -l 10 -t %0 tester@127.0.0.1` son errores del comando con los mensajes
> `command ssh-pane: unknown flag -v` y `command ssh-pane: unknown flag -l`,
> respectivamente.

#### FR-20 · Rechazar un pane objetivo inexistente

**Dado** un `-t` que no identifica ningún pane,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el comando falla con el mensaje de la resolución de targets,
`can't find pane: <target>` (`cmd-find.c:1272`).

> **VC-20** — `tm ssh-pane -t %99 -p 2222 tester@127.0.0.1` es un error del comando con
> el mensaje `can't find pane: %99`.

#### FR-21 · Rechazar cuando no hay espacio para el pane

**Dado** un pane objetivo demasiado chico para partirlo,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el comando falla con el mismo mensaje que `split-window` en la misma
situación, `no space for a new pane` (`layout.c:1692`), sin iniciar ninguna conexión.

> **VC-21** — Con un servidor arrancado con `tm new -d -x 80 -y 2`,
> `tm split-window -t %0` y `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1` son errores del
> comando con el mismo mensaje, `no space for a new pane`, y `/var/log/sshd-vc.log` no
> registra ninguna conexión nueva.

#### FR-22 · Reportar un solo error de invocación

**Dado** una invocación con más de un error de los que cubren FR-15 a FR-21,
**Cuando** la persona la ejecuta,
**Entonces** el comando informa uno solo, sale con código `1` y no crea ningún pane.
Cuál de los errores se informa no está especificado.

> **VC-22** — `tm ssh-pane -t %99 -p 0 @127.0.0.1` sale con código `1`, stderr es
> exactamente una línea, y `tm list-panes -a | wc -l` sigue siendo `1`.

### Verificación del host

En esta sección, `<dir-ssh>` es el de BR-3a y BR-3b; en los VCs, `$T/.ssh`.

#### FR-23 · Rechazar un host que no está en `known_hosts`

**Dado** un `<dir-ssh>/known_hosts` sin ninguna entrada para el host y el puerto,
**Cuando** la persona ejecuta `ssh-pane` hacia ese destino,
**Entonces** el hijo no se autentica, escribe
`ssh-pane: <host>:<port>: host key not found in known_hosts` en el pane y termina según
BR-5.

> **VC-23** — Con remain-on-exit y `$T/.ssh/known_hosts` con una sola línea,
> `[10.0.0.1]:2222 <HK_A>`, `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`: esperar
> `ssh-pane: 127.0.0.1:2222: host key not found in known_hosts`; el estado del pane es
> `1 255`; `/var/log/sshd-vc.log` no contiene `Accepted` ni `Failed` para esa conexión.

#### FR-24 · Rechazar cuando no existe `known_hosts`

**Dado** un `<dir-ssh>` sin archivo `known_hosts`,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el hijo termina con el mismo mensaje de FR-23 y el archivo sigue sin
existir.

> **VC-24** — Con remain-on-exit y sin `$T/.ssh/known_hosts`,
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`: esperar
> `ssh-pane: 127.0.0.1:2222: host key not found in known_hosts`; el estado del pane es
> `1 255`; `$T/.ssh/known_hosts` no existe.

#### FR-25 · Rechazar una clave de host distinta de la registrada

**Dado** un `known_hosts` con una entrada para el host y el puerto cuya clave no es la
que presenta el servidor,
**Cuando** la persona ejecuta `ssh-pane` hacia ese destino,
**Entonces** el hijo no se autentica, escribe
`ssh-pane: <host>:<port>: host key does not match known_hosts` y termina según BR-5.

> **VC-25** — Con remain-on-exit y `$T/.ssh/known_hosts` con la sola línea
> `[127.0.0.1]:2222 <HK_B>`, `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`: esperar
> `ssh-pane: 127.0.0.1:2222: host key does not match known_hosts`; el estado del pane
> es `1 255`.

#### FR-26 · No usar una entrada sin puerto para otro puerto

**Dado** un `known_hosts` que tiene la clave correcta solo en una entrada sin puerto
(`host <clave>`, que vale para el puerto 22),
**Cuando** la persona ejecuta `ssh-pane` hacia ese host en un puerto distinto de 22,
**Entonces** el host se trata como desconocido, según FR-23.

> **VC-26** — Con remain-on-exit y `$T/.ssh/known_hosts` con la sola línea
> `127.0.0.1 <HK_A>`, `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`: esperar
> `ssh-pane: 127.0.0.1:2222: host key not found in known_hosts`; el estado del pane es
> `1 255`.

#### FR-27 · Reconocer entradas hasheadas

**Dado** un `known_hosts` cuyas entradas están hasheadas (formato de
`HashKnownHosts yes`),
**Cuando** la persona ejecuta `ssh-pane` hacia un host registrado,
**Entonces** la clave se reconoce y la conexión sigue como en FR-6.

> **VC-27** — Con la configuración estándar de `$T/.ssh` después de
> `ssh-keygen -H -f $T/.ssh/known_hosts` (ninguna línea conserva `127.0.0.1` en claro),
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1` y la prueba de remoto muestran
> `R=127.0.0.1 `.

#### FR-28 · No consultar el `known_hosts` global

**Dado** un host cuya clave solo figura en `/etc/ssh/ssh_known_hosts`,
**Cuando** la persona ejecuta `ssh-pane` hacia ese host,
**Entonces** el host se trata como desconocido, según FR-23.

> **VC-28** — Con remain-on-exit, `/etc/ssh/ssh_known_hosts` con la línea
> `[127.0.0.1]:2222 <HK_A>` y `$T/.ssh/known_hosts` vacío,
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`: esperar
> `ssh-pane: 127.0.0.1:2222: host key not found in known_hosts`; el estado del pane es
> `1 255`.

### Autenticación

#### FR-29 · Autenticarse con el agente

**Dado** un `SSH_AUTH_SOCK` en el entorno del pane que apunta a un agente con una
identidad autorizada, y ninguna clave en `<dir-ssh>`,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el hijo se autentica con esa identidad y abre la shell remota.

> **VC-29** — `$T/.ssh` sin `id_*`; `ssh-agent -a $T/agent.sock` y
> `SSH_AUTH_SOCK=$T/agent.sock ssh-add K_agent`;
> `tm set-environment -g SSH_AUTH_SOCK $T/agent.sock`;
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; la prueba de remoto muestra
> `R=127.0.0.1 `, y `/var/log/sshd-vc.log` contiene `Accepted publickey for tester` con
> la huella de `K_agent`.

#### FR-30 · Autenticarse con una clave por defecto

**Dado** un entorno del pane sin `SSH_AUTH_SOCK` y un `<dir-ssh>/id_ed25519` sin
passphrase y autorizado,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el hijo se autentica con esa clave y abre la shell remota.

> **VC-30** — Con la configuración estándar de `$T/.ssh`,
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; la prueba de remoto muestra
> `R=127.0.0.1 `, y `/var/log/sshd-vc.log` contiene `Accepted publickey for tester` con
> la huella de `K_ed`.

#### FR-31 · Probar el agente antes que las claves

**Dado** un agente con una identidad autorizada y un `<dir-ssh>/id_ed25519` también
autorizado,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** la identidad aceptada es la del agente.

> **VC-31** — Con la configuración estándar de `$T/.ssh` y el agente de VC-29,
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; después de la prueba de remoto,
> `/var/log/sshd-vc.log` contiene `Accepted publickey for tester` con la huella de
> `K_agent`, y no la de `K_ed`.

#### FR-32a · Probar las claves por defecto en orden

**Dado** un entorno sin agente y `<dir-ssh>/id_ed25519`, `id_ecdsa` e `id_rsa`, las tres
sin passphrase y autorizadas,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** la clave aceptada es `id_ed25519`, la primera del orden
`id_ed25519`, `id_ecdsa`, `id_rsa`.

> **VC-32a** — `$T/.ssh` con `id_ed25519` = `K_ed`, `id_ecdsa` = `K_ec` e
> `id_rsa` = `K_rsa`; `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; después de la prueba
> de remoto, `/var/log/sshd-vc.log` contiene `Accepted publickey for tester` con la huella
> de `K_ed`.

#### FR-32b · Seguir con la próxima clave si el servidor rechaza una

**Dado** un entorno sin agente, un `<dir-ssh>/id_ed25519` que el servidor no autoriza y
un `<dir-ssh>/id_ecdsa` autorizado,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el hijo se autentica con `id_ecdsa`.

> **VC-32b** — `$T/.ssh` con `id_ed25519` = `K_unauth` e `id_ecdsa` = `K_ec`;
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; después de la prueba de remoto,
> `/var/log/sshd-vc.log` contiene `Accepted publickey for tester` con la huella de
> `K_ec`.

#### FR-33 · Saltear una clave con passphrase sin pedirla

**Dado** un entorno sin agente, un `<dir-ssh>/id_ed25519` con passphrase y un
`<dir-ssh>/id_ecdsa` sin passphrase y autorizado,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el hijo no pide la passphrase, saltea `id_ed25519` y se autentica con
`id_ecdsa`.

> **VC-33** — `$T/.ssh` con `id_ed25519` = `K_pass` e `id_ecdsa` = `K_ec`;
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; la prueba de remoto muestra
> `R=127.0.0.1 ` sin que se haya enviado ninguna tecla al pane antes;
> `tm capture-pane -p -S - -t %1 | grep -ci passphrase` es `0`; y
> `/var/log/sshd-vc.log` contiene `Accepted publickey for tester` con la huella de
> `K_ec`.

#### FR-34 · Fallar si no hay ninguna identidad aceptada

**Dado** un entorno sin agente y sin ninguna clave por defecto que el servidor acepte,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el hijo escribe
`ssh-pane: <user>@<host>:<port>: authentication failed` en el pane y termina según
BR-5.

> **VC-34** — Con remain-on-exit y `$T/.ssh` con `known_hosts` estándar e
> `id_ed25519` = `K_unauth`, `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`: esperar
> `ssh-pane: tester@127.0.0.1:2222: authentication failed`; el estado del pane es
> `1 255`.

#### FR-35 · Ignorar un `SSH_AUTH_SOCK` que no responde

**Dado** un `SSH_AUTH_SOCK` en el entorno del pane que apunta a una ruta sin agente, y
un `<dir-ssh>/id_ed25519` autorizado,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el hijo saltea el agente sin avisar y se autentica con la clave.

> **VC-35** — Con la configuración estándar de `$T/.ssh`,
> `tm set-environment -g SSH_AUTH_SOCK $T/no-such.sock`;
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; la prueba de remoto muestra
> `R=127.0.0.1 `, y `/var/log/sshd-vc.log` contiene la huella de `K_ed`.

### Sesión remota

#### FR-36 · Pedir el PTY remoto con el `TERM` del pane

**Dado** un valor de `default-terminal`,
**Cuando** la persona abre un `ssh-pane`,
**Entonces** en la shell remota `TERM` vale lo mismo que `default-terminal`.

> **VC-36** — `tm set -g default-terminal xterm-256color`;
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; remoto `echo "T=$TERM"`; esperar
> `T=xterm-256color`.

#### FR-37 · Abrir el PTY remoto con el tamaño del pane

**Dado** un pane nuevo de `W` columnas y `H` filas,
**Cuando** se abre la shell remota,
**Entonces** el PTY remoto mide `H` filas y `W` columnas.

> **VC-37** — `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; remoto `stty size`; esperar
> la línea que imprime `tm display -p -t %1 '#{pane_height} #{pane_width}'`.

#### FR-38 · Propagar el cambio de tamaño

**Dado** un pane SSH con la shell remota abierta,
**Cuando** el pane cambia de tamaño,
**Entonces** el PTY remoto toma el tamaño nuevo.

> **VC-38** — Con el escenario de VC-37, `tm resize-pane -t %1 -y 10`; remoto
> `stty size`; esperar `10 120`.

#### FR-39 · No enviar variables de entorno

**Dado** una variable definida en el entorno del pane y un servidor que acepta
cualquier variable (`AcceptEnv *`),
**Cuando** la persona abre un `ssh-pane`,
**Entonces** esa variable no existe en la shell remota.

> **VC-39** — `tm set-environment -g VC_VAR hola`;
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; remoto `echo "V=${VC_VAR-unset}"`;
> esperar `V=unset`.

#### FR-40 · No reenviar el agente

**Dado** un agente disponible en el entorno del pane y un servidor que permite el
reenvío de agente,
**Cuando** la persona abre un `ssh-pane`,
**Entonces** la shell remota no tiene `SSH_AUTH_SOCK`.

> **VC-40** — Con el escenario de VC-29, remoto `echo "A=${SSH_AUTH_SOCK-unset}"`;
> esperar `A=unset`.

#### FR-41 · Mandar los caracteres de control al remoto

**Dado** un pane SSH con un proceso remoto en primer plano,
**Cuando** la persona envía `C-c`,
**Entonces** se interrumpe el proceso remoto y el pane sigue vivo con la shell remota:
el PTY local está en modo crudo y no convierte `C-c` en una señal para el hijo.

> **VC-41** — Con el escenario de VC-6, remoto `sleep 100`; `tm send-keys -t %1 C-c`;
> remoto `echo after-cc`; esperar `after-cc` en menos de 10 s desde el `C-c`; y
> `tm display -p -t %1 '#{pane_dead}'` imprime `0`.

### Ciclo de vida

#### FR-42 · No esperar la conexión para terminar el comando

**Dado** un destino que acepta la conexión TCP y no responde,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el comando termina con éxito antes de que se resuelva la conexión, y el
pane nuevo existe y está vivo.

> **VC-42** — `time tm ssh-pane -t %0 -p 2223 tester@127.0.0.1` sale con código `0` en
> menos de 1 s, y en seguida `tm display -p -t %1 '#{pane_dead}'` imprime `0`.

#### FR-43 · Terminar con el código de la shell remota

**Dado** un pane SSH con la shell remota abierta y `remain-on-exit` en `on`,
**Cuando** la shell remota termina con el código `N`,
**Entonces** el pane queda muerto con `pane_dead_status` igual a `N`.

> **VC-43** — Con remain-on-exit, `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; prueba
> de remoto; remoto `exit 3`; en menos de 5 s, el estado del pane es `1 3`.

#### FR-44 · Cerrar el pane si `remain-on-exit` está en `off`

**Dado** un pane SSH con la shell remota abierta y `remain-on-exit` en `off`,
**Cuando** la shell remota termina,
**Entonces** el pane se cierra como cualquier pane cuyo proceso termina.

> **VC-44** — `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; prueba de remoto; remoto
> `exit 0`; en menos de 5 s, `tm list-panes -F '#{pane_id}'` imprime solo `%0`.

#### FR-45 · Informar una conexión perdida

**Dado** un pane SSH con la shell remota abierta,
**Cuando** la conexión se corta sin que la shell remota informe un código de salida,
**Entonces** el hijo escribe `ssh-pane: <host>:<port>: connection lost` en el pane y
termina según BR-5.

> **VC-45** — Con remain-on-exit, `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; prueba
> de remoto; `pkill -KILL -f '^sshd: tester'` (mata los procesos de `sshd` de esa
> sesión); esperar `ssh-pane: 127.0.0.1:2222: connection lost`; el estado del pane es
> `1 255`.

#### FR-46 · Cerrar la conexión al matar el pane

**Dado** un pane SSH con la shell remota abierta,
**Cuando** la persona ejecuta `kill-pane` sobre él,
**Entonces** el hijo cierra la conexión y termina, y del lado remoto no queda ninguna
sesión.

> **VC-46** — Con el escenario de VC-6, `tm kill-pane -t %1`; en menos de 5 s,
> `ss -Htn state established '( dport = :2222 )' | wc -l` es `0`,
> `pgrep -f '^sshd: tester'` no imprime nada, y el proceso `pane_pid` leído antes del
> `kill-pane` ya no existe.

#### FR-47 · Abandonar la conexión a los 30 s

**Dado** un destino que acepta la conexión TCP y no responde,
**Cuando** pasan 30 s desde que empezó el hijo sin que se abra la shell remota,
**Entonces** el hijo escribe `ssh-pane: <host>:<port>: timed out after 30s` en el pane
y termina según BR-5.

> **VC-47** — Con remain-on-exit, `tm ssh-pane -t %0 -p 2223 tester@127.0.0.1`: antes de
> 30 s, el estado del pane es `0 ` (vivo); a los 35 s, el pane muestra
> `ssh-pane: 127.0.0.1:2223: timed out after 30s` y su estado es `1 255`.

#### FR-48a · Informar una conexión rechazada

**Dado** un destino en el que nada escucha en el puerto,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el hijo escribe `ssh-pane: <host>:<port>: connection failed: <detalle>` en
el pane y termina según BR-5.

> **VC-48a** — Con remain-on-exit, `tm ssh-pane -t %0 -p 2224 tester@127.0.0.1`: en
> menos de 5 s, el pane muestra una línea que empieza con
> `ssh-pane: 127.0.0.1:2224: connection failed: ` y su estado es `1 255`.

#### FR-48b · Informar un host que no resuelve

**Dado** un nombre de host que no resuelve,
**Cuando** la persona ejecuta `ssh-pane` hacia él,
**Entonces** el hijo escribe `ssh-pane: <host>:<port>: connection failed: <detalle>` en
el pane y termina según BR-5.

> **VC-48b** — Con remain-on-exit, `tm ssh-pane -t %0 tester@vc-no-such-host.invalid`:
> en menos de 5 s, el pane muestra una línea que empieza con
> `ssh-pane: vc-no-such-host.invalid:22: connection failed: ` y su estado es `1 255`.

#### FR-49a · Reconectar un pane SSH vivo con `respawn-pane -k`

**Dado** un pane SSH con la shell remota abierta,
**Cuando** la persona ejecuta `respawn-pane -k` sin comando sobre él,
**Entonces** el pane vuelve a conectar al mismo destino con una conexión nueva.

> **VC-49a** — Con el escenario de VC-6, se anotan `pane_pid` y la línea `R=…`;
> `tm respawn-pane -k -t %1`; la prueba de remoto muestra una línea `R=…` con otro
> puerto de origen (segundo campo) y el mismo destino (` 127.0.0.1 2222`), y
> `pane_pid` cambió.

#### FR-49b · Reconectar un pane SSH muerto con `respawn-pane`

**Dado** un pane SSH muerto (con `remain-on-exit` en `on`),
**Cuando** la persona ejecuta `respawn-pane` sin comando sobre él,
**Entonces** el pane vuelve a conectar al mismo destino.

> **VC-49b** — Con remain-on-exit, `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; remoto
> `exit 0`; esperar el estado `1 0`; `tm respawn-pane -t %1`; la prueba de remoto muestra
> una línea que termina con ` 127.0.0.1 2222`.

#### FR-49c · Correr como local un pane SSH respawneado con comando

**Dado** un pane SSH,
**Cuando** la persona ejecuta `respawn-pane -k` con un comando sobre él,
**Entonces** el comando corre como en cualquier pane local: el hijo hace `exec` y no
hay conexión SSH.

> **VC-49c** — Con el escenario de VC-6,
> `tm respawn-pane -k -t %1 'echo local-cmd; sleep 100'`; esperar `local-cmd`;
> `readlink /proc/$(tm display -p -t %1 '#{pane_pid}')/exe` es `/usr/bin/dash`, y en
> menos de 5 s `ss -Htn state established '( dport = :2222 )' | wc -l` es `0`.

#### FR-49d · No reconectar un pane que dejó de ser SSH

**Dado** un pane que era SSH y fue respawneado con un comando (FR-49c),
**Cuando** la persona ejecuta `respawn-pane -k` sin comando sobre él,
**Entonces** se vuelve a ejecutar ese comando local, no la conexión SSH.

> **VC-49d** — Después de VC-49c, `tm clear-history -t %1`; `tm respawn-pane -k -t %1`;
> esperar `local-cmd`; `/var/log/sshd-vc.log` no registra ninguna conexión nueva.

#### FR-50 · Conservar la conexión al mover el pane

**Dado** un pane SSH con la shell remota abierta,
**Cuando** la persona lo mueve a otra ventana con `break-pane`,
**Entonces** la conexión sigue abierta y la shell remota es la misma.

> **VC-50** — Con el escenario de VC-6, se anota la línea `R=…`;
> `tm break-pane -d -s %1`; `tm send-keys -t %1 'echo "R2=$SSH_CONNECTION"' Enter`;
> esperar una línea `R2=…` cuyo valor es igual al de `R=…`.

### Eventos y formatos

#### FR-51 · Informar el destino en `pane-created`

**Dado** un hook `pane-created`,
**Cuando** se crea un pane con `ssh-pane`,
**Entonces** el hook recibe como `pane_command` la forma normalizada
`ssh-pane -p <port> <user>@<host>`, con el usuario y el puerto ya resueltos, y
`created_empty` = `0`. El evento no acredita que la conexión se haya establecido
(hallazgo 2).

> **VC-51** — `tm set -g @pc 0`;
> `tm set-hook -g pane-created 'set -gF @pc "#{hook_pane}:#{hook_pane_command}:#{hook_created_empty}"'`;
> `tm ssh-pane -t %0 127.0.0.1`; en menos de 5 s, `tm show -gv @pc` imprime
> `%1:ssh-pane -p 22 tester@127.0.0.1:0`.

#### FR-52 · Informar `tmux` como comando actual

**Dado** un pane SSH con la shell remota abierta,
**Cuando** se expande `pane_current_command`,
**Entonces** el valor es `tmux`, el nombre del proceso local del pane (D-14).

> **VC-52** — Con el escenario de VC-6, `tm display -p -t %1 '#{pane_current_command}'`
> imprime `tmux`.

#### FR-53 · Informar el directorio local como directorio actual

**Dado** un pane SSH con la shell remota abierta,
**Cuando** la shell remota cambia de directorio y se expande `pane_current_path`,
**Entonces** el valor sigue siendo el directorio local con el que se creó el pane
(D-14).

> **VC-53** — Con un servidor arrancado con `tm new -d -x 120 -y 40 -c /tmp`,
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; prueba de remoto; remoto `cd /usr`;
> `tm display -p -t %1 '#{pane_current_path}'` imprime `/tmp`.

#### FR-54 · No disparar `after-split-window`

**Dado** un hook `after-split-window`,
**Cuando** la persona ejecuta `ssh-pane`,
**Entonces** el hook no se ejecuta: `ssh-pane` no es `split-window`.

> **VC-54** — `tm set -g @as 0`; `tm set-hook -g after-split-window 'set -g @as 1'`;
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1`; después de la prueba de remoto,
> `tm show -gv @as` imprime `0`. Con `tm split-window -t %0` imprime `1`.

---

## Reglas de negocio

### BR-1 · El servidor no hace red ni toca credenciales

El proceso del servidor tmux no abre conexiones de red, no resuelve nombres, no lee
archivos de `<dir-ssh>` y no se conecta al agente. Todo eso lo hace el hijo del pane.

*Fundamento:* con un solo loop y sin hilos (hallazgo 3), cualquier espera de red en el
servidor congela todas las sesiones. Además, las credenciales quedan fuera del proceso
que atiende a todos los clientes (D-1, D-9).
*Excepciones:* ninguna.

> **VC-BR-1** — Con el escenario de VC-31 (agente y clave), y con
> `strace -o s.txt -e trace=connect,openat -p <pid del servidor>` corriendo (sin `-f`:
> solo el servidor, no sus hijos) mientras se ejecuta `ssh-pane` hasta la prueba de
> remoto: `s.txt` no contiene `AF_INET`, `AF_INET6`, `agent.sock`, `/.ssh/` ni
> `/etc/hosts`. Además, `ss -Htnp state established '( dport = :2222 )'` muestra la
> conexión con el `pid` del pane, no el del servidor.

### BR-2 · Nada interactivo antes de la shell remota

Antes de abrir la shell remota, el hijo no lee nada del PTY ni escribe nada que no
sean los mensajes de falla. Nunca intenta autenticarse con contraseña ni con
*keyboard-interactive*, aunque el servidor los ofrezca.

*Fundamento:* un secreto tipeado en el pane pasaría por un PTY cuya salida el servidor
puede copiar a `pipe-pane` y a los clientes de control (hallazgo 3). El agente cubre
las claves con passphrase (D-9).
*Excepciones:* ninguna.

> **VC-BR-2** — En el escenario de VC-34 (el servidor ofrece `password` y
> `keyboard-interactive`), el pane muestra el mensaje de falla sin que se le haya
> enviado ninguna tecla, `tm capture-pane -p -S - -t %1 | grep -ci password` es `0`, y
> las líneas de `/var/log/sshd-vc.log` de esa conexión no contienen `password` ni
> `keyboard-interactive`.

### BR-3a · El directorio SSH sale del entorno del pane

`<dir-ssh>` es `$HOME/.ssh`, con `HOME` del entorno del pane (el de
`environ_for_session`, `environ.c:251-269`), no el del proceso del servidor.

*Fundamento:* el hijo ya tiene el entorno del pane instalado (`spawn.c:544`), y el
entorno de prueba puede aislar `known_hosts` y las claves (D-10).
*Excepciones:* BR-3b, si el entorno del pane no define `HOME`.

> **VC-BR-3a** — Con `$T2` vacío, `tm set-environment -g HOME $T2` y remain-on-exit,
> `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1` muestra
> `ssh-pane: 127.0.0.1:2222: host key not found in known_hosts`, aunque `$T/.ssh`
> (el `HOME` con el que arrancó el servidor) tiene la configuración estándar.

### BR-3b · Sin `HOME`, el directorio SSH es el de la cuenta local

Si el entorno del pane no define `HOME`, `<dir-ssh>` es el directorio `.ssh` del home
de la cuenta local según `getpwuid(getuid())`.

*Fundamento:* es el mismo respaldo que usa tmux para la shell por defecto (D-10).
*Excepciones:* ninguna.

> **VC-BR-3b** — Con `/home/tester/.ssh/` en la configuración estándar y
> `tm set-environment -gu HOME`, `tm ssh-pane -t %0 -p 2222 tester@127.0.0.1` y la
> prueba de remoto muestran `R=127.0.0.1 `.

### BR-4 · `known_hosts` es de solo lectura

El comando nunca crea, modifica ni borra `<dir-ssh>/known_hosts`.

*Fundamento:* aceptar una clave es una decisión de la persona, no del comando (D-8).
*Excepciones:* ninguna.

> **VC-BR-4** — En VC-6, VC-23, VC-25, VC-26 y VC-27, `sha256sum` y `stat -c %Y` de
> `$T/.ssh/known_hosts` son iguales antes y después; en VC-24 el archivo no existe
> después.

### BR-5 · Cómo termina el hijo ante una falla

Toda falla de conexión, de verificación del host o de autenticación, y toda pérdida de
la conexión, termina así: el hijo escribe una sola línea que empieza con `ssh-pane: `
en el PTY y sale con código `255`. Después, el pane sigue `remain-on-exit` como
cualquier otro (`server-fn.c:354-435`). El comando `ssh-pane` ya había terminado con
éxito (FR-42).

*Fundamento:* no cambia el ciclo de vida de los panes (D-12). El código `255` es el
que usa `ssh`, así que un script distingue una falla de la conexión de una salida
normal de la shell remota (FR-43).
*Excepciones:* ninguna.

> **VC-BR-5** — En VC-23, VC-24, VC-25, VC-26, VC-28, VC-34, VC-45, VC-47, VC-48a y
> VC-48b, el estado del pane es `1 255` y `tm capture-pane -p -t %1` tiene exactamente
> una línea que empieza con `ssh-pane: `. Sin remain-on-exit, el mismo escenario de
> VC-48a deja solo `%0` en `tm list-panes -F '#{pane_id}'` en menos de 5 s.

---

## Invariantes

Cosas que tienen que seguir siendo verdad **después** del cambio. Varios invariantes se
verifican a mano: la CI de upstream no corre en forks, no corre en PRs y no cubre
FreeBSD ni `--enable-ssh` (hallazgo 11).

### INV-1 · macOS sigue compilando, sin el feature

> **VC-INV-1** — En macOS, `sh autogen.sh && ./configure --disable-jemalloc && make`
> termina sin warnings y la salida de `configure` contiene `configure: libssh: off`;
> `nm ./tmux | grep -ci ssh` es `0`; `./tmux -L vc -f /dev/null list-commands | wc -l`
> es `92`; `./tmux -L vc -f /dev/null ssh-pane x` sale con código `1` y stderr
> `unknown command: ssh-pane`; y la suite da **163 PASS, 1 FAIL**
> (`screen-redraw-menus.sh`), igual que la línea de base.

### INV-2 · FreeBSD sigue compilando, sin el feature

> **VC-INV-2** — En FreeBSD, `sh autogen.sh && ./configure && make` da el mismo
> resultado que su línea de base (a medir antes de implementar), la salida de
> `configure` contiene `configure: libssh: off`, `nm ./tmux | grep -ci ssh` es `0`,
> `list-commands` da `92` líneas y `ssh-pane` es `unknown command: ssh-pane`.

### INV-3 · Linux sin `--enable-ssh` queda como antes, aunque `libssh` esté instalada

> **VC-INV-3** — En `L`, con `libssh-dev` instalada,
> `./configure --enable-utf8proc --enable-asan && make` termina sin warnings; la salida
> de `configure` contiene `configure: libssh: off`; `ldd ./tmux | grep -c libssh` es
> `0`; `nm ./tmux | grep -ci ssh` es `0`; `list-commands` da `92` líneas; y
> `./tmux -L vc -f /dev/null ssh-pane x` sale con código `1` y stderr
> `unknown command: ssh-pane`.

### INV-4 · Los panes normales siguen haciendo `exec`

En un build con el feature, los panes que no son SSH se crean exactamente como antes:
el hijo hace `exec` de la shell o del comando (`spawn.c:546-575`).

> **VC-INV-4** — En `L` con `--enable-ssh`, `tm split-window -t %0` y
> `tm split-window -t %0 'sleep 100'`: `readlink /proc/<pane_pid>/exe` es `/usr/bin/dash`
> (la shell `/bin/sh`) y `/usr/bin/sleep`, respectivamente. `tm respawn-pane -k -t %1`
> sobre un pane creado con `split-window` vuelve a lanzar la shell local.

### INV-5 · La suite existente pasa en Linux con el feature

> **VC-INV-5** — En `L` con `--enable-utf8proc --enable-asan --enable-ssh`, la suite
> `regress/*.sh` con la receta de la línea de base da **164 PASS** en los 164 tests
> existentes, sin errores de AddressSanitizer, además de los tests nuevos de `ssh-pane`.

### INV-6 · Los tests nuevos se saltean sin el feature

Cada `regress/ssh-pane-*.sh` comprueba primero si el binario tiene `ssh-pane` y, si no
lo tiene, termina con código `0` sin ejecutar nada más. Así no rompe la suite en macOS,
FreeBSD ni en Linux sin `--enable-ssh`, ni la CI de upstream.

> **VC-INV-6** — En los builds de VC-INV-1, VC-INV-2 y VC-INV-3, cada
> `regress/ssh-pane-*.sh`, corrido con la receta de la línea de base, sale con código
> `0` en menos de 5 s y no intenta ninguna conexión TCP a `127.0.0.1:2222`.

### INV-7 · Las abreviaturas de los comandos existentes no cambian

Toda abreviatura que hoy resuelve a un comando lo sigue resolviendo en el build con el
feature (hallazgo 12).

> **VC-INV-7** — Para cada comando de `list-commands` de la línea de base y cada
> prefijo suyo que hoy es único, se ejecuta `tm <prefijo> -@` en el binario de la línea
> de base y en el de `--enable-ssh`. stderr es idéntico en ambos
> (`command <nombre>: invalid flag -@`, `arguments.c:233-235`, para el mismo `<nombre>`). En el de
> `--enable-ssh`, ningún caso da `ambiguous command`.

### INV-8 · El build estático oficial no cambia

> **VC-INV-8** — En `L`, con `libssh-dev` instalada,
> `./configure --enable-static --enable-utf8proc --disable-jemalloc && make` da un
> binario estático con los mismos 7 warnings del enlazador de la línea de base, la
> salida de `configure` contiene `configure: libssh: off`, y `nm ./tmux | grep -ci ssh`
> es `0`.

---

## Requerimientos no funcionales

### NFR-1 · El servidor no se bloquea mientras un pane conecta

Mientras un `ssh-pane` intenta conectar a un destino que no responde, los demás panes
del servidor responden a comandos en **≤ 1 s**.

Condiciones: `L`, servidor de prueba con un pane `%0` con `/bin/sh` y un `ssh-pane`
hacia `127.0.0.1:2223` (listener mudo), durante los 30 s del plazo de conexión.

*Fundamento del umbral:* si la conexión corriera en el servidor, cada comando esperaría
hasta el fin del plazo (30 s). Un umbral de 1 s está 30 veces por debajo de ese
bloqueo y muy por encima del tiempo normal de un `capture-pane`, así que separa sin
ambigüedad "bloqueado" de "no bloqueado" sin depender de la carga de la máquina.

> **VC-NFR-1** — Después de `tm ssh-pane -t %0 -p 2223 tester@127.0.0.1`, se repite
> 10 veces, a 2 s de distancia: `tm send-keys -t %0 'echo alive-<i>' Enter` y se mide el
> tiempo hasta que `tm capture-pane -p -t %0` contiene `alive-<i>`. Las 10 mediciones
> son ≤ 1 s y ocurren mientras el estado de `%1` es `0 ` (vivo).

### NFR-2 · Cota de tiempo ante un destino que no responde

Contra un destino que acepta TCP y no responde, el pane SSH termina **entre 30 y 35 s**
después de que se ejecuta `ssh-pane`, con el mensaje de FR-47.

> **VC-NFR-2** — Con remain-on-exit, se registra el instante `t0` antes de
> `tm ssh-pane -t %0 -p 2223 tester@127.0.0.1` y se lee el estado de `%1` cada 0,2 s
> hasta que empieza con `1`. El tiempo transcurrido desde `t0` está entre 30 y 35 s, en
> tres corridas.

---

## Tabla de trazabilidad

| Requerimiento | Origen | VC | Camino |
|---|---|---|---|
| FR-1 | Enunciado + D-2, D-3 | VC-1 | feliz (build) |
| FR-2 | Enunciado ("solo Linux") + D-3 | VC-2 | falla (build) |
| FR-3a | D-3 | VC-3a | falla (build) |
| FR-3b | D-2, D-3 | VC-3b | borde (versión) |
| FR-4 | D-3 + hallazgos 9 y 11 | VC-4 | falla (build) |
| FR-5 | D-3 | VC-5 | borde (build) |
| FR-6 | Enunciado + D-1, D-7 | VC-6 | feliz |
| FR-7 | Enunciado ("sin invocar `ssh`") + D-1 | VC-7 | invariante |
| FR-8 | D-7 | VC-8 | feliz |
| FR-9 | D-7 | VC-9 | feliz |
| FR-10 | D-7 | VC-10 | feliz |
| FR-11 | D-7 | VC-11 | feliz |
| FR-12 | D-7 | VC-12 | feliz |
| FR-13 | D-7 | VC-13 | borde (`USER`) |
| FR-14 | D-7 | VC-14 | feliz |
| FR-15 | D-7 | VC-15 | falla |
| FR-16 | D-7 | VC-16 | falla (sin valor) |
| FR-17 | D-7 | VC-17 | falla |
| FR-18a | D-7 | VC-18a | falla |
| FR-18b | D-7 | VC-18b | falla |
| FR-19 | D-7 | VC-19 | falla |
| FR-20 | Hallazgo 1 | VC-20 | falla |
| FR-21 | Hallazgo 1 | VC-21 | falla |
| FR-22 | D-7 | VC-22 | falla |
| FR-23 | D-8 | VC-23 | falla (host desconocido) |
| FR-24 | D-8 | VC-24 | borde (sin archivo) |
| FR-25 | D-8 | VC-25 | falla (clave cambiada) |
| FR-26 | D-8 | VC-26 | borde (puerto) |
| FR-27 | D-8 | VC-27 | borde (hasheado) |
| FR-28 | D-8 | VC-28 | borde (archivo global) |
| FR-29 | Enunciado ("auth por agent") + D-9 | VC-29 | feliz |
| FR-30 | Enunciado ("auth por claves") + D-9 | VC-30 | feliz |
| FR-31 | D-9 | VC-31 | borde (orden) |
| FR-32a | D-9 | VC-32a | borde (orden) |
| FR-32b | D-9 | VC-32b | borde (clave rechazada) |
| FR-33 | D-9 | VC-33 | borde (passphrase) |
| FR-34 | D-9 | VC-34 | falla |
| FR-35 | D-9 | VC-35 | borde (agente caído) |
| FR-36 | D-11 | VC-36 | feliz |
| FR-37 | D-11 + hallazgo 4 | VC-37 | feliz |
| FR-38 | D-11 + hallazgo 4 | VC-38 | feliz |
| FR-39 | D-11 | VC-39 | borde |
| FR-40 | D-11 | VC-40 | borde |
| FR-41 | D-11 + "Una integración posible" | VC-41 | borde (modo crudo) |
| FR-42 | D-12 + hallazgo 3 | VC-42 | invariante |
| FR-43 | D-12 + hallazgo 4 | VC-43 | feliz |
| FR-44 | D-12 + hallazgo 4 | VC-44 | feliz |
| FR-45 | D-12 | VC-45 | falla parcial |
| FR-46 | D-12 + hallazgo 4 | VC-46 | borde (cierre) |
| FR-47 | D-12 | VC-47 | falla (timeout) |
| FR-48a | D-12 | VC-48a | falla |
| FR-48b | D-12 | VC-48b | falla |
| FR-49a | D-13 | VC-49a | feliz |
| FR-49b | D-13 | VC-49b | feliz |
| FR-49c | D-13 | VC-49c | borde |
| FR-49d | D-13 | VC-49d | borde |
| FR-50 | D-1 | VC-50 | borde (mover) |
| FR-51 | D-13 + hallazgo 2 | VC-51 | feliz |
| FR-52 | D-14 + hallazgo 8 | VC-52 | borde |
| FR-53 | D-14 + hallazgo 8 | VC-53 | borde |
| FR-54 | D-7 + hallazgo 1 | VC-54 | borde |
| BR-1 | D-1, D-9 + hallazgo 3 | VC-BR-1 | invariante |
| BR-2 | D-9 | VC-BR-2 | invariante |
| BR-3a | D-10 | VC-BR-3a | borde |
| BR-3b | D-10 | VC-BR-3b | borde (sin `HOME`) |
| BR-4 | D-8 | VC-BR-4 | invariante |
| BR-5 | D-12 | VC-BR-5 | invariante |
| INV-1 | Enunciado + D-6, D-15 | VC-INV-1 | invariante (build) |
| INV-2 | Enunciado + D-6, D-15 | VC-INV-2 | invariante (build) |
| INV-3 | D-3, D-6 | VC-INV-3 | invariante (build) |
| INV-4 | Enunciado ("el modelo de PTY/panes no cambia") + D-1 | VC-INV-4 | invariante |
| INV-5 | Enunciado ("los comandos existentes no cambian") | VC-INV-5 | invariante |
| INV-6 | Hallazgo 11 + D-15 | VC-INV-6 | invariante |
| INV-7 | Hallazgo 12 + D-5 | VC-INV-7 | invariante |
| INV-8 | Hallazgo 11 + D-3 | VC-INV-8 | invariante (build) |
| NFR-1 | Hallazgo 3 + D-1 | VC-NFR-1 | medición |
| NFR-2 | D-12 | VC-NFR-2 | medición |

**77 requerimientos (61 FR, 6 BR, 8 INV, 2 NFR), 77 VCs, 0 huérfanos.**

## Preguntas abiertas

Ninguna. Las decisiones pendientes 1 a 11 de las notas se resolvieron en el base
context (D-1 a D-16).

## Qué sigue

El plan de iteraciones va en `ssh-pane-plan.md`. Su primer paso es medir la línea de
base de FreeBSD (INV-2).
