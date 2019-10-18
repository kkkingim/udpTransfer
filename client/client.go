package main

import (
	"log"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:12345")
	if err != nil {
		log.Fatal("连接错误")
	}
	log.Println("连接成功")

	conn.Close()

}
