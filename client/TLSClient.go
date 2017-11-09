package TLSClient

import (
	"github.com/op/go-logging"
	"os"
	"net"
	"sync"
	"io"
	"crypto/tls"
	"strings"
	"regexp"
)

var log *logging.Logger
var Log *logging.Logger

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
	Log = log
}

type TLSClient struct {
	ListenAddress  string
	BackendAddress string
	Domain         string
	VPNMode        bool
}

func (s *TLSClient) Init() *TLSClient {
	SS_LOCAL_HOST := os.Getenv("SS_LOCAL_HOST")
	SS_REMOTE_HOST := os.Getenv("SS_REMOTE_HOST")
	SS_LOCAL_PORT := os.Getenv("SS_LOCAL_PORT")
	SS_REMOTE_PORT := os.Getenv("SS_REMOTE_PORT")
	SS_PLUGIN_OPTIONS := os.Getenv("SS_PLUGIN_OPTIONS")
	s.BackendAddress = SS_REMOTE_HOST + ":" + SS_REMOTE_PORT
	s.ListenAddress = SS_LOCAL_HOST + ":" + SS_LOCAL_PORT
	ip_reg := `(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)`
	if ok, _ := regexp.MatchString(ip_reg, SS_REMOTE_PORT); !ok {
		s.Domain = SS_REMOTE_HOST
	}
	s.VPNMode = true
	s.LoadOption(SS_PLUGIN_OPTIONS)
	//s.BackendAddress = SS_REMOTE_HOST + ":" + SS_REMOTE_PORT
	return s
}

func String2Bool(input string) bool {
	switch input {
	case "false":
		return false
	case "0":
		return false
	case "False":
		return false
	default:
		return true
	}
}

func (s *TLSClient) LoadOption(option string) {
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
		case "Mode":
			s.VPNMode = String2Bool(value)
		}
	}
}

func (s *TLSClient) Listen() {
	ln, err := net.Listen("tcp", s.ListenAddress)
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

func (s *TLSClient) handleConn(conn net.Conn) {
	defer conn.Close()
	upConn := conn
	log.Debugf("accepted: %s", conn.RemoteAddr())
	tcpConn, err := net.Dial("tcp", s.BackendAddress)
	if err != nil {
		log.Warningf("TCP connect to %s failed: %s", s.BackendAddress, err)
		return
	}
	defer tcpConn.Close()
	downConn := tls.Client(tcpConn, &tls.Config{ServerName: s.Domain})
	err = downConn.Handshake()
	if err != nil {
		log.Warningf("TLS handshake to %s(%s) failed: %s", s.BackendAddress, s.Domain, err)
		return
	}
	if err := s.Pipe(upConn, downConn); err != nil {
		log.Warningf("pipe failed: %s", err)
	} else {
		log.Debugf("disconnected: %s", upConn.RemoteAddr())
	}
}

func (s *TLSClient) Pipe(a, b net.Conn) error {
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
