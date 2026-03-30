// Package pdf: renderizado OJ (gofpdf / unipdf).
//
// Folio correlativo: una HOJA del PDF = un folio (no “página” suelta; es la hoja física
// que se numera en el expediente). Se estampa el número de folio en la esquina superior derecha.
//
// Rúbrica: en CADA HOJA, repetir una rúbrica similar en CADA MARGEN (superior, inferior,
// izquierdo y derecho), como práctica forense/notarial de integridad de la hoja.
//
// Firma electrónica: el bloque completo (“Firma Electrónica Interna”, hash, rol, firma
// institucional, etc.) va solo al FINAL de CADA DOCUMENTO — es decir, en la última hoja
// de ese PDF (documento), no en cada hoja intermedia.
//
// Otros elementos: código de verificación (bloque superior derecho según formato),
// timbres CANG con SHA-256, QR a la URL pública de validación.
package pdf
