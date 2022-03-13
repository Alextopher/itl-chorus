package main

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/Alextopher/itl-chorus/client/generators"
	"github.com/Alextopher/itl-chorus/shared"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

var sr = beep.SampleRate(48000)

func main() {
	fmt.Println(runtime.GOOS, runtime.GOARCH)

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
			pkt := msg.Pkt.(*shared.PLAY_Packet)
			fmt.Println(pkt)

			play(pkt)
		case shared.QUIT:
			goto Start
		}
	}
}

func play(pkt *shared.PLAY_Packet) {
	freq := float64(pkt.Frequency)
	wl := int(float64(sr) / freq)

	g, err := generators.SawtoothTone(sr, freq)
	if err != nil {
		fmt.Println(err)
		return
	}

	// play note until next event
	amp := &Amplitude{streamer: g, amplitude: float64(pkt.Amplitude)}

	// the duration of the note is the difference between the next event and the current event
	// round down to the nearest frequency
	samples := sr.N(pkt.Duration)

	// make sure sample / wl is an integer
	samples = (samples / wl) * wl

	speaker.Play(beep.Take(samples, amp))
}

type Amplitude struct {
	streamer  beep.Streamer
	amplitude float64
}

// Stream streams the wrapped Streamer amplified by Gain.
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
