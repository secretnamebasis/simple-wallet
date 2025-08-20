// this is as generic as it gets.
// the disabled text color is too dark, imo; so...
// custom theme!
package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type customTheme struct{}

func (customTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameDisabled { // so that you can see disabled text
		return color.CMYK{ // darker green
			C: 100, M: 0, Y: 100, K: 61,
		}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (customTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (customTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameText {
		return 18
	}
	return theme.DefaultTheme().Size(name)
}
