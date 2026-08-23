package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func RunServer() {
	fmt.Println("Running server ...")
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	log.Println("Listening on :8080")

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("Received: %q\n", line)
		fmt.Fprintf(conn, "Echo: %s\n", line)
	}

	if err := scanner.Err(); err != nil {
		log.Println("read:", err)
	}
}

func main() {
	RunServer()
}
