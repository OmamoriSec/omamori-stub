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
	"os"
	"time"
)

func loadConf() {

	if err := config.LoadBlockedSites(); err != nil {
		log.Println("Failed to reload blocked sites:", err)
	}

	if err := config.LoadConfig(); err != nil {
		log.Println("Failed to reload upstream conf:", err)
	}

}

var dnsJobChan = make(chan dnsJob, 500)

type dnsJob struct {
	data   []byte
	conn   *net.UDPConn
	source *net.UDPAddr
}

func startDnsWorkerPool(num int) {
	log.Println("Starting DNS worker pool: ", num)
	for i := 0; i < num; i++ {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("Recovered from panic: ", r)
				}
			}()

			for job := range dnsJobChan {
				if job.data == nil {
					// terminate
					return
				}
				handleDNSRequest(job.data, job.conn, job.source)
			}
		}()
	}
}

func startUdpServer(ctx context.Context, host string, port int) {

	noOfDnsWorkers := 500
	go startDnsWorkerPool(noOfDnsWorkers) // starting workers to handle DNS request

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Println("Failed to resolve udp address:", err)
		os.Exit(1)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Println("Failed to bind to address", err)
		os.Exit(1)
	}

	// Log when server starts to confirm it's running
	log.Println("🚀 DNS Server started on port: ", port)

	buf := make([]byte, 512)
	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down UDP server gracefully")
			_ = udpConn.Close()
			for i := 0; i < noOfDnsWorkers; i++ {
				dnsJobChan <- dnsJob{}
			}
			return
		default:
			_ = udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
			size, source, err := udpConn.ReadFromUDP(buf)
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					continue // check context again
				}
				log.Println("Failed to read UDP packet:", err)
				return
			}

			// delegating to the go routine workers to avoid blocking the server
			dnsJobChan <- dnsJob{data: append([]byte(nil), buf[:size]...), conn: udpConn, source: source}

		}
	}

}

func writeResp(udpConn *net.UDPConn, resp []byte, addr *net.UDPAddr) {
	_, err := udpConn.WriteToUDP(resp, addr)
	if err != nil {
		log.Println("Failed to write response:", err)
	}

}

func handleDNSRequest(receivedData []byte, udpConn *net.UDPConn, source *net.UDPAddr) {

	dq, err := dns.DecodeDNSQuery(receivedData)
	if err != nil {
		log.Println("Failed to decode DNS query")
		// don't want to response to malformed packets
		return
	}

	dnsResponse := dns.Lookup(dq)
	writeResp(udpConn, dnsResponse, source)
}

func main() {
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
				go startUdpServer(dnsCtx, "127.0.0.1", configData.UdpServerPort)

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
