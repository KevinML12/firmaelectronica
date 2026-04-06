package plantillas

import "strings"

// Info alinea tipo_documento_oj con archivos en backend/plantillas/docx/ (build_docx.py).
type Info struct {
	CodigoTipo  string   `json:"codigo"`
	Etiqueta    string   `json:"etiqueta"`
	Plantilla   string   `json:"plantilla_docx,omitempty"`
	Roles       []string `json:"roles_sugeridos,omitempty"`
}

var porTipo = map[string]Info{
	"mtps_adjudicacion_denuncia": {
		Etiqueta:  "MTPS — Adjudicación / denuncia",
		Plantilla: "01_mtps_adjudicacion_denuncia.docx",
		Roles:     []string{"parte_actora", "inspectora_trabajo"},
	},
	"mtps_cedula_citacion": {
		Etiqueta:  "MTPS — Cédula de citación",
		Plantilla: "02_mtps_cedula_citacion.docx",
		Roles:     []string{"inspectora_trabajo"},
	},
	"mtps_acta_comparecencia": {
		Etiqueta:  "MTPS — Acta de comparecencia / audiencia",
		Plantilla: "03_mtps_acta_comparecencia.docx",
		Roles:     []string{"parte_actora", "representante_demandada", "inspectora_trabajo"},
	},
	"memorial_demanda": {
		Etiqueta:  "Memorial — demanda (juzgado)",
		Plantilla: "04_memorial_demanda_juzgado.docx",
		Roles:     []string{"parte_actora", "patrono_abogado"},
	},
	"memorial_pliego": {
		Etiqueta:  "Memorial — pliego de posiciones",
		Plantilla: "05_memorial_pliego_posiciones.docx",
		Roles:     []string{"parte_actora"},
	},
	"auto": {
		Etiqueta:  "Auto (juzgado 1.ª instancia laboral)",
		Plantilla: "06_juzgado_resolucion_o_auto.docx",
		Roles:     []string{"juez", "secretario"},
	},
	"resolucion": {
		Etiqueta:  "Resolución (juzgado 1.ª instancia laboral)",
		Plantilla: "06_juzgado_resolucion_o_auto.docx",
		Roles:     []string{"juez", "secretario"},
	},
	"sala_resolucion_apelacion": {
		Etiqueta:  "Sala — auto / resolución (apelaciones)",
		Plantilla: "07_sala_resolucion_apelacion.docx",
		Roles:     []string{"magistrado", "secretario"},
	},
	"notificacion": {
		Etiqueta:  "Notificación electrónica OJ (casillero)",
		Plantilla: "08_oj_notificacion_casillero.docx",
		Roles:     []string{"notificador"},
	},
	"oficio": {
		Etiqueta:  "Oficio — ministro ejecutor (ejecución)",
		Plantilla: "09_juzgado_oficio_ministro_ejecutor.docx",
		Roles:     []string{"juez", "secretario"},
	},
	"escrito": {
		Etiqueta:  "Escrito de parte (genérico; referencia: memorial demanda)",
		Plantilla: "04_memorial_demanda_juzgado.docx",
		Roles:     []string{"parte_actora", "patrono_abogado"},
	},
	"otro": {
		Etiqueta: "Otro PDF (sin plantilla fija)",
		Roles:    nil,
	},
}

var catalogoOrden = []string{
	"mtps_adjudicacion_denuncia",
	"mtps_cedula_citacion",
	"mtps_acta_comparecencia",
	"memorial_demanda",
	"memorial_pliego",
	"auto",
	"resolucion",
	"sala_resolucion_apelacion",
	"notificacion",
	"oficio",
	"escrito",
	"otro",
}

// PorTipo devuelve metadatos; tipos desconocidos se tratan como otro.
func PorTipo(codigo string) Info {
	c := strings.TrimSpace(strings.ToLower(codigo))
	if c == "" {
		c = "otro"
	}
	if i, ok := porTipo[c]; ok {
		i.CodigoTipo = c
		return i
	}
	return Info{
		CodigoTipo: c,
		Etiqueta:   "Tipo no catalogado (elija rol según el acta)",
		Plantilla:  "",
		Roles:      nil,
	}
}

// ValidTipo indica si el valor es un tipo_documento_oj permitido en API.
func ValidTipo(s string) bool {
	_, ok := porTipo[strings.TrimSpace(strings.ToLower(s))]
	return ok
}

// Catalogo lista ordenada para selectores (subida, documentación).
func Catalogo() []Info {
	out := make([]Info, 0, len(catalogoOrden))
	for _, k := range catalogoOrden {
		i := porTipo[k]
		i.CodigoTipo = k
		out = append(out, i)
	}
	return out
}
