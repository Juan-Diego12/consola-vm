// Package servicios instala en la VM los recursos de la carpeta recursos/ como servicios
// de systemd (transferencia, permisos y unidad) y ejecuta las seis acciones de control.
package servicios

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"consola-vm/internal/ejecucion"
)

// Recurso describe un programa de la carpeta recursos/ que se puede instalar como servicio.
// Su Nombre es también el nombre del servicio y de la carpeta en /opt.
type Recurso struct {
	Nombre      string `json:"nombre"`
	Descripcion string `json:"descripcion"`
	Carpeta     string `json:"-"`          // carpeta local con todo lo que se copia
	Ejecutable  string `json:"ejecutable"` // archivo que lanza la unidad
	Argumentos  string `json:"argumentos,omitempty"`
	Entorno     string `json:"entorno,omitempty"` // variable de entorno, p. ej. PORT=8000
	Puerto      int    `json:"puerto,omitempty"`
	Unidad      string `json:"unidad"` // contenido del archivo .service que se generará
}

// Recursos disponibles. Se identifican por nombre, así la interfaz nunca envía rutas.
var recursos = map[string]Recurso{
	"catalogo": {
		Nombre:      "catalogo",
		Descripcion: "Catalogo de libros (aplicacion web en Go, puerto 8000)",
		Carpeta:     "recursos/catalogo",
		Ejecutable:  "catalogo-libros",
		Entorno:     "PORT=8000",
		Puerto:      8000,
	},
	"fechahora": {
		Nombre:      "fechahora",
		Descripcion: "Agrega la fecha y hora a /var/log/fechahora.log cada 5 segundos",
		Carpeta:     "recursos/fechahora",
		Ejecutable:  "fechahora.sh",
		Argumentos:  "/var/log/fechahora.log",
	},
}

// Acciones de los seis botones.
var acciones = map[string]bool{
	"start": true, "stop": true, "restart": true, "status": true, "enable": true, "disable": true,
}

// Buscar devuelve el recurso con su unidad ya generada.
func Buscar(nombre string) (Recurso, bool) {
	r, ok := recursos[nombre]
	if ok {
		r.Unidad = Unidad(r)
	}
	return r, ok
}

// Lista devuelve los recursos ordenados por nombre (para la lista desplegable).
func Lista() []Recurso {
	lista := make([]Recurso, 0, len(recursos))
	for nombre := range recursos {
		r, _ := Buscar(nombre)
		lista = append(lista, r)
	}
	sort.Slice(lista, func(i, j int) bool { return lista[i].Nombre < lista[j].Nombre })
	return lista
}

func dirRemoto(r Recurso) string { return "/opt/" + r.Nombre }

// Unidad genera el archivo de unidad de systemd del recurso.
func Unidad(r Recurso) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[Unit]\nDescription=%s\nAfter=network.target\n\n", r.Descripcion)
	fmt.Fprintf(&b, "[Service]\nType=simple\nWorkingDirectory=%s\n", dirRemoto(r))
	if r.Entorno != "" {
		fmt.Fprintf(&b, "Environment=%s\n", r.Entorno)
	}
	inicio := dirRemoto(r) + "/" + r.Ejecutable
	if r.Argumentos != "" {
		inicio += " " + r.Argumentos
	}
	fmt.Fprintf(&b, "ExecStart=%s\nRestart=always\nRestartSec=1s\n\n", inicio)
	b.WriteString("[Install]\nWantedBy=multi-user.target\n")
	return b.String()
}

