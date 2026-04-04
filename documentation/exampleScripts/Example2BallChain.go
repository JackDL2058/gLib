package exampleScripts

import (
	"time"
	"github.com/JackDL2058/troglodyte"
)

func BallChain() {
	restore := troglodyte.Init()
	defer restore()

	troglodyte.Input.Start(true) // Enable mouse

	for {
		
		mx, my, clicked := troglodyte.Input.GetMouse()
		w, _ := troglodyte.GetTerminalSize()
		
		color := troglodyte.White
		if clicked { color = troglodyte.Green }
		
		// Draw a line from the top corner to the mouse
		troglodyte.DrawLine(w/2, 1, mx, my, "·", color, troglodyte.BgBlack)
		
		// Draw a rectangle at the mouse
		troglodyte.DrawRect(mx-2, my, 5, 2, "▒", troglodyte.Cyan, troglodyte.BgBlack)

		
		troglodyte.MainLoop()
		time.Sleep(16 * time.Millisecond)
	}
}