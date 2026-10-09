package lectura

import (
	"image"
	"image/png"
	"os"
)

func gardarPNG(img image.Image, ruta string) error {
	f, err := os.Create(ruta)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
