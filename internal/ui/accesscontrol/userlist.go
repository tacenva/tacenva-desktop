package accesscontrol

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/oklog/ulid/v2"
	"github.com/tacenva/tacenva-core/entity"
)

func newUserListSection(
	window fyne.Window,
	users []entity.User,
	onApprove func(entity.User),
	onRevoke func(entity.User),
	onDelete func(entity.User),
) fyne.CanvasObject {
	if len(users) == 0 {
		return container.NewCenter(
			widget.NewLabel("No users available."),
		)
	}

	// Shared row layout builder so header and list rows share exact column alignment
	buildRowContainer := func(
		actionObj fyne.CanvasObject,
		identityObj fyne.CanvasObject,
		joinAtObj fyne.CanvasObject,
		statusObj fyne.CanvasObject,
		deleteObj fyne.CanvasObject,
	) *fyne.Container {
		// Left fixed-width slot for Approve / Revoke / Reactivate button
		leftSlot := container.NewGridWrap(fyne.NewSize(130, 36), actionObj)

		// Right fixed-width slot for Delete icon button
		rightSlot := container.NewGridWrap(fyne.NewSize(40, 36), deleteObj)

		// Center 3 columns (Sot Identity, Join At, Status)
		centerGrid := container.NewGridWithColumns(
			3,
			container.NewBorder(nil, nil, nil, nil, identityObj),
			container.NewBorder(nil, nil, nil, nil, joinAtObj),
			container.NewBorder(nil, nil, nil, nil, statusObj),
		)

		return container.NewBorder(nil, nil, leftSlot, rightSlot, centerGrid)
	}

	// Header row
	hIdentity := widget.NewLabel("Sot Identity")
	hIdentity.TextStyle.Bold = true

	hJoinAt := widget.NewLabel("Join At")
	hJoinAt.TextStyle.Bold = true

	hStatus := widget.NewLabel("Status")
	hStatus.TextStyle.Bold = true

	header := buildRowContainer(
		widget.NewLabel(""),
		hIdentity,
		hJoinAt,
		hStatus,
		widget.NewLabel(""),
	)

	var userList *widget.List
	userList = widget.NewList(
		func() int {
			return len(users)
		},
		func() fyne.CanvasObject {
			actionButton := widget.NewButtonWithIcon("", theme.VisibilityIcon(), nil)

			identity := widget.NewLabel("")
			identity.Wrapping = fyne.TextWrapBreak
			identity.TextStyle = fyne.TextStyle{Bold: true}

			joinAt := widget.NewLabel("")

			// Placeholder container for status indicator widget
			statusHolder := container.NewMax()

			deleteButton := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			deleteButton.Importance = widget.DangerImportance

			// Wrap buttons in NewVBox so they keep a fixed natural height when identity wraps
			actionBox := container.NewVBox(
				container.NewHBox(
					layout.NewSpacer(),
					actionButton,
				),
			)

			deleteBox := container.NewVBox(deleteButton)

			return buildRowContainer(
				actionBox,
				identity,
				joinAt,
				statusHolder,
				deleteBox,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(users) {
				return
			}

			user := users[id]

			row, ok := obj.(*fyne.Container)
			if !ok || len(row.Objects) < 3 {
				return
			}

			// Border Objects order when center, left, right are all non-nil:
			// Objects[0] = centerGrid, Objects[1] = leftSlot, Objects[2] = rightSlot
			centerGrid, ok := row.Objects[0].(*fyne.Container)
			if !ok || len(centerGrid.Objects) < 3 {
				return
			}

			leftSlot, ok := row.Objects[1].(*fyne.Container)
			if !ok || len(leftSlot.Objects) == 0 {
				return
			}

			rightSlot, ok := row.Objects[2].(*fyne.Container)
			if !ok || len(rightSlot.Objects) == 0 {
				return
			}

			// Unwrap actionButton: leftSlot -> actionBox (VBox) -> HBox -> [Spacer, actionButton]
			actionBox, ok := leftSlot.Objects[0].(*fyne.Container)
			if !ok || len(actionBox.Objects) == 0 {
				return
			}

			actionHBox, ok := actionBox.Objects[0].(*fyne.Container)
			if !ok || len(actionHBox.Objects) < 2 {
				return
			}

			actionButton, ok := actionHBox.Objects[1].(*widget.Button)
			if !ok {
				return
			}

			// Unwrap centerGrid columns
			identityCell, ok := centerGrid.Objects[0].(*fyne.Container)
			if !ok || len(identityCell.Objects) == 0 {
				return
			}

			joinAtCell, ok := centerGrid.Objects[1].(*fyne.Container)
			if !ok || len(joinAtCell.Objects) == 0 {
				return
			}

			statusCell, ok := centerGrid.Objects[2].(*fyne.Container)
			if !ok || len(statusCell.Objects) == 0 {
				return
			}

			identity, ok := identityCell.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			joinAt, ok := joinAtCell.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			statusHolder, ok := statusCell.Objects[0].(*fyne.Container)
			if !ok {
				return
			}

			// Unwrap deleteButton: rightSlot -> deleteBox (VBox) -> deleteButton
			deleteBox, ok := rightSlot.Objects[0].(*fyne.Container)
			if !ok || len(deleteBox.Objects) == 0 {
				return
			}

			deleteButton, ok := deleteBox.Objects[0].(*widget.Button)
			if !ok {
				return
			}

			// 1. Configure action button based on status
			switch user.Status {
			case entity.UserStatusPending:
				actionButton.Show()
				actionButton.SetText("Approve")
				actionButton.SetIcon(theme.VisibilityIcon())
				actionButton.OnTapped = func() {
					if onApprove != nil {
						onApprove(user)
					}
				}

			case entity.UserStatusApproved:
				actionButton.Show()
				actionButton.SetText("Revoke")
				actionButton.SetIcon(theme.VisibilityOffIcon())
				actionButton.OnTapped = func() {
					if onRevoke != nil {
						onRevoke(user)
					}
				}

			case entity.UserStatusRevoked:
				actionButton.Show()
				actionButton.SetText("Reactivate")
				actionButton.SetIcon(theme.VisibilityIcon())
				actionButton.OnTapped = func() {
					if onApprove != nil {
						onApprove(user)
					}
				}

			default:
				actionButton.Hide()
			}

			// 2. Configure Sot Identity
			sotIdentity := user.SotIdentity
			if sotIdentity == "" {
				sotIdentity = "Unknown identity"
			}
			identity.SetText(sotIdentity)

			// 3. Configure Join At
			createdAt := "Unknown date"
			if parsedTime, err := ULIDToTime(user.ID); err == nil {
				createdAt = parsedTime.Format("02 Jan 2006 15:04:05")
			}
			joinAt.SetText(createdAt)

			// 4. Configure Status Indicator
			statusHolder.Objects = []fyne.CanvasObject{
				newStatusIndicator(user.Status),
			}
			statusHolder.Refresh()

			// 5. Configure Delete Button
			deleteButton.OnTapped = func() {
				displayIdentity := user.SotIdentity
				if displayIdentity == "" {
					displayIdentity = "Unknown identity"
				}

				dialog.ShowConfirm(
					"Delete User",
					fmt.Sprintf(
						"Are you sure you want to delete user %q?",
						displayIdentity,
					),
					func(confirmed bool) {
						if confirmed && onDelete != nil {
							onDelete(user)
						}
					},
					window,
				)
			}

			obj.Refresh()
			userList.SetItemHeight(id, obj.MinSize().Height)
		},
	)

	return container.NewBorder(
		header,
		nil,
		nil,
		nil,
		userList,
	)
}

func ULIDToTime(id string) (time.Time, error) {
	parsed, err := ulid.Parse(id)
	if err != nil {
		return time.Time{}, err
	}

	return time.UnixMilli(int64(parsed.Time())), nil
}

func newStatusIndicator(status entity.UserStatus) fyne.CanvasObject {
	var (
		statusColor color.Color
		statusText  string
	)

	switch status {
	case entity.UserStatusPending:
		statusColor = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
		statusText = "Pending"

	case entity.UserStatusApproved:
		statusColor = color.NRGBA{R: 46, G: 160, B: 100, A: 255}
		statusText = "Approved"

	case entity.UserStatusRevoked:
		statusColor = color.NRGBA{R: 220, G: 65, B: 65, A: 255}
		statusText = "Revoked"

	default:
		statusColor = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
		statusText = "Unknown"
	}

	circle := canvas.NewCircle(statusColor)
	circle.Resize(fyne.NewSize(8, 8))

	circleContainer := container.NewGridWrap(
		fyne.NewSize(8, 8),
		circle,
	)

	return container.NewHBox(
		container.NewCenter(circleContainer),
		container.NewCenter(widget.NewLabel(statusText)),
	)
}

func vCenter(obj fyne.CanvasObject) fyne.CanvasObject {
	return container.NewVBox(
		layout.NewSpacer(),
		obj,
		layout.NewSpacer(),
	)
}
