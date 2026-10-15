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
