"use strict";

// Archivo: comun.js
// Qué hace: comportamientos pequeños que comparten todas las páginas. Las
// páginas los piden con atributos HTML (data-...), sin escribir JavaScript propio:
//   - data-imprimir        → abre el diálogo de impresión al cargar.
//   - data-imprimir-ahora  → botón que imprime la página.
//   - data-confirmar       → pide confirmación antes de enviar un formulario.

// <body data-imprimir>: abre el diálogo de impresión al cargar (el ticket).
if (document.body.hasAttribute("data-imprimir")) {
  // window.print() abre el diálogo de impresión del navegador.
  window.print();
}

// <button data-imprimir-ahora>: imprime la página.
// Busca todos los botones con ese atributo y, a cada uno, le agrega un clic que imprime.
document.querySelectorAll("[data-imprimir-ahora]").forEach((boton) => {
  boton.addEventListener("click", () => window.print());
});

// <form data-confirmar="¿Seguro?">: pide confirmación antes de enviar.
document.querySelectorAll("form[data-confirmar]").forEach((formulario) => {
  // Cuando se intenta enviar el formulario...
  formulario.addEventListener("submit", (evento) => {
    // ...muestra la pregunta; si la persona dice "Cancelar", frena el envío.
    if (!confirm(formulario.dataset.confirmar)) {
      evento.preventDefault();
    }
  });
});
