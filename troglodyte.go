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
	buildNumber = "0.0.12" // Build version
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
	X, Y     int
	Width    int
	Height   int
	Pixels   [][]Pixel
	Tags     []string
	Visible  bool
	Parent   *Sprite
	Children []*Sprite
	mu       sync.RWMutex
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
		X: x, Y: y, Width: w, Height: h,
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

func (s *Sprite) Move(dx, dy int, moveChildren bool) {
	s.X += dx
	s.Y += dy
	if moveChildren {
		for _, child := range s.Children {
			child.Move(dx, dy, true)
		}
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
	if !s.Visible {
		return
	}
	offsetX := s.X - (s.Width / 2)
	offsetY := s.Y - (s.Height / 2)

	for y, row := range s.Pixels {
		for x, px := range row {
			SetPixel(offsetX+x, offsetY+y, px)
		}
	}
}

func DrawAllSprites() {
	spriteMu.RLock()
	defer spriteMu.RUnlock()
	for _, s := range allSprites {
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
	clicked        bool
}

var Input = &InputManager{
	PressedKeys: make(map[string]bool),
}

func (im *InputManager) Start(useMouse bool) {
	if useMouse {
		fmt.Print("\033[?1003h\033[?1006h")
	}
	go func() {
		b := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(b)
			if err != nil {
				return
			}
			inputStr := string(b[:n])

			im.mu.Lock()
			if strings.HasPrefix(inputStr, "\x1b[<") {
				fmt.Sscanf(inputStr, "\x1b[<%d;%d;%d", new(int), &im.mouseX, &im.mouseY)
				im.clicked = strings.HasSuffix(inputStr, "M")
			} else {
				key := inputStr
				if n == 1 && b[0] == 27 {
					key = "ESC"
				}
				if n == 1 && b[0] == 13 {
					key = "ENTER"
				}
				im.PressedKeys[key] = true
				go func(k string) {
					time.Sleep(50 * time.Millisecond)
					im.mu.Lock()
					delete(im.PressedKeys, k)
					im.mu.Unlock()
				}(key)
			}
			im.mu.Unlock()
		}
	}()
}

func (im *InputManager) GetMouse() (int, int, bool) {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.mouseX, im.mouseY, im.clicked
}

func (im *InputManager) IsPressed(key string) bool {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.PressedKeys[key]
}

// #endregion

func Version() { fmt.Println("Troglodyte Engine v" + buildNumber) }

// GetTerminalSize returns the current width and height of the terminal window.
// This matches the boundaries of the drawing buffer.
func GetTerminalSize() (int, int) {
	// We return the engine's internal tracking of the size.
	return termW, termH
}
