package vault

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/replica-core/app"
	"github.com/tacenva/replica-core/app/sourceoftruth"
	"github.com/tacenva/replica-core/app/vault"
	"github.com/tacenva/tacenva-desktop/internal/ui/vaultrecord"
	coreApp "github.com/tacenva/tacpass-core/app"
	"github.com/tacenva/tacpass-core/entity"
)

type Screen struct {
	Content fyne.CanvasObject
	vault   []entity.Vault
}

func New(
	window fyne.Window,
	appDeps *app.Deps,
	context *app.Context,
	masterKey string,
	coreService *coreApp.Services,
	sotService *sourceoftruth.Service,
) *Screen {
	vaultService := vault.NewService(
		appDeps,
		context,
		masterKey,
		coreService.Vault,
		coreService.Auth,
		sotService,
	)

	search := widget.NewEntry()
	search.SetPlaceHolder("Search passwords...")

	vaultAccesses, needSync, err := vaultService.List()
	if err != nil {
		dialog.ShowError(err, window)
		vaultAccesses = nil
	}

	var entryList *widget.List
	var syncButton *widget.Button

	reload := func() {
		updatedVaultAccesses, newNeedSync, err := vaultService.List()
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		vaultAccesses = updatedVaultAccesses

		if syncButton != nil {
			if newNeedSync {
				syncButton.Enable()
			} else {
				syncButton.Disable()
			}
		}

		if entryList != nil {
			entryList.Refresh()
		}
	}

	syncButton = widget.NewButtonWithIcon(
		"Sync",
		theme.ViewRefreshIcon(),
		func() {
			syncButton.Disable()

			_, err := vaultService.Remote.Sync()
			if err != nil {
				syncButton.Enable()
				dialog.ShowError(err, window)
				return
			}

			reload()
		},
	)

	if !needSync {
		syncButton.Disable()
	}

	entryList = widget.NewList(
		func() int {
			return len(vaultAccesses)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			updatedAt := widget.NewLabel("")

			updatedAt.TextStyle = fyne.TextStyle{
				Italic: true,
			}

			updateButton := widget.NewButtonWithIcon(
				"",
				theme.DocumentCreateIcon(),
				nil,
			)

			deleteButton := widget.NewButtonWithIcon(
				"",
				theme.DeleteIcon(),
				nil,
			)

			actions := container.NewHBox(
				updateButton,
				deleteButton,
			)

			nameCell := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				name,
			)

			dateCell := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				updatedAt,
			)

			actionCell := container.NewBorder(
				nil,
				nil,
				nil,
				actions,
				nil,
			)

			return container.NewGridWithColumns(
				3,
				nameCell,
				dateCell,
				actionCell,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(vaultAccesses) {
				return
			}

			vaultAccess := vaultAccesses[id]

			row := obj.(*fyne.Container)

			nameCell := row.Objects[0].(*fyne.Container)
			dateCell := row.Objects[1].(*fyne.Container)
			actionCell := row.Objects[2].(*fyne.Container)

			name := nameCell.Objects[0].(*widget.Label)
			updatedAt := dateCell.Objects[0].(*widget.Label)
			actions := actionCell.Objects[0].(*fyne.Container)

			updateButton := actions.Objects[0].(*widget.Button)
			deleteButton := actions.Objects[1].(*widget.Button)

			name.SetText(vaultAccess.Vault.Name)

			updatedAt.SetText(
				vaultAccess.Vault.UpdatedAt.Format(
					"02 Jan 2006 15:04",
				),
			)

			updateButton.OnTapped = func() {
				selectedVault := vaultAccess.Vault

				showForm(
					window,
					&selectedVault,
					func(updated *entity.Vault) {
						err := vaultService.UpdateVault(updated)
						if err != nil {
							dialog.ShowError(err, window)
							return
						}

						reload()
					},
				)
			}

			deleteButton.OnTapped = func() {
				dialog.ShowConfirm(
					"Delete Vault",
					"Vault \""+vaultAccess.Vault.Name+"\" akan dihapus. Lanjutkan?",
					func(ok bool) {
						if !ok {
							return
						}

						err := vaultService.DeleteVault(
							&vaultAccess.Vault,
						)
						if err != nil {
							dialog.ShowError(err, window)
							return
						}

						reload()
					},
					window,
				)
			}
		},
	)

	newEntryButton := widget.NewButtonWithIcon(
		"New Entry",
		theme.ContentAddIcon(),
		func() {
			showForm(
				window,
				nil,
				func(newVault *entity.Vault) {
					_, err := vaultService.CreateVault(
						newVault.Name,
					)
					if err != nil {
						dialog.ShowError(err, window)
						return
					}

					reload()
				},
			)
		},
	)

	// Content utama yang akan direplace ketika user masuk
	// ke Vault Records.
	content := container.NewMax()

	vaultContent := container.NewBorder(
		container.NewBorder(
			search,
			nil,
			nil,
			container.NewHBox(
				syncButton,
				newEntryButton,
			),
			widget.NewLabelWithStyle(
				"All Items",
				fyne.TextAlignLeading,
				fyne.TextStyle{
					Bold: true,
				},
			),
		),
		nil,
		nil,
		nil,
		entryList,
	)

	content.Objects = []fyne.CanvasObject{
		vaultContent,
	}

	entryList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(vaultAccesses) {
			return
		}

		selectedVaultAccess := vaultAccesses[id]

		recordScreen := vaultrecord.New(
			window,
			appDeps,
			context,
			masterKey,
			coreService,
			sotService,
			&selectedVaultAccess,
			func() {
				content.Objects = []fyne.CanvasObject{
					vaultContent,
				}
				content.Refresh()
			},
		)

		content.Objects = []fyne.CanvasObject{
			recordScreen.Content,
		}
		content.Refresh()

		entryList.Unselect(id)
	}

	return &Screen{
		Content: content,
	}
}

func showForm(
	window fyne.Window,
	selectedVault *entity.Vault,
	onSave func(vault *entity.Vault),
) {
	isEdit := selectedVault != nil

	var vault entity.Vault

	if isEdit {
		vault = *selectedVault
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetText(vault.Name)

	form := widget.NewForm(
		widget.NewFormItem(
			"Name",
			nameEntry,
		),
	)

	formContainer := container.NewGridWrap(
		fyne.NewSize(650, 420),
		form,
	)

	title := "New Vault"

	if isEdit {
		title = "Edit Vault"
	}

	dialog.ShowCustomConfirm(
		title,
		"Save",
		"Cancel",
		formContainer,
		func(ok bool) {
			if !ok {
				return
			}

			name := nameEntry.Text

			if name == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"Name wajib diisi.",
					window,
				)
				return
			}

			vault.Name = name

			onSave(&vault)
		},
		window,
	)
}
