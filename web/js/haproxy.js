// Página Balanceador de carga: servidores del backend de HAProxy y prueba del reparto.
"use strict";
(() => {
  const { $, RE, marcar } = Consola;
  Consola.marco("Balanceador de carga (HAProxy)", "balanceo");
  Consola.iniciarVM("haproxy", "NP2-HAProxy");
  let puertoFrontend = 80;

  function enlace() {
    const ip = $("ip").value.trim();
    $("lnk-haproxy").href = RE.ipv4.test(ip) ? `http://${ip}${puertoFrontend === 80 ? "" : ":" + puertoFrontend}` : "#";
  }
  $("ip").addEventListener("change", enlace);

  function pintar(datos) {
    if (!datos) return;
    if (datos.puertoFrontend) {
      puertoFrontend = datos.puertoFrontend;
      $("puerto-front").textContent = puertoFrontend;
    }
    const lista = datos.servidores || [];
    const tabla = $("s-tabla");
    tabla.innerHTML = "";
    if (!lista.length) {
      tabla.innerHTML = '<tr><td colspan="4" class="vacio">El backend no tiene servidores.</td></tr>';
      return;
    }
    lista.forEach((s) => {
      const fila = document.createElement("tr");
      [s.nombre, s.ip, s.puerto].forEach((v) => {
        const td = document.createElement("td");
        td.textContent = v;
        fila.append(td);
      });
      const td = document.createElement("td");
      const b = document.createElement("button");
      b.className = "boton rojo";
      b.type = "button";
      b.textContent = "Quitar";
      b.addEventListener("click", () => quitar(s.nombre));
      td.append(b);
      fila.append(td);
      tabla.append(fila);
    });
    enlace();
  }

  async function listar() {
    const c = Consola.vmActual();
    if (!c) return;
    const r = await Consola.ejecutar(`Leyendo /etc/haproxy/haproxy.cfg en ${c.vm}...`, "GET",
      `/api/haproxy/servidores?vm=${encodeURIComponent(c.vm)}&ip=${encodeURIComponent(c.ip)}`);
    if (r.exito) pintar(r.datos);
  }
  $("btn-servidores").addEventListener("click", listar);

  $("btn-agregar").addEventListener("click", async () => {
    const c = Consola.vmActual();
    if (!c) return;
    const nombre = $("s-nombre").value.trim(), ipServidor = $("s-ip").value.trim();
    const puerto = Number($("s-puerto").value);
    const okN = marcar($("s-nombre"), RE.servidor.test(nombre));
    const okI = marcar($("s-ip"), RE.ipv4.test(ipServidor));
    const okP = marcar($("s-puerto"), Number.isInteger(puerto) && puerto >= 1 && puerto <= 65535);
    if (!okN || !okI || !okP) {
      Consola.mostrar({ exito: false, mensaje: "Revise los datos: nombre (letras, números, guion o guion bajo, máx. 32), IPv4 válida y puerto entre 1 y 65535." });
      return;
    }
    const r = await Consola.ejecutar(`Agregando ${nombre} (${ipServidor}:${puerto}) al backend...`, "POST", "/api/haproxy/agregar",
      { ...c, nombre, ipServidor, puerto });
    if (r.exito) { pintar(r.datos); $("s-nombre").value = ""; $("s-ip").value = ""; }
  });

  async function quitar(nombre) {
    const c = Consola.vmActual();
    if (!c || !confirm(`¿Quitar el servidor «${nombre}» del backend?`)) return;
    const r = await Consola.ejecutar(`Quitando ${nombre} del backend...`, "POST", "/api/haproxy/quitar", { ...c, nombre });
    if (r.exito) pintar(r.datos);
  }

  $("btn-probar").addEventListener("click", async () => {
    const c = Consola.vmActual();
    if (!c) return;
    const n = Math.min(20, Math.max(1, Number($("n-peticiones").value) || 6));
    const r = await Consola.ejecutar(`Enviando ${n} peticiones a través de HAProxy (${c.ip})...`, "GET",
      `/api/haproxy/probar?ip=${encodeURIComponent(c.ip)}&n=${n}`);
    const respuestas = (r.datos && r.datos.respuestas) || [];
    // Resumen: cuántas peticiones atendió cada servidor.
    const conteo = {};
    respuestas.forEach((linea) => {
      const cuerpo = linea.replace(/^\d+:\s*/, "").replace(/^HTTP \d+\s*/, "");
      let quien = cuerpo;
      try {
        const j = JSON.parse(cuerpo);
        quien = typeof j === "string" ? j : (Object.values(j).find((v) => typeof v === "string") || cuerpo);
      } catch { /* no es JSON: se usa el texto tal cual */ }
      conteo[quien] = (conteo[quien] || 0) + 1;
    });
    const resumen = Object.entries(conteo).map(([k, v]) => `  ${k}: ${v} petición(es)`).join("\n");
    $("consola-balanceo").textContent = respuestas.length
      ? respuestas.join("\n") + "\n\nResumen por servidor:\n" + resumen
      : r.mensaje || "Sin respuestas.";
  });

  enlace();
})();
