package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"strings"
	"time"
)

type LogManager struct {
	app     *OmamoriApp
	logText *widget.Entry
}

func (o *OmamoriApp) createLogManager() *LogManager {
	logEntry := widget.NewMultiLineEntry()
	logEntry.SetPlaceHolder("Logs will appear here...")

	lm := &LogManager{
		app:     o,
		logText: logEntry,
	}
	return lm
}

func (lm *LogManager) AppendLog(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message = fmt.Sprintf("[%s] %s\n", timestamp, message)

	fyne.Do(func() {
		existingText := lm.logText.Text
		lm.logText.SetText(message + existingText)
		lines := strings.Split(lm.logText.Text, "\n")
		if len(lines) > 1000 {
			lm.logText.SetText(strings.Join(lines[:1000], "\n"))
		}
	})
}

func (lm *LogManager) AppendErrorLogs(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message = fmt.Sprintf("[%s] ERROR: %s\n", timestamp, message)

	fyne.Do(func() {
		existingText := lm.logText.Text
		lm.logText.SetText(message + existingText)
		lines := strings.Split(lm.logText.Text, "\n")
		if len(lines) > 1000 {
			lm.logText.SetText(strings.Join(lines[:1000], "\n"))
		}
	})
}

func (lm *LogManager) logTab() *fyne.Container {
	logScroll := container.NewScroll(lm.logText)
	logScroll.SetMinSize(fyne.NewSize(400, 400))
	clearButton := widget.NewButton("Clear Logs", func() {
		lm.logText.SetText("")
	})
	clearButton.Importance = widget.LowImportance

	return container.NewVBox(
		widget.NewCard("DNS Logs", "View DNS logs here (newest first)",
			container.NewVBox(
				container.NewHBox(
					widget.NewLabel(""), // Spacer
					clearButton,
				),
				logScroll,
			),
		),
	)
}
