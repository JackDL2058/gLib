package exampleScripts

import (
	"time"
	"github.com/JackDL2058/troglodyte"
)

func Example2() {
	restore := troglodyte.Init()
	defer restore()

	troglodyte.Input.Start(true) // Enable mouse

	for {
		troglodyte.ClearScreen()
		
		mx, my, clicked := troglodyte.Input.GetMouse()
		
		color := troglodyte.White
		if clicked { color = troglodyte.Green }
		
		// Draw a line from the top corner to the mouse
		troglodyte.DrawLine(1, 1, mx, my, "·", color, troglodyte.BgBlack)
		
		// Draw a rectangle at the mouse
		troglodyte.DrawRect(mx, my, 4, 2, "▒", troglodyte.Cyan, troglodyte.BgBlack)
		
		troglodyte.MainLoop()
		time.Sleep(16 * time.Millisecond)
	}
}