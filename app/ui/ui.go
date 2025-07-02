package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"log"
	"net"
	"omamori/app/core/config"
	"omamori/app/core/events"
	"strconv"
	"time"
)

type OmamoriApp struct {
	app             fyne.App
	window          fyne.Window
	config          *config.Config
	serverRunning   bool
	statusLabel     *widget.Label
	startStopButton *widget.Button
	blockedSites    []string
	dohsEnabled     bool

	// Form fields
	upstream1Entry *widget.Entry
	upstream2Entry *widget.Entry
	mapFileEntry   *widget.Entry
	dohsCheck      *widget.Check

	configChanged bool
}

func (o *OmamoriApp) createStatusBar() *fyne.Container {
	statusText := widget.NewLabel("Ready")
	statusText.Importance = widget.LowImportance

	return container.NewHBox(
		statusText,
		layout.NewSpacer(),
		widget.NewLabel("Omamori DNS Server v1.0"),
	)
}

func StartGUI() {
	omamori := &OmamoriApp{
		app:    app.New(),
		window: nil,
	}
	omamori.app.SetIcon(theme.ComputerIcon())
	omamori.window = omamori.app.NewWindow("Omamori")
	omamori.window.Resize(fyne.NewSize(800, 600))

	omamori.config = config.Global

	omamori.setup()
	omamori.window.ShowAndRun()
}

func (o *OmamoriApp) toggleServer() {
	if o.serverRunning {
		o.stopServer()
	} else {
		o.startServer()
	}
}

func (o *OmamoriApp) startServer() {
	o.serverRunning = true
	o.statusLabel.SetText("Server Status: Running")
	o.startStopButton.SetText("Stop Server")
	o.startStopButton.Importance = widget.DangerImportance
	o.startStopButton.Refresh()

	o.logMessage("Starting DNS server on port " + strconv.Itoa(o.config.UdpServerPort))
	events.GlobalEventChannel <- events.Event{Type: events.StartDnsServer}
}

func (o *OmamoriApp) stopServer() {
	events.GlobalEventChannel <- events.Event{Type: events.StopDnsServer}
	o.serverRunning = false
	o.statusLabel.SetText("Server Status: Stopped")
	o.startStopButton.SetText("Start Server")
	o.startStopButton.Importance = widget.HighImportance
	o.startStopButton.Refresh()

}

func (o *OmamoriApp) logMessage(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)

	log.Printf("%s %s", timestamp, logEntry)
}

func (o *OmamoriApp) saveConfig() {
	newConfig := &config.Config{
		Upstream1: o.upstream1Entry.Text,
		Upstream2: o.upstream2Entry.Text,
		MapFile:   o.mapFileEntry.Text,
	}

	// TODO: currently for simplicity no option to update file paths
	if !o.configChanged {
		return
	}
	o.logMessage("Config changed. Sending update event...")
	events.GlobalEventChannel <- events.Event{
		Type:    events.UpdateConfig,
		Payload: newConfig,
	}
}

func (o *OmamoriApp) serverControlTab() *fyne.Container {
	o.statusLabel = widget.NewLabel("Server Status: Stopped")
	o.statusLabel.Importance = widget.MediumImportance

	o.startStopButton = widget.NewButton("Start Server", o.toggleServer)
	o.startStopButton.Importance = widget.HighImportance

	//Quick actions
	quickActionsCard := widget.NewCard("Quick Actions", "",
		container.NewVBox(
			container.NewHBox(o.startStopButton),
		),
	)

	return container.NewVBox(
		o.statusLabel,
		container.NewHBox(
			container.NewVBox(quickActionsCard),
		),
	)
}

func (o *OmamoriApp) bindIPEntry(entry *widget.Entry, initial string) *widget.Entry {
	entry.SetText(initial)
	entry.OnChanged = func(input string) {
		if net.ParseIP(input) == nil {
			entry.SetValidationError(fmt.Errorf("invalid IP address"))
			o.configChanged = false
		} else {
			entry.SetValidationError(nil)
			o.configChanged = true
		}
	}
	return entry
}

func (o *OmamoriApp) bindCheck(check *widget.Check, initialValue bool, onToggle func(bool)) *widget.Check {
	check.SetChecked(initialValue)
	check.OnChanged = func(val bool) {
		o.configChanged = true
		onToggle(val)
	}
	return check
}

func (o *OmamoriApp) configurationTab() *container.Scroll {
	// Upstream DNS entries
	o.upstream1Entry = o.bindIPEntry(widget.NewEntry(), o.config.Upstream1)
	o.upstream2Entry = o.bindIPEntry(widget.NewEntry(), o.config.Upstream2)

	// Map file path
	o.mapFileEntry = widget.NewEntry()
	o.mapFileEntry.SetText(o.config.MapFile)

	mapFileBrowseButton := widget.NewButton("Browse", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				o.mapFileEntry.SetText(reader.URI().Path())
				if o.config.MapFile != reader.URI().Path() {
					o.configChanged = true
				}
				reader.Close()
			}
		}, o.window)
	})

	// DoHS checkbox (tracked separately too)
	o.dohsCheck = o.bindCheck(
		widget.NewCheck("Start DOHS", nil),
		o.dohsEnabled,
		func(enabled bool) {
			if enabled {
				events.GlobalEventChannel <- events.Event{Type: events.StartDOHServer}
			} else {
				events.GlobalEventChannel <- events.Event{Type: events.StopDOHServer}
			}
			o.dohsEnabled = enabled
		},
	)

	saveButton := widget.NewButton("Save Configuration", o.saveConfig)
	saveButton.Importance = widget.HighImportance

	form := container.NewVBox(
		widget.NewCard("Upstream DNS Server", "",
			container.NewVBox(
				widget.NewFormItem("Primary Upstream DNS", o.upstream1Entry).Widget,
				widget.NewFormItem("Secondary Upstream DNS", o.upstream2Entry).Widget,
			),
		),
		widget.NewCard("Site Map File", "",
			container.NewVBox(
				widget.NewFormItem("Blocked Sites File",
					container.NewBorder(nil, nil, nil, mapFileBrowseButton, o.mapFileEntry),
				).Widget,
			),
		),
		widget.NewCard("DOHS Settings", "",
			container.NewVBox(o.dohsCheck),
		),
		container.NewHBox(layout.NewSpacer(), saveButton),
	)

	return container.NewScroll(form)
}

func (o *OmamoriApp) setup() {
	tabs := container.NewAppTabs(
		container.NewTabItem("Server Control", o.serverControlTab()),
		container.NewTabItem("Configuration", o.configurationTab()), //widget.NewLabel("Settings will be here soon!")),
		container.NewTabItem("Site map", widget.NewLabel("Settings will be here soon!")),
		container.NewTabItem("Logs", widget.NewLabel("Settings will be here soon!")),
	)

	statusBar := o.createStatusBar()

	content := container.NewBorder(nil, statusBar, nil, nil, tabs)
	o.window.SetContent(content)
}
