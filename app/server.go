package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"slices"
	"strings"
)

const (

	// PROGRAM PROPERTIES
	STDOUT = 1
	STDERR = 2

	ERROR_FD = STDERR

	VERBOSE = true

	DEFAULT_DATA_DIR       = "/dev/null"
	FILE_CHUNK_BUFFER_SIZE = 512

	// CONNECTION PROPERTIES
	PORT         = "4221"
	LISTENING_IP = "0.0.0.0"

	// REQUEST PROPERTIES
	REQUEST_CHUNK_LENGTH = 1024
	REQUEST_MAX_LENGTH   = 1024 * 1024

	// HTTP HEADERS
	EOF_MARKER  = "\r\n"
	EOF_DMARKER = EOF_MARKER + EOF_MARKER

	HTTP_VERSION  = "HTTP/1.1"
	KEY_USERAGENT = "User-Agent"
	KEY_HOST      = "Host"

	// HTTP RESPONSE CODES
	STATUS_OK        = 200
	STATUS_NOT_FOUND = 404

	// HTTP RESPONSE CONTENT TYPES
	CONTENT_TYPE_TEXT_PLAIN   = "text/plain"
	CONTENT_TYPE_OCTET_STREAM = "application/octet-stream"
	CONTENT_TYPE_NO_TYPE      = ""

	// HTTP REQUEST METHODS
	METHOD_GET     = "GET"
	METHOD_HEAD    = "HEAD"
	METHOD_POST    = "POST"
	METHOD_PUT     = "PUT"
	METHOD_DELETE  = "DELETE"
	METHOD_CONNECT = "CONNECT"
	METHOD_OPTIONS = "OPTIONS"
	METHOD_TRACE   = "TRACE"

	// LOG LEVELS
	LOG_DEBUG   = "DEBUG"
	LOG_INFO    = "INFO"
	LOG_ERROR   = "ERROR"
	LOG_WARNING = "WARNING"
)

type processConfig struct {
	dirPath *string
	verbose *bool
}

type response struct {
	message       string
	contentType   string
	content       []byte
	contentLength int
	code          int
}

type request struct {
	res        *response
	method     string
	stringUrl  string
	rawRequest string
	headers    map[string]string
	splitedUrl []string
	lines      []string
	length     uint
}

func Ok(res *response) *response {
	res.code = STATUS_OK

	var strBuilder strings.Builder

	messageCode := fmt.Sprintf("%s %v OK", HTTP_VERSION, STATUS_OK)
	strBuilder.WriteString(messageCode)
	strBuilder.WriteString(EOF_MARKER)

	typeIsSet := res.contentType != CONTENT_TYPE_NO_TYPE
	contentIsNotEmpty := res.contentLength > 0

	if typeIsSet {
		messageContentType := fmt.Sprintf("Content-Type: %s", res.contentType)
		strBuilder.WriteString(messageContentType)
		strBuilder.WriteString(EOF_MARKER)
	}

	if contentIsNotEmpty {
		messageContentLength := fmt.Sprintf("Content-Length: %v", res.contentLength)
		strBuilder.WriteString(messageContentLength)
		strBuilder.WriteString(EOF_DMARKER)

		strBuilder.Write(res.content)
	}

	if !typeIsSet && !contentIsNotEmpty {
		strBuilder.WriteString(EOF_MARKER)
	}

	res.message = strBuilder.String()

	return res
}

func NotFound() *response {
	res := response{
		code:    STATUS_NOT_FOUND,
		message: fmt.Sprintf("%s %v Not Found %s", HTTP_VERSION, STATUS_NOT_FOUND, EOF_DMARKER),
	}

	return &res
}

func LogMessage(logLevel string, format string, vargs ...interface{}) (n int, err error) {
	if !*config.verbose && logLevel == LOG_DEBUG {
		return 0, nil
	}

	logLevelToken := fmt.Sprintf("[%s] ", logLevel)
	message := fmt.Sprintf(format, vargs...)

	outfd := os.Stdout

	if ERROR_FD != STDOUT && logLevel != LOG_INFO {
		outfd = os.Stderr
	}

	return fmt.Fprintln(outfd, logLevelToken+message)
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
	_, err := connection.Write([]byte(res.message))
	if err != nil {
		LogMessage(LOG_ERROR, "Error sending response: %s", err.Error())
		os.Exit(1)
	}
}

func EchoPath(req *request) (*response, error) {
	var strBuilder strings.Builder

	strBuilder.WriteString(strings.SplitN(req.stringUrl, "/", 3)[2])

	LogMessage(LOG_DEBUG, "echo : %s", strBuilder.String())

	res := new(response)
	res.content = []byte(strBuilder.String())
	res.contentType = CONTENT_TYPE_TEXT_PLAIN
	res.contentLength = len(res.content)

	return Ok(res), nil
}

func UserAgentPath(req *request) (*response, error) {
	var strBuilder strings.Builder

	if len(req.headers[KEY_USERAGENT]) == 0 {
		return NotFound(), BuildError(nil, "The header '%s' is not present in the request", KEY_USERAGENT)
	}

	strBuilder.WriteString(req.headers[KEY_USERAGENT])

	LogMessage(LOG_DEBUG, "echo : %s", strBuilder.String())

	res := new(response)
	res.content = []byte(strBuilder.String())
	res.contentType = CONTENT_TYPE_TEXT_PLAIN
	res.contentLength = len(res.content)

	return Ok(res), nil
}

