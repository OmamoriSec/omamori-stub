package ui

import (
	"fmt"
	"net"
	"omamori/app/core/config"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"omamori/app/core/channels"
)

type SiteListManager struct {
	app           *OmamoriApp
	blockedList   *widget.List
	customDNSList *widget.List
	blockedSites  []string
	customDNS     []CustomDNSEntry

	// Filtered data for search
	filteredBlockedSites []string

	// Input fields
	blockDomainEntry *widget.Entry
	dnsNameEntry     *widget.Entry
	dnsIPEntry       *widget.Entry
	searchEntry      *widget.Entry
}

type CustomDNSEntry struct {
	Domain string
	IP     string
}

var siteUpdatePayload = make(map[string]interface{}) // Global payload for site updates keys: [operation, siteData]

func (o *OmamoriApp) createSiteListManager() *SiteListManager {
	manager := &SiteListManager{
		app:                  o,
		blockedSites:         make([]string, 0),
		customDNS:            make([]CustomDNSEntry, 0), // Start empty
		filteredBlockedSites: make([]string, 0),
	}

	manager.loadBlockedSites()
	manager.loadCustomDNS()
	manager.setupUI()

	return manager
}

func (s *SiteListManager) setupUI() {
	s.blockDomainEntry = widget.NewEntry()
	s.blockDomainEntry.SetPlaceHolder("Enter domain to block (e.g., example.com)")

	s.dnsNameEntry = widget.NewEntry()
	s.dnsNameEntry.SetPlaceHolder("Domain name (e.g., myserver.local)")

	s.dnsIPEntry = widget.NewEntry()
	s.dnsIPEntry.SetPlaceHolder("IP address")

	// Search field for blocked domains
	s.searchEntry = widget.NewEntry()
	s.searchEntry.SetPlaceHolder("Search blocked domains...")
	s.searchEntry.OnChanged = s.filterBlockedSites

	// Initialize filtered list with all blocked sites
	s.filteredBlockedSites = make([]string, len(s.blockedSites))
	copy(s.filteredBlockedSites, s.blockedSites)

	// Setup lists
	s.setupBlockedList()
	s.setupCustomDNSList()
}

