package main

import (
	"context"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func home() *fyne.Container {
	if program.node.list[0].ip == "" {
		create := &dialog.ConfirmDialog{}
		content := widget.NewEntry()
		content.SetPlaceHolder("127.0.0.1:10102")
		callback := func(b bool) {
			if !b {
				return
			}
			_, err := url.Parse("http://" + content.Text)
			if err != nil {
				showError(err, program.window)
				return
			}
			program.node.list[0].ip = content.Text
			if err := savePreferred(content.Text); err != nil {
				showError(err, program.window)
				return
			}
			cancelConnection()
			ctxConnection, cancelConnection = context.WithCancel(context.Background())
			go maintain_connection()
			create.Dismiss()
		}
		content.OnSubmitted = func(s string) { callback(true) }
		create = dialog.NewCustomConfirm("Set preferred connection", confirm, dismiss, content, callback, program.window)
		create.Show()
	}

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
	program.hyperlinks.generate.OnTapped = integrated_address_generator

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
		layout.NewSpacer(),
		container.NewAdaptiveGrid(
			3,
			layout.NewSpacer(),
			container.NewVBox(
				container.NewCenter(
					program.labels.balance,
				),
				container.NewAdaptiveGrid(
					2,
					container.NewCenter(
						program.hyperlinks.address,
					),
					container.NewCenter(
						program.hyperlinks.generate,
					),
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

			program.containers.register,

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
	if fyne.CurrentDevice().IsMobile() {
		main_ui = container.NewVBox(
			program.containers.topbar,
			layout.NewSpacer(),
			container.NewCenter(
				container.NewVBox(
					title,
					subtitle,
					layout.NewSpacer(),
				),
			),
			layout.NewSpacer(),
			container.NewVBox(
				container.NewCenter(
					program.labels.balance,
				),
				container.NewGridWithColumns(
					2,
					container.NewCenter(
						program.hyperlinks.address,
					),
					container.NewCenter(
						program.hyperlinks.generate,
					),
				),
			),
			container.NewVBox(program.containers.send),
			program.containers.register,

			program.containers.dashboard,

			layout.NewSpacer(),
			program.containers.bottombar,
		)
	}
	main_ui.Refresh()

	return container.NewStack(
		main_ui,
	)
}
