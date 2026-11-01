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
	disabled bool
}

func newNavItem(
	menu menuItem,
	text string,
	icon fyne.Resource,
	disabled bool,
	onTap func(),
) *navItem {
	item := &navItem{
		menu:  menu,
		icon:  widget.NewIcon(icon),
		label: widget.NewLabel(text),
		bg: canvas.NewRectangle(
			theme.Color(theme.ColorNameBackground),
		),
		onTap:    onTap,
		disabled: disabled,
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
	if item.disabled || item.onTap == nil {
		return
	}

	item.onTap()
}

func (item *navItem) MouseIn(_ *desktop.MouseEvent) {
	if item.disabled {
		return
	}

	item.hover = true
	item.refreshBackground()
}

func (item *navItem) MouseOut() {
	item.hover = false
	item.refreshBackground()
}

func (item *navItem) SetSelected(selected bool) {
	if item.disabled {
		return
	}

	item.selected = selected
	item.refreshBackground()
}

func (item *navItem) MouseMoved(_ *desktop.MouseEvent) {
}

func (item *navItem) SetDisabled(disabled bool) {
	item.disabled = disabled

	if disabled {
		item.hover = false
		item.selected = false
	}

	item.refreshBackground()
	item.Refresh()
}

func (item *navItem) refreshBackground() {
	if item.disabled {
		item.bg.FillColor = theme.Color(theme.ColorNameBackground)
	} else if item.selected || item.hover {
		item.bg.FillColor = theme.Color(theme.ColorNameHover)
	} else {
		item.bg.FillColor = theme.Color(theme.ColorNameBackground)
	}

	item.label.Refresh()
	item.icon.Refresh()
	item.bg.Refresh()
}

func newSidebar(
	vaultMenu func(),
	accessControlMenu func(),
	settingsMenu func(),
) (fyne.CanvasObject, *navItem) {
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
		if item == nil || item.disabled {
			return
		}

		var onSelect func()

		switch item.menu {
		case menuVault:
			onSelect = vaultMenu
		case menuAccessControl:
			onSelect = accessControlMenu
		case menuSettings:
			onSelect = settingsMenu
		default:
			return
		}

		if onSelect == nil {
			return
		}

		onSelect()

		if selectedItem != nil {
			selectedItem.SetSelected(false)
		}

		selectedItem = item
		selectedItem.SetSelected(true)
	}

	vault = newNavItem(
		menuVault,
		"Vault",
		theme.ListIcon(),
		false,
		func() {
			selectItem(vault)
		},
	)

	accessControl = newNavItem(
		menuAccessControl,
		"Access Control",
		theme.AccountIcon(),
		accessControlMenu == nil,
		func() {
			selectItem(accessControl)
		},
	)

	settings = newNavItem(
		menuSettings,
		"Settings",
		theme.SettingsIcon(),
		false,
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

	return container.NewPadded(content), accessControl
}
