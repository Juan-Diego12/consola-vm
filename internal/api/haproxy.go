package api

import (
	"net/http"
	"strconv"

	"consola-vm/internal/ejecucion"
	"consola-vm/internal/haproxy"
)

// GET /api/haproxy/servidores?vm=NP2-HAProxy&ip=192.168.20.29
func listarServidores(w http.ResponseWriter, r *http.Request) {
	vm, ip := r.URL.Query().Get("vm"), r.URL.Query().Get("ip")
	llave, ok := conexionRoot(w, vm, ip)
	if !ok {
		return
	}
	lista, res := haproxy.Listar(ip, llave)
	responder(w, http.StatusOK, Respuesta{Exito: res.Exito,
		Mensaje: mensajeFinal(res.Exito, "servidores del backend", []ejecucion.Resultado{res}),
		Datos:   map[string]any{"servidores": lista, "puertoFrontend": haproxy.PuertoFrontend},
		Pasos:   []ejecucion.Resultado{res}})
}

// POST /api/haproxy/agregar
// JSON: {"vm","ip","nombre","ipServidor","puerto":8000}   (vm e ip son los de la VM de HAProxy)
func agregarServidor(w http.ResponseWriter, r *http.Request) {
	var e struct {
		VM         string `json:"vm"`
		IP         string `json:"ip"`
		Nombre     string `json:"nombre"`
		IPServidor string `json:"ipServidor"`
		Puerto     int    `json:"puerto"`
	}
	if !leerJSON(w, r, &e) {
		return
	}
	llave, ok := conexionRoot(w, e.VM, e.IP)
	if !ok {
		return
	}
	pasos, lista, exito := haproxy.Agregar(e.IP, llave, haproxy.Servidor{Nombre: e.Nombre, IP: e.IPServidor, Puerto: e.Puerto})
	responder(w, http.StatusOK, Respuesta{Exito: exito,
		Mensaje: mensajeFinal(exito, "servidor "+e.Nombre+" agregado al backend", pasos),
		Datos:   map[string]any{"servidores": lista}, Pasos: pasos})
}

// POST /api/haproxy/quitar
// JSON: {"vm","ip","nombre"}
func quitarServidor(w http.ResponseWriter, r *http.Request) {
	var e struct {
		VM     string `json:"vm"`
		IP     string `json:"ip"`
		Nombre string `json:"nombre"`
	}
	if !leerJSON(w, r, &e) {
		return
	}
	llave, ok := conexionRoot(w, e.VM, e.IP)
	if !ok {
		return
	}
	pasos, lista, exito := haproxy.Quitar(e.IP, llave, e.Nombre)
	responder(w, http.StatusOK, Respuesta{Exito: exito,
		Mensaje: mensajeFinal(exito, "servidor "+e.Nombre+" quitado del backend", pasos),
		Datos:   map[string]any{"servidores": lista}, Pasos: pasos})
}

// GET /api/haproxy/probar?ip=192.168.20.29&n=6
func probarBalanceo(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if !haproxy.IPv4Valida(ip) {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: "la IP de HAProxy no es una IPv4 válida"})
		return
	}
	n, err := strconv.Atoi(r.URL.Query().Get("n"))
	if err != nil || n < 1 || n > 20 {
		n = 6
	}
	respuestas := haproxy.Probar(ip, n)
	responder(w, http.StatusOK, Respuesta{Exito: true, Mensaje: "peticiones enviadas a través de HAProxy",
		Datos: map[string]any{"respuestas": respuestas}})
}
