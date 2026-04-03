// Copyright (c) 2026 Jack Durnin. All rights reserved.
// Use of this source code is governed by an MIT-style license.

package troglodyte

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

var (
	buildNumber = "0.0.13" // Build version
	GlobalCircleRatio = 2.0 // the ratio for fixed ratio circles to use, which the x is multiplied by to result in a wider circle, to conter tall terminal characters.
	out         = bufio.NewWriterSize(os.Stdout, 1024*128)

	// Buffers for differential rendering
	backBuffer   [][]Pixel
	frontBuffer  [][]Pixel
	termW, termH int

	// Global Registry
	allSprites []*Sprite
	spriteMu   sync.RWMutex
)

// #region Constants & ANSI
const (
	Black      = "\033[30m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Magenta    = "\033[35m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	Default    = "\033[39m"
	BgBlack    = "\033[40m"
	BgRed      = "\033[41m"
	BgGreen    = "\033[42m"
	BgYellow   = "\033[43m"
	BgBlue     = "\033[44m"
	BgMagenta  = "\033[45m"
	BgCyan     = "\033[46m"
	BgWhite    = "\033[47m"
	BgDefault  = "\033[49m"
	CursorHome = "\033[H"
	HideCursor = "\033[?25l"
	ShowCursor = "\033[?25h"
)

// #endregion

// #region Core Types

type Pixel struct {
	Char     string
	FgColour string
	BgColour string
}

type Sprite struct {
	X, Y     float64 // Current position (float for smooth delta-time)
	Width    int     // Calculated from pixel data
	Height   int     // Calculated from pixel data
	Pixels   [][]Pixel
	Tags     []string
	Visible  bool
	Parent   *Sprite
	Children []*Sprite
	mu       sync.RWMutex // This PROTECTS the X and Y values
}

// #endregion

// #region Buffer Management

// initBuffers initializes the grids based on current terminal size.
func initBuffers() {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w, h = 80, 24 // Fallback
	}
	termW, termH = w, h

	backBuffer = make([][]Pixel, termH)
	frontBuffer = make([][]Pixel, termH)
	for i := range backBuffer {
		backBuffer[i] = make([]Pixel, termW)
		frontBuffer[i] = make([]Pixel, termW)
		for j := 0; j < termW; j++ {
			empty := Pixel{Char: " ", FgColour: Default, BgColour: BgDefault}
			backBuffer[i][j] = empty
			frontBuffer[i][j] = empty
		}
	}
}

// SetPixel is the internal engine function to write to the backbuffer.
func SetPixel(x, y int, p Pixel) {
	// Terminal coordinates are 1-based for users, but 0-based for our buffer logic.
	// We adjust here so the user can think in 1-based terminal coords.
	tx, ty := x-1, y-1
	if ty >= 0 && ty < termH && tx >= 0 && tx < termW {
		backBuffer[ty][tx] = p
	}
}

// #endregion

// #region Sprite Logic

func NewSprite(x, y int, pixels [][]Pixel) *Sprite {
	h := len(pixels)
	w := 0
	if h > 0 {
		w = len(pixels[0])
	}

	s := &Sprite{
		X: float64(x), Y: float64(y), Width: w, Height: h,
		Pixels: pixels, Visible: true, Tags: []string{},
	}

	spriteMu.Lock()
	allSprites = append(allSprites, s)
	spriteMu.Unlock()
	return s
}

func (s *Sprite) AddChild(child *Sprite) {
	s.mu.Lock()
	defer s.mu.Unlock()
	child.Parent = s
	s.Children = append(s.Children, child)
}

var lastFrameTime = time.Now()

func GetDeltaTime() float64 {
	now := time.Now()
	dt := now.Sub(lastFrameTime).Seconds()
	lastFrameTime = now
	return dt
}

func (s *Sprite) Move(dx, dy float64, moveChildren bool) {
	s.mu.Lock()
	s.X += dx
	s.Y += dy
	s.mu.Unlock() // CRITICAL: Unlock the parent BEFORE moving children

	if moveChildren {
		for _, child := range s.Children {
			// This call will create its own independent lock
			child.Move(dx, dy, true)
		}
	}
}

