"use strict";

// Las páginas piden estos comportamientos con atributos, sin escribir JavaScript propio.

// <body data-imprimir>: abre el diálogo de impresión al cargar (el ticket).
if (document.body.hasAttribute("data-imprimir")) {
  window.print();
}

// <button data-imprimir-ahora>: imprime la página.
document.querySelectorAll("[data-imprimir-ahora]").forEach((boton) => {
  boton.addEventListener("click", () => window.print());
});

// <form data-confirmar="¿Seguro?">: pide confirmación antes de enviar.
document.querySelectorAll("form[data-confirmar]").forEach((formulario) => {
  formulario.addEventListener("submit", (evento) => {
    if (!confirm(formulario.dataset.confirmar)) {
      evento.preventDefault();
    }
  });
});
