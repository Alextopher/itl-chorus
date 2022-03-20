package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/Alextopher/itl-chorus/client/generators"
	"github.com/Alextopher/itl-chorus/shared"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

var sr = beep.SampleRate(48000)

func main() {
	if runtime.GOOS == "linux" {
		// run these two commands to unmute the speakers
		// amixer set Master 100%
		// amixer sset Master unmute
		// amixer set Speaker 100%
		// amixer sset Speaker unmute
		c1 := exec.Command("/bin/bash", "-c", "amixer set Master 50% ; amixer sset Master unmute ; amixer set Speaker 50% ; amixer sset Speaker unmute")
		o, err := c1.CombinedOutput()
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(string(o))
		fmt.Println("Speakers unmuted")
	}

	// initilize speaker
	err := speaker.Init(sr, sr.N(time.Second/1000))
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
	// the server will be listening somewhere in the local network
	broadcastAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:12074")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

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

Start:
	speaker.Clear()

	// Broadcast a CAPS packet until we get a response from the server
	ticker := time.NewTicker(time.Second)

	fmt.Println("Sending CAPS to", broadcastAddr, "...")
Loop:
	for {
		select {
		case <-ticker.C:
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
			pkt := msg.Pkt.(*shared.PLAY_Packet)
			fmt.Println(pkt)

			play(pkt)
		case shared.QUIT:
			fmt.Println("Received QUIT from", msg.Addr)
			goto Start
		}
	}
}

// play plays the given packet to the speakers
func play(pkt *shared.PLAY_Packet) {
	freq := float64(pkt.Frequency)
	wl := int(float64(sr) / freq)

	var g beep.Streamer
	var err error

	// voice encodes which generator to use for the note
	switch pkt.Voice {
	case 0:
		g, err = generators.SineTone(sr, freq)
	case 1:
		g, err = generators.SawtoothTone(sr, freq)
	case 2:
		g, err = generators.SquareTone(sr, freq)
	case 3:
		g, err = generators.TriangleTone(sr, freq)
	}

	// some notes are too short to play without popping
	if err != nil {
		fmt.Println(err)
		return
	}

	// play note until next event
	amp := &Amplitude{streamer: g, amplitude: float64(pkt.Amplitude)}
	samples := sr.N(pkt.Duration)

	// make sure we play an integer number of cycles to avoid "popping"
	samples = (samples / wl) * wl

	speaker.Play(beep.Take(samples, amp))
}

type Amplitude struct {
	streamer  beep.Streamer
	amplitude float64
}

// Stream streams the wrapped Streamer multiplied by max amplitude
func (g *Amplitude) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = g.streamer.Stream(samples)
	for i := range samples[:n] {
		samples[i][0] *= g.amplitude
		samples[i][1] *= g.amplitude
	}
	return n, ok
}

func (g *Amplitude) Err() error {
	return g.streamer.Err()
}
