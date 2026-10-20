// Package usuarios crea, elimina y lista usuarios en la VM (siempre por SSH como root).
package usuarios

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"consola-vm/internal/ejecucion"
	"consola-vm/internal/llaves"
)

// Mismo formato que acepta useradd en Debian: minúsculas, números, guion y guion bajo.
var nombreUsuario = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)

// Usuarios que la consola nunca crea ni elimina.
var protegidos = map[string]bool{"root": true, "nobody": true}

// NombreValido indica si el nombre es aceptable para un usuario de Linux.
func NombreValido(nombre string) bool {
	return nombreUsuario.MatchString(nombre) && !protegidos[nombre]
}

// Usuario es cada elemento de la lista que devuelve Listar.
type Usuario struct {
	Nombre string `json:"nombre"`
	UID    int    `json:"uid"`
	Home   string `json:"home"`
	Shell  string `json:"shell"`
	Sudo   bool   `json:"sudo"`  // pertenece al grupo sudo
	Llave  bool   `json:"llave"` // la consola tiene su llave privada (se puede descargar)
}

// Usuarios normales: UID entre 1000 y 65533. Después del separador va el grupo sudo.
const comandoListar = `getent passwd | awk -F: '$3>=1000 && $3<65534 {print $1":"$3":"$6":"$7}'
echo "--sudo--"
getent group sudo | cut -d: -f4`

// Listar devuelve los usuarios normales de la VM (para la lista desplegable).
func Listar(vm, host, llaveRoot string) ([]Usuario, ejecucion.Resultado) {
	lista := []Usuario{}
	r := ejecucion.EjecutarSSH(host, "root", llaveRoot, comandoListar)
	if !r.Exito {
		return lista, r
	}
	partes := strings.SplitN(r.Salida, "--sudo--", 2)
	enSudo := map[string]bool{}
	if len(partes) == 2 {
		for _, n := range strings.Split(strings.TrimSpace(partes[1]), ",") {
			if n = strings.TrimSpace(n); n != "" {
				enSudo[n] = true
			}
		}
	}
	for _, linea := range strings.Split(partes[0], "\n") {
		c := strings.Split(strings.TrimSpace(linea), ":")
		if len(c) != 4 {
			continue
		}
		uid, _ := strconv.Atoi(c[1])
		_, errLlave := os.Stat(llaves.RutaPrivada(vm, c[0]))
		lista = append(lista, Usuario{Nombre: c[0], UID: uid, Home: c[2], Shell: c[3],
			Sudo: enSudo[c[0]], Llave: errLlave == nil})
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].Nombre < lista[j].Nombre })
	return lista, r
}

// DatosCrear son las opciones del formulario de creación.
type DatosCrear struct {
	Nombre   string
	Password string // opcional si se marca Llaves; obligatoria si se marca Sudo
	Sudo     bool   // agregar al grupo sudo
	Llaves   bool   // generar llaves e instalarlas (misma lógica que root)
}

