package app

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	replicaCore "github.com/tacenva/replica-core/app"
	"github.com/tacenva/replica-core/app/sourceoftruth"
	"github.com/tacenva/tacenva-desktop/internal/ui/login"
	coreapp "github.com/tacenva/tacpass-core/app"
)

func Run() {
	a := app.NewWithID("com.tacenva.password")

	window := a.NewWindow("Tacenva Password")
	window.Resize(fyne.NewSize(1000, 650))
	appDeps, err := replicaCore.Setup(true)
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
		services.Permission,
	)
	loginScreen := login.New(window, appDeps, services, sotService)
	window.SetContent(loginScreen.Content)
	window.ShowAndRun()
}
