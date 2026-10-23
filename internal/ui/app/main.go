package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-desktop/internal/ui/accesscontrol"
	"github.com/tacenva/tacenva-desktop/internal/ui/setting"
	"github.com/tacenva/tacenva-desktop/internal/ui/vault"
	"github.com/tacenva/tacenva-services/app"
	"github.com/tacenva/tacenva-services/app/sourceoftruth"
	coreApp "github.com/tacenva/tacpass-core/app"
	"github.com/tacenva/tacpass-core/entity"
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

	isAdmin := context.SelectedSot.Privilege == entity.PrivilegeAdmin
	var acScreen *accesscontrol.Screen
	if isAdmin {
		acScreen = accesscontrol.New(
			window,
			appDeps,
			context,
			coreService,
		)
	}

	settingScreen := setting.New(
		window,
		func(
			currentPassword,
			newPassword string,
		) error {
			return sotService.ChangePassword(
				currentPassword,
				newPassword,
			)
		},
	)

	title := widget.NewLabelWithStyle(
		"Tacenva",
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)

	backButton := widget.NewButtonWithIcon(
		context.SelectedSot.Hostname,
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
	showVault := func() {
		mainContent.Objects = []fyne.CanvasObject{
			vaultScreen.Content,
		}
		mainContent.Refresh()
	}

	showSettings := func() {
		mainContent.Objects = []fyne.CanvasObject{
			settingScreen.Content,
		}
		mainContent.Refresh()
	}

	var sidebar fyne.CanvasObject
	if isAdmin {
		showAccessControl := func() {
			mainContent.Objects = []fyne.CanvasObject{
				acScreen.Content,
			}
			mainContent.Refresh()
		}

		sidebar = newSidebar(
			showVault,
			showAccessControl,
			showSettings,
		)
	} else {
		sidebar = newSidebar(
			showVault,
			nil,
			showSettings,
		)
	}

	body := container.NewBorder(
		nil,
		nil,
		sidebar,
		nil,
		container.NewPadded(
			mainContent,
		),
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
