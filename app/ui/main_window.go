package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"log"
	"omamori/app/core/channels"
	"omamori/app/core/config"
	"omamori/app/ui/assets"
	"time"
)

type OmamoriApp struct {
	app           fyne.App
	window        fyne.Window
	config        *config.Config
	serverRunning bool
	statusLabel   *widget.Label

	// Tabs
	serverManager   *ServerManager
	configManager   *ConfigManager
	siteListManager *SiteListManager
	logManager      *LogManager
}

func (o *OmamoriApp) logMessage(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)

	log.Printf("%s %s", timestamp, logEntry)
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

func (o *OmamoriApp) setup() {
	tabs := container.NewAppTabs(
		container.NewTabItem("Server Control", o.serverManager.serverControlTab()),
		container.NewTabItem("Configuration", o.configManager.configurationTab()),
		container.NewTabItem("Site List", o.siteListManager.siteListTab()),
		container.NewTabItem("Logs", o.logManager.logTab()),
	)

	statusBar := o.createStatusBar()

	content := container.NewBorder(nil, statusBar, nil, nil, tabs)
	o.window.SetContent(content)
}

func StartGUI() {
	omamori := &OmamoriApp{
		app:    app.New(),
		window: nil,
	}

	omamori.app.SetIcon(assets.ResourceOmamorilogoPng)

	omamori.window = omamori.app.NewWindow("Omamori")
	omamori.window.Resize(fyne.NewSize(800, 600))

	omamori.config = config.Global

	omamori.serverManager = omamori.createServerManager()
	omamori.configManager = omamori.createConfigManager()
	omamori.siteListManager = omamori.createSiteListManager()
	omamori.logManager = omamori.createLogManager()

	go func() {
		for data := range channels.LogEventChannel {
			payload, ok := data.Payload.(string)
			if !ok {
				continue
			}

			fyne.Do(func() {
				omamori.logManager.AppendLog(payload)
			})
		}
	}()

	omamori.setup()
	omamori.window.ShowAndRun()
}
