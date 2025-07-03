package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"omamori/app/core/config"
	"omamori/app/core/dns"
	"omamori/app/core/events"
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

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %s", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to address: %s", err)
	}

	log.Println("Listening on", udpConn.LocalAddr().String())

	return udpConn, nil

}

func handleDNSRequest(ctx context.Context, port int) {

	udpConn, err := startUdpServer(port)
	if err != nil {
		log.Println("Failed to start UDP server:", err)
		// send the error into the event channel
		events.GlobalEventChannel <- events.Event{Type: events.Error, Payload: err}
		return
	}

	buf := make([]byte, 512)

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down udp server gracefully")
			_ = udpConn.Close()
			return
		default:
			_ = udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
			size, source, err := udpConn.ReadFromUDP(buf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
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
	configData, err := config.EnsureDefaultConfig()
	if err != nil {
		log.Fatal(err)
	}

	//  Load blocked sites and conf continuously
	loadConf()

	var dnsCtx context.Context
	var dnsCancel context.CancelFunc

	var dohCtx context.Context
	var dohCancel context.CancelFunc

	go func() {
		for event := range events.GlobalEventChannel {
			switch event.Type {
			case events.StartDnsServer:
				// port argument is there for future extensibility
				if dnsCancel != nil {
					continue
				}
				dnsCtx, dnsCancel = context.WithCancel(context.Background())
				go handleDNSRequest(dnsCtx, configData.UdpServerPort)
			case events.StopDnsServer:
				if dnsCancel != nil {
					dnsCancel()
					dnsCancel = nil
				}
			case events.StartDOHServer:
				if dohCancel != nil {
					continue
				}
				dohCtx, dohCancel = context.WithCancel(context.Background())
				go dohs.RunHttpServer(dohCtx)
			case events.StopDOHServer:
				if dohCancel != nil {
					dohCancel()
					dohCancel = nil
				}
			case events.UpdateConfig:
				newConfig, ok := event.Payload.(*config.Config)
				if ok {
					err := config.UpdateConfig(newConfig)
					if err != nil {
						log.Println("Failed to save config:", err)
						events.GlobalEventChannel <- events.Event{
							Type: events.Error, Payload: err,
						}
					}
				}

			case events.UpdateSiteList:
				log.Printf("Received update site map %v", event.Payload)
			}
		}
	}()

	ui.StartGUI()
}
