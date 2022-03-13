package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Alextopher/itl-chorus/shared"
)

func main() {
	// initilize rng
	rand.Seed(time.Now().UnixNano())

	if len(os.Args) != 2 {
		fmt.Println("Usage: midi-reader <midifile>")
		os.Exit(1)
	}

	// Listen for CAPS packets
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 12074})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Spawn a goroutine to handle sending and receiving messages
	send := make(chan shared.Message, 10)
	recv := make(chan shared.Message, 10)

	go shared.Recv(conn, recv)
	go shared.Send(conn, send)

	fmt.Println("Listening on", conn.LocalAddr())

	// Create a ping packet
	ping := shared.RandomPing()

	clients := make([]*net.UDPAddr, 0)

	// Listen for incoming messages for 3 seconds
	timer := time.NewTimer(time.Second * 3)
Loop:
	for {
		select {
		case msg := <-recv:
			fmt.Println("Connection", msg.Pkt, "from", msg.Addr)
			switch msg.Pkt.(type) {
			case *shared.CAPS_Packet:
				// Send a PING packet
				send <- shared.Message{
					Pkt:  ping,
					Addr: msg.Addr,
				}

				clients = append(clients, msg.Addr)
			}
		case <-timer.C:
			break Loop
		}
	}

	fmt.Println("Found", len(clients), "clients")

	// Handle sys interrupt
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig

		pkt := &shared.QUIT_Packet{}
		for _, client := range clients {
			send <- shared.Message{
				Pkt:  pkt,
				Addr: client,
			}
		}

		// Wait for 1 second to make sure all packets are sent
		time.Sleep(time.Second)
		os.Exit(1)
	}()

	fmt.Println("Running makeIV...")

	start := time.Now()
	voices, err := makeIV(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	streams := merge(voices, len(clients))
	fmt.Println("Merged into", len(streams), "streams in", time.Since(start))

	if len(streams) != len(clients) {
		fmt.Println("Found", len(streams), "streams, but we have", len(clients), "clients. Ignoring the extra clients.")
		clients = clients[:len(voices)]
	}

	// begin streaming the voices
	start = time.Now()

	wg := &sync.WaitGroup{}
	for i, client := range clients {
		stream := streams[i]
		wg.Add(1)

		go func(client *net.UDPAddr) {
			defer wg.Done()

			for _, event := range stream.events {
				// Sleep until the event is due
				time.Sleep(time.Until(start.Add(event.rt)))
				fmt.Println(event.rt)

				pkt := shared.PLAY_Packet{
					Duration:  event.dur,
					Frequency: midiNoteToFreq(event.key),
					Amplitude: float32(event.vel) / float32(2048),
					Voice:     1,
				}

				send <- shared.Message{
					Pkt:  &pkt,
					Addr: client,
				}
			}
		}(client)
	}
	wg.Wait()

	pkt := &shared.QUIT_Packet{}
	for _, client := range clients {
		send <- shared.Message{
			Pkt:  pkt,
			Addr: client,
		}
	}

	// Wait for 1 second to make sure all packets are sent
	time.Sleep(time.Second)

	fmt.Println("Done")
}
