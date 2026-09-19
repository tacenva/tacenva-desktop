package login

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-desktop/internal/ui/discovery"
	"github.com/tacenva/tacenva-desktop/internal/ui/sourceoftruth"
	"github.com/tacenva/tacenva-services/app"
	rCoreSot "github.com/tacenva/tacenva-services/app/sourceoftruth"
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
	discoveryState *discovery.State,
	openNode sourceoftruth.OpenNodeFunc,
) *Screen {
	title := widget.NewLabelWithStyle(
		"Tacenva",
		fyne.TextAlignCenter,
		fyne.TextStyle{
			Bold: true,
		},
	)

	subtitle := widget.NewLabelWithStyle(
		"Enter your master password to unlock.",
		fyne.TextAlignCenter,
		fyne.TextStyle{},
	)

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Master password")

	errorLabel := widget.NewLabel("")
	errorLabel.Alignment = fyne.TextAlignCenter

	discoveryStatus := widget.NewLabel(
		"Discovering Source of Truth...",
	)

	discoveryLoading := widget.NewProgressBarInfinite()

	discoveryIndicator := container.NewHBox(
		discoveryLoading,
		discoveryStatus,
	)

	discoveryBar := container.New(
		layout.NewCustomPaddedLayout(
			8,
			8,
			12,
			12,
		),
		discoveryIndicator,
	)

	updateDiscoveryUI := func() {
		discoveryLoading.Hide()

		if discoveryState.Error() != nil {
			discoveryStatus.SetText(
				"Discovery failed",
			)
		} else {
			discoveryStatus.SetText(
				fmt.Sprintf(
					"Discovery complete · %d server(s) found",
					discoveryState.Count(),
				),
			)
		}

		discoveryBar.Refresh()
	}

	if discoveryState.Done() {
		updateDiscoveryUI()
	} else {
		discoveryState.OnDone(func() {
			fyne.Do(updateDiscoveryUI)
		})
	}

	unlock := func() {
		err := sotService.Access(password.Text)
		if err != nil {
			errorLabel.SetText(err.Error())
			password.SetText("")
			password.FocusGained()
			return
		}

		sotScreen := sourceoftruth.New(
			window,
			appDeps,
			password.Text,
			coreService,
			sotService,
			discoveryState,
			openNode,
		)

		window.SetContent(sotScreen.Content)
	}

	unlockButton := widget.NewButton(
		"Unlock",
		unlock,
	)

	password.OnSubmitted = func(_ string) {
		unlock()
	}

	form := container.NewVBox(
		title,
		subtitle,
		password,
		errorLabel,
		unlockButton,
	)

	loginContent := container.NewCenter(
		container.NewGridWrap(
			fyne.NewSize(360, 220),
			form,
		),
	)

	discoveryBarRight := container.NewHBox(
		layout.NewSpacer(),
		discoveryBar,
	)

	content := container.NewBorder(
		nil,
		discoveryBarRight,
		nil,
		nil,
		loginContent,
	)

	return &Screen{
		Content:    content,
		sotService: sotService,
	}
}
