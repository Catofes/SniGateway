package ProxyClient

import (
	"github.com/op/go-logging"
	"os"
	"net"
	"sync"
	"io"
	"strings"
	"regexp"
	"fmt"
	"errors"
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
	backendLeveled.SetLevel(logging.DEBUG, "")
	logging.SetBackend(backendLeveled)
	Log = log
}

type ProxyClient struct {
	ListenAddress  string
	BackendAddress string
	VPNMode        bool
	Host           string
	Port           string
	Id             string
	Key            string
	RemoteHost     string
	RemotePort     string
	RemoteDomain   string
}

func (s *ProxyClient) Init() *ProxyClient {
	SS_LOCAL_HOST := os.Getenv("SS_LOCAL_HOST")
	SS_REMOTE_HOST := os.Getenv("SS_REMOTE_HOST")
	SS_LOCAL_PORT := os.Getenv("SS_LOCAL_PORT")
	SS_REMOTE_PORT := os.Getenv("SS_REMOTE_PORT")
	SS_PLUGIN_OPTIONS := os.Getenv("SS_PLUGIN_OPTIONS")
	s.BackendAddress = SS_REMOTE_HOST + ":" + SS_REMOTE_PORT
	s.ListenAddress = SS_LOCAL_HOST + ":" + SS_LOCAL_PORT
	ip_reg := `(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)\.(25[0-5]|2[0-4]\d|[0-1]\d{2}|[1-9]?\d)`
	if ok, _ := regexp.MatchString(ip_reg, SS_REMOTE_PORT); !ok {
		s.RemoteHost = SS_REMOTE_HOST
	}
	s.VPNMode = true
	s.LoadOption(SS_PLUGIN_OPTIONS)
	s.RemoteDomain = s.RemoteHost + ":" + s.RemotePort
	return s
}

func (s *ProxyClient) LoadOption(option string) {
	data := strings.Split(option, ";")
	for _, v := range data {
		d := strings.Split(v, "=")
		if len(d) != 2 {
			continue
		}
		key := d[0]
		value := d[1]
		switch key {
		case "host":
			s.Host = value
		case "port":
			s.Port = value
		case "id":
			s.Id = value
		case "key":
			s.Key = value
		case "remotehost":
			s.RemoteHost = value
		case "remoteport":
			s.RemotePort = value
		}
	}
}

func (s *ProxyClient) Listen() {
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

func (s *ProxyClient) handshake(conn net.Conn) error {
	log.Debug("start handshake.")
	header := fmt.Sprintf("CONNECT %s HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Proxy-Connection: keep-alive\r\n"+
		//"User-Agent: Mozilla/5.0 (Linux; Android 8.0; ONEPLUS A5010 Build/OPR6.170623.013; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/57.0.2987.132 MQQBrowser/6.2 TBS/038316 Mobile Safari/537.36\r\n"+
		//"Q-UA2: QV=3&PL=ADR&PR=QB&PP=com.tencent.mtt&PPVN=8.3.0.4021&TBSVC=1&CO=BK&COVC=038316&PB=GE&VE=GA&DE=PHONE&CHID=72264&LCID=10291&MO= ONEPLUSA5010 &RL=1080*2034&OS=8.0.0&API=26\r\n"+
		"Q-GUID: %s\r\n"+
		"Q-Token: %s\r\n\r\n", s.RemoteDomain, s.RemoteDomain, s.Id, s.Key)
	log.Debug(header)
	_, err := conn.Write([]byte(header))
	log.Debug("Write to remote.")
	if err != nil {
		return err
	}
	buffer := make([]byte, 1000)
	log.Debug("Read from remote.")
	n, err := conn.Read(buffer)
	if err != nil {
		return err
	}
	buffer = buffer[:n]
	if strings.Contains(string(buffer), "Connection established") {
		log.Debug("handshake finished.")
		return nil
	}
	log.Debugf("Handshake failed: %s.", string(buffer))
	return errors.New("handshake failed")
}

func (s *ProxyClient) handleConn(conn net.Conn) {
	defer conn.Close()
	localConn := conn
	log.Debugf("accepted: %s", localConn.RemoteAddr())
	remoteConn, err := net.Dial("tcp", s.Host+":"+s.Port)
	if err != nil {
		log.Warningf("TCP connect to %s failed: %s", s.Host+":"+s.Port, err)
		return
	}
	defer remoteConn.Close()

	err = s.handshake(remoteConn)
	if err != nil {
		return
	}

	if err := s.Pipe(localConn, remoteConn); err != nil {
		log.Warningf("pipe failed: %s", err)
	} else {
		log.Debugf("disconnected: %s", localConn.RemoteAddr())
	}
}

func (s *ProxyClient) Pipe(a, b net.Conn) error {
	done := make(chan error, 1)
	download := func(a, b net.Conn) {
		n, err := io.Copy(a, b)
		log.Debugf("copied %d bytes from %s to %s", n, b.RemoteAddr(), a.RemoteAddr())
		b.(*net.TCPConn).CloseRead()
		a.(*net.TCPConn).CloseWrite()
		done <- err
	}
	upload := func(a, b net.Conn) {
		n, err := io.Copy(b, a)
		log.Debugf("copied %d bytes from %s to %s", n, a.RemoteAddr(), b.RemoteAddr())
		a.(*net.TCPConn).CloseRead()
		b.(*net.TCPConn).CloseWrite()
		done <- err
	}
	go download(a, b)
	go upload(a, b)
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
