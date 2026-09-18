package vault

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-desktop/internal/ui/loading"
	"github.com/tacenva/tacenva-services/app"
	"github.com/tacenva/tacenva-services/app/sourceoftruth"
	"github.com/tacenva/tacenva-services/app/vault"
	coreApp "github.com/tacenva/tacpass-core/app"
	"github.com/tacenva/tacpass-core/entity"
)

type Screen struct {
	Content fyne.CanvasObject
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
		coreService.VaultRecordService,
		coreService.Auth,
		sotService,
	)

	var searchTimer *time.Timer

	search := widget.NewEntry()
	search.SetPlaceHolder("Search collection...")

	allVaultAccesses := make(
		[]entity.VaultAccess,
		0,
	)

	filteredVaultAccesses := make(
		[]entity.VaultAccess,
		0,
	)

	var entryList *widget.List
	var syncButton *widget.Button

	content := container.NewMax()

	loadingView := loading.New(
		"Loading collections...",
	)

	filterVaultAccesses := func(
		accesses []entity.VaultAccess,
		query string,
	) []entity.VaultAccess {
		query = strings.ToLower(
			strings.TrimSpace(query),
		)

		if query == "" {
			return append(
				[]entity.VaultAccess(nil),
				accesses...,
			)
		}

		filtered := make(
			[]entity.VaultAccess,
			0,
			len(accesses),
		)

		for _, vaultAccess := range accesses {
			if strings.Contains(
				strings.ToLower(
					vaultAccess.Vault.Name,
				),
				query,
			) {
				filtered = append(
					filtered,
					vaultAccess,
				)
			}
		}

		return filtered
	}

	applyVaultAccesses := func(
		vaultAccesses []entity.VaultAccess,
		needSync bool,
	) {
		allVaultAccesses = vaultAccesses

		filteredVaultAccesses = filterVaultAccesses(
			allVaultAccesses,
			search.Text,
		)

		if needSync {
			syncButton.Enable()
		} else {
			syncButton.Disable()
		}

		entryList.UnselectAll()
		entryList.Refresh()
	}

	reload := func() {
		syncButton.Disable()

		go func() {
			updatedVaultAccesses, needSync, err :=
				vaultService.List()

			fyne.Do(func() {
				if err != nil {
					if updatedVaultAccesses == nil {
						dialog.ShowError(
							err,
							window,
						)

						syncButton.Enable()

						return
					}

					applyVaultAccesses(
						updatedVaultAccesses,
						needSync,
					)

					dialog.ShowError(
						err,
						window,
					)

					return
				}

				applyVaultAccesses(
					updatedVaultAccesses,
					needSync,
				)
			})
		}()
	}

	syncButton = widget.NewButtonWithIcon(
		"Sync",
		theme.ViewRefreshIcon(),
		func() {
			syncButton.Disable()

			go func() {
				_, err := vaultService.Sync()

				fyne.Do(func() {
					if err != nil {
						syncButton.Enable()

						dialog.ShowError(
							err,
							window,
						)

						return
					}

					reload()
				})
			}()
		},
	)

	entryList = widget.NewList(
		func() int {
			return len(filteredVaultAccesses)
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
		func(
			id widget.ListItemID,
			obj fyne.CanvasObject,
		) {
			if id < 0 ||
				id >= len(filteredVaultAccesses) {
				return
			}

			vaultAccess := filteredVaultAccesses[id]

			row := obj.(*fyne.Container)

			nameCell := row.Objects[0].(*fyne.Container)
			dateCell := row.Objects[1].(*fyne.Container)
			actionCell := row.Objects[2].(*fyne.Container)

			name := nameCell.Objects[0].(*widget.Label)
			updatedAt := dateCell.Objects[0].(*widget.Label)
			actions := actionCell.Objects[0].(*fyne.Container)

			updateButton := actions.Objects[0].(*widget.Button)
			deleteButton := actions.Objects[1].(*widget.Button)

			name.SetText(
				vaultAccess.Vault.Name,
			)

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
						updateButton.Disable()
						deleteButton.Disable()

						go func() {
							err := vaultService.UpdateVault(
								updated,
							)

							fyne.Do(func() {
								updateButton.Enable()
								deleteButton.Enable()

								if err != nil {
									dialog.ShowError(
										err,
										window,
									)

									return
								}

								dialog.ShowInformation(
									"Success",
									"Vault berhasil diperbarui.",
									window,
								)

								reload()
							})
						}()
					},
				)
			}

			deleteButton.OnTapped = func() {
				dialog.ShowConfirm(
					"Delete Vault",
					"Vault \""+
						vaultAccess.Vault.Name+
						"\" will be deleted. Continue?",
					func(ok bool) {
						if !ok {
							return
						}

						updateButton.Disable()
						deleteButton.Disable()

						go func() {
							err := vaultService.DeleteVault(
								&vaultAccess.Vault,
							)

							fyne.Do(func() {
								updateButton.Enable()
								deleteButton.Enable()

								if err != nil {
									dialog.ShowError(
										err,
										window,
									)

									return
								}

								dialog.ShowInformation(
									"Success",
									"Vault berhasil dihapus.",
									window,
								)

								reload()
							})
						}()
					},
					window,
				)
			}
		},
	)

	var newEntryButton *widget.Button

	newEntryButton = widget.NewButtonWithIcon(
		"New",
		theme.ContentAddIcon(),
		func() {
			showForm(
				window,
				nil,
				func(newVault *entity.Vault) {
					newEntryButton.Disable()

					go func() {
						_, err := vaultService.CreateVault(
							newVault.Name,
						)

						fyne.Do(func() {
							newEntryButton.Enable()

							if err != nil {
								dialog.ShowError(
									err,
									window,
								)

								return
							}

							dialog.ShowInformation(
								"Success",
								"Vault berhasil dibuat.",
								window,
							)

							reload()
						})
					}()
				},
			)
		},
	)

	toolbar := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(
			syncButton,
			newEntryButton,
		),
		search,
	)

	vaultContent := container.NewBorder(
		toolbar,
		nil,
		nil,
		nil,
		entryList,
	)

	entryList.OnSelected = func(
		id widget.ListItemID,
	) {
		if id < 0 ||
			id >= len(filteredVaultAccesses) {
			return
		}

		selectedVaultAccess := filteredVaultAccesses[id]

		recordScreen := NewRecordScreen(
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

	search.OnChanged = func(query string) {
		if searchTimer != nil {
			searchTimer.Stop()
		}

		searchTimer = time.AfterFunc(
			200*time.Millisecond,
			func() {
				filtered := filterVaultAccesses(
					allVaultAccesses,
					query,
				)

				fyne.Do(func() {
					filteredVaultAccesses = filtered

					entryList.UnselectAll()
					entryList.Refresh()
				})
			},
		)
	}

	content.Objects = []fyne.CanvasObject{
		loadingView.Content,
	}

	go func() {
		updatedVaultAccesses, needSync, err :=
			vaultService.List()

		fyne.Do(func() {
			if err != nil {
				if updatedVaultAccesses == nil {
					loadingView.ShowError(
						"Failed to load collections.",
					)

					dialog.ShowError(
						err,
						window,
					)

					return
				}

				applyVaultAccesses(
					updatedVaultAccesses,
					needSync,
				)

				loadingView.Show(
					vaultContent,
				)

				dialog.ShowError(
					err,
					window,
				)

				return
			}

			applyVaultAccesses(
				updatedVaultAccesses,
				needSync,
			)

			loadingView.Show(
				vaultContent,
			)
		})
	}()

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

			name := strings.TrimSpace(
				nameEntry.Text,
			)

			if name == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"Name is required.",
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
