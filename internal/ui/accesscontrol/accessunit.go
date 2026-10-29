package accesscontrol

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-core/contract"
	"github.com/tacenva/tacenva-core/entity"
)

func newAccessUnitSection(
	units []contract.AccessUnitWithUserCount,
	onViewUsers func(contract.AccessUnitWithUserCount),
	onSave func(*entity.AccessUnit) error,
	onDelete func(*entity.AccessUnit),
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
		unitList.Add(
			container.NewCenter(
				widget.NewLabel(
					"No access units assigned to this group.",
				),
			),
		)
	} else {
		for _, unit := range units {
			currentUnit := unit

			unitName := widget.NewLabelWithStyle(
				currentUnit.Name,
				fyne.TextAlignLeading,
				fyne.TextStyle{Bold: true},
			)

			metadataText := fmt.Sprintf(
				"Permission: %s  •  Users: %d",
				currentUnit.Level,
				len(currentUnit.Users),
			)

			if currentUnit.Revoked {
				metadataText += "  •  Revoked"
			}

			metadata := widget.NewLabel(metadataText)

			updateButton := widget.NewButtonWithIcon(
				"Update",
				theme.DocumentCreateIcon(),
				func() {
					showAccessUnitDialog(&currentUnit, onSave)
				},
			)

			deleteButton := widget.NewButtonWithIcon(
				"Delete",
				theme.DeleteIcon(),
				func() {
					current := entity.AccessUnit{
						ID:          currentUnit.ID,
						Name:        currentUnit.Name,
						GroupID:     currentUnit.GroupID,
						SotID:       currentUnit.SotID,
						Level:       currentUnit.Level,
						PublicKey:   currentUnit.PublicKey,
						AutoApprove: currentUnit.AutoApprove,
						Revoked:     currentUnit.Revoked,
					}

					showDeleteAccessUnitDialog(&current, onDelete)
				},
			)

			viewUsersButton := widget.NewButton(
				"View Users",
				func() {
					if onViewUsers != nil {
						onViewUsers(currentUnit)
					}
				},
			)

			actions := []fyne.CanvasObject{
				updateButton,
				deleteButton,
			}

			if currentUnit.Revoked {
				actions = append(
					actions,
					widget.NewButtonWithIcon(
						"Reactivate",
						theme.VisibilityIcon(),
						func() {
							showRevokeAccessUnitDialog(
								currentUnit.Name,
								true,
							)
						},
					),
				)
			} else {
				actions = append(
					actions,
					widget.NewButtonWithIcon(
						"Revoke",
						theme.VisibilityOffIcon(),
						func() {
							showRevokeAccessUnitDialog(
								currentUnit.Name,
								false,
							)
						},
					),
				)
			}

			actions = append(actions, viewUsersButton)

			unitCard := container.NewVBox(
				container.NewBorder(
					nil,
					nil,
					unitName,
					container.NewHBox(actions...),
					nil,
				),
				metadata,
			)

			unitList.Add(
				widget.NewCard(
					"",
					"",
					container.NewPadded(unitCard),
				),
			)
		}
	}

	return container.NewVBox(
		header,
		widget.NewSeparator(),
		unitList,
	)
}

func showAccessUnitDialog(
	unit *contract.AccessUnitWithUserCount,
	onSave func(*entity.AccessUnit) error,
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

	dialog.ShowForm(
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
				dialog.ShowInformation(
					"Error",
					"Save handler is not configured.",
					window,
				)
				return
			}

			accessUnit.Name = name
			accessUnit.Level = entity.AccessLevel(
				levelSelect.Selected,
			)
			accessUnit.AutoApprove = autoApproveCheck.Checked

			if err := onSave(accessUnit); err != nil {
				dialog.ShowError(err, window)
				return
			}
		},
		window,
	)
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
	unitName string,
	revoked bool,
) {
	windows := fyne.CurrentApp().Driver().AllWindows()
	if len(windows) == 0 {
		return
	}

	window := windows[0]

	action := "Revoke"
	actionTitle := "Revoke Access Unit"
	successMessage := "revoked"

	if revoked {
		action = "Reactivate"
		actionTitle = "Reactivate Access Unit"
		successMessage = "reactivated"
	}

	dialog.ShowConfirm(
		actionTitle,
		fmt.Sprintf(
			"Are you sure you want to %s access unit %q?",
			strings.ToLower(action),
			unitName,
		),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			// TODO: Sambungkan ke service Revoke/Reactivate.
			dialog.ShowInformation(
				"Not Implemented",
				fmt.Sprintf(
					"Revoke/reactivate service has not been connected yet. Requested action: %s.",
					successMessage,
				),
				window,
			)
		},
		window,
	)
}
