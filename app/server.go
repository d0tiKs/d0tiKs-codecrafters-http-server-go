package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

const (

	// PROGRAM PROPERTIES
	PORT = "8080"

	// HTTP HEADER
	EOF_MARKER  = "\r\n"
	EOF_DMARKER = EOF_MARKER + EOF_MARKER

	HTTP_VERSION = "HTTP/1.1"

	// HTTP RESPONSE CODES
	HTTP_OK                = "200"
	HTTP_OK_REPONSE        = HTTP_VERSION + " " + HTTP_OK + " OK" + EOF_DMARKER
	HTTP_NOT_FOUND         = "404"
	HTTP_NOT_FOUND_REPONSE = HTTP_VERSION + " " + HTTP_NOT_FOUND + " Not Found" + EOF_DMARKER

	// HTTP REQUEST METHODS
	HTTP_GET     = "GET"
	HTTP_HEAD    = "HEAD"
	HTTP_POST    = "POST"
	HTTP_PUT     = "PUT"
	HTTP_DELETE  = "DELETE"
	HTTP_CONNECT = "CONNECT"
	HTTP_OPTIONS = "OPTIONS"
	HTTP_TRACE   = "TRACE"

	// LOG LEVELS
	LOG_DEBUG   = "DEBUG"
	LOG_INFO    = "INFO"
	LOG_ERROR   = "ERROR"
	LOG_WARNING = "WARNING"
)

func LogMessage(logLevel string, format string, vargs ...interface{}) (n int, err error) {
	logLevelToken := fmt.Sprintf("[%s] ", logLevel)
	message := fmt.Sprintf(format, vargs...)
	return fmt.Println(logLevelToken + message)
}

func SendResponse(connection net.Conn, response []byte) {
	// send the HTTP 200 OK response
	_, err := connection.Write(response)
	// if there's an error exit
	if err != nil {
		LogMessage(LOG_ERROR, "Error sending response: %s", err.Error())
		os.Exit(1)
	}
}

func BuildLine(data []byte, length int) (int, error) {
	lineLen := 0
	currentChar := byte(0)
	lastChar := byte(0)

	i := 0
	for currentChar != '\n' && lastChar != '\r' {
		i += 1

		if i > length {
			return i, errors.New("reached the end of the string without finding EOF MARKER")
		}

		lastChar = data[i-1]
		currentChar = data[i]
	}

	lineLen = i - 1

	return lineLen, nil
}

func ParseMethod(methodLine string) {
	tokens := strings.Split(methodLine, " ")

	for _, token := range tokens {
		LogMessage(LOG_DEBUG, "token : %s", token)
	}
}

func ParseRequest(request []byte, length int) error {
	request_string := string(request)
	lines := strings.Split(request_string, EOF_MARKER)

	ParseMethod(lines[0])

	for _, line := range lines {
		LogMessage(LOG_DEBUG, line)
	}

	return nil
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	LogMessage(LOG_DEBUG, "Logs from your program will appear here!")

	// open tcp port
	l, err := net.Listen("tcp", "0.0.0.0:"+PORT)
	// if there's an error exit
	if err != nil {
		LogMessage(LOG_ERROR, "Failed to bind to port: "+PORT)
		LogMessage(LOG_ERROR, err.Error())

		os.Exit(1)
	}

	// accept the connection
	connection, err := l.Accept()
	// if there's an error exit
	if err != nil {
		LogMessage(LOG_ERROR, " Error accepting connection: %s", err.Error())
		os.Exit(1)
	}

	request := make([]byte, 256)
	len, err := connection.Read(request)
	// if there's an error exit
	if err != nil {
		LogMessage(LOG_ERROR, " Error reading request: %s", err.Error())
		os.Exit(1)
	}

	ParseRequest(request, len)

	SendResponse(connection, []byte(HTTP_OK_REPONSE))

	connection.Close()

	LogMessage(LOG_DEBUG, "resquest length is '%v'\n%s", len, request)
}
