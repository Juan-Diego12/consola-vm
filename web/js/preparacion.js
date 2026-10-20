// Página Preparación del sistema: llaves de root, paquetes, /etc/issue y hostname.
"use strict";
(() => {
  const { $, RE, marcar } = Consola;
  Consola.marco("Preparación del sistema", "engranaje");
  Consola.iniciarVM("base", "NP2-Base");

  const q = (c) => `vm=${encodeURIComponent(c.vm)}&ip=${encodeURIComponent(c.ip)}`;

  // ---------- Llaves de root ----------
  $("btn-llaves").addEventListener("click", async () => {
    const vm = $("vm").value.trim();
    if (!marcar($("vm"), RE.vm.test(vm))) {
      Consola.mostrar({ exito: false, mensaje: "Escriba el nombre de la VM tal como aparece en VirtualBox." });
      return;
    }
    const r = await Consola.ejecutar(`Configurando las llaves de root en ${vm} (Guest Additions y luego SSH)...`,
      "POST", "/api/llaves/root", { vm, password: $("pw-root").value, reemplazar: $("reemplazar").checked });
    $("pw-root").value = "";
    if (r.datos && r.datos.ip) {
      $("ip").value = r.datos.ip;
      $("ip").dispatchEvent(new Event("change"));
      marcar($("ip"), true);
    }
  });

  $("btn-descargar").addEventListener("click", () => {
    const vm = $("vm").value.trim();
    if (!marcar($("vm"), RE.vm.test(vm))) {
      Consola.mostrar({ exito: false, mensaje: "Nombre de VM no válido." });
      return;
    }
    Consola.descargarLlave(vm, "root");
  });

  // ---------- Paquetes ----------
  async function paquete(accion) {
    const c = Consola.vmActual();
    if (!c) return;
    const p = $("paquete").value.trim().toLowerCase();
    $("paquete").value = p;
    if (!marcar($("paquete"), RE.paquete.test(p))) {
      Consola.mostrar({ exito: false, mensaje: "Nombre de paquete no válido: use minúsculas, números y los signos + - . (por ejemplo: tree)." });
      return;
    }
    if (accion === "consultar") {
      await Consola.ejecutar(`Consultando el paquete ${p}...`, "GET", `/api/sistema/paquete?${q(c)}&paquete=${encodeURIComponent(p)}`);
      return;
    }
    if (accion === "eliminar" && !confirm(`¿Eliminar el paquete «${p}» de ${c.vm}?`)) return;
    const verbo = accion === "instalar" ? "Instalando" : "Eliminando";
    await Consola.ejecutar(`${verbo} ${p} en ${c.vm} con apt-get...`, "POST", "/api/sistema/paquete", { ...c, paquete: p, accion });
  }
  $("btn-instalar").addEventListener("click", () => paquete("instalar"));
  $("btn-eliminar").addEventListener("click", () => paquete("eliminar"));
  $("btn-consultar").addEventListener("click", () => paquete("consultar"));

  // ---------- /etc/issue ----------
  $("btn-issue").addEventListener("click", async () => {
    const c = Consola.vmActual();
    if (!c) return;
    const opciones = {
      hostname: $("ch-host").checked, fecha: $("ch-fecha").checked,
      hora: $("ch-hora").checked, direccionIP: $("ch-ip").checked,
      mensaje: $("mensaje").value.trim(),
    };
    const ninguno = !opciones.hostname && !opciones.fecha && !opciones.hora && !opciones.direccionIP && !opciones.mensaje;
    if (ninguno) {
      Consola.mostrar({ exito: false, mensaje: "Seleccione al menos un dato o escriba un mensaje personalizado." });
      return;
    }
    if (!marcar($("mensaje"), opciones.mensaje.length <= 120 && !/[\u0000-\u001f\u007f]/.test(opciones.mensaje))) {
      Consola.mostrar({ exito: false, mensaje: "El mensaje admite máximo 120 caracteres y sin saltos de línea." });
      return;
    }
    const r = await Consola.ejecutar(`Actualizando /etc/issue en ${c.vm}...`, "POST", "/api/sistema/issue", { ...c, ...opciones });
    if (r.exito) Consola.mostrar(r, r.mensaje + ". La pantalla de entrada de la VM ya muestra el nuevo mensaje.", true);
  });

  // ---------- Estado actual (útil para mostrar el "antes" y el "después") ----------
  $("btn-info").addEventListener("click", async () => {
    const c = Consola.vmActual();
    if (!c) return;
    Consola.ocupado(true, `Consultando el estado de ${c.vm}...`);
    const r = await Consola.llamar("GET", `/api/sistema/info?${q(c)}`);
    Consola.ocupado(false);
    if (r.exito && r.datos) {
      const d = r.datos;
      Consola.mostrar(r, `Nombre del host: ${d.hostname || "?"}\n/etc/hosts: ${d.hosts || "(sin línea 127.0.1.1)"}\n` +
        `Debian: ${d.debian || "?"}\n\n/etc/issue:\n${d.issue || "(vacío)"}`);
    } else {
      Consola.mostrar(r);
    }
  });

  // ---------- Hostname ----------
  $("btn-hostname").addEventListener("click", async () => {
    const c = Consola.vmActual();
    if (!c) return;
    const h = $("hostname").value.trim().toLowerCase();
    $("hostname").value = h;
    if (!marcar($("hostname"), RE.hostname.test(h))) {
      Consola.mostrar({ exito: false, mensaje: "Nombre de host no válido: minúsculas, números y guiones, sin empezar ni terminar en guion (máximo 63)." });
      return;
    }
    await Consola.ejecutar(`Cambiando el nombre de ${c.vm} a ${h}...`, "POST", "/api/sistema/hostname", { ...c, hostname: h });
  });
})();
