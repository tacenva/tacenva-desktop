package vault

import (
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	tacenvaentity "github.com/tacenva/tacenva-core/entity"
	"github.com/tacenva/tacenva-desktop/internal/ui/loading"
	"github.com/tacenva/tacenva-services/app"
	"github.com/tacenva/tacenva-services/app/sourceoftruth"
	vaultServicePackage "github.com/tacenva/tacenva-services/app/vault"
	coreApp "github.com/tacenva/tacpass-core/app"
	tacpassentity "github.com/tacenva/tacpass-core/entity"
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
	vaultService := vaultServicePackage.NewService(
		appDeps,
		context,
		coreService.Vault,
		coreService.VaultRecord,
		sotService,
	)

	isWriter := context.SelectedSot.Auth.Level.AllowsWrite()

	loadAccessGroups := func(refresh bool) ([]tacenvaentity.AccessGroup, error) {
		return vaultService.AllAccessGroup(refresh)
	}

	var searchTimer *time.Timer

	search := widget.NewEntry()
	search.SetPlaceHolder("Search collection...")

	allVault := make([]tacpassentity.Vault, 0)
	filteredVaults := make([]tacpassentity.Vault, 0)

	var entryList *widget.List
	var syncButton *widget.Button
	var reloadButton *widget.Button
	var newEntryButton *widget.Button
	var vaultContent fyne.CanvasObject

	needSyncState := false
	busy := false
	listLoading := true

	var stateMu sync.Mutex

	getOperationState := func() (bool, bool) {
		stateMu.Lock()
		defer stateMu.Unlock()
		return busy, listLoading
	}

	filterVault := func(vaults []tacpassentity.Vault, query string) []tacpassentity.Vault {
		query = strings.ToLower(strings.TrimSpace(query))

		if query == "" {
			return append([]tacpassentity.Vault(nil), vaults...)
		}

		filtered := make([]tacpassentity.Vault, 0, len(vaults))

		for _, item := range vaults {
			if strings.Contains(strings.ToLower(item.Name), query) {
				filtered = append(filtered, item)
			}
		}

		return filtered
	}

	updateActionStates := func() {
		currentBusy, currentLoading := getOperationState()
		actionsEnabled := !currentBusy && !currentLoading

		if syncButton != nil {
			if actionsEnabled && needSyncState {
				syncButton.Enable()
			} else {
				syncButton.Disable()
			}
		}

		if newEntryButton != nil {
			if actionsEnabled && isWriter {
				newEntryButton.Enable()
			} else {
				newEntryButton.Disable()
			}
		}

		if entryList != nil {
			entryList.Refresh()
		}
	}

	content := container.NewMax()
	loadingView := loading.New("Loading collections...")

	applyVault := func(vaults []tacpassentity.Vault, needSync bool) {
		allVault = append([]tacpassentity.Vault(nil), vaults...)
		filteredVaults = filterVault(allVault, search.Text)
		needSyncState = needSync

		updateActionStates()

		entryList.UnselectAll()
		entryList.Refresh()
	}

	var reload func()

	reload = func() {
		stateMu.Lock()
		if listLoading || busy {
			stateMu.Unlock()
			return
		}
		listLoading = true
		stateMu.Unlock()

		updateActionStates()

		go func() {
			updatedVaults, needSync, err := vaultService.List()

			fyne.Do(func() {
				stateMu.Lock()
				listLoading = false
				stateMu.Unlock()

				if err != nil && updatedVaults == nil {
					loadingView.Show(vaultContent)
					updateActionStates()
					dialog.ShowError(err, window)
					return
				}

				applyVault(updatedVaults, needSync)
				loadingView.Show(vaultContent)

				if err != nil {
					dialog.ShowError(err, window)
				}
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
				stateMu.Lock()
				busy = false
				stateMu.Unlock()
				updateActionStates()
			},
			nil,
		)
	}

	if context.IsRemote {
		syncButton = widget.NewButtonWithIcon(
			"Sync",
			theme.ViewRefreshIcon(),
			func() {
				currentBusy, currentLoading := getOperationState()
				if currentBusy || currentLoading {
					return
				}

				runOperation(
					"Syncing Vaults...",
					func() error {
						return vaultService.SyncVault()
					},
					func() {
						reload()
					},
				)
			},
		)
	} else {
		reloadButton = widget.NewButtonWithIcon(
			"Reload",
			theme.ViewRefreshIcon(),
			func() {
				reload()
			},
		)
	}

	openVaultForm := func(
		selectedVault *tacpassentity.Vault,
		onSave func(*tacpassentity.Vault, []string),
	) {
		currentBusy, currentLoading := getOperationState()
		if currentBusy || currentLoading {
			return
		}

		go func() {
			groups, err := loadAccessGroups(false)

			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				showForm(
					window,
					selectedVault,
					context.SelectedSot.AccessGroup.ID,
					groups,
					loadAccessGroups,
					onSave,
				)
			})
		}()
	}

	entryList = widget.NewList(
		func() int {
			return len(filteredVaults)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			updatedAt := widget.NewLabel("")
			updatedAt.TextStyle = fyne.TextStyle{Italic: true}

			var updateButton *widget.Button
			var deleteButton *widget.Button

			if isWriter {
				updateButton = widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), nil)
				deleteButton = widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			} else {
				updateButton = widget.NewButton("", nil)
				deleteButton = widget.NewButton("", nil)
				updateButton.Hide()
				deleteButton.Hide()
			}

			nameCell := container.NewBorder(nil, nil, nil, nil, name)
			dateCell := container.NewBorder(nil, nil, nil, nil, updatedAt)

			if isWriter {
				actions := container.NewHBox(updateButton, deleteButton)
				actionCell := container.NewBorder(nil, nil, nil, actions, nil)

				return container.NewGridWithColumns(3, nameCell, dateCell, actionCell)
			}

			return container.NewGridWithColumns(2, nameCell, dateCell)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(filteredVaults) {
				return
			}

			selected := filteredVaults[id]
			row := obj.(*fyne.Container)

			nameCell := row.Objects[0].(*fyne.Container)
			dateCell := row.Objects[1].(*fyne.Container)

			name := nameCell.Objects[0].(*widget.Label)
			updatedAt := dateCell.Objects[0].(*widget.Label)

			name.SetText(selected.Name)
			updatedAt.SetText(selected.UpdatedAt.Format("02 Jan 2006 15:04"))

			if !isWriter {
				return
			}

			actionCell := row.Objects[2].(*fyne.Container)
			actions := actionCell.Objects[0].(*fyne.Container)

			updateButton := actions.Objects[0].(*widget.Button)
			deleteButton := actions.Objects[1].(*widget.Button)

			currentBusy, currentLoading := getOperationState()
			actionsEnabled := !currentBusy && !currentLoading

			if actionsEnabled {
				updateButton.Enable()
				deleteButton.Enable()
			} else {
				updateButton.Disable()
				deleteButton.Disable()
			}

			updateButton.OnTapped = func() {
				currentBusy, currentLoading := getOperationState()
				if currentBusy || currentLoading {
					return
				}

				selectedVault := selected

				openVaultForm(&selectedVault, func(updated *tacpassentity.Vault, groupIDs []string) {
					updated.GroupIDs = groupIDs

					runOperation(
						"Saving Vault changes...",
						func() error {
							return vaultService.UpdateVault(updated)
						},
						func() {
							dialog.ShowInformation("Success", "Vault berhasil diperbarui.", window)
							reload()
						},
					)
				})
			}

			deleteButton.OnTapped = func() {
				currentBusy, currentLoading := getOperationState()
				if currentBusy || currentLoading {
					return
				}

				dialog.ShowConfirm(
					"Delete Vault",
					"Vault \""+selected.Name+"\" will be deleted. Continue?",
					func(ok bool) {
						if !ok {
							return
						}

						currentBusy, currentLoading := getOperationState()
						if currentBusy || currentLoading {
							dialog.ShowInformation("Offline", "Koneksi tidak tersedia. Penghapusan dibatalkan.", window)
							return
						}

						runOperation(
							"Deleting Vault...",
							func() error {
								return vaultService.DeleteVault(&selected)
							},
							func() {
								dialog.ShowInformation("Success", "Vault berhasil dihapus.", window)
								reload()
							},
						)
					},
					window,
				)
			}
		},
	)

	if isWriter {
		newEntryButton = widget.NewButtonWithIcon(
			"New",
			theme.ContentAddIcon(),
			func() {
				currentBusy, currentLoading := getOperationState()
				if currentBusy || currentLoading {
					return
				}

				openVaultForm(nil, func(newVault *tacpassentity.Vault, groupIDs []string) {
					runOperation(
						"Creating Vault...",
						func() error {
							return vaultService.CreateVault(newVault.Name, groupIDs)
						},
						func() {
							dialog.ShowInformation("Success", "Vault berhasil dibuat.", window)
							reload()
						},
					)
				})
			},
		)
	}

	toolbarButtons := make([]fyne.CanvasObject, 0, 2)

	if context.IsRemote {
		toolbarButtons = append(toolbarButtons, syncButton)
	} else {
		toolbarButtons = append(toolbarButtons, reloadButton)
	}

	if isWriter {
		toolbarButtons = append(toolbarButtons, newEntryButton)
	}

	toolbar := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(toolbarButtons...),
		search,
	)

	vaultContent = container.NewBorder(toolbar, nil, nil, nil, entryList)

	entryList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(filteredVaults) {
			return
		}

		selectedVault := filteredVaults[id]
		wrappedKey, err := vaultService.GetWrappedVaultKey(selectedVault.ID)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		recordScreen := NewRecordScreen(
			window,
			appDeps,
			context,
			masterKey,
			coreService,
			sotService,
			&selectedVault,
			wrappedKey,
			func() {
				content.Objects = []fyne.CanvasObject{vaultContent}
				content.Refresh()
			},

			vaultService,
			isWriter,
		)

		content.Objects = []fyne.CanvasObject{recordScreen.Content}
		content.Refresh()
		entryList.Unselect(id)
	}

	search.OnChanged = func(query string) {
		if searchTimer != nil {
			searchTimer.Stop()
		}

		searchTimer = time.AfterFunc(200*time.Millisecond, func() {
			filtered := filterVault(allVault, query)

			fyne.Do(func() {
				filteredVaults = filtered
				entryList.UnselectAll()
				entryList.Refresh()
			})
		})
	}

	content.Objects = []fyne.CanvasObject{loadingView.Content}

	go func() {
		updatedVaults, needSync, err := vaultService.List()

		fyne.Do(func() {
			stateMu.Lock()
			listLoading = false
			stateMu.Unlock()

			if err != nil && updatedVaults == nil {
				applyVault(allVault, false)
				loadingView.Show(vaultContent)
				updateActionStates()
				dialog.ShowError(err, window)
				return
			}

			applyVault(updatedVaults, needSync)
			loadingView.Show(vaultContent)

			if err != nil {
				dialog.ShowError(err, window)
			}
		})
	}()

	return &Screen{Content: content}
}

