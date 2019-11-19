package main

import (
	"os"
	"time"

	"github.com/tflyons/tcp-demo/client"
	"github.com/tflyons/tcp-demo/server"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	asyncMessages := make(chan []byte, 1)

	s, err := server.NewServer(server.Config{
		Timeout:       time.Second * 3,
		Address:       "0.0.0.0:8888",
		AsyncMessages: asyncMessages,
	})
	check(err)

	go func() {
		c, err := client.NewClient("localhost:8888")
		check(err)

		check(c.SendPing())
		check(c.ReadResponse())
		check(c.Echo("Hello"))
		check(c.ReadResponse())

		file, err := os.Open("hithere.txt")
		check(err)
		check(c.UploadFile(file))

		asyncMessages <- []byte("async message")
		check(c.ReadResponse())
	}()

	err = s.Start()
	check(err)
}
