// Package llaves configura el acceso por llaves SSH para root:
// genera el par de llaves, instala la pública con Guest Additions,
// verifica el acceso y desactiva la autenticación por contraseña.
package llaves

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"consola-vm/internal/ejecucion"
)

// Solo letras, números, punto, guion y guion bajo: evita rutas como "../" y caracteres
// especiales en los nombres de archivo y en los comandos.
var nombreValido = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// NombreValido indica si un nombre de VM o de usuario es seguro para usarlo.
func NombreValido(nombre string) bool { return nombreValido.MatchString(nombre) }

// RutaPrivada devuelve la ruta de la llave privada de un usuario en una VM.
// Ejemplo: llaves/DebianDev_root
func RutaPrivada(vm, usuario string) string {
	return filepath.Join(ejecucion.DirLlaves, vm+"_"+usuario)
}

// Generar crea el par de llaves ed25519 sin frase de paso.
// Si ya existe y reemplazar es false, lo deja como está.
func Generar(vm, usuario string, reemplazar bool) ejecucion.Resultado {
	if !NombreValido(vm) || !NombreValido(usuario) {
		return ejecucion.Fallo("local", vm, "nombre de VM o de usuario no válido")
	}
	privada := RutaPrivada(vm, usuario)
	if _, err := os.Stat(privada); err == nil {
		if !reemplazar {
			return ejecucion.Resultado{Operacion: "local", Destino: vm, Exito: true, Duracion: "0s",
				Comando: "(sin cambios)", Salida: "la llave ya existía, se reutiliza: " + privada}
		}
		os.Remove(privada)
		os.Remove(privada + ".pub")
	}
	if err := os.MkdirAll(ejecucion.DirLlaves, 0o700); err != nil {
		return ejecucion.Fallo("local", vm, "no se pudo crear la carpeta de llaves: "+err.Error())
	}
	comentario := fmt.Sprintf("consola-vm@%s:%s", vm, usuario)
	return ejecucion.EjecutarLocal(vm, "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C", comentario, "-f", privada)
}

// LeerPublica devuelve el contenido de la llave pública (una sola línea).
func LeerPublica(vm, usuario string) (string, error) {
	datos, err := os.ReadFile(RutaPrivada(vm, usuario) + ".pub")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(datos)), nil
}

// ComandoInstalarPublica arma el comando que agrega la llave pública a authorized_keys
// del usuario indicado (sin duplicarla). Lo reutiliza el módulo de usuarios.
func ComandoInstalarPublica(usuario, publica string) string {
	home := "/home/" + usuario
	if usuario == "root" {
		home = "/root"
	}
	return fmt.Sprintf(`set -e
mkdir -p %[1]s/.ssh
chmod 700 %[1]s/.ssh
touch %[1]s/.ssh/authorized_keys
grep -qxF '%[2]s' %[1]s/.ssh/authorized_keys || echo '%[2]s' >> %[1]s/.ssh/authorized_keys
chmod 600 %[1]s/.ssh/authorized_keys
chown -R %[3]s:%[3]s %[1]s/.ssh
echo "llave instalada en %[1]s/.ssh/authorized_keys"`, home, publica, usuario)
}

// comandoDesactivarPassword deja solo el acceso por llaves. Se usa un archivo en
// sshd_config.d (Debian lo incluye al inicio de sshd_config, así que tiene prioridad).
const comandoDesactivarPassword = `set -e
cat > /etc/ssh/sshd_config.d/10-consola-vm.conf <<'FIN'
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin prohibit-password
FIN
sshd -t
systemctl reload ssh
sshd -T | grep -Ei '^(passwordauthentication|kbdinteractiveauthentication|permitrootlogin) '`

// ConfigurarRoot ejecuta todo el proceso para root y devuelve cada paso.
// Se detiene en el primer paso que falle.
func ConfigurarRoot(vm, passwordRoot string, reemplazar bool) (ip string, pasos []ejecucion.Resultado, exito bool) {
	agregar := func(r ejecucion.Resultado) bool { pasos = append(pasos, r); return r.Exito }

	// 1. Generar el par de llaves en el PC anfitrión.
	if !agregar(Generar(vm, "root", reemplazar)) {
		return
	}
	publica, err := LeerPublica(vm, "root")
	if err != nil {
		agregar(ejecucion.Fallo("local", vm, "no se pudo leer la llave pública: "+err.Error()))
		return
	}
	// 2. Instalar la pública con Guest Additions (todavía no hay acceso por llave).
	if !agregar(ejecucion.EjecutarGuest(vm, "root", passwordRoot, ComandoInstalarPublica("root", publica))) {
		return
	}
	// 3. Averiguar la IP de la VM.
	ip, r := ejecucion.ObtenerIP(vm)
	if !agregar(r) {
		return
	}
	llave := RutaPrivada(vm, "root")
	// 4. Comprobar que el acceso por llave funciona ANTES de quitar las contraseñas.
	if !agregar(ejecucion.EjecutarSSH(ip, "root", llave, "whoami && hostname")) {
		return
	}
	// 5. Desactivar la autenticación por contraseña.
	if !agregar(ejecucion.EjecutarSSH(ip, "root", llave, comandoDesactivarPassword)) {
		return
	}
	exito = true
	return
}
