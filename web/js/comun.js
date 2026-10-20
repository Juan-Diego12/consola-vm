// Utilidades comunes de la consola: encabezado, selección de la VM, llamadas a la API,
// validación en el navegador y presentación de resultados.
// La validación definitiva siempre la hace el servidor; aquí solo se evita enviar datos
// claramente inválidos y se avisa al usuario antes.
"use strict";

const Consola = (() => {
  // ---------- Expresiones de validación (las mismas que usa el servidor) ----------
  const RE = {
    vm: /^[A-Za-z0-9._-]{1,64}$/,
    ipv4: /^(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/,
    paquete: /^[a-z0-9][a-z0-9+.-]{1,62}$/,
    hostname: /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/,
    usuario: /^[a-z_][a-z0-9_-]{0,31}$/,
    servidor: /^[A-Za-z0-9_-]{1,32}$/,
  };

  // ---------- Íconos (SVG en línea, sin archivos externos) ----------
  const ICONOS = {
    nube: '<svg viewBox="0 0 64 40" aria-hidden="true"><path fill="#0d1a6b" d="M50 38H15C7 38 1 32 1 24.5 1 17.6 6.2 12 13 11.2 15.3 4.6 21.6 0 29 0c8.3 0 15.2 5.8 16.8 13.6C56 14.4 63 21 63 27.5 63 33.3 57.3 38 50 38z"/></svg>',
    engranaje: '<svg class="icono" viewBox="0 0 24 24" aria-hidden="true"><path fill="#222" d="M19.4 13a7.6 7.6 0 0 0 0-2l2.1-1.6-2-3.5-2.5 1a7.4 7.4 0 0 0-1.7-1L15 3h-4l-.3 2.9a7.4 7.4 0 0 0-1.7 1l-2.5-1-2 3.5L6.6 11a7.6 7.6 0 0 0 0 2l-2.1 1.6 2 3.5 2.5-1a7.4 7.4 0 0 0 1.7 1L11 21h4l.3-2.9a7.4 7.4 0 0 0 1.7-1l2.5 1 2-3.5L19.4 13zM13 15.5A3.5 3.5 0 1 1 13 8.5a3.5 3.5 0 0 1 0 7z"/></svg>',
    usuario: '<svg class="icono" viewBox="0 0 24 24" aria-hidden="true"><circle cx="9" cy="7" r="4" fill="#222"/><path fill="#222" d="M1 21c0-4.4 3.6-7 8-7s8 2.6 8 7z"/><path fill="none" stroke="#222" stroke-width="2.2" d="M15.5 10.5l2.3 2.3 4.7-4.8"/></svg>',
    servicios: '<svg class="icono" viewBox="0 0 24 24" aria-hidden="true"><g fill="#222"><rect x="2" y="3" width="4" height="3"/><rect x="8" y="3" width="14" height="3"/><rect x="2" y="9" width="4" height="3"/><rect x="8" y="9" width="14" height="3"/><rect x="2" y="15" width="4" height="3"/><rect x="8" y="15" width="14" height="3"/></g></svg>',
    balanceo: '<svg class="icono" viewBox="0 0 24 24" aria-hidden="true"><g fill="#222"><rect x="9" y="1" width="6" height="5" rx="1"/><rect x="1" y="17" width="6" height="5" rx="1"/><rect x="9" y="17" width="6" height="5" rx="1"/><rect x="17" y="17" width="6" height="5" rx="1"/></g><path fill="none" stroke="#222" stroke-width="1.8" d="M12 6v5M4 17v-3h16v3M12 11v6"/></svg>',
  };

  const $ = (id) => document.getElementById(id);
  let claveVM = "general";

  // ---------- Encabezado y pie comunes ----------
  function marco(titulo, icono) {
    const enc = $("encabezado");
    enc.className = "encabezado";
    enc.innerHTML = `<a class="marca" href="index.html" title="Ir al inicio">${ICONOS.nube}
      <div><h1>Consola de automatización de servicios</h1><p>Computación en la nube 2026 - 2</p></div></a>`;
    const pie = $("pie");
    pie.className = "pie";
    pie.textContent = "Juan Diego García Albarracín - Daniel Josué Narváez Hincapié · Universidad del Quindío";
    if (titulo) {
      const t = $("titulo-modulo");
      t.className = "titulo-modulo";
      t.innerHTML = `${ICONOS[icono] || ""}<h2></h2><a class="volver" href="index.html">← Volver al inicio</a>`;
      t.querySelector("h2").textContent = titulo;
      document.title = titulo + " - Consola de automatización";
    }
  }

  // ---------- Almacenamiento local (solo comodidad: recordar la VM y la IP) ----------
  function leer(k) { try { return localStorage.getItem(k); } catch { return null; } }
  function guardar(k, v) { try { localStorage.setItem(k, v); } catch { /* sin almacenamiento */ } }

  // ---------- Barra de la máquina virtual ----------
  function iniciarVM(clave, vmPorDefecto) {
    claveVM = clave;
    const vm = $("vm"), ip = $("ip");
    vm.value = leer(`consola.${clave}.vm`) || vmPorDefecto;
    ip.value = leer(`consola.${clave}.ip`) || "";
    vm.addEventListener("change", () => { guardar(`consola.${clave}.vm`, vm.value.trim()); ip.value = ""; });
    ip.addEventListener("change", () => guardar(`consola.${clave}.ip`, ip.value.trim()));
    $("btn-ip").addEventListener("click", detectarIP);
  }

  async function detectarIP() {
    const vm = $("vm").value.trim();
    if (!marcar($("vm"), RE.vm.test(vm))) {
      mostrar({ exito: false, mensaje: "Escriba el nombre de la VM tal como aparece en VirtualBox (letras, números, punto, guion o guion bajo)." });
      return null;
    }
    ocupado(true, `Consultando la IP de ${vm} con Guest Additions...`);
    const r = await llamar("GET", "/api/vm/ip?vm=" + encodeURIComponent(vm));
    ocupado(false);
    if (r.exito && r.datos && r.datos.ip) {
      $("ip").value = r.datos.ip;
      guardar(`consola.${claveVM}.ip`, r.datos.ip);
      marcar($("ip"), true);
      mostrar(r, `IP de ${vm}: ${r.datos.ip}`);
      return r.datos.ip;
    }
    mostrar(r, r.mensaje || "No se pudo obtener la IP: verifique que la VM esté encendida y tenga Guest Additions.");
    return null;
  }

  // Devuelve {vm, ip} validados, o null (y muestra el error).
  function vmActual() {
    const vm = $("vm").value.trim(), ip = $("ip").value.trim();
    const okVM = marcar($("vm"), RE.vm.test(vm));
    const okIP = marcar($("ip"), RE.ipv4.test(ip));
    if (!okVM) { mostrar({ exito: false, mensaje: "Nombre de VM no válido." }); return null; }
    if (!okIP) { mostrar({ exito: false, mensaje: "Indique la IP de la VM (botón «Detectar IP»)." }); return null; }
    guardar(`consola.${claveVM}.vm`, vm);
    guardar(`consola.${claveVM}.ip`, ip);
    return { vm, ip };
  }

  // Marca visualmente un campo inválido y devuelve el mismo valor booleano.
  function marcar(campo, valido) {
    campo.classList.toggle("invalido", !valido);
    return valido;
  }

  // ---------- Llamadas a la API ----------
  async function llamar(metodo, ruta, cuerpo) {
    const opciones = { method: metodo, headers: {} };
    if (cuerpo !== undefined) {
      opciones.headers["Content-Type"] = "application/json";
      opciones.body = JSON.stringify(cuerpo);
    }
    let resp;
    try {
      resp = await fetch(ruta, opciones);
    } catch (e) {
      return { exito: false, mensaje: "No se pudo contactar la consola (¿está en ejecución?): " + e.message };
    }
    let datos;
    try {
      datos = await resp.json();
    } catch {
      datos = { exito: false, mensaje: `Respuesta inesperada del servidor (HTTP ${resp.status}).` };
    }
    if (!resp.ok) datos.exito = false;
    return datos;
  }

  // ---------- Estado "trabajando" ----------
  function ocupado(si, texto) {
    document.querySelectorAll("main button").forEach((b) => { b.disabled = si; });
    if (si) {
      const caja = $("resultado");
      caja.className = "resultado trabajando";
      caja.textContent = "⏳ " + (texto || "Ejecutando...") + " (puede tardar si se instalan paquetes)";
    }
  }

  // Ejecuta una operación mostrando el estado de trabajo y luego el resultado.
  async function ejecutar(texto, metodo, ruta, cuerpo) {
    ocupado(true, texto);
    const r = await llamar(metodo, ruta, cuerpo);
    ocupado(false);
    mostrar(r);
    return r;
  }

  // ---------- Presentación del resultado ----------
  function mostrar(r, mensajeAlterno, abrirDetalle) {
    const caja = $("resultado");
    const pasos = Array.isArray(r.pasos) ? r.pasos : [];
    const advertencia = r.exito && pasos.some((p) => p.exito && p.errores);
    caja.className = "resultado " + (!r.exito ? "error" : advertencia ? "aviso" : "ok");
    caja.innerHTML = "";

    const msg = document.createElement("div");
    msg.className = "resultado-msg";
    const icono = !r.exito ? "✖ " : advertencia ? "⚠ " : "✔ ";
    msg.textContent = icono + (mensajeAlterno || r.mensaje || (r.exito ? "Operación completada." : "La operación falló."));
    caja.append(msg);
    if (advertencia) {
      const nota = document.createElement("div");
      nota.className = "ayuda";
      nota.textContent = "La operación terminó bien, pero algún paso dejó mensajes en la salida de error (ver detalle).";
      caja.append(nota);
    }

    if (pasos.length) {
      const det = document.createElement("details");
      det.open = !r.exito || !!abrirDetalle;
      const sum = document.createElement("summary");
      sum.textContent = `Detalle técnico (${pasos.length} paso${pasos.length > 1 ? "s" : ""})`;
      det.append(sum);
      pasos.forEach((p, i) => {
        const div = document.createElement("div");
        div.className = "paso " + (p.exito ? (p.errores ? "aviso" : "ok") : "error");
        const cab = document.createElement("div");
        cab.className = "paso-cab";
        cab.textContent = `Paso ${i + 1} · ${p.exito ? "correcto" : "falló"} · ${p.operacion} · ${p.destino} · código ${p.codigo} · ${p.duracion}`;
        div.append(cab);
        bloque(div, "Comando", p.comando);
        bloque(div, "Salida", p.salida);
        bloque(div, p.exito ? "Mensajes (salida de error)" : "Errores", p.errores);
        det.append(div);
      });
      caja.append(det);
    }
    caja.scrollIntoView({ behavior: "smooth", block: "nearest" });
  }

  function bloque(padre, titulo, texto) {
    if (!texto) return;
    const h = document.createElement("h4");
    h.textContent = titulo;
    const pre = document.createElement("pre");
    pre.textContent = texto;
    padre.append(h, pre);
  }

  // ---------- Descarga de llaves privadas con recomendaciones de uso ----------
  async function descargarLlave(vm, usuario) {
    const url = `/api/llaves/descargar?vm=${encodeURIComponent(vm)}&usuario=${encodeURIComponent(usuario)}`;
    let resp;
    try { resp = await fetch(url); } catch (e) {
      mostrar({ exito: false, mensaje: "No se pudo contactar la consola: " + e.message });
      return false;
    }
    if (!resp.ok) {
      const t = (await resp.text()).trim();
      mostrar({ exito: false, mensaje: `No se pudo descargar la llave de ${usuario}: ${t}. Configure primero sus llaves.` });
      return false;
    }
    const blob = await resp.blob();
    const nombre = `${vm}_${usuario}`;
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = nombre;
    document.body.append(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(a.href), 2000);
    recomendaciones(nombre, usuario);
    return true;
  }

  // Ventana con las recomendaciones para el uso correcto de la llave privada.
  function recomendaciones(archivo, usuario) {
    let dlg = $("dlg-llave");
    if (!dlg) {
      dlg = document.createElement("dialog");
      dlg.id = "dlg-llave";
      document.body.append(dlg);
    }
    const ip = ($("ip") && $("ip").value.trim()) || "IP_DE_LA_VM";
    dlg.innerHTML = `
      <div class="dlg-cab">🔑 Llave privada descargada: <code></code></div>
      <div class="dlg-cuerpo">
        <p>La llave privada da acceso a la máquina virtual sin contraseña: trátela como una contraseña.</p>
        <ul>
          <li><b>No la comparta</b> ni la envíe por correo o chat, y <b>nunca la suba a un repositorio</b>.</li>
          <li>Guárdela en una carpeta privada, por ejemplo <code>C:\\Users\\SU_USUARIO\\.ssh\\</code>, y no la deje en Descargas.</li>
          <li>Restrinja sus permisos para que solo usted pueda leerla:<br>
            <code>icacls ARCHIVO /inheritance:r /grant:r "%USERNAME%:R"</code></li>
          <li>Úsela así: <code class="uso"></code></li>
          <li>Si cree que alguien más la obtuvo, genere una llave nueva desde la consola (opción <i>Reemplazar</i>): la anterior deja de servir.</li>
        </ul>
      </div>
      <div class="dlg-pie"><button class="boton azul" type="button">Entendido</button></div>`;
    dlg.querySelector(".dlg-cab code").textContent = archivo;
    dlg.querySelector(".uso").textContent = `ssh -i ${archivo} ${usuario}@${ip}`;
    dlg.querySelector("button").addEventListener("click", () => dlg.close());
    dlg.showModal();
  }

  return { RE, $, icono: (n) => ICONOS[n] || "", marco, iniciarVM, detectarIP, vmActual, marcar, llamar, ejecutar, mostrar, ocupado, descargarLlave };
})();
