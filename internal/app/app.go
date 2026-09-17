package app

import (
	"fmt"

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
		fmt.Print(err)
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
	loginScreen := login.New(window, appDeps, services, sotService)
	window.SetContent(loginScreen.Content)
	window.ShowAndRun()
}
