package api

import (
	"net/http"

	"consola-vm/internal/ejecucion"
	"consola-vm/internal/sistema"
)

// RegistrarSistema agrega las rutas del módulo Preparación del sistema (Daniel).
// Se registra aparte para no modificar api.go.
func RegistrarSistema(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sistema/info", sisInfo)
	mux.HandleFunc("GET /api/sistema/paquete", sisEstadoPaquete)
	mux.HandleFunc("POST /api/sistema/paquete", sisPaquete)
	mux.HandleFunc("POST /api/sistema/issue", sisIssue)
	mux.HandleFunc("POST /api/sistema/hostname", sisHostname)
}

// GET /api/sistema/info?vm=NP2-Base&ip=192.168.20.27
func sisInfo(w http.ResponseWriter, r *http.Request) {
	vm, ip := r.URL.Query().Get("vm"), r.URL.Query().Get("ip")
	llave, ok := conexionRoot(w, vm, ip)
	if !ok {
		return
	}
	datos, res := sistema.Info(ip, llave)
	pasos := []ejecucion.Resultado{res}
	info := map[string]any{}
	for k, v := range datos {
		info[k] = v
	}
	responder(w, http.StatusOK, Respuesta{Exito: res.Exito,
		Mensaje: mensajeFinal(res.Exito, "estado actual del sistema", pasos), Datos: info, Pasos: pasos})
}

// GET /api/sistema/paquete?vm=X&ip=Y&paquete=tree
func sisEstadoPaquete(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	paquete := q.Get("paquete")
	if err := sistema.ValidarPaquete(paquete, "consultar"); err != nil {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: err.Error()})
		return
	}
	llave, ok := conexionRoot(w, q.Get("vm"), q.Get("ip"))
	if !ok {
		return
	}
	res := sistema.EstadoPaquete(q.Get("ip"), llave, paquete)
	pasos := []ejecucion.Resultado{res}
	responder(w, http.StatusOK, Respuesta{Exito: res.Exito,
		Mensaje: mensajeFinal(res.Exito, res.Salida, pasos), Pasos: pasos})
}

// POST /api/sistema/paquete   JSON: {"vm","ip","paquete","accion"}   accion: instalar | eliminar
func sisPaquete(w http.ResponseWriter, r *http.Request) {
	var e struct {
		VM      string `json:"vm"`
		IP      string `json:"ip"`
		Paquete string `json:"paquete"`
		Accion  string `json:"accion"`
	}
	if !leerJSON(w, r, &e) {
		return
	}
	if e.Accion != "instalar" && e.Accion != "eliminar" {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: "acción no válida: use instalar o eliminar"})
		return
	}
	if err := sistema.ValidarPaquete(e.Paquete, e.Accion); err != nil {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: err.Error()})
		return
	}
	llave, ok := conexionRoot(w, e.VM, e.IP)
	if !ok {
		return
	}
	pasos, exito := sistema.Paquete(e.IP, llave, e.Paquete, e.Accion)
	texto := "paquete " + e.Paquete + " instalado"
	if e.Accion == "eliminar" {
		texto = "paquete " + e.Paquete + " eliminado"
	}
	responder(w, http.StatusOK, Respuesta{Exito: exito, Mensaje: mensajeFinal(exito, texto, pasos), Pasos: pasos})
}

// POST /api/sistema/issue   JSON: {"vm","ip","hostname":bool,"fecha":bool,"hora":bool,"direccionIP":bool,"mensaje"}
func sisIssue(w http.ResponseWriter, r *http.Request) {
	var e struct {
		VM          string `json:"vm"`
		IP          string `json:"ip"`
		Hostname    bool   `json:"hostname"`
		Fecha       bool   `json:"fecha"`
		Hora        bool   `json:"hora"`
		DireccionIP bool   `json:"direccionIP"`
		Mensaje     string `json:"mensaje"`
	}
	if !leerJSON(w, r, &e) {
		return
	}
	opciones := sistema.OpcionesIssue{Hostname: e.Hostname, Fecha: e.Fecha, Hora: e.Hora, IP: e.DireccionIP, Mensaje: e.Mensaje}
	if _, err := sistema.ContenidoIssue(opciones); err != nil {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: err.Error()})
		return
	}
	llave, ok := conexionRoot(w, e.VM, e.IP)
	if !ok {
		return
	}
	pasos, exito := sistema.Issue(e.IP, llave, opciones)
	responder(w, http.StatusOK, Respuesta{Exito: exito,
		Mensaje: mensajeFinal(exito, "mensaje de entrada (/etc/issue) actualizado", pasos), Pasos: pasos})
}

// POST /api/sistema/hostname   JSON: {"vm","ip","hostname"}
func sisHostname(w http.ResponseWriter, r *http.Request) {
	var e struct {
		VM       string `json:"vm"`
		IP       string `json:"ip"`
		Hostname string `json:"hostname"`
	}
	if !leerJSON(w, r, &e) {
		return
	}
	nuevo, err := sistema.ValidarHostname(e.Hostname)
	if err != nil {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: err.Error()})
		return
	}
	llave, ok := conexionRoot(w, e.VM, e.IP)
	if !ok {
		return
	}
	pasos, exito := sistema.Hostname(e.IP, llave, nuevo)
	responder(w, http.StatusOK, Respuesta{Exito: exito,
		Mensaje: mensajeFinal(exito, "nombre del equipo cambiado a "+nuevo+" (hostname y /etc/hosts)", pasos),
		Datos:   map[string]any{"hostname": nuevo}, Pasos: pasos})
}