func FilesPath(req *request) (*response, error) {
	rootPath := *(config.dirPath)
	filePath := rootPath + strings.SplitN(req.stringUrl, "/", 3)[2]
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	var dataBuffer []byte
	var cursor int64 = 0

	chunkBuffer := make([]byte, FILE_CHUNK_BUFFER_SIZE)
	dataRead, err := file.Read(chunkBuffer)
	LogMessage(LOG_DEBUG, "dataRead : %v", dataRead)

	for err == nil && dataRead > 0 {
		LogMessage(LOG_DEBUG, "dataRead: %v, cursor: %v", dataRead, cursor)
		cursor += int64(dataRead)
		dataBuffer = append(dataBuffer, chunkBuffer...)
		dataRead, err = file.ReadAt(chunkBuffer, cursor)

	}

	if err != nil && err != io.EOF {
		return nil, BuildError(err, "Error occured while reading file '%s', at index %v", filePath, cursor)
	}

	file.Close()

	res := new(response)
	res.content = dataBuffer
	res.contentType = CONTENT_TYPE_OCTET_STREAM
	res.contentLength = int(cursor)

	return Ok(res), nil
}

func GetPaths(req *request) (func(*request) (*response, error), error) {
	req.splitedUrl = strings.Split(req.stringUrl, "/")

	switch path := req.splitedUrl[1]; path {

	case "echo":
		return EchoPath, nil
	case "user-agent":
		return UserAgentPath, nil
	case "files":
		return FilesPath, nil
	default:
		return nil, BuildError(nil, "Get Path '%s' is not implemented.", req.splitedUrl[1])
	}
}

func GetResource(req *request) (*response, error) {
	uri := req.stringUrl

	if uri == "/" {
		res := new(response)
		res.contentType = CONTENT_TYPE_NO_TYPE
		return Ok(res), nil
	}

	getPath, err := GetPaths(req)
	if err != nil {
		return nil, err
	}

	return getPath(req)
}

func GetMethod(req *request, linesTokens []string) (*request, error) {
	req.method = METHOD_GET
	req.stringUrl = linesTokens[1]

	res, err := GetResource(req)
	if err != nil {

		if res == nil {
			res = NotFound()
		}

		err = BuildError(err, "Ressource not found at %s", req.stringUrl)
	}

	// LogMessage(LOG_DEBUG, "response content :\n%s", res.message)

	req.res = res
	return req, err
}

func ParseMethod(req *request) (*request, error) {
	tokens := strings.Split(req.lines[0], " ")

	switch method := tokens[0]; method {

	case METHOD_GET:
		return GetMethod(req, tokens)

	case METHOD_HEAD:
	case METHOD_POST:
	case METHOD_PUT:
	case METHOD_DELETE:
	case METHOD_CONNECT:
	case METHOD_OPTIONS:
	case METHOD_TRACE:
		return nil, BuildError(nil, "Non implemented HTTP method '%s'", method)
	default:
		return nil, BuildError(nil, "invalid HTTP method '%s'. Check RFC 7231 section 4.3", method)
	}

	return req, nil
}

func ParseHeaders(req *request) (*request, error) {
	req.headers = make(map[string]string)

	if len(req.lines) <= 1 {
		return req, nil
	}

	regex := regexp.MustCompile(`^[a-zA-Z\\-]+: .*`)

	for _, line := range req.lines[1:] {
		if len(line) == 0 || line == EOF_MARKER {
			continue
		}

		matched := regex.MatchString(line)

		if !matched && len(line) > 1 {
			LogMessage(LOG_WARNING, "Unable to parse header '%s'", line)
			continue
		}

		kvp := strings.SplitN(line, ": ", 2)
		req.headers[kvp[0]] = kvp[1]
	}

	return req, nil
}

func ParseRequest(req *request) (*request, error) {
	lines := strings.Split(req.rawRequest, EOF_MARKER)
	req.lines = lines

	req, err := ParseHeaders(req)
	if err != nil {
		LogMessage(LOG_WARNING, "Error while parsing headers : %s", err.Error())
	}

	req, err = ParseMethod(req)
	if err != nil {

		if req == nil {
			return nil, err
		}

		LogMessage(LOG_WARNING, err.Error())
	}

	if *config.verbose {
		for _, line := range req.lines {
			LogMessage(LOG_DEBUG, line)
		}
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

		// LogMessage(LOG_DEBUG, "chunklen : %v", chunkLength)
		// LogMessage(LOG_DEBUG, "chunk : %s", requestChunk)
		// LogMessage(LOG_DEBUG, "chunkslince : %v", requestChunk)

		bytesRead += chunkLength
		doubleEOFReached = slices.Equal(requestChunk[chunkLength-4:chunkLength], []byte(EOF_DMARKER))
		reqStringBuilder.Write(requestChunk[:chunkLength])
	}

	req.rawRequest = reqStringBuilder.String()
	req.length = uint(bytesRead)

	return req, nil
}

func handleConnection(connection net.Conn) {
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
}

func initCLI() {
	config.dirPath = flag.String("directory", DEFAULT_DATA_DIR, "The data directory the http server can access.")
	config.verbose = flag.Bool("verbose", VERBOSE, "Enable debug logs.")

	flag.Parse()

	LogMessage(LOG_DEBUG, "dirpath: %s, verbosity: %v", *config.dirPath, *config.verbose)
}

var config processConfig

func main() {
	initCLI()

	l, err := net.Listen("tcp", LISTENING_IP+":"+PORT)
	if err != nil {
		LogMessage(LOG_ERROR, "Failed to bind to "+LISTENING_IP+":"+PORT)
		LogMessage(LOG_ERROR, err.Error())

		os.Exit(1)
	}

	for {
		connection, err := l.Accept()
		if err != nil {
			LogMessage(LOG_ERROR, " Error accepting connection: %s", err.Error())
			continue
		}
		go handleConnection(connection)
	}
}
