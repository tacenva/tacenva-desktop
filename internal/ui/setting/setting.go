package setting

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Screen struct {
	Content fyne.CanvasObject
}

func New(
	window fyne.Window,
	onChangePassword func(
		currentPassword string,
		newPassword string,
	) error,
) *Screen {
	currentPasswordEntry := widget.NewPasswordEntry()
	currentPasswordEntry.SetPlaceHolder("Current master password")

	newPasswordEntry := widget.NewPasswordEntry()
	newPasswordEntry.SetPlaceHolder("New master password")

	confirmPasswordEntry := widget.NewPasswordEntry()
	confirmPasswordEntry.SetPlaceHolder("Confirm new master password")

	form := widget.NewForm(
		widget.NewFormItem(
			"Current Password",
			currentPasswordEntry,
		),
		widget.NewFormItem(
			"New Password",
			newPasswordEntry,
		),
		widget.NewFormItem(
			"Confirm Password",
			confirmPasswordEntry,
		),
	)

	var changeButton *widget.Button

	changeButton = widget.NewButton(
		"Change Master Password",
		func() {
			currentPassword := currentPasswordEntry.Text
			newPassword := newPasswordEntry.Text
			confirmPassword := confirmPasswordEntry.Text

			if currentPassword == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"Current master password wajib diisi.",
					window,
				)
				return
			}

			if newPassword == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"New master password wajib diisi.",
					window,
				)
				return
			}

			if confirmPassword == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"Confirm password wajib diisi.",
					window,
				)
				return
			}

			if newPassword != confirmPassword {
				dialog.ShowInformation(
					"Invalid Input",
					"Password baru dan konfirmasi password tidak sama.",
					window,
				)
				return
			}

			if currentPassword == newPassword {
				dialog.ShowInformation(
					"Invalid Input",
					"Password baru harus berbeda dari password lama.",
					window,
				)
				return
			}

			changeButton.Disable()

			err := onChangePassword(
				currentPassword,
				newPassword,
			)

			changeButton.Enable()

			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			currentPasswordEntry.SetText("")
			newPasswordEntry.SetText("")
			confirmPasswordEntry.SetText("")

			dialog.ShowInformation(
				"Success",
				"Master password berhasil diubah.",
				window,
			)
		},
	)

	content := container.NewVBox(
		widget.NewLabelWithStyle(
			"Change Master Password",
			fyne.TextAlignLeading,
			fyne.TextStyle{
				Bold: true,
			},
		),
		form,
		changeButton,
	)

	formContainer := container.NewGridWrap(
		fyne.NewSize(600, 300),
		content,
	)

	return &Screen{
		Content: container.NewCenter(
			formContainer,
		),
	}
}
