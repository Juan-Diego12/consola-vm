// Página Gestión de servicios (systemd): instalar recursos como servicios y controlarlos.
"use strict";
(() => {
  const { $ } = Consola;
  Consola.marco("Gestión de Servicios", "servicios");
  Consola.iniciarVM("base", "NP2-Base");
  let recursos = [];
  const NOMBRES = { start: "Iniciar", stop: "Detener", restart: "Reiniciar", status: "Estado", enable: "Habilitar", disable: "Deshabilitar" };

  const actual = () => recursos.find((r) => r.nombre === $("recurso").value);

  function describir() {
    const r = actual();
    $("recurso-desc").textContent = r ? r.descripcion : "";
    const lnk = $("lnk-app");
    const ip = $("ip").value.trim();
    if (r && r.puerto && Consola.RE.ipv4.test(ip)) {
      lnk.href = `http://${ip}:${r.puerto}`;
      lnk.textContent = `Abrir la aplicación (puerto ${r.puerto})`;
      lnk.hidden = false;
    } else {
      lnk.hidden = true;
    }
  }

  async function cargarRecursos() {
    const r = await Consola.llamar("GET", "/api/servicios/recursos");
    const sel = $("recurso");
    sel.innerHTML = "";
    recursos = (r.datos && r.datos.recursos) || [];
    if (!recursos.length) {
      sel.innerHTML = '<option value="">(no hay recursos)</option>';
      Consola.mostrar(r.exito ? { exito: false, mensaje: "La consola no tiene recursos disponibles en recursos/." } : r);
      return;
    }
    recursos.forEach((x) => {
      const op = document.createElement("option");
      op.value = x.nombre;
      op.textContent = x.nombre;
      sel.append(op);
    });
    describir();
  }
  $("recurso").addEventListener("change", describir);
  $("ip").addEventListener("change", describir);

  $("btn-unidad").addEventListener("click", () => {
    const r = actual();
    if (!r) return;
    $("consola-servicio").textContent = `# /etc/systemd/system/${r.nombre}.service (se crea al instalar)\n\n${r.unidad}`;
  });

  $("btn-desplegar").addEventListener("click", async () => {
    const c = Consola.vmActual(), r = actual();
    if (!c || !r) return;
    const res = await Consola.ejecutar(`Instalando ${r.nombre} como servicio en ${c.vm}: transferencia, permisos y unidad...`,
      "POST", "/api/servicios/desplegar", { ...c, recurso: r.nombre });
    const ultimo = (res.pasos || []).slice(-1)[0];
    if (ultimo) $("consola-servicio").textContent = [ultimo.salida, ultimo.errores].filter(Boolean).join("\n") || "(sin salida)";
    describir();
  });

  document.querySelectorAll("[data-accion]").forEach((b) => {
    b.addEventListener("click", async () => {
      const c = Consola.vmActual(), r = actual();
      if (!c || !r) return;
      const accion = b.dataset.accion;
      const res = await Consola.ejecutar(`${NOMBRES[accion]}: systemctl ${accion} ${r.nombre} en ${c.vm}...`,
        "POST", "/api/servicios/accion", { ...c, recurso: r.nombre, accion });
      const p = (res.pasos || [])[0];
      $("consola-servicio").textContent = p
        ? `$ systemctl ${accion} ${r.nombre}\n\n` + ([p.salida, p.errores].filter(Boolean).join("\n") || "(sin salida)")
        : res.mensaje || "";
      describir();
    });
  });

  cargarRecursos();
})();
