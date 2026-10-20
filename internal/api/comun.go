package api

import (
	"encoding/json"
	"net/http"
	"os"

	"consola-vm/internal/haproxy"
	"consola-vm/internal/llaves"
)

// leerJSON decodifica el cuerpo de la petición; si falla, responde 400.
func leerJSON(w http.ResponseWriter, r *http.Request, destino any) bool {
	if err := json.NewDecoder(r.Body).Decode(destino); err != nil {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: "datos inválidos: se esperaba JSON"})
		return false
	}
	return true
}

// conexionRoot valida la VM y la IP y devuelve la llave de root de esa VM.
// La IP llega siempre desde la interfaz (el router la asigna y puede cambiar).
// Si hay un problema, responde 400 y devuelve ok = false.
func conexionRoot(w http.ResponseWriter, vm, ip string) (llave string, ok bool) {
	switch {
	case !llaves.NombreValido(vm):
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: "nombre de VM no válido"})
		return "", false
	case !haproxy.IPv4Valida(ip):
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: "la IP de la VM no es una IPv4 válida"})
		return "", false
	}
	llave = llaves.RutaPrivada(vm, "root")
	if _, err := os.Stat(llave); err != nil {
		responder(w, http.StatusBadRequest, Respuesta{Mensaje: "no hay llave de root para la VM " + vm +
			": configure primero el acceso por llaves en Preparación del sistema"})
		return "", false
	}
	return llave, true
}
