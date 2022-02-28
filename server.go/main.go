package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"time"

	"github.com/Alextopher/itl-chorus/generators"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"gitlab.com/gomidi/midi/reader"
)

// Makes sure that the key is in the map
func check(track int16, channel, key uint8) {
	if IVs[track] == nil {
		IVs[track] = make(map[uint8]map[uint8]*voice)
	}

	if IVs[track][channel] == nil {
		IVs[track][channel] = make(map[uint8]*voice)
	}

	if IVs[track][channel][key] == nil {
		IVs[track][channel][key] = &voice{track: track, channel: channel, key: key}
	}
}

func noteOn(p *reader.Position, channel, key, vel uint8) {
	check(p.Track, channel, key)

	rt := *reader.TimeAt(rd, p.AbsoluteTicks)
	IVs[p.Track][channel][key].events = append(IVs[p.Track][channel][key].events, event{
		ticks: p.AbsoluteTicks,
		rt:    rt,
		isOn:  true,
		vel:   vel,
	})

	IVs[p.Track][channel][key].lastOn = rt
}

func noteOff(p *reader.Position, channel, key, vel uint8) {
	check(p.Track, channel, key)

	rt := *reader.TimeAt(rd, p.AbsoluteTicks)
	IVs[p.Track][channel][key].events = append(IVs[p.Track][channel][key].events, event{
		ticks: p.AbsoluteTicks,
		rt:    rt,
		isOn:  false,
		vel:   vel,
	})

	IVs[p.Track][channel][key].totalOnTime += rt - IVs[p.Track][channel][key].lastOn
}

type voice struct {
	events      []event
	track       int16
	channel     uint8
	key         uint8
	totalOnTime time.Duration
	lastOn      time.Duration
}

type event struct {
	ticks uint64
	rt    time.Duration
	isOn  bool
	vel   uint8
}

var IVs map[int16]map[uint8]map[uint8]*voice
var rd *reader.Reader
var sr = beep.SampleRate(44100)
var voices []*voice

func main() {
	speaker.Init(sr, sr.N(time.Second)/1000)

	if len(os.Args) != 2 {
		fmt.Println("Usage: midi-reader <midifile>")
		os.Exit(1)
	}

	IVs = make(map[int16]map[uint8]map[uint8]*voice)

	// to disable logging, pass mid.NoLogger() as option
	rd = reader.New(reader.NoLogger(),
		// set the functions for the messages you are interested in
		reader.NoteOn(noteOn),
		reader.NoteOff(noteOff),
	)

	err := reader.ReadSMFFile(rd, os.Args[1])

	if err != nil {
		fmt.Printf("could not read SMF file %v\n", os.Args[1])
	}

	// extract used voices
	voices = make([]*voice, 0)

	for _, channels := range IVs {
		for _, notes := range channels {
			for _, voice := range notes {
				voices = append(voices, voice)
			}
		}
	}

	// sort by total on time
	sort.Slice(voices, func(i, j int) bool {
		return voices[i].totalOnTime > voices[j].totalOnTime
	})

	// print voices
	for _, voice := range voices {
		fmt.Printf("%v\t%v\t%v\t%v\t\n", voice.track, voice.channel, voice.key, voice.totalOnTime)
	}

	// create a goroutine per voice to print the events
	for _, voice := range voices {
		go printEvents(voice)
	}

	select {}
}

// convert midi note to freq
func midiToFreq(midiNote int) float64 {
	return 440 * math.Pow(2, (float64(midiNote)-69)/12)
}

func printEvents(voice *voice) {
	freq := midiToFreq(int(voice.key))
	wl := int(float64(sr) / freq)

	// sleep until the first event
	time.Sleep(voice.events[0].rt)

	for i := 0; i < len(voice.events)-1; i++ {
		event := voice.events[i]
		next := voice.events[i+1]

		// time change
		d := next.rt - event.rt

		if event.isOn {
			// play note until next event
			g, _ := generators.SawtoothTone(sr, freq)
			amp := &Amplitude{streamer: g, amplitude: 2 * (float64(event.vel) / 127) / float64(len(voices))}

			// the duration of the note is the difference between the next event and the current event
			// round down to the nearest frequency
			samples := sr.N(d)

			// make sure sample / wl is an integer
			samples = (samples / wl) * wl

			// fade := &TakeAndFade{streamer: amp, duration: 10000, total: samples}
			speaker.Play(beep.Take(samples, amp))
		}

		// sleep until next event
		time.Sleep(d)
	}
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
