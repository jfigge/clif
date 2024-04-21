/*
 * Copyright (C) 2024 by Jason Figge
 */

package screen

const (
	Bell = "\a"

	SetColumn   = "\u001b[%dG"    // moves cursor to column n
	SetPosition = "\u001b[%d;%dH" // moves cursor to row n column m

	MoveUp    = "\u001b[%dA" // n rows up
	MoveDown  = "\u001b[%dB" // n rows down
	MoveRight = "\u001b[%dC" // n columns right
	MoveLeft  = "\u001b[%dD" // n columns left

	ClearDown   = "\u001b[0J" // clears from cursor until end of screen
	ClearUp     = "\u001b[1J" // clears from cursor to beginning of screen
	ClearScreen = "\u001b[2J" // clears entire screen

	ClearEnd   = "\u001b[0K" // clears from cursor to end of line
	ClearStart = "\u001b[1K" // clears from cursor to start of line
	ClearLine  = "\u001b[2K" // clears entire line

)
