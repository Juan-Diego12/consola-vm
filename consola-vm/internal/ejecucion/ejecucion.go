// Package ejecucion es la capa que ejecuta comandos en las máquinas virtuales.
// Los módulos (llaves, usuarios, systemd, HAProxy) solo usan estas funciones:
//
//	EjecutarGuest(vm, usuario, password, comando) Resultado  -> arranque inicial (Guest Additions)
//	EjecutarSSH(host, usuario, llave, comando) Resultado     -> todo lo demás
//	CopiarArchivo(origen, host, usuario, llave, destino) Resultado
//
// Cada operación queda registrada en el log técnico (logs/operaciones.log).
package ejecucion

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Resultado es lo que devuelve TODA operación de la capa de ejecución.
// La interfaz lo recibe tal cual en JSON.
type Resultado struct {
	Operacion string `json:"operacion"` // "guest", "ssh", "scp" o "local"
	Destino   string `json:"destino"`   // VM o usuario@host
	Comando   string `json:"comando"`   // comando ejecutado (sin contraseñas)
	Salida    string `json:"salida"`    // salida estándar
	Errores   string `json:"errores"`   // salida de error o descripción del fallo
	Codigo    int    `json:"codigo"`    // código de salida (0 = éxito, -1 = no se pudo ejecutar)
	Duracion  string `json:"duracion"`  // tiempo que tomó
	Exito     bool   `json:"exito"`     // true si Codigo == 0
}

// Configuración (se puede cambiar con variables de entorno).
var (
	VBoxManage   = valorEntorno("VBOXMANAGE", `C:\Program Files\Oracle\VirtualBox\VBoxManage.exe`)
	ProgramaSSH  = valorEntorno("SSH_BIN", "ssh")
	ProgramaSCP  = valorEntorno("SCP_BIN", "scp")
	DirLlaves    = valorEntorno("DIR_LLAVES", "llaves")
	TiempoMaximo = 3 * time.Minute
)

var registro = log.New(os.Stdout, "", log.LstdFlags)

func valorEntorno(clave, porDefecto string) string {
	if v := os.Getenv(clave); v != "" {
		return v
	}
	return porDefecto
}

// IniciarLog abre (o crea) el archivo del log técnico. Los mensajes también salen por consola.
func IniciarLog(ruta string) error {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(ruta, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	registro = log.New(io.MultiWriter(f, os.Stdout), "", log.LstdFlags)
	return nil
}

// EjecutarLocal corre un programa en el PC anfitrión (por ejemplo ssh-keygen o VBoxManage).
func EjecutarLocal(destino, programa string, args ...string) Resultado {
	visible := programa + " " + strings.Join(args, " ")
	return ejecutar("local", destino, visible, programa, args...)
}

// Fallo arma un Resultado de error sin ejecutar nada (validaciones, archivos que faltan, etc.).
func Fallo(operacion, destino, mensaje string) Resultado {
	r := Resultado{Operacion: operacion, Destino: destino, Errores: mensaje, Codigo: -1, Duracion: "0s"}
	registrar(r)
	return r
}

// ejecutar corre el programa con tiempo máximo, captura salidas y registra la operación.
// comandoVisible es lo que se guarda en el log y se muestra en la interfaz (sin contraseñas).
func ejecutar(operacion, destino, comandoVisible, programa string, args ...string) Resultado {
	ctx, cancelar := context.WithTimeout(context.Background(), TiempoMaximo)
	defer cancelar()

	cmd := exec.CommandContext(ctx, programa, args...)
	var salida, errores bytes.Buffer
	cmd.Stdout = &salida
	cmd.Stderr = &errores

	inicio := time.Now()
	err := cmd.Run()

	r := Resultado{
		Operacion: operacion,
		Destino:   destino,
		Comando:   comandoVisible,
		Salida:    strings.TrimSpace(salida.String()),
		Errores:   strings.TrimSpace(errores.String()),
		Duracion:  time.Since(inicio).Round(time.Millisecond).String(),
	}
	if err != nil {
		var errSalida *exec.ExitError
		if errors.As(err, &errSalida) {
			r.Codigo = errSalida.ExitCode()
		} else {
			// El programa ni siquiera arrancó (no existe, ruta mala, etc.).
			r.Codigo = -1
			r.Errores = strings.TrimSpace(r.Errores + "\n" + err.Error())
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			r.Codigo = -1
			r.Errores = strings.TrimSpace(r.Errores + "\ntiempo máximo agotado (" + TiempoMaximo.String() + ")")
		}
	}
	r.Exito = r.Codigo == 0
	registrar(r)
	return r
}

// registrar escribe una línea en el log técnico por cada operación.
func registrar(r Resultado) {
	estado := "OK"
	if !r.Exito {
		estado = "ERROR"
	}
	linea := fmt.Sprintf("[%s] %s | destino=%s | codigo=%d | %s | cmd=%q",
		r.Operacion, estado, r.Destino, r.Codigo, r.Duracion, recortar(r.Comando, 300))
	if !r.Exito && r.Errores != "" {
		linea += fmt.Sprintf(" | errores=%q", recortar(r.Errores, 300))
	}
	registro.Println(linea)
}

func recortar(s string, max int) string {
	if len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "..."
}
