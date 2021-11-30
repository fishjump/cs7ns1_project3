package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	p2pserver "github.com/fishjump/cs7ns1_project3/p2p-server"
	"github.com/fishjump/cs7ns1_project3/p2p-server/entities"

	"github.com/withmandala/go-log"
)

var (
	dir              string
	externalHostName string
	internalHostName string
	initialIndexHost string
	externalPort     int
	internalPort     int

	clientToken   map[string]string
	deviceDataMap map[string]entities.DeviceData

	internal *p2pserver.P2PServer
	external *p2pserver.P2PServer
	client   *http.Client

	wg sync.WaitGroup

	logger *log.Logger
)

func externalMsgCbk(name string, req *entities.MessageRequest) {
	data := &entities.DeviceData{}
	if err := json.Unmarshal([]byte(req.Data), data); err != nil {
		logger.Error(err)
		return
	}

	deviceDataMap[name] = *data

	str, err := json.Marshal(deviceDataMap)
	if err != nil {
		logger.Error(err)
		return
	}
	ioutil.WriteFile(dir+"/data.json", str, 0644)
}

func internalMsgCbk(name string, req *entities.MessageRequest) {
	data := &entities.SensorData{}
	if err := json.Unmarshal([]byte(req.Data), data); err != nil {
		logger.Error(err)
		return
	}

	deviceDataMap[externalHostName].Data[name] = *data

	str, err := json.Marshal(deviceDataMap)
	if err != nil {
		logger.Error(err)
		return
	}

	ioutil.WriteFile(dir+"/data.json", str, 0644)
}

func runBackground(fn func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn()
	}()
}

func init() {
	logger = log.New(os.Stderr)

	flag.StringVar(&dir, "dir", ".", "directory to save data")
	flag.StringVar(&externalHostName, "host", "rasp-019.scss.tcd.ie", "")
	flag.IntVar(&externalPort, "port", 33000, "")
	flag.StringVar(&internalHostName, "subhost", "127.0.0.1", "")
	flag.IntVar(&internalPort, "subport", 443, "")
	flag.StringVar(&initialIndexHost, "index", "rasp-019.scss.tcd.ie", "")
	flag.Parse()

	clientToken = make(map[string]string)
	deviceDataMap = make(map[string]entities.DeviceData)
	deviceDataMap[externalHostName] = entities.DeviceData{
		Name: externalHostName,
		Data: make(map[string]entities.SensorData),
	}

	internal = p2pserver.NewServer(internalHostName, internalPort,
		dir+"/bundled.key",
		dir+"/bundled.crt",
		dir+"/ca.crt",
		internalMsgCbk)

	external = p2pserver.NewServer(externalHostName, externalPort,
		dir+"/bundled.key",
		dir+"/bundled.crt",
		dir+"/ca.crt",
		externalMsgCbk)
	external.Record.Add(entities.GenToken(initialIndexHost), initialIndexHost)

	certPair, err := tls.LoadX509KeyPair(dir+"/bundled.crt", dir+"/bundled.key")
	if err != nil {
		logger.Error(err)
		return
	}

	caCert, err := ioutil.ReadFile(dir + "/ca.crt")
	if err != nil {
		logger.Error(err)
		return
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{certPair},
			},
		},
	}
}

func main() {
	runBackground(internal.RunTLS)
	runBackground(external.RunTLS)
	runBackground(c.Start)

	wg.Wait()
}
