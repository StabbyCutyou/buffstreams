package buffstreams

import (
	"errors"
	"io"
	"net"
	"sync"
)

var (
	// ErrZeroBytesReadHeader is thrown when the value parsed from the header is not valid
	ErrZeroBytesReadHeader = errors.New("0 Bytes parsed from header. Connection Closed")
	// ErrLessThanZeroBytesReadHeader is thrown when the value parsed from the header caused some kind of underrun
	ErrLessThanZeroBytesReadHeader = errors.New("Less than zero bytes parsed from header. Connection Closed")
)

// TCPConn is an abstraction over the normal net.TCPConn, but optimized for wtiting
// data encoded in a length+data format, like you would treat networked protocol
// buffer messages
type TCPConn struct {
	// General
	socket         *net.TCPConn
	address        string
	headerByteSize int
	maxMessageSize int

	// For processing incoming data
	incomingHeaderBuffer []byte

	// For processing outgoing data
	writeLock          sync.Mutex
	outgoingDataBuffer []byte
}

// TCPConnConfig representss the information needed to begin listening for
// writing messages.
type TCPConnConfig struct {
	// Controls how large the largest Message may be. The server will reject any messages whose clients
	// header size does not match this configuration.
	MaxMessageSize int
	// Address is the address to connect to for writing streaming messages.
	Address string

	//Delimiter if any
	DelimiterPresent bool
	Delimiter byte
}

func newTCPConn(cfg *TCPConnConfig) (*TCPConn, error) {
	maxMessageSize := DefaultMaxMessageSize
	// 0 is the default, and the message must be atleast 1 byte large
	if cfg.MaxMessageSize != 0 {
		maxMessageSize = cfg.MaxMessageSize
	}

	headerByteSize := messageSizeToBitLength(maxMessageSize)

	return &TCPConn{
		maxMessageSize:       maxMessageSize,
		headerByteSize:       headerByteSize,
		address:              cfg.Address,
		incomingHeaderBuffer: make([]byte, headerByteSize),
		writeLock:            sync.Mutex{},
		outgoingDataBuffer:   make([]byte, maxMessageSize),
	}, nil
}

// DialTCP creates a TCPWriter, and dials a connection to the remote
// endpoint. It does not begin writing anything until you begin to do so.
func DialTCP(cfg *TCPConnConfig) (*TCPConn, error) {
	c, err := newTCPConn(cfg)
	if err != nil {
		return nil, err
	}
	if err := c.open(); err != nil {
		return nil, err
	}
	return c, nil
}

// open will dial a connection to the remote endpoint.
func (c *TCPConn) open() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", c.address)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	}
	c.socket = conn
	return err
}

// Reopen allows you to close and re-establish a connection to the existing Address
// without needing to create a whole new TCPWriter object.
func (c *TCPConn) Reopen() error {
	if err := c.Close(); err != nil {
		return err
	}

	if err := c.open(); err != nil {
		return err
	}

	return nil
}

// Close will immediately call close on the connection to the remote endpoint. Per
// the golang source code for the netFD object, this call uses a special mutex to
// control access to the underlying pool of readers/writers. This call should be
// threadsafe, so that any other threads writing will finish, or be blocked, when
// this is invoked.
func (c *TCPConn) Close() error {
	return c.socket.Close()
}

// Write allows you to send a stream of bytes as messages. Each array of bytes
// you pass in will be pre-pended with it's size. If the connection isn't open
// you will receive an error. If not all bytes can be written, Write will keep
// trying until the full message is delivered, or the connection is broken.
func (c *TCPConn) Write(data []byte) (int, error) {
	// Calculate how big the message is, using a consistent header size.
	// Append the size to the message, so now it has a header
	c.outgoingDataBuffer = append(intToByteArray(int64(len(data)), c.headerByteSize), data...)

	toWriteLen := len(c.outgoingDataBuffer)

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
	// is recoverable (is the connection in a bad state) or not.

	var writeError error
	var totalBytesWritten = 0
	var bytesWritten = 0
	// First, read the number of bytes required to determine the message length
	for totalBytesWritten < toWriteLen && writeError == nil {
		// While we haven't read enough yet
		// If there are remainder bytes, adjust the contents of toWrite
		// totalBytesWritten will be the index of the nextByte waiting to be read
		bytesWritten, writeError = c.socket.Write(c.outgoingDataBuffer[totalBytesWritten:])
		totalBytesWritten += bytesWritten
	}
	if writeError != nil {
		c.Close()
	}

	// Return the bytes written, any error
	return totalBytesWritten, writeError
}

func (c *TCPConn) lowLevelRead(buffer []byte) (int, error) {
	var totalBytesRead = 0
	var err error
	var bytesRead = 0
	var toRead = len(buffer)
	// This fills the buffer
	bytesRead, err = c.socket.Read(buffer)
	totalBytesRead += bytesRead
	for totalBytesRead < toRead && err == nil {
		bytesRead, err = c.socket.Read(buffer[totalBytesRead:])
		totalBytesRead += bytesRead
	}

	// Output the content of the bytes to the queue
	if totalBytesRead == 0 && err != nil && err == io.EOF {
		// "End of individual transmission"
		// We're just done reading from that conn
		return totalBytesRead, err
	} else if err != nil {
		//"Underlying network failure?"
		// Not sure what this error would be, but it could exist and i've seen it handled
		// as a general case in other networking code. Following in the footsteps of (greatness|madness)
		return totalBytesRead, err
	}
	// Read some bytes, return the length

	return totalBytesRead, nil
}

func (c *TCPConn) Read(b []byte) (int, error) {
	// Read the header
	hLength, err := c.lowLevelRead(c.incomingHeaderBuffer)
	if err != nil {
		return hLength, err
	}
	// Decode it
	msgLength, bytesParsed := byteArrayToUInt32(c.incomingHeaderBuffer)
	if bytesParsed == 0 {
		// "Buffer too small"
		c.Close()
		return hLength, ErrZeroBytesReadHeader
	} else if bytesParsed < 0 {
		// "Buffer overflow"
		c.Close()
		return hLength, ErrLessThanZeroBytesReadHeader
	}

	// Using the header, read the remaining body
	bLength, err := c.lowLevelRead(b[:msgLength])
	if err != nil {
		c.Close()
	}
	return bLength, err
}
