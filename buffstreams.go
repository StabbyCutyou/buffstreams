// Package buffstreams provides a simple interface for creating and storing
// sockets that have a pre-defined set of behaviors, making them simple to use
// for consuming streaming protocol buffer messages over TCP
package buffstreams

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"sync"
)

// BuffManager represents the object used to govern interactions between tcp endpoints.
// You can use it to read from and write to streaming or non-streaming TCP connections
// and have it handle packaging data with a header describing the size of the data payload.
// This is to make it easy to work with wire formats like ProtocolBuffers, which require
// a custom-delimeter situation to be sent in a streaming fashion.
type BuffManager struct {
	dialedConnections map[string]*net.TCPConn
	listeningSockets  map[string]*net.TCPListener
	// TODO find a way to sanely provide this to a Dialer or a Receiver on a per-connection basis
	// TODO I could control access to the maps better if I centralized how they got accessed - less locking code littered around
	headerByteSize int
	maxMessageSize int
	enableLogging  bool
	sync.RWMutex
}

// BuffManagerConfig represents a set of options that the person building systems ontop of
type BuffManagerConfig struct {
	// Controls how large the largest Message may be. The server will reject any messages whose clients
	// header size does not match this configuration
	MaxMessageSize int
	// Controls the ability to enable logging errors occuring in the library
	EnableLogging bool
}

// New creates a new *BuffManager based on the provided BuffManagerConfig
func New(cfg BuffManagerConfig) *BuffManager {
	bm := &BuffManager{
		dialedConnections: make(map[string]*net.TCPConn),
		listeningSockets:  make(map[string]*net.TCPListener),
		enableLogging:     cfg.EnableLogging,
	}
	maxMessageSize := 4096
	// 0 is the default, and the message must be atleast 1 byte large
	if cfg.MaxMessageSize != 0 {
		maxMessageSize = cfg.MaxMessageSize
	}
	bm.maxMessageSize = maxMessageSize
	bm.headerByteSize = messageSizeToBitLength(maxMessageSize)
	return bm
}

// ListenCallback is a function type that calling code will need to implement in order
// to receive arrays of bytes from the socket. Each slice of bytes will be stripped of the
// size header, meaning you can directly serialize the raw slice. You would then perform your
// custom logic for interpretting the message, before returning. You can optionally
// return an error, which in turn will be logged if EnableLogging is set to true.
type ListenCallback func([]byte) error

// FormatAddress is to cover the event that you want/need a programmtically correct way
// to format an address/port to use with StartListening or WriteTo
func FormatAddress(address string, port string) string {
	return address + ":" + port
}

// StartListening is an asyncrhonous, non-blocking method. It begins listening on the given
// port, and fire off a goroutine for every client connection it receives. That goroutine will
// read the fixed header, then the message payload, and then invoke the povided ListenCallbacl.
// In the event of an transport error, it will disconnect the client. It is the clients responsibility
// to re-connect if needed.
func (bm *BuffManager) StartListening(port string, cb ListenCallback) error {
	address := FormatAddress("", port)
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return err
	}
	receiveSocket, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	bm.startListening(address, receiveSocket, cb)
	return nil
}

func (bm *BuffManager) startListening(address string, socket *net.TCPListener, cb ListenCallback) {
	bm.Lock()
	bm.listeningSockets[address] = socket
	bm.Unlock()

	go func(address string, headerByteSize int, maxMessageSize int, enableLogging bool, listener *net.TCPListener) {
		for {
			// Wait for someone to connect
			conn, err := listener.AcceptTCP()
			if err != nil {
				if enableLogging {
					log.Printf("Error attempting to accept connection: %s", err)
				}
			} else {
				// Hand this off and immediately listen for more
				go handleListenedConn(address, conn, headerByteSize, maxMessageSize, enableLogging, cb)
			}
		}
	}(address, bm.headerByteSize, bm.maxMessageSize, bm.enableLogging, socket)
}

