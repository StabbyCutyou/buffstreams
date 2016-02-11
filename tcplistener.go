package buffstreams

import (
	"io"
	"log"
	"net"
)

// ListenCallback is a function type that calling code will need to implement in order
// to receive arrays of bytes from the socket. Each slice of bytes will be stripped of the
// size header, meaning you can directly serialize the raw slice. You would then perform your
// custom logic for interpretting the message, before returning. You can optionally
// return an error, which in turn will be logged if EnableLogging is set to true.
type ListenCallback func([]byte) error

// TCPListener represents the abstraction over a raw TCP socket for reading streaming
// protocolbuffer data without having to write a ton of boilerplate
type TCPListener struct {
	socket          *net.TCPListener
	address         string
	headerByteSize  int
	maxMessageSize  int
	enableLogging   bool
	callback        ListenCallback
	shutdownChannel chan struct{}
}

// TCPListenerConfig representss the information needed to begin listening for
// incoming messages.
type TCPListenerConfig struct {
	// Controls how large the largest Message may be. The server will reject any messages whose clients
	// header size does not match this configuration
	MaxMessageSize int
	// Controls the ability to enable logging errors occuring in the library
	EnableLogging bool
	// The local address to listen for incoming connections on. Typically, you exclude
	// the ip, and just provide port, ie: ":5031"
	Address string
	// The callback to invoke once a full set of message bytes has been received. It
	// is your responsibility to handle parsing the incoming message and handling errors
	// inside the callback
	Callback ListenCallback
}

// ListenTCP creates a TCPListener, and opens it's local connection to
// allow it to begin receiving, once you're ready to. So the connection is open,
// but it is not yet attempting to handle connections.
func ListenTCP(cfg TCPListenerConfig) (*TCPListener, error) {
	maxMessageSize := DefaultMaxMessageSize
	// 0 is the default, and the message must be atleast 1 byte large
	if cfg.MaxMessageSize != 0 {
		maxMessageSize = cfg.MaxMessageSize
	}
	btl := &TCPListener{
		enableLogging:   cfg.EnableLogging,
		maxMessageSize:  maxMessageSize,
		headerByteSize:  messageSizeToBitLength(maxMessageSize),
		callback:        cfg.Callback,
		shutdownChannel: make(chan struct{}, 1),
		address:         cfg.Address,
	}

	if err := btl.openSocket(); err != nil {
		return nil, err
	}

	return btl, nil
}

// Actually blocks the thread it's running on, and begins handling incoming
// requests
func (btl *TCPListener) blockListen() error {
	for {
		// Wait for someone to connect
		conn, err := btl.socket.AcceptTCP()
		if err != nil {
			if btl.enableLogging {
				log.Printf("Error attempting to accept connection: %s", err)
			}
			// Stole this approach from http://zhen.org/blog/graceful-shutdown-of-go-net-dot-listeners/
			// Benefits of a channel for the simplicity of use, but don't have to even check it
			// unless theres an error, so performance impact to incoming conns should be lower
			select {
			case <-btl.shutdownChannel:
				return nil
			default:
				// Nothing, continue to the top of the loop
			}
		} else {
			// Hand this off and immediately listen for more
			go handleListenedConn(btl.address, conn, btl.headerByteSize, btl.maxMessageSize, btl.enableLogging, btl.callback, btl.shutdownChannel)
		}
	}
}

// This is only ever called from either StartListening or StartListeningAsync
// Theres no need to lock, it will only ever be called upon choosing to start
// to listen, by design. Maybe that'll have to change at some point.
func (btl *TCPListener) openSocket() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", btl.address)
	if err != nil {
		return err
	}
	receiveSocket, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	btl.socket = receiveSocket
	return err
}

// StartListening represents a way to start accepting TCP connections, which are
// handled by the Callback provided upon initialization. This method will block
// the current executing thread / go-routine.
func (btl *TCPListener) StartListening() error {
	return btl.blockListen()
}

// Close represents a way to signal to the Listener that it should no longer accept
// incoming connections, and begin to shutdown.
func (btl *TCPListener) Close() {
	close(btl.shutdownChannel)
}

// StartListeningAsync represents a way to start accepting TCP connections, which are
// handled by the Callback provided upon initialization. It does the listening
// in a go-routine, so as not to block.
func (btl *TCPListener) StartListeningAsync() error {
	var err error
	go func() {
		err = btl.blockListen()
	}()
	return err
}

