// Package api expone los módulos como endpoints JSON que consume la interfaz.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"consola-vm/internal/ejecucion"
	"consola-vm/internal/llaves"
)

// Respuesta es el formato común de todos los endpoints.
type Respuesta struct {
	Exito   bool                  `json:"exito"`
	Mensaje string                `json:"mensaje"`
	Datos   map[string]any        `json:"datos,omitempty"`
	Pasos   []ejecucion.Resultado `json:"pasos,omitempty"`
}

func responder(w http.ResponseWriter, estado int, r Respuesta) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(estado)
	json.NewEncoder(w).Encode(r)
}

// Registrar agrega las rutas de la API al mux.
func Registrar(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/vm/ip", obtenerIP)
	mux.HandleFunc("POST /api/llaves/root", configurarRoot)
	mux.HandleFunc("GET /api/llaves/descargar", descargarLlave)
}

// GET /api/vm/ip?vm=NombreVM
func obtenerIP(w http.ResponseWriter, r *http.Request) {
	vm := r.URL.Query().Get("vm")
	if !llaves.NombreValido(vm) {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: "nombre de VM no válido"})
		return
	}
	ip, res := ejecucion.ObtenerIP(vm)
	responder(w, http.StatusOK, Respuesta{Exito: res.Exito, Mensaje: res.Errores,
		Datos: map[string]any{"ip": ip}, Pasos: []ejecucion.Resultado{res}})
}

// POST /api/llaves/root   JSON: {"vm": "...", "password": "...", "reemplazar": false}
func configurarRoot(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		VM         string `json:"vm"`
		Password   string `json:"password"`
		Reemplazar bool   `json:"reemplazar"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: "datos inválidos"})
		return
	}
	if !llaves.NombreValido(entrada.VM) || entrada.Password == "" {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: "indique un nombre de VM válido y la contraseña de root"})
		return
	}
	ip, pasos, exito := llaves.ConfigurarRoot(entrada.VM, entrada.Password, entrada.Reemplazar)
	mensaje := "acceso por llaves configurado; autenticación por contraseña desactivada"
	if !exito {
		mensaje = fmt.Sprintf("el proceso se detuvo en el paso %d; revise el detalle", len(pasos))
	}
	responder(w, http.StatusOK, Respuesta{Exito: exito, Mensaje: mensaje,
		Datos: map[string]any{"ip": ip}, Pasos: pasos})
}

// GET /api/llaves/descargar?vm=NombreVM&usuario=root
func descargarLlave(w http.ResponseWriter, r *http.Request) {
	vm, usuario := r.URL.Query().Get("vm"), r.URL.Query().Get("usuario")
	if usuario == "" {
		usuario = "root"
	}
	if !llaves.NombreValido(vm) || !llaves.NombreValido(usuario) {
		http.Error(w, "parámetros no válidos", http.StatusBadRequest)
		return
	}
	ruta := llaves.RutaPrivada(vm, usuario)
	if _, err := os.Stat(ruta); err != nil {
		http.Error(w, "la llave no existe", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(ruta)+`"`)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, ruta)
}
