package accesscontrol

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-core/contract"
	"github.com/tacenva/tacenva-core/entity"
)

type AccessUnitSaveCallback func(
	unit *entity.AccessUnit,
	onSuccess func(),
	onError func(error),
)

type AccessUnitRevokeCallback func(
	unit *entity.AccessUnit,
	revoked bool,
)

func newAccessUnitSection(
	units []contract.AccessUnitWithUserCount,
	onViewUsers func(contract.AccessUnitWithUserCount),
	onSave AccessUnitSaveCallback,
	onDelete func(*entity.AccessUnit),
	onRevoke AccessUnitRevokeCallback,
) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		"Access Units",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	unitList := container.NewVBox()

	createButton := widget.NewButtonWithIcon(
		"Create",
		theme.ContentAddIcon(),
		func() {
			showAccessUnitDialog(nil, onSave)
		},
	)

	header := container.NewBorder(
		nil,
		nil,
		title,
		createButton,
		nil,
	)

	if len(units) == 0 {
		unitList.Add(container.NewCenter(
			widget.NewLabel("No access units available."),
		))
	}

	for _, unit := range units {
		currentUnit := unit

		unitName := currentUnit.Name
		if unitName == "" {
			unitName = "Unnamed Access Unit"
		}

		nameLabel := widget.NewLabelWithStyle(
			unitName,
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		)

		statusColor := theme.SuccessColor()
		if currentUnit.Revoked {
			statusColor = theme.ErrorColor()
		}

		statusIndicator := canvas.NewCircle(statusColor)
		statusIndicator.Resize(fyne.NewSize(8, 8))

		nameWithStatus := container.NewHBox(
			nameLabel,
			container.NewCenter(
				container.NewGridWrap(
					fyne.NewSize(8, 8),
					statusIndicator,
				),
			),
		)

		metadata := widget.NewLabel(
			fmt.Sprintf(
				"Permission: %s  •  Users: %d",
				currentUnit.Level,
				currentUnit.UserCount,
			),
		)

		// Three-dot menu
		menuButton := widget.NewButtonWithIcon(
			"",
			theme.MoreVerticalIcon(),
			nil,
		)

		menu := fyne.NewMenu("",
			fyne.NewMenuItem("Update", func() {
				showAccessUnitDialog(
					&currentUnit,
					onSave,
				)
			}),
			fyne.NewMenuItem("Delete", func() {
				if onDelete != nil {
					showDeleteAccessUnitDialog(
						&currentUnit.AccessUnit,
						onDelete,
					)
				}
			}),
		)

		menu.Items[0].Icon = theme.DocumentCreateIcon()
		menu.Items[1].Icon = theme.DeleteIcon()

		menuButton.OnTapped = func() {
			canvas := fyne.CurrentApp().Driver().CanvasForObject(menuButton)

			widget.ShowPopUpMenuAtRelativePosition(
				menu,
				canvas,
				fyne.NewPos(-95, 0),
				menuButton,
			)
		}

		menuButton.Importance = widget.LowImportance

		// Header
		cardHeader := container.NewBorder(
			nil,
			nil,
			nameWithStatus,
			menuButton,
			nil,
		)

		// List Users button
		var revokeButton *widget.Button
		if currentUnit.Revoked {
			revokeButton = widget.NewButtonWithIcon(
				"  Reactivate",
				theme.VisibilityIcon(),
				func() {
					if onRevoke != nil {
						showRevokeAccessUnitDialog(
							&currentUnit.AccessUnit,
							false,
							onRevoke,
						)
					}
				},
			)
		} else {
			revokeButton = widget.NewButtonWithIcon(
				"  Revoke",
				theme.VisibilityOffIcon(),
				func() {
					if onRevoke != nil {
						showRevokeAccessUnitDialog(
							&currentUnit.AccessUnit,
							true,
							onRevoke,
						)
					}
				},
			)
		}

		revokeButton.Importance = widget.MediumImportance

		viewUsersButton := widget.NewButtonWithIcon(
			"List Users  ",
			theme.NavigateNextIcon(),
			func() {
				if onViewUsers != nil {
					onViewUsers(currentUnit)
				}
			},
		)

		viewUsersButton.IconPlacement = widget.ButtonIconTrailingText

		// Bottom-right action
		cardFooter := container.NewHBox(
			revokeButton,
			layout.NewSpacer(),
			viewUsersButton,
		)

		cardContent := container.NewVBox(
			cardHeader,
			metadata,
			container.NewGridWrap(
				fyne.NewSize(1, 10),
				canvas.NewRectangle(color.Transparent),
			),
			cardFooter,
		)

		unitList.Add(
			widget.NewCard(
				"",
				"",
				container.NewPadded(cardContent),
			),
		)
	}

	return container.NewVBox(
		header,
		widget.NewSeparator(),
		unitList,
	)
}

