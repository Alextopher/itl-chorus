package main

import (
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/Alextopher/itl-chorus/shared"
	"golang.org/x/crypto/ssh/terminal"
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
	send := make(chan shared.Message, 50)
	recv := make(chan shared.Message, 50)

	go shared.Recv(conn, recv)
	go shared.Send(conn, send)

	fmt.Println("Listening on", conn.LocalAddr())

	// Create a ping packet
	ping := shared.RandomPing()

	clients := make([]*net.UDPAddr, 0)

	// Listen for incoming messages for 5 second
	timer := time.NewTimer(time.Second * 5)
Loop:
	for {
		select {
		case msg := <-recv:
			switch msg.Pkt.(type) {
			case *shared.CAPS_Packet:
				// We can only support "gogo" clients
				if msg.Pkt.(*shared.CAPS_Packet).Name != "gogo" {
					fmt.Println("Unsupported client:", msg.Pkt.(*shared.CAPS_Packet).Name)
					break
				}

				// Check if we already have this client
				for _, client := range clients {
					if client.String() == msg.Addr.String() {
						continue Loop
					}
				}

				clients = append(clients, msg.Addr)
				fmt.Println("Client connected:", msg.Addr)

				// Send a PING packet
				send <- shared.Message{
					Pkt:  ping,
					Addr: msg.Addr,
				}
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

	start := time.Now()
	voices, err := makeIV(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	streams, duration := merge(voices, len(clients))
	fmt.Println("Duration:", duration)

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

		go func(client *net.UDPAddr, i uint32) {
			defer wg.Done()

			for _, event := range stream.events {
				// Sleep until the event is due
				time.Sleep(time.Until(start.Add(event.rt)))

				pkt := shared.PLAY_Packet{
					Duration:  event.dur,
					Frequency: midiNoteToFreq(event.key),
					Amplitude: float32(math.Sqrt(float64(event.vel)/float64(128))) / 2, // TODO Amplitude should be dependent on the number of clients
					Voice:     1,
				}

				send <- shared.Message{
					Pkt:  &pkt,
					Addr: client,
				}
			}
		}(client, uint32(i))
	}

	// progress bar
	go func() {
		// Calculate the terminal width
		width, _, err := terminal.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Println(err)
			return
		}

		// trim the width so there is room to print "Progress: " and the percentage
		width -= 20

		start := time.Now()
		for {
			// Calculate the progress
			progress := float64(time.Since(start)) / float64(duration)
			if progress > 1 {
				progress = 1
			}

			// Print the progress bar
			fmt.Printf("\rProgress: [%s%s] %.2f%%",
				strings.Repeat("=", int(progress*float64(width))),
				strings.Repeat(" ", int((1-progress)*float64(width))),
				progress*100,
			)

			time.Sleep(time.Millisecond * 100)
		}
	}()

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
	fmt.Print("\n")
}
