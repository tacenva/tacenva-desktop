package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type menuItem string

const (
	menuVault         menuItem = "vault"
	menuAccessControl menuItem = "access_control"
	menuSettings      menuItem = "settings"
)

type navItem struct {
	widget.BaseWidget

	menu     menuItem
	icon     *widget.Icon
	label    *widget.Label
	bg       *canvas.Rectangle
	onTap    func()
	hover    bool
	selected bool
}

func newNavItem(
	menu menuItem,
	text string,
	icon fyne.Resource,
	onTap func(),
) *navItem {
	item := &navItem{
		menu:  menu,
		icon:  widget.NewIcon(icon),
		label: widget.NewLabel(text),
		bg: canvas.NewRectangle(
			theme.Color(theme.ColorNameBackground),
		),
		onTap: onTap,
	}

	item.ExtendBaseWidget(item)

	return item
}

func (item *navItem) CreateRenderer() fyne.WidgetRenderer {
	content := container.NewHBox(
		item.icon,
		item.label,
	)

	content = container.NewPadded(content)

	return widget.NewSimpleRenderer(
		container.NewMax(
			item.bg,
			content,
		),
	)
}

func (item *navItem) Tapped(_ *fyne.PointEvent) {
	if item.onTap != nil {
		item.onTap()
	}
}

func (item *navItem) MouseIn(_ *desktop.MouseEvent) {
	item.hover = true
	item.refreshBackground()
}

func (item *navItem) MouseOut() {
	item.hover = false
	item.refreshBackground()
}

func (item *navItem) MouseMoved(_ *desktop.MouseEvent) {
}

func (item *navItem) SetSelected(selected bool) {
	item.selected = selected
	item.refreshBackground()
}

func (item *navItem) refreshBackground() {
	if item.selected || item.hover {
		item.bg.FillColor = theme.Color(theme.ColorNameHover)
	} else {
		item.bg.FillColor = theme.Color(theme.ColorNameBackground)
	}

	item.bg.Refresh()
}

func newSidebar(
	vaultMenu func(),
	accessControlMenu func(),
	settingsMenu func(),
) fyne.CanvasObject {
	logo := widget.NewLabelWithStyle(
		"Menu",
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)

	var selectedItem *navItem

	var vault *navItem
	var accessControl *navItem
	var settings *navItem

	selectItem := func(item *navItem) {
		if selectedItem != nil {
			selectedItem.SetSelected(false)
		}

		selectedItem = item
		selectedItem.SetSelected(true)

		switch item.menu {
		case menuVault:
			vaultMenu()

		case menuAccessControl:
			accessControlMenu()

		case menuSettings:
			settingsMenu()
		}
	}

	vault = newNavItem(
		menuVault,
		"Vault",
		theme.ListIcon(),
		func() {
			selectItem(vault)
		},
	)

	accessControl = newNavItem(
		menuAccessControl,
		"Access Control",
		theme.AccountIcon(),
		func() {
			selectItem(accessControl)
		},
	)

	settings = newNavItem(
		menuSettings,
		"Settings",
		theme.SettingsIcon(),
		func() {
			selectItem(settings)
		},
	)

	menu := container.NewVBox(
		vault,
		accessControl,
	)

	top := container.NewVBox(
		logo,
		widget.NewSeparator(),
		menu,
	)

	content := container.NewBorder(
		top,
		settings,
		nil,
		nil,
		layout.NewSpacer(),
	)

	// Vault menjadi menu default.
	selectItem(vault)

	return container.NewPadded(content)
}
