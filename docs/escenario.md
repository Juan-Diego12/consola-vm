# Escenario oficial de pruebas: Nota Parcial 2

Escenario montado en el PC de Daniel, donde se graban los videos de pruebas.

## Equipo anfitrión

| Elemento | Valor |
|---|---|
| Sistema operativo | Windows 11 |
| Virtualización | Oracle VirtualBox 7.1.6 |
| Red de las VMs | Adaptador puente sobre Intel(R) Wireless-AC 9462 (Wi-Fi) |
| Red local | 192.168.20.0/24, IPs asignadas por DHCP del router |

## Máquinas virtuales

| VM | Función | Hostname | RAM | Disco | IP (18-oct) | Snapshot |
|---|---|---|---|---|---|---|
| NP2-Base | VM de pruebas de la consola | np2-base | 1024 MB | NP2-Base.vdi (20 GB, normal) | 192.168.20.27 | base-limpia |
| NP2-HAProxy | Balanceador de carga | np2-haproxy | 1024 MB | Clon completo de NP2-Base | 192.168.20.29 | haproxy-limpio |
| NP2-Backend1 | Servidor del catálogo 1 | np2-backend1 | 512 MB | NP2-Backend-Plantilla.vdi (multiconexión) | 192.168.20.32 | — |
| NP2-Backend2 | Servidor del catálogo 2 | np2-backend2 | 512 MB | NP2-Backend-Plantilla.vdi (multiconexión) | 192.168.20.31 | — |

**Importante:** las IPs las asigna el router y pueden cambiar entre sesiones. La consola debe recibirlas como un campo o una configuración, nunca fijas en el código. Para consultarlas: `hostname -I` dentro de cada VM.

## Configuración común (todas las VMs)

- Debian 13.7.0 instalado en modo texto, actualizado (`apt update` y `apt upgrade`).
- Programas: solo SSH server y utilidades estándar del sistema.
- Guest Additions instaladas (necesarias para `VBoxManage guestcontrol`).
- Usuario normal: `daniel`. Usuario `root` con contraseña. **La contraseña no se publica en el repo: se comparte por fuera.**
- Acceso con contraseña por SSH aún habilitado; la consola lo desactiva al configurar las llaves.

## Detalles por VM

**NP2-Base:** estado limpio, sin modificaciones adicionales. Antes de cada video se restaura el snapshot `base-limpia` para que el estado anterior sea siempre el mismo.

**NP2-HAProxy:** HAProxy instalado con su configuración predeterminada (`/etc/haproxy/haproxy.cfg` sin frontend ni backend definidos).

**Disco multiconexión de backends:** `F:\VirtualBox\VMs\NP2-Backend-Plantilla\NP2-Backend-Plantilla.vdi`, con:
- La app del catálogo en `/opt/catalogo/` (binario `catalogo-libros`, `static/`, `data/` y `script.sh`).
- El servicio `catalogo.service`, habilitado, que escucha en el **puerto 8000**.
- `/etc/machine-id` vacío, para que cada VM genere el suyo en el primer arranque y reciba su propia IP.

Cada backend muestra su propio hostname en la página (`/api/servidor`), lo que permite comprobar a cuál servidor envía HAProxy cada petición.

## Apagado de las VMs

Siempre con apagado ordenado, nunca con "Power off":

    VBoxManage controlvm "NOMBRE_VM" acpipowerbutton

## Evidencias del montaje

En `docs/evidencias/`:

| Evidencia | Contenido |
|---|---|
| E01 | Creación de la VM base en PowerShell |
| E02 | Selección de programas: SSH server y utilidades estándar |
| E03 | Detalles de NP2-Base en VirtualBox |
| E04 | Versión 13.7 y SSH activo |
| E05 | Final de la instalación de las Guest Additions |
| E06 | Versión de las Guest Additions y respuesta de guestcontrol |
| E07 | Snapshot base-limpia |
| E08 | Hostname, /etc/hosts e IP de NP2-HAProxy |
| E09 | HAProxy instalado y activo con su configuración predeterminada |
| E10 | Copia del catálogo a la plantilla con scp |
| E11 | Servicio catalogo activo y habilitado |
| E12 | Catálogo respondiendo desde las VMs |
| E13 | Disco de la plantilla convertido a multiconexión |
| E14 | Backends 1 y 2 respondiendo, cada uno con su servidor |
