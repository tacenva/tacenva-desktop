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
	"github.com/tacenva/tacenva-core/entity"
)

// Dummy data menggunakan struktur kontrak yang sudah ada.
// Tidak ada pemanggilan service atau penyimpanan data.
func dummyAccessGroupTrees() []*contract.AccessGroupTree {
	backend := entity.AccessGroup{
		ID:   "01BACKEND00000000000000001",
		Name: "Backend",
		AccessUnits: []entity.AccessUnit{
			{
				ID:        "01UNITBACKEND000000000001",
				Name:      "Backend API",
				GroupID:   "01BACKEND00000000000000001",
				Level:     entity.LevelAdmin,
				PublicKey: "",
			},
			{
				ID:        "01UNITBACKEND000000000002",
				Name:      "Database",
				GroupID:   "01BACKEND00000000000000001",
				Level:     entity.LevelWrite,
				PublicKey: "",
			},
		},
	}

	frontend := entity.AccessGroup{
		ID:   "01FRONTEND0000000000000001",
		Name: "Frontend",
		AccessUnits: []entity.AccessUnit{
			{
				ID:        "01UNITFRONTEND0000000001",
				Name:      "Frontend Dashboard",
				GroupID:   "01FRONTEND0000000000000001",
				Level:     entity.LevelRead,
				PublicKey: "",
			},
		},
	}

	infrastructure := entity.AccessGroup{
		ID:   "01INFRA000000000000000001",
		Name: "Infrastructure",
		AccessUnits: []entity.AccessUnit{
			{
				ID:        "01UNITINFRA00000000000001",
				Name:      "Server Management",
				GroupID:   "01INFRA000000000000000001",
				Level:     entity.LevelAdmin,
				PublicKey: "",
			},
		},
	}

	engineering := entity.AccessGroup{
		ID:   "01ENGINEERING0000000000001",
		Name: "Engineering",
		AccessUnits: []entity.AccessUnit{
			{
				ID:        "01UNITENGINEERING00000001",
				Name:      "Source Code",
				GroupID:   "01ENGINEERING0000000000001",
				Level:     entity.LevelWrite,
				PublicKey: "",
			},
		},
	}

	operations := entity.AccessGroup{
		ID:   "01OPERATIONS00000000000001",
		Name: "Operations",
		AccessUnits: []entity.AccessUnit{
			{
				ID:        "01UNITOPERATIONS00000001",
				Name:      "Deployment",
				GroupID:   "01OPERATIONS00000000000001",
				Level:     entity.LevelAdmin,
				PublicKey: "",
			},
		},
	}

	// Simulasi jumlah unit dan user.
	backendTree := &contract.AccessGroupTree{
		Group:           backend,
		AccessUnitCount: 2,
		UserCount:       8,
	}

	frontendTree := &contract.AccessGroupTree{
		Group:           frontend,
		AccessUnitCount: 1,
		UserCount:       5,
	}

	infrastructureTree := &contract.AccessGroupTree{
		Group:           infrastructure,
		AccessUnitCount: 1,
		UserCount:       3,
	}

	engineeringTree := &contract.AccessGroupTree{
		Group:           engineering,
		AccessUnitCount: 1,
		UserCount:       12,
		Children: []contract.AccessGroupHierarchyWithCount{
			{
				AccessGroupHierarchy: entity.AccessGroupHierarchy{
					ParentGroupID: engineering.ID,
					ChildGroupID:  backend.ID,
					Child:         backend,
				},
				AccessUnitCount: 2,
				UserCount:       8,
			},
			{
				AccessGroupHierarchy: entity.AccessGroupHierarchy{
					ParentGroupID: engineering.ID,
					ChildGroupID:  frontend.ID,
					Child:         frontend,
				},
				AccessUnitCount: 1,
				UserCount:       5,
			},
		},
	}

	operationsTree := &contract.AccessGroupTree{
		Group:           operations,
		AccessUnitCount: 1,
		UserCount:       6,
		Children: []contract.AccessGroupHierarchyWithCount{
			{
				AccessGroupHierarchy: entity.AccessGroupHierarchy{
					ParentGroupID: operations.ID,
					ChildGroupID:  infrastructure.ID,
					Child:         infrastructure,
				},
				AccessUnitCount: 1,
				UserCount:       3,
			},
		},
	}

	// Semua group tersedia agar navigasi ke subgroup bisa menemukan tree-nya.
	// Engineering menjadi root awal.
	return []*contract.AccessGroupTree{
		engineeringTree,
		backendTree,
		frontendTree,
		operationsTree,
		infrastructureTree,
	}
}

func newAccessGroupCard(
	group *contract.AccessGroupTree,
	onExplore func(),
	onEdit func(string),
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
				onEdit(group.Group.Name)
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
		2,
		newGroupStatCard(
			"Access Units",
			fmt.Sprintf("%d", group.AccessUnitCount),
		),
		newGroupStatCard(
			"Users",
			fmt.Sprintf("%d", group.UserCount),
		),
	)

	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		stats,
	)

	card := newWhiteBorder(container.NewPadded(content))

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

	return newWhiteBorder(
		container.NewPadded(
			container.NewVBox(
				labelObject,
				valueObject,
			),
		),
	)
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
	onEdit func(string),
	onNew func(),
	onDelete func(string),
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
	onEdit func(string),
	onDelete func(string),
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
				onEdit(group.Group.Name)
			}
		},
	)

	deleteButton := widget.NewButtonWithIcon(
		"",
		theme.DeleteIcon(),
		func() {
			if onDelete != nil {
				onDelete(group.Group.Name)
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
		2,
		newCompactStatCard(
			"Access Units",
			fmt.Sprintf("%d", group.AccessUnitCount),
		),
		newCompactStatCard(
			"Users",
			fmt.Sprintf("%d", group.UserCount),
		),
	)

	return newWhiteBorder(
		container.NewPadded(
			container.NewVBox(
				header,
				stats,
			),
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
