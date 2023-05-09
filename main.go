package main

import (
	"log"
	gonet "net"

	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/udp"
	"io"
)

func handleMcast(w mux.ResponseWriter, r *mux.Message) {
	path, err := r.Options.Path()
	if err != nil {
		log.Printf("cannot get path: %v", err)
		return
	}
	b, _ := io.ReadAll(r.Body)

	log.Printf("Got mcast message: path=%q: from %v: %s", path, w.Client().RemoteAddr(),
		string(b))
}

func main() {
	m := mux.NewRouter()
	m.Handle("/cit/d", mux.HandlerFunc(handleMcast))
	m.Handle("/cit/s", mux.HandlerFunc(handleMcast))

	l, err := net.NewListenUDP("udp4", "224.0.1.187:5683")
	if err != nil {
		log.Fatal(err)
	}

	ifaces, err := gonet.Interfaces()
	if err != nil {
		log.Fatal(err)
	}

	g, err := gonet.ResolveUDPAddr("udp", "224.0.1.187:5683")
	if err != nil {
		log.Fatal(err)
	}

	for _, iface := range ifaces {
		err := l.JoinGroup(&iface, g)
		if err != nil {
			log.Printf("cannot JoinGroup(%v, %v): %v", iface, g, err)
		}
	}

	err = l.SetMulticastLoopback(true)
	if err != nil {
		log.Fatal(err)
	}

	defer l.Close()

	s := udp.NewServer(udp.WithMux(m))
	defer s.Stop()
	log.Fatal(s.Serve(l))
}
