package main

import (
	"fmt"
	"math"
	"sort"
	"time"

	"gitlab.com/gomidi/midi/reader"
)

type stream struct {
	// Measures the total time played
	totalOnTime time.Duration

	events []streamEvent
}

type streamEvent struct {
	key uint8
	vel uint8
	dur time.Duration
	rt  time.Duration
}

func midiNoteToFreq(note uint8) uint32 {
	return uint32(math.Pow(2, float64(note)/12.0) * 8.1758)
}

type voice struct {
	events  []voiceEvent
	track   int16
	channel uint8
	key     uint8

	// Measures the total time the key was on
	totalOnTime time.Duration

	// lastOn is the last time the voice was on
	lastOn time.Duration
}

type voiceEvent struct {
	// Midi ticks when the event happened
	ticks uint64
	// Real time when the event happened (accounts for tempo)
	rt time.Duration
	// Is this event a note on or off
	isOn bool
	// The velocity of the note, intrepreted as amplitude
	vel uint8
}

// TODO pass these values as arguments through a closure
var IVs map[int16]map[uint8]map[uint8]*voice
var rd *reader.Reader

func makeIV(filename string) ([]*voice, error) {
	IVs = make(map[int16]map[uint8]map[uint8]*voice)

	// to disable logging, pass mid.NoLogger() as option
	rd = reader.New(reader.NoLogger(),
		reader.NoteOn(noteOn),
		reader.NoteOff(noteOff),
	)

	err := reader.ReadSMFFile(rd, filename)
	if err != nil {
		return nil, err
	}

	// the extract used voices
	voices := make([]*voice, 0)
	for _, channels := range IVs {
		for _, notes := range channels {
			for _, voice := range notes {
				voices = append(voices, voice)
			}
		}
	}

	// sort by total on time, using this we can fairly merge the voices
	sort.Slice(voices, func(i, j int) bool {
		return voices[i].totalOnTime > voices[j].totalOnTime
	})

	return voices, nil
}

// Fairly merges all the voice events into n voices
func merge(voices []*voice, n int) []stream {
	if n == 0 {
		panic("n must be > 0")
	}

	// Convert the voice structs into stream structs
	streams := make([]stream, len(voices))

	// turn the voices into streams
	for i, voice := range voices {
		stream := stream{
			events:      make([]streamEvent, 0),
			totalOnTime: voice.totalOnTime,
		}

		for i := 0; i < len(voice.events)-1; i++ {
			event := voice.events[i]
			next := voice.events[i+1]

			// how long the note was on
			d := next.rt - event.rt

			if event.isOn {
				stream.events = append(stream.events, streamEvent{
					key: voice.key,
					vel: event.vel,
					dur: d,
					rt:  event.rt,
				})
			}
		}

		streams[i] = stream
	}

	fmt.Println(len(streams))

	// group the streams into n groups
	groups := make([]stream, n)
	totals := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		groups[i].events = make([]streamEvent, 0)
	}

	for _, stream := range streams {
		// for i := 0; i < 4; i++ {
		// 	stream := streams[i]

		// find the smallest group
		min := 0
		for i := 1; i < n; i++ {
			if totals[i] < totals[min] {
				min = i
			}
		}

		// add all events to the group
		groups[min].events = append(groups[min].events, stream.events...)
		totals[min] += stream.totalOnTime
	}

	// Sort groups events by real time
	for i := 0; i < n; i++ {
		sort.Slice(groups[i].events, func(j, k int) bool {
			return groups[i].events[j].rt < groups[i].events[k].rt
		})
	}

	fmt.Println("Total times for each stream:", totals)

	return groups
}

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
	IVs[p.Track][channel][key].events = append(IVs[p.Track][channel][key].events, voiceEvent{
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
	IVs[p.Track][channel][key].events = append(IVs[p.Track][channel][key].events, voiceEvent{
		ticks: p.AbsoluteTicks,
		rt:    rt,
		isOn:  false,
		vel:   vel,
	})

	IVs[p.Track][channel][key].totalOnTime += rt - IVs[p.Track][channel][key].lastOn
}
