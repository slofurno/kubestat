package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

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
}

var hub = &Hub{
	connections: []*Conn{},
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

func (s *Hub) Broadcast(n []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.connections {
		s.connections[i].send <- n
	}
}

type Conn struct {
	*websocket.Conn
	send chan []byte
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

	conn := &Conn{ws, make(chan []byte, 32)}
	remove := hub.Add(conn)
	defer remove()

	go conn.writeLoop()

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			return
		}
	}
}

func pushStats(w http.ResponseWriter, r *http.Request) {

	b, _ := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	//var stats []PodStat
	//json.Unmarshal(b, &stats)

	w.Write([]byte("OK"))

	hub.Broadcast(b)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/stats", pushStats)
	mux.HandleFunc("/ws", websocketHandler)
	mux.Handle("/", http.FileServer(http.Dir("static")))

	server := &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
		Addr:         ":8080",
	}

	server.ListenAndServe()
}

type PodStat struct {
	Id string
	//nanoseconds
	Cpuacct_usage    int64
	Cpuacct_usage_d  int64
	Nr_throttled     int64
	Throttled_time   int64
	Throttled_time_d int64
	Total_rss        int64
	Total_cache      int64

	//microseconds
	Cpu_cfs_quota_us  int64
	Cpu_cfs_period_us int64

	Time time.Time
}
