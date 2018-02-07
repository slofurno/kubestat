package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var incoming = make(chan []byte, 4096)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	connections []*Conn
	mu          sync.Mutex
	history     [][]byte
}

var hub = &Hub{
	connections: []*Conn{},
	history:     [][]byte{},
}

func (s *Hub) Add(conn *Conn) func() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.connections = append(s.connections, conn)
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i := range s.connections {
			if conn == s.connections[i] {
				s.connections = append(s.connections[:i], s.connections[i+1:]...)
				close(conn.send)
				return
			}
		}
	}
}

func (s *Hub) History() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.history[:]
}

func (s *Hub) Broadcast(n []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.connections {
		s.connections[i].send <- n
	}

	s.history = append(s.history, n)
	if len(s.history) > 300 {
		s.history = s.history[1:]
	}
}

type Conn struct {
	*websocket.Conn
	send chan []byte
}

func (s *Conn) Send(x []byte) {
	s.send <- x
}

func (s *Conn) writeLoop() {
	defer func() {
		s.Close()
	}()

	for {
		select {
		case msg, ok := <-s.send:
			if !ok {
				s.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			wc, err := s.NextWriter(websocket.TextMessage)
			if err != nil {
				fmt.Println(err)
				return
			}
			_, err = wc.Write(msg)
			if err != nil {
				fmt.Println(err)
				return
			}
			wc.Close()
		}
	}
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	conn := &Conn{ws, make(chan []byte, 64)}
	remove := hub.Add(conn)
	defer remove()

	go conn.writeLoop()

	history := hub.History()
	for i := range history {
		conn.Send(history[i])
	}

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			return
		}
	}
}

var qp int = 0

func pushStats(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err == nil {
		select {
		case incoming <- b:
		default:
			if qp++; qp&31 == 0 {
				log.Printf("incoming queue full\n")
			}
		}
		return
	}

	w.Write([]byte("OK"))
}

func getPodStats(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	start, _ := strconv.Atoi(query.Get("start"))
	end, _ := strconv.Atoi(query.Get("end"))

	q := PodStatQuery{
		start: start,
		end:   end,
		name:  query.Get("name"),
	}

	xs, err := store.Get(q)
	if err != nil {
		log.Println(err)
		w.WriteHeader(503)
		return
	}

	b, err := json.Marshal(xs)
	if err != nil {
		log.Println(err)
	}
	w.Write(b)
}

var port string
var store *Store

func init() {
	flag.StringVar(&port, "port", "8080", "port")
	flag.Parse()
}

func main() {
	var err error
	store, err = NewPostgresStore("postgres://postgres:postgres@localhost/postgres?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for b := range incoming {
			var xs []PodStat
			if err := json.Unmarshal(b, &xs); err != nil {
				log.Println(err)
				return
			}

			hub.Broadcast(b)

			if err := store.Put(xs); err != nil {
				log.Println(err)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/stats", pushStats)
	mux.HandleFunc("/ws", websocketHandler)
	mux.HandleFunc("/api/stats", getPodStats)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.Marshal(struct {
			Incoming int
		}{
			Incoming: len(incoming),
		})

		w.Write(b)
	})

	mux.Handle("/", http.FileServer(http.Dir("static")))

	server := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
		Addr:         ":" + port,
	}

	server.ListenAndServe()
}
