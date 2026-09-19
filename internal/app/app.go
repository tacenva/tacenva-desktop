package app

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/tacenva/database"

	uiapp "github.com/tacenva/tacenva-desktop/internal/ui/app"
	"github.com/tacenva/tacenva-desktop/internal/ui/discovery"
	"github.com/tacenva/tacenva-desktop/internal/ui/login"

	replicaCore "github.com/tacenva/tacenva-services/app"
	"github.com/tacenva/tacenva-services/app/sourceoftruth"
	"github.com/tacenva/tacenva-services/entity"

	coreapp "github.com/tacenva/tacpass-core/app"
	"github.com/tacenva/tacpass-core/config"
)

func Run(dev bool) {
	a := app.NewWithID("com.tacenva.password")

	window := a.NewWindow("Tacenva Password")
	window.Resize(fyne.NewSize(1000, 650))

	appDeps, err := replicaCore.Setup(dev)
	if err != nil {
		fmt.Println(err)
		return
	}

	services := coreapp.NewServices(
		appDeps.SqliteDB,
		appDeps.AppDB,
	)

	sotService := sourceoftruth.NewService(
		appDeps,
		services.Auth,
		services.AccessControl,
	)

	discoveryState := discovery.NewState()

	openNode := func(
		selectedSot *entity.SourceOfTruth,
		masterKey string,
		onBack func(),
	) fyne.CanvasObject {
		nodeDBDir := appDeps.Config.Path(
			config.NodeDirName,
			selectedSot.ID,
			"vault",
		)

		nodeContext := &replicaCore.Context{
			SelectedSoT: selectedSot,
			NodeDB:      database.New(nodeDBDir),
			IsRemote:    selectedSot.Address != "localhost",
		}

		nodeScreen := uiapp.New(
			window,
			appDeps,
			nodeContext,
			masterKey,
			services,
			sotService,
			onBack,
		)

		return nodeScreen.Content
	}

	loginScreen := login.New(
		window,
		appDeps,
		services,
		sotService,
		discoveryState,
		openNode,
	)

	window.SetContent(loginScreen.Content)
	window.Show()

	go func() {
		fmt.Println("starting server discovery...")

		servers, err := appDeps.ServerService.Discover(
			5 * time.Second,
		)

		if err != nil {
			fmt.Printf(
				"server discovery error: %v\n",
				err,
			)
		} else {
			fmt.Printf(
				"server discovery found %d server(s)\n",
				len(servers),
			)

			for _, server := range servers {
				address := server.Address()
				dialAddress := server.DialAddress()

				fmt.Printf(
					"server: source=%s name=%q host=%q ip=%s port=%d\n",
					server.Source,
					server.Name,
					server.Host,
					server.IP,
					server.Port,
				)

				fmt.Printf(
					"configuring dial address: %s -> %s\n",
					address,
					dialAddress,
				)

				appDeps.Client.ConfigureDialAddress(
					address,
					dialAddress,
				)
			}
		}

		// Harus selalu Complete, termasuk saat discovery error.
		// Kalau tidak, UI akan tetap disabled selamanya.
		discoveryState.Complete(
			len(servers),
			err,
		)
	}()

	window.ShowAndRun()
}
