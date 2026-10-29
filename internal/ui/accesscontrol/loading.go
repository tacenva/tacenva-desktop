package accesscontrol

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func runWithLoading(
	window fyne.Window,
	title string,
	message string,
	action func() error,
	onSuccess func(),
	onError ...func(error),
) {
	progress := widget.NewProgressBarInfinite()
	label := widget.NewLabel(message)

	loadingDialog := dialog.NewCustomWithoutButtons(
		title,
		container.NewVBox(label, progress),
		window,
	)

	progress.Start()
	loadingDialog.Show()

	go func() {
		err := action()

		fyne.Do(func() {
			progress.Stop()
			loadingDialog.Hide()

			if err != nil {
				if len(onError) > 0 && onError[0] != nil {
					onError[0](err)
					return
				}

				dialog.ShowError(err, window)
				return
			}

			if onSuccess != nil {
				onSuccess()
			}
		})
	}()
}
