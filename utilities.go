/*
 * Copyright (C) 2024 by Jason Figge
 */

package clif

import (
	"fmt"
)

func toInt(key string, value interface{}) (int, error) {
	var i int
	var f float64
	var ok bool
	if i, ok = value.(int); ok {
		return i, nil
	} else if f, ok = value.(float64); ok {
		return int(f), nil
	} else {
		return 0, fmt.Errorf("invalid data type - %s != %s", key, "int")
	}
}

func toBool(key string, value interface{}) (bool, error) {
	var b bool
	var ok bool
	if b, ok = value.(bool); ok {
		return b, nil
	} else {
		return false, fmt.Errorf("invalid data type - %s != %s", key, "int")
	}
}

func toString(key string, value interface{}) (string, error) {
	var s string
	var ok bool
	if s, ok = value.(string); ok {
		return s, nil
	} else {
		return "", fmt.Errorf("invalid data type - %s != %s", key, "int")
	}
}
