// Troglosprite: An addition to troglodyte. Capable of converting a png image to pixel data for troglodyte.

package troglosprite

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"

	"github.com/JackDL2058/troglodyte"
)

var buildNumber = "0.0.1"

// Returns the version of troglosprite.
func Version() string {
	return buildNumber
}

// FastImage stores the raw RGBA bytes and dimensions
type FastImage struct {
	Pixels []uint8
	Width  int
	Height int
	Stride int // Stride is the number of bytes per horizontal row
}

// OpenPNG loads a PNG file and converts it into our FastImage struct
func OpenPNG(path string) (*FastImage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	// Type assert to *image.RGBA for direct access to the pixel slice
	// If it's not RGBA (e.g. Grayscale), we draw it onto an RGBA canvas first
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	
	rgbaImg := image.NewRGBA(bounds)
	// This ensures even indexed or grayscale PNGs become standard RGBA
	// image.Draw is the standard way to handle this conversion

	draw.Draw(rgbaImg, bounds, img, bounds.Min, draw.Src)

	return &FastImage{
		Pixels: rgbaImg.Pix,
		Width:  w,
		Height: h,
		Stride: rgbaImg.Stride,
	}, nil
}

// GetRGBA returns the R, G, B, A values at a specific x, y coordinate
func (fi *FastImage) GetRGBA(x, y int) (r, g, b, a uint8) {
	// Safety check: ensure coordinates are within bounds
	if x < 0 || x >= fi.Width || y < 0 || y >= fi.Height {
		return 0, 0, 0, 0
	}

	// Calculate the starting index in the 1D slice
	// Index = (Y * Stride) + (X * 4 bytes per pixel)
	i := (y * fi.Stride) + (x * 4)

	return fi.Pixels[i], fi.Pixels[i+1], fi.Pixels[i+2], fi.Pixels[i+3]
}

// Converts a png into 2D pixel data usable in troglodyte as a sprite. FixRatio attempts to fix the ratio for large sprites, by
// printing 2 characters for every pixel, stretching horizontally, so that a 16 by 16 sprite doesn't look like a rectangle.
// AutoCalculateBrightness, if true, will calculate how bright the pixel should be based on the average of its rgba components.
// If false, the pixels will all be maximum brightness. Disclaimer: This function only outputs block characters of varying
// brightness, like this: █▓▒░, and this will NOT output other symbols. If you want other symbols, make the sprite yourself.
// Other functions, maybe even an editor, will be coming to troglosprite in future!
func PngToPixel(path string, fixRatio bool, autoCalculateBrightness bool) [][]troglodyte.Pixel {
    img, err := OpenPNG(path)
    if err != nil {
        fmt.Println("Error:", err)
        return [][]troglodyte.Pixel{}
    }

    // result slice will always have img.Height rows
    result := make([][]troglodyte.Pixel, 0, img.Height)

    for y := 0; y < img.Height; y++ {
        // If fixRatio is true, the row width is doubled
        rowWidth := img.Width
        if fixRatio {
            rowWidth = img.Width * 2
        }
        
        row := make([]troglodyte.Pixel, 0, rowWidth)
        
        for x := 0; x < img.Width; x++ {
            r, g, b, a := img.GetRGBA(x, y)
            tp := CalculateTrogPixel(r, g, b, a, autoCalculateBrightness)

            if fixRatio {
                // Append the same pixel twice to stretch it horizontally
                row = append(row, tp, tp)
            } else {
                row = append(row, tp)
            }
        }
        result = append(result, row)
    }

    return result
}

// Internal function, but can technically be used outside of troglosprite code.
func CalculateTrogPixel(r, g, b, a uint8, autoCalculateBrightness bool) troglodyte.Pixel {
	// Initialize with defaults
	result := troglodyte.Pixel{
		Char:     " ",
		FgColour: troglodyte.Default,
		BgColour: troglodyte.BgDefault,
	}

	// Transparency check first
	if a < 128 { // Use a threshold (usually 50% transparency)
		return result
	}

	// Convert to int to avoid uint8 overflow during addition
	brightness := (int(r) + int(g) + int(b)) / 3
	bPercent := (float64(brightness) / 255.0) * 100

	// Set Character based on brightness, if autoCalculateBrightness is true
	result.Char = "█" // Default 100% brightness
	if autoCalculateBrightness{
		switch {
		case bPercent > 80: result.Char = "█"
		case bPercent > 60: result.Char = "▓"
		case bPercent > 40: result.Char = "▒"
		case bPercent > 20: result.Char = "░"
		default:            result.Char = " "
		}
	}

	// Set Color using AND logic
	switch {
	case r == 19 && g == 19 && b == 19:
		result.FgColour = troglodyte.Black
	case r == 196 && g == 36 && b == 48:
		result.FgColour = troglodyte.Red
	case r == 51 && g == 152 && b == 75:
		result.FgColour = troglodyte.Green
	case r == 255 && g == 235 && b == 87:
		result.FgColour = troglodyte.Yellow
	case r == 48 && g == 3 && b == 217:
		result.FgColour = troglodyte.Blue
	case r == 219 && g == 63 && b == 253:
		result.FgColour = troglodyte.Magenta
	case r == 15 && g == 155 && b == 155:
		result.FgColour = troglodyte.Cyan
	case r == 255 && g == 255 && b == 255:
		result.FgColour = troglodyte.White
	}

	return result
}