package login

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/tacenva/replica-core/app"
	rCoreSot "github.com/tacenva/replica-core/app/sourceoftruth"
	"github.com/tacenva/tacenva-desktop/internal/ui/sourceoftruth"
	coreApp "github.com/tacenva/tacpass-core/app"
)

type Screen struct {
	Content    fyne.CanvasObject
	sotService *rCoreSot.Service
}

func New(
	window fyne.Window,
	appDeps *app.Deps,
	coreService *coreApp.Services,
	sotService *rCoreSot.Service,
) *Screen {
	title := widget.NewLabelWithStyle(
		"Tacenva",
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)

	subtitle := widget.NewLabel(
		"Masukkan master password untuk membuka Tacenva.",
	)

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Master password")

	errorLabel := widget.NewLabel("")
	errorLabel.Alignment = fyne.TextAlignCenter

	unlockButton := widget.NewButton("Unlock", func() {
		err := sotService.Access(password.Text)
		if err != nil {
			errorLabel.SetText(err.Error())
			return
		}

		sotScreen := sourceoftruth.New(
			window,
			appDeps,
			password.Text,
			coreService,
			sotService,
		)
		window.SetContent(sotScreen.Content)
	})

	form := container.NewVBox(
		title,
		subtitle,
		password,
		errorLabel,
		unlockButton,
	)

	content := container.NewCenter(
		container.NewGridWrap(
			fyne.NewSize(360, 220),
			form,
		),
	)

	return &Screen{
		Content: content,
	}
}
