package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/replica-core/app"
	"github.com/tacenva/replica-core/app/sourceoftruth"
	"github.com/tacenva/tacenva-desktop/internal/ui/accesscontrol"
	"github.com/tacenva/tacenva-desktop/internal/ui/vault"
	coreApp "github.com/tacenva/tacpass-core/app"
)

type Screen struct {
	Content     fyne.CanvasObject
	vaultScreen *vault.Screen
}

func New(
	window fyne.Window,
	appDeps *app.Deps,
	context *app.Context,
	masterKey string,
	coreService *coreApp.Services,
	sotService *sourceoftruth.Service,
	onBack func(),
) *Screen {
	vaultScreen := vault.New(
		window,
		appDeps,
		context,
		masterKey,
		coreService,
		sotService,
	)

	acScreen := accesscontrol.New(
		window,
		appDeps,
		context,
		coreService,
	)

	title := widget.NewLabelWithStyle(
		"Tacenva",
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)

	backButton := widget.NewButtonWithIcon(
		context.SelectedSoT.Hostname,
		theme.NavigateBackIcon(),
		onBack,
	)

	header := container.NewBorder(
		nil,
		nil,
		title,
		backButton,
		nil,
	)

	mainContent := container.NewMax()

	sidebar := newSidebar(
		func() {
			mainContent.Objects = []fyne.CanvasObject{
				vaultScreen.Content,
			}
			mainContent.Refresh()
		},
		func() {
			mainContent.Objects = []fyne.CanvasObject{
				acScreen.Content,
			}
			mainContent.Refresh()
		},
		func() {
			mainContent.Objects = []fyne.CanvasObject{}
			mainContent.Refresh()
		},
	)

	body := container.NewBorder(
		nil,
		nil,
		sidebar,
		nil,
		mainContent,
	)

	content := container.NewBorder(
		header,
		nil,
		nil,
		nil,
		body,
	)

	return &Screen{
		Content:     content,
		vaultScreen: vaultScreen,
	}
}
