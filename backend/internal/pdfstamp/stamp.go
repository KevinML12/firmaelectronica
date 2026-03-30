package pdfstamp

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	qrcode "github.com/skip2/go-qrcode"
)

// PageFolio describe el folio impreso en cada hoja (orden 1..N igual al PDF).
type PageFolio struct {
	FolioNumero int64
}

// Apply capas OJ: folio y código arriba derecha, QR validación, rúbricas en márgenes.
func Apply(srcPath, dstPath string, pages []PageFolio, codigoVerif, validateURL string) error {
	n, err := api.PageCountFile(srcPath)
	if err != nil {
		return fmt.Errorf("pdf páginas: %w", err)
	}
	if n != len(pages) {
		return fmt.Errorf("el PDF tiene %d hojas pero hay %d folios en el expediente", n, len(pages))
	}

	qrFile, err := writeQRTemp(validateURL)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(qrFile) }()

	conf := model.NewDefaultConfiguration()
	m := make(map[int][]*model.Watermark, n)

	for i := 0; i < n; i++ {
		p := i + 1
		folio := pages[i].FolioNumero

		wmFolio, err := api.TextWatermark(
			fmt.Sprintf("Folio %d", folio),
			"font:Helvetica, points:11, scale:1 abs, pos:tr, off:-16 -12, fillcol:#0c1e3d, rot:0",
			true, false, types.POINTS,
		)
		if err != nil {
			return err
		}
		wmCod, err := api.TextWatermark(
			fmt.Sprintf("Cód. verif. %s", codigoVerif),
			"font:Helvetica, points:8, scale:1 abs, pos:tr, off:-16 -30, fillcol:#1e293b, rot:0",
			true, false, types.POINTS,
		)
		if err != nil {
			return err
		}
		wmQR, err := api.ImageWatermark(
			qrFile,
			"pos:tl, off:10 -10, scale:.11 abs, rot:0",
			true, false, types.POINTS,
		)
		if err != nil {
			return err
		}

		rubDesc := "font:Helvetica, points:7, scale:1 abs, fillcol:#64748b, rot:0"
		rTL, _ := api.TextWatermark("OJ·", rubDesc+", pos:tl, off:8 -8", true, false, types.POINTS)
		rTR, _ := api.TextWatermark("OJ·", rubDesc+", pos:tr, off:-8 -8", true, false, types.POINTS)
		rBL, _ := api.TextWatermark("OJ·", rubDesc+", pos:bl, off:8 8", true, false, types.POINTS)
		rBR, _ := api.TextWatermark("OJ·", rubDesc+", pos:br, off:-8 8", true, false, types.POINTS)

		rL, _ := api.TextWatermark("RÚBRICA", "font:Helvetica, points:6, scale:1 abs, pos:l, off:10 0, fillcol:#94a3b8, rot:90", true, false, types.POINTS)
		rR, _ := api.TextWatermark("RÚBRICA", "font:Helvetica, points:6, scale:1 abs, pos:r, off:-10 0, fillcol:#94a3b8, rot:90", true, false, types.POINTS)

		m[p] = []*model.Watermark{wmFolio, wmCod, wmQR, rTL, rTR, rBL, rBR, rL, rR}
	}

	if err := api.AddWatermarksSliceMapFile(srcPath, dstPath, m, conf); err != nil {
		return fmt.Errorf("estampar: %w", err)
	}
	return nil
}

func writeQRTemp(url string) (string, error) {
	f, err := os.CreateTemp("", "ojqr-*.png")
	if err != nil {
		return "", err
	}
	name := f.Name()
	_ = f.Close()
	if err := qrcode.WriteFile(url, qrcode.Medium, 180, name); err != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("qr: %w", err)
	}
	return filepath.Clean(name), nil
}

// FirmaElectrónicaBloque añade al final de la última hoja el bloque tipo notificación OJ.
func FirmaElectrónicaBloque(srcPath, dstPath string, nombre, hashInterno, rolEtiqueta string, lastPage int) error {
	text := fmt.Sprintf(
		"Firma Electrónica Interna:\n%s\n%s\n%s\n\nFirma Electrónica Institucional ORGANISMO JUDICIAL:\n[registro en sistema — validar con QR]",
		nombre, hashInterno, rolEtiqueta,
	)
	wm, err := api.TextWatermark(
		text,
		"font:Helvetica, points:9, scale:1 abs, pos:bc, off:0 52, fillcol:#0f172a, rot:0",
		true, false, types.POINTS,
	)
	if err != nil {
		return err
	}
	sel := []string{fmt.Sprintf("%d", lastPage)}
	return api.AddWatermarksFile(srcPath, dstPath, sel, wm, nil)
}
