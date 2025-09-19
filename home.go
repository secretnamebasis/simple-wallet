package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

func home() *fyne.Container {
	// when ever we go home, let's do this
	program.hyperlinks.home.OnTapped = func() {
		updateHeader(program.hyperlinks.home)

		// sometimes things fall through the cracks in the login screen
		if program.entries.wallet.Text != "" || program.entries.pass.Text != "" {
			program.entries.wallet.SetText("")
			program.entries.pass.SetText("")
		}

		// set container
		setContentAsHome()
	}

	// let's be clear about the software
	program.labels.notice = makeCenteredWrappedLabel(`
THIS SOFTWARE IS ALPHA STAGE SOFTWARE
USE ONLY FOR TESTING & EVALUATION PURPOSES 
`)

	title := canvas.NewText(`DERO`, theme.Color(theme.ColorNameForeground))
	title.TextSize = 48
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Refresh()
	middle := fyne.NewPos(
		program.size.Width/2,
		program.size.Height/2,
	)
	title.Move(middle)
	title.Resize(fyne.NewSize(program.size.Width/3, program.size.Height/3))

	subtitle := canvas.NewText(`PRIVACY TOGETHER`, theme.Color(theme.ColorNameForeground))
	subtitle.TextSize = 16
	subtitle.Alignment = fyne.TextAlignCenter
	subtitle.TextStyle = fyne.TextStyle{Italic: true}
	subtitle.Refresh()
	subtitle.Move(middle)

	program.labels.balance.TextStyle = fyne.TextStyle{
		Monospace: true,
	}

	main_ui := container.NewVBox(
		program.containers.topbar,
		layout.NewSpacer(),
		container.NewCenter(
			container.NewVBox(
				title,
				subtitle,
				layout.NewSpacer(),
			),
		),
		program.labels.notice,
		layout.NewSpacer(),
		container.NewAdaptiveGrid(
			3,
			layout.NewSpacer(),
			container.NewAdaptiveGrid(
				2,
				container.NewCenter(
					program.hyperlinks.address,
				),
				container.NewCenter(
					program.labels.balance,
				),
			),
			layout.NewSpacer(),
		),
		container.NewAdaptiveGrid(
			3,
			layout.NewSpacer(),
			container.NewVBox(program.containers.send),
			layout.NewSpacer(),
		),
		container.NewAdaptiveGrid(
			3,
			layout.NewSpacer(),

			container.NewCenter(program.containers.register),

			layout.NewSpacer(),
		),

		container.NewAdaptiveGrid(
			3,
			layout.NewSpacer(),
			program.containers.dashboard,
			layout.NewSpacer(),
		),

		layout.NewSpacer(),
		program.containers.bottombar,
	)
	main_ui.Refresh()
	return container.NewStack(
		main_ui,
	)
}
