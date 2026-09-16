package accesscontrol

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
	"github.com/tacenva/tacenva-services/app/accesscontrol"
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
	coreService *coreApp.Services,
) *Screen {
	acService := accesscontrol.NewService(
		appDeps,
		context,
		coreService.AccessControl,
		coreService.Auth,
	)

	var searchTimer *time.Timer

	search := widget.NewEntry()
	search.SetPlaceHolder("Search permissions...")

	allACL := make([]entity.Permission, 0)
	filteredACL := make([]entity.Permission, 0)

	var entryList *widget.List

	content := container.NewMax()

	loadingView := loading.New(
		"Loading access controls...",
	)

	filterACL := func(
		permissions []entity.Permission,
		query string,
	) []entity.Permission {
		query = strings.ToLower(
			strings.TrimSpace(query),
		)

		if query == "" {
			return append(
				[]entity.Permission(nil),
				permissions...,
			)
		}

		filtered := make(
			[]entity.Permission,
			0,
			len(permissions),
		)

		for _, permission := range permissions {
			name := strings.ToLower(
				permission.Name,
			)

			privilege := strings.ToLower(
				string(permission.Privilege),
			)

			if strings.Contains(name, query) ||
				strings.Contains(privilege, query) {
				filtered = append(
					filtered,
					permission,
				)
			}
		}

		return filtered
	}

	reload := func() {
		go func() {
			updatedACL, err := acService.List()

			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(
						err,
						window,
					)

					return
				}

				allACL = updatedACL

				filteredACL = filterACL(
					allACL,
					search.Text,
				)

				if entryList != nil {
					entryList.UnselectAll()
					entryList.Refresh()
				}
			})
		}()
	}

	entryList = widget.NewList(
		func() int {
			return len(filteredACL)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			privilege := widget.NewLabel("")

			editButton := widget.NewButtonWithIcon(
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
				editButton,
				deleteButton,
			)

			nameCell := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				name,
			)

			privilegeCell := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				privilege,
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
				privilegeCell,
				actionCell,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(filteredACL) {
				return
			}

			acData := filteredACL[id]

			row := obj.(*fyne.Container)

			nameCell := row.Objects[0].(*fyne.Container)
			privilegeCell := row.Objects[1].(*fyne.Container)
			actionCell := row.Objects[2].(*fyne.Container)

			name := nameCell.Objects[0].(*widget.Label)
			privilege := privilegeCell.Objects[0].(*widget.Label)
			actions := actionCell.Objects[0].(*fyne.Container)

			editButton := actions.Objects[0].(*widget.Button)
			deleteButton := actions.Objects[1].(*widget.Button)

			name.SetText(acData.Name)

			privilege.SetText(
				string(acData.Privilege),
			)

			editButton.OnTapped = func() {
				selected := acData

				showForm(
					window,
					&selected,
					func(updated *entity.Permission) {
						editButton.Disable()
						deleteButton.Disable()

						go func() {
							var err error

							if updated.Name != selected.Name {
								err = acService.ChangeName(
									updated.ID,
									updated.Name,
								)

								if err != nil {
									fyne.Do(func() {
										editButton.Enable()
										deleteButton.Enable()

										dialog.ShowError(
											err,
											window,
										)
									})

									return
								}
							}

							if updated.Privilege != selected.Privilege {
								err = acService.ChangePrivilege(
									updated.ID,
									updated.Privilege,
								)

								if err != nil {
									fyne.Do(func() {
										editButton.Enable()
										deleteButton.Enable()

										dialog.ShowError(
											err,
											window,
										)
									})

									return
								}
							}

							fyne.Do(func() {
								editButton.Enable()
								deleteButton.Enable()

								dialog.ShowInformation(
									"Success",
									"Access control berhasil diperbarui.",
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
					"Delete Access Control",
					"Access control \""+
						acData.Name+
						"\" will be deleted. Continue?",
					func(ok bool) {
						if !ok {
							return
						}

						editButton.Disable()
						deleteButton.Disable()

						go func() {
							err := acService.DeleteAccessControl(
								acData.ID,
							)

							fyne.Do(func() {
								editButton.Enable()
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
									"Access control berhasil dihapus.",
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
				func(newPermission *entity.Permission) {
					newEntryButton.Disable()

					go func() {
						_, _, keyPair, err := acService.Create(
							newPermission.Name,
							newPermission.Privilege,
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
								"Access control berhasil dibuat.",
								window,
							)

							reload()

							if keyPair == nil {
								return
							}

							showKeyPair(
								window,
								keyPair.PublicKey,
								keyPair.PrivateKey,
							)
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
		newEntryButton,
		search,
	)

	acContent := container.NewBorder(
		toolbar,
		nil,
		nil,
		nil,
		entryList,
	)

	content.Objects = []fyne.CanvasObject{
		loadingView.Content,
	}

	entryList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(filteredACL) {
			return
		}

		selectedPermission := filteredACL[id]

		userScreen := NewUser(
			window,
			appDeps,
			context,
			coreService,
			acService,
			&selectedPermission,
			func() {
				content.Objects = []fyne.CanvasObject{
					acContent,
				}

				content.Refresh()
			},
		)

		content.Objects = []fyne.CanvasObject{
			userScreen.Content,
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
				filtered := filterACL(
					allACL,
					query,
				)

				fyne.Do(func() {
					filteredACL = filtered

					entryList.UnselectAll()
					entryList.Refresh()
				})
			},
		)
	}

	// Tampilkan loading terlebih dahulu.
	content.Objects = []fyne.CanvasObject{
		loadingView.Content,
	}

	// Jalankan List() di background.
	go func() {
		updatedACL, err := acService.List()

		fyne.Do(func() {
			if err != nil {
				loadingView.ShowError(
					"Failed to load access controls.",
				)

				dialog.ShowError(
					err,
					window,
				)

				return
			}

			allACL = updatedACL

			filteredACL = filterACL(
				allACL,
				search.Text,
			)

			entryList.Refresh()

			loadingView.Show(
				acContent,
			)
		})
	}()

	return &Screen{
		Content: content,
	}
}

func showForm(
	window fyne.Window,
	selectedPermission *entity.Permission,
	onSave func(permission *entity.Permission),
) {
	isEdit := selectedPermission != nil

	var permission entity.Permission

	if isEdit {
		permission = *selectedPermission
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetText(permission.Name)

	privilegeRadio := widget.NewRadioGroup(
		[]string{
			string(entity.PrivilegeAdmin),
			string(entity.PrivilegeWrite),
			string(entity.PrivilegeRead),
		},
		nil,
	)

	privilegeRadio.Horizontal = false

	if permission.Privilege.IsValid() {
		privilegeRadio.SetSelected(
			string(permission.Privilege),
		)
	} else {
		privilegeRadio.SetSelected(
			string(entity.PrivilegeRead),
		)
	}

	form := widget.NewForm(
		widget.NewFormItem(
			"Name",
			nameEntry,
		),
		widget.NewFormItem(
			"Privilege",
			privilegeRadio,
		),
	)

	formContainer := container.NewGridWrap(
		fyne.NewSize(650, 420),
		form,
	)

	title := "New Access Control"

	if isEdit {
		title = "Edit Access Control"
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
					"Name is required.",
					window,
				)

				return
			}

			if privilegeRadio.Selected == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"Privilege is required.",
					window,
				)

				return
			}

			permission.Name = name
			permission.Privilege = entity.Privilege(
				privilegeRadio.Selected,
			)

			onSave(&permission)
		},
		window,
	)
}

func showKeyPair(
	window fyne.Window,
	publicKey string,
	privateKey string,
) {
	publicKeyEntry := widget.NewMultiLineEntry()
	publicKeyEntry.SetText(publicKey)
	publicKeyEntry.SetMinRowsVisible(6)
	publicKeyEntry.Disable()

	privateKeyEntry := widget.NewMultiLineEntry()
	privateKeyEntry.SetText(privateKey)
	privateKeyEntry.SetMinRowsVisible(8)
	privateKeyEntry.Disable()

	content := container.NewVBox(
		widget.NewLabelWithStyle(
			"Public Key",
			fyne.TextAlignLeading,
			fyne.TextStyle{
				Bold: true,
			},
		),
		publicKeyEntry,

		widget.NewLabelWithStyle(
			"Private Key",
			fyne.TextAlignLeading,
			fyne.TextStyle{
				Bold: true,
			},
		),
		privateKeyEntry,

		widget.NewLabel(
			"The private key is only displayed now. Save it securely before closing this dialog.",
		),
	)

	dialog.ShowCustom(
		"Access Control Key Pair",
		"Close",
		content,
		window,
	)
}