// Handles each incoming connection, run within it's own goroutine. This method will
// loop until the client disconnects or another error occurs and is not handled
func handleListenedConn(address string, conn *net.TCPConn, headerByteSize int, maxMessageSize int, enableLogging bool, cb ListenCallback, sdChan chan struct{}) {
	// If there is any error, close the connection officially and break out of the listen-loop.
	// We don't store these connections anywhere else, and if we can't recover from an error on the socket
	// we want to kill the connection, exit the goroutine, and let the client handle re-connecting if need be.
	// Handle getting the data header

	// We can cheat a tiny bit here, and only allocate this buffer one time. It will be overwritten on each call
	// to read, and we always pass in a slice the size of the total bytes read so far, so there should
	// never be any resultant cross-contamination from earlier runs of the loop.
	headerBuffer := make([]byte, headerByteSize)
	dataBuffer := make([]byte, maxMessageSize)
	// Start an asyncrhonous call that will wait on the shutdown channel, and then close
	// the connection. This will let us respond to the shutdown but also not incur
	// a cost for checking the channel on each run of the loop
	go func(c *net.TCPConn, s chan struct{}) {
		<-s
		c.Close()
	}(conn, sdChan)
	for {
		var headerReadError error
		var totalHeaderBytesRead = 0
		var bytesRead = 0
		// First, read the number of bytes required to determine the message length
		for totalHeaderBytesRead < headerByteSize && headerReadError == nil {
			// While we haven't read enough yet, pass in the slice that represents where we are in the buffer
			bytesRead, headerReadError = readFromConnection(conn, headerBuffer[totalHeaderBytesRead:])
			totalHeaderBytesRead += bytesRead
		}
		if headerReadError != nil {
			if enableLogging {
				if headerReadError != io.EOF {
					// Log the error we got from the call to read
					log.Printf("Error when trying to read from address %s. Tried to read %d, actually read %d. Underlying error: %s", address, headerByteSize, totalHeaderBytesRead, headerReadError)
				} else {
					// Client closed the conn
					log.Printf("Address %s: Client closed connection during header read. Underlying error: %s", address, headerReadError)
				}
			}
			conn.Close()
			return
		}
		// Now turn that buffer of bytes into an integer - represnts size of message body
		msgLength, bytesParsed := byteArrayToUInt32(headerBuffer)
		iMsgLength := int(msgLength)
		// Not sure what the correct way to handle these errors are. For now, bomb out
		if bytesParsed == 0 {
			// "Buffer too small"
			if enableLogging {
				log.Printf("Address %s: 0 Bytes parsed from header. Underlying error: %s", address, headerReadError)
			}
			conn.Close()
			return
		} else if bytesParsed < 0 {
			// "Buffer overflow"
			if enableLogging {
				log.Printf("Address %s: Buffer Less than zero bytes parsed from header. Underlying error: %s", address, headerReadError)
			}
			conn.Close()
			return
		}
		var dataReadError error
		var totalDataBytesRead = 0
		bytesRead = 0
		for totalDataBytesRead < iMsgLength && dataReadError == nil {
			// While we haven't read enough yet, pass in the slice that represents where we are in the buffer
			bytesRead, dataReadError = readFromConnection(conn, dataBuffer[totalDataBytesRead:iMsgLength])
			totalDataBytesRead += bytesRead
		}

		if dataReadError != nil {
			if enableLogging {
				if dataReadError != io.EOF {
					// log the error from the call to read
					log.Printf("Address %s: Failure to read from connection. Was told to read %d by the header, actually read %d. Underlying error: %s", address, msgLength, totalDataBytesRead, dataReadError)
				} else {
					// The client wrote the header but closed the connection
					log.Printf("Address %s: Client closed connection during data read. Underlying error: %s", address, dataReadError)
				}
			}
			conn.Close()
			return
		}
		// If we read bytes, there wasn't an error, or if there was it was only EOF
		// And readbytes + EOF is normal, just as readbytes + no err, next read 0 bytes EOF
		// So... we take action on the actual message data
		if totalDataBytesRead > 0 && (dataReadError == nil || (dataReadError != nil && dataReadError == io.EOF)) {
			err := cb(dataBuffer[:iMsgLength])
			if err != nil && enableLogging {
				log.Printf("Error in Callback: %s", err)
				// TODO if it's a protobuffs error, it means we likely had an issue and can't
				// deserialize data? Should we kill the connection and have the client start over?
				// At this point, there isn't a reliable recovery mechanic for the server
			}
		}
	}
}

// Handles reading from a given connection.
func readFromConnection(reader *net.TCPConn, buffer []byte) (int, error) {
	// This fills the buffer
	bytesLen, err := reader.Read(buffer)
	// Output the content of the bytes to the queue
	if bytesLen == 0 {
		if err != nil && err == io.EOF {
			// "End of individual transmission"
			// We're just done reading from that conn
			return bytesLen, err
		}
	}

	if err != nil {
		//"Underlying network failure?"
		// Not sure what this error would be, but it could exist and i've seen it handled
		// as a general case in other networking code. Following in the footsteps of (greatness|madness)
		return bytesLen, err
	}
	// Read some bytes, return the length
	return bytesLen, nil
}