func (s *SiteListManager) setupBlockedList() {
	s.blockedList = widget.NewList(
		func() int {
			return len(s.filteredBlockedSites)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Template Domain")
			label.TextStyle = fyne.TextStyle{}

			removeBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			removeBtn.Importance = widget.LowImportance

			return container.NewBorder(nil, nil, nil, removeBtn, label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			border, ok := obj.(*fyne.Container)
			if !ok {
				return
			}

			label := border.Objects[0].(*widget.Label)
			button := border.Objects[1].(*widget.Button)

			if id < len(s.filteredBlockedSites) {
				domain := s.filteredBlockedSites[id]
				label.SetText(domain)
				button.OnTapped = func() {
					// Find the original index in the full list
					originalIndex := -1
					for i, site := range s.blockedSites {
						if site == domain {
							originalIndex = i
							break
						}
					}
					if originalIndex != -1 {
						s.removeBlockedSite(originalIndex)
					}
				}
			}
		},
	)
}

func (s *SiteListManager) setupCustomDNSList() {
	s.customDNSList = widget.NewList(
		func() int {
			return len(s.customDNS)
		},
		func() fyne.CanvasObject {
			domainLabel := widget.NewLabel("domain.example")
			arrow := widget.NewLabel("→")
			ipLabel := widget.NewLabel("192.168.1.1")

			removeBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
			removeBtn.Importance = widget.LowImportance

			content := container.NewHBox(domainLabel, arrow, ipLabel)
			return container.NewBorder(nil, nil, nil, removeBtn, content)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			border, ok := obj.(*fyne.Container)
			if !ok {
				return
			}

			contentBox := border.Objects[0].(*fyne.Container)
			domainLabel := contentBox.Objects[0].(*widget.Label)
			ipLabel := contentBox.Objects[2].(*widget.Label)
			button := border.Objects[1].(*widget.Button)

			if id < len(s.customDNS) {
				domainLabel.SetText(s.customDNS[id].Domain)
				ipLabel.SetText(s.customDNS[id].IP)
				button.OnTapped = func() {
					s.removeCustomDNS(id)
				}
			}
		},
	)
}

func (s *SiteListManager) filterBlockedSites(searchText string) {
	searchText = strings.ToLower(strings.TrimSpace(searchText))

	if searchText == "" {
		// Show all sites when search is empty
		s.filteredBlockedSites = make([]string, len(s.blockedSites))
		copy(s.filteredBlockedSites, s.blockedSites)
	} else {
		// Filter sites that contain the search text
		s.filteredBlockedSites = s.filteredBlockedSites[:0] // Clear slice but keep capacity
		for _, site := range s.blockedSites {
			if strings.Contains(strings.ToLower(site), searchText) {
				s.filteredBlockedSites = append(s.filteredBlockedSites, site)
			}
		}
	}

	s.blockedList.Refresh()
}

func (s *SiteListManager) addBlockedSite() {
	domain := strings.TrimSpace(s.blockDomainEntry.Text)
	if domain == "" {
		dialog.ShowError(fmt.Errorf("please enter a domain name"), s.app.window)
		return
	}

	// Basic domain validation
	if !s.isValidDomain(domain) {
		dialog.ShowError(fmt.Errorf("invalid domain format"), s.app.window)
		return
	}

	// Check if already exists
	for _, existing := range s.blockedSites {
		if existing == domain {
			dialog.ShowError(fmt.Errorf("domain already blocked"), s.app.window)
			return
		}
	}

	s.blockedSites = append(s.blockedSites, domain)
	// Update filtered list
	s.filterBlockedSites(s.searchEntry.Text)
	s.blockDomainEntry.SetText("")

	// Send event to update the radix tree and save to file
	siteUpdatePayload["operation"] = "add"
	siteUpdatePayload["siteData"] = config.SiteData{
		IP:     "0.0.0.0",
		Domain: domain,
	}

	channels.GlobalEventChannel <- channels.Event{
		Type:    channels.UpdateSiteList,
		Payload: siteUpdatePayload,
	}

	s.app.logMessage(fmt.Sprintf("Added blocked domain: %s", domain))
}

func (s *SiteListManager) addCustomDNS() {
	domain := strings.TrimSpace(s.dnsNameEntry.Text)
	ip := strings.TrimSpace(s.dnsIPEntry.Text)

	if domain == "" || ip == "" {
		dialog.ShowError(fmt.Errorf("please enter both domain and IP address"), s.app.window)
		return
	}

	// Validate domain and IP
	if !s.isValidDomain(domain) {
		dialog.ShowError(fmt.Errorf("invalid domain format"), s.app.window)
		return
	}

	if net.ParseIP(ip) == nil {
		dialog.ShowError(fmt.Errorf("invalid IP address format"), s.app.window)
		return
	}

	// Check if already exists
	for _, existing := range s.customDNS {
		if existing.Domain == domain {
			dialog.ShowError(fmt.Errorf("domain already has custom DNS entry"), s.app.window)
			return
		}
	}

	s.customDNS = append(s.customDNS, CustomDNSEntry{Domain: domain, IP: ip})
	s.customDNSList.Refresh()
	s.dnsNameEntry.SetText("")
	s.dnsIPEntry.SetText("")

	// Send event to update the custom DNS mapping
	siteUpdatePayload["operation"] = "add"
	siteUpdatePayload["siteData"] = config.SiteData{
		Domain: domain,
		IP:     ip,
	}

	channels.GlobalEventChannel <- channels.Event{
		Type:    channels.UpdateSiteList,
		Payload: siteUpdatePayload,
	}

	s.app.logMessage(fmt.Sprintf("Added custom DNS: %s -> %s", domain, ip))
}

func (s *SiteListManager) removeBlockedSite(index int) {
	if index >= 0 && index < len(s.blockedSites) {
		domain := s.blockedSites[index]
		s.blockedSites = append(s.blockedSites[:index], s.blockedSites[index+1:]...)
		// Update filtered list
		s.filterBlockedSites(s.searchEntry.Text)

		// Send event to update the radix tree and save to file
		siteUpdatePayload["operation"] = "delete"
		siteUpdatePayload["siteData"] = config.SiteData{
			Domain: domain,
		}

		channels.GlobalEventChannel <- channels.Event{
			Type:    channels.UpdateSiteList,
			Payload: siteUpdatePayload,
		}

		s.app.logMessage(fmt.Sprintf("Removed blocked domain: %s", domain))
	}
}

func (s *SiteListManager) removeCustomDNS(index int) {
	if index >= 0 && index < len(s.customDNS) {
		entry := s.customDNS[index]
		s.customDNS = append(s.customDNS[:index], s.customDNS[index+1:]...)
		s.customDNSList.Refresh()

		// Send event to update the custom DNS mapping
		siteUpdatePayload["operation"] = "delete"
		siteUpdatePayload["siteData"] = config.SiteData{
			Domain: entry.Domain,
			IP:     entry.IP,
		}

		channels.GlobalEventChannel <- channels.Event{
			Type:    channels.UpdateSiteList,
			Payload: siteUpdatePayload,
		}

		s.app.logMessage(fmt.Sprintf("Removed custom DNS: %s -> %s", entry.Domain, entry.IP))
	}
}

func (s *SiteListManager) isValidDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}

	// Basic domain validation
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}

	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			return false
		}
	}

	return true
}

