package main

import (
	"fyne.io/fyne/app"
	"fyne.io/fyne/widget"
)

func main() {
	a := app.New()
	w := a.NewWindow("Noid")
	w.SetContent(widget.NewLabel("Welcome to Noid"))
	w.ShowAndRun()
}
