package main

//import e "github.com/JackDL2058/troglodyte/documentation/exampleScripts"
import (
	"fmt"
	"time"

	"github.com/JackDL2058/troglodyte"
	s "github.com/JackDL2058/troglodyte/troglosprite"
)

func main() {
	//e.Movement2Example4()
	//e.Web()
	sprite := s.PngToPixel("asepriteStuffIgnore/bigpalette.png", true, false)
	// CRITICAL: Check this BEFORE Init()

	if len(sprite) == 0 {
		fmt.Println("Error: Sprite data is empty. Is the path correct?")
		return
	}

	restore := troglodyte.Init()
	defer restore()
	troglodyte.Input.Start(false)
	logo := troglodyte.NewSprite(16, 9, sprite)
	logo.AddTag("logo")

	for {
		if troglodyte.Input.IsPressed("a") {
			logo.Move(2, 0, false)
		}
		if troglodyte.Input.IsPressed("d") {
			logo.Move(-2, 0, false)
		}
		if troglodyte.Input.IsPressed("s") {
			logo.Move(0, -1, false)
		}
		if troglodyte.Input.IsPressed("w") {
			logo.Move(0, 1, false)
		}
		troglodyte.DrawSpritesWithTag("logo")
		//troglodyte.DrawAllSprites()
		troglodyte.MainLoop()
		time.Sleep(16 * time.Millisecond)
	}
}
