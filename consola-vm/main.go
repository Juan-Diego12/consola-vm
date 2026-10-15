package main

import (
	"log"
	"net/http"
	"os"

	"consola-vm/internal/api"
	"consola-vm/internal/ejecucion"
)

func main() {
	if err := ejecucion.IniciarLog("logs/operaciones.log"); err != nil {
		log.Fatal("no se pudo abrir el log: ", err)
	}
	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "8080"
	}

	mux := http.NewServeMux()
	api.Registrar(mux)                                // endpoints JSON de los módulos (Diego)
	mux.Handle("/", http.FileServer(http.Dir("web"))) // páginas de la interfaz (Daniel)

	// Solo escucha en el propio PC: la consola tiene control total sobre las VMs.
	direccion := "127.0.0.1:" + puerto
	log.Printf("Consola disponible en http://%s", direccion)
	log.Fatal(http.ListenAndServe(direccion, mux))
}