func (s *SiteListManager) loadBlockedSites() {
	// Load from the main blocked file (StevenBlack hosts list)
	blockedFileList := config.ListSiteMap()

	for i := 0; i < len(blockedFileList); i++ {
		line := blockedFileList[i]
		if line.IP == "0.0.0.0" || line.IP == "::" {
			s.blockedSites = append(s.blockedSites, line.Domain)
		}
	}

	s.app.logMessage(fmt.Sprintf("Loaded %d blocked domains from %s", len(s.blockedSites), s.app.config.MapFile))
}

func (s *SiteListManager) loadCustomDNS() {
	// Load from map.txt (custom DNS mappings)
	fileList := config.ListSiteMap()

	for i := 0; i < len(fileList); i++ {
		line := fileList[i]
		if line.IP == "0.0.0.0" || line.IP == "::" {
			continue
		}
		s.customDNS = append(s.customDNS, CustomDNSEntry{
			Domain: line.Domain,
			IP:     line.IP,
		})
	}
}

func (s *SiteListManager) blockDNSTab() *fyne.Container {
	addBlockedButton := widget.NewButton("Block Domain", s.addBlockedSite)
	addBlockedButton.Importance = widget.DangerImportance

	// Use container.NewBorder to make the input field stretch
	blockedInputContainer := container.NewBorder(nil, nil, nil, addBlockedButton, s.blockDomainEntry)

	// Make search field stretch too
	searchContainer := container.NewBorder(nil, nil, widget.NewIcon(theme.SearchIcon()), nil, s.searchEntry)

	blockedScrollContainer := container.NewScroll(s.blockedList)
	blockedScrollContainer.SetMinSize(fyne.NewSize(400, 400))

	statsLabel := widget.NewLabel(fmt.Sprintf("Total Blocked Domains: %d", len(s.blockedSites)))
	statsLabel.Importance = widget.MediumImportance

	blockedSitesCard := widget.NewCard("Blocked Domains", "Domains that will be blocked by the DNS server",
		container.NewVBox(
			blockedInputContainer,
			searchContainer,
			blockedScrollContainer,
		),
	)

	return container.NewVBox(
		statsLabel,
		widget.NewSeparator(),
		blockedSitesCard,
	)
}

func (s *SiteListManager) customDNSTab() *fyne.Container {
	addDNSButton := widget.NewButton("Add DNS Entry", s.addCustomDNS)
	addDNSButton.Importance = widget.HighImportance
	// Set explicit minimum sizes for better control
	s.dnsNameEntry.Resize(fyne.NewSize(300, 0)) // Make domain field much wider
	// Create a custom container with manual positioning
	inputContainer := container.NewWithoutLayout(
		s.dnsNameEntry,
		container.NewHBox(widget.NewLabel("→")),
		s.dnsIPEntry,
		addDNSButton,
	)

	// Position elements manually for precise control
	inputContainer.Add(s.dnsNameEntry)
	s.dnsNameEntry.Move(fyne.NewPos(0, 0))
	s.dnsNameEntry.Resize(fyne.NewSize(300, 40))

	arrowLabel := widget.NewLabel("→")
	inputContainer.Add(arrowLabel)
	arrowLabel.Move(fyne.NewPos(300, 5))

	inputContainer.Add(s.dnsIPEntry)
	s.dnsIPEntry.Move(fyne.NewPos(330, 0))
	s.dnsIPEntry.Resize(fyne.NewSize(200, 40))

	inputContainer.Add(addDNSButton)
	addDNSButton.Move(fyne.NewPos(540, 0))
	addDNSButton.Resize(fyne.NewSize(120, 40))

	// Set container size
	inputContainer.Resize(fyne.NewSize(670, 50))

	customDNSScrollContainer := container.NewScroll(s.customDNSList)
	customDNSScrollContainer.SetMinSize(fyne.NewSize(400, 400))

	statsLabel := widget.NewLabel(fmt.Sprintf("Custom DNS Entries: %d", len(s.customDNS)))
	statsLabel.Importance = widget.MediumImportance

	customDNSCard := widget.NewCard("Custom DNS Mappings", "Map domains to specific IP addresses",
		container.NewVBox(
			inputContainer,
			customDNSScrollContainer,
		),
	)

	return container.NewVBox(
		statsLabel,
		widget.NewSeparator(),
		customDNSCard,
	)
}

func (s *SiteListManager) createUI() *fyne.Container {

	subTabs := container.NewAppTabs(
		container.NewTabItem("Blocked Sites", s.blockDNSTab()),
		container.NewTabItem("Custom DNS", s.customDNSTab()),
	)

	subTabs.SetTabLocation(container.TabLocationLeading)

	return container.NewBorder(nil, nil, nil, nil, subTabs)
}

func (s *SiteListManager) siteListTab() *fyne.Container {
	return s.createUI()
}