func showAccessUnitDialog(
	unit *contract.AccessUnitWithUserCount,
	onSave AccessUnitSaveCallback,
) {
	windows := fyne.CurrentApp().Driver().AllWindows()
	if len(windows) == 0 {
		return
	}

	window := windows[0]

	isUpdate := unit != nil
	title := "Create Access Unit"
	saveLabel := "Create"

	var accessUnit *entity.AccessUnit

	if isUpdate {
		title = "Update Access Unit"
		saveLabel = "Save"

		accessUnitCopy := unit.AccessUnit
		accessUnit = &accessUnitCopy
	} else {
		accessUnit = &entity.AccessUnit{
			Level:       entity.LevelRead,
			AutoApprove: false,
		}
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Enter access unit name")
	nameEntry.SetText(accessUnit.Name)

	levelSelect := widget.NewSelect(
		[]string{
			string(entity.LevelAdmin),
			string(entity.LevelWrite),
			string(entity.LevelRead),
		},
		nil,
	)

	levelSelect.SetSelected(string(accessUnit.Level))

	autoApproveCheck := widget.NewCheck(
		"Automatically approve users",
		nil,
	)

	autoApproveCheck.SetChecked(accessUnit.AutoApprove)

	var formDialog *dialog.FormDialog

	formDialog = dialog.NewForm(
		title,
		saveLabel,
		"Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Permission", levelSelect),
			widget.NewFormItem("Auto Approve", autoApproveCheck),
		},
		func(confirmed bool) {
			if !confirmed {
				return
			}

			name := strings.TrimSpace(nameEntry.Text)

			if name == "" {
				dialog.ShowInformation(
					"Validation Error",
					"Access unit name is required.",
					window,
				)
				return
			}

			if levelSelect.Selected == "" {
				dialog.ShowInformation(
					"Validation Error",
					"Permission is required.",
					window,
				)
				return
			}

			if onSave == nil {
				dialog.ShowError(
					fmt.Errorf(
						"fungsi penyimpanan Access Unit belum tersedia",
					),
					window,
				)
				return
			}

			accessUnit.Name = name
			accessUnit.Level = entity.AccessLevel(
				levelSelect.Selected,
			)
			accessUnit.AutoApprove = autoApproveCheck.Checked

			onSave(
				accessUnit,
				func() {
					formDialog.Hide()
				},
				func(err error) {
					dialog.ShowError(err, window)
				},
			)
		},
		window,
	)

	formDialog.Show()
}

func showDeleteAccessUnitDialog(
	unit *entity.AccessUnit,
	onDelete func(*entity.AccessUnit),
) {
	windows := fyne.CurrentApp().Driver().AllWindows()
	if len(windows) == 0 {
		return
	}

	window := windows[0]

	dialog.ShowConfirm(
		"Delete Access Unit",
		fmt.Sprintf(
			"Are you sure you want to delete access unit %q?",
			unit.Name,
		),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			if onDelete == nil {
				dialog.ShowInformation(
					"Error",
					"Delete handler is not configured.",
					window,
				)
				return
			}

			onDelete(unit)
		},
		window,
	)
}

func showRevokeAccessUnitDialog(
	unit *entity.AccessUnit,
	revoked bool,
	onRevoke AccessUnitRevokeCallback,
) {
	windows := fyne.CurrentApp().Driver().AllWindows()
	if len(windows) == 0 {
		return
	}

	window := windows[0]

	action := "revoke"
	actionTitle := "Revoke Access Unit"

	if !revoked {
		action = "reactivate"
		actionTitle = "Reactivate Access Unit"
	}

	dialog.ShowConfirm(
		actionTitle,
		fmt.Sprintf(
			"Are you sure you want to %s access unit %q?",
			action,
			unit.Name,
		),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			if onRevoke == nil {
				dialog.ShowInformation(
					"Error",
					"Revoke handler is not configured.",
					window,
				)
				return
			}

			onRevoke(unit, revoked)
		},
		window,
	)
}
