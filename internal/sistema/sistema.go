// Package sistema implementa el módulo Preparación del sistema:
// instalar y eliminar paquetes, personalizar el mensaje de entrada (/etc/issue)
// y cambiar el nombre del equipo (hostname y /etc/hosts).
// Todo se ejecuta por SSH como root, con la llave que configura el módulo de llaves.
package sistema

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"consola-vm/internal/ejecucion"
)

var (
	// Nombres de paquete según la política de Debian: minúsculas, números y los signos + - .
	// (mínimo 2 caracteres y empezando por letra o número). Así no puede colarse ningún
	// carácter con significado para la shell (espacios, ;, |, &, $, comillas...).
	nombrePaquete = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]{1,62}$`)

	// Hostname: una etiqueta DNS (RFC 1123) en minúsculas, de 1 a 63 caracteres,
	// sin empezar ni terminar en guion.
	nombreHost = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

// Paquetes que la consola no permite eliminar: sin ellos la VM deja de arrancar,
// pierde la red o la consola pierde el acceso por SSH.
var protegidos = map[string]bool{
	"openssh-server": true, "openssh-sftp-server": true, "openssh-client": true,
	"systemd": true, "systemd-sysv": true, "init": true, "apt": true, "dpkg": true,
	"bash": true, "coreutils": true, "libc6": true, "login": true, "passwd": true,
	"base-files": true, "util-linux": true, "ifupdown": true, "isc-dhcp-client": true,
	"dhcpcd-base": true, "iproute2": true, "linux-image-amd64": true,
	"grub-pc": true, "grub-efi-amd64": true,
}

// LargoMaximoMensaje es la longitud máxima del mensaje personalizado de /etc/issue.
const LargoMaximoMensaje = 120

// ValidarPaquete revisa el nombre del paquete y la acción ("instalar" o "eliminar").
func ValidarPaquete(paquete, accion string) error {
	switch {
	case !nombrePaquete.MatchString(paquete):
		return errors.New("nombre de paquete no válido: use minúsculas, números y los signos + - . (por ejemplo: tree, curl, nginx)")
	case accion != "instalar" && accion != "eliminar" && accion != "consultar":
		return errors.New("acción no válida: use instalar, eliminar o consultar")
	case accion == "eliminar" && protegidos[paquete]:
		return fmt.Errorf("el paquete %s es necesario para el sistema o para la consola y no se puede eliminar", paquete)
	}
	return nil
}

// ValidarHostname normaliza (minúsculas, sin espacios) y valida el nuevo nombre del equipo.
func ValidarHostname(nombre string) (string, error) {
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	if !nombreHost.MatchString(nombre) {
		return "", errors.New("nombre de host no válido: use letras minúsculas, números y guiones (máximo 63), sin empezar ni terminar en guion")
	}
	return nombre, nil
}

// OpcionesIssue son las casillas y el mensaje del formulario de personalización.
type OpcionesIssue struct {
	Hostname bool
	Fecha    bool
	Hora     bool
	IP       bool
	Mensaje  string
}

// ContenidoIssue arma el texto de /etc/issue. Las secuencias \n, \4, \d y \t las
// reemplaza agetty al mostrar la pantalla de entrada. Las barras invertidas del mensaje
// del usuario se duplican para que agetty las muestre tal cual y no las interprete.
func ContenidoIssue(o OpcionesIssue) (string, error) {
	mensaje := strings.TrimSpace(o.Mensaje)
	if utf8.RuneCountInString(mensaje) > LargoMaximoMensaje {
		return "", fmt.Errorf("el mensaje personalizado admite máximo %d caracteres", LargoMaximoMensaje)
	}
	for _, r := range mensaje {
		if unicode.IsControl(r) {
			return "", errors.New("el mensaje personalizado no puede tener saltos de línea ni caracteres de control")
		}
	}
	if !o.Hostname && !o.Fecha && !o.Hora && !o.IP && mensaje == "" {
		return "", errors.New("seleccione al menos un dato o escriba un mensaje personalizado")
	}
	mensaje = strings.ReplaceAll(mensaje, `\`, `\\`)

	const borde = "+----------------------------------------------+\n"
	var b strings.Builder
	b.WriteString(borde)
	b.WriteString("  \\S{PRETTY_NAME} \\l\n\n")
	if o.Hostname {
		b.WriteString("  Nombre del host: \\n\n")
	}
	if o.IP {
		b.WriteString("  Dirección IP:    \\4\n")
	}
	if o.Fecha {
		b.WriteString("  Fecha:           \\d\n")
	}
	if o.Hora {
		b.WriteString("  Hora:            \\t\n")
	}
	if mensaje != "" {
		b.WriteString("\n  " + mensaje + "\n")
	}
	b.WriteString(borde)
	b.WriteString("\n")
	return b.String(), nil
}

// comandoEstado informa si un paquete está instalado y en qué versión.
func comandoEstado(paquete string) string {
	return fmt.Sprintf(`if dpkg-query -W -f='${Status}' %[1]s 2>/dev/null | grep -q 'install ok installed'; then
  echo "estado: %[1]s instalado (version $(dpkg-query -W -f='${Version}' %[1]s))"
else
  echo "estado: %[1]s no instalado"
