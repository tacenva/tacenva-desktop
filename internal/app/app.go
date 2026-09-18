package app

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

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
		fmt.Println(err)
		return
	}

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

	services := coreapp.NewServices(
		appDeps.SqliteDB,
		appDeps.AppDB,
	)

	sotService := sourceoftruth.NewService(
		appDeps,
		services.Auth,
		services.AccessControl,
	)

	loginScreen := login.New(
		window,
		appDeps,
		services,
		sotService,
	)

	window.SetContent(loginScreen.Content)
	window.ShowAndRun()
}