func showForm(
	window fyne.Window,
	selectedVault *tacpassentity.Vault,
	userGroupID string,
	groups []tacenvaentity.AccessGroup,
	loadGroups func(bool) ([]tacenvaentity.AccessGroup, error),
	onSave func(*tacpassentity.Vault, []string),
) {
	isEdit := selectedVault != nil

	var currentVault tacpassentity.Vault
	if isEdit {
		currentVault = *selectedVault
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetText(currentVault.Name)
	nameEntry.SetPlaceHolder("Vault name...")

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search groups...")

	selectedGroups := make(map[string]bool)

	if isEdit {
		for _, groupID := range currentVault.GroupIDs {
			selectedGroups[groupID] = true
		}
	}

	if userGroupID != "" {
		selectedGroups[userGroupID] = true
	}

	groupList := container.NewVBox()

	var updateGroupList func(string)

	updateGroupList = func(query string) {
		query = strings.ToLower(strings.TrimSpace(query))
		groupList.Objects = nil

		for _, group := range groups {
			if query != "" && !strings.Contains(strings.ToLower(group.Name), query) {
				continue
			}

			groupID := group.ID

			check := widget.NewCheck(group.Name, func(checked bool) {
				if groupID == userGroupID {
					selectedGroups[groupID] = true
					return
				}
				selectedGroups[groupID] = checked
			})

			check.SetChecked(selectedGroups[groupID])

			if groupID == userGroupID {
				check.Disable()
			}

			groupList.Add(check)
		}

		groupList.Refresh()
	}

	searchEntry.OnChanged = updateGroupList

	selectAll := widget.NewButton("Select All", func() {
		query := strings.ToLower(strings.TrimSpace(searchEntry.Text))

		for _, group := range groups {
			if query == "" || strings.Contains(strings.ToLower(group.Name), query) {
				selectedGroups[group.ID] = true
			}
		}

		if userGroupID != "" {
			selectedGroups[userGroupID] = true
		}

		updateGroupList(searchEntry.Text)
	})

	clearSelection := widget.NewButton("Deselect All", func() {
		query := strings.ToLower(strings.TrimSpace(searchEntry.Text))

		for _, group := range groups {
			if group.ID == userGroupID {
				continue
			}

			if query == "" || strings.Contains(strings.ToLower(group.Name), query) {
				delete(selectedGroups, group.ID)
			}
		}

		if userGroupID != "" {
			selectedGroups[userGroupID] = true
		}

		updateGroupList(searchEntry.Text)
	})

	refreshButton := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), nil)

	refreshButton.OnTapped = func() {
		refreshButton.Disable()

		go func() {
			updatedGroups, err := loadGroups(true)

			fyne.Do(func() {
				refreshButton.Enable()

				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				groups = updatedGroups
				updateGroupList(searchEntry.Text)
			})
		}()
	}

	groupToolbar := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(selectAll, clearSelection, refreshButton),
		searchEntry,
	)

	groupScroll := container.NewVScroll(groupList)
	groupScroll.SetMinSize(fyne.NewSize(420, 220))

	groupContent := container.NewBorder(groupToolbar, nil, nil, nil, groupScroll)
	nameRow := newLabelRow("Name", nameEntry)
	groupsRow := newLabelRow("Groups", groupContent)
	formContent := container.NewVBox(nameRow, groupsRow)
	formContainer := container.NewGridWrap(fyne.NewSize(760, 340), formContent)

	title := "New Vault"
	if isEdit {
		title = "Edit Vault"
	}

	updateGroupList("")

	dialog.ShowCustomConfirm(
		title,
		"Save",
		"Cancel",
		formContainer,
		func(ok bool) {
			if !ok {
				return
			}

			name := strings.TrimSpace(nameEntry.Text)
			if name == "" {
				dialog.ShowInformation("Invalid Input", "Name is required.", window)
				return
			}

			if userGroupID != "" {
				selectedGroups[userGroupID] = true
			}

			groupIDs := make([]string, 0, len(selectedGroups))
			for id, selected := range selectedGroups {
				if selected {
					groupIDs = append(groupIDs, id)
				}
			}

			sort.Strings(groupIDs)
			currentVault.Name = name

			if onSave != nil {
				onSave(&currentVault, groupIDs)
			}
		},
		window,
	)
}

