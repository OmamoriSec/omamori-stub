package main

import (
	"log"
	"net"
	"time"
)

func writeResp(udpConn *net.UDPConn, resp []byte, addr *net.UDPAddr) {
	_, err := udpConn.WriteToUDP(resp, addr)
	if err != nil {
		log.Println("Failed to send response:", err)
	}

}

func reloadBlockedSitesPeriodically(path string, interval time.Duration) {
	for {
		if err := loadBlockedSites(path); err != nil {
			log.Println("Failed to reload blocked sites:", err)
		}
		time.Sleep(interval)
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
		dq, err := decodeDNSQuery(receivedData)
		if err != nil {
			log.Println("Failed to decode DNS header:", err)
			writeResp(udpConn, []byte("Failed to decode DNS header"), source)
			continue
		}

		dnsResponse := lookup(dq)

		response, err := dnsResponse.encode()

		if err != nil {
			log.Println("Error encoding DNS header:", err)
		}

		writeResp(udpConn, response, source)
	}
}

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
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

	//  Load File Periodically
	go reloadBlockedSitesPeriodically("blocked_file.txt", time.Second*30)

	handleDNSRequest(udpConn)
}
