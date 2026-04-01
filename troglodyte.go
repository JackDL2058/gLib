// Copyright (c) 2026 Jack Durnin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package troglodyte

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/term"
)

var (
	buildNumber = "0.0.10" // The current build number, in the format major.minor.patch. See Version() for more info on how the version number works.

	keyHandled = make(map[string]bool) // flags to prevent continuous key adding
	// Optimization: bufio.Writer writes directly to the OS buffer, bypasses heavy string allocations.
	out             = bufio.NewWriterSize(os.Stdout, 1024*64)
	nextAvailableID = 0 // used for creating objects.
)

// #region graphics, objects

// Strings that represent colours in the terminal using ANSI escape codes.
// To use them, add the colour before the text you want to be coloured, like so:
//
// fmt.Println(troglodyte.Red + "This text will be red!" + troglodyte.Default)
//
// These are also the colours used in the graphics drawer. You could set the colour of a
// graphics drawer to Green, and draw text at certain coordinates with that colour, and
// the default code at the end wouldn't be needed, as the grapics drawer would handle that
// automatially.
//
// If you don't want to deal with the graphics drawer stuff, you could use a text drawer or
// printAt, to print text at a certain location. You could also use MoveCursor to move to
// a certain position and simply use fmt.Print to print your coloured text.
const (
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Default = "\033[39m"

	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
	BgDefault = "\033[49m"

	BgGray          = "\033[100m"
	BgBrightRed     = "\033[101m"
	BgBrightGreen   = "\033[102m"
	BgBrightYellow  = "\033[103m"
	BgBrightBlue    = "\033[104m"
	BgBrightMagenta = "\033[105m"
	BgBrightCyan    = "\033[106m"
	BgBrightWhite   = "\033[107m"

	// cursor control ANSI escape codes:

	// Moves cursor to row 1, column 1.
	CursorHome = "\033[H"
	// Clears from cursor to the end of the screen
	ClearEnd = "\033[J"
	// Optional: Makes the cursor (the text input symbol in the terminal, not your mouse) invisible.
	HideCursor = "\033[?25l"
	// Optional: Brings cursor back into view. See HideCursor.
	ShowCursor = "\033[?25h"
)

// A single pixel, for use in sprites.
type Pixel struct {
	symbol   string
	colour   string
	bgColour string
}

// A list of pixels. A single row of a sprite. Contains columns for that row.
type SpriteRow struct {
	pixels []Pixel // A list of pixels in that row.
}

// A large data structure to store pixel data for an object's sprite.
type Sprite struct {
	size int // the vertical size of the sprite, amount of rows.
	rows []SpriteRow
}

// A basic object with a graphics drawrer and sprite component. It has an x and y position that can be used in the draw
// functions of the graphics drawer, like drawing the sprite component at whatever position.
// To create one with certain components, use these functions:
//
// objectWithoutSpriteOrGraphics := NewBlankObject(x, y)
// objectWithSpriteAndGraphics := NewObject(x, y, graphicsDrawer, sprite) // sprite is optional
type Object struct {
	graphicsDrawer *GraphicsDrawer // The graphics drawer. Can be left empty, but then you won't have any graphics functions or sprite drawing.
	sprite         *Sprite         // The attached sprite. Can be left as empty. To use sprites, you'll need a graphics drawer.
	x              int
	y              int
	children       []int // children ids, other objects, set using object.AddChild(id). An object can't be its own child/parent. Children follow their parent's movement.
	id             int   // self id, will be set automatically when creating a new object.
}

// Adds a child to an object using its ID. A child can only be an object, not a component. Children follow their parent's movement.
// If you use a number that isn't attached to an object, that WILL crash the game by index out of range. To get the Id of an object to
// add it as a child to another object, use newChild.id as the id. Make sure newChild is an object, not a component, or everything breaks.
func (o *Object) AddChild(id int) {
	if id == o.id {
		return // prevent self parent/child relationship
	}

	if slices.Contains(o.children, id) {
		return // already a child, do nothing
	}

	o.children = append(o.children, id) // add the child
}

// Creates a new object. Children, graphics, and sprite can be added later if needed using the AddChild, AddGraphicsDrawer and AddSprite functions respectively.
// When adding graphics and sprites in this constructor function, use each one's constructor function: newObjectGraphics() and newObjectSprite(). You need to use
// the versions of these functions with 'object' in the name so that they are linked to the object.
func NewObject(x, y int, graphics *GraphicsDrawer, sprite *Sprite) *Object {
	nextAvailableID += 1 // increment ID
	return &Object{graphics, sprite, x, y, []int{}, nextAvailableID}
}

