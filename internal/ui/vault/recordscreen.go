package vault

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-core/util/credential"
	"github.com/tacenva/tacenva-desktop/internal/ui/loading"
	"github.com/tacenva/tacenva-services/app"
	"github.com/tacenva/tacenva-services/app/sourceoftruth"
	vaultServicePackage "github.com/tacenva/tacenva-services/app/vault"
	coreApp "github.com/tacenva/tacpass-core/app"
	tacpassentity "github.com/tacenva/tacpass-core/entity"
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
	selectedVault *tacpassentity.Vault,
	wrappedKey string,
	onBack func(),

	vaultService *vaultServicePackage.Service,
	isWriter bool,
) *RecordScreen {
	var searchTimer *time.Timer

	search := widget.NewEntry()
	search.SetPlaceHolder("Search passwords...")

	allVaultRecords := make([]tacpassentity.VaultRecord, 0)
	filteredVaultRecords := make([]tacpassentity.VaultRecord, 0)

	var entryList *widget.List
	var syncButton *widget.Button
	var newEntryButton *widget.Button
	var recordContent fyne.CanvasObject

	var stateMu sync.Mutex
	busy := false
	listLoading := true
	needSyncState := false

	content := container.NewMax()
	loadingView := loading.New("Loading passwords...")

	isBusy := func() bool {
		stateMu.Lock()
		defer stateMu.Unlock()
		return busy
	}

	isListLoading := func() bool {
		stateMu.Lock()
		defer stateMu.Unlock()
		return listLoading
	}

	setBusy := func(value bool) {
		stateMu.Lock()
		busy = value
		stateMu.Unlock()
	}

	setListLoading := func(value bool) {
		stateMu.Lock()
		listLoading = value
		stateMu.Unlock()
	}

	filterVaultRecords := func(
		records []tacpassentity.VaultRecord,
		query string,
	) []tacpassentity.VaultRecord {
		query = strings.ToLower(strings.TrimSpace(query))

		if query == "" {
			return append([]tacpassentity.VaultRecord(nil), records...)
		}

		filtered := make([]tacpassentity.VaultRecord, 0, len(records))

		for _, record := range records {
			identifier := strings.ToLower(record.Identifier)
			endpoint := strings.ToLower(record.Endpoint)

			if strings.Contains(identifier, query) ||
				strings.Contains(endpoint, query) {
				filtered = append(filtered, record)
			}
		}

		return filtered
	}

	updateActionStates := func() {
		working := isBusy() || isListLoading()

		if syncButton != nil {
			if !working && needSyncState {
				syncButton.Enable()
			} else {
				syncButton.Disable()
			}
		}

		if newEntryButton != nil {
			if !working && isWriter {
				newEntryButton.Enable()
			} else {
				newEntryButton.Disable()
			}
		}

		if entryList != nil {
			entryList.Refresh()
		}
	}

	applyRecords := func(
		records []tacpassentity.VaultRecord,
		needSync bool,
	) {
		allVaultRecords = append([]tacpassentity.VaultRecord(nil), records...)
		filteredVaultRecords = filterVaultRecords(
			allVaultRecords,
			search.Text,
		)
		needSyncState = needSync

		entryList.UnselectAll()
		updateActionStates()
	}

	var reload func()

	reload = func() {
		stateMu.Lock()
		if busy || listLoading {
			stateMu.Unlock()
			return
		}
		listLoading = true
		stateMu.Unlock()

		updateActionStates()

		go func() {
			var (
				records  []tacpassentity.VaultRecord
				needSync bool
				err      error
			)
			err = vaultService.SelectVault(
				selectedVault,
				wrappedKey,
			)
			if err == nil {
				records, needSync, err = vaultService.ListRecord(
					selectedVault,
				)
			}

			fyne.Do(func() {
				setListLoading(false)

				if err != nil {
					if records == nil {
						if len(allVaultRecords) == 0 {
							loadingView.ShowError(
								"Failed to load passwords.",
							)
						}
						updateActionStates()
						dialog.ShowError(err, window)
						return
					}

					applyRecords(records, needSync)
					loadingView.Show(recordContent)
					dialog.ShowError(err, window)
					return
				}

				applyRecords(records, needSync)
				loadingView.Show(recordContent)
			})
		}()
	}

	runOperation := func(
		message string,
		operation func() error,
		onSuccess func(),
	) {
		stateMu.Lock()
		if busy || listLoading {
			stateMu.Unlock()
			return
		}
		busy = true
		stateMu.Unlock()

		updateActionStates()

		runWithLoading(
			window,
			message,
			operation,
			onSuccess,
			func() {
				setBusy(false)
				updateActionStates()
			},
			nil,
		)
	}

	backButton := widget.NewButtonWithIcon(
		selectedVault.Name,
		theme.NavigateBackIcon(),
		onBack,
	)

	syncButton = widget.NewButtonWithIcon(
		"Sync",
		theme.ViewRefreshIcon(),
		func() {
			if isBusy() || isListLoading() {
				return
			}

			runOperation(
				"Syncing Records...",
				func() error {
					return vaultService.SyncRecord(selectedVault)
				},
				func() {
					dialog.ShowInformation(
						"Success",
						"Records berhasil disinkronkan.",
						window,
					)
					reload()
				},
			)
		},
	)

	if isWriter {
		newEntryButton = widget.NewButtonWithIcon(
			"New",
			theme.ContentAddIcon(),
			func() {
				if isBusy() || isListLoading() {
					return
				}

				showRecordForm(
					window,
					nil,
					func(newRecord *tacpassentity.VaultRecord) {
						runOperation(
							"Creating Record...",
							func() error {
								_, err := vaultService.AppendRecord(
									selectedVault,
									newRecord,
								)
								return err
							},
							func() {
								dialog.ShowInformation(
									"Success",
									"Record berhasil dibuat.",
									window,
								)
								reload()
							},
						)
					},
				)
			},
		)
	}

	entryList = widget.NewList(
		func() int {
			return len(filteredVaultRecords)
		},
		func() fyne.CanvasObject {
			identifier := widget.NewLabel("")
			endpoint := widget.NewLabel("")
			password := widget.NewLabel(hiddenPassword)
			renewalDate := widget.NewLabel("")

			identifier.TextStyle = fyne.TextStyle{Bold: true}
			renewalDate.TextStyle = fyne.TextStyle{Italic: true}

			copyButton := widget.NewButtonWithIcon(
				"",
				theme.ContentCopyIcon(),
				nil,
			)

			passwordCell := container.NewBorder(
				nil, nil, nil, copyButton, password,
			)

			identifierCell := container.NewBorder(
				nil, nil, nil, nil, identifier,
			)

			endpointCell := container.NewBorder(
				nil, nil, nil, nil, endpoint,
			)

			renewalDateIcon := widget.NewIcon(nil)

			renewalDateCell := container.NewBorder(
				nil, nil, nil, nil,
				container.NewHBox(
					renewalDateIcon,
					renewalDate,
				),
			)

			info := container.NewGridWithColumns(
				4,
				identifierCell,
				endpointCell,
				passwordCell,
				renewalDateCell,
			)

			if !isWriter {
				return info
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

			actionCell := container.NewBorder(
				nil, nil, nil, actions, nil,
			)

			return container.NewBorder(
				nil, nil, nil, actionCell, info,
			)
		},
		func(
			id widget.ListItemID,
			obj fyne.CanvasObject,
		) {
			if id < 0 || id >= len(filteredVaultRecords) {
				return
			}

			record := filteredVaultRecords[id]

			row, ok := obj.(*fyne.Container)
			if !ok {
				return
			}

			var info *fyne.Container
			var actionCell *fyne.Container

			if isWriter {
				if len(row.Objects) < 2 {
					return
				}

				info, ok = row.Objects[0].(*fyne.Container)
				if !ok {
					return
				}

				actionCell, ok = row.Objects[1].(*fyne.Container)
				if !ok {
					return
				}
			} else {
				info = row
			}

			if len(info.Objects) < 4 {
				return
			}

			identifierCell, ok := info.Objects[0].(*fyne.Container)
			if !ok || len(identifierCell.Objects) == 0 {
				return
			}

			endpointCell, ok := info.Objects[1].(*fyne.Container)
			if !ok || len(endpointCell.Objects) == 0 {
				return
			}

			passwordCell, ok := info.Objects[2].(*fyne.Container)
			if !ok || len(passwordCell.Objects) < 2 {
				return
			}

			identifier, ok := identifierCell.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			endpoint, ok := endpointCell.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			password, ok := passwordCell.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			copyButton, ok := passwordCell.Objects[1].(*widget.Button)
			if !ok {
				return
			}

			renewalDateBorder, ok := info.Objects[3].(*fyne.Container)
			if !ok || len(renewalDateBorder.Objects) == 0 {
				return
			}

			renewalDateBox, ok := renewalDateBorder.Objects[0].(*fyne.Container)
			if !ok || len(renewalDateBox.Objects) < 2 {
				return
			}

			renewalDateIcon, ok := renewalDateBox.Objects[0].(*widget.Icon)
			if !ok {
				return
			}

			renewalDate, ok := renewalDateBox.Objects[1].(*widget.Label)
			if !ok {
				return
			}

			var updateButton *widget.Button
			var deleteButton *widget.Button

			if isWriter {
				if len(actionCell.Objects) == 0 {
					return
				}

				actions, ok := actionCell.Objects[0].(*fyne.Container)
				if !ok || len(actions.Objects) < 2 {
					return
				}

				updateButton, ok = actions.Objects[0].(*widget.Button)
				if !ok {
					return
				}

				deleteButton, ok = actions.Objects[1].(*widget.Button)
				if !ok {
					return
				}
			}

			identifier.SetText(record.Identifier)
			endpoint.SetText(record.Endpoint)
			password.SetText(hiddenPassword)

			copyButton.OnTapped = func() {
				window.Clipboard().SetContent(record.Password)

				dialog.ShowInformation(
					"Password Copied",
					"Password copied to clipboard.",
					window,
				)
			}

			if record.RenewalPeriod <= 0 || record.LastRenewedAt.IsZero() {
				renewalDate.SetText("-")
				renewalDate.Importance = widget.LowImportance
				renewalDateIcon.SetResource(nil)
			} else {
				nextRenewal := record.LastRenewedAt.Add(record.RenewalPeriod)

				renewalDate.SetText(nextRenewal.Format("02 Jan 2006"))

				now := time.Now()
				today := time.Date(
					now.Year(),
					now.Month(),
					now.Day(),
					0, 0, 0, 0,
					time.UTC,
				)

				renewal := nextRenewal.In(now.Location())
				renewalDay := time.Date(
					renewal.Year(),
					renewal.Month(),
					renewal.Day(),
					0, 0, 0, 0,
					time.UTC,
				)

				daysRemaining := int(renewalDay.Sub(today) / (24 * time.Hour))

				switch {
				case daysRemaining < 0:
					renewalDate.Importance = widget.DangerImportance
					renewalDateIcon.SetResource(theme.ErrorIcon())

				case daysRemaining <= 7:
					renewalDate.Importance = widget.WarningImportance
					renewalDateIcon.SetResource(theme.WarningIcon())

				default:
					renewalDate.Importance = widget.MediumImportance
					renewalDateIcon.SetResource(nil)
				}
			}

			renewalDate.Refresh()
			renewalDateIcon.Refresh()

			if !isWriter {
				return
			}

			actionsEnabled := !isBusy() && !isListLoading()

			if actionsEnabled {
				updateButton.Enable()
				deleteButton.Enable()
			} else {
				updateButton.Disable()
				deleteButton.Disable()
			}

			updateButton.OnTapped = func() {
				if isBusy() || isListLoading() {
					return
				}

				selectedRecord := record

				showRecordForm(
					window,
					&selectedRecord,
					func(updated *tacpassentity.VaultRecord) {
						runOperation(
							"Saving Record...",
							func() error {
								_, err := vaultService.UpdateRecord(
									selectedVault,
									updated,
								)
								return err
							},
							func() {
								dialog.ShowInformation(
									"Success",
									"Record berhasil diperbarui.",
									window,
								)
								reload()
							},
						)
					},
				)
			}

			deleteButton.OnTapped = func() {
				if isBusy() || isListLoading() {
					return
				}

				dialog.ShowConfirm(
					"Delete Record",
					"Record \""+record.Identifier+
						"\" will be deleted. Continue?",
					func(ok bool) {
						if !ok {
							return
						}

						if isBusy() || isListLoading() {
							dialog.ShowInformation(
								"Offline",
								"Koneksi tidak tersedia. Penghapusan dibatalkan.",
								window,
							)
							return
						}

						runOperation(
							"Deleting Record...",
							func() error {
								return vaultService.DeleteRecord(
									selectedVault,
									&record,
								)
							},
							func() {
								dialog.ShowInformation(
									"Success",
									"Record berhasil dihapus.",
									window,
								)
								reload()
							},
						)
					},
					window,
				)
			}
		},
	)

	toolbarActions := container.NewHBox(syncButton)

	if isWriter {
		toolbarActions.Add(newEntryButton)
	}

	toolbar := container.NewBorder(
		nil,
		nil,
		backButton,
		toolbarActions,
		search,
	)

	recordContent = container.NewBorder(
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

		recordsSnapshot := append(
			[]tacpassentity.VaultRecord(nil),
			allVaultRecords...,
		)

		searchTimer = time.AfterFunc(
			200*time.Millisecond,
			func() {
				filtered := filterVaultRecords(
					recordsSnapshot,
					query,
				)

				fyne.Do(func() {
					filteredVaultRecords = filtered
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
		var (
			records  []tacpassentity.VaultRecord
			needSync bool
			err      error
		)
		err = vaultService.SelectVault(
			selectedVault,
			wrappedKey,
		)
		if err == nil {
			records, needSync, err = vaultService.ListRecord(
				selectedVault,
			)
		}

		fyne.Do(func() {
			setListLoading(false)

			if err != nil {
				if records == nil {
					loadingView.ShowError(
						"Failed to load passwords.",
					)
					updateActionStates()
					dialog.ShowError(err, window)
					return
				}

				applyRecords(records, needSync)
				loadingView.Show(recordContent)
				dialog.ShowError(err, window)
				return
			}

			applyRecords(records, needSync)
			loadingView.Show(recordContent)
		})
	}()

	return &RecordScreen{
		Content: content,
	}
}

func showRecordForm(
	window fyne.Window,
	selectedRecord *tacpassentity.VaultRecord,
	onSave func(record *tacpassentity.VaultRecord),
) {
	isEdit := selectedRecord != nil

	var record tacpassentity.VaultRecord

	if isEdit {
		record = *selectedRecord
	}

	identifierEntry := widget.NewEntry()
	identifierEntry.SetPlaceHolder("e.g. john.doe@gmail.com, @johndoe, postgres_user")
	identifierEntry.SetText(record.Identifier)

	endpointEntry := widget.NewEntry()
	endpointEntry.SetPlaceHolder("e.g. instagram.com, db.example.com, postgresql://localhost:5432")
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
		nil, nil, nil, generatePasswordButton, passwordEntry,
	)

	renewalPeriodEntry := widget.NewEntry()
	renewalPeriodEntry.SetPlaceHolder("Leave empty to disable")

	renewalPeriodUnit := widget.NewLabel("days")
	renewalPeriodUnit.TextStyle = fyne.TextStyle{
		Italic: true,
	}

	renewalPeriodContainer := container.NewBorder(
		nil,
		nil,
		nil,
		renewalPeriodUnit,
		renewalPeriodEntry,
	)

	if record.RenewalPeriod > 0 {
		days := int(record.RenewalPeriod / (24 * time.Hour))
		renewalPeriodEntry.SetText(strconv.Itoa(days))
	}

	form := widget.NewForm(
		widget.NewFormItem("Identifier", identifierEntry),
		widget.NewFormItem("Endpoint", endpointEntry),
		widget.NewFormItem("Password", passwordContainer),
		widget.NewFormItem("Renewal Period (days)", renewalPeriodContainer),
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

			identifier := strings.TrimSpace(identifierEntry.Text)
			endpoint := strings.TrimSpace(endpointEntry.Text)
			password := passwordEntry.Text

			if identifier == "" || endpoint == "" || password == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"Identifier, endpoint, and password are required.",
					window,
				)
				return
			}

			record.Identifier = identifier
			record.Endpoint = endpoint
			record.Password = password

			renewalPeriod := int64(0)

			periodText := strings.TrimSpace(renewalPeriodEntry.Text)
			if periodText != "" {
				parsed, err := strconv.ParseInt(periodText, 10, 64)
				if err != nil {
					dialog.ShowInformation(
						"Invalid Input",
						"Renewal period must be a valid number of days.",
						window,
					)
					return
				}

				if parsed < 0 {
					dialog.ShowInformation(
						"Invalid Input",
						"Renewal period cannot be negative.",
						window,
					)
					return
				}

				const maxRenewalDays = int64(106751)

				if parsed > maxRenewalDays {
					dialog.ShowInformation(
						"Invalid Input",
						"Renewal period is too large.",
						window,
					)
					return
				}

				renewalPeriod = parsed
			}

			record.RenewalPeriod = time.Duration(renewalPeriod) * 24 * time.Hour

			if onSave != nil {
				onSave(&record)
			}
		},
		window,
	)
}
