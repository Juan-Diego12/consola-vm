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

// mensajeFinal arma el mensaje para el usuario según el resultado de los pasos.
func mensajeFinal(exito bool, textoExito string, pasos []ejecucion.Resultado) string {
	if exito {
		return textoExito
	}
	if len(pasos) > 0 && pasos[len(pasos)-1].Errores != "" {
		return fmt.Sprintf("falló el paso %d: %s", len(pasos), pasos[len(pasos)-1].Errores)
	}
	return fmt.Sprintf("el proceso se detuvo en el paso %d; revise el detalle", len(pasos))
}

// Registrar agrega las rutas de la API al mux.
func Registrar(mux *http.ServeMux) {
	// Llaves de root y utilidades
	mux.HandleFunc("GET /api/vm/ip", obtenerIP)
	mux.HandleFunc("POST /api/llaves/root", configurarRoot)
	mux.HandleFunc("GET /api/llaves/descargar", descargarLlave)
	// Usuarios
	mux.HandleFunc("GET /api/usuarios", listarUsuarios)
	mux.HandleFunc("POST /api/usuarios/crear", crearUsuario)
	mux.HandleFunc("POST /api/usuarios/eliminar", eliminarUsuario)
	// systemd
	mux.HandleFunc("GET /api/servicios/recursos", listarRecursos)
	mux.HandleFunc("POST /api/servicios/desplegar", desplegarServicio)
	mux.HandleFunc("POST /api/servicios/accion", accionServicio)
	// HAProxy
	mux.HandleFunc("GET /api/haproxy/servidores", listarServidores)
	mux.HandleFunc("POST /api/haproxy/agregar", agregarServidor)
	mux.HandleFunc("POST /api/haproxy/quitar", quitarServidor)
	mux.HandleFunc("GET /api/haproxy/probar", probarBalanceo)
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
// Si "password" llega vacío, se usa la variable de entorno VBOX_GUEST_PASSWORD.
func configurarRoot(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		VM         string `json:"vm"`
		Password   string `json:"password"`
		Reemplazar bool   `json:"reemplazar"`
	}
	if !leerJSON(w, r, &entrada) {
		return
	}
	if entrada.Password == "" {
		entrada.Password = os.Getenv("VBOX_GUEST_PASSWORD")
	}
	if !llaves.NombreValido(entrada.VM) || entrada.Password == "" {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: "indique un nombre de VM válido y la contraseña de root (en el formulario o en VBOX_GUEST_PASSWORD)"})
		return
	}
	ip, pasos, exito := llaves.ConfigurarRoot(entrada.VM, entrada.Password, entrada.Reemplazar)
	responder(w, http.StatusOK, Respuesta{Exito: exito,
		Mensaje: mensajeFinal(exito, "acceso por llaves configurado; autenticación por contraseña desactivada", pasos),
		Datos:   map[string]any{"ip": ip}, Pasos: pasos})
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
