package accesscontrol

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-core/entity"
)

func newUserListSection(
	window fyne.Window,
	users []entity.User,
	onApprove func(entity.User),
	onRevoke func(entity.User),
	onDelete func(entity.User),
) fyne.CanvasObject {
	// title := widget.NewLabelWithStyle(
	// 	"Users",
	// 	fyne.TextAlignLeading,
	// 	fyne.TextStyle{Bold: true},
	// )

	rows := container.NewVBox()

	if len(users) == 0 {
		emptyState := container.NewCenter(
			widget.NewLabel("No users available."),
		)
		rows.Add(emptyState)
	}

	for _, user := range users {
		currentUser := user

		hostname := currentUser.Hostname
		if hostname == "" {
			hostname = "Unknown host"
		}

		statusText := string(currentUser.Status)
		if statusText == "" {
			statusText = "Unknown"
		}

		hostnameLabel := widget.NewLabelWithStyle(
			hostname,
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		)

		statusLabel := widget.NewLabel(
			"Status: " + statusText,
		)

		approveButton := widget.NewButton(
			"Approve",
			func() {
				if onApprove != nil {
					onApprove(currentUser)
				}
			},
		)

		revokeButton := widget.NewButton(
			"Revoke",
			func() {
				if onRevoke != nil {
					onRevoke(currentUser)
				}
			},
		)

		deleteButton := widget.NewButton(
			"Delete",
			func() {
				dialog.ShowConfirm(
					"Delete User",
					fmt.Sprintf(
						"Are you sure you want to delete user %q?",
						hostname,
					),
					func(confirmed bool) {
						if confirmed && onDelete != nil {
							onDelete(currentUser)
						}
					},
					window,
				)
			},
		)

		actions := container.NewHBox(
			approveButton,
			revokeButton,
			deleteButton,
		)

		userDetails := container.NewVBox(
			hostnameLabel,
			statusLabel,
		)

		row := container.NewBorder(
			nil,
			nil,
			userDetails,
			actions,
			nil,
		)

		rows.Add(container.NewPadded(row))
	}

	return container.NewVBox(
		widget.NewSeparator(),
		rows,
	)
}