// Removes a sprite from the scene.
func (s *Sprite) Destroy() {
	spriteMu.Lock()
	defer spriteMu.RUnlock() // Note: Use Lock/Unlock for writing to the slice

	for i, sprite := range allSprites {
		if sprite == s {
			// Remove from the global registry
			allSprites = append(allSprites[:i], allSprites[i+1:]...)
			break
		}
	}

	// If it has a parent, remove it from the parent's children list too
	if s.Parent != nil {
		s.Parent.mu.Lock()
		for i, child := range s.Parent.Children {
			if child == s {
				s.Parent.Children = append(s.Parent.Children[:i], s.Parent.Children[i+1:]...)
				break
			}
		}
		s.Parent.mu.Unlock()
	}
}

func (s *Sprite) AddTag(tag string) { s.Tags = append(s.Tags, tag) }

func (s *Sprite) HasTag(tag string) bool {
	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (s *Sprite) Draw() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.Visible || s.Pixels == nil {
		return
	}

	// Crucial: Cast to int AFTER the math to prevent jitter
	// We use floor math to ensure the sprite stays on the integer grid correctly
	offsetX := int(math.Round(s.X)) - (s.Width / 2)
	offsetY := int(math.Round(s.Y)) - (s.Height / 2)

	for y, row := range s.Pixels {
		for x, px := range row {
			// SetPixel handles the bounds checking internally
			SetPixel(offsetX+x, offsetY+y, px)
		}
	}
}

func DrawAllSprites() {
	spriteMu.RLock()
	// Create a local copy of pointers so we don't hold the Registry lock for long
	tempSprites := make([]*Sprite, len(allSprites))
	copy(tempSprites, allSprites)
	spriteMu.RUnlock()

	for _, s := range tempSprites {
		s.Draw()
	}
}

func DrawSpritesWithTag(tag string) {
	spriteMu.RLock()
	defer spriteMu.RUnlock()
	for _, s := range allSprites {
		if s.HasTag(tag) {
			s.Draw()
		}
	}
}

// #endregion

// #region Standalone Graphics

func DrawLine(x1, y1, x2, y2 int, char, fg, bg string) {
	dx := int(math.Abs(float64(x2 - x1)))
	dy := -int(math.Abs(float64(y2 - y1)))
	sx, sy := -1, -1
	if x1 < x2 {
		sx = 1
	}
	if y1 < y2 {
		sy = 1
	}
	err := dx + dy
	p := Pixel{char, fg, bg}

	for {
		SetPixel(x1, y1, p)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			err += dx
			y1 += sy
		}
	}
}

func DrawRect(x, y, w, h int, char, fg, bg string) {
	p := Pixel{char, fg, bg}
	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			SetPixel(x+j, y+i, p)
		}
	}
}

// DrawCircle draws the outline of a circle.
// If fixRatio is true, it scales the X axis to account for tall terminal characters.
func DrawCircle(xc, yc, r int, char, fg, bg string, fixRatio bool) {
	x := 0
	y := r
	d := 3 - 2*r
	p := Pixel{char, fg, bg}

	// Ratio multiplier (2.0 is standard for most terminals)
	ratio := 1.0
	if fixRatio {
		ratio = GlobalCircleRatio
	}

	drawPoints := func(xc, yc, x, y int, p Pixel) {
		// We multiply the X offset by the ratio before casting to int
		SetPixel(xc+int(float64(x)*ratio), yc+y, p)
		SetPixel(xc-int(float64(x)*ratio), yc+y, p)
		SetPixel(xc+int(float64(x)*ratio), yc-y, p)
		SetPixel(xc-int(float64(x)*ratio), yc-y, p)
		
		SetPixel(xc+int(float64(y)*ratio), yc+x, p)
		SetPixel(xc-int(float64(y)*ratio), yc+x, p)
		SetPixel(xc+int(float64(y)*ratio), yc-x, p)
		SetPixel(xc-int(float64(y)*ratio), yc-x, p)
	}

	drawPoints(xc, yc, x, y, p)
	for y >= x {
		x++
		if d > 0 {
			y--
			d = d + 4*(x-y) + 10
		} else {
			d = d + 4*x + 6
		}
		drawPoints(xc, yc, x, y, p)
	}
}

