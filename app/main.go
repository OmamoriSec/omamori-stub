package main

import (
	"fmt"
	"log"
	"net"
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

func loadConf(refreshAfter int, continuous ...bool) {
	isContinuous := false
	if len(continuous) > 0 {
		isContinuous = continuous[0]
	}

	for {
		if err := config.LoadBlockedSites(); err != nil {
			log.Println("Failed to reload blocked sites:", err)
		}

		if err := config.LoadConfig(); err != nil {
			log.Println("Failed to reload upstream conf:", err)
		}

		if !isContinuous {
			break
		}

		time.Sleep(time.Duration(refreshAfter) * time.Second)
	}
}

func handleDNSRequest(udpConn *net.UDPConn) {
	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			log.Println("Error receiving data:", err)
			break
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

func main() {
	configData, err := config.EnsureDefaultConfig()
	if err != nil {
		log.Fatal(err)
	}

	//  Load blocked sites and conf
	loadConf(0)

	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", configData.UdpServerPort))
	if err != nil {
		log.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Println("Failed to bind to address:", err)
		return
	}

	log.Println("Listening on", udpConn.LocalAddr().String())

	defer func(udpConn *net.UDPConn) {
		err := udpConn.Close()
		if err != nil {
			log.Println("Failed to close UDP connection:", err)
		}
	}(udpConn)

	//  Load blocked sites and conf continuously
	go loadConf(15, true)

	go dohs.RunHttpServer()

	go handleDNSRequest(udpConn)

	ui.StartGUI()
}
