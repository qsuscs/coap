package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	gonet "net"
	"strings"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/net"
	"github.com/plgd-dev/go-coap/v2/udp"
)

const (
	coiotBase           message.OptionID = 3332 // non critical, save to forward, no cache key
	coiotGlobalDevid    message.OptionID = 3332
	coiotStatusValidity message.OptionID = 3412
	coiotStatusSerial   message.OptionID = 3420
)

func handleMcast(w mux.ResponseWriter, r *mux.Message) {
	path, err := r.Options.Path()
	if err != nil {
		log.Printf("cannot get path: %v", err)
		return
	}
	b, _ := io.ReadAll(r.Body)

	var sb strings.Builder

	for _, o := range r.Options {
		switch o.ID {
		case coiotGlobalDevid:
			sb.WriteString("Device ID: ")
			sb.WriteString(string(o.Value))
			sb.WriteString(", ")
		case coiotStatusValidity:
			sb.WriteString("Status validity: ")
			t := binary.BigEndian.Uint16(o.Value)
			var tm time.Duration
			if t&1 == 0 {
				tm = time.Microsecond * 10 * time.Duration(t)
			} else {
				tm = time.Second * 4 * time.Duration(t)
			}
			sb.WriteString(tm.String())
			sb.WriteString(", ")
		case coiotStatusSerial:
			sb.WriteString("Status serial: ")
			s := binary.BigEndian.Uint16(o.Value)
			sb.WriteString(fmt.Sprint(s))
			sb.WriteString(", ")
		}
	}
	coiot := sb.String()

	log.Printf("Got mcast message: path=%q, source=%v: %s\n%s",
		path, w.Client().RemoteAddr(), string(b),
		coiot[:len(coiot)-2])
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
