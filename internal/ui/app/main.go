package app

import (
	stdcontext "context"
	"image/color"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-desktop/internal/ui/accesscontrol"
	"github.com/tacenva/tacenva-desktop/internal/ui/setting"
	"github.com/tacenva/tacenva-desktop/internal/ui/vault"
	"github.com/tacenva/tacenva-services/app"
	"github.com/tacenva/tacenva-services/app/health"
	"github.com/tacenva/tacenva-services/app/sourceoftruth"
	coreApp "github.com/tacenva/tacpass-core/app"
)

type Screen struct {
	Content           fyne.CanvasObject
	vaultScreen       *vault.Screen
	statusCircle      *canvas.Circle
	statusLabel       *widget.Label
	accessControlItem *navItem

	accessControlScreen *accesscontrol.Screen
	isAdmin             bool

	Context *app.Context

	cancelReachability stdcontext.CancelFunc
}

func New(
	window fyne.Window,
	appDeps *app.Deps,
	context *app.Context,
	version string,
	masterKey string,
	coreService *coreApp.Services,
	sotService *sourceoftruth.Service,
	onBack func(),
) *Screen {
	healthService := health.NewService(
		appDeps,
		context,
	)

	// Create a cancellable context only for remote nodes.
	var watcherContext stdcontext.Context
	var cancelWatcher stdcontext.CancelFunc

	if context.IsRemote {
		watcherContext, cancelWatcher = stdcontext.WithCancel(
			stdcontext.Background(),
		)

		// Complete the initial health check before creating the vault screen.
		// vault.New() immediately starts the initial vault list request.
		healthService.CheckReachabilityContext(watcherContext)
	}

	vaultScreen := vault.New(
		window,
		appDeps,
		context,
		masterKey,
		coreService,
		sotService,
	)

	isAdmin := context.SelectedSot.Auth.Level.AllowsAdmin()

	var screen *Screen
	var acScreen *accesscontrol.Screen

	settingScreen := setting.New(
		window,
		version,
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
		"Back",
		theme.NavigateBackIcon(),
		onBack,
	)

	statusCircle := canvas.NewCircle(color.NRGBA{
		R: 220,
		G: 53,
		B: 69,
		A: 255,
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

	showVault := func() {
		mainContent.Objects = []fyne.CanvasObject{
			vaultScreen.Content,
		}
		mainContent.Refresh()
	}

	showAccessControl := func() {
		if !isAdmin {
			return
		}

		if acScreen == nil {
			acScreen = accesscontrol.New(
				window,
				appDeps,
				context,
				coreService,
			)

			screen.accessControlScreen = acScreen
		}

		mainContent.Objects = []fyne.CanvasObject{
			acScreen.Content,
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
	var accessControlItem *navItem

	if isAdmin {
		sidebar, accessControlItem = newSidebar(
			showVault,
			showAccessControl,
			showSettings,
		)
	} else {
		sidebar, accessControlItem = newSidebar(
			showVault,
			nil,
			showSettings,
		)
	}

	sotIdentityLabel := widget.NewLabelWithStyle(
		"Source of Truth: "+context.SelectedSot.SotIdentity,
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	contentArea := container.NewBorder(
		container.NewPadded(sotIdentityLabel),
		nil,
		nil,
		nil,
		container.NewPadded(mainContent),
	)

	body := container.NewBorder(
		nil,
		nil,
		sidebar,
		nil,
		contentArea,
	)

	content := container.NewBorder(
		header,
		nil,
		nil,
		nil,
		body,
	)

	screen = &Screen{
		Content:           content,
		vaultScreen:       vaultScreen,
		statusCircle:      statusCircle,
		statusLabel:       statusLabel,
		accessControlItem: accessControlItem,

		accessControlScreen: acScreen,
		isAdmin:             isAdmin,

		Context:            context,
		cancelReachability: cancelWatcher,
	}

	showVault()

	if context.IsRemote {
		// Display the result of the initial health check.
		screen.updateReachabilityUI(
			watcherContext,
			healthService.IsReachable(),
			false,
		)

		// Configure the initial polling interval.
		if healthService.IsReachable() {
			appDeps.Client.SetAdaptiveInterval(60 * time.Second)
		} else {
			appDeps.Client.SetAdaptiveInterval(5 * time.Second)
		}

		// Start the background reachability watcher.
		go screen.watchReachability(
			watcherContext,
			healthService,
			appDeps.Client.SetAdaptiveInterval,
			appDeps.Client.AdaptiveInterval,
		)
	}

	return screen
}

// Stop cancels the reachability watcher when the node screen is closed.
func (s *Screen) Stop() {
	if s.cancelReachability != nil {
		s.cancelReachability()
		s.cancelReachability = nil
	}
}

// updateReachabilityUI updates the connection indicator on the Fyne UI thread.
func (s *Screen) updateReachabilityUI(
	ctx stdcontext.Context,
	reachable bool,
	reconnected bool,
) {
	fyne.Do(func() {
		// Ignore queued UI updates after the watcher has been stopped.
		if ctx.Err() != nil {
			return
		}

		if reachable {
			s.statusCircle.FillColor = color.NRGBA{
				R: 46,
				G: 160,
				B: 67,
				A: 255,
			}
			s.statusLabel.SetText("Connected")
		} else {
			s.statusCircle.FillColor = color.NRGBA{
				R: 220,
				G: 53,
				B: 69,
				A: 255,
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

		if reconnected &&
			s.isAdmin &&
			s.accessControlScreen != nil &&
			s.accessControlScreen.Reload != nil {
			s.accessControlScreen.Reload()
		}
	})
}

// watchReachability periodically checks the remote node connection status.
func (s *Screen) watchReachability(
	ctx stdcontext.Context,
	healthService *health.Service,
	setAdaptiveInterval func(time.Duration),
	getAdaptiveInterval func() time.Duration,
) {
	failures := 0
	wasReachable := healthService.IsReachable()

	for {
		interval := getAdaptiveInterval()
		if interval <= 0 {
			interval = 5 * time.Second
		}

		// Add a small random jitter to avoid synchronized polling.
		jitterRange := int64(interval / 5)
		var jitter time.Duration

		if jitterRange > 0 {
			jitter = time.Duration(rand.Int63n(jitterRange+1)) -
				interval/10
		}

		timer := time.NewTimer(interval + jitter)

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		if ctx.Err() != nil {
			return
		}

		// Use the cancellable context for the health check request.
		healthService.CheckReachabilityContext(ctx)

		if ctx.Err() != nil {
			return
		}

		reachable := healthService.IsReachable()
		reconnected := reachable && !wasReachable

		wasReachable = reachable

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

		s.updateReachabilityUI(
			ctx,
			reachable,
			reconnected,
		)
	}
}
