package loading

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type View struct {
	Content *fyne.Container
}

func New(text string) *View {
	spinner := widget.NewProgressBarInfinite()

	label := widget.NewLabel(text)

	content := container.NewMax(
		container.NewCenter(
			container.NewVBox(
				spinner,
				label,
			),
		),
	)

	return &View{
		Content: content,
	}
}

func (v *View) Show(content fyne.CanvasObject) {
	v.Content.Objects = []fyne.CanvasObject{
		content,
	}

	v.Content.Refresh()
}

func (v *View) ShowError(message string) {
	v.Content.Objects = []fyne.CanvasObject{
		container.NewCenter(
			widget.NewLabel(message),
		),
	}

	v.Content.Refresh()
}
