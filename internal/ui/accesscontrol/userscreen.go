package accesscontrol

import (
	"strings"
	"time"

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
	onBack func(),
) *UserScreen {
	var searchTimer *time.Timer

	search := widget.NewEntry()
	search.SetPlaceHolder("Search users...")

	allUsers, err := acService.UserList(selectedPermission.ID)
	if err != nil {
		dialog.ShowError(err, window)
		allUsers = nil
	}

	filteredUsers := append(
		[]entity.User(nil),
		allUsers...,
	)

	var entryList *widget.List

	reload := func() {
		updatedUsers, err := acService.UserList(
			selectedPermission.ID,
		)
		if err != nil {
			dialog.ShowError(err, window)
			return
		}

		allUsers = updatedUsers

		query := strings.ToLower(
			strings.TrimSpace(search.Text),
		)

		filteredUsers = filteredUsers[:0]

		for _, user := range allUsers {
			hostname := strings.ToLower(user.Hostname)
			status := strings.ToLower(string(user.Status))

			if strings.Contains(hostname, query) ||
				strings.Contains(status, query) {
				filteredUsers = append(
					filteredUsers,
					user,
				)
			}
		}

		if entryList != nil {
			entryList.UnselectAll()
			entryList.Refresh()
		}
	}

	backButton := widget.NewButtonWithIcon(
		selectedPermission.Name,
		theme.NavigateBackIcon(),
		onBack,
	)

	entryList = widget.NewList(
		func() int {
			return len(filteredUsers)
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

			hostnameCell := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				hostname,
			)

			statusCell := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				status,
			)

			actionCell := container.NewBorder(
				nil,
				nil,
				nil,
				actions,
				nil,
			)

			return container.NewGridWithColumns(
				3,
				hostnameCell,
				statusCell,
				actionCell,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(filteredUsers) {
				return
			}

			userData := filteredUsers[id]

			row := obj.(*fyne.Container)

			hostnameCell := row.Objects[0].(*fyne.Container)
			statusCell := row.Objects[1].(*fyne.Container)
			actionCell := row.Objects[2].(*fyne.Container)

			hostname := hostnameCell.Objects[0].(*widget.Label)
			status := statusCell.Objects[0].(*widget.Label)
			actions := actionCell.Objects[0].(*fyne.Container)

			approveButton := actions.Objects[0].(*widget.Button)
			revokeButton := actions.Objects[1].(*widget.Button)

			hostname.SetText(userData.Hostname)
			status.SetText(string(userData.Status))

			approveButton.OnTapped = func() {
				dialog.ShowConfirm(
					"Approve User",
					"User \""+
						userData.Hostname+
						"\" akan di-approve. Lanjutkan?",
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
					"User \""+
						userData.Hostname+
						"\" akan di-revoke. Lanjutkan?",
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

	toolbar := container.NewBorder(
		nil,
		nil,
		backButton,
		nil,
		search,
	)

	content := container.NewBorder(
		toolbar,
		nil,
		nil,
		nil,
		entryList,
	)

	search.OnChanged = func(query string) {
		if searchTimer != nil {
			searchTimer.Stop()
		}

		searchTimer = time.AfterFunc(
			200*time.Millisecond,
			func() {
				query = strings.ToLower(
					strings.TrimSpace(query),
				)

				filtered := make(
					[]entity.User,
					0,
					len(allUsers),
				)

				for _, user := range allUsers {
					hostname := strings.ToLower(
						user.Hostname,
					)

					status := strings.ToLower(
						string(user.Status),
					)

					if strings.Contains(hostname, query) ||
						strings.Contains(status, query) {
						filtered = append(
							filtered,
							user,
						)
					}
				}

				fyne.Do(func() {
					filteredUsers = filtered

					entryList.UnselectAll()
					entryList.Refresh()
				})
			},
		)
	}

	return &UserScreen{
		Content: content,
	}
}
