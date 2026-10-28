package accesscontrol

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-core/entity"
)

func newAccessUnitSection(
	group *entity.AccessGroup,
) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		"Access Units",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)

	unitList := container.NewVBox()

	// Placeholder actions: belum memanggil service atau mengubah data.
	showCreateDialog := func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Enter access unit name")

		levelSelect := widget.NewSelect(
			[]string{"admin", "write", "read"},
			nil,
		)
		levelSelect.SetSelected("read")

		formContent := container.NewVBox(
			widget.NewLabel("Name"),
			nameEntry,
			widget.NewLabel("Permission"),
			levelSelect,
		)

		dialog.ShowForm(
			"Create Access Unit",
			"Create",
			"Cancel",
			[]*widget.FormItem{
				widget.NewFormItem("Name", nameEntry),
				widget.NewFormItem("Permission", levelSelect),
			},
			func(confirmed bool) {
				if !confirmed {
					return
				}

				if nameEntry.Text == "" || levelSelect.Selected == "" {
					dialog.ShowInformation(
						"Validation",
						"Name and permission are required.",
						fyne.CurrentApp().Driver().AllWindows()[0],
					)
					return
				}

				dialog.ShowInformation(
					"Success",
					fmt.Sprintf(
						"Access unit \"%s\" created successfully.",
						nameEntry.Text,
					),
					fyne.CurrentApp().Driver().AllWindows()[0],
				)
			},
			fyne.CurrentApp().Driver().AllWindows()[0],
		)

		_ = formContent
	}

	if group == nil {
		unitList.Add(widget.NewLabel(
			"Select an access group to view its access units.",
		))

		return container.NewVBox(
			title,
			widget.NewSeparator(),
			unitList,
		)
	}

	createButton := widget.NewButtonWithIcon(
		"Create",
		theme.ContentAddIcon(),
		showCreateDialog,
	)

	header := container.NewBorder(
		nil,
		nil,
		title,
		createButton,
		nil,
	)

	if len(group.AccessUnits) == 0 {
		unitList.Add(
			container.NewCenter(
				widget.NewLabel(
					"No access units assigned to this group.",
				),
			),
		)
	} else {
		for _, unit := range group.AccessUnits {
			currentUnit := unit

			unitName := widget.NewLabelWithStyle(
				currentUnit.Name,
				fyne.TextAlignLeading,
				fyne.TextStyle{Bold: true},
			)

			metadata := widget.NewLabel(
				fmt.Sprintf(
					"Permission: %s  •  Users: %d",
					currentUnit.Level,
					len(currentUnit.Users),
				),
			)

			if currentUnit.Revoked {
				metadata.SetText(
					fmt.Sprintf(
						"Permission: %s  •  Users: %d  •  Revoked",
						currentUnit.Level,
						len(currentUnit.Users),
					),
				)
			}

			updateButton := widget.NewButtonWithIcon(
				"Update",
				theme.DocumentCreateIcon(),
				func() {
					showUpdateAccessUnitDialog(currentUnit)
				},
			)

			deleteButton := widget.NewButtonWithIcon(
				"Delete",
				theme.DeleteIcon(),
				func() {
					showDeleteAccessUnitDialog(currentUnit.Name)
				},
			)

			actions := []fyne.CanvasObject{
				updateButton,
				deleteButton,
			}

			if currentUnit.Revoked {
				actions = append(actions, widget.NewButtonWithIcon(
					"Reactivate",
					theme.VisibilityIcon(),
					func() {
						showRevokeAccessUnitDialog(currentUnit.Name, true)
					},
				))
			} else {
				actions = append(actions, widget.NewButtonWithIcon(
					"Revoke",
					theme.VisibilityOffIcon(),
					func() {
						showRevokeAccessUnitDialog(currentUnit.Name, false)
					},
				))
			}

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

func showUpdateAccessUnitDialog(unit entity.AccessUnit) {
	window := fyne.CurrentApp().Driver().AllWindows()[0]

	nameEntry := widget.NewEntry()
	nameEntry.SetText(unit.Name)

	levelSelect := widget.NewSelect(
		[]string{"admin", "write", "read"},
		nil,
	)
	levelSelect.SetSelected(string(unit.Level))

	dialog.ShowForm(
		"Update Access Unit",
		"Save",
		"Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Permission", levelSelect),
		},
		func(confirmed bool) {
			if !confirmed {
				return
			}

			if nameEntry.Text == "" || levelSelect.Selected == "" {
				dialog.ShowInformation(
					"Validation",
					"Name and permission are required.",
					window,
				)
				return
			}

			dialog.ShowInformation(
				"Success",
				fmt.Sprintf(
					"Access unit \"%s\" updated successfully.",
					unit.Name,
				),
				window,
			)
		},
		window,
	)
}

func showDeleteAccessUnitDialog(unitName string) {
	window := fyne.CurrentApp().Driver().AllWindows()[0]

	dialog.ShowConfirm(
		"Delete Access Unit",
		fmt.Sprintf(
			"Are you sure you want to delete access unit \"%s\"?",
			unitName,
		),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			dialog.ShowInformation(
				"Success",
				fmt.Sprintf(
					"Access unit \"%s\" deleted successfully.",
					unitName,
				),
				window,
			)
		},
		window,
	)
}
func showRevokeAccessUnitDialog(unitName string, revoked bool) {
	window := fyne.CurrentApp().Driver().AllWindows()[0]

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
			"Are you sure you want to %s access unit \"%s\"?",
			strings.ToLower(action),
			unitName,
		),
		func(confirmed bool) {
			if !confirmed {
				return
			}

			dialog.ShowInformation(
				"Success",
				fmt.Sprintf(
					"Access unit \"%s\" %s successfully.",
					unitName,
					successMessage,
				),
				window,
			)
		},
		window,
	)
}
