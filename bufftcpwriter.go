package buffstreams

import (
	"log"
	"net"
)

// BuffTCPWriter represents the abstraction over a raw TCP socket for writing streaming
// protocolbuffer data without having to write a ton of boilerplate
type BuffTCPWriter struct {
	socket         *net.TCPConn
	address        string
	headerByteSize int
	maxMessageSize int
	enableLogging  bool
}

// BuffTCPWriterConfig represents
type BuffTCPWriterConfig struct {
	// Controls how large the largest Message may be. The server will reject any messages whose clients
	// header size does not match this configuration
	MaxMessageSize int
	// Controls the ability to enable logging errors occuring in the library
	EnableLogging bool
	// Address is the address to connect to for writing streaming messages
	Address string
}

// Open represents
func (btw *BuffTCPWriter) open() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", btw.address)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	}
	btw.socket = conn
	return err
}

// Close represents
func (btw *BuffTCPWriter) Close() error {
	return btw.socket.Close()
}

// DialBuffTCP represents
func DialBuffTCP(cfg BuffTCPWriterConfig) (*BuffTCPWriter, error) {
	maxMessageSize := DefaultMaxMessageSize
	// 0 is the default, and the message must be atleast 1 byte large
	if cfg.MaxMessageSize != 0 {
		maxMessageSize = cfg.MaxMessageSize
	}

	btw := &BuffTCPWriter{
		enableLogging:  cfg.EnableLogging,
		maxMessageSize: maxMessageSize,
		headerByteSize: messageSizeToBitLength(maxMessageSize),
		address:        cfg.Address,
	}
	if err := btw.open(); err != nil {
		return nil, err
	}
	return btw, nil
}

// Write allows you to send a stream of bytes as messages. Each array of bytes
// you pass in will be pre-pended with it's size. If the connection isn't open
// you will receive an error. If not all bytes can be written, Write will keep
// trying until the full message is delivered, or the connection is broken.
func (btw *BuffTCPWriter) Write(data []byte) (int, error) {
	// Calculate how big the message is, using a consistent header size.
	msgLenHeader := uInt16ToByteArray(uint16(len(data)), btw.headerByteSize)
	// Append the size to the message, so now it has a header
	toWrite := append(msgLenHeader, data...)

	toWriteLen := len(toWrite)

	// Three conditions could have occured:
	// 1. There was an error
	// 2. Not all bytes were written
	// 3. Both 1 and 2

	// If there was an error, that should take handling precedence. If the connection
	// was closed, or is otherwise in a bad state, we have to abort and re-open the connection
	// to try again, as we can't realistically finish the write. We have to retry it, or return
	// and error to the user?

	// TODO configurable message retries

	// If there was not an error, and we simply didn't finish the write, we should enter
	// a write-until-complete loop, where we continue to write the data until the server accepts
	// all of it.

	// If both issues occurred, we'll need to find a way to determine if the error
	// is recoverable (is the connection in a bad state) or not

	var writeError error
	var totalBytesWritten = 0
	var bytesWritten = 0
	// First, read the number of bytes required to determine the message length
	for totalBytesWritten < toWriteLen && writeError == nil {
		// While we haven't read enough yet
		// If there are remainder bytes, adjust the contents of toWrite
		// totalBytesWritten will be the index of the nextByte waiting to be read
		bytesWritten, writeError = btw.socket.Write(toWrite[totalBytesWritten:])
		totalBytesWritten += bytesWritten
	}

	if writeError != nil {
		if btw.enableLogging {
			log.Printf("Error while writing data to %s. Expected to write %d, actually wrote %d. Underlying error: %s", btw.address, len(toWrite), totalBytesWritten, writeError)
		}
		writeError = btw.Close()
		if writeError != nil {
			// TODO ponder the following:
			// What if some bytes written, then failure, then also the close throws an error
			// []error is a better return type, but not sure if thats a thing you're supposed to do...
			// Possibilities for error not as complicated as i'm thinking?
			if btw.enableLogging {
				// The error will get returned up the stack, no need to log it here?
				log.Printf("There was a subsequent error cleaning up the connection to %s", btw.address)
			}
			return totalBytesWritten, writeError
		}
	}

	// Return the bytes written, any error
	return totalBytesWritten, writeError
}
