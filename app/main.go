package main

import (
	"fmt"
	"net"
)

func writeResp(udpConn *net.UDPConn, resp []byte, addr *net.UDPAddr) {
	_, err := udpConn.WriteToUDP(resp, addr)
	if err != nil {
		fmt.Println("Failed to send response:", err)
	}

}

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}

	fmt.Println("Listening on", udpConn.LocalAddr().String())

	defer func(udpConn *net.UDPConn) {
		err := udpConn.Close()
		if err != nil {
			fmt.Println("Failed to close UDP connection:", err)
		}
	}(udpConn)

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := buf[:size]
		dq, err := DecodeDNSQuery(receivedData)
		if err != nil {
			fmt.Println("Failed to decode DNS header:", err)
			writeResp(udpConn, []byte("Failed to decode DNS header"), source)
			continue
		}

		// update header according to answer

		dq.Header.QDCOUNT = 1
		dq.Header.ARCOUNT = 0
		dq.Header.ANCOUNT = 1
		dq.Header.FLAGS = dq.Header.FLAGS | 1<<15

		// create answer
		dq.Answer = &DNSAnswer{dq.Questions.Name,
			dq.Questions.Type,
			dq.Questions.Class,
			60,
			1 << 2,
			net.ParseIP("8.8.8.8").To4(),
		}

		response, err := dq.Encode()

		if err != nil {
			fmt.Println("Error encoding DNS header:", err)
		}

		writeResp(udpConn, response, source)
	}
}
