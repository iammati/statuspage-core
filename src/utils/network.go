package utils

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"infraops.dev/statuspage-core/config"
)

func HostMetrics(hostname string) (dnsResolutionTime, tcpConnectionTime time.Duration, reachable bool) {
	host, port, err := net.SplitHostPort(hostname)
	if err != nil {
		host = hostname
		port = "443"
	}

	start := time.Now()
	ips, err := net.LookupIP(host)
	dnsResolutionTime = time.Since(start)
	if err != nil || len(ips) == 0 {
		return dnsResolutionTime, 0, false
	}

	resolvedHost := net.JoinHostPort(ips[0].String(), port)
	conn, err := net.DialTimeout("tcp", resolvedHost, 5*time.Second)
	// Subtract DNS time to retrieve true TCP time
	tcpConnectionTime = time.Since(start) - dnsResolutionTime
	if err != nil {
		return dnsResolutionTime, tcpConnectionTime, false
	}
	conn.Close()
	return dnsResolutionTime, tcpConnectionTime, true
}

func FetchCertInfo(host string) ([]CertInfo, error) {
	hostName := strings.Split(host, ":")[0]
	conn, err := tls.Dial("tcp", host, &tls.Config{
		RootCAs:    config.RootCAs,
		ServerName: hostName,
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var certInfos []CertInfo
	for _, cert := range conn.ConnectionState().PeerCertificates {
		certInfos = append(certInfos, CertInfo{
			Issuer:     cert.Issuer.String(),
			Subject:    cert.Subject.String(),
			Expiration: cert.NotAfter.Format(time.RFC3339),
			Valid:      cert.NotAfter.After(time.Now()),
		})
	}
	return certInfos, nil
}

func HttpError(w http.ResponseWriter, errorMsg string, code int) {
	http.Error(w, errorMsg, code)
}

func JsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		return
	}
	_, writeErr := w.Write(jsonData)
	if writeErr != nil {
		log.Printf("Failed to write JSON response: %v", writeErr)
	}
}

type CertInfo struct {
	Issuer     string `json:"issuer"`
	Subject    string `json:"subject"`
	Expiration string `json:"expiration"`
	Valid      bool   `json:"valid"`
}
