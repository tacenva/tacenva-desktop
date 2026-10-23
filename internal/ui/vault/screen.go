package vault

import (
	"encoding/json"
	"log"
	"sort"
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
		coreService.VaultRecord,
		coreService.Auth,
		sotService,
	)

	isWriter := context.SelectedSot.Privilege == entity.PrivilegeWrite || context.SelectedSot.Privilege == entity.PrivilegeAdmin

	loadPermissionGroups := func(
		refresh bool,
	) ([]entity.PermissionGroup, error) {
		return vaultService.ListPermissionGroup(refresh)
	}

	var searchTimer *time.Timer

	search := widget.NewEntry()
	search.SetPlaceHolder("Search collection...")

	allVault := make([]entity.Vault, 0)
	filteredVaults := make([]entity.Vault, 0)

	var entryList *widget.List
	var syncButton *widget.Button
	var newEntryButton *widget.Button

	content := container.NewMax()
	loadingView := loading.New("Loading collections...")

	filterVault := func(
		vaults []entity.Vault,
		query string,
	) []entity.Vault {
		query = strings.ToLower(strings.TrimSpace(query))

		if query == "" {
			return append([]entity.Vault(nil), vaults...)
		}

		filtered := make([]entity.Vault, 0, len(vaults))

		for _, vault := range vaults {
			if strings.Contains(
				strings.ToLower(vault.Name),
				query,
			) {
				filtered = append(filtered, vault)
			}
		}

		return filtered
	}

	applyVault := func(
		vaults []entity.Vault,
		needSync bool,
	) {
		filteredVaults = filterVault(
			vaults,
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

	var reload func()

	reload = func() {
		syncButton.Disable()

		go func() {
			updatedVault, needSync, err := vaultService.List()

			fyne.Do(func() {
				if err != nil {
					if updatedVault == nil {
						dialog.ShowError(err, window)
						syncButton.Enable()
						return
					}

					applyVault(updatedVault, needSync)
					dialog.ShowError(err, window)
					return
				}

				applyVault(updatedVault, needSync)
			})
		}()
	}

	syncButton = widget.NewButtonWithIcon(
		"Sync",
		theme.ViewRefreshIcon(),
		func() {
			syncButton.Disable()
			if newEntryButton != nil {
				newEntryButton.Disable()
			}

			runWithLoading(
				window,
				"Syncing Vaults...",
				func() error {
					return vaultService.SyncVault()
				},
				func() {
					if newEntryButton != nil {
						newEntryButton.Enable()
					}
					reload()
				},
			)
		},
	)

	// Membuka form dengan group dari service asli.
	openVaultForm := func(
		selectedVault *entity.Vault,
		onSave func(*entity.Vault, []string),
	) {
		go func() {
			groups, err := loadPermissionGroups(false)

			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, window)
					return
				}

				showForm(
					window,
					selectedVault,
					context.SelectedSot.GroupID,
					groups,
					loadPermissionGroups,
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

			updatedAt.TextStyle = fyne.TextStyle{
				Italic: true,
			}

			var updateButton *widget.Button
			var deleteButton *widget.Button

			if isWriter {
				updateButton = widget.NewButtonWithIcon(
					"",
					theme.DocumentCreateIcon(),
					nil,
				)

				deleteButton = widget.NewButtonWithIcon(
					"",
					theme.DeleteIcon(),
					nil,
				)
			} else {
				updateButton = widget.NewButton("", nil)
				deleteButton = widget.NewButton("", nil)

				updateButton.Hide()
				deleteButton.Hide()
			}

			nameCell := container.NewBorder(
				nil, nil, nil, nil, name,
			)

			dateCell := container.NewBorder(
				nil, nil, nil, nil, updatedAt,
			)

			if isWriter {
				actions := container.NewHBox(
					updateButton,
					deleteButton,
				)

				actionCell := container.NewBorder(
					nil, nil, nil, actions, nil,
				)

				return container.NewGridWithColumns(
					3,
					nameCell,
					dateCell,
					actionCell,
				)
			}

			return container.NewGridWithColumns(
				2,
				nameCell,
				dateCell,
			)
		},
		func(
			id widget.ListItemID,
			obj fyne.CanvasObject,
		) {
			if id < 0 || id >= len(filteredVaults) {
				return
			}

			vault := filteredVaults[id]
			row := obj.(*fyne.Container)

			nameCell := row.Objects[0].(*fyne.Container)
			dateCell := row.Objects[1].(*fyne.Container)

			name := nameCell.Objects[0].(*widget.Label)
			updatedAt := dateCell.Objects[0].(*widget.Label)

			name.SetText(vault.Name)
			updatedAt.SetText(
				vault.UpdatedAt.Format("02 Jan 2006 15:04"),
			)

			if !isWriter {
				return
			}

			actionCell := row.Objects[2].(*fyne.Container)
			actions := actionCell.Objects[0].(*fyne.Container)

			updateButton := actions.Objects[0].(*widget.Button)
			deleteButton := actions.Objects[1].(*widget.Button)

			updateButton.OnTapped = func() {
				selectedVault := vault

				openVaultForm(
					&selectedVault,
					func(
						updated *entity.Vault,
						groupIDs []string,
					) {
						updated.GroupIDs = groupIDs

						updateButton.Disable()
						deleteButton.Disable()
						if newEntryButton != nil {
							newEntryButton.Disable()
						}

						runWithLoading(
							window,
							"Saving Vault changes...",
							func() error {
								return vaultService.UpdateVault(updated)
							},
							func() {
								updateButton.Enable()
								deleteButton.Enable()
								newEntryButton.Enable()

								dialog.ShowInformation(
									"Success",
									"Vault berhasil diperbarui.",
									window,
								)

								reload()
							},
						)
					},
				)
			}

			deleteButton.OnTapped = func() {
				dialog.ShowConfirm(
					"Delete Vault",
					"Vault \""+vault.Name+
						"\" will be deleted. Continue?",
					func(ok bool) {
						if !ok {
							return
						}

						updateButton.Disable()
						deleteButton.Disable()

						go func() {
							err := vaultService.DeleteVault(
								&vault,
							)

							fyne.Do(func() {
								updateButton.Enable()
								deleteButton.Enable()

								if err != nil {
									dialog.ShowError(err, window)
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

	if isWriter {
		newEntryButton = widget.NewButtonWithIcon(
			"New",
			theme.ContentAddIcon(),
			func() {
				openVaultForm(
					nil,
					func(
						newVault *entity.Vault,
						groupIDs []string,
					) {
						syncButton.Disable()
						if newEntryButton != nil {
							newEntryButton.Disable()
						}

						runWithLoading(
							window,
							"Creating Vault...",
							func() error {
								return vaultService.CreateVault(
									newVault.Name,
									groupIDs,
								)
							},
							func() {
								syncButton.Enable()

								dialog.ShowInformation(
									"Success",
									"Vault berhasil dibuat.",
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

	toolbarButtons := []fyne.CanvasObject{
		syncButton,
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

	vaultContent := container.NewBorder(
		toolbar,
		nil,
		nil,
		nil,
		entryList,
	)

	entryList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(filteredVaults) {
			return
		}

		selectedVault := filteredVaults[id]
		vaultJSON, err := json.MarshalIndent(selectedVault, "", "  ")
		if err != nil {
			log.Printf("[DEBUG] Marshal selectedVault error: %v", err)
		} else {
			log.Printf("[DEBUG] selectedVault JSON:\n%s", string(vaultJSON))
		}

		wrappedKey := selectedVault.WrappedKey
		if !context.IsRemote {
			var err error
			wrappedKey, err = vaultService.GetWrappedKey(selectedVault.ID)
			if err != nil {
				return
			}
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
				filtered := filterVault(
					allVault,
					query,
				)

				fyne.Do(func() {
					filteredVaults = filtered
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
		updatedVaultAccesses, needSync, err := vaultService.List()

		fyne.Do(func() {
			if err != nil {
				if updatedVaultAccesses == nil {
					loadingView.ShowError(
						"Failed to load collections.",
					)
					dialog.ShowError(err, window)
					return
				}

				applyVault(updatedVaultAccesses, needSync)
				loadingView.Show(vaultContent)
				dialog.ShowError(err, window)
				return
			}

			applyVault(updatedVaultAccesses, needSync)
			loadingView.Show(vaultContent)
		})
	}()

	return &Screen{
		Content: content,
	}
}

func showForm(
	window fyne.Window,
	selectedVault *entity.Vault,
	userGroupID string,
	groups []entity.PermissionGroup,
	loadGroups func(bool) ([]entity.PermissionGroup, error),
	onSave func(vault *entity.Vault, groupIDs []string),
) {
	isEdit := selectedVault != nil

	var vault entity.Vault
	if isEdit {
		vault = *selectedVault
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetText(vault.Name)
	nameEntry.SetPlaceHolder("Vault name...")

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search groups...")

	selectedGroups := make(map[string]bool)

	if isEdit {
		for _, groupID := range vault.GroupIDs {
			selectedGroups[groupID] = true
		}

		debugJSON, err := json.MarshalIndent(vault.GroupIDs, "", "  ")
		if err != nil {
			log.Printf("[DEBUG] failed to marshal vault.GroupIDs: %v", err)
		} else {
			log.Printf("[DEBUG] vault.GroupIDs:\n%s", debugJSON)
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
			if query != "" &&
				!strings.Contains(strings.ToLower(group.Name), query) {
				continue
			}

			groupID := group.ID

			check := widget.NewCheck(
				group.Name,
				func(checked bool) {
					if groupID == userGroupID {
						selectedGroups[groupID] = true
						return
					}

					selectedGroups[groupID] = checked
				},
			)

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
			if query == "" ||
				strings.Contains(strings.ToLower(group.Name), query) {
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

			if query == "" ||
				strings.Contains(strings.ToLower(group.Name), query) {
				delete(selectedGroups, group.ID)
			}
		}

		if userGroupID != "" {
			selectedGroups[userGroupID] = true
		}

		updateGroupList(searchEntry.Text)
	})

	var refreshButton *widget.Button

	refreshButton = widget.NewButtonWithIcon(
		"Refresh",
		theme.ViewRefreshIcon(),
		func() {
			refreshButton.Disable()

			go func() {
				updatedGroups, err := loadGroups(true)

				fyne.Do(func() {
					defer refreshButton.Enable()

					if err != nil {
						dialog.ShowError(err, window)
						return
					}

					groups = updatedGroups
					updateGroupList(searchEntry.Text)
				})
			}()
		},
	)

	groupToolbar := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(
			selectAll,
			clearSelection,
			refreshButton,
		),
		searchEntry,
	)

	groupScroll := container.NewVScroll(groupList)
	groupScroll.SetMinSize(fyne.NewSize(420, 220))

	groupContent := container.NewBorder(
		groupToolbar,
		nil,
		nil,
		nil,
		groupScroll,
	)

	nameRow := newLabelRow("Name", nameEntry)
	groupsRow := newLabelRow("Groups", groupContent)

	formContent := container.NewVBox(
		nameRow,
		groupsRow,
	)

	formContainer := container.NewGridWrap(
		fyne.NewSize(760, 340),
		formContent,
	)

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
				dialog.ShowInformation(
					"Invalid Input",
					"Name is required.",
					window,
				)
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
			vault.Name = name

			if onSave != nil {
				onSave(&vault, groupIDs)
			}
		},
		window,
	)
}

func newLabelRow(
	label string,
	content fyne.CanvasObject,
) fyne.CanvasObject {
	return container.New(
		&labelRowLayout{labelWidth: 80},
		widget.NewLabel(label),
		content,
	)
}

type labelRowLayout struct {
	labelWidth float32
}

func (l *labelRowLayout) Layout(
	objects []fyne.CanvasObject,
	size fyne.Size,
) {
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

func (l *labelRowLayout) MinSize(
	objects []fyne.CanvasObject,
) fyne.Size {
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
) {
	loadingDialog := dialog.NewProgressInfinite(
		"Please wait",
		message,
		window,
	)
	loadingDialog.Show()

	go func() {
		err := operation()

		fyne.Do(func() {
			loadingDialog.Hide()

			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			if onSuccess != nil {
				onSuccess()
			}
		})
	}()
}