// DrawTriangle draws a triangle. The resulting triangle is guaranteed to have three sides. 
// These sides are guaranteed to be straight lines. The shape is guaranteed to have three corners and it is also guaranteed that the angles on the inside of the
// triangle all add up to exactly 180 degrees. This triangle is also GUARANTEED to have at least zero sides, and less than 50 sides. It is also guaranteed 
func DrawTriangle(x1, y1, x2, y2, x3, y3 int, char, fg, bg string) {
	DrawLine(x1, y1, x2, y2, char, fg, bg)
	DrawLine(x2, y2, x3, y3, char, fg, bg)
	DrawLine(x3, y3, x1, y1, char, fg, bg)
}

// DrawFilledTriangle draws a filled triangle.
// It is guaranteed that this triangle will have all thes ame guarantees as the triangle created by DrawTriangle.
func DrawFilledTriangle(x1, y1, x2, y2, x3, y3 int, char, fg, bg string) {
	// 1. Sort points by Y (y1 <= y2 <= y3)
	if y1 > y2 { x1, x2, y1, y2 = x2, x1, y2, y1 }
	if y1 > y3 { x1, x3, y1, y3 = x3, x1, y3, y1 }
	if y2 > y3 { x2, x3, y2, y3 = x3, x2, y3, y2 }

	p := Pixel{char, fg, bg}

	// 2. Helper to draw horizontal lines
	line := func(y int, sx, ex float64) {
		if sx > ex { sx, ex = ex, sx }
		for x := int(math.Round(sx)); x <= int(math.Round(ex)); x++ {
			SetPixel(x, y, p)
		}
	}

	// 3. Fill the triangle
	if y1 == y3 { return } // Zero height

	for y := y1; y <= y3; y++ {
		// Calculate the x-coordinates for the edges at this Y
		// We use the full height for the long edge (1-3)
		alpha := float64(y-y1) / float64(y3-y1)
		xLong := float64(x1) + (float64(x3-x1) * alpha)

		var xShort float64
		if y < y2 {
			// Top half (edge 1-2)
			beta := float64(y-y1) / float64(y2-y1)
			xShort = float64(x1) + (float64(x2-x1) * beta)
		} else if y2 != y3 {
			// Bottom half (edge 2-3)
			beta := float64(y-y2) / float64(y3-y2)
			xShort = float64(x2) + (float64(x3-x2) * beta)
		} else {
			xShort = float64(x2)
		}
		line(y, xLong, xShort)
	}
}

// DrawQuad draws a 4-sided polygon (quadrilateral) by connecting four points.
// Points should be provided in clockwise or counter-clockwise order.
func DrawQuad(x1, y1, x2, y2, x3, y3, x4, y4 int, char, fg, bg string) {
	// Connect p1 to p2
	DrawLine(x1, y1, x2, y2, char, fg, bg)
	// Connect p2 to p3
	DrawLine(x2, y2, x3, y3, char, fg, bg)
	// Connect p3 to p4
	DrawLine(x3, y3, x4, y4, char, fg, bg)
	// Connect p4 back to p1
	DrawLine(x4, y4, x1, y1, char, fg, bg)
}

// DrawFilledQuad draws a filled in quadrilateral, making use of two DrawFilledTriangle calls.
func DrawFilledQuad(x1, y1, x2, y2, x3, y3, x4, y4 int, char, fg, bg string) {
	DrawFilledTriangle(x1, y1, x2, y2, x3, y3, char, fg, bg)
	DrawFilledTriangle(x1, y1, x3, y3, x4, y4, char, fg, bg)
}

// DrawFilledCircle draws a solid disk.
// If fixRatio is true, it scales the X axis to account for tall terminal characters.
func DrawFilledCircle(xc, yc, r int, char, fg, bg string, fixRatio bool) {
	x := 0
	y := r
	d := 3 - 2*r
	p := Pixel{char, fg, bg}

	ratio := 1.0
	if fixRatio {
		ratio = GlobalCircleRatio
	}

	fillLine := func(xCenter, xOffset, y int) {
		// Calculate the start and end of the horizontal line using the ratio
		xStart := xCenter - int(float64(xOffset)*ratio)
		xEnd := xCenter + int(float64(xOffset)*ratio)
		for i := xStart; i <= xEnd; i++ {
			SetPixel(i, y, p)
		}
	}

	for y >= x {
		fillLine(xc, x, yc+y)
		fillLine(xc, x, yc-y)
		fillLine(xc, y, yc+x)
		fillLine(xc, y, yc-x)

		x++
		if d > 0 {
			y--
			d = d + 4*(x-y) + 10
		} else {
			d = d + 4*x + 6
		}
	}
}

