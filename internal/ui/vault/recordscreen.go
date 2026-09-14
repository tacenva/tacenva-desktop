package vault

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/database"
	"github.com/tacenva/tacenva-services/app"
	"github.com/tacenva/tacenva-services/app/sourceoftruth"
	"github.com/tacenva/tacenva-services/app/vault"
	coreApp "github.com/tacenva/tacpass-core/app"
	"github.com/tacenva/tacpass-core/config"
	"github.com/tacenva/tacpass-core/entity"
	"github.com/tacenva/tacpass-core/util/credential"
)

const hiddenPassword = "••••••••"

type RecordScreen struct {
	Content fyne.CanvasObject
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

	var searchTimer *time.Timer

	search := widget.NewEntry()
	search.SetPlaceHolder("Search passwords...")

	allVaultRecords, needSync, err := vaultService.ListRecords(
		vaultAccess,
	)
	if err != nil {
		dialog.ShowError(err, window)
		allVaultRecords = nil
	}

	filteredVaultRecords := append(
		[]entity.VaultRecord(nil),
		allVaultRecords...,
	)

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

		allVaultRecords = updatedVaultRecords

		query := strings.ToLower(
			strings.TrimSpace(search.Text),
		)

		filteredVaultRecords = filteredVaultRecords[:0]

		for _, record := range allVaultRecords {
			name := strings.ToLower(record.Name)
			endpoint := strings.ToLower(record.Endpoint)

			if strings.Contains(name, query) ||
				strings.Contains(endpoint, query) {
				filteredVaultRecords = append(
					filteredVaultRecords,
					record,
				)
			}
		}

		if syncButton != nil {
			if newNeedSync {
				syncButton.Enable()
			} else {
				syncButton.Disable()
			}
		}

		if entryList != nil {
			entryList.UnselectAll()
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
				allVaultRecords,
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
			return len(filteredVaultRecords)
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

			copyButton := widget.NewButtonWithIcon(
				"",
				theme.ContentCopyIcon(),
				nil,
			)

			passwordCell := container.NewBorder(
				nil,
				nil,
				nil,
				copyButton,
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
			if id < 0 || id >= len(filteredVaultRecords) {
				return
			}

			record := filteredVaultRecords[id]

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
			copyButton := passwordCell.Objects[1].(*widget.Button)
			expiredAt := expiredAtCell.Objects[0].(*widget.Label)

			actions := actionCell.Objects[0].(*fyne.Container)

			updateButton := actions.Objects[0].(*widget.Button)
			deleteButton := actions.Objects[1].(*widget.Button)

			name.SetText(record.Name)
			endpoint.SetText(record.Endpoint)

			// Row bisa di-recycle oleh Fyne.
			password.SetText(hiddenPassword)

			copyButton.OnTapped = func() {
				window.Clipboard().SetContent(record.Password)

				dialog.ShowInformation(
					"Password Copied",
					"Password copied to clipboard.",
					window,
				)
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
					"Record \""+
						record.Name+
						"\" will be deleted. Continue?",
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
		"New",
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
		search,
	)

	content := container.NewBorder(
		toolbar,
		nil,
		nil,
		nil,
		entryList,
	)

	search.OnChanged = func(query string) {
		if searchTimer != nil {
			searchTimer.Stop()
		}

		searchTimer = time.AfterFunc(
			200*time.Millisecond,
			func() {
				query = strings.ToLower(
					strings.TrimSpace(query),
				)

				filtered := make(
					[]entity.VaultRecord,
					0,
					len(allVaultRecords),
				)

				for _, record := range allVaultRecords {
					name := strings.ToLower(record.Name)
					endpoint := strings.ToLower(record.Endpoint)

					if strings.Contains(name, query) ||
						strings.Contains(endpoint, query) {
						filtered = append(
							filtered,
							record,
						)
					}
				}

				fyne.Do(func() {
					filteredVaultRecords = filtered

					entryList.UnselectAll()
					entryList.Refresh()
				})
			},
		)
	}

	return &RecordScreen{
		Content: content,
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
					"Name, endpoint, and password are required.",
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
