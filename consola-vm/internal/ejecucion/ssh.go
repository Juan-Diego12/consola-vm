package ejecucion

import "fmt"

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
	}
}

// EjecutarSSH ejecuta un comando en la VM por SSH con la llave privada indicada.
func EjecutarSSH(host, usuario, llave, comando string) Resultado {
	destino := usuario + "@" + host
	args := append(opcionesSSH(llave), destino, comando)
	visible := fmt.Sprintf("ssh -i %s -o BatchMode=yes %s %q", llave, destino, comando)
	return ejecutar("ssh", destino, visible, ProgramaSSH, args...)
}

// CopiarArchivo copia un archivo local a la VM con scp (rutaDestino es la ruta en la VM).
func CopiarArchivo(origen, host, usuario, llave, rutaDestino string) Resultado {
	destino := usuario + "@" + host
	remoto := destino + ":" + rutaDestino
	args := append(opcionesSSH(llave), origen, remoto)
	visible := fmt.Sprintf("scp -i %s -o BatchMode=yes %s %s", llave, origen, remoto)
	return ejecutar("scp", destino, visible, ProgramaSCP, args...)
}
