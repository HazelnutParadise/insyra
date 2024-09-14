package plot

import (
	// 新增 embed 套件
	_ "embed"
	"fmt"
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts/Noto_Sans_TC/NotoSansTC-VariableFont_wght.ttf
var NotoSansTC []byte

// drawText 在圖片上繪製文字
func drawText(img *image.RGBA, text string, x, y int, col color.Color, fontSize int) {
	// 解析字體
	f, err := opentype.Parse(NotoSansTC)
	if err != nil {
		fmt.Println("解析字體失敗:", err)
		return
	}

	// 設定字體大小和 DPI
	const dpi = 72
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    float64(fontSize),
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		fmt.Println("建立字體面失敗:", err)
		return
	}
	defer face.Close()

	// 設定字體繪製器
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
		Dot: fixed.Point26_6{
			X: fixed.I(x),
			Y: fixed.I(y),
		},
	}

	// 繪製文字
	d.DrawString(text)
}
