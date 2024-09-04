package plot

import (
	"image"
	"image/color"

	"golang.org/x/image/math/fixed"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

// drawText 使用 basicfont 繪製文字
func drawText(img *image.RGBA, text string, x, y int, col color.Color) {
	// 使用 basicfont.Drawer 繪製文字
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13, // 使用內建的 basicfont 字體
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}
