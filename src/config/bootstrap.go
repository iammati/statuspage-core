package config

import (
	"crypto/x509"
	"log"
	"os"
	"path/filepath"
)

var RootCAs *x509.CertPool

func Bootstrap() {
	var err error
	RootCAs, err = x509.SystemCertPool()
	if err != nil || RootCAs == nil {
		RootCAs = x509.NewCertPool()
		log.Println("Using new cert pool.")
	}

	os.Mkdir("."+string(filepath.Separator)+"logs", 0700)
}
