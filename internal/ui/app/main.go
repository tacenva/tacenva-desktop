package app

import (
	"image/color"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	tacenvaentity "github.com/tacenva/tacenva-core/entity"
	"github.com/tacenva/tacenva-desktop/internal/ui/accesscontrol"
	"github.com/tacenva/tacenva-desktop/internal/ui/setting"
	"github.com/tacenva/tacenva-services/app"
	"github.com/tacenva/tacenva-services/app/health"
	"github.com/tacenva/tacenva-services/app/sourceoftruth"
	coreApp "github.com/tacenva/tacpass-core/app"
)

type Screen struct {
	Content fyne.CanvasObject
	// vaultScreen       *vault.Screen
	statusCircle      *canvas.Circle
	statusLabel       *widget.Label
	accessControlItem *navItem

	accessControlScreen *accesscontrol.Screen
	isAdmin             bool

	Context *app.Context
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
	// vaultScreen := vault.New(
	// 	window,
	// 	appDeps,
	// 	context,
	// 	masterKey,
	// 	coreService,
	// 	sotService,
	// )

	isAdmin := context.SelectedSot.Auth.Level == tacenvaentity.LevelAdmin

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
		func(currentPassword, newPassword string) error {
			return sotService.ChangePassword(
				currentPassword,
				newPassword,
			)
		},
	)

	title := widget.NewLabelWithStyle(
		"Tacenva",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	backButton := widget.NewButtonWithIcon(
		context.SelectedSot.Hostname,
		theme.NavigateBackIcon(),
		onBack,
	)

	statusCircle := canvas.NewCircle(color.NRGBA{
		R: 220, G: 53, B: 69, A: 255,
	})
	statusCircle.Resize(fyne.NewSize(10, 10))

	statusLabel := widget.NewLabel("Disconnected...")

	statusRow := container.NewHBox(
		container.NewCenter(
			container.NewGridWrap(
				fyne.NewSize(10, 10),
				statusCircle,
			),
		),
		statusLabel,
	)

	var headerRight fyne.CanvasObject

	if context.IsRemote {
		statusIndicator := container.NewGridWrap(
			fyne.NewSize(
				statusRow.MinSize().Width,
				backButton.MinSize().Height,
			),
			container.NewCenter(statusRow),
		)

		headerRight = container.NewHBox(
			container.NewPadded(statusIndicator),
			container.NewGridWrap(
				fyne.NewSize(16, 1),
				container.NewWithoutLayout(),
			),
			backButton,
		)
	} else {
		headerRight = container.NewHBox(backButton)
	}

	header := container.NewBorder(
		nil,
		nil,
		title,
		headerRight,
		nil,
	)

	mainContent := container.NewMax()

	// showVault := func() {
	// 	mainContent.Objects = []fyne.CanvasObject{
	// 		vaultScreen.Content,
	// 	}
	// 	mainContent.Refresh()
	// }

	showSettings := func() {
		mainContent.Objects = []fyne.CanvasObject{
			settingScreen.Content,
		}
		mainContent.Refresh()
	}

	var sidebar fyne.CanvasObject
	var accessControlItem *navItem

	if isAdmin {
		showAccessControl := func() {
			if acScreen == nil {
				return
			}

			mainContent.Objects = []fyne.CanvasObject{
				acScreen.Content,
			}
			mainContent.Refresh()
		}

		sidebar, accessControlItem = newSidebar(
			// showVault,
			nil,
			showAccessControl,
			showSettings,
		)
	} else {
		sidebar, accessControlItem = newSidebar(
			// showVault,
			nil,
			nil,
			showSettings,
		)
	}

	body := container.NewBorder(
		nil,
		nil,
		sidebar,
		nil,
		container.NewPadded(mainContent),
	)

	content := container.NewBorder(
		header,
		nil,
		nil,
		nil,
		body,
	)

	screen := &Screen{
		Content: content,
		// vaultScreen:       vaultScreen,
		statusCircle:      statusCircle,
		statusLabel:       statusLabel,
		accessControlItem: accessControlItem,

		accessControlScreen: acScreen,
		isAdmin:             isAdmin,

		Context: context,
	}

	if context.IsRemote {
		healthService := health.NewService(
			appDeps,
			context,
		)

		go screen.watchReachability(
			healthService.CheckReachability,
			appDeps.Client.IsReachable,
			appDeps.Client.SetAdaptiveInterval,
			appDeps.Client.AdaptiveInterval,
		)
	}

	return screen
}

func (s *Screen) watchReachability(
	check func(),
	getReachable func() bool,
	setAdaptiveInterval func(time.Duration),
	getAdaptiveInterval func() time.Duration,
) {
	failures := 0
	hasPreviousStatus := false
	wasReachable := false

	for {
		check()
		reachable := getReachable()

		reconnected := hasPreviousStatus && reachable && !wasReachable

		wasReachable = reachable
		hasPreviousStatus = true

		fyne.Do(func() {
			if reachable {
				s.statusCircle.FillColor = color.NRGBA{
					R: 46, G: 160, B: 67, A: 255,
				}
				s.statusLabel.SetText("Connected")
			} else {
				s.statusCircle.FillColor = color.NRGBA{
					R: 220, G: 53, B: 69, A: 255,
				}
				s.statusLabel.SetText("Disconnected...")
			}

			if s.accessControlItem != nil {
				s.accessControlItem.SetDisabled(
					!reachable || !s.isAdmin,
				)
			}

			s.statusCircle.Refresh()
			s.statusLabel.Refresh()

			// Reload hanya saat koneksi pulih.
			if reconnected &&
				s.isAdmin &&
				s.accessControlScreen != nil &&
				s.accessControlScreen.Reload != nil {
				s.accessControlScreen.Reload()
			}
		})

		if reachable {
			failures = 0
			setAdaptiveInterval(60 * time.Second)
		} else {
			failures++

			switch failures {
			case 1:
				setAdaptiveInterval(5 * time.Second)
			case 2:
				setAdaptiveInterval(10 * time.Second)
			default:
				setAdaptiveInterval(20 * time.Second)
			}
		}

		interval := getAdaptiveInterval()
		if interval <= 0 {
			interval = 5 * time.Second
		}

		jitter := time.Duration(rand.Int63n(int64(interval/5)+1)) -
			interval/10

		time.Sleep(interval + jitter)
	}
}
