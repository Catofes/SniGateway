package main

import (
	"github.com/op/go-logging"
	"errors"
	"net"
	"io"
	"os"
	"regexp"
	"encoding/json"
	"io/ioutil"
	"strconv"
	"flag"
	"time"
)

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

const (
	extensionServerName uint16 = 0
)

var (
	log                   *logging.Logger
	errInvaildClientHello error = errors.New("Invalid TLS ClientHello data")
)

type SNIHandler struct {
	Rules         []map[string]string
	ListenAddress string
	ListenPort    int
}

func (s *SNIHandler) ParseSNI(data []byte) (host string, err error) {
	//TLS Package Type
	if data[0] != 0x16 {
		return "", errInvaildClientHello
	}
	//TLS Version
	versionBytes := data[1:3]
	if versionBytes[0] < 3 || (versionBytes[0] == 3 && versionBytes[1] < 1) {
		return "", errInvaildClientHello
	}

	//Handshake Message
	data = data[5:]
	if len(data) < 42 {
		return "", errInvaildClientHello
	}
	if data[0] != 0x1 {
		return "", errInvaildClientHello
	}
	sessionIdLen := int(data[38])
	if sessionIdLen > 32 || len(data) < 39+sessionIdLen {
		return "", errInvaildClientHello
	}
	data = data[39+sessionIdLen:]
	if len(data) < 2 {
		return "", errInvaildClientHello
	}
	cipherSuiteLen := int(data[0])<<8 | int(data[1])
	if cipherSuiteLen%2 == 1 || len(data) < 2+cipherSuiteLen {
		return "", errInvaildClientHello
	}
	data = data[2+cipherSuiteLen:]
	if len(data) < 1 {
		return "", errInvaildClientHello
	}
	compressionMethodsLen := int(data[0])
	if len(data) < 1+compressionMethodsLen {
		return "", errInvaildClientHello
	}
	data = data[1+compressionMethodsLen:]
	serverName := ""

	if len(data) == 0 {
		// ClientHello is optionally followed by extension data
		return "", nil
	}
	if len(data) < 2 {
		return "", errInvaildClientHello
	}
	extensionsLength := int(data[0])<<8 | int(data[1])
	data = data[2:]
	if extensionsLength != len(data) {
		return "", errInvaildClientHello
	}

	for len(data) != 0 {
		if len(data) < 4 {
			return "", errInvaildClientHello
		}
		extension := uint16(data[0])<<8 | uint16(data[1])
		length := int(data[2])<<8 | int(data[3])
		data = data[4:]
		if len(data) < length {
			return "", errInvaildClientHello
		}

		switch extension {
		case extensionServerName:
			d := data[:length]
			if len(d) < 2 {
				return "", errInvaildClientHello
			}
			namesLen := int(d[0])<<8 | int(d[1])
			d = d[2:]
			if len(d) != namesLen {
				return "", errInvaildClientHello
			}
			for len(d) > 0 {
				if len(d) < 3 {
					return "", errInvaildClientHello
				}
				nameType := d[0]
				nameLen := int(d[1])<<8 | int(d[2])
				d = d[3:]
				if len(d) < nameLen {
					return "", errInvaildClientHello
				}
				if nameType == 0 {
					serverName = string(d[:nameLen])
					break
				}
				d = d[nameLen:]
			}
		}
		data = data[length:]
	}
	return serverName, nil
}

func (s *SNIHandler) GetServer(sni string) string {
	for _, ruleSet := range s.Rules {
		for reg, value := range ruleSet {
			if ok, _ := regexp.MatchString(reg, sni); ok {
				return value
			}
		}
	}
	return ""
}

func (s *SNIHandler) Init(path string) *SNIHandler {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Cannot open config file. %s", err.Error())
	}
	json.Unmarshal(f, s)
	return s
}

func (s *SNIHandler) Pipe(a, b net.Conn) error {
	done := make(chan error, 1)
	cp := func(r, w net.Conn) {
		n, err := io.Copy(w, r)
		log.Debugf("copied %d bytes from %s to %s", n, r.RemoteAddr(), w.RemoteAddr())
		done <- err
	}
	go cp(a, b)
	go cp(b, a)
	err1 := <-done
	log.Debugf("Done1.")
	err2 := <-done
	log.Debugf("Finish.")
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func (s *SNIHandler) Handle(lc net.Conn) {
	log.Debugf("Handle connection %v\n", lc.RemoteAddr())
	defer lc.Close()
	var err error
	b := make([]byte, 1024)
	n, err := lc.Read(b)
	if err != nil {
		log.Debugf("Read error: %v\n", err)
		return
	}
	b = b[:n]

	host, err := s.ParseSNI(b)
	if err != nil {
		log.Warningf("ParseSNI error: %v\n", err)
		return
	}
	log.Debugf("ParseSNI get %v", host)

	if server := s.GetServer(host); server != "" {
		log.Debugf("Dail to %v", server)
		rc, err := net.DialTimeout("tcp", server, 2*time.Second)
		if err != nil {
			log.Warningf("Dial %v error: %v\n", server, err)
			return
		}
		defer rc.Close()
		_, err = rc.Write(b)
		log.Debugf("Write bytes %d to remote.", n)
		if err != nil {
			log.Warningf("Write %v error: %v\n", rc, err)
			return
		}
		err = s.Pipe(lc, rc)
		if err != nil {
			log.Debugf("Pipe return error. %s", err.Error())
		}
	}
}

func (s *SNIHandler) StartListen() {
	listener, err := net.Listen("tcp", s.ListenAddress+":"+strconv.Itoa(s.ListenPort))
	if err != nil {
		log.Warningf("Couldn't start listening. %s", err.Error())
		return
	}
	log.Infof("Started proxy on %s:%d -- listening", s.ListenAddress, s.ListenPort)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Warningf("Accept error. %s", err.Error())
			continue
		}
		go s.Handle(conn)
	}
}

func main() {
	conf := flag.String("conf", "config.json", "Bind Specific IP Address")
	flag.Parse()
	(&SNIHandler{}).Init(*conf).StartListen()
}
