package accesscontrol

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/replica-core/app"
	"github.com/tacenva/replica-core/app/accesscontrol"
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

	search := widget.NewEntry()
	search.SetPlaceHolder("Search access controls...")

	acl, err := acService.List()
	if err != nil {
		dialog.ShowError(err, window)
		acl = nil
	}

	var entryList *widget.List

	reload := func() {
		updatedACL, err := acService.List()
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		acl = updatedACL

		if entryList != nil {
			entryList.Refresh()
		}
	}

	entryList = widget.NewList(
		func() int {
			return len(acl)
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

			return container.NewGridWithColumns(
				3,
				name,
				privilege,
				actions,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(acl) {
				return
			}

			acData := acl[id]

			row := obj.(*fyne.Container)

			name := row.Objects[0].(*widget.Label)
			privilege := row.Objects[1].(*widget.Label)
			actions := row.Objects[2].(*fyne.Container)

			editButton := actions.Objects[0].(*widget.Button)
			deleteButton := actions.Objects[1].(*widget.Button)

			name.SetText(acData.Name)
			privilege.SetText(string(acData.Privilege))

			editButton.OnTapped = func() {
				selected := acData

				showForm(
					window,
					&selected,
					func(updated *entity.Permission) {
						if updated.Name != selected.Name {
							if err := acService.ChangeName(
								updated.ID,
								updated.Name,
							); err != nil {
								dialog.ShowError(err, window)
								return
							}
						}
						if updated.Privilege != selected.Privilege {
							if err := acService.ChangePrivilege(
								updated.ID,
								updated.Privilege,
							); err != nil {
								dialog.ShowError(err, window)
								return
							}
						}

						reload()
					},
				)
			}

			deleteButton.OnTapped = func() {
				dialog.ShowConfirm(
					"Delete Access Control",
					"Access control \""+acData.Name+"\" akan dihapus. Lanjutkan?",
					func(ok bool) {
						if !ok {
							return
						}

						if err := acService.DeleteAccessControl(
							acData.ID,
						); err != nil {
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
				func(newPermission *entity.Permission) {
					_, _, keyPair, err := acService.Create(
						newPermission.Name,
						newPermission.Privilege,
					)
					if err != nil {
						dialog.ShowError(err, window)
						return
					}

					reload()

					if keyPair == nil {
						return
					}

					showKeyPair(
						window,
						keyPair.PublicKey,
						keyPair.PrivateKey,
					)
				},
			)
		},
	)

	content := container.NewMax()

	acContent := container.NewBorder(
		container.NewBorder(
			search,
			nil,
			nil,
			container.NewHBox(
				newEntryButton,
			),
			widget.NewLabelWithStyle(
				"Access Control",
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
		acContent,
	}

	entryList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(acl) {
			return
		}

		selectedPermission := acl[id]

		userScreen := NewUser(
			window,
			appDeps,
			context,
			coreService,
			acService,
			&selectedPermission,
		)

		content.Objects = []fyne.CanvasObject{
			userScreen.Content,
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
		privilegeRadio.SetSelected(string(permission.Privilege))
	} else {
		privilegeRadio.SetSelected(string(entity.PrivilegeRead))
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
					"Name wajib diisi.",
					window,
				)
				return
			}

			if privilegeRadio.Selected == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"Privilege wajib dipilih.",
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
			"Private Key hanya ditampilkan sekarang. Simpan dengan aman sebelum menutup dialog.",
		),
	)

	dialog.ShowCustom(
		"Access Control Key Pair",
		"Close",
		content,
		window,
	)
}
