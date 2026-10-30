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

	table := widget.NewTable(
		func() (int, int) {
			return len(users) + 1, 5
		},
		func() fyne.CanvasObject {
			return container.NewMax(
				widget.NewLabel(""),
			)
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			cellContainer := cell.(*fyne.Container)
			cellContainer.Objects = nil

			if id.Row == 0 {
				headers := []string{
					"",
					"Hostname",
					"Join At",
					"Status",
					"",
				}

				label := widget.NewLabel(headers[id.Col])
				label.TextStyle.Bold = true

				cellContainer.Add(label)
				cellContainer.Refresh()
				return
			}

			user := users[id.Row-1]

			switch id.Col {
			case 0:
				var button *widget.Button

				switch user.Status {
				case entity.UserStatusPending:
					button = widget.NewButtonWithIcon(
						"Approve",
						theme.VisibilityIcon(),
						func() {
							if onApprove != nil {
								onApprove(user)
							}
						},
					)

				case entity.UserStatusApproved:
					button = widget.NewButtonWithIcon(
						"Revoke",
						theme.VisibilityOffIcon(),
						func() {
							if onRevoke != nil {
								onRevoke(user)
							}
						},
					)

				case entity.UserStatusRevoked:
					button = widget.NewButtonWithIcon(
						"Reactivate",
						theme.VisibilityIcon(),
						func() {
							if onApprove != nil {
								onApprove(user)
							}
						},
					)
				}

				if button != nil {
					cellContainer.Add(
						container.New(
							layout.NewCustomPaddedLayout(
								2, // top
								2, // bottom
								0, // left
								0, // right
							),
							container.NewHBox(
								layout.NewSpacer(),
								button,
							),
						),
					)
				}

			case 1:
				hostname := user.Hostname
				if hostname == "" {
					hostname = "Unknown host"
				}

				label := widget.NewLabelWithStyle(
					hostname,
					fyne.TextAlignLeading,
					fyne.TextStyle{Bold: true},
				)

				cellContainer.Add(vCenter(label))

			case 2:
				createdAt := "Unknown date"

				if parsedTime, err := ULIDToTime(user.ID); err == nil {
					createdAt = parsedTime.Format("02 Jan 2006 15:04:05")
				}

				cellContainer.Add(vCenter(widget.NewLabel(createdAt)))

			case 3:
				cellContainer.Add(
					container.New(
						layout.NewCustomPaddedLayout(
							0, 0, 8, 0,
						),
						newStatusIndicator(user.Status),
					),
				)

			case 4:
				deleteButton := widget.NewButtonWithIcon(
					"",
					theme.DeleteIcon(),
					func() {
						hostname := user.Hostname
						if hostname == "" {
							hostname = "Unknown host"
						}

						dialog.ShowConfirm(
							"Delete User",
							fmt.Sprintf(
								"Are you sure you want to delete user %q?",
								hostname,
							),
							func(confirmed bool) {
								if confirmed && onDelete != nil {
									onDelete(user)
								}
							},
							window,
						)
					},
				)

				deleteButton.Importance = widget.DangerImportance

				cellContainer.Add(
					container.New(
						layout.NewCustomPaddedLayout(
							2, // top
							2, // bottom
							0, // left
							4, // right
						),
						deleteButton,
					),
				)
			}

			cellContainer.Refresh()
		},
	)

	table.SetColumnWidth(0, 130)
	table.SetColumnWidth(1, 150)
	table.SetColumnWidth(2, 200)
	table.SetColumnWidth(3, 200)
	table.SetColumnWidth(4, 40)

	table.SetRowHeight(0, 36)

	for i := 1; i <= len(users); i++ {
		table.SetRowHeight(i, 42)
	}

	table.HideSeparators = true

	return table
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
