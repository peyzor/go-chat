package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"io"
	"log"
	"net/http"
)

type Server struct {
	conns map[*websocket.Conn]string
}

func NewServer() *Server {
	return &Server{
		conns: make(map[*websocket.Conn]string),
	}
}

func (s *Server) handleWS(ws *websocket.Conn) {
	defer func(ws *websocket.Conn) {
		err := ws.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(ws)

	clientID := ws.Request().URL.Path[len("/ws/"):]

	s.conns[ws] = clientID
	s.readLoop(ws)
}

func (s *Server) readLoop(ws *websocket.Conn) {
	for {
		var msg string
		err := websocket.Message.Receive(ws, &msg)
		if err != nil {
			delete(s.conns, ws)

			if err == io.EOF {
				break
			}
			fmt.Println("read error: ", err)
			continue
		}

		message, err := json.Marshal(map[string]any{
			"userId": s.conns[ws],
			"msg":    msg,
		})
		s.broadcast(ws, string(message))
	}

}

func (s *Server) broadcast(sender *websocket.Conn, msg string) {
	for ws := range s.conns {
		if ws == sender {
			continue
		}

		go func(ws *websocket.Conn) {
			err := websocket.Message.Send(ws, msg)
			if err != nil {
				fmt.Println("write error: ", err)
			}
		}(ws)
	}
}

func main() {
	server := NewServer()
	fs := http.FileServer(http.Dir("client"))
	http.Handle("/client/", http.StripPrefix("/client/", fs))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "client/index.html")
	})

	http.Handle("/ws/", websocket.Handler(server.handleWS))
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal(err)
	}
}
