package ejecucion

import (
	"fmt"
	"strings"
)

// EjecutarGuest ejecuta un comando de bash dentro de la VM usando las Guest Additions
// (VBoxManage guestcontrol). Solo se usa para el arranque inicial, cuando todavía no
// hay llaves SSH. Requiere la VM encendida y con Guest Additions instaladas.
func EjecutarGuest(vm, usuario, password, comando string) Resultado {
	args := []string{
		"guestcontrol", vm, "run",
		"--username", usuario,
		"--password", password,
		"--exe", "/bin/bash",
		"--", "-c", comando, // VirtualBox 7.2 toma argv[0] de --exe; aquí van solo los argumentos
	}
	// La contraseña nunca se guarda en el log ni se devuelve a la interfaz.
	visible := fmt.Sprintf("VBoxManage guestcontrol %s run --username %s --password **** --exe /bin/bash -- -c %q",
		vm, usuario, comando)
	return ejecutar("guest", vm, visible, VBoxManage, args...)
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
