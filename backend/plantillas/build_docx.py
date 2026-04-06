#!/usr/bin/env python3
"""Genera DOCX mínimos (OOXML) con marcadores {{...}}. Sin huecos de firma manuscrita: la firma electrónica y sellos van en el PDF vía tramitador."""
from __future__ import annotations

import zipfile
from pathlib import Path

W = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"


def esc(s: str) -> str:
	return (
		s.replace("&", "&amp;")
		.replace("<", "&lt;")
		.replace(">", "&gt;")
		.replace('"', "&quot;")
	)


def para(text: str) -> str:
	t = esc(text)
	return f'<w:p><w:r><w:t xml:space="preserve">{t}</w:t></w:r></w:p>'


def document_xml(paragraphs: list[str]) -> str:
	body = "".join(para(x) for x in paragraphs)
	return f"""<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="{W}">
<w:body>{body}<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr></w:body>
</w:document>"""


CONTENT_TYPES = """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>"""

RELS = """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>"""

DOC_RELS = """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>
"""


def write_docx(path: Path, paragraphs: list[str]) -> None:
	path.parent.mkdir(parents=True, exist_ok=True)
	with zipfile.ZipFile(path, "w", zipfile.ZIP_DEFLATED) as zf:
		zf.writestr("[Content_Types].xml", CONTENT_TYPES)
		zf.writestr("_rels/.rels", RELS)
		zf.writestr("word/_rels/document.xml.rels", DOC_RELS)
		zf.writestr("word/document.xml", document_xml(paragraphs))


TEMPLATES: dict[str, list[str]] = {
	"01_mtps_adjudicacion_denuncia.docx": [
		"PLANTILLA — Adjudicación / acta inicial de denuncia (Inspección de Trabajo)",
		"Adjudicación No. {{ADJUDICACION_NUMERO}} · Fecha y hora: {{FECHA_HORA}} · Lugar: {{LUGAR_MTPS}}",
		"",
		"Compareciente: {{NOMBRE_COMPARECIENTE}} · DPI: {{DPI}}",
		"Empleador denunciado: {{EMPLEADOR}} · Representante: {{REPRESENTANTE_EMPLEADOR}}",
		"",
		"CUERPO (hechos y peticiones):",
		"{{CUERPO_DENUNCIA}}",
		"",
		"Audiencia fijada: {{FECHA_AUDIENCIA}}",
	],
	"02_mtps_cedula_citacion.docx": [
		"PLANTILLA — Cédula de citación (conciliación MTPS)",
		"Adjudicación No. {{ADJUDICACION_NUMERO}}",
		"Interpuesta por: {{ACTOR}} · Citado: {{DEMANDADO_CITACION}}",
		"",
		"{{TEXTO_CITACION}}",
	],
	"03_mtps_acta_comparecencia.docx": [
		"PLANTILLA — Acta de comparecencia / audiencia / agotamiento vía administrativa",
		"Adjudicación No. {{ADJUDICACION_NUMERO}} · Fecha: {{FECHA}}",
		"",
		"Comparecientes: {{COMPARECIENTES}}",
		"",
		"{{ACTA_DESARROLLO}}",
	],
	"04_memorial_demanda_juzgado.docx": [
		"PLANTILLA — Demanda / memorial (juicio ordinario laboral — juzgado)",
		"Juzgado: {{JUZGADO}} · Expediente: {{EXPEDIENTE_JUDICIAL}}",
		"Actora: {{ACTOR}} · Demandada: {{DEMANDADA}}",
		"",
		"RELACIÓN DE HECHOS:",
		"{{HECHOS}}",
		"",
		"MEDIOS DE PRUEBA Y PETICIONES:",
		"{{PRUEBAS_PETICIONES}}",
		"",
		"Lugar y fecha: {{LUGAR_FECHA}}",
	],
	"05_memorial_pliego_posiciones.docx": [
		"PLANTILLA — Plica / pliego de posiciones (confesional)",
		"Juicio: {{REFERENCIA_JUICIO}}",
		"",
		"{{INTRO_PLIEGO}}",
		"",
		"Preguntas (posiciones):",
		"{{PLIEGO_POSICIONES}}",
	],
	"06_juzgado_resolucion_o_auto.docx": [
		"PLANTILLA — Resolución / auto (juzgado 1.ª instancia laboral)",
		"Juzgado: {{JUZGADO}} · Expediente: {{EXPEDIENTE_JUDICIAL}} · Oficio: {{OFICIO}}",
		"",
		"{{VISTOS_CONSIDERANDOS}}",
		"",
		"RESUELVE / AUTO:",
		"{{FALLO}}",
	],
	"07_sala_resolucion_apelacion.docx": [
		"PLANTILLA — Auto / resolución (Sala de Apelaciones)",
		"Sala: {{SALA}} · Recurso: {{REFERENCIA_RECURSO}}",
		"",
		"{{CONSIDERANDOS_SALA}}",
		"",
		"RESUELVE:",
		"{{FALLO_SALA}}",
	],
	"08_oj_notificacion_casillero.docx": [
		"PLANTILLA — Constancia de notificación electrónica (OJ / casillero)",
		"Notificación No. {{NO_NOTIFICACION}} · Expediente: {{EXPEDIENTE_JUDICIAL}}",
		"Notificado: {{NOTIFICADO}} · Casillero: {{CASILLERO}}",
		"Resolución publicada: {{TITULO_RESOLUCION}} · Fecha publicación: {{FECHA_PUBLICACION}}",
		"",
		"{{TEXTO_ADICIONAL}}",
	],
	"09_juzgado_oficio_ministro_ejecutor.docx": [
		"PLANTILLA — Oficio / despacho a ministro ejecutor (ejecución laboral)",
		"Juzgado: {{JUZGADO}} · Expediente: {{EXPEDIENTE_JUDICIAL}}",
		"",
		"{{CUERPO_OFICIO}}",
		"Monto: {{MONTO}} · Dirección ejecución: {{DIRECCION}}",
		"",
		"(Ministro ejecutor actúa en diligencia externa; firma en acta propia: ministro_ejecutor)",
	],
}


def main() -> None:
	root = Path(__file__).resolve().parent
	out = root / "docx"
	for name, paras in TEMPLATES.items():
		write_docx(out / name, paras)
		print("OK", out / name)


if __name__ == "__main__":
	main()
