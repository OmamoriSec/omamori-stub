package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"omamori/app/core/events"
	"strconv"
)

type ServerManager struct {
	app             *OmamoriApp
	startStopButton *widget.Button
	dohsEnabled     bool
	dohsCheck       *widget.Check
}

func (o *OmamoriApp) createServerManager() *ServerManager {
	return &ServerManager{
		app:         o,
		dohsEnabled: false,
	}
}

func (s *ServerManager) toggleServer() {
	if s.app.serverRunning {
		s.stopServer()
	} else {
		s.startServer()
	}
}

func (s *ServerManager) startServer() {
	s.app.serverRunning = true
	s.app.statusLabel.SetText("Server Status: Running")
	s.startStopButton.SetText("Stop Server")
	s.startStopButton.Importance = widget.DangerImportance
	s.startStopButton.Refresh()

	s.app.logMessage("Starting DNS server on port " + strconv.Itoa(s.app.config.UdpServerPort))
	events.GlobalEventChannel <- events.Event{Type: events.StartDnsServer}
}

func (s *ServerManager) stopServer() {
	events.GlobalEventChannel <- events.Event{Type: events.StopDnsServer}
	s.app.serverRunning = false
	s.app.statusLabel.SetText("Server Status: Stopped")
	s.startStopButton.SetText("Start Server")
	s.startStopButton.Importance = widget.HighImportance
	s.startStopButton.Refresh()

}

func (s *ServerManager) serverControlTab() *fyne.Container {
	s.app.statusLabel = widget.NewLabel("Server Status: Stopped")
	s.app.statusLabel.Importance = widget.MediumImportance

	s.startStopButton = widget.NewButton("Start Server", s.toggleServer)
	s.startStopButton.Importance = widget.HighImportance

	//Quick actions
	quickActionsCard := widget.NewCard("Quick Actions", "",
		container.NewVBox(
			container.NewHBox(s.startStopButton),
		),
	)

	return container.NewVBox(
		s.app.statusLabel,
		container.NewHBox(
			container.NewVBox(quickActionsCard),
		),
	)
}