// Draws graphics to the terminal like shapes and lines. This is basically to differentiate between objects,
// so you can use the graphicsDrawer functions outside of the graphicsDrawer if you want, like just calling line(),
// but you'll have to specify colour and all that stuff with every call. GraphicsDrawer can also be used as
// a component of an Object struct by calling newGraphicsDrawer() in the constructor for that Object,
// and then using object.graphicsDrawer.Line().
//
// A GraphicsDrawer is required to draw the sprite of an object
// using the DrawSprite function. The DrawSpecificSprite function draws a specified sprite struct rather than
// the one attached to the object. This is useful if you have multiple sprites for one object, for whatever reason.
// Sprite drawing does not take into account the colour and symbol of the graphics drawer.
// When creating a new graphics drawer and a new object in the same constructor, like so:
//
// object := NewObject(x, y, newGraphicsDrawer(colour, bgColour, symbol), sprite)
//
// the graphics drawer will automatically be given the same ID as the object, so they will link together.
// Once linked, the graphics drawer uses the position of the object when drawing. For example, to draw a line from the parent object to 50, 78 you could use:
//
// object.graphicsDrawer.Line(x, y, 50, 78) // the x and y values are the parent object's position.
//
// You can still use it without x and y and it will work like a normal graphics drawer. However, when linked, and when you do this:
//
// object.GraphicsDrawer.DrawSprite()
//
// The sprite drawn is actually the sprite value of the parent object. To draw a different sprite, simply use:
//
// object.GraphicsDrawer.DrawSpecificSprite(spriteName)
//
// The DrawSprite() function isn't available outside of a linked graphicsDrawer. If called by a regular one, it does nothing.
type GraphicsDrawer struct {
	colour   string
	bgColour string
	x        int // the x of the parent object, if ther is one. 0 if not.
	y        int // the y of the parent object, if there is one. 0 if not.
	symbol   string
	id       int // self ID, 0 if not used as a component of an object.
}

