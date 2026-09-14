package vault

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/database"
	"github.com/tacenva/replica-core/app"
	"github.com/tacenva/replica-core/app/sourceoftruth"
	"github.com/tacenva/replica-core/app/vault"
	coreApp "github.com/tacenva/tacpass-core/app"
	"github.com/tacenva/tacpass-core/config"
	"github.com/tacenva/tacpass-core/entity"
	"github.com/tacenva/tacpass-core/util/credential"
)

const hiddenPassword = "••••••••"

type RecordScreen struct {
	Content      fyne.CanvasObject
	vaultRecords []entity.VaultRecord
}

func NewRecordScreen(
	window fyne.Window,
	appDeps *app.Deps,
	context *app.Context,
	masterKey string,
	coreService *coreApp.Services,
	sotService *sourceoftruth.Service,
	vaultAccess *entity.VaultAccess,
	onBack func(),
) *RecordScreen {
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

	vaultRecords, needSync, err := vaultService.ListRecords(vaultAccess)
	if err != nil {
		dialog.ShowError(err, window)
		vaultRecords = nil
	}

	var entryList *widget.List
	var syncButton *widget.Button

	reload := func() {
		updatedVaultRecords, newNeedSync, err := vaultService.ListRecords(
			vaultAccess,
		)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		vaultRecords = updatedVaultRecords

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

	backButton := widget.NewButtonWithIcon(
		vaultAccess.Vault.Name,
		theme.NavigateBackIcon(),
		onBack,
	)

	syncButton = widget.NewButtonWithIcon(
		"Sync",
		theme.ViewRefreshIcon(),
		func() {
			syncButton.Disable()

			vaultDB := database.New(
				appDeps.Config.Path(
					config.NodeDirName,
					context.SelectedSoT.ID,
					"vault",
				),
			)

			vaultKey, err := context.SelectedSoT.KeyPair.Open(
				vaultAccess.VaultKey,
			)
			if err != nil {
				syncButton.Enable()
				dialog.ShowError(err, window)
				return
			}

			vaultFile, err := vaultDB.File(
				vaultAccess.VaultID,
				string(vaultKey),
				database.FileModeOpenOrCreate,
			)
			if err != nil {
				syncButton.Enable()
				dialog.ShowError(err, window)
				return
			}

			_, _, err = vaultService.Remote.SyncRecords(
				vaultAccess.VaultID,
				vaultFile,
				vaultRecords,
			)
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
			return len(vaultRecords)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			endpoint := widget.NewLabel("")
			password := widget.NewLabel(hiddenPassword)
			expiredAt := widget.NewLabel("")

			name.TextStyle = fyne.TextStyle{
				Bold: true,
			}

			expiredAt.TextStyle = fyne.TextStyle{
				Italic: true,
			}

			passwordButton := widget.NewButtonWithIcon(
				"",
				theme.VisibilityIcon(),
				nil,
			)

			passwordCell := container.NewBorder(
				nil,
				nil,
				nil,
				passwordButton,
				password,
			)

			nameCell := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				name,
			)

			endpointCell := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				endpoint,
			)

			expiredAtCell := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				expiredAt,
			)

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

			info := container.NewGridWithColumns(
				4,
				nameCell,
				endpointCell,
				passwordCell,
				expiredAtCell,
			)

			actionCell := container.NewBorder(
				nil,
				nil,
				nil,
				actions,
				nil,
			)

			return container.NewBorder(
				nil,
				nil,
				nil,
				actionCell,
				info,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(vaultRecords) {
				return
			}

			record := vaultRecords[id]

			row := obj.(*fyne.Container)

			info := row.Objects[0].(*fyne.Container)
			actionCell := row.Objects[1].(*fyne.Container)

			nameCell := info.Objects[0].(*fyne.Container)
			endpointCell := info.Objects[1].(*fyne.Container)
			passwordCell := info.Objects[2].(*fyne.Container)
			expiredAtCell := info.Objects[3].(*fyne.Container)

			name := nameCell.Objects[0].(*widget.Label)
			endpoint := endpointCell.Objects[0].(*widget.Label)
			password := passwordCell.Objects[0].(*widget.Label)
			passwordButton := passwordCell.Objects[1].(*widget.Button)
			expiredAt := expiredAtCell.Objects[0].(*widget.Label)

			actions := actionCell.Objects[0].(*fyne.Container)

			updateButton := actions.Objects[0].(*widget.Button)
			deleteButton := actions.Objects[1].(*widget.Button)

			name.SetText(record.Name)
			endpoint.SetText(record.Endpoint)

			// Setiap kali row di-recycle, password kembali hidden.
			password.SetText(hiddenPassword)

			// Password asli hanya disimpan di closure tombol ini.
			actualPassword := record.Password
			showPassword := false

			passwordButton.SetIcon(
				theme.VisibilityIcon(),
			)

			passwordButton.OnTapped = func() {
				showPassword = !showPassword

				if showPassword {
					password.SetText(actualPassword)
					passwordButton.SetIcon(
						theme.VisibilityOffIcon(),
					)
				} else {
					password.SetText(hiddenPassword)
					passwordButton.SetIcon(
						theme.VisibilityIcon(),
					)
				}

				password.Refresh()
				passwordButton.Refresh()
			}

			if record.ExpiredAt.IsZero() {
				expiredAt.SetText("-")
			} else {
				expiredAt.SetText(
					record.ExpiredAt.Format("02 Jan 2006"),
				)
			}

			updateButton.OnTapped = func() {
				selectedRecord := record

				showRecordForm(
					window,
					&selectedRecord,
					func(updated *entity.VaultRecord) {
						_, err := vaultService.UpdateRecord(
							vaultAccess,
							updated,
						)
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
					"Delete Record",
					"Record \""+record.Name+"\" akan dihapus. Lanjutkan?",
					func(ok bool) {
						if !ok {
							return
						}

						err := vaultService.DeleteRecord(
							vaultAccess,
							&record,
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
			showRecordForm(
				window,
				nil,
				func(newRecord *entity.VaultRecord) {
					_, err := vaultService.AppendRecord(
						vaultAccess,
						newRecord,
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

	toolbar := container.NewBorder(
		nil,
		nil,
		backButton,
		container.NewHBox(
			syncButton,
			newEntryButton,
		),
		nil,
	)

	entryHeader := container.NewVBox(
		toolbar,
		search,
	)

	content := container.NewBorder(
		entryHeader,
		nil,
		nil,
		nil,
		entryList,
	)

	return &RecordScreen{
		Content:      content,
		vaultRecords: vaultRecords,
	}
}

func showRecordForm(
	window fyne.Window,
	selectedRecord *entity.VaultRecord,
	onSave func(record *entity.VaultRecord),
) {
	isEdit := selectedRecord != nil

	var record entity.VaultRecord

	if isEdit {
		record = *selectedRecord
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetText(record.Name)

	endpointEntry := widget.NewEntry()
	endpointEntry.SetText(record.Endpoint)

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetText(record.Password)

	generatePasswordButton := widget.NewButtonWithIcon(
		"Generate",
		theme.ViewRefreshIcon(),
		func() {
			password, err := credential.Generate(27)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			passwordEntry.SetText(password)
		},
	)

	passwordContainer := container.NewBorder(
		nil,
		nil,
		nil,
		generatePasswordButton,
		passwordEntry,
	)

	expiredAtEntry := widget.NewDateEntry()

	if !record.ExpiredAt.IsZero() {
		expiredAt := record.ExpiredAt
		expiredAtEntry.SetDate(&expiredAt)
	}

	form := widget.NewForm(
		widget.NewFormItem(
			"Name",
			nameEntry,
		),
		widget.NewFormItem(
			"Endpoint",
			endpointEntry,
		),
		widget.NewFormItem(
			"Password",
			passwordContainer,
		),
		widget.NewFormItem(
			"Expired At",
			expiredAtEntry,
		),
	)

	formContainer := container.NewGridWrap(
		fyne.NewSize(650, 420),
		form,
	)

	title := "New Record"

	if isEdit {
		title = "Edit Record"
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
			endpoint := endpointEntry.Text
			password := passwordEntry.Text

			if name == "" || endpoint == "" || password == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"Name, endpoint, dan password wajib diisi.",
					window,
				)
				return
			}

			record.Name = name
			record.Endpoint = endpoint
			record.Password = password

			if expiredAtEntry.Date == nil {
				record.ExpiredAt = time.Time{}
			} else {
				record.ExpiredAt = *expiredAtEntry.Date
			}

			onSave(&record)
		},
		window,
	)
}