func newLabelRow(label string, content fyne.CanvasObject) fyne.CanvasObject {
	return container.New(
		&labelRowLayout{labelWidth: 80},
		widget.NewLabel(label),
		content,
	)
}

type labelRowLayout struct {
	labelWidth float32
}

func (l *labelRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}

	label := objects[0]
	content := objects[1]

	label.Resize(fyne.NewSize(l.labelWidth, size.Height))
	label.Move(fyne.NewPos(0, 0))

	content.Resize(fyne.NewSize(
		maxFloat32(0, size.Width-l.labelWidth),
		size.Height,
	))
	content.Move(fyne.NewPos(l.labelWidth, 0))
}

func (l *labelRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(0, 0)
	}

	labelMin := objects[0].MinSize()
	contentMin := objects[1].MinSize()

	return fyne.NewSize(
		l.labelWidth+contentMin.Width,
		maxFloat32(labelMin.Height, contentMin.Height),
	)
}

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func runWithLoading(
	window fyne.Window,
	message string,
	operation func() error,
	onSuccess func(),
	onFinished func(),
	onComplete func(),
) {
	loadingDialog := dialog.NewProgressInfinite("Please wait", message, window)
	loadingDialog.Show()

	go func() {
		err := operation()

		fyne.Do(func() {
			loadingDialog.Hide()

			if onFinished != nil {
				onFinished()
			}

			if err != nil {
				dialog.ShowError(err, window)
				if onComplete != nil {
					onComplete()
				}
				return
			}

			if onSuccess != nil {
				onSuccess()
			}

			if onComplete != nil {
				onComplete()
			}
		})
	}()
}
