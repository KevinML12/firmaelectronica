package pdfpages

import (
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// Count devuelve el número de hojas del PDF en disco.
func Count(path string) (int, error) {
	n, err := api.PageCountFile(path)
	if err != nil {
		return 0, fmt.Errorf("pdf: %w", err)
	}
	if n < 1 {
		return 0, fmt.Errorf("pdf: sin hojas")
	}
	return n, nil
}
