package ejecucion

import (
	"fmt"
	"path/filepath"
)

// opcionesSSH son comunes a ssh y scp. BatchMode=yes hace que nunca se pida
// contraseña ni confirmación: si la llave no sirve, falla de inmediato.
func opcionesSSH(llave string) []string {
	return []string{
		"-i", llave,
		"-o", "BatchMode=yes",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=" + DirLlaves + "/known_hosts", // relativo: sin espacios
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR", // oculta avisos informativos como "Permanently added..."
	}
}

// EjecutarSSH ejecuta un comando en la VM por SSH con la llave privada indicada.
func EjecutarSSH(host, usuario, llave, comando string) Resultado {
	destino := usuario + "@" + host
	args := append(opcionesSSH(llave), destino, comando)
	visible := fmt.Sprintf("ssh -i %s -o BatchMode=yes %s %q", llave, destino, comando)
	return ejecutar("ssh", destino, visible, ProgramaSSH, args...)
}

// EjecutarSSHConEntrada es igual a EjecutarSSH, pero envía "entrada" por la entrada
// estándar del comando remoto (por ejemplo, "usuario:clave" para chpasswd).
// La entrada no se guarda en el log ni se devuelve a la interfaz.
func EjecutarSSHConEntrada(host, usuario, llave, comando, entrada string) Resultado {
	destino := usuario + "@" + host
	args := append(opcionesSSH(llave), destino, comando)
	visible := fmt.Sprintf("ssh -i %s -o BatchMode=yes %s %q  (datos por entrada estándar, no registrados)", llave, destino, comando)
	return ejecutarConEntrada("ssh", destino, visible, entrada, ProgramaSSH, args...)
}

// CopiarArchivo copia un archivo o una carpeta local a la VM con scp -r
// (rutaDestino es la ruta en la VM).
func CopiarArchivo(origen, host, usuario, llave, rutaDestino string) Resultado {
	origen = filepath.ToSlash(origen) // scp en Windows maneja mejor las barras normales
	destino := usuario + "@" + host
	remoto := destino + ":" + rutaDestino
	args := append(opcionesSSH(llave), "-r", origen, remoto)
	visible := fmt.Sprintf("scp -r -i %s -o BatchMode=yes %s %s", llave, origen, remoto)
	return ejecutar("scp", destino, visible, ProgramaSCP, args...)
}
