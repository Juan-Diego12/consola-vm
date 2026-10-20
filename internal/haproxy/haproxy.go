// Package haproxy agrega y quita servidores del backend de HAProxy.
// La consola administra solo un bloque marcado dentro de /etc/haproxy/haproxy.cfg;
// el resto del archivo (global, defaults) se conserva tal cual.
package haproxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"consola-vm/internal/ejecucion"
)

const (
	archivoCfg     = "/etc/haproxy/haproxy.cfg"
	marcaInicio    = "# >>> consola-vm: inicio (bloque administrado por la consola, no editar a mano)"
	marcaFin       = "# <<< consola-vm: fin"
	PuertoFrontend = 80
)

// Servidor es cada servidor del backend.
type Servidor struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int    `json:"puerto"`
}

var (
	reServidor = regexp.MustCompile(`^\s*server\s+(\S+)\s+([0-9.]+):(\d+)`)
	reNombre   = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
)

// IPv4Valida indica si el texto es una dirección IPv4.
func IPv4Valida(ip string) bool {
	p := net.ParseIP(ip)
	return p != nil && p.To4() != nil && !strings.Contains(ip, ":")
}

// leer obtiene el haproxy.cfg actual de la VM.
func leer(host, llaveRoot string) (string, ejecucion.Resultado) {
	r := ejecucion.EjecutarSSH(host, "root", llaveRoot,
		"command -v haproxy >/dev/null || { echo 'HAProxy no esta instalado en esta VM' >&2; exit 1; }\ncat "+archivoCfg)
	contenido := r.Salida
	if r.Exito {
		// En la interfaz no se muestra el archivo completo, solo un resumen.
		r.Salida = fmt.Sprintf("%s leido (%d lineas)", archivoCfg, strings.Count(contenido, "\n")+1)
	}
	return contenido, r
}

// separar divide el archivo en lo que no es de la consola y la lista de servidores del bloque.
func separar(cfg string) (base string, servidores []Servidor) {
	servidores = []Servidor{}
	var fuera []string
	dentro := false
	for _, l := range strings.Split(strings.ReplaceAll(cfg, "\r", ""), "\n") {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "# >>> consola-vm"):
			dentro = true
			continue
		case strings.HasPrefix(t, "# <<< consola-vm"):
			dentro = false
			continue
		}
		if dentro {
			if m := reServidor.FindStringSubmatch(l); m != nil {
				p, _ := strconv.Atoi(m[3])
				servidores = append(servidores, Servidor{Nombre: m[1], IP: m[2], Puerto: p})
			}
			continue
		}
		fuera = append(fuera, l)
	}
	return strings.TrimRight(strings.Join(fuera, "\n"), "\n") + "\n", servidores
}

// bloque genera el frontend y el backend administrados por la consola.
func bloque(servidores []Servidor) string {
	var b strings.Builder
	b.WriteString(marcaInicio + "\n")
	fmt.Fprintf(&b, "frontend consola_web\n    bind *:%d\n    mode http\n    default_backend consola_backend\n\n", PuertoFrontend)
	b.WriteString("backend consola_backend\n    mode http\n    balance roundrobin\n")
	for _, s := range servidores {
		fmt.Fprintf(&b, "    server %s %s:%d check\n", s.Nombre, s.IP, s.Puerto)
	}
	b.WriteString(marcaFin + "\n")
	return b.String()
}

// aplicar copia la nueva configuración, la valida con haproxy -c y, si es válida,
// reemplaza la actual (dejando copia .bak) y recarga HAProxy.
func aplicar(host, llaveRoot, base string, servidores []Servidor) []ejecucion.Resultado {
	var pasos []ejecucion.Resultado
	destino := "root@" + host
	nuevo := base + "\n" + bloque(servidores)

	if err := os.MkdirAll("tmp", 0o755); err != nil {
		return append(pasos, ejecucion.Fallo("local", destino, "no se pudo crear la carpeta tmp: "+err.Error()))
	}
	f, err := os.CreateTemp("tmp", "haproxy-*.cfg")
	if err != nil {
		return append(pasos, ejecucion.Fallo("local", destino, "no se pudo crear el archivo temporal: "+err.Error()))
	}
	local := f.Name()
	defer os.Remove(local)
	_, err = f.WriteString(nuevo)
	f.Close()
	if err != nil {
		return append(pasos, ejecucion.Fallo("local", destino, "no se pudo escribir el archivo temporal: "+err.Error()))
	}

	r := ejecucion.CopiarArchivo(local, host, "root", llaveRoot, archivoCfg+".nuevo")
	pasos = append(pasos, r)
	if !r.Exito {
		return pasos
	}
	validar := fmt.Sprintf(`haproxy -c -f %[1]s.nuevo || { rm -f %[1]s.nuevo; echo "configuracion invalida: no se aplico ningun cambio" >&2; exit 1; }
cp %[1]s %[1]s.bak
mv %[1]s.nuevo %[1]s
systemctl enable haproxy >/dev/null 2>&1 || true
systemctl reload-or-restart haproxy
echo "configuracion valida, HAProxy recargado (copia anterior en %[1]s.bak)"
systemctl is-active haproxy`, archivoCfg)
	return append(pasos, ejecucion.EjecutarSSH(host, "root", llaveRoot, validar))
}

