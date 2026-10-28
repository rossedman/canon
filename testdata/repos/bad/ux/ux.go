// Package ux imports a terminal library the canon replaces.
package ux

import "github.com/fatih/color"

// Warn prints in red.
func Warn(s string) {
	color.Red(s)
}
