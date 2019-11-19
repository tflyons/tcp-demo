package server

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/tflyons/tcp-demo/types"
)

type Config struct {
	Timeout       time.Duration
	Address       string
	AsyncMessages chan []byte
}

type Server struct {
	config Config
}

func NewServer(config Config) (*Server, error) {
	return &Server{
		config: config,
	}, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go s.handle(conn)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	var lock sync.Mutex

	// handle sending messages asynchronously
	go func() {
		var prefix [10]byte

		// range over channel until chanel is closed
		for msg := range s.config.AsyncMessages {
			binary.BigEndian.PutUint16(prefix[:2], types.MsgV1)
			binary.BigEndian.PutUint64(prefix[2:], uint64(len(msg)))
			log.Println("Sending a message:", string(msg))

			// lock before writing to prevent conflicts with the other writers
			lock.Lock()
			_, err := conn.Write(append(prefix[:], msg...))
			lock.Unlock()
			check(err)
		}
	}()

	var prefix [10]byte
	for {
		// if the connection does not send a message within the timeout period, close the connection
		err := conn.SetReadDeadline(time.Now().Add(s.config.Timeout))
		check(err)

		// read the prefix
		_, err = io.ReadFull(conn, prefix[:])
		check(err)

		// based on the prefix type perform the method for that message
		switch binary.BigEndian.Uint16(prefix[:2]) {
		case types.PingV1:
			log.Println("Got a ping message")

			// write pong response
			binary.BigEndian.PutUint16(prefix[:2], types.PongV1)
			lock.Lock()
			_, err = conn.Write(prefix[:])
			lock.Unlock()
			check(err)

		case types.EchoV1:
			// read the remaining message to echo back
			buf := make([]byte, binary.BigEndian.Uint64(prefix[2:]))
			_, err = io.ReadFull(conn, buf)
			check(err)
			log.Println("Got a message:", string(buf))

			// write echo response
			lock.Lock()
			_, err = conn.Write(append(prefix[:], buf...))
			lock.Unlock()
			check(err)

		case types.FileV1:
			// read the file name
			buf := make([]byte, binary.BigEndian.Uint64(prefix[2:]))
			_, err = io.ReadFull(conn, buf)
			check(err)
			log.Println("Got a file:", string(buf[8:]))

			// create a new file
			file, err := os.Create("server/" + string(buf[8:]))
			check(err)

			// read the file length directly into new file from conn
			fileLen := binary.BigEndian.Uint64(buf[:8])
			_, err = io.CopyN(file, conn, int64(fileLen))
			check(err)
		}
	}
}