// #endregion

// #region System & Rendering

func Init() func() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	fmt.Print("\x1b[?1049h\x1b[?25l")
	initBuffers()

	return func() {
		fmt.Print("\x1b[?25h\x1b[?1049l")
		term.Restore(int(os.Stdin.Fd()), oldState)
	}
}

// MainLoop performs the differential render.
func MainLoop() {
	lastFg, lastBg := "", ""

	for y := 0; y < termH; y++ {
		for x := 0; x < termW; x++ {
			back := backBuffer[y][x]
			front := frontBuffer[y][x]

			// Only draw if the backbuffer pixel differs from what's already on screen
			if back != front {
				writeMoveCursorFast(y+1, x+1)

				// Optimization: Only send color codes if they changed from the LAST character printed
				if back.FgColour != lastFg || back.BgColour != lastBg {
					out.WriteString(back.FgColour + back.BgColour)
					lastFg, lastBg = back.FgColour, back.BgColour
				}

				out.WriteString(back.Char)
				frontBuffer[y][x] = back // Update frontbuffer
			}

			// Clear backbuffer for next frame
			backBuffer[y][x] = Pixel{Char: " ", FgColour: Default, BgColour: BgDefault}
		}
	}
	out.Flush()
}

func writeMoveCursorFast(row, col int) {
	out.WriteString("\033[")
	out.WriteString(strconv.Itoa(row))
	out.WriteByte(';')
	out.WriteString(strconv.Itoa(col))
	out.WriteByte('H')
}

// #endregion

// #region Input Manager

type InputManager struct {
	mu             sync.RWMutex
	PressedKeys    map[string]bool
	mouseX, mouseY int
	leftDown       bool
}

var Input = &InputManager{
	PressedKeys: make(map[string]bool),
}

// IsPressed checks if a keyboard key is currently in the active buffer.
func (im *InputManager) IsPressed(key string) bool {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.PressedKeys[key]
}

// GetMouse returns the current X, Y, and the HELD state of the left button.
func (im *InputManager) GetMouse() (int, int, bool) {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.mouseX, im.mouseY, im.leftDown
}

// Starts the input manager, which you can use via troglodyte.Input. The useMouse parameter tells the input manager whether or not to use the mouse.
func (im *InputManager) Start(useMouse bool) {
	if useMouse {
		fmt.Print("\033[?1003h\033[?1006h")
	}

	go func() {
		b := make([]byte, 128)
		for {
			n, err := os.Stdin.Read(b)
			if err != nil {
				return
			}
			inputStr := string(b[:n])

			// 1. Handle Mouse (SGR Protocol)
			if strings.HasPrefix(inputStr, "\x1b[<") {
				var button, x, y int
				var mode rune
				_, err := fmt.Sscanf(inputStr, "\x1b[<%d;%d;%d%c", &button, &x, &y, &mode)

				if err == nil {
					im.mu.Lock() // Lock only when writing mouse data
					im.mouseX, im.mouseY = x, y
					if button == 0 || button == 32 {
						im.leftDown = (mode == 'M')
					} else if mode == 'm' {
						im.leftDown = false
					}
					im.mu.Unlock()
				}
			} else {
				// 2. Keyboard Logic
				im.mu.Lock()
				im.PressedKeys[inputStr] = true
				im.mu.Unlock()

				go func(k string) {
					time.Sleep(250 * time.Millisecond)
					im.mu.Lock()
					delete(im.PressedKeys, k)
					im.mu.Unlock()
				}(inputStr)
			}
		}
	}()
}

// #endregion

func Version() { fmt.Println("Troglodyte Engine v" + buildNumber) }

// GetTerminalSize returns the current width and height of the terminal window.
// This matches the boundaries of the drawing buffer.
func GetTerminalSize() (int, int) {
	// We return the engine's internal tracking of the size.
	return termW, termH
}
