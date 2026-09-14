package sourceoftruth

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tacenva/database"
	rCoreApp "github.com/tacenva/replica-core/app"
	"github.com/tacenva/replica-core/app/sourceoftruth"
	"github.com/tacenva/replica-core/entity"
	"github.com/tacenva/tacenva-desktop/internal/ui/app"
	coreApp "github.com/tacenva/tacpass-core/app"
	"github.com/tacenva/tacpass-core/config"
	"github.com/tacenva/tacpass-core/util/keyring"
)

type Screen struct {
	Content    fyne.CanvasObject
	sotService *sourceoftruth.Service
}

func New(
	window fyne.Window,
	appDeps *rCoreApp.Deps,
	masterKey string,
	coreServices *coreApp.Services,
	sotService *sourceoftruth.Service,
) *Screen {
	sotList, err := sotService.List()
	if err != nil {
		sotList = nil
	}

	title := widget.NewLabelWithStyle(
		"Source of Truth",
		fyne.TextAlignLeading,
		fyne.TextStyle{
			Bold: true,
		},
	)

	subtitle := widget.NewLabel(
		"Pilih source of truth yang ingin dibuka.",
	)

	headerContent := container.NewVBox(
		title,
		subtitle,
	)

	header := container.New(
		layout.NewCustomPaddedLayout(0, 0, 24, 24),
		headerContent,
	)

	var list *widget.List
	list = widget.NewList(
		func() int {
			return len(sotList)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			address := widget.NewLabel("")

			name.TextStyle = fyne.TextStyle{
				Bold: true,
			}

			editButton := widget.NewButtonWithIcon(
				"",
				theme.DocumentCreateIcon(),
				nil,
			)

			deleteButton := widget.NewButtonWithIcon(
				"",
				theme.DeleteIcon(),
				nil,
			)

			actions := container.NewHBox(
				editButton,
				deleteButton,
			)

			nameCell := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				name,
			)

			addressCell := container.NewBorder(
				nil,
				nil,
				nil,
				nil,
				address,
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
				nameCell,
				addressCell,
				actionCell,
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(sotList) {
				return
			}

			selectedSot := sotList[id]

			row := obj.(*fyne.Container)

			nameCell := row.Objects[0].(*fyne.Container)
			addressCell := row.Objects[1].(*fyne.Container)
			actionCell := row.Objects[2].(*fyne.Container)

			name := nameCell.Objects[0].(*widget.Label)
			address := addressCell.Objects[0].(*widget.Label)
			actions := actionCell.Objects[0].(*fyne.Container)

			editButton := actions.Objects[0].(*widget.Button)
			deleteButton := actions.Objects[1].(*widget.Button)

			name.SetText(selectedSot.Hostname)
			address.SetText(selectedSot.Address)

			editButton.OnTapped = func() {
				showForm(
					window,
					&selectedSot,
					func(updatedSot *entity.SourceOfTruth) {
						err := sotService.Update(updatedSot)
						if err != nil {
							dialog.ShowError(err, window)
							return
						}

						sotList[id] = *updatedSot
						list.RefreshItem(id)
					},
				)
			}

			deleteButton.OnTapped = func() {
				showDeleteConfirm(
					window,
					selectedSot.Hostname,
					func() {
						sotService.Del(&selectedSot)
						sotList = append(sotList[:id], sotList[id+1:]...)
						list.Refresh()
					},
				)
			}
		},
	)

	listContent := container.New(
		layout.NewCustomPaddedLayout(0, 0, 24, 24),
		list,
	)

	newButton := widget.NewButtonWithIcon(
		"New",
		theme.ContentAddIcon(),
		func() {
			showForm(
				window,
				nil,
				func(newSot *entity.SourceOfTruth) {
					sotId, err := sotService.Create(
						newSot.Hostname,
						newSot.Address,
						newSot.KeyPair,
						newSot.SyncMode,
					)
					if err != nil {
						dialog.ShowError(err, window)
						return
					}
					newSot.ID = sotId
					sotList = append(sotList, *newSot)
					list.Refresh()
				},
			)
		},
	)

	newButtonContainer := container.NewGridWrap(
		fyne.NewSize(90, 36),
		newButton,
	)

	actionsContent := container.NewHBox(
		newButtonContainer,
		layout.NewSpacer(),
	)

	actions := container.New(
		layout.NewCustomPaddedLayout(0, 0, 24, 24),
		actionsContent,
	)

	content := container.NewBorder(
		header,
		actions,
		nil,
		nil,
		listContent,
	)

	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(sotList) {
			return
		}

		selectedSot := sotList[id]

		nodeDBDir := appDeps.Config.Path(
			config.NodeDirName,
			selectedSot.ID,
			"vault",
		)

		appScreen := app.New(
			window,
			appDeps,
			&rCoreApp.Context{
				SelectedSoT: &selectedSot,
				NodeDB:      database.New(nodeDBDir),
				IsRemote:    selectedSot.Address != "localhost",
			},
			masterKey,
			coreServices,
			sotService,
			func() {
				window.SetContent(content)
				appDeps.Client.ClearToken()
				appDeps.Client.ClearTLS(selectedSot.Address)
				list.Unselect(id)
			},
		)

		window.SetContent(appScreen.Content)
	}

	return &Screen{
		Content:    content,
		sotService: sotService,
	}
}

