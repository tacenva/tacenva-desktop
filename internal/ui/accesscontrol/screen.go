package accesscontrol

import (
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-core/contract"
	"github.com/tacenva/tacenva-core/entity"
	"github.com/tacenva/tacenva-services/app"
	"github.com/tacenva/tacenva-services/app/accesscontrol"
	coreApp "github.com/tacenva/tacpass-core/app"
)

type Screen struct {
	Content fyne.CanvasObject
	Reload  func()
	window  fyne.Window
}

func New(
	window fyne.Window,
	appDeps *app.Deps,
	context *app.Context,
	coreService *coreApp.Services,
) *Screen {
	accesscontrolService := accesscontrol.NewService(appDeps, context, coreService.AccessControl)

	var loadingUnits bool
	var unitRequestID uint64

	groupLoadingBar := widget.NewProgressBarInfinite()
	groupLoadingLabel := widget.NewLabel("Loading access groups...")

	groupLoadingView := container.NewCenter(
		container.NewVBox(
			groupLoadingBar,
			groupLoadingLabel,
		),
	)

	unitLoadingBar := widget.NewProgressBarInfinite()
	unitLoadingLabel := widget.NewLabel("Loading access units...")

	unitLoadingView := container.NewCenter(
		container.NewVBox(
			unitLoadingBar,
			unitLoadingLabel,
		),
	)

	var groupTree []contract.AccessGroupTree
	var units []contract.AccessUnitWithUserCount
	var loadGroups func()
	var loadUnits func(groupID string)

	var selectedGroupTree *contract.AccessGroupTree
	var navigationStack []*contract.AccessGroupTree
	var selectedUnit *contract.AccessUnitWithUserCount
	var unitsByGroup = make(map[string][]contract.AccessUnitWithUserCount)

	var render func()
	var selectGroup func(*contract.AccessGroupTree)

	panelOpen := false
	showingUsers := false

	pageTitle := widget.NewLabelWithStyle(
		"Access Control",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	pageSubtitle := widget.NewLabel(
		"Manage access groups and their permissions.",
	)

	groupContainer := container.NewVBox()
	unitContainer := container.NewVBox()
	contentBody := container.NewVBox()

	panelContainer := container.NewMax()
	root := container.NewMax()

	backButton := widget.NewButtonWithIcon(
		"Back",
		theme.NavigateBackIcon(),
		func() {
			// Dari halaman Users, kembali ke Access Units.
			if showingUsers {
				showingUsers = false
				selectedUnit = nil
				render()
				return
			}

			// Navigasi parent group seperti sebelumnya.
			if len(navigationStack) <= 1 {
				return
			}

			navigationStack = navigationStack[:len(navigationStack)-1]
			selectedGroupTree = navigationStack[len(navigationStack)-1]

			units = unitsByGroup[selectedGroupTree.Group.ID]
			render()
		},
	)

	reloadButton := widget.NewButtonWithIcon(
		"Reload",
		theme.ViewRefreshIcon(),
		func() {
			loadGroups()
		},
	)

	header := container.NewVBox(
		container.NewBorder(
			nil,
			nil,
			container.NewHBox(backButton, pageTitle),
			reloadButton,
			nil,
		),
		pageSubtitle,
	)

	mainContent := container.NewPadded(
		container.NewScroll(
			container.NewVBox(
				header,
				widget.NewSeparator(),
				contentBody,
			),
		),
	)

	updateLayout := func() {
		if panelOpen && !showingUsers {
			split := container.NewHSplit(mainContent, panelContainer)
			split.SetOffset(0.68)
			root.Objects = []fyne.CanvasObject{split}
		} else {
			root.Objects = []fyne.CanvasObject{mainContent}
		}

		root.Refresh()
	}

	openPanel := func() {
		panelOpen = true
		render()
	}

	showEditPlaceholder := func(group *entity.AccessGroup) {
		groupNameEntry := widget.NewEntry()
		groupNameEntry.SetText(group.Name)
		groupNameEntry.SetPlaceHolder("Enter group name")

		form := dialog.NewForm(
			"Edit Access Group",
			"Save",
			"Cancel",
			[]*widget.FormItem{
				widget.NewFormItem("Group Name", groupNameEntry),
			},
			func(confirmed bool) {
				if !confirmed {
					return
				}

				newName := strings.TrimSpace(groupNameEntry.Text)
				if newName == "" {
					dialog.ShowError(
						errors.New("Group name cannot be empty"),
						window,
					)
					return
				}

				updatedGroup := *group
				updatedGroup.Name = newName

				if err := accesscontrolService.UpdateGroup(&updatedGroup); err != nil {
					dialog.ShowError(err, window)
					return
				}

				group.Name = newName

				dialog.ShowInformation(
					"Success",
					fmt.Sprintf(
						"Access group %q updated successfully.",
						updatedGroup.Name,
					),
					window,
				)
			},
			window,
		)

		form.Resize(fyne.NewSize(400, 180))
		form.Show()
	}

	showDeletePlaceholder := func(group *entity.AccessGroup) {
		dialog.ShowConfirm(
			"Delete Subgroup",
			fmt.Sprintf(
				"Are you sure you want to delete subgroup %q?",
				group.Name,
			),
			func(confirmed bool) {
				if !confirmed {
					return
				}

				if err := accesscontrolService.DeleteGroup(group.ID); err != nil {
					dialog.ShowError(err, window)
					return
				}

				dialog.ShowInformation(
					"Success",
					fmt.Sprintf(
						"Subgroup %q deleted successfully.",
						group.Name,
					),
					window,
				)
			},
			window,
		)
	}

	showNewPlaceholder := func() {
		groupNameEntry := widget.NewEntry()
		groupNameEntry.SetPlaceHolder("Enter access group name")

		parentAccessCheck := widget.NewCheck(
			"Allow access to parent group",
			nil,
		)
		parentAccessCheck.SetChecked(true)

		adminNameEntry := widget.NewEntry()
		adminNameEntry.SetPlaceHolder("Enter admin name")

		autoApproveCheck := widget.NewCheck(
			"Automatically approve users",
			nil,
		)

		groupSection := container.NewVBox(
			widget.NewLabelWithStyle(
				"Access Group",
				fyne.TextAlignLeading,
				fyne.TextStyle{Bold: true},
			),
			widget.NewForm(
				widget.NewFormItem("Group Name", groupNameEntry),
			),
			parentAccessCheck,
		)

		adminSection := container.NewVBox(
			widget.NewLabelWithStyle(
				"Admin Access Unit",
				fyne.TextAlignLeading,
				fyne.TextStyle{Bold: true},
			),
			widget.NewForm(
				widget.NewFormItem("Admin Name", adminNameEntry),
			),
			autoApproveCheck,
		)

		content := container.NewVBox(
			groupSection,
			widget.NewSeparator(),
			adminSection,
		)

		form := dialog.NewCustomConfirm(
			"Create Access Group",
			"Create",
			"Cancel",
			content,
			func(confirmed bool) {
				if !confirmed {
					return
				}

				groupName := strings.TrimSpace(groupNameEntry.Text)
				adminName := strings.TrimSpace(adminNameEntry.Text)

				if groupName == "" {
					dialog.ShowError(
						fmt.Errorf("group name is required"),
						window,
					)
					return
				}

				if adminName == "" {
					dialog.ShowError(
						fmt.Errorf("admin name is required"),
						window,
					)
					return
				}

				newGroup := &entity.AccessGroup{
					Name: groupName,
				}

				adminUnit := &entity.AccessUnit{
					Name:        adminName,
					Level:       entity.LevelAdmin,
					AutoApprove: autoApproveCheck.Checked,
				}

				parentAccess := parentAccessCheck.Checked

				go func() {
					err := accesscontrolService.CreateGroup(
						newGroup,
						adminUnit,
						parentAccess,
					)

					fyne.Do(func() {
						if err != nil {
							dialog.ShowError(err, window)
							return
						}

						dialog.ShowInformation(
							"Success",
							fmt.Sprintf(
								"Access group %q created successfully.",
								groupName,
							),
							window,
						)

						loadGroups()
					})
				}()
			},
			window,
		)

		form.Resize(fyne.NewSize(440, 400))
		form.Show()
	}

	// Membuka halaman Users di area konten utama, bukan dialog.
	showUnitUsers := func(unit contract.AccessUnitWithUserCount) {
		currentUnit := unit
		selectedUnit = &currentUnit
		showingUsers = true

		// Tutup panel subgroup ketika pindah halaman.
		panelOpen = false

		render()
	}

	render = func() {
		if selectedGroupTree == nil {
			return
		}

		contentBody.RemoveAll()

		if showingUsers && selectedUnit != nil {
			backButton.Enable()

			levelTag := widget.NewLabel(
				fmt.Sprintf("Level: %s", selectedUnit.Level),
			)

			statusText := "Active"
			if selectedUnit.Revoked {
				statusText = "Revoked"
			}

			statusTag := widget.NewLabel(
				fmt.Sprintf("Status: %s", statusText),
			)

			stats := container.NewGridWithColumns(
				3,
				newGroupStatCard(
					"Users",
					fmt.Sprintf("%d", selectedUnit.UserCount),
				),
				newGroupStatCard(
					"Pending",
					fmt.Sprintf("%d", selectedUnit.PendingCount),
				),
				newGroupStatCard(
					"Revoked",
					fmt.Sprintf("%d", selectedUnit.RevokedCount),
				),
			)

			unitInfo := container.NewVBox(
				widget.NewLabelWithStyle(
					selectedUnit.Name,
					fyne.TextAlignLeading,
					fyne.TextStyle{Bold: true},
				),
				container.NewHBox(
					levelTag,
					statusTag,
				),
				stats,
			)

			unitInfoCard := container.NewHBox(
				container.New(
					layout.NewGridWrapLayout(
						fyne.NewSize(680, unitInfo.MinSize().Height),
					),
					unitInfo,
				),
			)

			contentBody.Objects = []fyne.CanvasObject{
				unitInfoCard,
				widget.NewLabelWithStyle(
					"Users",
					fyne.TextAlignLeading,
					fyne.TextStyle{Bold: true},
				),
				newUserListSection(
					window,
					selectedUnit.Users,
					func(user entity.User) {
						showUserActionPlaceholder(
							window,
							"Approve User",
							fmt.Sprintf("Approve user %q?", user.Hostname),
						)
					},
					func(user entity.User) {
						showUserActionPlaceholder(
							window,
							"Revoke User",
							fmt.Sprintf("Revoke access for user %q?", user.Hostname),
						)
					},
					func(user entity.User) {
						dialog.ShowInformation(
							"Not Implemented",
							"Delete user has not been connected to the service yet.",
							window,
						)
					},
				),
			}

			contentBody.Refresh()
			updateLayout()
			return
		}

		// Halaman Access Group dan Access Units.
		groupContainer.RemoveAll()
		groupContainer.Add(
			newAccessGroupCard(
				selectedGroupTree,
				openPanel,
				func(group *contract.AccessGroupTree) {
					showEditPlaceholder(&group.Group)
				},
			),
		)
		groupContainer.Refresh()

		if !loadingUnits {
			unitContainer.Objects = []fyne.CanvasObject{
				newAccessUnitSection(
					units,
					showUnitUsers,
					func(accessUnit *entity.AccessUnit) {
						window := fyne.CurrentApp().Driver().AllWindows()[0]

						if selectedGroupTree == nil {
							dialog.ShowInformation(
								"Error",
								"Select an access group first.",
								window,
							)
							return
						}

						if accessUnit.ID == "" {
							// Create Access Unit
							if err := accesscontrolService.CreateUnit(accessUnit); err != nil {
								dialog.ShowError(err, window)
								return
							}
						} else {
							// Update Access Unit
							if err := accesscontrolService.UpdateUnit(accessUnit); err != nil {
								dialog.ShowError(err, window)
								return
							}
						}

						dialog.ShowInformation(
							"Success",
							"Access unit saved successfully.",
							window,
						)

						loadUnits(selectedGroupTree.Group.ID)
					},
					func(au *entity.AccessUnit) {
						window := fyne.CurrentApp().Driver().AllWindows()[0]

						if selectedGroupTree == nil {
							dialog.ShowInformation(
								"Error",
								"Select an access group first.",
								window,
							)
							return
						}

						if err := accesscontrolService.DeleteUnit(au.ID); err != nil {
							dialog.ShowError(err, window)
							return
						}

						dialog.ShowInformation(
							"Success",
							fmt.Sprintf(
								"Access unit %q deleted successfully.",
								au.Name,
							),
							window,
						)

						loadUnits(selectedGroupTree.Group.ID)
					},
				),
			}
			unitContainer.Refresh()
		}

		if len(navigationStack) > 1 {
			backButton.Enable()
		} else {
			backButton.Disable()
		}

		contentBody.Add(
			container.NewGridWrap(
				fyne.NewSize(1, 12),
				widget.NewLabel(""),
			),
		)
		contentBody.Add(groupContainer)

		contentBody.Add(
			container.NewGridWrap(
				fyne.NewSize(1, 24),
				widget.NewLabel(""),
			),
		)
		contentBody.Add(unitContainer)

		if panelOpen {
			panelContainer.Objects = []fyne.CanvasObject{
				newAccessGroupPanel(
					selectedGroupTree,
					func(group *contract.AccessGroupTree) {
						selectGroup(group)
					},
					func(group *contract.AccessGroupTree) {
						showEditPlaceholder(&group.Group)
					},
					showNewPlaceholder,
					func(group *contract.AccessGroupTree) {
						showDeletePlaceholder(&group.Group)
					},
					func() {
						panelOpen = false
						updateLayout()
					},
				),
			}
			panelContainer.Refresh()
		}

		updateLayout()
	}

	selectGroup = func(group *contract.AccessGroupTree) {
		if group == nil {
			return
		}

		showingUsers = false
		selectedUnit = nil

		// Jika group sudah ada di navigation stack,
		// kembali ke posisi group tersebut.
		for i, existing := range navigationStack {
			if existing.Group.ID == group.Group.ID {
				navigationStack = navigationStack[:i+1]
				selectedGroupTree = existing

				units = unitsByGroup[group.Group.ID]
				render()

				if _, exists := unitsByGroup[group.Group.ID]; !exists {
					loadUnits(group.Group.ID)
				}
				return
			}
		}

		selectedGroupTree = group
		navigationStack = append(navigationStack, group)
		accesscontrolService.SetSelectedGroup(&group.Group)

		units = unitsByGroup[group.Group.ID]
		render()

		if _, exists := unitsByGroup[group.Group.ID]; !exists {
			loadUnits(group.Group.ID)
		}
	}

	loadGroups = func() {
		reloadButton.Disable()

		groupLoadingBar.Start()
		groupContainer.Objects = []fyne.CanvasObject{
			groupLoadingView,
		}
		groupContainer.Refresh()

		// Simpan posisi navigasi saat ini.
		currentGroupID := ""
		if selectedGroupTree != nil {
			currentGroupID = selectedGroupTree.Group.ID
		}

		go func() {
			var parentGroup *entity.AccessGroup
			if selectedGroupTree != nil {
				parentGroup = &selectedGroupTree.Group
			}

			result, err := accesscontrolService.GroupWithChildren(parentGroup)

			fyne.Do(func() {
				groupLoadingBar.Stop()
				reloadButton.Enable()

				if err != nil {
					groupContainer.Objects = []fyne.CanvasObject{
						widget.NewLabel("Failed to load access groups."),
					}
					groupContainer.Refresh()
					dialog.ShowError(err, window)
					return
				}

				groupTree = result

				if len(groupTree) == 0 {
					selectedGroupTree = nil
					navigationStack = nil
					units = nil

					groupContainer.Objects = []fyne.CanvasObject{
						widget.NewLabel("No access groups found."),
					}
					groupContainer.Refresh()

					unitContainer.Objects = []fyne.CanvasObject{
						widget.NewLabel("No access group selected."),
					}
					unitContainer.Refresh()

					updateLayout()
					return
				}

				// Kalau belum ada group terpilih, gunakan root pertama.
				if currentGroupID == "" || selectedGroupTree == nil {
					navigationStack = nil
					selectedGroupTree = nil
					selectGroup(&groupTree[0])
					return
				}

				// GroupWithChildren dipanggil dengan parent aktif.
				// Pertahankan group aktif jika masih ada di hasil.
				var refreshedGroup *contract.AccessGroupTree

				for i := range groupTree {
					if groupTree[i].Group.ID == currentGroupID {
						refreshedGroup = &groupTree[i]
						break
					}
				}

				if refreshedGroup != nil {
					// Ganti pointer group aktif dengan data terbaru,
					// tanpa mereset stack.
					selectedGroupTree = refreshedGroup

					if len(navigationStack) > 0 {
						navigationStack[len(navigationStack)-1] = refreshedGroup
					}

					delete(unitsByGroup, currentGroupID)
					render()
					loadUnits(currentGroupID)
					return
				}

				// Jika hasil query tidak menyertakan group aktif,
				// jangan diam-diam pindah ke root pertama.
				groupContainer.Objects = []fyne.CanvasObject{
					widget.NewLabel("Unable to refresh the selected access group."),
				}
				groupContainer.Refresh()
				updateLayout()
			})
		}()
	}

	loadUnits = func(groupID string) {
		unitRequestID++
		requestID := unitRequestID

		loadingUnits = true
		unitLoadingBar.Start()

		unitContainer.Objects = []fyne.CanvasObject{
			unitLoadingView,
		}
		unitContainer.Refresh()

		go func() {
			result, err := accesscontrolService.UnitList(
				groupID,
			)

			fyne.Do(func() {
				if requestID != unitRequestID {
					return
				}

				unitLoadingBar.Stop()
				loadingUnits = false

				if err != nil {
					unitContainer.Objects = []fyne.CanvasObject{
						widget.NewLabel("Failed to load access units."),
					}
					unitContainer.Refresh()
					dialog.ShowError(err, window)
					return
				}

				units = result
				unitsByGroup[groupID] = result
				unitContainer.Objects = []fyne.CanvasObject{
					newAccessUnitSection(
						units,
						showUnitUsers,
						func(accessUnit *entity.AccessUnit) {
							window := fyne.CurrentApp().Driver().AllWindows()[0]

							if selectedGroupTree == nil {
								dialog.ShowInformation(
									"Error",
									"Select an access group first.",
									window,
								)
								return
							}

							if accessUnit.ID == "" {
								// Create Access Unit
								if err := accesscontrolService.CreateUnit(accessUnit); err != nil {
									dialog.ShowError(err, window)
									return
								}
							} else {
								// Update Access Unit
								if err := accesscontrolService.UpdateUnit(accessUnit); err != nil {
									dialog.ShowError(err, window)
									return
								}
							}

							dialog.ShowInformation(
								"Success",
								"Access unit saved successfully.",
								window,
							)

							loadUnits(selectedGroupTree.Group.ID)
						},
						func(au *entity.AccessUnit) {
							window := fyne.CurrentApp().Driver().AllWindows()[0]

							if selectedGroupTree == nil {
								dialog.ShowInformation(
									"Error",
									"Select an access group first.",
									window,
								)
								return
							}

							if err := accesscontrolService.DeleteUnit(au.ID); err != nil {
								dialog.ShowError(err, window)
								return
							}

							dialog.ShowInformation(
								"Success",
								fmt.Sprintf(
									"Access unit %q deleted successfully.",
									au.Name,
								),
								window,
							)

							loadUnits(selectedGroupTree.Group.ID)
						},
					),
				}
				unitContainer.Refresh()
			})
		}()
	}

	loadGroups()

	return &Screen{
		Content: root,
		Reload: func() {
			loadGroups()
		},
		window: window,
	}
}

func showUserActionPlaceholder(
	window fyne.Window,
	title string,
	message string,
) {
	dialog.ShowConfirm(
		title,
		message,
		func(confirmed bool) {
			if !confirmed {
				return
			}

			dialog.ShowInformation(
				"Not Implemented",
				"This action has not been connected to the service yet.",
				window,
			)
		},
		window,
	)
}
