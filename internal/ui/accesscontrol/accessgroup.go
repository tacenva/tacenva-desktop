package accesscontrol

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-core/contract"
)

func newAccessGroupCard(
	group *contract.AccessGroupTree,
	onExplore func(),
	onEdit func(*contract.AccessGroupTree),
) fyne.CanvasObject {
	if group == nil {
		return widget.NewLabel("No access group selected")
	}

	title := widget.NewLabelWithStyle(
		group.Group.Name,
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	editButton := widget.NewButtonWithIcon(
		"Edit",
		theme.DocumentCreateIcon(),
		func() {
			if onEdit != nil {
				onEdit(group)
			}
		},
	)

	exploreButton := widget.NewButtonWithIcon(
		"Explore Subgroups",
		theme.FolderOpenIcon(),
		func() {
			if onExplore != nil {
				onExplore()
			}
		},
	)

	actions := container.NewHBox(editButton, exploreButton)

	header := container.NewBorder(
		nil,
		nil,
		title,
		actions,
		nil,
	)

	stats := container.NewGridWithColumns(
		3,
		newGroupStatCard(
			"Access Units",
			fmt.Sprintf("%d", group.AccessUnitCount),
		),
		newGroupStatCard(
			"Users",
			fmt.Sprintf("%d", group.UserCount),
		),
		newGroupStatCard(
			"Pending",
			fmt.Sprintf("%d", group.PendingUserCount),
		),
	)

	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		stats,
	)

	card := container.NewPadded(content)

	return container.NewHBox(
		container.New(
			layout.NewGridWrapLayout(
				fyne.NewSize(680, card.MinSize().Height),
			),
			card,
		),
	)
}

func newGroupStatCard(label, value string) fyne.CanvasObject {
	labelObject := widget.NewLabel(label)

	valueObject := canvas.NewText(value, theme.ForegroundColor())
	valueObject.TextSize = 30
	valueObject.TextStyle = fyne.TextStyle{
		Bold: true,
	}

	return container.NewPadded(newWhiteBorder(
		container.NewPadded(
			container.NewVBox(
				labelObject,
				valueObject,
			),
		),
	))
}

func newWhiteBorder(content fyne.CanvasObject) fyne.CanvasObject {
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = color.White
	border.StrokeWidth = 1
	border.CornerRadius = 8

	return container.NewStack(
		border,
		container.NewPadded(content),
	)
}

func newAccessGroupPanel(
	group *contract.AccessGroupTree,
	onSelect func(*contract.AccessGroupTree),
	onEdit func(*contract.AccessGroupTree),
	onNew func(),
	onDelete func(*contract.AccessGroupTree),
	onClose func(),
) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		"Subgroups",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	newButton := widget.NewButtonWithIcon(
		"New",
		theme.ContentAddIcon(),
		func() {
			if onNew != nil {
				onNew()
			}
		},
	)

	closeButton := widget.NewButtonWithIcon(
		"",
		theme.CancelIcon(),
		func() {
			if onClose != nil {
				onClose()
			}
		},
	)

	header := container.NewBorder(
		nil,
		nil,
		title,
		container.NewHBox(newButton, closeButton),
		nil,
	)

	subgroupList := container.NewVBox()

	if group == nil || len(group.Children) == 0 {
		subgroupList.Add(
			container.NewCenter(
				widget.NewLabel("No subgroups available."),
			),
		)
	} else {
		for _, child := range group.Children {
			childGroup := child.Child

			subgroupTree := &contract.AccessGroupTree{
				Group:           childGroup,
				AccessUnitCount: child.AccessUnitCount,
				UserCount:       child.UserCount,
			}

			subgroupList.Add(
				newCompactAccessGroupCard(
					subgroupTree,
					func(selected *contract.AccessGroupTree) {
						if onSelect != nil {
							onSelect(selected)
						}
					},
					onEdit,
					onDelete,
				),
			)
		}
	}

	return container.NewBorder(
		header,
		nil,
		nil,
		nil,
		container.NewPadded(
			container.NewScroll(subgroupList),
		),
	)
}

func newCompactAccessGroupCard(
	group *contract.AccessGroupTree,
	onSelect func(*contract.AccessGroupTree),
	onEdit func(*contract.AccessGroupTree),
	onDelete func(*contract.AccessGroupTree),
) fyne.CanvasObject {
	if group == nil {
		return widget.NewLabel("Invalid subgroup")
	}

	title := widget.NewLabelWithStyle(
		group.Group.Name,
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	editButton := widget.NewButtonWithIcon(
		"",
		theme.DocumentCreateIcon(),
		func() {
			if onEdit != nil {
				onEdit(group)
			}
		},
	)

	deleteButton := widget.NewButtonWithIcon(
		"",
		theme.DeleteIcon(),
		func() {
			if onDelete != nil {
				onDelete(group)
			}
		},
	)

	exploreButton := widget.NewButtonWithIcon(
		"",
		theme.FolderOpenIcon(),
		func() {
			if onSelect != nil {
				onSelect(group)
			}
		},
	)

	header := container.NewBorder(
		nil,
		nil,
		title,
		container.NewHBox(
			deleteButton,
			editButton,
			exploreButton,
		),
		nil,
	)

	stats := container.NewGridWithColumns(
		3,
		newCompactStatCard(
			"Units",
			fmt.Sprintf("%d", group.AccessUnitCount),
		),
		newCompactStatCard(
			"Users",
			fmt.Sprintf("%d", group.UserCount),
		),
		newCompactStatCard(
			"Pending",
			fmt.Sprintf("%d", group.PendingUserCount),
		),
	)

	return container.NewPadded(
		container.NewVBox(
			header,
			stats,
		),
	)
}

func newCompactStatCard(label, value string) fyne.CanvasObject {
	labelObject := widget.NewLabel(label)

	valueObject := widget.NewLabelWithStyle(
		value,
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	return container.NewVBox(
		labelObject,
		valueObject,
	)
}
