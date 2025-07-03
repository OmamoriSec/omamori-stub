package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type LogManager struct {
	app *OmamoriApp
}

func (o *OmamoriApp) createLogManager() *LogManager {
	return &LogManager{
		app: o,
	}
}

func (l *LogManager) logTab() *fyne.Container {
	// Placeholder for metrics
	return container.NewVBox(
		widget.NewCard("DNS Query / Event Logs", "Coming Soon",
			widget.NewLabel("Logs will be displayed here"),
		),
	)
}
