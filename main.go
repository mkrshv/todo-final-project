package main

import (
	"os"
	"test/server"
)

func main() {

	port := ":" + os.Getenv("TODO_PORT")
	if port == ":" {
		port = ":7540"
	}

	srv := server.NewSrv()

	srv.Run(port)
}
