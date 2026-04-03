package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

func generateIcon(c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 22, 22))
	for y := 0; y < 22; y++ {
		for x := 0; x < 22; x++ {
			dx, dy := float64(x)-10.5, float64(y)-10.5
			if dx*dx+dy*dy <= 100 {
				img.SetRGBA(x, y, c)
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

var (
	iconGray  = generateIcon(color.RGBA{R: 156, G: 163, B: 175, A: 255})
	iconBlue  = generateIcon(color.RGBA{R: 59, G: 130, B: 246, A: 255})
	iconGreen = generateIcon(color.RGBA{R: 34, G: 197, B: 94, A: 255})
	iconRed   = generateIcon(color.RGBA{R: 239, G: 68, B: 68, A: 255})
)