// Draws a line using the graphics drawer's colour, bgColour and symbol. From (x1, y1) to (x2, y2). Uses Bresenham's line algorithm.
func (gd *GraphicsDrawer) Line(x1, y1, x2, y2 int) {
	pixel := gd.colour + gd.bgColour + gd.symbol + Default
	// Bresenham's line algorithm
	dx := math.Abs(float64(x2 - x1))
	dy := math.Abs(float64(y2 - y1))
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx - dy

	for {
		PrintAt(pixel, x1, y1)
		if x1 == x2 && y1 == y2 {
			break
		}
		err2 := err * 2
		if err2 > -dy {
			err -= dy
			x1 += sx
		}
		if err2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// Draws a square using the graphics drawer's colour, bgColour and symbol.
func (gd *GraphicsDrawer) Square(x, y, size int) {

	// precalculate pixel for optimization, also means that if the GraphicsDrawer data is edited, it still keeps printing the same colour
	pixel := gd.bgColour + gd.colour + gd.symbol

	for i := 0; i < size; i++ { // rows
		for k := 0; k < size; k++ { // columns
			PrintAt(pixel, x+k, y+i)
		}
	}
}

// writes text into the screen buffer.
// Optimization: Writes directly to the buffered writer to avoid string builder overhead.
func PrintAt(text string, x, y int) {
	writeMoveCursorFast(y, x)
	out.WriteString(text)
}

// writeMoveCursorFast provides an allocation-free way to move the cursor via the buffer.
func writeMoveCursorFast(row, col int) {
	out.WriteString("\033[")
	out.WriteString(strconv.Itoa(row))
	out.WriteByte(';')
	out.WriteString(strconv.Itoa(col))
	out.WriteByte('H')
}

// #endregion

// #region Input
type InputManager struct {
	mu              sync.RWMutex
	PressedKeys     map[string]bool
	prevPressedKeys map[string]bool
	// Mouse state
	mouseX       int
	mouseY       int
	leftClicked  bool
	rightClicked bool
}

var Input = &InputManager{
	PressedKeys:     make(map[string]bool),
	prevPressedKeys: make(map[string]bool),
}

// JustReleased returns true only on the single frame a key transitions from pressed to not pressed.
func (im *InputManager) JustReleased(key string) bool {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return !im.PressedKeys[key] && im.prevPressedKeys[key]
}

// starts the input manager.
func (im *InputManager) Start(useMouse bool) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	// Enable mouse tracking: 1003 (all motion), 1006 (SGR extended mode)
	if useMouse {
		fmt.Print("\033[?1003h\033[?1006h")
	}

	go func() {
		defer term.Restore(int(os.Stdin.Fd()), oldState)
		// Disable mouse tracking on exit
		defer fmt.Print("\033[?1003l\033[?1006l")

		//b := make([]byte, 3)
		b := make([]byte, 64) // large buffer to accommodate mouse sequences
		for {
			n, err := os.Stdin.Read(b)
			if err != nil {
				break
			}

			key := ""
			inputStr := string(b[:n])

			if useMouse {
				// Check for Mouse SGR Sequence: ESC[<button;x;y;M (Press) or m (Release)
				if n > 5 && strings.HasPrefix(inputStr, "\x1b[<") {
					var button, x, y int
					var mode rune
					_, err := fmt.Sscanf(inputStr, "\x1b[<%d;%d;%d%c", &button, &x, &y, &mode)
					if err == nil {
						im.mu.Lock()
						im.mouseX = x
						im.mouseY = y
						// button 0 = left click, 'M' = pressed
						if button == 0 && mode == 'M' {
							im.leftClicked = true
						}
						im.mu.Unlock()
					}
					continue
				}
			}

			if n == 1 {
				switch b[0] {
				case 27:
					key = "ESC"
				case 13:
					key = "ENTER"
				case 127:
					key = "BACKSPACE"
				default:
					if b[0] >= 1 && b[0] <= 26 {
						// Ctrl + letter (Ctrl+A = 1, ..., Ctrl+Z = 26)
						letter := string(b[0] + 64)
						key = "Ctrl+" + letter
					} else {
						key = string(b[0])
					}
				}
			} else if n > 1 && b[0] == 27 {
				switch string(b[1:n]) {
				case "[A":
					key = "UP"
				case "[B":
					key = "DOWN"
				case "[C":
					key = "RIGHT"
				case "[D":
					key = "LEFT"
				}
			}

			if key != "" {
				im.mu.Lock()
				im.PressedKeys[key] = true
				im.mu.Unlock()

				go func(k string) {
					time.Sleep(100 * time.Millisecond)
					im.mu.Lock()
					delete(im.PressedKeys, k)
					im.mu.Unlock()
				}(key)
			}
		}
	}()
}

// If a certain key is currently being held down OR just pressed. To see if a key is being held down, use !JustPressed(key).
// To check if a key was just pressed, use JustHandled(key).
func (im *InputManager) IsPressed(key string) bool {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.PressedKeys[key]
}

// gets the current mouse position in terminal coordinates. Note that this is not supported in all
// terminals, and may not work as expected in some environments.
func (im *InputManager) GetCurrentMousePos() (int, int) {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.mouseX, im.mouseY
}

// JustLeftClicked returns true if a left click occurred since the last check
func (im *InputManager) JustLeftClicked() bool {
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.leftClicked {
		im.leftClicked = false
		return true
	}
	return false
}

// JustRightClicked returns true if a right click occurred since the last check
func (im *InputManager) JustRightClicked() bool {
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.rightClicked {
		im.rightClicked = false
		return true
	}
	return false
}

// Deprecated: JustPressed returns true if the key is currently pressed but wasn't pressed in the previous frame.
// use JustHandled instead, it is more reliable and works better overall.
func (im *InputManager) JustPressed(key string) bool {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.PressedKeys[key] && !im.prevPressedKeys[key]
}

// JustHandled returns true if the specified key has just been pressed, and returns false if the key has been held down for more than 1 frame.
// Similar to JustPressed, but uses better system of 'handling' keys so that they don't get constantly typed if you press for more than a frame.
// This solution also accounts for echo, so if you purposefully hold down a key it works like you would expect in something like word.
func (im *InputManager) JustHandled(key string) bool {
	im.mu.Lock()
	defer im.mu.Unlock()

	isDown := im.PressedKeys[key]

	// If the key is released, reset the handled state for next time
	if !isDown {
		keyHandled[key] = false
		return false
	}

	// If it's down but already handled, return false
	if keyHandled[key] {
		return false
	}

	// If it's down and NOT handled, handle it now and return true
	keyHandled[key] = true
	return true
}

// GetCurrentTypableKey returns the first pressed typable key (printable character), so no keys like ctrl + A,
// or returns an empty string if none are pressed. This can be used for debugging purposes to see if Input is running.
// it is not recommended to use this for something like text input, because it works every frame, unlike JustHandled.
//
// here's some more info about keys if you're interested:
// control keys like Ctrl + A can be detected using JustHandled("Ctrl+A"), but they won't show up in GetCurrentTypableKey since
// they aren't printable characters. stuff like enter, esc, backspace and arrow keys are represented as "ENTER", "ESC",
// "BACKSPACE", "UP", "DOWN", "LEFT", "RIGHT" respectively in JustHandled. Something like space is represented as " "
// and stuff like Ctrl + Shift + A would be "Ctrl+Shift+A" in JustHandled. Tab is represented as "\t" and function
// keys like F1 and F4 are not supported at all in the current version of troglodyte, but may be added in the future.
func (im *InputManager) GetCurrentTypableKey() string {
	im.mu.RLock()
	defer im.mu.RUnlock()
	for key, pressed := range im.PressedKeys {
		if pressed && len(key) == 1 && unicode.IsPrint(rune(key[0])) {
			return key
		}
	}
	return ""
}

// #endregion

// Prints the version number. Same as version(). See version() for more info on how the version number works.
func Test() { fmt.Println(buildNumber) }

// A developer test function, you can call this in your main function to see if troglodyte has been initialized correctly.
// Gives the build number for testing purposes. Here's how the version number works:
// first number: Major version, version 1.0.0 would be the first major stable release. A complete rewrite or restructuring would be a new major version.
// second number: minor version, works almost the same as previous version, just additional features like adding new functions.
// third number: patch version, for small correction, but patches don't add new features.
//
// If you care, there's also a bugfix version, like 0.1.4-6, where -6 is the bugfix version. This number essentially doesn't matter because it's
// actually just the amount of commits since the last patch version, meaning if multiple commits are spent on one patch, the bugfix goes up. It's
// basically useless, but keep it in mind as it could be helpful to know if you're on a slightly different version than someone else.
//
// The version number only goes up for this file, so license changes or changes to other stuff won't change the version numbers but will change the commit
// identifier or version number on github. It's also not updated automatically, so some bugfix numbers might not be correct.
func Version() { fmt.Println(buildNumber) }

// Starts troglodyte and prepares the terminal to display graphics. Does not start Input. Use troglodyte.Input.Start()
// to start the input manager if you need it. Init must be called before any graphics or Input stuff,
// or bad things will happen, like spamming your terminal with control bytes or closing it entirely.
// So make sure to call Init before other troglodyte functions! This also doesn't have to be for troglodyte functions.
// This works perfectly fine to put the terminal into a good state for custom drawing logic or other stuff.
func Init() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	defer RestoreTerminal()
	InitializeTerminal()

}

