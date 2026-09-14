package accesscontrol

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/replica-core/app"
	"github.com/tacenva/replica-core/app/accesscontrol"
	coreApp "github.com/tacenva/tacpass-core/app"
	"github.com/tacenva/tacpass-core/entity"
)

type UserScreen struct {
	Content fyne.CanvasObject
}

func NewUser(
	window fyne.Window,
	appDeps *app.Deps,
	context *app.Context,
	coreService *coreApp.Services,
	acService *accesscontrol.Service,
	selectedPermission *entity.Permission,
) *UserScreen {
	userList, err := acService.UserList(selectedPermission.ID)
	if err != nil {
		dialog.ShowError(err, window)
		userList = nil
	}

	search := widget.NewEntry()
	search.SetPlaceHolder("Search users...")

	var entryList *widget.List

	reload := func() {
		updatedUserList, err := acService.UserList(selectedPermission.ID)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		userList = updatedUserList

		if entryList != nil {
			entryList.Refresh()
		}
	}

	entryList = widget.NewList(
		func() int {
			return len(userList)
		},
		func() fyne.CanvasObject {
			hostname := widget.NewLabel("")
			status := widget.NewLabel("")

			approveButton := widget.NewButtonWithIcon(
				"",
				theme.ConfirmIcon(),
				nil,
			)

			revokeButton := widget.NewButtonWithIcon(
				"",
				theme.DeleteIcon(),
				nil,
			)

			actions := container.NewHBox(
				approveButton,
				revokeButton,
			)

			return container.NewGridWithColumns(
				3,
				hostname,
				status,
				actions,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(userList) {
				return
			}

			userData := userList[id]

			row := obj.(*fyne.Container)

			hostname := row.Objects[0].(*widget.Label)
			status := row.Objects[1].(*widget.Label)
			actions := row.Objects[2].(*fyne.Container)

			approveButton := actions.Objects[0].(*widget.Button)
			revokeButton := actions.Objects[1].(*widget.Button)

			hostname.SetText(userData.Hostname)
			status.SetText(string(userData.Status))

			approveButton.OnTapped = func() {
				dialog.ShowConfirm(
					"Approve User",
					"User \""+userData.Hostname+"\" akan di-approve. Lanjutkan?",
					func(ok bool) {
						if !ok {
							return
						}

						_, err := acService.ApproveUser(
							userData.ID,
						)
						if err != nil {
							dialog.ShowError(err, window)
							return
						}

						reload()
					},
					window,
				)
			}

			revokeButton.OnTapped = func() {
				dialog.ShowConfirm(
					"Revoke User",
					"User \""+userData.Hostname+"\" akan di-revoke. Lanjutkan?",
					func(ok bool) {
						if !ok {
							return
						}

						_, err := acService.RevokeUser(
							userData.ID,
						)
						if err != nil {
							dialog.ShowError(err, window)
							return
						}

						reload()
					},
					window,
				)
			}

			// Atur tombol berdasarkan status user.
			switch userData.Status {
			case entity.UserStatusPending:
				approveButton.Show()
				revokeButton.Show()

			case entity.UserStatusApproved:
				approveButton.Hide()
				revokeButton.Show()

			case entity.UserStatusRevoked:
				approveButton.Show()
				revokeButton.Hide()

			default:
				approveButton.Hide()
				revokeButton.Hide()
			}

			actions.Refresh()
		},
	)

	content := container.NewMax()

	userContent := container.NewBorder(
		container.NewBorder(
			search,
			nil,
			nil,
			nil,
			widget.NewLabelWithStyle(
				"Users",
				fyne.TextAlignLeading,
				fyne.TextStyle{
					Bold: true,
				},
			),
		),
		nil,
		nil,
		nil,
		entryList,
	)

	content.Objects = []fyne.CanvasObject{
		userContent,
	}

	return &UserScreen{
		Content: content,
	}
}
