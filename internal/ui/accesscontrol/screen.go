package accesscontrol

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-core/contract"
	"github.com/tacenva/tacenva-core/entity"
	"github.com/tacenva/tacenva-services/app"
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
	// Dependencies belum digunakan karena data masih simulasi.
	_, _, _ = appDeps, context, coreService

	groups := dummyAccessGroupTrees()

	var selectedGroup *contract.AccessGroupTree
	var navigationStack []*contract.AccessGroupTree
	var selectedUnit *entity.AccessUnit

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

	breadcrumb := widget.NewLabel("Access Groups")

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
			selectedGroup = navigationStack[len(navigationStack)-1]

			render()
		},
	)

	reloadButton := widget.NewButtonWithIcon(
		"Reload",
		theme.ViewRefreshIcon(),
		func() {
			render()
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
		breadcrumb,
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

	showEditPlaceholder := func(groupName string) {
		dialog.ShowInformation(
			"Success",
			fmt.Sprintf(
				"Access group %q updated successfully.",
				groupName,
			),
			window,
		)
	}

	showDeletePlaceholder := func(groupName string) {
		dialog.ShowConfirm(
			"Delete Subgroup",
			fmt.Sprintf(
				"Are you sure you want to delete subgroup %q?",
				groupName,
			),
			func(confirmed bool) {
				if !confirmed {
					return
				}

				dialog.ShowInformation(
					"Success",
					fmt.Sprintf(
						"Subgroup %q deleted successfully.",
						groupName,
					),
					window,
				)
			},
			window,
		)
	}

	showNewPlaceholder := func() {
		dialog.ShowInformation(
			"Success",
			"Subgroup created successfully.",
			window,
		)
	}

	// Membuka halaman Users di area konten utama, bukan dialog.
	showUnitUsers := func(unit entity.AccessUnit) {
		currentUnit := unit
		selectedUnit = &currentUnit
		showingUsers = true

		// Tutup panel subgroup ketika pindah halaman.
		panelOpen = false

		render()
	}

	render = func() {
		if selectedGroup == nil {
			return
		}

		contentBody.RemoveAll()

		if showingUsers && selectedUnit != nil {
			backButton.Enable()
			breadcrumb.SetText("")

			levelTag := widget.NewLabel(
				fmt.Sprintf("Level: %s", selectedUnit.Level),
			)

			usersTag := widget.NewLabel(
				fmt.Sprintf("Users: %d", len(selectedUnit.Users)),
			)

			statusText := "Active"
			if selectedUnit.Revoked {
				statusText = "Revoked"
			}

			statusTag := widget.NewLabel(
				fmt.Sprintf("Status: %s", statusText),
			)

			unitInfo := container.NewVBox(
				widget.NewLabelWithStyle(
					selectedUnit.Name,
					fyne.TextAlignLeading,
					fyne.TextStyle{Bold: true},
				),
				container.NewHBox(
					levelTag,
					usersTag,
					statusTag,
				),
			)

			unitInfoCard := newWhiteBorder(
				container.NewPadded(unitInfo),
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
				selectedGroup,
				openPanel,
				showEditPlaceholder,
			),
		)
		groupContainer.Refresh()

		unitContainer.RemoveAll()
		unitContainer.Add(
			newAccessUnitSection(
				&selectedGroup.Group,
				selectedGroup.AccessUnitCount,
				showUnitUsers,
			),
		)
		unitContainer.Refresh()

		if len(navigationStack) > 1 {
			parent := navigationStack[len(navigationStack)-2]

			breadcrumb.SetText(
				fmt.Sprintf(
					"%s / %s",
					parent.Group.Name,
					selectedGroup.Group.Name,
				),
			)

			backButton.Enable()
		} else {
			breadcrumb.SetText(selectedGroup.Group.Name)
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
					selectedGroup,
					func(group *contract.AccessGroupTree) {
						selectGroup(group)
					},
					showEditPlaceholder,
					showNewPlaceholder,
					showDeletePlaceholder,
					func() {
						panelOpen = false
						updateLayout()
					},
				),
			}
			panelContainer.Refresh()
		}

		breadcrumb.Refresh()
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
				selectedGroup = existing
				render()
				return
			}
		}

		selectedGroup = group
		navigationStack = append(navigationStack, group)

		render()
	}

	if len(groups) > 0 {
		selectGroup(groups[0])
	}

	return &Screen{
		Content: root,
		Reload: func() {
			render()
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
