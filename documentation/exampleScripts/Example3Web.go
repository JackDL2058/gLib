package exampleScripts

import (
	"math"
	"time"

	"github.com/JackDL2058/troglodyte"
)

var tension = 60       // the max distance a line can go before going under tension
var cx, cy = 0, 0      // position of center object
var drawTension = true // whether to draw red lines to show tension

func Web() {
	restore := troglodyte.Init()
	defer restore()

	troglodyte.Input.Start(true)     // Enable mouse
	centerVelX, centerVelY := 0., 0. // Velocity of center point
	basePullStrength := 3            // the base strength of each spring
	color := troglodyte.White

	for {
		dt := troglodyte.GetDeltaTime()

		mx, my, lc := troglodyte.Input.GetMouse()
		w, h := troglodyte.GetTerminalSize()

		if lc {
			centerVelX, centerVelY = 0, 0
			cx = mx
			cy = my
		} else {

			// I know this is horrible code but I can't think of anything else so i'm going with this

			// hardcode the line positions
			x1, y1, x2, y2 := w/2, 1, cx, cy   // Line 1: Top-Center
			x3, y3, x4, y4 := w/2, h, cx, cy   // Line 2: Bottom-Center
			x5, y5, x6, y6 := w, h/2, cx, cy   // Line 3: Right-Center
			x7, y7, x8, y8 := 1, h/2, cx, cy   // Line 4: Left-Center
			x9, y9, x10, y10 := w, h, cx, cy   // Line 5: Bottom-Right
			x11, y11, x12, y12 := w, 1, cx, cy // Line 6: Top-Right
			x13, y13, x14, y14 := 1, h, cx, cy // Line 7: Bottom-Left
			x15, y15, x16, y16 := 1, 1, cx, cy // Line 8: Top-Left

			// the base position of each line, corresponds to distances slice
			var basesX = []int{w / 2, w / 2, w, 1, w, w, 1, 1}
			var basesY = []int{1, h, h / 2, h / 2, h, 1, h, 1}

			// calculate every line's distance components
			dx1, dy1 := float64(x1-x2), float64(y1-y2)
			dx2, dy2 := float64(x3-x4), float64(y3-y4)
			dx3, dy3 := float64(x5-x6), float64(y5-y6)
			dx4, dy4 := float64(x7-x8), float64(y7-y8)
			dx5, dy5 := float64(x9-x10), float64(y9-y10)
			dx6, dy6 := float64(x11-x12), float64(y11-y12)
			dx7, dy7 := float64(x13-x14), float64(y13-y14)
			dx8, dy8 := float64(x15-x16), float64(y15-y16)

			// calculate the actual diagonal distances:
			// distance = sqrt(dx^2 + dy^2)
			distances := []float64{0, 0, 0, 0, 0, 0, 0, 0} // stores the distances of each line
			distances[0] = math.Sqrt(dx1*dx1 + dy1*dy1)
			distances[1] = math.Sqrt(dx2*dx2 + dy2*dy2)
			distances[2] = math.Sqrt(dx3*dx3 + dy3*dy3)
			distances[3] = math.Sqrt(dx4*dx4 + dy4*dy4)
			distances[4] = math.Sqrt(dx5*dx5 + dy5*dy5)
			distances[5] = math.Sqrt(dx6*dx6 + dy6*dy6)
			distances[6] = math.Sqrt(dx7*dx7 + dy7*dy7)
			distances[7] = math.Sqrt(dx8*dx8 + dy8*dy8)

			// actually do the physics calculations
			for i, v := range distances { // loop through distances
				if v > float64(tension) { // only pull if over tension
					// calculate strength
					additionalPullStrength := (v - float64(tension)) / 2 // more distance = more force
					totalPullStrength := additionalPullStrength + float64(basePullStrength)

					relativeX := basesX[i] - cx
					relativeY := basesY[i] - cy
					additionalXVel := math.Cos(math.Atan2(float64(relativeY), float64(relativeX)))
					additionalYVel := math.Sin(math.Atan2(float64(relativeY), float64(relativeX)))

					centerVelX += additionalXVel * totalPullStrength
					centerVelY += additionalYVel * totalPullStrength
				}
			}

			// make the center actually move
			cx += int(centerVelX * dt)
			cy += int(centerVelY * dt)

			// make velocities go down so that energy isn't created

			// Smooth damping (90% retention) instead of hard subtraction
			friction := 0.92
			centerVelX *= friction
			centerVelY *= friction

			// Stop it completely if it's crawling to prevent "infinite jitter"
			if math.Abs(centerVelX) < 0.1 {
				centerVelX = 0
			}
			if math.Abs(centerVelY) < 0.1 {
				centerVelY = 0
			}
		}
		// Draw a line from the top/center to the mouse
		drawLineUnderTension(w/2, 1, cx, cy, "·", color, troglodyte.BgBlack)

		// draw more lines on the center of each side
		drawLineUnderTension(w/2, h, cx, cy, "·", color, troglodyte.BgBlack)
		drawLineUnderTension(w, h/2, cx, cy, "·", color, troglodyte.BgBlack)
		drawLineUnderTension(1, h/2, cx, cy, "·", color, troglodyte.BgBlack)

		// draw lines from corners
		drawLineUnderTension(w, h, cx, cy, "·", color, troglodyte.BgBlack)
		drawLineUnderTension(w, 1, cx, cy, "·", color, troglodyte.BgBlack)
		drawLineUnderTension(1, h, cx, cy, "·", color, troglodyte.BgBlack)
		drawLineUnderTension(1, 1, cx, cy, "·", color, troglodyte.BgBlack)

		// draw circles
		troglodyte.GlobalFixRatio = 2.5
		troglodyte.DrawCircle(cx, cy, 5, ".", color, troglodyte.BgBlack, true)
		troglodyte.DrawCircle(cx, cy, 15, ".", color, troglodyte.BgBlack, true)
		troglodyte.DrawCircle(cx, cy, 25, ".", color, troglodyte.BgBlack, true)
		troglodyte.DrawCircle(cx, cy, 35, ".", color, troglodyte.BgBlack, true)
		troglodyte.DrawCircle(cx, cy, 50, ".", color, troglodyte.BgBlack, true)

		// Draw a rectangle at the center point
		troglodyte.DrawRect(cx, cy, 4, 2, "▒", troglodyte.Cyan, troglodyte.BgBlack)

		troglodyte.MainLoop()
		time.Sleep(10 * time.Millisecond)
	}
}

// draws a line but in red if it exceeds a certain tension point.
func drawLineUnderTension(x1, y1, x2, y2 int, char, fg, bg string) {
	// 1. Calculate the difference for each axis
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)

	// 2. Calculate the actual diagonal distance (C)
	// distance = sqrt(dx^2 + dy^2)
	actualDistance := math.Sqrt(dx*dx + dy*dy)

	if actualDistance > float64(tension) {
		// under tension
		if drawTension {
			troglodyte.DrawLine(x1, y1, x2, y2, char, troglodyte.Red, troglodyte.BgRed)
		}
	} else {
		// normal drawing
		troglodyte.DrawLine(x1, y1, x2, y2, char, fg, bg)
	}
}
