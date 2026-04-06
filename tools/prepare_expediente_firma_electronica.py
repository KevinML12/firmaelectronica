#!/usr/bin/env python3
"""
Pipeline único para expediente laboral / MTPS (mezcla de split + limpieza):

1) Detecta inicio de cada sección por cabecera amarilla (como split_pdf_yellow_headers.py)
   — se guardan los índices ANTES de modificar el PDF.
2) En páginas normales:
   - Quita sellos / timbres gráficos al pie (imágenes pequeñas abajo; no toca logos anchos arriba).
   - Quita relleno amarillo y el texto encima (redacción).
   - Quita marcos negros (fs blanco + trazo negro): cabecera ancha arriba entera;
     cajas con mucho texto (credenciales, bloques con nombre) en pie o centro;
     NO borra cajas vacías del pie = espacio reservado para firma electrónica.
   - Líneas negras horizontales finas tipo regla.
3) Páginas de cédula de notificación: solo quita sellos al pie; deja títulos y contenido.
4) Escribe:
   - <stem>_listo_firma.pdf (documento completo limpio)
   - parte_XXX_….pdf (mismos cortes que antes)
   - MANIFIESTO.txt
   - <stem>_con_separadores_en_blanco.pdf

Requiere: pip install pymupdf

Uso:
  python tools/prepare_expediente_firma_electronica.py entrada.pdf [carpeta_salida]
"""
from __future__ import annotations

import sys
import unicodedata
from pathlib import Path

import fitz


def is_yellow_fill(item: dict) -> bool:
    f = item.get("fill")
    if not f or len(f) < 3:
        return False
    r, g, b = f[0], f[1], f[2]
    return r > 0.9 and g > 0.9 and b < 0.2


def page_has_yellow_header(page: fitz.Page, top_frac: float = 0.35) -> bool:
    h = page.rect.height
    y_cut = page.rect.y0 + h * top_frac
    for d in page.get_drawings():
        if not is_yellow_fill(d):
            continue
        r = d.get("rect")
        if r and r.y0 < y_cut:
            return True
    return False


def texto_normalizado(s: str) -> str:
    s = unicodedata.normalize("NFD", s)
    s = "".join(c for c in s if unicodedata.category(c) != "Mn")
    return s.upper()


def es_cedula_notificacion(page: fitz.Page) -> bool:
    t = texto_normalizado(page.get_text())
    if "CEDULA" not in t:
        return False
    return "NOTIFICACION" in t or "CASILLERO" in t or "ELECTRONICA" in t or "ELECTRONICO" in t


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


def pick_redacts(rects: list[fitz.Rect]) -> list[fitz.Rect]:
    rects = sorted(rects, key=lambda r: r.get_area(), reverse=True)
    out: list[fitz.Rect] = []
    for r in rects:
        if r.is_empty or r.get_area() < 2:
            continue
        if any(o.contains(r.tl) and o.contains(r.br) for o in out):
            continue
        out.append(r)
    return out


def infl(r: fitz.Rect, m: float = 2) -> fitz.Rect:
    return fitz.Rect(r.x0 - m, r.y0 - m, r.x1 + m, r.y1 + m)


def palabras_en_rect(page: fitz.Page, rect: fitz.Rect) -> int:
    n = 0
    for w in page.get_text("words"):
        wr = fitz.Rect(w[:4])
        inter = wr & rect
        if not inter.is_empty and inter.get_area() > 0.5:
            n += 1
    return n


def agregar_redacts_sellos(page: fitz.Page, redacts: list[fitz.Rect]) -> None:
    """Imágenes pequeñas abajo (sellos); no logos anchos ni banners."""
    ph = page.rect.height
    pw = page.rect.width
    area_p = pw * ph
    for img in page.get_images(full=True):
        xref = img[0]
        for ir in page.get_image_rects(xref):
            r = fitz.Rect(ir)
            if r.y0 < ph * 0.62:
                continue
            if r.width > pw * 0.48:
                continue
            if r.get_area() > area_p * 0.09:
                continue
            if r.height < 28 and r.width < 28:
                continue
            redacts.append(infl(r, 1))


