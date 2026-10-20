# Contrato entre la interfaz (Daniel) y los módulos (Diego)

## Capa de ejecución (`internal/ejecucion`)

| Función | Uso |
|---|---|
| `EjecutarGuest(vm, usuario, password, comando) Resultado` | Solo arranque inicial, con Guest Additions |
| `EjecutarSSH(host, usuario, llave, comando) Resultado` | Todo lo demás, por llave y `BatchMode=yes` |
| `CopiarArchivo(origen, host, usuario, llave, destino) Resultado` | Copia con scp |
| `ObtenerIP(vm) (string, Resultado)` | IP de la VM desde Guest Additions |

Toda operación devuelve un `Resultado` y queda en `logs/operaciones.log`:

```json
{ "operacion": "ssh", "destino": "root@192.168.20.30", "comando": "...",
  "salida": "...", "errores": "...", "codigo": 0, "duracion": "120ms", "exito": true }
```

## Respuesta de todos los endpoints

```json
{ "exito": true, "mensaje": "texto para el usuario", "datos": { }, "pasos": [ Resultado, ... ] }
```

La interfaz muestra `mensaje` y, en un detalle desplegable, cada paso con `comando`, `salida` y `errores`.

## Endpoints

| Método y ruta | Entrada | Datos |
|---|---|---|
| `GET /api/vm/ip?vm=X` | | `ip` |
| `POST /api/llaves/root` | `{"vm","password","reemplazar"}` | `ip` |
| `GET /api/llaves/descargar?vm=X&usuario=root` | | archivo de la llave privada |

Los endpoints de usuarios, systemd y HAProxy se agregan aquí a medida que se construyan.

## Reglas

- Nombres de VM y usuario: `^[A-Za-z0-9._-]{1,64}$`. Daniel valida en el servidor el resto de entradas.
- Las contraseñas nunca se guardan ni se escriben en el log.
- Llaves en `llaves/<VM>_<usuario>` (carpeta ignorada por Git).

## Cambios en la capa de ejecución (20 de octubre)

Las firmas existentes no cambian. Se agregó:

| Función | Uso |
|---|---|
| `EjecutarSSHConEntrada(host, usuario, llave, comando, entrada) Resultado` | Igual que `EjecutarSSH`, pero envía `entrada` por la entrada estándar (contraseñas). La entrada no se registra |

`CopiarArchivo` ahora usa `scp -r`: copia archivos o carpetas completas.

## Reglas comunes de los módulos

- `vm` es el nombre de la VM en VirtualBox (sirve para ubicar la llave `llaves/<vm>_root`) e `ip` es su IP actual. La IP **siempre** la envía la interfaz; se puede precargar con `GET /api/vm/ip?vm=X`.
- Antes de usar un módulo en una VM, esa VM debe tener configuradas las llaves de root (`POST /api/llaves/root`). Si no, la respuesta es 400 con un mensaje que lo indica.
- Código HTTP 400 = datos inválidos (no se ejecutó nada). Código 200 = se ejecutó; el resultado está en `exito` y el detalle en `pasos`.
- Si un paso tiene `exito: true` pero trae `errores`, mostrarlo como advertencia, no como fallo.
- `POST /api/llaves/root`: si `password` llega vacío se usa la variable de entorno `VBOX_GUEST_PASSWORD`.

## Usuarios

| Método y ruta | Entrada | `datos` |
|---|---|---|
| `GET /api/usuarios?vm=X&ip=Y` | | `usuarios`: lista de `{nombre, uid, home, shell, sudo, llave}` (solo UID ≥ 1000) |
| `POST /api/usuarios/crear` | `{"vm","ip","nombre","password","sudo":bool,"llaves":bool}` | `nombre`, `llave` |
| `POST /api/usuarios/eliminar` | `{"vm","ip","nombre","conArchivos":bool}` | |

- Nombre: `^[a-z_][a-z0-9_-]{0,31}$`; `root` y `nobody` están protegidos.
- `password` es obligatoria si `sudo` es true, y también si `llaves` es false. No puede tener `:` ni saltos de línea.
- Con `llaves` true, la llave privada del usuario se descarga con `GET /api/llaves/descargar?vm=X&usuario=nombre`.
- Al eliminar, también se borra la llave local del usuario.

## systemd

| Método y ruta | Entrada | `datos` |
|---|---|---|
| `GET /api/servicios/recursos` | | `recursos`: lista de `{nombre, descripcion, ejecutable, argumentos, entorno, puerto, unidad}` |
| `POST /api/servicios/desplegar` | `{"vm","ip","recurso"}` | |
| `POST /api/servicios/accion` | `{"vm","ip","recurso","accion"}` | |

- Recursos: `catalogo` (puerto 8000) y `fechahora` (escribe en `/var/log/fechahora.log`). La interfaz envía solo el nombre, nunca rutas.
- `desplegar` hace cuatro pasos: preparar `/opt/<recurso>`, transferir, permisos y crear la unidad `/etc/systemd/system/<recurso>.service` (no lo inicia).
- `accion`: `start`, `stop`, `restart`, `status`, `enable` o `disable`. Cada respuesta termina con una línea `activo: ... | habilitado: ...`.
- El campo `unidad` de cada recurso trae el texto del archivo `.service`, para mostrarlo en la página.

## HAProxy

| Método y ruta | Entrada | `datos` |
|---|---|---|
| `GET /api/haproxy/servidores?vm=X&ip=Y` | | `servidores`: lista de `{nombre, ip, puerto}`, `puertoFrontend` |
| `POST /api/haproxy/agregar` | `{"vm","ip","nombre","ipServidor","puerto"}` | `servidores` |
| `POST /api/haproxy/quitar` | `{"vm","ip","nombre"}` | `servidores` |
| `GET /api/haproxy/probar?ip=Y&n=6` | | `respuestas`: qué backend respondió cada petición |

- `vm` e `ip` son los de la VM de HAProxy; `ipServidor` y `puerto` los del backend (por ejemplo 8000).
- La consola administra un bloque marcado en `/etc/haproxy/haproxy.cfg` (`frontend consola_web` en el puerto 80 y `backend consola_backend` con `roundrobin`). La primera vez lo crea; el resto del archivo no se toca.
- Cada cambio se valida con `haproxy -c` antes de aplicarse. Si es inválido, no se cambia nada; si es válido, queda copia en `haproxy.cfg.bak` y se recarga HAProxy.
- Nombre del servidor: `^[A-Za-z0-9_-]{1,32}$`. No se permiten nombres repetidos ni la misma IP:puerto dos veces.

