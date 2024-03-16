package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
	"strings"
)

const (

	// CONNECTION PROPERTIES
	PORT = "4221"

	// REQUEST PROPERTIES
	REQUEST_CHUNK_LENGTH = 1024
	REQUEST_MAX_LENGTH   = 1024 * 1024

	// HTTP HEADER
	EOF_MARKER  = "\r\n"
	EOF_DMARKER = EOF_MARKER + EOF_MARKER

	HTTP_VERSION = "HTTP/1.1"

	// HTTP RESPONSE CODES
	HTTP_OK        = 200
	HTTP_NOT_FOUND = 404

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

type response struct {
	message string
	code    int
}

type request struct {
	res        *response
	method     string
	stringUrl  string
	rawRequest string
	lines      []string
	length     uint
}

func Ok() *response {
	res := response{
		code:    HTTP_OK,
		message: fmt.Sprintf("%s %v OK %s", HTTP_VERSION, HTTP_OK, EOF_DMARKER),
	}

	return &res
}

func NotFound() *response {
	res := response{
		code:    HTTP_NOT_FOUND,
		message: fmt.Sprintf("%s %v Not Found %s", HTTP_VERSION, HTTP_NOT_FOUND, EOF_DMARKER),
	}

	return &res
}

func LogMessage(logLevel string, format string, vargs ...interface{}) (n int, err error) {
	logLevelToken := fmt.Sprintf("[%s] ", logLevel)
	message := fmt.Sprintf(format, vargs...)
	return fmt.Println(logLevelToken + message)
}

func BuildError(err error, format string, vars ...interface{}) error {
	errorMessage := fmt.Sprintf(format, vars...)

	if err == nil {
		return errors.New(errorMessage)
	}

	embeddedError := fmt.Sprintf("\nSee error bellow:\n%s", err.Error())
	return errors.New(errorMessage + embeddedError)
}

func SendResponse(connection net.Conn, res *response) {
	// send the HTTP 200 OK response
	_, err := connection.Write([]byte(res.message))
	// if there's an error exit
	if err != nil {
		LogMessage(LOG_ERROR, "Error sending response: %s", err.Error())
		os.Exit(1)
	}
}

func GetRessource(uri string) (*response, error) {
	if uri == "/" {
		return Ok(), nil
	}

	return nil, BuildError(nil, "Not implemented!")
}

func GetMethod(req *request, linesTokens []string) (*request, error) {
	req.method = HTTP_GET
	req.stringUrl = linesTokens[1]

	// TODO: search for the path to request try to get
	res, err := GetRessource(req.stringUrl)
	if err != nil {

		if res == nil {
			res = NotFound()
		}

		err = BuildError(err, "Ressource not found at %s", req.stringUrl)
	}

	req.res = res
	return req, err
}

func ParseMethod(req *request) (*request, error) {
	tokens := strings.Split(req.lines[0], " ")

	switch method := tokens[0]; method {

	case HTTP_GET:
		return GetMethod(req, tokens)

	case HTTP_HEAD:
	case HTTP_POST:
	case HTTP_PUT:
	case HTTP_DELETE:
	case HTTP_CONNECT:
	case HTTP_OPTIONS:
	case HTTP_TRACE:
		return nil, BuildError(nil, "Non implemented HTTP method '%s'", method)
	default:
		return nil, BuildError(nil, "invalid HTTP method '%s'. Check RFC 7231 section 4.3", method)
	}

	return req, nil
}

func ParseRequest(req *request) (*request, error) {
	lines := strings.Split(req.rawRequest, EOF_MARKER)
	req.lines = lines

	req, err := ParseMethod(req)
	if err != nil {

		if req == nil {
			return nil, err
		}

		LogMessage(LOG_WARNING, err.Error())
	}

	for _, line := range req.lines {
		LogMessage(LOG_DEBUG, line)
	}

	return req, nil
}

func ReadRequest(connection net.Conn) (*request, error) {
	var reqStringBuilder strings.Builder

	bytesRead := 0
	doubleEOFReached := false
	req := &request{}

	for bytesRead <= REQUEST_MAX_LENGTH &&
		!doubleEOFReached {
		requestChunk := make([]byte, REQUEST_CHUNK_LENGTH)
		chunkLength, err := connection.Read(requestChunk)
		if err != nil {
			return nil, BuildError(err, "Error reading request at len %v", bytesRead)
		}

		LogMessage(LOG_DEBUG, "chunklen : %v", chunkLength)
		LogMessage(LOG_DEBUG, "chunk : %s", requestChunk)

		bytesRead += chunkLength
		doubleEOFReached = slices.Equal(requestChunk[chunkLength-4:chunkLength], []byte(EOF_DMARKER))
		reqStringBuilder.Write(requestChunk)
	}

	req.rawRequest = reqStringBuilder.String()
	req.length = uint(bytesRead)

	return req, nil
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

	req, err := ReadRequest(connection)
	if err != nil {
		LogMessage(LOG_ERROR, " Error reading request: %s", err.Error())
		os.Exit(1)
	}

	req, err = ParseRequest(req)
	if err != nil {
		LogMessage(LOG_ERROR, "Parsing request : %s", err.Error())
		os.Exit(1)
	}

	SendResponse(connection, req.res)

	connection.Close()

	LogMessage(LOG_DEBUG, "resquest length is '%v'\n%s", req.length, req.rawRequest)
}
