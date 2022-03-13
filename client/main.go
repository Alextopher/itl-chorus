package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/Alextopher/itl-chorus/client/generators"
	"github.com/Alextopher/itl-chorus/shared"
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
)

var sr = beep.SampleRate(44100)

func main() {
	// initilize speaker
	err := speaker.Init(sr, sr.N(time.Second/100))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// initilize rng
	rand.Seed(time.Now().UnixNano())

	// Listen on random local port
	localAddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// define the broadcast address
	broadcastAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:12074")

	// Spawn a goroutine to handle sending and receiving messages
	send := make(chan shared.Message)
	recv := make(chan shared.Message)

	go shared.Recv(conn, recv)
	go shared.Send(conn, send)

	fmt.Println("Listening on", conn.LocalAddr())

	// Choose a random 24 byte identifier
	var id [24]byte
	_, err = rand.Read(id[:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Broadcast a CAPS packet until we get a response from the server
	ticker := time.NewTicker(time.Second)

Loop:
	for {
		select {
		case <-ticker.C:
			fmt.Println("Sending CAPS to", broadcastAddr)
			send <- shared.Message{
				Pkt: &shared.CAPS_Packet{
					Name:      "gogo",
					NumVoices: 1,
					Identity:  id,
				},
				Addr: broadcastAddr,
			}
		case msg := <-recv:
			if msg.Pkt.Type() == shared.PING {
				fmt.Println("Received ping from", msg.Addr)
				break Loop
			}
		}
	}

	// Start listening for PLAY packets
	for msg := range recv {
		switch msg.Pkt.Type() {
		case shared.PLAY:
			fmt.Println("Received PLAY from", msg.Addr)
			pkt := msg.Pkt.(*shared.PLAY_Packet)

			play(pkt)
		}
	}
}

func play(pkt *shared.PLAY_Packet) {
	g, err := generators.TriangleTone(sr, float64(pkt.Frequency))
	if err != nil {
		fmt.Println(err)
		return
	}

	volume := &effects.Volume{
		Streamer: g,
		Base:     2,
		Volume:   -5,
		Silent:   false,
	}

	speaker.Play(beep.Take(sr.N(pkt.Duration), volume))
}
