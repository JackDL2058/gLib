// Copyright (c) 2026 Jack Durnin. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package troglodyte

import (
	"fmt"
	"sync"
	"golang.org/x/term"
	"time"
	"os"
	"unicode"
	"strings"
)

var( 
	keyHandled = make(map[string]bool) // flags to prevent continuous key adding
)

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
	if useMouse {fmt.Print("\033[?1003h\033[?1006h")}

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

			if useMouse{
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

// A developer test function, you can call this in your main function to see if troglodyte has been initialized correctly.
func Test() {
	fmt.Println("test")
}

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

// Developer function. This is handled automatically when you call Init.
// puts the terminal into an alternate screen buffer to preserve previous commands until restoreTerminal is called.
// This also hides the cursor so you can print without having a blinking cursor flashing across the screen.
func InitializeTerminal() { fmt.Print("\x1b[?1049h\x1b[?25l") }
// Developer function. This is handled when you call Init.
// Fixes everything in InitializeTerminal, and puts things back to normal, showing the cursor and 
// going back to the main screen buffer.
func RestoreTerminal()    { fmt.Print("\x1b[?25h\x1b[?1049l") }

// Uses a variable to prevent errors, for temporary usage if you're trying to make many things work at once.
// Another thing you could do is declare a variable and put this on the same line, like 
// 
// 'var h = "hello"; Use(h);' 
// 
// to prevent the 'h declared and not used' error. Remeber to remove this when you're done, since it doesn't actually do anything.
func Use(h any) {d := h;h = d;}

// Does nothing.
func Pass(){}

// Clears the terminal screen.
func ClearScreen() {
	// \033[H moves to top-left, \033[2J clears the screen buffer
	fmt.Print("\033[H\033[2J")
}
func MoveCursor(row, col int) {
	fmt.Printf("\033[%d;%dH", row, col)
}