// Crear crea el usuario y devuelve cada paso. Se detiene en el primero que falle.
func Crear(vm, host, llaveRoot string, d DatosCrear) (pasos []ejecucion.Resultado, exito bool) {
	agregar := func(r ejecucion.Resultado) bool { pasos = append(pasos, r); return r.Exito }
	destino := "root@" + host

	switch {
	case !NombreValido(d.Nombre):
		agregar(ejecucion.Fallo("ssh", destino, "nombre de usuario no válido: use minúsculas, números, guion o guion bajo, y que empiece por letra"))
		return
	case d.Sudo && d.Password == "":
		agregar(ejecucion.Fallo("ssh", destino, "un usuario con sudo necesita contraseña: sudo la pide al usarlo"))
		return
	case !d.Llaves && d.Password == "":
		agregar(ejecucion.Fallo("ssh", destino, "indique una contraseña o marque la opción de llaves; si no, el usuario no podría iniciar sesión"))
		return
	case strings.ContainsAny(d.Password, ":\r\n"):
		agregar(ejecucion.Fallo("ssh", destino, "la contraseña no puede contener ':' ni saltos de línea"))
		return
	}

	// 1. Crear el usuario con su carpeta personal y bash.
	crear := fmt.Sprintf(`if id %[1]s >/dev/null 2>&1; then echo "el usuario %[1]s ya existe" >&2; exit 1; fi
useradd -m -s /bin/bash %[1]s
echo "usuario %[1]s creado con carpeta /home/%[1]s"`, d.Nombre)
	if !agregar(ejecucion.EjecutarSSH(host, "root", llaveRoot, crear)) {
		return
	}

	// 2. Contraseña: va por la entrada estándar, nunca en el comando ni en el log.
	if d.Password != "" {
		if !agregar(ejecucion.EjecutarSSHConEntrada(host, "root", llaveRoot,
			"chpasswd && echo 'contrasena asignada'", d.Nombre+":"+d.Password+"\n")) {
			return
		}
	}

	// 3. Grupo sudo.
	if d.Sudo {
		sudo := fmt.Sprintf(`usermod -aG sudo %[1]s
id %[1]s
command -v sudo >/dev/null || echo "aviso: el paquete sudo no esta instalado (lo instala Preparacion del sistema)"`, d.Nombre)
		if !agregar(ejecucion.EjecutarSSH(host, "root", llaveRoot, sudo)) {
			return
		}
	}

	// 4. Llaves: se reutiliza la misma lógica que para root.
	if d.Llaves {
		if !agregar(llaves.Generar(vm, d.Nombre, true)) {
			return
		}
		publica, err := llaves.LeerPublica(vm, d.Nombre)
		if err != nil {
			agregar(ejecucion.Fallo("local", vm, "no se pudo leer la llave pública: "+err.Error()))
			return
		}
		if !agregar(ejecucion.EjecutarSSH(host, "root", llaveRoot, llaves.ComandoInstalarPublica(d.Nombre, publica))) {
			return
		}
		// Comprobar que el nuevo usuario entra con su propia llave.
		if !agregar(ejecucion.EjecutarSSH(host, d.Nombre, llaves.RutaPrivada(vm, d.Nombre), "whoami && id")) {
			return
		}
	}
	exito = true
	return
}

// Eliminar borra el usuario. Con conArchivos también borra su carpeta personal y su correo.
func Eliminar(vm, host, llaveRoot, nombre string, conArchivos bool) (pasos []ejecucion.Resultado, exito bool) {
	agregar := func(r ejecucion.Resultado) bool { pasos = append(pasos, r); return r.Exito }
	if !NombreValido(nombre) {
		agregar(ejecucion.Fallo("ssh", "root@"+host, "nombre de usuario no válido o protegido"))
		return
	}
	opcion, mensaje := "", "se conserva la carpeta /home/"+nombre
	if conArchivos {
		opcion, mensaje = "-r ", "carpeta personal eliminada"
	}
	borrar := fmt.Sprintf(`uid=$(id -u %[1]s 2>/dev/null) || { echo "el usuario %[1]s no existe" >&2; exit 1; }
if [ "$uid" -lt 1000 ]; then echo "no se permite eliminar usuarios del sistema" >&2; exit 1; fi
pkill -KILL -u %[1]s 2>/dev/null || true
sleep 1
userdel %[2]s%[1]s || exit $?
echo "usuario %[1]s eliminado; %[3]s"`, nombre, opcion, mensaje)
	if !agregar(ejecucion.EjecutarSSH(host, "root", llaveRoot, borrar)) {
		return
	}
	// La llave de ese usuario ya no sirve: se borra también en la consola.
	privada := llaves.RutaPrivada(vm, nombre)
	if _, err := os.Stat(privada); err == nil {
		os.Remove(privada)
		os.Remove(privada + ".pub")
		agregar(ejecucion.Resultado{Operacion: "local", Destino: vm, Comando: "eliminar " + privada,
			Salida: "llave local del usuario eliminada", Duracion: "0s", Exito: true})
	}
	exito = true
	return
}
