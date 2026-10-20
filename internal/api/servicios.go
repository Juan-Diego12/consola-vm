package api

import (
	"net/http"

	"consola-vm/internal/ejecucion"
	"consola-vm/internal/servicios"
)

// GET /api/servicios/recursos
func listarRecursos(w http.ResponseWriter, r *http.Request) {
	responder(w, http.StatusOK, Respuesta{Exito: true, Mensaje: "recursos disponibles",
		Datos: map[string]any{"recursos": servicios.Lista()}})
}

// POST /api/servicios/desplegar
// JSON: {"vm","ip","recurso"}   recurso: "catalogo" o "fechahora"
func desplegarServicio(w http.ResponseWriter, r *http.Request) {
	var e struct {
		VM      string `json:"vm"`
		IP      string `json:"ip"`
		Recurso string `json:"recurso"`
	}
	if !leerJSON(w, r, &e) {
		return
	}
	llave, ok := conexionRoot(w, e.VM, e.IP)
	if !ok {
		return
	}
	pasos, exito := servicios.Desplegar(e.IP, llave, e.Recurso)
	responder(w, http.StatusOK, Respuesta{Exito: exito,
		Mensaje: mensajeFinal(exito, "servicio "+e.Recurso+" instalado; use los botones para iniciarlo", pasos),
		Pasos:   pasos})
}

// POST /api/servicios/accion
// JSON: {"vm","ip","recurso","accion"}   accion: start|stop|restart|status|enable|disable
func accionServicio(w http.ResponseWriter, r *http.Request) {
	var e struct {
		VM      string `json:"vm"`
		IP      string `json:"ip"`
		Recurso string `json:"recurso"`
		Accion  string `json:"accion"`
	}
	if !leerJSON(w, r, &e) {
		return
	}
	llave, ok := conexionRoot(w, e.VM, e.IP)
	if !ok {
		return
	}
	res := servicios.Accion(e.IP, llave, e.Recurso, e.Accion)
	pasos := []ejecucion.Resultado{res}
	responder(w, http.StatusOK, Respuesta{Exito: res.Exito,
		Mensaje: mensajeFinal(res.Exito, e.Accion+" "+e.Recurso+": listo", pasos), Pasos: pasos})
}
