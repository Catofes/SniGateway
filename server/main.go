package main

import (
	"golang.org/x/crypto/acme/autocert"
	"crypto/tls"
	"sync"
	"net"
	"github.com/op/go-logging"
	"os"
	"io"
	"strings"
)

var log *logging.Logger

func init() {
	log = logging.MustGetLogger("example")
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter(
		`%{color}%{time:0102 15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	backendLeveled := logging.AddModuleLevel(backendFormatter)
	backendLeveled.SetLevel(logging.WARNING, "")
	logging.SetBackend(backendLeveled)

}

type TLSServer struct {
	certManager    *autocert.Manager
	ListenAddress  string
	BackendAddress string
	Domain         string
	certPath       string
	keyPath        string
}

func (s *TLSServer) Init() *TLSServer {
	SS_LOCAL_HOST := os.Getenv("SS_LOCAL_HOST")
	SS_REMOTE_HOST := os.Getenv("SS_REMOTE_HOST")
	SS_LOCAL_PORT := os.Getenv("SS_LOCAL_PORT")
	SS_REMOTE_PORT := os.Getenv("SS_REMOTE_PORT")
	SS_PLUGIN_OPTIONS := os.Getenv("SS_PLUGIN_OPTIONS")
	s.ListenAddress = SS_REMOTE_HOST + ":" + SS_REMOTE_PORT
	s.BackendAddress = SS_LOCAL_HOST + ":" + SS_LOCAL_PORT
	s.LoadOption(SS_PLUGIN_OPTIONS)
	s.certManager = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.Domain),
		Cache:      autocert.DirCache("certs"),
	}
	return s
}

func (s *TLSServer) LoadOption(option string) {
	data := strings.Split(option, ";")
	for _, v := range data {
		d := strings.Split(v, "=")
		if len(d) != 2 {
			continue
		}
		key := d[0]
		value := d[1]
		switch key {
		case "domain":
			s.Domain = value
		case "cert":
			s.certPath = value
		case "key":
			s.keyPath = value
		}
	}
}

func (s *TLSServer) Listen() {
	var config *tls.Config
	if s.certPath == "" {
		config = &tls.Config{GetCertificate: s.certManager.GetCertificate}
	} else {
		cert, err := tls.LoadX509KeyPair(s.certPath, s.keyPath)
		if err != nil {
			log.Fatalf("Load cert failed. %s", err.Error())
		}
		config = &tls.Config{}
		config.Certificates = append(config.Certificates, cert)
	}
	ln, err := tls.Listen("tcp", s.ListenAddress, config)
	if err != nil {
		log.Fatalf("Error Listen Port. %s", err.Error())
	}
	defer ln.Close()
	wg := &sync.WaitGroup{}
	for {
		conn, err := ln.Accept()
		log.Debug("Accept connection.")
		if err != nil {
			log.Warningf("Can not accept conn. %s", err.Error())
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.handleConn(conn)
		}()
	}
	wg.Wait()
}

func (s *TLSServer) handleConn(conn net.Conn) {
	defer conn.Close()
	upConn := conn.(*tls.Conn)
	err := upConn.Handshake()
	if err != nil {
		log.Debugf("TLS handshake failed. %s", err.Error())
	}
	log.Debugf("accepted: %s", conn.RemoteAddr())
	downConn, err := net.Dial("tcp", s.BackendAddress)
	if err != nil {
		log.Warningf("unable to connect to %s: %s", s.BackendAddress, err)
		return
	}
	defer downConn.Close()
	if err := s.Pipe(upConn, downConn); err != nil {
		log.Warningf("pipe failed: %s", err)
	} else {
		log.Debugf("disconnected: %s", upConn.RemoteAddr())
	}
}

func (s *TLSServer) Pipe(a, b net.Conn) error {
	done := make(chan error, 1)
	cp := func(r, w net.Conn) {
		n, err := io.Copy(w, r)
		log.Debugf("copied %d bytes from %s to %s", n, r.RemoteAddr(), w.RemoteAddr())
		switch wc := w.(type){
		case *tls.Conn:
			wc.CloseWrite()
		case *net.TCPConn:
			wc.CloseWrite()
		}
		switch rc := r.(type){
		case *tls.Conn:
		case *net.TCPConn:
			rc.CloseRead()
		}
		done <- err
	}
	go cp(a, b)
	go cp(b, a)
	err1 := <-done
	err2 := <-done
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func main() {
	(&TLSServer{}).Init().Listen()
}
