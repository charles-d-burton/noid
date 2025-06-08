package main

import (
	"flag"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	AppID    = "com.github.charles-d-burton.noid"
	SetupKey = "setup"
)

func main() {
	var devMode bool
	flag.BoolVar(&devMode, "dev", false, "Set to enable development mode")
	flag.Parse()
	a := app.NewWithID(AppID)

	if devMode {
		slog.Info("running in dev mode")
	}

	setup := a.Preferences().Bool(SetupKey)

	if !setup && !devMode {
		slog.Info("setup has not been ran, running initial setup")

		a.Preferences().SetBool(SetupKey, true)
	}
	slog.Info("starting application")
	w := a.NewWindow("Noid")
	w.SetContent(container.NewBorder(widget.NewLabel("Welcome to Noid"), nil, chats(), nil, nil))

	w.ShowAndRun()
}

func chats() *widget.List {
	friends := []string{"Bob", "Alice"}
	list := widget.NewList(
		func() int {
			return len(friends)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Friends")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			l := o.(*widget.Label)
			l.Selectable = true
			l.SetText(friends[i])
		})
	return list
}
