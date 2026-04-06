Plantillas DOCX (laboral + OJ)
==============================

Modelo de producto (intención)
------------------------------
- Las plantillas DOCX son guía: podés elaborar el contenido en Word con libertad y subir el PDF acá.
- Excepción: las NOTIFICACIONES electrónicas tipo OJ (constancia casillero, etc.) — generadas COMPLETAS por el sistema a partir de datos (plantilla 08 + motor futuro), sin subir el cuerpo a mano.
- Flujo actual para PDF subido: Subir → Procesar (folios, código, QR a validación en Vercel) → Firmar por rol (hash en backend) → Cerrar.

- Carpeta docx/: archivos generados con build_docx.py (volver a ejecutar si cambia el script).
- Marcadores {{VARIABLE}} para reemplazo manual o futuro motor de plantillas.
- Cada bloque "F. {{...}}" indica el rol del sistema (tramitador → firma electrónica en PDF) que debe usarse al estampar la última hoja.

Archivos:
  01_mtps_*        Ministerio de Trabajo (adjudicación, cédula, acta).
  04–05            Memoriales de parte (demanda, pliego).
  06, 09           Juzgado 1ª instancia (auto/resolución, oficio ministro ejecutor).
  07               Sala de apelaciones.
  08               Notificación electrónica OJ.

Roles de firma en API (POST .../firmar, campo "rol"):
  juez, secretario, oficial_v, notificador, magistrado, ministro_ejecutor,
  parte_actora, patrono_abogado, representante_demandada, inspectora_trabajo
