package accesscontrol

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-core/contract"
	"github.com/tacenva/tacenva-services/app"
	coreApp "github.com/tacenva/tacpass-core/app"
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
	// Dependencies belum dipakai karena data dan aksi masih simulasi.
	_, _, _ = appDeps, context, coreService

	groups := dummyAccessGroupTrees()

	var selectedGroup *contract.AccessGroupTree
	var navigationStack []*contract.AccessGroupTree
	var render func()
	var selectGroup func(*contract.AccessGroupTree)

	panelOpen := false

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
	panelContainer := container.NewMax()
	root := container.NewMax()

	backButton := widget.NewButtonWithIcon(
		"Back",
		theme.NavigateBackIcon(),
		func() {
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
				container.NewGridWrap(
					fyne.NewSize(1, 12),
					widget.NewLabel(""),
				),
				groupContainer,
				container.NewGridWrap(
					fyne.NewSize(1, 24),
					widget.NewLabel(""),
				),
				unitContainer,
			),
		),
	)

	updateLayout := func() {
		if panelOpen {
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

	// Placeholder saja. Belum menyimpan atau mengubah data.
	showEditPlaceholder := func(groupName string) {
		dialog.ShowInformation(
			"Success",
			fmt.Sprintf(
				"Access group \"%s\" updated successfully.",
				groupName,
			),
			window,
		)
	}

	showDeletePlaceholder := func(groupName string) {
		dialog.ShowConfirm(
			"Delete Subgroup",
			fmt.Sprintf(
				"Are you sure you want to delete subgroup \"%s\"?",
				groupName,
			),
			func(confirmed bool) {
				if !confirmed {
					return
				}

				dialog.ShowInformation(
					"Success",
					fmt.Sprintf(
						"Subgroup \"%s\" deleted successfully.",
						groupName,
					),
					window,
				)
			},
			window,
		)
	}

	// Placeholder saja. Belum membuat subgroup sungguhan.
	showNewPlaceholder := func() {
		dialog.ShowInformation(
			"Success",
			"Subgroup created successfully.",
			window,
		)
	}

	render = func() {
		if selectedGroup == nil {
			return
		}

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
			),
		)
		unitContainer.Refresh()

		if len(navigationStack) > 1 {
			parent := navigationStack[len(navigationStack)-2]
			breadcrumb.SetText(
				fmt.Sprintf(
					"%s  /  %s",
					parent.Group.Name,
					selectedGroup.Group.Name,
				),
			)
			backButton.Enable()
		} else {
			breadcrumb.SetText(selectedGroup.Group.Name)
			backButton.Disable()
		}

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

		// Jika group sudah ada di jalur navigasi, kembali ke posisi tersebut.
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
	}
}
