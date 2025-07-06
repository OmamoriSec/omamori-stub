package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
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
		lm.logText.SetText(lm.logText.Text + message)
	})
}

func (lm *LogManager) AppendErrorLogs(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message = fmt.Sprintf("[%s] ERROR: %s\n", timestamp, message)

	fyne.Do(func() {
		lm.logText.SetText(lm.logText.Text + message)
	})
}

func (lm *LogManager) logTab() *fyne.Container {
	logScroll := container.NewScroll(lm.logText)
	logScroll.SetMinSize(fyne.NewSize(400, 400))

	return container.NewVBox(
		widget.NewCard("DNS Logs", "View DNS logs here", logScroll),
	)
}