fi`, paquete)
}

// EstadoPaquete consulta si el paquete está instalado (sirve para mostrar el estado antes y después).
func EstadoPaquete(host, llaveRoot, paquete string) ejecucion.Resultado {
	if err := ValidarPaquete(paquete, "consultar"); err != nil {
		return ejecucion.Fallo("ssh", "root@"+host, err.Error())
	}
	return ejecucion.EjecutarSSH(host, "root", llaveRoot, comandoEstado(paquete))
}

// Paquete instala o elimina un paquete con apt-get, de forma no interactiva.
// Pasos: 1) estado antes, 2) instalar o eliminar, 3) estado después.
func Paquete(host, llaveRoot, paquete, accion string) (pasos []ejecucion.Resultado, exito bool) {
	agregar := func(r ejecucion.Resultado) bool { pasos = append(pasos, r); return r.Exito }
	if accion == "consultar" {
		accion = "" // consultar no es válido aquí: solo instalar o eliminar
	}
	if err := ValidarPaquete(paquete, accion); err != nil {
		agregar(ejecucion.Fallo("ssh", "root@"+host, err.Error()))
		return
	}

	// 1. Estado antes del cambio.
	if !agregar(ejecucion.EjecutarSSH(host, "root", llaveRoot, comandoEstado(paquete))) {
		return
	}

	// 2. Instalar o eliminar (DEBIAN_FRONTEND evita cualquier pregunta interactiva).
	var comando string
	if accion == "instalar" {
		comando = fmt.Sprintf(`export DEBIAN_FRONTEND=noninteractive
apt-get update -q
if ! apt-cache show %[1]s >/dev/null 2>&1; then
  echo "el paquete %[1]s no existe en los repositorios configurados" >&2
  exit 1
fi
apt-get install -y -q %[1]s`, paquete)
	} else {
		comando = fmt.Sprintf(`export DEBIAN_FRONTEND=noninteractive
if ! dpkg-query -W -f='${Status}' %[1]s 2>/dev/null | grep -q 'install ok installed'; then
  echo "el paquete %[1]s no esta instalado" >&2
  exit 1
fi
apt-get purge -y -q %[1]s
apt-get autoremove -y -q`, paquete)
	}
	if !agregar(ejecucion.EjecutarSSH(host, "root", llaveRoot, comando)) {
		return
	}

	// 3. Estado después del cambio.
	if !agregar(ejecucion.EjecutarSSH(host, "root", llaveRoot, comandoEstado(paquete))) {
		return
	}
	exito = true
	return
}

// comandoIssue guarda una copia del /etc/issue original (solo la primera vez), escribe
// el contenido nuevo que llega por la entrada estándar y pide a agetty que actualice la
// pantalla de entrada. El texto del usuario nunca forma parte del comando.
const comandoIssue = `set -e
[ -f /etc/issue.orig ] || cp /etc/issue /etc/issue.orig
cat > /etc/issue
echo "--- /etc/issue actualizado (copia del original en /etc/issue.orig) ---"
cat /etc/issue
agetty --reload 2>/dev/null || true`

// Issue escribe el nuevo /etc/issue en la VM.
func Issue(host, llaveRoot string, o OpcionesIssue) (pasos []ejecucion.Resultado, exito bool) {
	contenido, err := ContenidoIssue(o)
	if err != nil {
		pasos = append(pasos, ejecucion.Fallo("ssh", "root@"+host, err.Error()))
		return
	}
	r := ejecucion.EjecutarSSHConEntrada(host, "root", llaveRoot, comandoIssue, contenido)
	pasos = append(pasos, r)
	return pasos, r.Exito
}

// Hostname cambia el nombre del equipo con hostnamectl y actualiza la línea 127.0.1.1
// de /etc/hosts (si no existe, la agrega). Al final comprueba que el nombre resuelva.
func Hostname(host, llaveRoot, nuevo string) (pasos []ejecucion.Resultado, exito bool) {
	nuevo, err := ValidarHostname(nuevo)
	if err != nil {
		pasos = append(pasos, ejecucion.Fallo("ssh", "root@"+host, err.Error()))
		return
	}
	comando := fmt.Sprintf(`set -e
anterior=$(hostname)
hostnamectl set-hostname %[1]s
if grep -qE '^127\.0\.1\.1[[:space:]]' /etc/hosts; then
  sed -i -E 's/^127\.0\.1\.1[[:space:]].*/127.0.1.1\t%[1]s/' /etc/hosts
else
  printf '127.0.1.1\t%[1]s\n' >> /etc/hosts
fi
echo "hostname: $anterior -> $(hostname)"
echo "/etc/hosts: $(grep -E '^127\.0\.1\.1' /etc/hosts)"
getent hosts %[1]s
agetty --reload 2>/dev/null || true`, nuevo)
	r := ejecucion.EjecutarSSH(host, "root", llaveRoot, comando)
	pasos = append(pasos, r)
	return pasos, r.Exito
}

// comandoInfo reúne el estado actual del sistema (sirve para mostrar el "antes" en las pruebas).
const comandoInfo = `echo "hostname=$(hostname)"
echo "hosts=$(grep -E '^127\.0\.1\.1' /etc/hosts | head -n1)"
echo "debian=$(cat /etc/debian_version)"
echo "---issue---"
cat /etc/issue`

// Info devuelve el hostname, la línea 127.0.1.1 de /etc/hosts, la versión de Debian
// y el contenido de /etc/issue.
func Info(host, llaveRoot string) (map[string]string, ejecucion.Resultado) {
	datos := map[string]string{}
	r := ejecucion.EjecutarSSH(host, "root", llaveRoot, comandoInfo)
	if !r.Exito {
		return datos, r
	}
	partes := strings.SplitN(r.Salida, "---issue---", 2)
	for _, linea := range strings.Split(partes[0], "\n") {
		if clave, valor, ok := strings.Cut(strings.TrimSpace(linea), "="); ok {
			datos[clave] = valor
		}
	}
	if len(partes) == 2 {
		datos["issue"] = strings.Trim(partes[1], "\n")
	}
	return datos, r
}