def preparar_pagina_completa(page: fitz.Page) -> None:
    drawings = page.get_drawings()
    pw, ph = page.rect.width, page.rect.height
    redacts: list[fitz.Rect] = []

    agregar_redacts_sellos(page, redacts)

    for d in drawings:
        if d.get("type") == "f" and is_yellow_fill(d):
            redacts.append(infl(fitz.Rect(d["rect"]), 2))

    black_fs = [d for d in drawings if is_black_stroke_white_fill(d)]
    black_fs.sort(key=lambda d: d["rect"].get_area(), reverse=True)

    for d in black_fs:
        outer = fitz.Rect(d["rect"])
        wide_top = outer.width > 0.38 * pw and outer.y0 < 110
        zona_pie = outer.y0 > 0.50 * ph

        if wide_top:
            redacts.append(infl(outer, 2))
        elif zona_pie:
            if palabras_en_rect(page, outer) > 8:
                redacts.append(infl(outer, 2))
        else:
            if outer.get_area() > 600:
                redacts.append(infl(outer, 2))

    for d in drawings:
        if d.get("type") != "f":
            continue
        f = d.get("fill")
        if not f or len(f) < 3 or f[0] > 0.1 or f[1] > 0.1 or f[2] > 0.1:
            continue
        r = fitz.Rect(d["rect"])
        if r.height < 6 and r.width > 35:
            redacts.append(infl(r, 1))

    to_apply = pick_redacts(redacts)
    for r in to_apply:
        page.add_redact_annot(r, fill=(1, 1, 1))
    if to_apply:
        page.apply_redactions()


def preparar_pagina_cedula_solo_sellos(page: fitz.Page) -> None:
    redacts: list[fitz.Rect] = []
    agregar_redacts_sellos(page, redacts)
    to_apply = pick_redacts(redacts)
    for r in to_apply:
        page.add_redact_annot(r, fill=(1, 1, 1))
    if to_apply:
        page.apply_redactions()


def main() -> int:
    if len(sys.argv) < 2:
        print(__doc__)
        return 2
    src = Path(sys.argv[1]).expanduser().resolve()
    if not src.is_file():
        print("No existe:", src)
        return 1
    out_dir = Path(sys.argv[2]).expanduser().resolve() if len(sys.argv) > 2 else src.parent / (src.stem + "_listo_firma")
    out_dir.mkdir(parents=True, exist_ok=True)

    doc = fitz.open(str(src))

    hits_1based: list[int] = []
    for i in range(doc.page_count):
        if page_has_yellow_header(doc[i]):
            hits_1based.append(i + 1)

    for i in range(doc.page_count):
        p = doc[i]
        if es_cedula_notificacion(p):
            preparar_pagina_cedula_solo_sellos(p)
        else:
            preparar_pagina_completa(p)

    full_path = out_dir / f"{src.stem}_listo_firma.pdf"
    doc.save(str(full_path), garbage=4, deflate=True, clean=True)

    manifest: list[str] = []
    if hits_1based:
        n = len(hits_1based)
        for idx in range(n):
            p0 = hits_1based[idx] - 1
            p1 = hits_1based[idx + 1] - 2 if idx + 1 < n else doc.page_count - 1
            if p1 < p0:
                p1 = p0
            part = fitz.open()
            part.insert_pdf(doc, from_page=p0, to_page=p1)
            name = f"parte_{idx + 1:03d}_pag{hits_1based[idx]}-{p1 + 1}.pdf"
            part.save(str(out_dir / name))
            part.close()
            manifest.append(f"{name}\tpágs. orig. {hits_1based[idx]}–{p1 + 1}\t({p1 - p0 + 1} págs.)")

        comb = fitz.open()
        for idx in range(n):
            p0 = hits_1based[idx] - 1
            p1 = hits_1based[idx + 1] - 2 if idx + 1 < n else doc.page_count - 1
            if p1 < p0:
                p1 = p0
            if idx > 0:
                comb.new_page(width=doc[p0].rect.width, height=doc[p0].rect.height)
            comb.insert_pdf(doc, from_page=p0, to_page=p1)
        comb.save(str(out_dir / f"{src.stem}_con_separadores_en_blanco.pdf"))
        comb.close()

        man_path = out_dir / "MANIFIESTO.txt"
        man_path.write_text(
            "\n".join(
                [
                    f"Origen: {src}",
                    f"Pipeline: prepare_expediente_firma_electronica.py",
                    f"Secciones (cabecera amarilla original): {n}",
                    "",
                    *manifest,
                ]
            ),
            encoding="utf-8",
        )
        lista_lines = [f"{i:02d}. {line.split('\t')[0]}" for i, line in enumerate(manifest, start=1)]
        (out_dir / "LISTA_PARTES.txt").write_text(
            "\n".join(
                [
                    f"Origen (PDF maestro): {src}",
                    f"Total partes numeradas: {n}",
                    "",
                    *lista_lines,
                ]
            ),
            encoding="utf-8",
        )
    else:
        man_path = out_dir / "MANIFIESTO.txt"
        man_path.write_text(
            "\n".join(
                [
                    f"Origen: {src}",
                    "No hubo cabeceras amarillas en el tercio superior: solo PDF completo limpio.",
                ]
            ),
            encoding="utf-8",
        )

    doc.close()
    print("PDF completo:", full_path)
    print("Carpeta:", out_dir)
    if hits_1based:
        print("Partes:", len(hits_1based), "+ MANIFIESTO.txt + LISTA_PARTES.txt + separadores")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
