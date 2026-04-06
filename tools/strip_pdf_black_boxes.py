#!/usr/bin/env python3
"""
[Legado] Para flujo completo (sin amarillo, sellos, partes, cédulas): usar
prepare_expediente_firma_electronica.py

Quita marcos negros (trazo ~0.75 pt) típicos de plantillas MTPS/OJ en PDF:

- Cabecera ancha con fondo blanco + título en amarillo: solo borra la «caja» negra
  alrededor del amarillo (franjas entre rect exterior e interior), conservando texto
  amarillo y logos (suelen ser imagen fuera de esa franja).
- Cuadros más chicos (credencial, firmas, pies): rellena todo el rectángulo en blanco
  y elimina borde + texto interior.
- Líneas horizontales negras muy finas (relleno negro, altura < 4 pt): las tapa en blanco.

Requiere: pip install pymupdf

Uso:
  python tools/strip_pdf_black_boxes.py entrada.pdf [salida.pdf]
"""
from __future__ import annotations

import sys
from pathlib import Path

import fitz


def is_yellow_fill(d: dict) -> bool:
    f = d.get("fill")
    if not f or len(f) < 3:
        return False
    r, g, b = f[0], f[1], f[2]
    return r > 0.9 and g > 0.9 and b < 0.2


def is_black_stroke_white_fill(d: dict) -> bool:
    if d.get("type") != "fs":
        return False
    f, c = d.get("fill"), d.get("color")
    if not f or len(f) < 3 or not c or len(c) < 3:
        return False
    if f[0] < 0.95 or f[1] < 0.95 or f[2] < 0.95:
        return False
    if c[0] > 0.05 or c[1] > 0.05 or c[2] > 0.05:
        return False
    return True


def yellow_overlap_ratio(outer: fitz.Rect, yr: fitz.Rect) -> float:
    inter = outer & yr
    if inter.is_empty:
        return 0.0
    a = yr.get_area()
    return inter.get_area() / a if a > 0 else 0.0


def union_yellows_inside(outer: fitz.Rect, yellow_rects: list[fitz.Rect]) -> fitz.Rect | None:
    inside = [yr for yr in yellow_rects if yellow_overlap_ratio(outer, yr) > 0.88]
    if not inside:
        return None
    u = inside[0]
    for yr in inside[1:]:
        u |= yr
    return u


def frame_strips(outer: fitz.Rect, inner: fitz.Rect) -> list[fitz.Rect]:
    strips: list[fitz.Rect] = []
    if inner.y0 > outer.y0 + 0.5:
        strips.append(fitz.Rect(outer.x0, outer.y0, outer.x1, inner.y0))
    if outer.y1 > inner.y1 + 0.5:
        strips.append(fitz.Rect(outer.x0, inner.y1, outer.x1, outer.y1))
    if inner.x0 > outer.x0 + 0.5:
        strips.append(fitz.Rect(outer.x0, inner.y0, inner.x0, inner.y1))
    if outer.x1 > inner.x1 + 0.5:
        strips.append(fitz.Rect(inner.x1, inner.y0, outer.x1, inner.y1))
    return strips


def pick_redacts(rects: list[fitz.Rect]) -> list[fitz.Rect]:
    """Evita redactar el mismo sitio dos veces (cuadros anidados): queda el rect mayor."""
    rects = sorted(rects, key=lambda r: r.get_area(), reverse=True)
    out: list[fitz.Rect] = []
    for r in rects:
        if r.is_empty or r.get_area() < 2:
            continue
        if any(o.contains(r.tl) and o.contains(r.br) for o in out):
            continue
        out.append(r)
    return out


def strip_page(page: fitz.Page) -> None:
    pw = page.rect.width
    drawings = page.get_drawings()
    yellow_rects = [d["rect"] for d in drawings if d.get("type") == "f" and is_yellow_fill(d)]
    black_fs = [d for d in drawings if is_black_stroke_white_fill(d)]

    black_fs.sort(key=lambda d: d["rect"].get_area(), reverse=True)

    redacts: list[fitz.Rect] = []
    frame_strips_to_draw: list[fitz.Rect] = []

    for d in black_fs:
        outer = fitz.Rect(d["rect"])
        wide = outer.width > 0.42 * pw
        top_band = outer.y0 < 95
        inner = union_yellows_inside(outer, yellow_rects) if (wide and top_band) else None

        if inner is not None:
            for s in frame_strips(outer, inner):
                if s.is_empty or s.get_area() < 1:
                    continue
                frame_strips_to_draw.append(s)
        else:
            pad = fitz.Rect(outer.x0 - 2, outer.y0 - 2, outer.x1 + 2, outer.y1 + 2)
            redacts.append(pad)

    for d in drawings:
        if d.get("type") != "f":
            continue
        f = d.get("fill")
        if not f or len(f) < 3 or f[0] > 0.1 or f[1] > 0.1 or f[2] > 0.1:
            continue
        r = fitz.Rect(d["rect"])
        if r.height < 5 and r.width > 40:
            redacts.append(fitz.Rect(r.x0 - 1, r.y0 - 1, r.x1 + 1, r.y1 + 1))

    to_redact = pick_redacts(redacts)
    for r in to_redact:
        page.add_redact_annot(r, fill=(1, 1, 1))
    if to_redact:
        page.apply_redactions()

    for s in frame_strips_to_draw:
        page.draw_rect(s, color=(1, 1, 1), fill=(1, 1, 1), width=0, overlay=True)


def main() -> int:
    if len(sys.argv) < 2:
        print(__doc__)
        return 2
    src = Path(sys.argv[1]).expanduser().resolve()
    if not src.is_file():
        print("No existe:", src)
        return 1
    dst = (
        Path(sys.argv[2]).expanduser().resolve()
        if len(sys.argv) > 2
        else src.with_name(src.stem + "_sin_cuadros.pdf")
    )

    doc = fitz.open(str(src))
    for i in range(doc.page_count):
        strip_page(doc[i])
    doc.save(str(dst), garbage=4, deflate=True, clean=True)
    doc.close()
    print("Guardado:", dst)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
