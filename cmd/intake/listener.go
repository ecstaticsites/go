package intake

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

type Listener struct {
	port    string
	msgChan chan []byte
}

func (l Listener) Listen() {

	listen, err := net.Listen("tcp", fmt.Sprintf(":%v", l.port))
	if err != nil {
		log.Printf("[ERROR] Could not listen on port %v: %v", l.port, err)
	}

	defer listen.Close()

	for {

		conn, err := listen.Accept()
		if err != nil {
			log.Printf("[ERROR] Could not accept incoming connection: %v", err)
		}

		log.Printf("[INFO] Got new connection from %v", conn.RemoteAddr())

		// Handle the connection in a new goroutine.
		// The loop then returns to accepting, so that
		// multiple connections may be served concurrently.
		go readFromConnection(conn, l.msgChan)
	}
}

func readFromConnection(conn net.Conn, msgChan chan []byte) {

	reader := bufio.NewReader(conn)

	// note -- we don't call conn.Read() below!
	// that blocks until it gets an EOF, which bunny CDN never sends
	// (it leaves the conn open to use for more access logs)
	// that's why we deliberately scan for braces here
	for {

		// throw away the "headers" of the syslog entry, which we don't care about
		_, err := reader.ReadSlice(byte('{'))
		if err != nil {
			log.Printf("[ERROR] Could not read bytes from TCP connection: %v", err)
			break
		}

		// parse out the JSON blob in the body of the entry, SUPER niavely
		jsonString, err := reader.ReadSlice(byte('}'))
		if err != nil {
			log.Printf("[ERROR] Could not read bytes from TCP connection: %v", err)
			break
		}

		jsonString = append([]byte{'{'}, jsonString...)
		msgChan <- jsonString
	}

	conn.Close()
	log.Printf("[INFO] Closed the TCP connection due to the above")
}
