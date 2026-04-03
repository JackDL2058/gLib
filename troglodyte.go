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
	buildNumber = "0.0.11" // Updated version for the Scene Tree rewrite
	out         = bufio.NewWriterSize(os.Stdout, 1024*64)

	// Global Registry for DrawAll and Tagged drawing
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

// Pixel represents a single character cell in the terminal.
type Pixel struct {
	Char     string
	FgColour string
	BgColour string
}

// Sprite is the fundamental building block of Troglodyte.
// It acts as a node in the Scene Tree and a graphical object.
type Sprite struct {
	X, Y    int       // Coordinates of the CENTRE of the sprite
	Width   int       // Calculated automatically
	Height  int       // Calculated automatically
	Pixels  [][]Pixel // Row-major: [y][x]
	Tags    []string
	Visible bool

	Parent   *Sprite
	Children []*Sprite
	mu       sync.RWMutex
}

// #endregion

// #region Sprite Management

// NewSprite creates a new sprite and adds it to the global registry.
// Pixels should be provided as [rows][columns].
func NewSprite(x, y int, pixels [][]Pixel) *Sprite {
	h := len(pixels)
	w := 0
	if h > 0 {
		w = len(pixels[0])
	}

	s := &Sprite{
		X:       x,
		Y:       y,
		Width:   w,
		Height:  h,
		Pixels:  pixels,
		Visible: true,
		Tags:    []string{},
	}

	spriteMu.Lock()
	allSprites = append(allSprites, s)
	spriteMu.Unlock()
	return s
}

// AddChild links a child sprite to this parent.
func (s *Sprite) AddChild(child *Sprite) {
	s.mu.Lock()
	defer s.mu.Unlock()
	child.Parent = s
	s.Children = append(s.Children, child)
}

// Move changes the sprite's position. If moveChildren is true, the delta is applied to all descendants.
func (s *Sprite) Move(dx, dy int, moveChildren bool) {
	s.X += dx
	s.Y += dy

	if moveChildren {
		for _, child := range s.Children {
			child.Move(dx, dy, true)
		}
	}
}

// AddTag adds a descriptive tag for group operations.
func (s *Sprite) AddTag(tag string) {
	s.Tags = append(s.Tags, tag)
}

// HasTag checks if the sprite contains a specific tag.
func (s *Sprite) HasTag(tag string) bool {
	for _, t := range s.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// #endregion

// #region Rendering Logic

// Draw renders the sprite to the buffer using its centre offset.
func (s *Sprite) Draw() {
	if !s.Visible {
		return
	}

	// Calculate top-left based on centre-pivot
	offsetX := s.X - (s.Width / 2)
	offsetY := s.Y - (s.Height / 2)

	for y, row := range s.Pixels {
		for x, px := range row {
			targetX := offsetX + x
			targetY := offsetY + y

			// Simple bounds check (terminal is generally 1-indexed for cursor)
			if targetX < 1 || targetY < 1 {
				continue
			}

			PrintAt(px.FgColour+px.BgColour+px.Char+Default+BgDefault, targetX, targetY)
		}
	}
}

// DrawAllSprites renders every sprite currently registered in the engine.
func DrawAllSprites() {
	spriteMu.RLock()
	defer spriteMu.RUnlock()
	for _, s := range allSprites {
		s.Draw()
	}
}

// DrawSpritesWithTag renders only sprites that possess the specified tag.
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

// DrawLine draws a line between two points using Bresenham's algorithm.
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

	for {
		PrintAt(fg+bg+char+Default+BgDefault, x1, y1)
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

// DrawRect draws a filled rectangle.
func DrawRect(x, y, w, h int, char, fg, bg string) {
	pixel := fg + bg + char + Default + BgDefault
	for i := 0; i < h; i++ {
		writeMoveCursorFast(y+i, x)
		for j := 0; j < w; j++ {
			out.WriteString(pixel)
		}
	}
}

// #endregion

// #region System & Terminal

// Init prepares the terminal and returns a restore function to be deferred.
func Init() func() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	// Enter alt buffer and hide cursor
	fmt.Print("\x1b[?1049h\x1b[?25l")

	return func() {
		fmt.Print("\x1b[?25h\x1b[?1049l")
		term.Restore(int(os.Stdin.Fd()), oldState)
	}
}

func MainLoop() {
	out.Flush()
	out.WriteString(CursorHome)
}

func PrintAt(text string, x, y int) {
	writeMoveCursorFast(y, x)
	out.WriteString(text)
}

func writeMoveCursorFast(row, col int) {
	out.WriteString("\033[")
	out.WriteString(strconv.Itoa(row))
	out.WriteByte(';')
	out.WriteString(strconv.Itoa(col))
	out.WriteByte('H')
}

func ClearScreen() {
	out.WriteString("\033[H\033[2J")
}

// #endregion

// #region Input Manager (Updated)
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
				// Simple key mapping
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
