/*
 * Copyright (C) 2024 by Jason Figge
 */

package screen

const (
	At   = "\u001B[%d;%dH%s"
	Bell = "\a"

	SetColumn   = "\u001B[%dG"    // moves cursor to column n
	SetPosition = "\u001B[%d;%dH" // moves cursor to row n column m

	MoveUp        = "\u001B[%dA" // n rows up
	MoveDown      = "\u001B[%dB" // n rows down
	MoveRight     = "\u001B[%dC" // n columns right
	MoveLeft      = "\u001B[%dD" // n columns left
	MoveEndOfLine = "\u001B[K"

	ClearDown   = "\u001B[0J"          // clears from cursor until end of screen
	ClearUp     = "\u001B[1J"          // clears from cursor to beginning of screen
	ClearScreen = "\u001Bc\u001B[?25l" // clears entire screen

	ClearToEnd   = "\u001B[0K" // clears from cursor to end of line
	ClearToStart = "\u001B[1K" // clears from cursor to start of line
	ClearLine    = "\u001B[2K" // clears entire line
)
