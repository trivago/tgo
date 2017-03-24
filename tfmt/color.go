package tfmt

import (
	"strconv"
)

// Color provides ANSI color support. As Color implements Stringer you can
// simply add a color to your print commands like fmt.Print(Red,"Hello world").
type Color int

const (
	// NoColor use the default color
	NoColor = Color(0)
	// Black color
	Black = Color(30)
	// Red color
	Red = Color(31)
	// Green color
	Green = Color(32)
	// Yellow color
	Yellow = Color(33)
	// Blue color
	Blue = Color(34)
	// Purple color
	Purple = Color(35)
	// Cyan color
	Cyan = Color(36)
	// Gray color
	Gray = Color(37)
	// DarkGray color (bold Black)
	DarkGray = Color(-30)
	// BrightRed color (bold Red)
	BrightRed = Color(-31)
	// BrightGreen color (bold Green)
	BrightGreen = Color(-32)
	// BrightYellow color (bold Yellow)
	BrightYellow = Color(-33)
	// BrightBlue color (bold Blue)
	BrightBlue = Color(-34)
	// BrightPurple color (bold Purple)
	BrightPurple = Color(-35)
	// BrightCyan color (bold Cyan)
	BrightCyan = Color(-36)
	// White color (bold Gray)
	White = Color(-37)
)

// String implements the stringer interface for color
func (c Color) String() string {
	if int(c) < 0 {
		return "\x1b[1m\x1b[" + strconv.Itoa(int(c)) + "m\x1b[22m"
	}
	return "\x1b[" + strconv.Itoa(int(c)) + "m"
}
