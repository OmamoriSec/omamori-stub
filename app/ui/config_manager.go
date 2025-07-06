package ui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"net"
	"omamori/app/core/channels"
	"omamori/app/core/config"
)

type ConfigManager struct {
	app            *OmamoriApp
	upstream1Entry *widget.Entry
	upstream2Entry *widget.Entry
	mapFileEntry   *widget.Entry
	configChanged  bool
}

func (o *OmamoriApp) createConfigManager() *ConfigManager {
	return &ConfigManager{
		app:           o,
		configChanged: false,
	}
}

func (c *ConfigManager) bindCheck(check *widget.Check, initialValue bool, onToggle func(bool)) *widget.Check {
	check.SetChecked(initialValue)
	check.OnChanged = func(val bool) {
		c.configChanged = true
		onToggle(val)
	}
	return check
}

func (c *ConfigManager) bindIPEntry(entry *widget.Entry, initial string) *widget.Entry {
	entry.SetText(initial)
	entry.OnSubmitted = func(input string) {
		c.validateIP(entry, input)
	}

	entry.OnChanged = func(input string) {
		c.configChanged = true
		entry.SetValidationError(nil)
	}
	return entry
}

func (c *ConfigManager) validateIP(entry *widget.Entry, input string) {
	if net.ParseIP(input) == nil && input != "" {
		entry.SetValidationError(fmt.Errorf("invalid IP address"))
		channels.LogEventChannel <- channels.Event{
			Type:    channels.Error,
			Payload: fmt.Sprintf("Invalid IP address entered: %s", input),
		}
		c.configChanged = false
	} else {
		entry.SetValidationError(nil)
		c.configChanged = true
	}
}

func (c *ConfigManager) saveConfig() {
	newConfig := &config.Config{
		Upstream1: c.upstream1Entry.Text,
		Upstream2: c.upstream2Entry.Text,
		MapFile:   c.mapFileEntry.Text,
	}

	// TODO: currently for simplicity no option to update file paths
	if !c.configChanged {
		return
	}
	c.app.logMessage("Config changed. Sending update event...")
	channels.GlobalEventChannel <- channels.Event{
		Type:    channels.UpdateConfig,
		Payload: newConfig,
	}
}

func (c *ConfigManager) configurationTab() *container.Scroll {
	// Upstream DNS entries
	c.upstream1Entry = c.bindIPEntry(widget.NewEntry(), c.app.config.Upstream1)
	c.upstream2Entry = c.bindIPEntry(widget.NewEntry(), c.app.config.Upstream2)

	// Map file path
	c.mapFileEntry = widget.NewEntry()
	c.mapFileEntry.SetText(c.app.config.MapFile)

	mapFileBrowseButton := widget.NewButton("Browse", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				c.mapFileEntry.SetText(reader.URI().Path())
				if c.app.config.MapFile != reader.URI().Path() {
					c.configChanged = true
				}
				err := reader.Close()
				if err != nil {
					return
				}
			}
		}, c.app.window)
	})

	// DoHS checkbox (tracked separately too)
	c.app.serverManager.dohsCheck = c.bindCheck(
		widget.NewCheck("Start DOHS", nil),
		c.app.serverManager.dohsEnabled,
		func(enabled bool) {
			if enabled {
				channels.GlobalEventChannel <- channels.Event{Type: channels.StartDOHServer}
			} else {
				channels.GlobalEventChannel <- channels.Event{Type: channels.StopDOHServer}
			}
			c.app.serverManager.dohsEnabled = enabled
		},
	)

	saveButton := widget.NewButton("Save Configuration", c.saveConfig)
	saveButton.Importance = widget.HighImportance

	form := container.NewVBox(
		widget.NewCard("Upstream DNS Server", "",
			container.NewVBox(
				widget.NewFormItem("Primary Upstream DNS", c.upstream1Entry).Widget,
				widget.NewFormItem("Secondary Upstream DNS", c.upstream2Entry).Widget,
			),
		),
		widget.NewCard("Site Map File", "",
			container.NewVBox(
				widget.NewFormItem("Blocked Sites File",
					container.NewBorder(nil, nil, nil, mapFileBrowseButton, c.mapFileEntry),
				).Widget,
			),
		),
		widget.NewCard("DOHS Settings", "",
			container.NewVBox(c.app.serverManager.dohsCheck),
		),
		container.NewHBox(layout.NewSpacer(), saveButton),
	)

	return container.NewScroll(form)
}
