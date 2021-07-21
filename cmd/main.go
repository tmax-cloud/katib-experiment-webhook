package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func main() {
	log.SetOutput(os.Stdout)
	var parameters WhSvrParameters

	// get CLI Parameters
	flag.IntVar(&parameters.port, "port", 8080, "Webhook Server Port")
	flag.StringVar(&parameters.certFile, "tlsCertFile", "/etc/webhook/certs/tls.crt", "x509 Certificate File for HTTPS")
	flag.StringVar(&parameters.keyFile, "tlsKeyFile", "/etc/webhook/certs/tls.key", "x509 Certificate Private Key")	

	flag.Parse()
	

	pair, err := tls.LoadX509KeyPair(parameters.certFile, parameters.keyFile)
	if err != nil {
		log.Errorf("Failed to load key pair : %v", err)
	}

	webhookServer := WebhookServer{		
		server: &http.Server{
			Addr: fmt.Sprintf(":%v", parameters.port),
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{pair},
			},
		},
	}

	log.Info("Configuration Load Done")

	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", webhookServer.Serve)
	webhookServer.server.Handler = mux
	// start webhook server in new rountine
	go func() {
		if err := webhookServer.server.ListenAndServeTLS("", ""); err != nil {
			log.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	log.Info("Start Server in New Routine")

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Infof("Got OS shutdown signal, shutting down webhook server gracefully...")
	webhookServer.server.Shutdown(context.Background())
}

