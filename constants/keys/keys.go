/*
 * Copyright (C) 2024 by Jason Figge
 */

package keys

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	ModifierNone    = 0
	ModifierControl = 1
)

type Key struct {
	code     rune
	modifier int
	name     string
	custom   bool
}

var (
	Null  = Key{modifier: ModifierControl, code: 0, name: "Null"}
	CtrlA = Key{modifier: ModifierControl, code: 1, name: "Ctrl+A"}
	CtrlB = Key{modifier: ModifierControl, code: 2, name: "Ctrl+B"}
	CtrlC = Key{modifier: ModifierControl, code: 3, name: "Ctrl+C"}
	CtrlD = Key{modifier: ModifierControl, code: 4, name: "Ctrl+D"}
	CtrlE = Key{modifier: ModifierControl, code: 5, name: "Ctrl+E"}
	CtrlF = Key{modifier: ModifierControl, code: 6, name: "Ctrl+F"}
	CtrlG = Key{modifier: ModifierControl, code: 7, name: "Ctrl+G"}
	CtrlH = Key{modifier: ModifierControl, code: 8, name: "Ctrl+H"}
	CtrlI = Key{modifier: ModifierControl, code: 9, name: "Ctrl+I"}
	CtrlJ = Key{modifier: ModifierControl, code: 10, name: "Ctrl+J"}
	CtrlK = Key{modifier: ModifierControl, code: 11, name: "Ctrl+K"}
	CtrlL = Key{modifier: ModifierControl, code: 12, name: "Ctrl+L"}
	CtrlM = Key{modifier: ModifierControl, code: 13, name: "Ctrl+M"}
	CtrlN = Key{modifier: ModifierControl, code: 14, name: "Ctrl+N"}
	CtrlO = Key{modifier: ModifierControl, code: 15, name: "Ctrl+O"}
	CtrlP = Key{modifier: ModifierControl, code: 16, name: "Ctrl+P"}
	CtrlQ = Key{modifier: ModifierControl, code: 17, name: "Ctrl+Q"}
	CtrlR = Key{modifier: ModifierControl, code: 18, name: "Ctrl+R"}
	CtrlS = Key{modifier: ModifierControl, code: 19, name: "Ctrl+S"}
	CtrlT = Key{modifier: ModifierControl, code: 20, name: "Ctrl+T"}
	CtrlU = Key{modifier: ModifierControl, code: 21, name: "Ctrl+U"}
	CtrlV = Key{modifier: ModifierControl, code: 22, name: "Ctrl+V"}
	CtrlW = Key{modifier: ModifierControl, code: 23, name: "Ctrl+W"}
	CtrlX = Key{modifier: ModifierControl, code: 24, name: "Ctrl+X"}
	CtrlY = Key{modifier: ModifierControl, code: 25, name: "Ctrl+Y"}
	CtrlZ = Key{modifier: ModifierControl, code: 26, name: "Ctrl+Z"}
	Esc   = Key{modifier: ModifierControl, code: 27, name: "Escape"}

	Up    = Key{modifier: 38, name: "Cursor Up"}
	Down  = Key{modifier: 40, name: "Cursor Down"}
	Left  = Key{modifier: 37, name: "Cursor Left"}
	Right = Key{modifier: 39, name: "Cursor Right"}

	A = Key{code: 'a', name: "A"}
	B = Key{code: 'b', name: "B"}
	C = Key{code: 'c', name: "C"}
	D = Key{code: 'd', name: "D"}
	E = Key{code: 'e', name: "E"}
	F = Key{code: 'f', name: "F"}
	G = Key{code: 'g', name: "G"}
	H = Key{code: 'h', name: "H"}
	I = Key{code: 'i', name: "I"}
	J = Key{code: 'j', name: "J"}
	K = Key{code: 'k', name: "K"}
	L = Key{code: 'l', name: "L"}
	M = Key{code: 'm', name: "M"}
	N = Key{code: 'n', name: "N"}
	O = Key{code: 'o', name: "O"}
	P = Key{code: 'p', name: "P"}
	Q = Key{code: 'q', name: "Q"}
	R = Key{code: 'r', name: "R"}
	S = Key{code: 's', name: "S"}
	T = Key{code: 't', name: "T"}
	U = Key{code: 'u', name: "U"}
	V = Key{code: 'v', name: "V"}
	W = Key{code: 'w', name: "W"}
	X = Key{code: 'x', name: "X"}
	Y = Key{code: 'y', name: "Y"}
	Z = Key{code: 'z', name: "Z"}
)

func ptr(k Key) *Key {
	return &k
}
func newKey(modifier int, code rune) *Key {
	return &Key{
		modifier: modifier,
		code:     code,
	}
}

func MakeKey(bs []byte) (*Key, bool) {
	size := len(bs)

	if size == 3 && bs[0] == 27 && bs[1] == 91 {
		switch bs[2] {
		case 65:
			return ptr(Up), true
		case 66:
			return ptr(Down), true
		case 67:
			return ptr(Right), true
		case 68:
			return ptr(Left), true
		}
	} else if size == 1 && bs[0] < 27 {
		return newKey(ModifierControl, rune(bs[0]+'A'-1)), true
	} else if size == 1 {
		return newKey(ModifierNone, rune(bs[0])), true
	} else if size == 2 && bs[0] == 27 {
		bs2 := make([]byte, 5)
		n := utf8.EncodeRune(bs2, 'ǎ')
		fmt.Printf("%d", n)
		r, _ := utf8.DecodeRune(bs[1:])
		fmt.Sprintf("%s", string(bs))
		return newKey(ModifierNone, r), true
	} else if size == 2 {
		r, _ := utf8.DecodeRune(bs)
		if r != utf8.RuneError {
			return newKey(ModifierNone, r), true
		} else {
			fmt.Printf("Hm...")
		}
	} else if size == 3 {
		k := newKey(0, 0)
		k.name = fmt.Sprintf("%s %s %s", string(bs[0]), string(bs[1]), string(bs[2]))
		return k, true
	} else if size == 4 {
		k := newKey(0, 0)
		k.name = fmt.Sprintf("%d %d %d %d", bs[0], bs[1], bs[2], bs[3])
		return k, true
	} else if size == 5 {
		k := newKey(0, 0)
		k.name = fmt.Sprintf("%d %d %d %d %d", bs[0], bs[1], bs[2], bs[3], bs[4])
		return k, true
	}
	return nil, false
}

func (k *Key) Ascii() rune {
	return k.code
}

func (k *Key) Modifier() int {
	return k.modifier
}

func (k *Key) Is(r rune) bool {
	return k.code == r
}

func (k *Key) Name() interface{} {
	if k.name != "" {
		return k.name
	}
	var name string
	if k.modifier == 0 {
		name = fmt.Sprintf("%s", string(rune(k.code)))
		if name == "" {
			name = fmt.Sprintf("Error: %+v", k)
		}
	} else {
		var modifiers []string
		if k.modifier&ModifierControl != 0 {
			modifiers = append(modifiers, "Ctrl")
		}
		name = fmt.Sprintf("%s+%s", strings.Join(modifiers, "+"), string(k.code))
	}
	return name
}
