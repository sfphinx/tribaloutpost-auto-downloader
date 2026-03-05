package dialog

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// PickGameDataDir shows a GUI dialog for the user to select their Tribes 2 GameData directory.
// If candidates is non-empty, they are shown as clickable options.
// Returns the selected path, or empty string if the user cancelled.
func PickGameDataDir(candidates []string) string {
	a := app.NewWithID("com.tribaloutpost.autodownload.setup")
	w := a.NewWindow("TribalOutpost AutoDownload - Setup")
	w.Resize(fyne.NewSize(550, 350))
	w.CenterOnScreen()

	result := make(chan string, 1)

	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("/path/to/Tribes2/GameData")

	var content *fyne.Container

	if len(candidates) > 0 {
		label := widget.NewLabel("Multiple Tribes 2 installations found.\nSelect one or browse for a different location:")
		label.Wrapping = fyne.TextWrapWord

		list := widget.NewList(
			func() int { return len(candidates) },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(id widget.ListItemID, obj fyne.CanvasObject) {
				obj.(*widget.Label).SetText(candidates[id])
			},
		)
		list.OnSelected = func(id widget.ListItemID) {
			pathEntry.SetText(candidates[id])
		}

		content = container.NewVBox(
			label,
			container.New(layout.NewMaxLayout(), container.NewVScroll(list)),
		)
	} else {
		label := widget.NewLabel("Could not auto-detect your Tribes 2 installation.\nPlease select your GameData directory:")
		label.Wrapping = fyne.TextWrapWord
		content = container.NewVBox(label)
	}

	browseBtn := widget.NewButton("Browse...", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			pathEntry.SetText(uri.Path())
		}, w)
		d.Resize(fyne.NewSize(600, 400))
		d.Show()
	})

	okBtn := widget.NewButton("OK", func() {
		result <- pathEntry.Text
		w.Close()
	})
	okBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButton("Cancel", func() {
		result <- ""
		w.Close()
	})

	buttons := container.NewHBox(layout.NewSpacer(), cancelBtn, okBtn)
	pathRow := container.NewBorder(nil, nil, nil, browseBtn, pathEntry)

	w.SetContent(container.NewBorder(
		content,
		container.NewVBox(pathRow, buttons),
		nil, nil,
	))

	w.SetOnClosed(func() {
		select {
		case result <- "":
		default:
		}
	})

	w.ShowAndRun()
	selected := <-result
	return selected
}
