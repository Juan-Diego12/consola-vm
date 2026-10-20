// Página Gestión de usuarios: crear, eliminar, listar y descargar llaves.
"use strict";
(() => {
  const { $, RE, marcar } = Consola;
  Consola.marco("Gestión de Usuarios", "usuario");
  Consola.iniciarVM("base", "NP2-Base");
  const PROTEGIDOS = ["root", "nobody"];

  function pintar(lista) {
    const tabla = $("u-tabla"), sel = $("u-lista");
    tabla.innerHTML = "";
    sel.innerHTML = "";
    if (!lista.length) {
      tabla.innerHTML = '<tr><td colspan="6" class="vacio">La VM no tiene usuarios normales (UID ≥ 1000).</td></tr>';
      sel.innerHTML = '<option value="">(sin usuarios)</option>';
      return;
    }
    const etiqueta = (si) => `<span class="etiqueta ${si ? "si" : "no"}">${si ? "sí" : "no"}</span>`;
    lista.forEach((u) => {
      const fila = document.createElement("tr");
      ["nombre", "uid", "home", "shell"].forEach((k) => {
        const td = document.createElement("td");
        td.textContent = u[k];
        fila.append(td);
      });
      fila.insertAdjacentHTML("beforeend", `<td>${etiqueta(u.sudo)}</td><td>${etiqueta(u.llave)}</td>`);
      tabla.append(fila);
      const op = document.createElement("option");
      op.value = u.nombre;
      op.textContent = u.nombre + (u.sudo ? " (sudo)" : "");
      sel.append(op);
    });
  }

  async function listar(silencioso) {
    const c = Consola.vmActual();
    if (!c) return;
    if (!silencioso) Consola.ocupado(true, `Consultando los usuarios de ${c.vm}...`);
    const r = await Consola.llamar("GET", `/api/usuarios?vm=${encodeURIComponent(c.vm)}&ip=${encodeURIComponent(c.ip)}`);
    if (!silencioso) Consola.ocupado(false);
    if (r.exito && r.datos) pintar(r.datos.usuarios || []);
    if (!silencioso) Consola.mostrar(r, r.exito ? `${(r.datos.usuarios || []).length} usuario(s) en ${c.vm}` : null);
  }
  $("btn-listar").addEventListener("click", () => listar(false));

  $("btn-crear").addEventListener("click", async () => {
    const c = Consola.vmActual();
    if (!c) return;
    const nombre = $("u-nombre").value.trim();
    const password = $("u-pw").value;
    const sudo = $("u-sudo").checked, llaves = $("u-llaves").checked;
    let error = "";
    if (!RE.usuario.test(nombre) || PROTEGIDOS.includes(nombre)) error = "Nombre de usuario no válido: minúsculas, números, guion o guion bajo, empezando por letra (root y nobody están protegidos).";
    else if (sudo && !password) error = "Un usuario con sudo necesita contraseña: sudo la pide al usarlo.";
    else if (!llaves && !password) error = "Indique una contraseña o marque «Generar llave SSH»; si no, el usuario no podría iniciar sesión.";
    else if (/[:\r\n]/.test(password)) error = "La contraseña no puede contener «:» ni saltos de línea.";
    marcar($("u-nombre"), !error.startsWith("Nombre"));
    if (error) { Consola.mostrar({ exito: false, mensaje: error }); return; }

    const r = await Consola.ejecutar(`Creando el usuario ${nombre} en ${c.vm}...`, "POST", "/api/usuarios/crear",
      { ...c, nombre, password, sudo, llaves });
    $("u-pw").value = "";
    if (r.exito) {
      Consola.mostrar(r, r.mensaje + (llaves ? ". Descargue su llave privada con el botón «Descargar llave privada»." : "."));
      listar(true);
    }
  });

  $("btn-u-descargar").addEventListener("click", () => {
    const vm = $("vm").value.trim(), nombre = $("u-nombre").value.trim() || $("u-lista").value;
    if (!marcar($("vm"), RE.vm.test(vm)) || !RE.usuario.test(nombre)) {
      Consola.mostrar({ exito: false, mensaje: "Escriba (o seleccione) el usuario cuya llave desea descargar." });
      return;
    }
    Consola.descargarLlave(vm, nombre);
  });

  $("btn-eliminar-u").addEventListener("click", async () => {
    const c = Consola.vmActual();
    if (!c) return;
    const nombre = $("u-lista").value;
    if (!nombre) { Consola.mostrar({ exito: false, mensaje: "Cargue la lista y seleccione un usuario." }); return; }
    const conArchivos = $("u-archivos").checked;
    const detalle = conArchivos ? " junto con su carpeta personal y sus archivos" : " (se conservan sus archivos)";
    if (!confirm(`¿Eliminar el usuario «${nombre}» de ${c.vm}${detalle}?`)) return;
    const r = await Consola.ejecutar(`Eliminando el usuario ${nombre}...`, "POST", "/api/usuarios/eliminar", { ...c, nombre, conArchivos });
    if (r.exito) listar(true);
  });
})();
