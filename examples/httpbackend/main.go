package main

import (
	"encoding/json"
	"log"
	"net"
	"sync"

	"github.com/gorilla/mux"

	"net/http"
)

type Peer struct {
	PublicKey []byte
	Endpoint  string
	IP        *net.IP
}

type Store struct {
	store map[string]Peer
	mutex *sync.RWMutex
}

func (s *Store) write(key string, val Peer) {
	s.mutex.Lock()
	s.store[key] = val
	s.mutex.Unlock()
}

func (s *Store) read() map[string]Peer {
	s.mutex.RLock()
	res := s.store
	s.mutex.RUnlock()
	return res
}

func joinHandler(s *Store) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		sha := mux.Vars(r)["publickeysha"]
		d := json.NewDecoder(r.Body)

		peer := Peer{}
		err := d.Decode(&peer)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		s.write(sha, peer)
		w.WriteHeader(http.StatusCreated)
	}
}

func getPeersHandler(s *Store) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		result := s.read()

		list := []Peer{}
		for _, v := range result {
			list = append(list, v)
		}

		resBody, err := json.Marshal(list)

		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resBody)
	}
}

func main() {
	// just an ephemeral store for this example
	store := &Store{
		mutex: &sync.RWMutex{},
		store: map[string]Peer{},
	}
	r := mux.NewRouter()
	r.HandleFunc("/{ifname}/{publickeysha}", joinHandler(store)).Methods("POST")
	r.HandleFunc("/{ifname}", getPeersHandler(store)).Methods("GET")

	log.Fatal(http.ListenAndServe("0.0.0.0:8080", r))
}