func Exit() {
	RestoreTerminal()
	os.Exit(0)
}

// The MainLoop function must be run each frame through a loop on the user end. The user's code handles logic like where characters should move, what
// the inputs do to affect the gameplay, etc, but once that's done and the variables are changed, they won't visibly change until mainLoop() is called.
// This function handles the actual rendering of graphics to the terminal screen. The user simply tells the program what to render, through graphicsDrawers,
// but this function is what actually draws stuff each frame.
func MainLoop() {
	out.WriteString(CursorHome) // Move cursor to top-left before drawing the frame
	out.Flush()                 // Flush the high-speed buffer directly to Stdout
}

// Developer function. This is handled automatically when you call Init.
// puts the terminal into an alternate screen buffer to preserve previous commands until restoreTerminal is called.
// This also hides the cursor so you can print without having a blinking cursor flashing across the screen.
func InitializeTerminal() { fmt.Print("\x1b[?1049h\x1b[?25l") }

// Developer function. This is handled when you call Init.
// Fixes everything in InitializeTerminal, and puts things back to normal, showing the cursor and
// going back to the main screen buffer.
func RestoreTerminal() { fmt.Print("\x1b[?25h\x1b[?1049l") }

// Uses a variable to prevent errors, for temporary usage if you're trying to make many things work at once.
// Another thing you could do is declare a variable and put this on the same line, like this:
//
// var h = "hello"; Use(h);
//
// to prevent the 'h declared and not used' error. Remember to remove this from your code when you're done, since it doesn't actually do anything, and wastes memory.
// (This was added for my convenience. I'm the kind of programmer to need this kind of thing.)
func Use(h any) { d := h; h = d }

// Does nothing.
// (This was added for my convenience. I'm the kind of programmer to need this kind of thing.)
func Pass() {}

// Clears the terminal screen.
func ClearScreen() {
	// \033[H moves to top-left, \033[2J clears the screen buffer
	out.WriteString("\033[H\033[2J")
	out.Flush()
}

func MoveCursor(row, col int) {
	writeMoveCursorFast(row, col)
	out.Flush()
}

func SMoveCursor(row, col int) string {
	return "\033[" + strconv.Itoa(row) + ";" + strconv.Itoa(col) + "H"
}