package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"omamori/app/core/channels"
	"omamori/app/core/config"
	"omamori/app/core/dns"
	"omamori/app/dohs"
	"omamori/app/ui"
	"time"
)

func writeResp(udpConn *net.UDPConn, resp []byte, addr *net.UDPAddr) {
	_, err := udpConn.WriteToUDP(resp, addr)
	if err != nil {
		log.Println("Failed to send response:", err)
	}

}

func loadConf() {

	if err := config.LoadBlockedSites(); err != nil {
		log.Println("Failed to reload blocked sites:", err)
	}

	if err := config.LoadConfig(); err != nil {
		log.Println("Failed to reload upstream conf:", err)
	}

}

func startUdpServer(port int) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", 53))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %s", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to address: %s", err)
	}

	log.Println("Listening on", udpConn.LocalAddr().String())

	// Log when server starts to confirm it's running
	channels.LogEventChannel <- channels.Event{
		Type:    channels.Log,
		Payload: fmt.Sprintf("🚀 DNS Server started on port %d", port),
	}

	return udpConn, nil

}

func handleDNSRequest(ctx context.Context, port int) {

	udpConn, err := startUdpServer(port)
	if err != nil {
		log.Println("Failed to start UDP server:", err)
		// send the error into the event channel
		channels.GlobalEventChannel <- channels.Event{Type: channels.Error, Payload: err}
		return
	}

	buf := make([]byte, 512)

	channels.LogEventChannel <- channels.Event{
		Type:    channels.Log,
		Payload: "🎯 DNS Server ready to receive queries...",
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down udp server gracefully")
			channels.LogEventChannel <- channels.Event{
				Type:    channels.Log,
				Payload: "🛑 DNS Server stopped",
			}
			_ = udpConn.Close()
			return
		default:
			_ = udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
			size, source, err := udpConn.ReadFromUDP(buf)
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					continue // check context again
				}
				log.Println("Error receiving data:", err)
				return
			}

			receivedData := buf[:size]
			dq, err := dns.DecodeDNSQuery(receivedData)
			if err != nil {
				log.Println("Failed to decode DNS packet:", err)
				// don't want to response to malformed packets
				continue
			}

			dnsResponse := dns.Lookup(dq)
			response, err := dnsResponse.Encode()

			if err != nil {
				log.Println("Error encoding DNS header:", err)
			}

			writeResp(udpConn, response, source)
		}
	}
}

func main() {
	config.InitDNSConfig("127.0.0.1")

	if err := config.CheckDNSPrivileges(); err != nil {
		log.Printf("Privilege check failed: %v", err)
		log.Println("System DNS configuration will be disabled")
	} else {
		log.Println("Sufficient privileges for DNS configuration")
	}

	configData, err := config.EnsureDefaultConfig()
	if err != nil {
		channels.LogEventChannel <- channels.Event{
			Type:    channels.Error,
			Payload: fmt.Sprintf("Failed to ensure default config: %v", err),
		}
		log.Fatal(err)
	}

	//  Load blocked sites and conf continuously
	loadConf()

	var dnsCtx context.Context
	var dnsCancel context.CancelFunc

	var dohCtx context.Context
	var dohCancel context.CancelFunc

	go func() {
		for event := range channels.GlobalEventChannel {
			switch event.Type {
			case channels.StartDnsServer:
				// port argument is there for future extensibility
				if dnsCancel != nil {
					continue
				}
				dnsCtx, dnsCancel = context.WithCancel(context.Background())
				go handleDNSRequest(dnsCtx, configData.UdpServerPort)

				if err := config.ConfigureSystemDNS(); err != nil {
					channels.LogEventChannel <- channels.Event{
						Type:    channels.Error,
						Payload: fmt.Sprintf("Failed to configure system DNS: %v", err),
					}
				}
			case channels.StopDnsServer:
				if dnsCancel != nil {
					dnsCancel()
					dnsCancel = nil
				}

				if err := config.RestoreSystemDNS(); err != nil {
					channels.LogEventChannel <- channels.Event{
						Type:    channels.Error,
						Payload: fmt.Sprintf("Failed to restore system DNS: %v", err),
					}
				}
			case channels.StartDOHServer:
				if dohCancel != nil {
					continue
				}
				dohCtx, dohCancel = context.WithCancel(context.Background())
				go dohs.RunHttpServer(dohCtx)
			case channels.StopDOHServer:
				if dohCancel != nil {
					dohCancel()
					dohCancel = nil
				}
			case channels.UpdateConfig:
				newConfig, ok := event.Payload.(*config.Config)
				if ok {
					err := config.UpdateConfig(newConfig)
					if err != nil {
						log.Println("Failed to save config:", err)
						channels.GlobalEventChannel <- channels.Event{
							Type: channels.Error, Payload: err,
						}
					}
				}

			case channels.UpdateSiteList:
				payload := event.Payload.(map[string]interface{})
				err := config.UpdateSiteList(payload["operation"].(string), payload["siteData"].(config.SiteData))
				if err != nil {
					log.Println("Failed to update site list:", err)
					channels.GlobalEventChannel <- channels.Event{
						Type: channels.Error, Payload: err,
					}
				}
			}
		}
	}()

	ui.StartGUI()
}
