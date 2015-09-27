// Package buffstreams provides a simple interface for creating and storing
// sockets that have a pre-defined set of behaviors, making them simple to use
// for consuming streaming protocol buffer messages over TCP
package buffstreams

// DefaultMaxMessageSize is the value that is used if a ManagerConfig indicates
// a MaxMessageSize of 0
const DefaultMaxMessageSize int = 4096

// FormatAddress is to cover the event that you want/need a programmtically correct way
// to format an address/port to use with StartListening or WriteTo
func FormatAddress(address string, port string) string {
	return address + ":" + port
}
