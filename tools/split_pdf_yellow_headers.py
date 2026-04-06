#!/usr/bin/env python3
"""
[Legado] Para pipeline completo: prepare_expediente_firma_electronica.py

Divide un PDF en archivos por secciones marcadas con rectángulos de relleno amarillo
(RGB ~1,1,0) en la parte superior de la página (cabecera tipo enunciado).

Uso:
  python tools/split_pdf_yellow_headers.py entrada.pdf [carpeta_salida]
"""
from __future__ import annotations

import sys
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


def main() -> int:
    if len(sys.argv) < 2:
        print(__doc__)
        return 2
    src = Path(sys.argv[1]).expanduser().resolve()
    if not src.is_file():
        print("No existe:", src)
        return 1
    out_dir = Path(sys.argv[2]).expanduser().resolve() if len(sys.argv) > 2 else src.parent / (src.stem + "_por_secciones")
    out_dir.mkdir(parents=True, exist_ok=True)

    doc = fitz.open(str(src))
    hits_1based: list[int] = []
    for i in range(doc.page_count):
        if page_has_yellow_header(doc[i]):
            hits_1based.append(i + 1)

    if not hits_1based:
        print("No se encontraron páginas con cabecera amarilla en el tercio superior.")
        doc.close()
        return 1

    manifest: list[str] = []
    n = len(hits_1based)
    for idx in range(n):
        p0 = hits_1based[idx] - 1
        if idx + 1 < n:
            p1 = hits_1based[idx + 1] - 2
        else:
            p1 = doc.page_count - 1
        if p1 < p0:
            p1 = p0
        part = fitz.open()
        part.insert_pdf(doc, from_page=p0, to_page=p1)
        name = f"parte_{idx + 1:03d}_pag{hits_1based[idx]}-{p1 + 1}.pdf"
        out_path = out_dir / name
        part.save(str(out_path))
        part.close()
        manifest.append(f"{name}\tpáginas originales {hits_1based[idx]}–{p1 + 1}\t({p1 - p0 + 1} págs.)")

    man_path = out_dir / "MANIFIESTO.txt"
    man_path.write_text(
        "\n".join(
            [
                f"Origen: {src}",
                f"Secciones (cabecera amarilla): {n}",
                "",
                *manifest,
            ]
        ),
        encoding="utf-8",
    )

    # PDF único con página en blanco entre secciones (solo separador visual)
    combined = fitz.open()
    for idx in range(n):
        p0 = hits_1based[idx] - 1
        if idx + 1 < n:
            p1 = hits_1based[idx + 1] - 2
        else:
            p1 = doc.page_count - 1
        if p1 < p0:
            p1 = p0
        if idx > 0:
            combined.new_page(width=doc[p0].rect.width, height=doc[p0].rect.height)
        combined.insert_pdf(doc, from_page=p0, to_page=p1)
    comb_path = out_dir / f"{src.stem}_con_separadores_en_blanco.pdf"
    combined.save(str(comb_path))
    combined.close()

    doc.close()
    print(f"Listo: {n} archivos en {out_dir}")
    print(f"Manifiesto: {man_path}")
    print(f"Combinado con separadores: {comb_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
