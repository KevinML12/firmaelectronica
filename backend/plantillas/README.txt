Plantillas DOCX (laboral + OJ)
==============================

Modelo de producto (intención)
------------------------------
- Las plantillas DOCX son guía: podés elaborar el contenido en Word con libertad y subir el PDF acá.
- Excepción: las NOTIFICACIONES electrónicas tipo OJ (constancia casillero, etc.) — generadas COMPLETAS por el sistema a partir de datos (plantilla 08 + motor futuro), sin subir el cuerpo a mano.
- Flujo actual para PDF subido: Subir → Procesar (folios, código, QR a validación en Vercel) → Firmar por rol (hash en backend) → Cerrar.

- Carpeta docx/: archivos generados con build_docx.py (volver a ejecutar si cambia el script).
- Marcadores {{VARIABLE}} para reemplazo manual o futuro motor de plantillas.
- No se reservan líneas de firma manuscrita en el DOCX: la firma electrónica y sellos se estampan en el PDF al procesar/firmar desde el tramitador (última hoja). Los roles sugeridos siguen en el API y en internal/plantillas/tipo.go.

Archivos:
  01_mtps_*        Ministerio de Trabajo (adjudicación, cédula, acta).
  04–05            Memoriales de parte (demanda, pliego).
  06, 09           Juzgado 1ª instancia (auto/resolución, oficio ministro ejecutor).
  07               Sala de apelaciones.
  08               Notificación electrónica OJ.

tipo_documento_oj (BD / subida) ↔ plantilla (código en internal/plantillas/tipo.go):
  mtps_adjudicacion_denuncia → 01_mtps_adjudicacion_denuncia.docx
  mtps_cedula_citacion       → 02_mtps_cedula_citacion.docx
  mtps_acta_comparecencia    → 03_mtps_acta_comparecencia.docx
  memorial_demanda           → 04_memorial_demanda_juzgado.docx
  memorial_pliego            → 05_memorial_pliego_posiciones.docx
  auto, resolucion           → 06_juzgado_resolucion_o_auto.docx
  sala_resolucion_apelacion  → 07_sala_resolucion_apelacion.docx
  notificacion               → 08_oj_notificacion_casillero.docx
  oficio                     → 09_juzgado_oficio_ministro_ejecutor.docx
  escrito                    → 04 (referencia genérica de escrito de parte)
  otro                       → (sin plantilla fija)

Roles de firma en API (POST .../firmar, campo "rol"):
  juez, secretario, oficial_v, notificador, magistrado, ministro_ejecutor,
  parte_actora, patrono_abogado, representante_demandada, inspectora_trabajo
