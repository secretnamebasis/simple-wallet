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
	if style.Monospace {
		return theme.DefaultTheme().Font(style)
	}
	if style.Bold {
		if style.Italic {
			return resourceNotoSansBoldItalicTtf
		}
		return resourceNotoSansBoldTtf
	}
	if style.Italic {
		return resourceNotoSansItalicTtf
	}
	return resourceNotoSansRegularTtf
}

func (customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (customTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (customTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
