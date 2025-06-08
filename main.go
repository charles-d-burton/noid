package main

import (
	"log/slog"

	"fyne.io/fyne/app"
	"fyne.io/fyne/widget"
)

const (
	AppID    = "com.github.charles-d-burton.noid"
	SetupKey = "setup"
)

func main() {
	a := app.NewWithID(AppID)
	setup := a.Preferences().Bool(SetupKey)
	if !setup {
		slog.Info("setup has not been ran, running initial setup")

		a.Preferences().SetBool(SetupKey, true)
	}
	if setup {
		slog.Info("setup was ran, starting application")
	}

	w := a.NewWindow("Noid")
	w.SetContent(widget.NewLabel("Welcome to Noid"))
	w.ShowAndRun()
}
