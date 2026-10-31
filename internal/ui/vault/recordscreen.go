package vault

import (
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
	isOnline := func() bool {
		if !context.IsRemote {
			return true
		}
		return appDeps.Client.IsReachable()
	}

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
			name := strings.ToLower(record.Name)
			endpoint := strings.ToLower(record.Endpoint)

			if strings.Contains(name, query) ||
				strings.Contains(endpoint, query) {
				filtered = append(filtered, record)
			}
		}

		return filtered
	}

	updateActionStates := func() {
		online := isOnline()
		working := isBusy() || isListLoading()

		if syncButton != nil {
			if online && !working && needSyncState {
				syncButton.Enable()
			} else {
				syncButton.Disable()
			}
		}

		if newEntryButton != nil {
			if online && !working && isWriter {
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
		if !isOnline() {
			dialog.ShowInformation(
				"Offline",
				"Koneksi tidak tersedia.",
				window,
			)
			return
		}

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
			if !isOnline() || isBusy() || isListLoading() {
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
				if !isOnline() || isBusy() || isListLoading() {
					return
				}

				showRecordForm(
					window,
					nil,
					isOnline,
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
			name := widget.NewLabel("")
			endpoint := widget.NewLabel("")
			password := widget.NewLabel(hiddenPassword)
			expiredAt := widget.NewLabel("")

			name.TextStyle = fyne.TextStyle{Bold: true}
			expiredAt.TextStyle = fyne.TextStyle{Italic: true}

			copyButton := widget.NewButtonWithIcon(
				"",
				theme.ContentCopyIcon(),
				nil,
			)

			passwordCell := container.NewBorder(
				nil, nil, nil, copyButton, password,
			)

			nameCell := container.NewBorder(
				nil, nil, nil, nil, name,
			)

			endpointCell := container.NewBorder(
				nil, nil, nil, nil, endpoint,
			)

			expiredAtCell := container.NewBorder(
				nil, nil, nil, nil, expiredAt,
			)

			info := container.NewGridWithColumns(
				4,
				nameCell,
				endpointCell,
				passwordCell,
				expiredAtCell,
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

			nameCell, ok := info.Objects[0].(*fyne.Container)
			if !ok || len(nameCell.Objects) == 0 {
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

			expiredAtCell, ok := info.Objects[3].(*fyne.Container)
			if !ok || len(expiredAtCell.Objects) == 0 {
				return
			}

			name, ok := nameCell.Objects[0].(*widget.Label)
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

			expiredAt, ok := expiredAtCell.Objects[0].(*widget.Label)
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

			name.SetText(record.Name)
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

			if record.ExpiredAt.IsZero() {
				expiredAt.SetText("-")
			} else {
				expiredAt.SetText(
					record.ExpiredAt.Format("02 Jan 2006"),
				)
			}

			if !isWriter {
				return
			}

			actionsEnabled := isOnline() &&
				!isBusy() &&
				!isListLoading()

			if actionsEnabled {
				updateButton.Enable()
				deleteButton.Enable()
			} else {
				updateButton.Disable()
				deleteButton.Disable()
			}

			updateButton.OnTapped = func() {
				if !isOnline() || isBusy() || isListLoading() {
					return
				}

				selectedRecord := record

				showRecordForm(
					window,
					&selectedRecord,
					isOnline,
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
				if !isOnline() || isBusy() || isListLoading() {
					return
				}

				dialog.ShowConfirm(
					"Delete Record",
					"Record \""+record.Name+
						"\" will be deleted. Continue?",
					func(ok bool) {
						if !ok {
							return
						}

						if !isOnline() || isBusy() || isListLoading() {
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

		searchTimer = time.AfterFunc(
			200*time.Millisecond,
			func() {
				filtered := filterVaultRecords(
					allVaultRecords,
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

	if context.IsRemote {
		go func() {
			wasOnline := isOnline()

			for {
				time.Sleep(5 * time.Second)

				online := isOnline()

				fyne.Do(func() {
					updateActionStates()
				})

				if online && !wasOnline {
					fyne.Do(func() {
						reload()
					})
				}

				wasOnline = online
			}
		}()
	}

	return &RecordScreen{
		Content: content,
	}
}

func showRecordForm(
	window fyne.Window,
	selectedRecord *tacpassentity.VaultRecord,
	isOnline func() bool,
	onSave func(record *tacpassentity.VaultRecord),
) {
	isEdit := selectedRecord != nil

	var record tacpassentity.VaultRecord

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
		nil, nil, nil, generatePasswordButton, passwordEntry,
	)

	expiredAtEntry := widget.NewDateEntry()

	if !record.ExpiredAt.IsZero() {
		expiredAt := record.ExpiredAt
		expiredAtEntry.SetDate(&expiredAt)
	}

	form := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Endpoint", endpointEntry),
		widget.NewFormItem("Password", passwordContainer),
		widget.NewFormItem("Expired At", expiredAtEntry),
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

			if isOnline == nil || !isOnline() {
				dialog.ShowInformation(
					"Offline",
					"Koneksi terputus. Perubahan tidak disimpan.",
					window,
				)
				return
			}

			name := strings.TrimSpace(nameEntry.Text)
			endpoint := strings.TrimSpace(endpointEntry.Text)
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

			if onSave != nil {
				onSave(&record)
			}
		},
		window,
	)
}
