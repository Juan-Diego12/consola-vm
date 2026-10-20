package api

import (
	"net/http"

	"consola-vm/internal/ejecucion"
	"consola-vm/internal/usuarios"
)

// GET /api/usuarios?vm=NP2-Base&ip=192.168.20.27
func listarUsuarios(w http.ResponseWriter, r *http.Request) {
	vm, ip := r.URL.Query().Get("vm"), r.URL.Query().Get("ip")
	llave, ok := conexionRoot(w, vm, ip)
	if !ok {
		return
	}
	lista, res := usuarios.Listar(vm, ip, llave)
	responder(w, http.StatusOK, Respuesta{Exito: res.Exito,
		Mensaje: mensajeFinal(res.Exito, "usuarios obtenidos", []ejecucion.Resultado{res}),
		Datos:   map[string]any{"usuarios": lista}, Pasos: []ejecucion.Resultado{res}})
}

// POST /api/usuarios/crear
// JSON: {"vm","ip","nombre","password","sudo":bool,"llaves":bool}
func crearUsuario(w http.ResponseWriter, r *http.Request) {
	var e struct {
		VM       string `json:"vm"`
		IP       string `json:"ip"`
		Nombre   string `json:"nombre"`
		Password string `json:"password"`
		Sudo     bool   `json:"sudo"`
		Llaves   bool   `json:"llaves"`
	}
	if !leerJSON(w, r, &e) {
		return
	}
	llave, ok := conexionRoot(w, e.VM, e.IP)
	if !ok {
		return
	}
	pasos, exito := usuarios.Crear(e.VM, e.IP, llave, usuarios.DatosCrear{
		Nombre: e.Nombre, Password: e.Password, Sudo: e.Sudo, Llaves: e.Llaves})
	responder(w, http.StatusOK, Respuesta{Exito: exito,
		Mensaje: mensajeFinal(exito, "usuario "+e.Nombre+" creado", pasos),
		Datos:   map[string]any{"nombre": e.Nombre, "llave": exito && e.Llaves}, Pasos: pasos})
}

// POST /api/usuarios/eliminar
// JSON: {"vm","ip","nombre","conArchivos":bool}
func eliminarUsuario(w http.ResponseWriter, r *http.Request) {
	var e struct {
		VM          string `json:"vm"`
		IP          string `json:"ip"`
		Nombre      string `json:"nombre"`
		ConArchivos bool   `json:"conArchivos"`
	}
	if !leerJSON(w, r, &e) {
		return
	}
	llave, ok := conexionRoot(w, e.VM, e.IP)
	if !ok {
		return
	}
	pasos, exito := usuarios.Eliminar(e.VM, e.IP, llave, e.Nombre, e.ConArchivos)
	responder(w, http.StatusOK, Respuesta{Exito: exito,
		Mensaje: mensajeFinal(exito, "usuario "+e.Nombre+" eliminado", pasos), Pasos: pasos})
}
