// img2logo converts a PNG logo to terminal block-character art (▀▄█).
package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: img2logo <png> [width]")
		os.Exit(1)
	}
	width := 32
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &width)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	b := img.Bounds()
	x0, y0, x1, y1 := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if filled(img.At(x, y)) {
				if x < x0 {
					x0 = x
				}
				if x > x1 {
					x1 = x
				}
				if y < y0 {
					y0 = y
				}
				if y > y1 {
					y1 = y
				}
			}
		}
	}
	crop := image.Rect(x0, y0, x1+1, y1+1)
	cw, ch := crop.Dx(), crop.Dy()

	// Terminal cells are ~2× taller than wide; sample 2 image rows per char row.
	charH := ch * width / cw / 2
	if charH < 6 {
		charH = 6
	}
	pixW, pixH := width, charH*2

	lines := make([]string, 0, charH)
	grid := make([][]rune, charH)
	for row := 0; row < pixH; row += 2 {
		r := row / 2
		grid[r] = make([]rune, pixW)
		for col := 0; col < pixW; col++ {
			sx := x0 + col*cw/pixW
			sx2 := x0 + (col+1)*cw/pixW
			if sx2 <= sx {
				sx2 = sx + 1
			}
			sy := y0 + row*ch/pixH
			sy2 := y0 + (row+1)*ch/pixH
			sy3 := y0 + (row+2)*ch/pixH
			if sy3 > y0+ch {
				sy3 = y0 + ch
			}

			top := sampleFilled(img, sx, sy, sx2, sy2)
			bot := sampleFilled(img, sx, sy2, sx2, sy3)
			switch {
			case top && bot:
				grid[r][col] = '█'
			case top:
				grid[r][col] = '▀'
			case bot:
				grid[r][col] = '▄'
			default:
				grid[r][col] = ' '
			}
		}
	}
	markEye(grid)
	for _, row := range grid {
		lines = append(lines, strings.TrimRight(string(row), " "))
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	fmt.Printf("// crop %dx%d -> %d cols x %d rows\n", cw, ch, width, len(lines))
	for _, l := range lines {
		fmt.Println(l)
	}
}

func filled(c color.Color) bool {
	r, g, b, a := c.RGBA()
	if a < 0x8000 {
		return false
	}
	// Non-white / colored pixel (whale is blue).
	return r < 0xf000 || g < 0xf000 || b < 0xf000
}

func sampleFilled(img image.Image, x0, y0, x1, y1 int) bool {
	n, hit := 0, 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			n++
			if filled(img.At(x, y)) {
				hit++
			}
		}
	}
	if n == 0 {
		return false
	}
	return hit*2 >= n // majority
}

// markEye turns the single-pixel interior hole (orca eye) into ●.
func markEye(grid [][]rune) {
	h := len(grid)
	if h == 0 {
		return
	}
	w := len(grid[0])
	filled := func(r rune) bool { return r != ' ' && r != '●' }

	bestY, bestX := -1, -1
	bestScore := -1
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			if grid[y][x] != ' ' {
				continue
			}
			n := 0
			for _, d := range [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}} {
				if filled(grid[y+d[0]][x+d[1]]) {
					n++
				}
			}
			if n != 4 {
				continue
			}
			// Prefer holes in the lower-right head area (image eye location).
			score := y*10 + x
			if score > bestScore {
				bestScore = score
				bestY, bestX = y, x
			}
		}
	}
	if bestY >= 0 {
		grid[bestY][bestX] = '●'
	}
}
