package accesscontrol

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/tacenva-desktop/internal/ui/loading"
	"github.com/tacenva/tacenva-services/app"
	"github.com/tacenva/tacenva-services/app/accesscontrol"
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

	allUsers := make([]entity.User, 0)
	filteredUsers := make([]entity.User, 0)

	var entryList *widget.List

	content := container.NewMax()

	loadingView := loading.New(
		"Loading users...",
	)

	filterUsers := func(
		users []entity.User,
		query string,
	) []entity.User {
		query = strings.ToLower(
			strings.TrimSpace(query),
		)

		if query == "" {
			return append(
				[]entity.User(nil),
				users...,
			)
		}

		filtered := make(
			[]entity.User,
			0,
			len(users),
		)

		for _, user := range users {
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

		return filtered
	}

	reload := func() {
		go func() {
			updatedUsers, err := acService.UserList(
				selectedPermission.ID,
			)

			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(
						err,
						window,
					)

					return
				}

				allUsers = updatedUsers

				filteredUsers = filterUsers(
					allUsers,
					search.Text,
				)

				if entryList != nil {
					entryList.UnselectAll()
					entryList.Refresh()
				}
			})
		}()
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

			statusButton := widget.NewButton("", nil)

			actionCell := container.NewBorder(
				nil,
				nil,
				nil,
				statusButton,
				nil,
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

			row, ok := obj.(*fyne.Container)
			if !ok || len(row.Objects) < 3 {
				return
			}

			hostnameCell, ok := row.Objects[0].(*fyne.Container)
			if !ok || len(hostnameCell.Objects) == 0 {
				return
			}

			statusCell, ok := row.Objects[1].(*fyne.Container)
			if !ok || len(statusCell.Objects) == 0 {
				return
			}

			actionCell, ok := row.Objects[2].(*fyne.Container)
			if !ok || len(actionCell.Objects) == 0 {
				return
			}

			hostname, ok := hostnameCell.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			status, ok := statusCell.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			statusButton, ok := actionCell.Objects[0].(*widget.Button)
			if !ok {
				return
			}

			hostname.SetText(userData.Hostname)
			status.SetText(string(userData.Status))

			var (
				actionName   string
				confirmTitle string
				successMsg   string
				targetStatus entity.UserStatus
			)

			switch userData.Status {
			case entity.UserStatusPending, entity.UserStatusRevoked:
				actionName = "Approve"
				confirmTitle = "Approve User"
				successMsg = "User approved successfully."
				targetStatus = entity.UserStatusApproved

			case entity.UserStatusApproved:
				actionName = "Revoke"
				confirmTitle = "Revoke User"
				successMsg = "User revoked successfully."
				targetStatus = entity.UserStatusRevoked

			default:
				statusButton.Hide()
				return
			}

			statusButton.SetText(actionName)
			statusButton.Show()

			statusButton.OnTapped = func() {
				currentUser := userData

				dialog.ShowConfirm(
					confirmTitle,
					"User \""+currentUser.Hostname+
						"\" will be "+strings.ToLower(actionName)+
						". Continue?",
					func(ok bool) {
						if !ok {
							return
						}

						statusButton.Disable()

						go func() {
							err := acService.SetUserStatus(
								currentUser.ID,
								targetStatus,
							)

							fyne.Do(func() {
								statusButton.Enable()

								if err != nil {
									dialog.ShowError(err, window)
									return
								}

								dialog.ShowInformation(
									"Success",
									successMsg,
									window,
								)

								reload()
							})
						}()
					},
					window,
				)
			}
		},
	)

	toolbar := container.NewBorder(
		nil,
		nil,
		backButton,
		nil,
		search,
	)

	userContent := container.NewBorder(
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
				filtered := filterUsers(
					allUsers,
					query,
				)

				fyne.Do(func() {
					filteredUsers = filtered

					entryList.UnselectAll()
					entryList.Refresh()
				})
			},
		)
	}

	// Tampilkan loading terlebih dahulu.
	content.Objects = []fyne.CanvasObject{
		loadingView.Content,
	}

	// Jalankan UserList() di background.
	go func() {
		updatedUsers, err := acService.UserList(
			selectedPermission.ID,
		)

		fyne.Do(func() {
			if err != nil {
				loadingView.ShowError(
					"Failed to load users.",
				)

				dialog.ShowError(
					err,
					window,
				)

				return
			}

			allUsers = updatedUsers

			filteredUsers = filterUsers(
				allUsers,
				search.Text,
			)

			entryList.Refresh()

			loadingView.Show(
				userContent,
			)
		})
	}()

	return &UserScreen{
		Content: content,
	}
}