func handleListenedConn(address string, conn *net.TCPConn, headerByteSize int, maxMessageSize int, enableLogging bool, cb ListenCallback) {
	// If there is any error, close the connection officially and break out of the listen-loop.
	// We don't store these connections anywhere else, and if we can't recover from an error on the socket
	// we want to kill the connection, exit the goroutine, and let the client handle re-connecting if need be.
	// Handle getting the data header

	// We can cheat a tiny bit here, and only allocate this buffer one time. It will be overwritten on each call
	// to read, and we always pass in a slice the size of the total bytes read so far, so there should
	// never be any resultant cross-contamination from earlier runs of the loop.
	headerBuffer := make([]byte, headerByteSize)
	dataBuffer := make([]byte, maxMessageSize)
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
		msgLength, bytesParsed := binary.Uvarint(headerBuffer)
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
		if totalDataBytesRead == 0 && (dataReadError == nil || (dataReadError != nil && dataReadError == io.EOF)) {
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

func (bm *BuffManager) dialOut(address string) (*net.TCPConn, error) {
	// We want to hold this lock for both the lookup as well as for setting it in the map
	// Skipping defer for now, need to profile it's performance
	bm.Lock()
	if _, ok := bm.dialedConnections[address]; ok {
		bm.Unlock()
		return nil, errors.New("You have a connection to this ip and port open already")
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		bm.Unlock()
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		bm.Unlock()
		return nil, err
	}

	// Store the connection, it's valid
	bm.dialedConnections[address] = conn
	bm.Unlock()
	return conn, nil
}

func (bm *BuffManager) closeDialer(address string) error {
	bm.Lock()
	// We want to hold this lock for both the lookup as well as the delete
	// Either way, we can't release until we're ready to return.
	//Skipping defer for now, need to profile it's performance
	if conn, ok := bm.dialedConnections[address]; ok {
		err := conn.Close()
		// Grab lock to delete from the map
		delete(bm.dialedConnections, address)
		// Release immediately
		bm.Unlock()
		return err
	}
	bm.Unlock()
	return nil
}

// WriteTo allows you to dial to a remote or local TCP endpoint, and send either 1 or a stream
// of bytes as messages. Each array of bytes you pass in will be pre-pended with it's size
// within the size of the pre-defined maximum message size. If the connection isn't open yet,
// WriteTo will open it, and cache it. If for anyreason the connection breaks, it will be disposed
// and upon the next write, a new one will dial out.
func (bm *BuffManager) WriteTo(address string, data []byte, persist bool) (int, error) {
	var conn *net.TCPConn
	var err error
	var ok bool

	// Get the connection if it's cached, or open a new one
	bm.RLock()
	conn, ok = bm.dialedConnections[address]
	bm.RUnlock()
	if !ok {
		conn, err = bm.dialOut(address)
		if err != nil {
			// Error dialing out, cannot write
			// bail
			return 0, err
		}
	}
	// Calculate how big the message is, using a consistent header size.
	msgLenHeader := uInt16ToByteArray(uint16(len(data)), bm.headerByteSize)
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
		bytesWritten, writeError = conn.Write(toWrite[totalBytesWritten:])
		totalBytesWritten += bytesWritten
	}

	if writeError != nil || !persist {
		if writeError != nil && bm.enableLogging {
			log.Printf("Error while writing data to %s. Expected to write %d, actually wrote %d. Underlying error: %s", address, len(toWrite), totalBytesWritten, writeError)
		}
		writeError = bm.closeDialer(address)
		if writeError != nil {
			// TODO ponder the following:
			// What if some bytes written, then failure, then also the close throws an error
			// []error is a better return type, but not sure if thats a thing you're supposed to do...
			// Possibilities for error not as complicated as i'm thinking?
			if bm.enableLogging {
				// The error will get returned up the stack, no need to log it here?
				log.Print("There was a subsequent error cleaning up the connection to %s")
			}
			return totalBytesWritten, writeError
		}
	}

	// Return the bytes written, any error
	return totalBytesWritten, writeError
}
