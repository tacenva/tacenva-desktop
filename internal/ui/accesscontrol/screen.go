package accesscontrol

import (
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-desktop/internal/ui/loading"
	"github.com/tacenva/tacenva-desktop/internal/util"
	"github.com/tacenva/tacenva-services/app"
	acServicePackage "github.com/tacenva/tacenva-services/app/accesscontrol"
	coreApp "github.com/tacenva/tacpass-core/app"
	"github.com/tacenva/tacpass-core/entity"
)

type Screen struct {
	Content fyne.CanvasObject
	Reload  func()
}

func New(
	window fyne.Window,
	appDeps *app.Deps,
	context *app.Context,
	coreService *coreApp.Services,
) *Screen {
	var acContent fyne.CanvasObject

	acService := acServicePackage.NewService(
		appDeps,
		context,
		coreService.AccessControl,
		coreService.Auth,
	)

	var searchTimer *time.Timer
	var reloadMu sync.Mutex
	reloading := false

	search := widget.NewEntry()
	search.SetPlaceHolder("Search permissions...")

	allACL := make([]entity.Permission, 0)
	filteredACL := make([]entity.Permission, 0)
	allGroups := make([]entity.PermissionGroup, 0)

	var entryList *widget.List
	var showManageGroups func()
	var reloadButton *widget.Button

	content := container.NewMax()
	loadingView := loading.New("Loading access controls...")
	content.Objects = []fyne.CanvasObject{loadingView.Content}

	filterACL := func(
		permissions []entity.Permission,
		query string,
	) []entity.Permission {
		query = strings.ToLower(strings.TrimSpace(query))

		if query == "" {
			return append([]entity.Permission(nil), permissions...)
		}

		filtered := make([]entity.Permission, 0)

		for _, permission := range permissions {
			name := strings.ToLower(permission.Name)
			privilege := strings.ToLower(string(permission.Privilege))
			group := strings.ToLower(permission.Group.Name)

			if strings.Contains(name, query) ||
				strings.Contains(privilege, query) ||
				strings.Contains(group, query) {
				filtered = append(filtered, permission)
			}
		}

		return filtered
	}

	setContent := func(obj fyne.CanvasObject) {
		content.Objects = []fyne.CanvasObject{obj}
		content.Refresh()
	}

	setReloading := func(value bool) {
		reloadMu.Lock()
		reloading = value
		reloadMu.Unlock()

		if reloadButton == nil {
			return
		}

		if value {
			reloadButton.Disable()
		} else {
			reloadButton.Enable()
		}
	}

	reload := func() {
		reloadMu.Lock()

		if reloading {
			reloadMu.Unlock()
			return
		}

		reloading = true
		reloadMu.Unlock()

		fyne.Do(func() {
			if reloadButton != nil {
				reloadButton.Disable()
			}
		})

		go func() {
			permissions, err := acService.List()
			if err != nil {
				fyne.Do(func() {
					setReloading(false)

					if acContent != nil {
						setContent(acContent)
					}

					dialog.ShowError(err, window)
				})
				return
			}

			groups, err := acService.GroupList()
			if err != nil {
				fyne.Do(func() {
					setReloading(false)

					if acContent != nil {
						setContent(acContent)
					}

					dialog.ShowError(err, window)
				})
				return
			}

			fyne.Do(func() {
				allACL = permissions
				allGroups = groups
				filteredACL = filterACL(allACL, search.Text)

				entryList.UnselectAll()
				entryList.Refresh()

				setContent(acContent)
				setReloading(false)
			})
		}()
	}

	entryList = widget.NewList(
		func() int {
			return len(filteredACL)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			group := widget.NewLabel("")
			privilege := widget.NewLabel("")

			editButton := widget.NewButtonWithIcon(
				"",
				theme.DocumentCreateIcon(),
				nil,
			)

			revokeButton := widget.NewButton("Revoke", nil)

			deleteButton := widget.NewButtonWithIcon(
				"",
				theme.DeleteIcon(),
				nil,
			)

			actions := container.NewHBox(
				editButton,
				revokeButton,
				deleteButton,
			)

			columns := container.NewGridWithColumns(
				3,
				name,
				group,
				privilege,
			)

			return container.NewBorder(
				nil,
				nil,
				nil,
				actions,
				columns,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(filteredACL) {
				return
			}

			acData := filteredACL[id]
			row := obj.(*fyne.Container)

			columns := row.Objects[0].(*fyne.Container)
			actions := row.Objects[1].(*fyne.Container)

			name := columns.Objects[0].(*widget.Label)
			group := columns.Objects[1].(*widget.Label)
			privilege := columns.Objects[2].(*widget.Label)

			editButton := actions.Objects[0].(*widget.Button)
			revokeButton := actions.Objects[1].(*widget.Button)
			deleteButton := actions.Objects[2].(*widget.Button)

			name.SetText(acData.Name)
			group.SetText(acData.Group.Name)

			if acData.Revoked {
				privilege.SetText(
					string(acData.Privilege) + " (REVOKED)",
				)
				revokeButton.SetText("Restore")
			} else {
				privilege.SetText(string(acData.Privilege))
				revokeButton.SetText("Revoke")
			}

			editButton.OnTapped = func() {
				selected := acData

				showForm(
					window,
					&selected,
					allGroups,
					func(updated *entity.Permission) {
						editButton.Disable()
						revokeButton.Disable()
						deleteButton.Disable()

						go func() {
							err := acService.Update(updated)

							fyne.Do(func() {
								editButton.Enable()
								revokeButton.Enable()
								deleteButton.Enable()

								if err != nil {
									dialog.ShowError(err, window)
									return
								}

								dialog.ShowInformation(
									"Success",
									"Access control updated successfully.",
									window,
								)

								reload()
							})
						}()
					},
				)
			}

			revokeButton.OnTapped = func() {
				current := acData
				newRevoked := !current.Revoked

				action := "revoke"
				if !newRevoked {
					action = "restore"
				}

				dialog.ShowConfirm(
					"Confirm Permission Status",
					"Are you sure you want to "+
						action+
						" permission \""+
						current.Name+
						"\"?",
					func(ok bool) {
						if !ok {
							return
						}

						editButton.Disable()
						revokeButton.Disable()
						deleteButton.Disable()

						go func() {
							err := acService.SetRevoked(
								current.ID,
								newRevoked,
							)

							fyne.Do(func() {
								editButton.Enable()
								revokeButton.Enable()
								deleteButton.Enable()

								if err != nil {
									dialog.ShowError(err, window)
									return
								}

								message := "Permission revoked successfully."
								if !newRevoked {
									message = "Permission restored successfully."
								}

								dialog.ShowInformation(
									"Success",
									message,
									window,
								)

								reload()
							})
						}()
					},
					window,
				)
			}

			deleteButton.OnTapped = func() {
				current := acData

				dialog.ShowConfirm(
					"Delete Access Control",
					"Are you sure you want to permanently delete access control \""+
						current.Name+
						"\"? This action cannot be undone.",
					func(ok bool) {
						if !ok {
							return
						}

						editButton.Disable()
						revokeButton.Disable()
						deleteButton.Disable()

						go func() {
							err := acService.DeleteAccessControl(current.ID)

							fyne.Do(func() {
								editButton.Enable()
								revokeButton.Enable()
								deleteButton.Enable()

								if err != nil {
									dialog.ShowError(err, window)
									return
								}

								dialog.ShowInformation(
									"Success",
									"Access control deleted successfully.",
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
				allGroups,
				func(newPermission *entity.Permission) {
					newEntryButton.Disable()

					go func() {
						_, keyPair, err := acService.Create(newPermission)

						fyne.Do(func() {
							newEntryButton.Enable()

							if err != nil {
								dialog.ShowError(err, window)
								return
							}

							reload()

							if keyPair != nil {
								showKeyPair(
									window,
									keyPair.PublicKey,
									keyPair.PrivateKey,
								)
							}
						})
					}()
				},
			)
		},
	)

	reloadButton = widget.NewButtonWithIcon(
		"Reload",
		theme.ViewRefreshIcon(),
		func() {
			reload()
		},
	)

	manageGroupsButton := widget.NewButtonWithIcon(
		"Manage Groups",
		theme.SettingsIcon(),
		func() {
			showManageGroups()
		},
	)

	toolbarActions := container.NewHBox(
		reloadButton,
		manageGroupsButton,
		newEntryButton,
	)

	toolbar := container.NewBorder(
		nil,
		nil,
		nil,
		toolbarActions,
		search,
	)

	acContent = container.NewBorder(
		toolbar,
		nil,
		nil,
		nil,
		entryList,
	)

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
				setContent(acContent)
			},
		)

		setContent(userScreen.Content)
		entryList.Unselect(id)
	}

	search.OnChanged = func(query string) {
		if searchTimer != nil {
			searchTimer.Stop()
		}

		searchTimer = time.AfterFunc(200*time.Millisecond, func() {
			filtered := filterACL(allACL, query)

			fyne.Do(func() {
				filteredACL = filtered
				entryList.UnselectAll()
				entryList.Refresh()
			})
		})
	}

	showManageGroups = func() {
		groups, err := acService.GroupList()
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		var groupList *widget.List

		groupList = widget.NewList(
			func() int {
				return len(groups)
			},
			func() fyne.CanvasObject {
				name := widget.NewLabel("")

				edit := widget.NewButtonWithIcon(
					"",
					theme.DocumentCreateIcon(),
					nil,
				)

				remove := widget.NewButtonWithIcon(
					"",
					theme.DeleteIcon(),
					nil,
				)

				return container.NewBorder(
					nil,
					nil,
					nil,
					container.NewHBox(edit, remove),
					name,
				)
			},
			func(id widget.ListItemID, obj fyne.CanvasObject) {
				if id < 0 || id >= len(groups) {
					return
				}

				group := groups[id]
				row := obj.(*fyne.Container)

				name := row.Objects[0].(*widget.Label)
				buttons := row.Objects[1].(*fyne.Container)

				editButton := buttons.Objects[0].(*widget.Button)
				deleteButton := buttons.Objects[1].(*widget.Button)

				name.SetText(group.Name)

				editButton.OnTapped = func() {
					currentGroup := group

					nameEntry := widget.NewEntry()
					nameEntry.SetText(currentGroup.Name)
					nameEntry.SetPlaceHolder("Enter group name")

					form := container.NewVBox(
						widget.NewLabelWithStyle(
							"Group Name",
							fyne.TextAlignLeading,
							fyne.TextStyle{Bold: true},
						),
						nameEntry,
					)

					d := dialog.NewCustomConfirm(
						"Update Group Name",
						"Save",
						"Cancel",
						container.NewPadded(form),
						func(ok bool) {
							if !ok {
								return
							}

							newName := strings.TrimSpace(nameEntry.Text)
							if newName == "" {
								dialog.ShowInformation(
									"Invalid Input",
									"Group name is required.",
									window,
								)
								return
							}

							for _, existing := range groups {
								if existing.ID != currentGroup.ID &&
									strings.EqualFold(existing.Name, newName) {
									dialog.ShowInformation(
										"Duplicate Group",
										"This group name is already in use.",
										window,
									)
									return
								}
							}

							updatedGroup := currentGroup
							updatedGroup.Name = newName

							editButton.Disable()

							go func() {
								err := acService.UpdateGroup(&updatedGroup)

								fyne.Do(func() {
									editButton.Enable()

									if err != nil {
										dialog.ShowError(err, window)
										return
									}

									dialog.ShowInformation(
										"Success",
										"Permission group updated successfully.",
										window,
									)

									showManageGroups()
									reload()
								})
							}()
						},
						window,
					)

					d.Resize(fyne.NewSize(550, 220))
					d.Show()
				}

				deleteButton.OnTapped = func() {
					currentGroup := group

					dialog.ShowConfirm(
						"Delete Permission Group",
						"Are you sure you want to delete permission group \""+
							currentGroup.Name+
							"\"? Groups that are still in use may not be deletable.",
						func(ok bool) {
							if !ok {
								return
							}

							deleteButton.Disable()

							go func() {
								err := acService.DeleteGroup(currentGroup.ID)

								fyne.Do(func() {
									deleteButton.Enable()

									if err != nil {
										dialog.ShowError(err, window)
										return
									}

									dialog.ShowInformation(
										"Success",
										"Permission group deleted successfully.",
										window,
									)

									showManageGroups()
									reload()
								})
							}()
						},
						window,
					)
				}
			},
		)

		d := dialog.NewCustom(
			"Manage Permission Groups",
			"Close",
			container.NewGridWrap(
				fyne.NewSize(600, 400),
				groupList,
			),
			window,
		)

		d.Resize(fyne.NewSize(650, 450))
		d.Show()
	}

	reload()

	return &Screen{
		Content: content,
		Reload:  reload,
	}
}

func showForm(
	window fyne.Window,
	selectedPermission *entity.Permission,
	groups []entity.PermissionGroup,
	onSave func(permission *entity.Permission),
) {
	isEdit := selectedPermission != nil

	var permission entity.Permission
	if isEdit {
		permission = *selectedPermission
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetText(permission.Name)
	nameEntry.SetPlaceHolder("Permission name...")

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

	groupOptions := make([]string, 0, len(groups))
	groupByName := make(map[string]entity.PermissionGroup, len(groups))

	for _, group := range groups {
		groupOptions = append(groupOptions, group.Name)
		groupByName[strings.ToLower(strings.TrimSpace(group.Name))] = group
	}

	groupEntry := widget.NewSelectEntry(groupOptions)
	groupEntry.SetPlaceHolder("Enter or select a permission group")

	if isEdit {
		if group, exists := groupByName[strings.ToLower(strings.TrimSpace(permission.Group.Name))]; exists {
			groupEntry.SetText(group.Name)
		} else if permission.Group.Name != "" {
			groupEntry.SetText(permission.Group.Name)
		}
	}

	if groupEntry.Text == "" && len(groups) == 1 {
		groupEntry.SetText(groups[0].Name)
	}

	autoApprove := permission.AutoApprove
	autoApproveButton := widget.NewButton("", nil)

	updateAutoApproveButton := func() {
		if autoApprove {
			autoApproveButton.SetText("Auto Approval: ON")
		} else {
			autoApproveButton.SetText("Auto Approval: OFF")
		}
	}

	autoApproveButton.OnTapped = func() {
		autoApprove = !autoApprove
		updateAutoApproveButton()
	}

	updateAutoApproveButton()

	form := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Privilege", privilegeRadio),
		widget.NewFormItem("Group", groupEntry),
		widget.NewFormItem("Auto Approval", autoApproveButton),
	)

	title := "New Access Control"
	if isEdit {
		title = "Edit Access Control"
	}

	d := dialog.NewCustomConfirm(
		title,
		"Save",
		"Cancel",
		container.NewScroll(container.NewVBox(form)),
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

			if privilegeRadio.Selected == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"Privilege is required.",
					window,
				)
				return
			}

			groupName := strings.TrimSpace(groupEntry.Text)
			if groupName == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"Permission group is required.",
					window,
				)
				return
			}

			key := strings.ToLower(groupName)

			permission.Name = name
			permission.Privilege = entity.Privilege(privilegeRadio.Selected)
			permission.AutoApprove = autoApprove

			if group, exists := groupByName[key]; exists {
				permission.Group = group
				permission.GroupID = group.ID
				permission.SotID = group.SotID
			} else {
				permission.Group = entity.PermissionGroup{
					Name: groupName,
				}
				permission.GroupID = ""
				permission.SotID = ""
			}

			if onSave != nil {
				onSave(&permission)
			}
		},
		window,
	)

	d.Resize(fyne.NewSize(700, 500))
	d.Show()
}

func showKeyPair(
	window fyne.Window,
	publicKey string,
	privateKey string,
) {
	secretKey, err := util.EncodeSecretKey(publicKey, privateKey)
	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	secretEntry := widget.NewMultiLineEntry()
	secretEntry.SetText(secretKey)
	secretEntry.SetMinRowsVisible(8)
	secretEntry.Wrapping = fyne.TextWrapBreak

	copyButton := widget.NewButtonWithIcon(
		"Copy Secret Key",
		theme.ContentCopyIcon(),
		func() {
			window.Clipboard().SetContent(secretKey)

			dialog.ShowInformation(
				"Copied",
				"Secret key copied to clipboard.",
				window,
			)
		},
	)

	content := container.NewVBox(
		widget.NewLabelWithStyle(
			"Secret Key (Base64)",
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		),
		secretEntry,
		widget.NewLabel(
			"The secret key contains the public and private keys in encoded format. Store it securely.",
		),
		copyButton,
	)

	dialog.ShowCustom(
		"Access Control Secret Key",
		"Close",
		content,
		window,
	)
}
