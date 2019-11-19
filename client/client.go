package client

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"os"

	"github.com/tflyons/tcp-demo/types"
)

type Client struct {
	conn net.Conn
}

func NewClient(address string) (*Client, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn: conn,
	}, nil
}

func (c *Client) SendPing() error {
	var prefix [10]byte
	binary.BigEndian.PutUint16(prefix[:2], types.PingV1)

	n, err := c.conn.Write(prefix[:])
	if err != nil {
		return err
	}
	if n != len(prefix) {
		return errors.New("invalid number of bytes written")
	}
	return nil
}

func (c *Client) Echo(message string) error {
	var prefix [10]byte
	binary.BigEndian.PutUint16(prefix[:2], types.EchoV1)
	binary.BigEndian.PutUint64(prefix[2:], uint64(len(message)))

	n, err := c.conn.Write(append(prefix[:], []byte(message)...))
	if err != nil {
		return err
	}
	if n != len(prefix)+len(message) {
		return errors.New("invalid number of bytes written")
	}
	return nil
}

func (c *Client) UploadFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}

	var prefix [10]byte
	var fileLen [8]byte
	binary.BigEndian.PutUint16(prefix[:2], types.FileV1)
	binary.BigEndian.PutUint64(prefix[2:], 8+uint64(len(info.Name())))
	binary.BigEndian.PutUint64(fileLen[:], uint64(info.Size()))

	n, err := c.conn.Write(append(append(prefix[:], fileLen[:]...), []byte(info.Name())...))
	if err != nil {
		return err
	}
	if n != len(prefix)+len(info.Name())+len(fileLen) {
		return errors.New("invalid number of bytes written")
	}
	_, err = io.Copy(c.conn, file)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) ReadResponse() error {
	var prefix [10]byte
	_, err := io.ReadFull(c.conn, prefix[:])
	if err != nil {
		return err
	}

	switch binary.BigEndian.Uint16(prefix[:2]) {
	case types.PongV1:
		log.Println("Got pong message")
	case types.EchoV1:
		buf := make([]byte, binary.BigEndian.Uint64(prefix[2:]))
		_, err = io.ReadFull(c.conn, buf)
		if err != nil {
			return err
		}
		log.Println("Got an echo:", string(buf))
	case types.MsgV1:
		buf := make([]byte, binary.BigEndian.Uint64(prefix[2:]))
		_, err = io.ReadFull(c.conn, buf)
		if err != nil {
			return err
		}
		log.Println("Got an async message:", string(buf))
	}

	return nil
}
