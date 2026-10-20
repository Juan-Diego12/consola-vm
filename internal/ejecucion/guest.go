package ejecucion

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// marcaFin la imprime la VM al terminar el comando, seguida de su código de salida real.
const marcaFin = "__CONSOLA_VM_FIN__:"

// EjecutarGuest ejecuta un comando de bash dentro de la VM usando las Guest Additions
// (VBoxManage guestcontrol). Solo se usa para el arranque inicial, cuando todavía no
// hay llaves SSH. Requiere la VM encendida y con Guest Additions instaladas.
//
// Compatibilidad con VirtualBox 7.1 en Windows:
//   - VBoxManage 7.1.6 se cae (0xC0000005) si el argumento trae saltos de línea, así que el
//     script viaja codificado en base64 (una sola línea, sin comillas) y la VM lo decodifica.
//   - VBoxManage puede caerse al terminar (0xC0000374) aunque el comando haya funcionado.
//     Por eso la VM imprime al final una marca con el código real del comando, y ese código
//     es el que decide si la operación tuvo éxito.
func EjecutarGuest(vm, usuario, password, comando string) Resultado {
	codificado := base64.StdEncoding.EncodeToString([]byte(comando))
	envoltura := "echo " + codificado + " | base64 -d | bash; echo " + marcaFin + "$?"
	args := []string{
		"guestcontrol", vm, "run",
		"--username", usuario,
		"--password", password,
		"--exe", "/bin/bash",
		"--", "-c", envoltura, // VirtualBox 7.1 y 7.2 toman argv[0] de --exe; aquí van solo los argumentos
	}
	// La contraseña nunca se guarda en el log ni se devuelve a la interfaz.
	visible := fmt.Sprintf("VBoxManage guestcontrol %s run --username %s --password **** --exe /bin/bash -- -c \"echo <script en base64> | base64 -d | bash\"\n# script:\n%s",
		vm, usuario, comando)
	r := correr("guest", vm, visible, "", VBoxManage, args...)
	interpretarMarca(&r)
	registrar(r)
	return r
}

// interpretarMarca busca la marca de fin en la salida, la quita y usa el código real del
// comando en la VM. Si VBoxManage terminó con error pero el comando sí funcionó, deja un aviso.
func interpretarMarca(r *Resultado) {
	i := strings.LastIndex(r.Salida, marcaFin)
	if i < 0 {
		return // la VM no llegó a imprimir la marca: se conserva el resultado de VBoxManage
	}
	texto := strings.TrimSpace(r.Salida[i+len(marcaFin):])
	if j := strings.IndexAny(texto, "\r\n "); j >= 0 {
		texto = texto[:j]
	}
	codigoVM, err := strconv.Atoi(texto)
	if err != nil {
		return
	}
	r.Salida = strings.TrimSpace(r.Salida[:i])
	if r.Codigo != 0 && codigoVM == 0 {
		aviso := fmt.Sprintf("aviso: VBoxManage terminó con código %d (0x%X) después de completar el comando; "+
			"el comando en la VM terminó bien (código 0). Es una falla conocida de VBoxManage en Windows.",
			r.Codigo, uint32(r.Codigo))
		r.Errores = strings.TrimSpace(r.Errores + "\n" + aviso)
	}
	r.Codigo = codigoVM
	r.Exito = codigoVM == 0
}

// ObtenerIP consulta a las Guest Additions la IPv4 del primer adaptador de la VM
// (con adaptador puente es la IP de la red física).
func ObtenerIP(vm string) (string, Resultado) {
	r := EjecutarLocal(vm, VBoxManage, "guestproperty", "get", vm, "/VirtualBox/GuestInfo/Net/0/V4/IP")
	// Salida esperada: "Value: 192.168.20.30"   Si no hay dato: "No value set!"
	if !r.Exito || !strings.HasPrefix(r.Salida, "Value:") {
		r.Exito = false
		if r.Errores == "" {
			r.Errores = "no se pudo obtener la IP: verifique que la VM esté encendida y tenga Guest Additions"
		}
		return "", r
	}
	return strings.TrimSpace(strings.TrimPrefix(r.Salida, "Value:")), r
}
