package utils

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

// WriteNotificationIcon 在指定路径生成一个剪切板同步主题的 PNG 图标，
// 用于 toast 通知左侧的 appLogoOverride 图标（128x128）。
func WriteNotificationIcon(path string) error {
	const size = 128
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	blue := color.RGBA{0x2f, 0x6f, 0xed, 0xff}  // 背景
	white := color.RGBA{0xff, 0xff, 0xff, 0xff} // 剪切板
	green := color.RGBA{0x1b, 0xb5, 0x4f, 0xff} // 同步箭头

	fillRoundedRect(img, 0, 0, size, size, 26, blue)

	// 剪切板主体（白色圆角矩形）
	fillRoundedRect(img, 42, 34, 86, 100, 10, white)
	// 顶部夹子（用背景色盖出缺口）
	draw.Draw(img, image.Rect(54, 22, 74, 38), &image.Uniform{C: blue}, image.Point{}, draw.Src)

	// 剪切板中央：上下两个绿色同步箭头（三角）
	fillTriangle(img, [3]image.Point{{58, 66}, {70, 54}, {70, 66}}, green)
	fillTriangle(img, [3]image.Point{{70, 58}, {70, 70}, {82, 58}}, green)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// fillRoundedRect 用 c 填充左上角 (x0,y0) 到右下角 (x1,y1)、圆角半径 r 的矩形
func fillRoundedRect(img *image.RGBA, x0, y0, x1, y1, r int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if inRoundedRect(x, y, x0, y0, x1, y1, r) {
				img.Set(x, y, c)
			}
		}
	}
}

// inRoundedRect 判断点 (x,y) 是否位于圆角矩形内部
func inRoundedRect(x, y, x0, y0, x1, y1, r int) bool {
	if x < x0+r && y < y0+r {
		return (x-(x0+r))*(x-(x0+r))+(y-(y0+r))*(y-(y0+r)) <= r*r
	}
	if x >= x1-r && y < y0+r {
		return (x-(x1-r))*(x-(x1-r))+(y-(y0+r))*(y-(y0+r)) <= r*r
	}
	if x < x0+r && y >= y1-r {
		return (x-(x0+r))*(x-(x0+r))+(y-(y1-r))*(y-(y1-r)) <= r*r
	}
	if x >= x1-r && y >= y1-r {
		return (x-(x1-r))*(x-(x1-r))+(y-(y1-r))*(y-(y1-r)) <= r*r
	}
	return true
}

// fillTriangle 用 c 填充三个顶点围成的三角形
func fillTriangle(img *image.RGBA, pts [3]image.Point, c color.RGBA) {
	minX, maxX := pts[0].X, pts[0].X
	minY, maxY := pts[0].Y, pts[0].Y
	for _, p := range pts {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if pointInTriangle(image.Pt(x, y), pts) {
				img.Set(x, y, c)
			}
		}
	}
}

// pointInTriangle 判断点 p 是否在三角形 pts 内（含边界）
func pointInTriangle(p image.Point, pts [3]image.Point) bool {
	d1 := sign(p, pts[0], pts[1])
	d2 := sign(p, pts[1], pts[2])
	d3 := sign(p, pts[2], pts[0])
	hasNeg := (d1 < 0) || (d2 < 0) || (d3 < 0)
	hasPos := (d1 > 0) || (d2 > 0) || (d3 > 0)
	return !(hasNeg && hasPos)
}

func sign(p1, p2, p3 image.Point) int {
	return (p1.X-p3.X)*(p2.Y-p3.Y) - (p2.X-p3.X)*(p1.Y-p3.Y)
}
