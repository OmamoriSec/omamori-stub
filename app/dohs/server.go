// This File will expose the necessary methods for the DOHS

package dohs

import (
	"context"
	b64 "encoding/base64"
	"errors"
	"io"
	"log"
	"net/http"
	"omamori/app/core/config"
	"omamori/app/core/dns"
	"strings"
	"time"
)

var (
	publicKEY  = config.Global.CertPath
	privateKEY = config.Global.KeyPath
)

func RunHttpServer(ctx context.Context) {
	serv := &http.Server{
		Addr:    ":443",
		Handler: http.HandlerFunc(dohsHandler),
	}

	go func() {
		log.Println("Starting DOHS server on port 443")
		if err := serv.ListenAndServeTLS(publicKEY, privateKEY); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("DOHS server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	log.Println("Shutting down DOHS server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := serv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down DOHS server: %v", err)
	}
}

func dohsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s", r.Proto)
	switch r.Method {
	case "GET":
		handleGet(w, r)
	case "POST":
		handlePost(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	// extract data from url query param ?dns=
	// decode the base64URL
	// validate

	data, err := b64.URLEncoding.DecodeString(r.URL.Query().Get("dns"))
	if err != nil {
		writeError(w, 400, "Content-Type must be application/dns-message")
		return
	}
	generateDnsResponse(w, data)

}

func handlePost(w http.ResponseWriter, r *http.Request) {
	// check content-type
	// Read raw body bytes
	// validate it
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/dns-message") {
		writeError(w, 400, "Content-Type must be application/dns-message")
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, 400, "Content-Type must be application/dns-message")
		return
	}
	generateDnsResponse(w, data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}

func generateDnsResponse(w http.ResponseWriter, data []byte) {
	log.Println(data)

	dnsQuery, err := dns.DecodeDNSQuery(data)
	if err != nil {
		writeError(w, 400, "Content-Type must be application/dns-message")
		return
	}
	log.Println(dnsQuery)
	dnsResp := dns.Lookup(dnsQuery)
	resp, _ := dnsResp.Encode()
	w.Header().Set("Content-Type", "application/dns-message")
	_, _ = w.Write(resp)
}
