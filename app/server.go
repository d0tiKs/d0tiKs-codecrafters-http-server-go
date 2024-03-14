package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	port := "4221"

	// open tcp port
	l, err := net.Listen("tcp", "0.0.0.0:"+port)
	// if there's an error exit
	if err != nil {
		fmt.Println("Failed to bind to port " + port)
		os.Exit(1)
	}

	// accept the connection
	connection, err := l.Accept()
	// if there's an error exit
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	// send the HTTP 200 OK resposne
	_, err = connection.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	// if there's an error exit
	if err != nil {
		fmt.Println("Error sending response: ", err.Error())
		os.Exit(1)
	}
}
