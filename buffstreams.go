package buffstreams

import ()

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"sync"
)

type BuffManager struct {
	dialedConnections map[string]*net.TCPConn
	listeningSockets  map[string]*net.TCPListener
	// TODO find a way to sanely provide this to a Dialer or a Receiver on a per-connection basis
	MaxMessageSizeBitLength int
	EnableLogging           bool
	// TODO I could control access to the maps better if I centralized how they got accessed - less locking code littered around
	sync.RWMutex
}

type BuffManagerConfig struct {
	MaxMessageSize int
	EnableLogging  bool
}

func New(cfg BuffManagerConfig) *BuffManager {
	bm := &BuffManager{
		dialedConnections: make(map[string]*net.TCPConn),
		listeningSockets:  make(map[string]*net.TCPListener),
		EnableLogging:     cfg.EnableLogging,
	}
	maxMessageSize := 4096
	// 0 is the default, and the message must be atleast 1 byte large
	if cfg.MaxMessageSize != 0 {
		maxMessageSize = cfg.MaxMessageSize
	}
	bm.MaxMessageSizeBitLength = MessageSizeToBitLength(maxMessageSize)
	return bm
}

type ListenCallback func([]byte) error

// Incase someone wants a programmtically correct way to format an address/port
// for use with StartListening or WriteTo
func FormatAddress(address string, port string) string {
	return address + ":" + port
}

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

	go func(address string, maxMessageSizeBitLength int, enableLogging bool, listener net.Listener) {
		for {
			// Wait for someone to connect
			conn, err := listener.Accept()
			if err != nil {
				if enableLogging == true {
					log.Printf("Error attempting to accept connection: %s", err)
				}
			} else {
				// Hand this off and immediately listen for more
				go handleListenedConn(address, conn, bm.MaxMessageSizeBitLength, enableLogging, cb)
			}
		}
	}(address, bm.MaxMessageSizeBitLength, bm.EnableLogging, socket)
}

func handleListenedConn(address string, conn net.Conn, maxMessageSize int, enableLogging bool, cb ListenCallback) {
	// If there is any error, close the connection officially and break out of the listen-loop.
	// We don't store these connections anywhere else, and if we can't recover from an error on the socket
	// we want to kill the connection, exit the goroutine, and let the client handle re-connecting if need be.
	for {
		// Handle getting the data header
		headerByteSize := maxMessageSize
		headerBuffer := make([]byte, headerByteSize)
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
			if enableLogging == true {
				if headerReadError != io.EOF {
					// Log the error we got from the call to read
					log.Print("Error when trying to read from address %s. Tried to read %d, actually read %d. Underlying error: %s", address, headerByteSize, totalHeaderBytesRead, headerReadError)
				} else {
					// Client closed the conn
					log.Printf("Address %s: Client closed connection. Underlying error: %s", address, headerReadError)
				}
			}
			conn.Close()
			return
		}

		// Now turn that buffer of bytes into an integer - represnts size of message body
		msgLength, bytesParsed := binary.Uvarint(headerBuffer)
		// Not sure what the correct way to handle these errors are. For now, bomb out
		if bytesParsed == 0 {
			// "Buffer too small"
			if enableLogging == true {
				log.Printf("Address %s: 0 Bytes parsed from header. Underlying error: %s", address, headerReadError)
			}
			conn.Close()
			return
		} else if bytesParsed < 0 {
			// "Buffer overflow"
			if enableLogging == true {
				log.Printf("Address %s: Buffer Less than zero bytes parsed from header. Underlying error: %s", address, headerReadError)
			}
			conn.Close()
			return
		}
		dataBuffer := make([]byte, msgLength)
		var dataReadError error
		var totalDataBytesRead = 0
		bytesRead = 0
		for totalDataBytesRead < int(msgLength) && dataReadError == nil {
			// While we haven't read enough yet, pass in the slice that represents where we are in the buffer
			bytesRead, dataReadError = readFromConnection(conn, dataBuffer[totalDataBytesRead:])
			totalDataBytesRead += bytesRead
		}

		if dataReadError != nil {
			if enableLogging == true {
				if dataReadError != io.EOF {
					// log the error from the call to read
					log.Printf("Address %s: Failure to read from connection. Was told to read %d by the header, actually read %d. Underlying error: %s", address, msgLength, totalDataBytesRead, dataReadError)
				} else {
					// The client wrote the header but closed the connection
					log.Printf("Address %s: Client closed connection. Underlying error: %s", address, dataReadError)
				}
			}
			conn.Close()
			return
		}

		// If we read bytes, there wasn't an error, or if there was it was only EOF
		// And readbytes + EOF is normal, just as readbytes + no err, next read 0 bytes EOF
		// So... we take action on the actual message data
		if totalDataBytesRead > 0 && (dataReadError == nil || (dataReadError != nil && dataReadError.Error() == "EOF")) {
			err := cb(dataBuffer)
			if err != nil && enableLogging == true {
				log.Printf("Error in Callback: %s", err)
				// TODO if it's a protobuffs error, it means we likely had an issue and can't
				// deserialize data? Should we kill the connection and have the client start over?
				// At this point, there isn't a reliable recovery mechanic for the server
			}
		}
	}
}

func readFromConnection(reader net.Conn, buffer []byte) (int, error) {
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
	} else {
		// Store the connection, it's valid
		bm.dialedConnections[address] = conn
		bm.Unlock()
	}
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

// Write data and dial out if the conn isn't open
// TODO throw a fit if they try to write data > maxSize
func (bm *BuffManager) WriteTo(address string, data []byte, persist bool) (int, error) {
	var conn *net.TCPConn
	var err error
	var ok bool

	// Get the connection if it's cached, or open a new one
	bm.RLock()
	conn, ok = bm.dialedConnections[address]
	bm.RUnlock()
	if ok != true {
		conn, err = bm.dialOut(address)
		if err != nil {
			// Error dialing out, cannot write
			// bail
			return 0, err
		}
	}
	// Calculate how big the message is, using a consistent header size.
	msgLenHeader := UInt16ToByteArray(uint16(len(data)), bm.MaxMessageSizeBitLength)
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

	if writeError != nil || persist == false {
		if writeError != nil && bm.EnableLogging == true {
			log.Printf("Error while writing data to %s. Expected to write %d, actually wrote %d", address, len(toWrite), totalBytesWritten)
			log.Print(writeError)
		}
		writeError = bm.closeDialer(address)
		if writeError != nil {
			// TODO ponder the following:
			// What if some bytes written, then failure, then also the close throws an error
			// []error is a better return type, but not sure if thats a thing you're supposed to do...
			// Possibilities for error not as complicated as i'm thinking?
			if bm.EnableLogging == true {
				// The error will get returned up the stack, no need to log it here?
				log.Print("There was a subsequent error cleaning up the connection to %s")
			}
			return totalBytesWritten, writeError
		}
	}

	// Return the bytes written, any error
	return totalBytesWritten, writeError
}