func ultimoExito(pasos []ejecucion.Resultado) bool {
	return len(pasos) > 0 && pasos[len(pasos)-1].Exito
}

// Listar devuelve los servidores del backend que administra la consola.
func Listar(host, llaveRoot string) ([]Servidor, ejecucion.Resultado) {
	cfg, r := leer(host, llaveRoot)
	if !r.Exito {
		return []Servidor{}, r
	}
	_, servidores := separar(cfg)
	return servidores, r
}

// Agregar añade un servidor al backend. La primera vez crea el frontend y el backend.
func Agregar(host, llaveRoot string, s Servidor) (pasos []ejecucion.Resultado, lista []Servidor, exito bool) {
	destino := "root@" + host
	switch {
	case !reNombre.MatchString(s.Nombre):
		return append(pasos, ejecucion.Fallo("ssh", destino, "nombre de servidor no válido (letras, números, guion o guion bajo)")), nil, false
	case !IPv4Valida(s.IP):
		return append(pasos, ejecucion.Fallo("ssh", destino, "la IP del servidor no es una IPv4 válida")), nil, false
	case s.Puerto < 1 || s.Puerto > 65535:
		return append(pasos, ejecucion.Fallo("ssh", destino, "el puerto debe estar entre 1 y 65535")), nil, false
	}
	cfg, r := leer(host, llaveRoot)
	pasos = append(pasos, r)
	if !r.Exito {
		return pasos, nil, false
	}
	base, lista := separar(cfg)
	for _, x := range lista {
		if x.Nombre == s.Nombre {
			return append(pasos, ejecucion.Fallo("ssh", destino, "ya existe un servidor llamado "+s.Nombre)), lista, false
		}
		if x.IP == s.IP && x.Puerto == s.Puerto {
			return append(pasos, ejecucion.Fallo("ssh", destino, fmt.Sprintf("%s:%d ya está en el backend como %s", s.IP, s.Puerto, x.Nombre))), lista, false
		}
	}
	lista = append(lista, s)
	pasos = append(pasos, aplicar(host, llaveRoot, base, lista)...)
	return pasos, lista, ultimoExito(pasos)
}

// Quitar elimina un servidor del backend por su nombre.
func Quitar(host, llaveRoot, nombre string) (pasos []ejecucion.Resultado, lista []Servidor, exito bool) {
	destino := "root@" + host
	cfg, r := leer(host, llaveRoot)
	pasos = append(pasos, r)
	if !r.Exito {
		return pasos, nil, false
	}
	base, actuales := separar(cfg)
	lista = []Servidor{}
	encontrado := false
	for _, x := range actuales {
		if x.Nombre == nombre {
			encontrado = true
			continue
		}
		lista = append(lista, x)
	}
	if !encontrado {
		return append(pasos, ejecucion.Fallo("ssh", destino, "no hay un servidor llamado "+nombre+" en el backend")), actuales, false
	}
	pasos = append(pasos, aplicar(host, llaveRoot, base, lista)...)
	return pasos, lista, ultimoExito(pasos)
}

// Probar hace n peticiones a /api/servidor a través de HAProxy y devuelve qué
// servidor respondió cada una (sirve para mostrar el balanceo en el video).
func Probar(ipHAProxy string, n int) []string {
	cliente := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true}, // cada petición es una conexión nueva
	}
	url := fmt.Sprintf("http://%s:%d/api/servidor", ipHAProxy, PuertoFrontend)
	respuestas := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		resp, err := cliente.Get(url)
		if err != nil {
			respuestas = append(respuestas, fmt.Sprintf("%d: error: %v", i, err))
			continue
		}
		cuerpo, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		respuestas = append(respuestas, fmt.Sprintf("%d: HTTP %d %s", i, resp.StatusCode, strings.TrimSpace(string(cuerpo))))
	}
	return respuestas
}
