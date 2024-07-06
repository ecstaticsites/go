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

	for {

		garbage, err := reader.ReadSlice(byte('{'))
		if err != nil {
			log.Printf("could not read: %v", err)
			break
		}

		log.Printf("garbage: %s", garbage)

		goodstuff, err := reader.ReadSlice(byte('}'))
		if err != nil {
			log.Printf("could not read: %v", err)
			break
		}

		goodstuff = append([]byte{'{'}, goodstuff...)

		msgChan <- goodstuff
	}

	conn.Close()
	log.Printf("closed the conn")
}
