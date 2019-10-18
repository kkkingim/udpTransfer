package main

import (
	"io"
	"log"
	"net"
	"sync"
)

func main() {
	listener, err := net.Listen("tcp", ":12345")
	if err != nil {
		log.Fatal("端口监听错误")
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("连接错误")
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(uconn net.Conn) {
	log.Println(uconn.RemoteAddr())

	tconn, err := net.Dial("tcp")

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
