package exampleScripts

import (
	"time"

	"github.com/JackDL2058/troglodyte"
)

// Quick helper to create a pixel with just a character and color
func p(char string, fg string) troglodyte.Pixel {
	return troglodyte.Pixel{Char: char, FgColour: fg, BgColour: troglodyte.BgDefault}
}

// Player Pixels: A little knight/adventurer
// Size: 7x5
var playerPixels = [][]troglodyte.Pixel{
	{p(" ", ""), p(" ", ""), p("[", troglodyte.White), p("-", troglodyte.White), p("]", troglodyte.White), p(" ", ""), p(" ", "")},
	{p(" ", ""), p("(", troglodyte.Cyan), p("o", troglodyte.White), p("_", troglodyte.Cyan), p("o", troglodyte.White), p(")", troglodyte.Cyan), p(" ", "")},
	{p("/", troglodyte.Yellow), p("[", troglodyte.Red), p(" ", ""), p("T", troglodyte.Red), p(" ", ""), p("]", troglodyte.Red), p("\\", troglodyte.Yellow)},
	{p(" ", ""), p(" ", ""), p("[", troglodyte.Blue), p("_", troglodyte.Blue), p("]", troglodyte.Blue), p(" ", ""), p(" ", "")},
	{p(" ", ""), p(" ", ""), p("d", troglodyte.White), p(" ", ""), p("b", troglodyte.White), p(" ", ""), p(" ", "")},
}

// Hat Pixels: A simple wizard hat
// Size: 5x2
var wizardHatPixels = [][]troglodyte.Pixel{
	{p(" ", ""), p(" ", ""), p("^", troglodyte.Magenta), p(" ", ""), p(" ", "")},
	{p("/", troglodyte.Magenta), p("-", troglodyte.Magenta), p("-", troglodyte.Magenta), p("-", troglodyte.Magenta), p("\\", troglodyte.Magenta)},
}

func Movement2Example4() {
	restore := troglodyte.Init()
	defer restore()

	// 1. Create the Player at the center of the screen
	w, h := troglodyte.GetTerminalSize()
	player := troglodyte.NewSprite(w/2, h/2, playerPixels)

	// 2. Create the Hat.
	// Because of center-offset, placing it at the same X as the player
	// but slightly higher Y (Player Y - 3) will sit it perfectly on the head.
	hat := troglodyte.NewSprite(w/2, (h/2)-3, wizardHatPixels)

	// 3. Link them!
	player.AddChild(hat)

	troglodyte.Input.Start(true)

	vspeed := 20.
	hspeed := 40.

	for {
		dt := troglodyte.GetDeltaTime() // Get time since last frame

		// Move based on speed * dt
		if troglodyte.Input.IsPressed("w") {
			player.Move(0, -vspeed*dt, true)
		}
		if troglodyte.Input.IsPressed("s") {
			player.Move(0, vspeed*dt, true)
		}
		if troglodyte.Input.IsPressed("a") {
			player.Move(-hspeed*dt, 0, true)
		}
		if troglodyte.Input.IsPressed("d") {
			player.Move(hspeed*dt, 0, true)
		}

		troglodyte.DrawAllSprites()
		troglodyte.MainLoop()

		// We still sleep slightly to prevent 100% CPU usage
		time.Sleep(10 * time.Millisecond)
	}
}
