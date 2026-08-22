package client

import (
	"bufio"
	"fmt"
	"net"
)

type Client struct {
	server_ip_address    string
	client_input_buffer  string
	server_output_buffer *[]byte
}

func RunClient() {
	fmt.Println("start client.")
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println(err)
		return
	}

	reader := bufio.NewReader(conn)

	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("error reading:", err)
		return
	}

	fmt.Print(response)

}
