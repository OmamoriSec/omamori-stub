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
	"omamori/app/core/config"
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
	portEntry      *widget.Entry
	mapFileEntry   *widget.Entry
	certPathEntry  *widget.Entry
	keyPathEntry   *widget.Entry
	dohsCheck      *widget.Check
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

	omamori.config = config.NewConfig()

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

	// TODO: Start actual DNS server here
}

func (o *OmamoriApp) stopServer() {
	o.serverRunning = false
	o.statusLabel.SetText("Server Status: Stopped")
	o.startStopButton.SetText("Start Server")
	o.startStopButton.Importance = widget.HighImportance
	o.startStopButton.Refresh()

	o.logMessage("DNS server stopped")
}

func (o *OmamoriApp) testDNSQuery() {
	domainEntry := widget.NewEntry()
	domainEntry.SetPlaceHolder("Enter domain to test (e.g., google.com)")

	dialog.ShowForm("Test DNS Query", "Test", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Domain", domainEntry),
	}, func(confirmed bool) {
		if confirmed && domainEntry.Text != "" {
			o.performDNSTest(domainEntry.Text)
		}
	}, o.window)
}

func (o *OmamoriApp) performDNSTest(domain string) {
	o.logMessage(fmt.Sprintf("Testing DNS query for: %s", domain))

	// TODO: Implement actual DNS test query

	o.logMessage(fmt.Sprintf("DNS query test completed for: %s", domain))
}

func (o *OmamoriApp) logMessage(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)

	log.Printf("%s %s", timestamp, logEntry)
}

func (o *OmamoriApp) saveConfig() {
	// Validate and save the configuration
}

func (o *OmamoriApp) serverControlTab() *fyne.Container {
	o.statusLabel = widget.NewLabel("Server Status: Stopped")
	o.statusLabel.Importance = widget.MediumImportance

	o.startStopButton = widget.NewButton("Start Server", o.toggleServer)
	o.startStopButton.Importance = widget.HighImportance

	testButton := widget.NewButton("Test DNS Query", o.testDNSQuery)

	//Quick actions
	quickActionsCard := widget.NewCard("Quick Actions", "",
		container.NewVBox(
			container.NewHBox(o.startStopButton, testButton),
		),
	)

	return container.NewVBox(
		o.statusLabel,
		container.NewHBox(
			container.NewVBox(quickActionsCard),
		),
	)
}

func (o *OmamoriApp) configurationTab() *container.Scroll {
	// upstream server
	o.upstream1Entry = widget.NewEntry()
	o.upstream1Entry.SetText(o.config.Upstream1)

	o.upstream2Entry = widget.NewEntry()
	o.upstream2Entry.SetText(o.config.Upstream2)

	o.portEntry = widget.NewEntry()
	o.portEntry.SetText(strconv.Itoa(o.config.UdpServerPort))
	portLabel := widget.NewLabel("Port:")
	portHbox := container.NewBorder(nil, nil, portLabel, nil, o.portEntry)

	// DoHS settings
	o.certPathEntry = widget.NewEntry()
	o.certPathEntry.SetText(o.config.CertPath)
	certBrowseButton := widget.NewButton("Browse", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				o.certPathEntry.SetText(reader.URI().Path())
				reader.Close()
			}
		}, o.window)
	})
	certHbox := container.NewBorder(nil, nil, nil, certBrowseButton, o.certPathEntry)

	o.keyPathEntry = widget.NewEntry()
	o.keyPathEntry.SetText(o.config.KeyPath)
	keyBrowseButton := widget.NewButton("Browse", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				o.keyPathEntry.SetText(reader.URI().Path())
				reader.Close()
			}
		}, o.window)
	})
	keyHbox := container.NewBorder(nil, nil, nil, keyBrowseButton, o.keyPathEntry)

	o.dohsCheck = widget.NewCheck("Enable DoH Server", func(enabled bool) {
		o.dohsEnabled = enabled
	})

	saveButton := widget.NewButton("Save Configuration", o.saveConfig)
	saveButton.Importance = widget.HighImportance

	form := container.NewVBox(
		widget.NewCard("Upstream DNS Server", "",
			container.NewVBox(
				widget.NewFormItem("Primary Upstream DNS", o.upstream1Entry).Widget,
				widget.NewFormItem("Secondary Upstream DNS", o.upstream2Entry).Widget,
				portHbox,
			),
		),
		widget.NewCard("HTTPS / DoH Settings", "",
			container.NewVBox(
				o.dohsCheck,
				widget.NewFormItem("Certificate Path", certHbox).Widget,
				widget.NewFormItem("Private Key Path", keyHbox).Widget,
			),
		),
		container.NewHBox(layout.NewSpacer(), saveButton),
	)

	return container.NewScroll(form)
}

func (o *OmamoriApp) setup() {
	tabs := container.NewAppTabs(
		container.NewTabItem("Server Control", o.serverControlTab()),
		container.NewTabItem("Configuration", o.configurationTab()), //widget.NewLabel("Settings will be here soon!")),
		container.NewTabItem("Blocked Sites", widget.NewLabel("Settings will be here soon!")),
		container.NewTabItem("Custom DNS", widget.NewLabel("Settings will be here soon!")),
		container.NewTabItem("Logs", widget.NewLabel("Settings will be here soon!")),
	)

	statusBar := o.createStatusBar()

	content := container.NewBorder(nil, statusBar, nil, nil, tabs)
	o.window.SetContent(content)
}
