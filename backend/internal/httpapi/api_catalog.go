package httpapi

import (
	"net/http"

	"github.com/firmaelectronica/expedientes-oj/internal/plantillas"
)

func (d *RouterDeps) listTiposDocumento(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, plantillas.Catalogo())
}
