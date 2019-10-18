package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
)

//compiler-settable version
var VERSION = "0.0.0-src"

//detailed statistics
var count uint64

type Conf struct {
	Logfile string  `json:"logfile"`
	Proxy   []Proxy `json:"proxy"`
}

type Proxy struct {
	Docker   string `json:"docker"`
	Socket   string `json:"socket"`
	Port     string `json:"port"`
	DockerIp string `json:"docker_ip"`
	//store ip like 172.1.1.1
	DockerAddr string `json:"docker_addr"`
	//store tcp like 172.1.1.1:80
}

func (p *Proxy) StartForward() {
	log.Println("[INFO] Forwarding " + p.Socket + " to " + p.DockerAddr)

	l, err := net.Listen("unix", p.Socket)
	exec.Command("chmod", "777", p.Socket).Run()
	if err != nil {
		log.Fatal(err)
	}
	for {
		uconn, err := l.Accept()
		if err != nil {
			log.Println("[ERROR] For Forwarding " + p.Socket + " to " + p.DockerAddr + " " + err.Error())
			continue
		}
		go forward(p, uconn)
	}
}

func forward(p *Proxy, uconn net.Conn) {
	id := atomic.AddUint64(&count, 1)

	tconn, err := net.Dial("tcp", p.DockerAddr)
	if err != nil {
		log.Printf("[ERROR]Local dial failed: %s "+p.Socket+" to "+p.DockerAddr+"\n", err)
		return
	}
	log.Printf("[%d] connected from "+p.Socket+" to "+p.DockerAddr, id)

	var wg sync.WaitGroup
	go func(uconn net.Conn, tconn net.Conn) {
		wg.Add(1)
		defer wg.Done()
		io.Copy(uconn, tconn)
		uconn.Close()
	}(uconn, tconn)
	go func(uconn net.Conn, tconn net.Conn) {
		wg.Add(1)
		defer wg.Done()
		io.Copy(tconn, uconn)
		tconn.Close()
	}(uconn, tconn)
	wg.Wait()
}

var conf Conf
var configFilePath string
var debug bool

var dockerclient *docker.Client

func init() {
	debug = (runtime.GOOS == "darwin")
	configFilePath = "/etc/socket-proxy/proxy.json"
	jsonString, err := ReadToString(configFilePath)
	if err != nil {
		if !debug {
			log.Fatal(err)
		}
	}
	json.Unmarshal([]byte(jsonString), &conf)

	dockerclient, err = docker.NewClientFromEnv()
	if err != nil {
		if !debug {
			log.Fatal(err)
		}
	}

	//fire on the fly
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {

	for k, v := range conf.Proxy {
		name, err := dockerclient.InspectContainer(v.Docker)
		if err != nil {
			log.Fatal(err)
		}

		v.DockerIp = name.NetworkSettings.IPAddress
		v.DockerAddr = name.NetworkSettings.IPAddress + ":" + v.Port
		conf.Proxy[k] = v
	}

	for _, v := range conf.Proxy {
		go v.StartForward()
	}

	c := make(chan os.Signal)
	//we do not want to clean the signals
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGHUP, syscall.SIGTERM)
	//hang the process and wait for kill
	<-c

	for _, v := range conf.Proxy {
		os.Remove(v.Socket)
	}
	log.Println("Closed listener signal")
	os.Exit(0)
}
