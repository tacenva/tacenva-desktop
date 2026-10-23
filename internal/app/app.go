package app

import (
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/tacenva/tacenva-desktop/internal/ui/discovery"
	"github.com/tacenva/tacenva-desktop/internal/ui/login"

	replicaCore "github.com/tacenva/tacenva-services/app"
	"github.com/tacenva/tacenva-services/app/sourceoftruth"

	coreapp "github.com/tacenva/tacpass-core/app"
)

func Run(dev bool) {
	a := app.NewWithID("com.tacenva.password")

	window := a.NewWindow("Tacenva Password")
	window.Resize(fyne.NewSize(1000, 650))

	appDeps, err := replicaCore.Setup(dev)
	if err != nil {
		log.Println(err)
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

	loginScreen := login.New(
		window,
		appDeps,
		services,
		sotService,
		discoveryState,
	)

	window.SetContent(loginScreen.Content)
	window.Show()

	go func() {
		log.Println("[SERVER DISCOVERY] starting...")

		servers, err := appDeps.ServerService.Discover(
			5 * time.Second,
		)

		if err != nil {
			log.Printf(
				"[SERVER DISCOVERY] warn: %v\n",
				err,
			)
		} else {
			log.Printf(
				"[SERVER DISCOVERY] found %d server(s)\n",
				len(servers),
			)

			for _, server := range servers {
				address := server.Address()
				dialAddress := server.DialAddress()

				log.Printf(
					"server: source=%s name=%q host=%q ip=%s port=%d\n",
					server.Source,
					server.Name,
					server.Host,
					server.IP,
					server.Port,
				)

				log.Printf(
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

		log.Println("[SERVER DISCOVERY] done")

		// Harus selalu Complete, termasuk saat discovery error.
		// Kalau tidak, UI akan tetap disabled selamanya.
		discoveryState.Complete(
			len(servers),
			err,
		)
	}()

	window.ShowAndRun()
}
