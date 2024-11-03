package server

import (
	"fmt"
	"net/http"
	"test/handlers"
)

type Server struct {
	HttpServer *http.Server
	Handler    handlers.HandleProcesser
}

func NewSrv() Server {
	h := handlers.NewHandler()
	srv := new(http.Server)
	return Server{HttpServer: srv, Handler: h}
}

// Инициализирует сервер со всем нужными ручками для работы с фронтом.
func (s Server) Run(port string) {

	webDir := "./web"

	http.Handle("/", http.FileServer(http.Dir(webDir)))

	http.HandleFunc("/api/nextdate", s.Handler.HandleDate)

	http.HandleFunc("/api/task", s.Handler.AuthMiddleware(s.Handler.HandleTask))

	http.HandleFunc("/api/tasks", s.Handler.AuthMiddleware(s.Handler.GetTasksHandle))

	http.HandleFunc("/api/task/done", s.Handler.AuthMiddleware(s.Handler.DoneTaskeHandle))

	http.HandleFunc("/api/signin", s.Handler.Auth)

	fmt.Println("Server starting at", port)

	err := http.ListenAndServe(port, nil)
	if err != nil {
		panic(err)
	}
}