// Desplegar copia el recurso a /opt/<nombre>, ajusta permisos y crea la unidad.
// No inicia el servicio: eso se hace con los botones.
func Desplegar(host, llaveRoot, nombre string) (pasos []ejecucion.Resultado, exito bool) {
	agregar := func(r ejecucion.Resultado) bool { pasos = append(pasos, r); return r.Exito }
	destino := "root@" + host

	r, ok := Buscar(nombre)
	if !ok {
		agregar(ejecucion.Fallo("ssh", destino, "recurso desconocido: "+nombre))
		return
	}
	entradas, err := os.ReadDir(r.Carpeta)
	if err != nil {
		agregar(ejecucion.Fallo("local", destino, "no se encontró la carpeta local "+r.Carpeta))
		return
	}
	if _, err := os.Stat(filepath.Join(r.Carpeta, r.Ejecutable)); err != nil {
		agregar(ejecucion.Fallo("local", destino, "falta el ejecutable "+r.Carpeta+"/"+r.Ejecutable))
		return
	}
	dir := dirRemoto(r)

	// 1. Preparar la carpeta (si el servicio ya existía, se detiene y se reemplaza).
	preparar := fmt.Sprintf(`systemctl stop %[1]s 2>/dev/null || true
rm -rf %[2]s
mkdir -p %[2]s
echo "carpeta %[2]s lista"`, r.Nombre, dir)
	if !agregar(ejecucion.EjecutarSSH(host, "root", llaveRoot, preparar)) {
		return
	}

	// 2. Transferencia: se copia cada archivo o carpeta del recurso.
	for _, e := range entradas {
		if strings.HasPrefix(e.Name(), ".") || e.Name() == "LEEME.txt" {
			continue // .gitkeep, instrucciones y similares
		}
		if !agregar(ejecucion.CopiarArchivo(filepath.Join(r.Carpeta, e.Name()), host, "root", llaveRoot, dir+"/")) {
			return
		}
	}

	// 3. Permisos: lectura para todos, ejecución para el programa.
	permisos := fmt.Sprintf("chown -R root:root %[1]s\nchmod -R u=rwX,go=rX %[1]s\n", dir)
	if strings.HasSuffix(r.Ejecutable, ".sh") {
		// Por si el script llegó con saltos de línea de Windows (CRLF).
		permisos += fmt.Sprintf("sed -i 's/\\r$//' %s/%s\n", dir, r.Ejecutable)
	}
	permisos += fmt.Sprintf("chmod 755 %[1]s/%[2]s\nls -l %[1]s", dir, r.Ejecutable)
	if !agregar(ejecucion.EjecutarSSH(host, "root", llaveRoot, permisos)) {
		return
	}

	// 4. Unidad de systemd y recarga.
	unidad := fmt.Sprintf(`set -e
cat > /etc/systemd/system/%[1]s.service <<'FIN'
%[2]sFIN
systemctl daemon-reload
echo "unidad /etc/systemd/system/%[1]s.service creada:"
cat /etc/systemd/system/%[1]s.service`, r.Nombre, r.Unidad)
	if !agregar(ejecucion.EjecutarSSH(host, "root", llaveRoot, unidad)) {
		return
	}
	exito = true
	return
}

// Accion ejecuta uno de los seis botones sobre el servicio y muestra su estado.
func Accion(host, llaveRoot, nombre, accion string) ejecucion.Resultado {
	destino := "root@" + host
	if _, ok := recursos[nombre]; !ok {
		return ejecucion.Fallo("ssh", destino, "recurso desconocido: "+nombre)
	}
	if !acciones[accion] {
		return ejecucion.Fallo("ssh", destino, "acción no permitida: "+accion)
	}
	estado := fmt.Sprintf(`echo "activo: $(systemctl is-active %[1]s) | habilitado: $(systemctl is-enabled %[1]s 2>/dev/null; true)"`, nombre)

	var comando string
	if accion == "status" {
		// status devuelve 3 cuando el servicio está detenido: eso no es un error de la consola.
		comando = fmt.Sprintf("systemctl status %[1]s --no-pager -l -n 15\nc=$?\necho\n%[2]s\n[ $c -le 3 ] || exit $c", nombre, estado)
	} else {
		comando = fmt.Sprintf("systemctl %s %s || exit $?\n%s", accion, nombre, estado)
	}
	return ejecucion.EjecutarSSH(host, "root", llaveRoot, comando)
}
