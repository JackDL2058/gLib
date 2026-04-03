package exampleScripts

import (
	"time"
	"github.com/JackDL2058/troglodyte"
)

func Example1() {
	restore := troglodyte.Init()
	defer restore()

	// Create a 3x3 block (The Parent)
	px := troglodyte.Pixel{Char: "█", FgColour: troglodyte.Red}
	pixels := [][]troglodyte.Pixel{{px, px, px}, {px, px, px}, {px, px, px}}
	player := troglodyte.NewSprite(20, 10, pixels)

	// Create a child "hat" for the player
	hatPx := troglodyte.Pixel{Char: "^", FgColour: troglodyte.Yellow}
	hat := troglodyte.NewSprite(20, 8, [][]troglodyte.Pixel{{hatPx, hatPx, hatPx}})
	player.AddChild(hat)
	hat.AddTag("hat") // Add tag to the hat

	troglodyte.Input.Start(false)

	// main game loop
	for {
		
		if troglodyte.Input.IsPressed("w") { player.Move(0, -1, true) }
		if troglodyte.Input.IsPressed("s") { player.Move(0, 1, true) }
		if troglodyte.Input.IsPressed("a") { player.Move(-2, 0, true) }
		if troglodyte.Input.IsPressed("d") { player.Move(2, 0, true) }
		
		troglodyte.DrawAllSprites()

		troglodyte.MainLoop()
		time.Sleep(30 * time.Millisecond)
	}
}