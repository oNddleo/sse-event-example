package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

type Broker struct {
	clients        map[chan string]bool
	newClients     chan chan string
	defunctClients chan chan string
	messages       chan string
}

func (b *Broker) Start() {
	go func() {
		for {
			select {
			case s := <-b.newClients:
				b.clients[s] = true
				log.Println("Added new client")

			case s := <-b.defunctClients:
				delete(b.clients, s)
				close(s)
				log.Println("Removed client")

			case msg := <-b.messages:
				for s := range b.clients {
					s <- msg
				}
				log.Printf("Broadcast message to %d clients", len(b.clients))
			}
		}
	}()
}

func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	messageChan := make(chan string)
	b.newClients <- messageChan

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Use the context to listen for the connection closing.
	go func() {
		<-ctx.Done()
		b.defunctClients <- messageChan
		log.Println("HTTP connection just closed.")
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	for {
		msg, open := <-messageChan

		if !open {
			break
		}

		fmt.Fprintf(w, "data: Message: %s\n\n", msg)
		f.Flush()
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal("Error parsing the template.")
	}

	t.Execute(w, "friend")
	log.Println("Finished HTTP request at", r.URL.Path)
}

func main() {
	b := &Broker{
		clients:        make(map[chan string]bool),
		newClients:     make(chan (chan string)),
		defunctClients: make(chan (chan string)),
		messages:       make(chan string),
	}

	b.Start()
	// handle events
	http.Handle("/events/", b)

	go func() {
		for i := 0; ; i++ {
			b.messages <- fmt.Sprintf("%d - the time is %v", i, time.Now())
			log.Printf("Sent message %d ", i)
			time.Sleep(1e9)
		}
	}()
	// handle open browser
	http.Handle("/", http.HandlerFunc(handler))
	http.ListenAndServe(":8000", nil)
}