func showForm(
	window fyne.Window,
	selectedSot *entity.SourceOfTruth,
	onSave func(sot *entity.SourceOfTruth),
) {
	isEdit := selectedSot != nil

	var sot entity.SourceOfTruth

	if isEdit {
		sot = *selectedSot
	}

	hostnameEntry := widget.NewEntry()
	hostnameEntry.SetText(sot.Hostname)
	hostnameEntry.SetPlaceHolder("Optional alias")

	addressEntry := widget.NewEntry()
	addressEntry.SetText(sot.Address)
	addressEntry.SetPlaceHolder("Source of Truth address")

	publicKeyEntry := widget.NewMultiLineEntry()
	publicKeyEntry.SetText(sot.KeyPair.PublicKey)
	publicKeyEntry.SetMinRowsVisible(4)

	privateKeyEntry := widget.NewMultiLineEntry()
	privateKeyEntry.SetText(sot.KeyPair.PrivateKey)
	privateKeyEntry.SetMinRowsVisible(4)

	syncModeSelect := widget.NewSelect(
		[]string{
			string(entity.SyncModeManual),
			string(entity.SyncModeAuto),
		},
		nil,
	)

	if sot.SyncMode != "" {
		syncModeSelect.SetSelected(
			string(sot.SyncMode),
		)
	} else {
		syncModeSelect.SetSelected(
			string(entity.SyncModeManual),
		)
	}

	formItems := []*widget.FormItem{
		widget.NewFormItem(
			"Hostname",
			hostnameEntry,
		),
		widget.NewFormItem(
			"Address",
			addressEntry,
		),
		widget.NewFormItem(
			"Public Key",
			publicKeyEntry,
		),
		widget.NewFormItem(
			"Private Key",
			privateKeyEntry,
		),
	}

	// Sync Mode hanya ditampilkan untuk remote Source of Truth.
	if addressEntry.Text != "localhost" {
		formItems = append(
			formItems,
			widget.NewFormItem(
				"Sync Mode",
				syncModeSelect,
			),
		)
	}

	form := widget.NewForm(
		formItems...,
	)

	formContainer := container.NewGridWrap(
		fyne.NewSize(650, 420),
		form,
	)

	title := "New Source of Truth"

	if isEdit {
		title = "Edit Source of Truth"
	}

	dialog.ShowCustomConfirm(
		title,
		"Save",
		"Cancel",
		formContainer,
		func(ok bool) {
			if !ok {
				return
			}

			hostname := hostnameEntry.Text
			address := addressEntry.Text

			if address == "" {
				dialog.ShowInformation(
					"Invalid Input",
					"Address wajib diisi.",
					window,
				)

				return
			}

			sot.Hostname = hostname
			sot.Address = address

			sot.KeyPair = *keyring.FromKeys(
				publicKeyEntry.Text,
				privateKeyEntry.Text,
			)

			if address != "localhost" {
				sot.SyncMode = entity.SyncMode(
					syncModeSelect.Selected,
				)
			}

			onSave(&sot)
		},
		window,
	)
}

func showDeleteConfirm(
	window fyne.Window,
	hostname string,
	onDelete func(),
) {
	dialog.ShowConfirm(
		"Delete Source of Truth",
		"Source of Truth \""+hostname+"\" akan dihapus. Lanjutkan?",
		func(ok bool) {
			if !ok {
				return
			}

			onDelete()
		},
		window,
	)
}
