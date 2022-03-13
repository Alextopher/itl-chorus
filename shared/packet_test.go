package shared

import (
	"fmt"
	"testing"
	"time"
)

func TestPlay(t *testing.T) {
	// Create a play packet and check it serializes and deserializes correctly
	play := PLAY_Packet{
		Duration:  time.Second * 5,
		Frequency: 440,
		Amplitude: 0.5,
		Voice:     1,
	}
	fmt.Println(play)

	b := play.Serialize()

	p := &PLAY_Packet{}
	err := p.DeSerialize(b)
	if err != nil {
		t.Error(err)
	}

	if p.Duration != play.Duration {
		t.Errorf("Expected duration %v, got %v", play.Duration, p.Duration)
	}

	if p.Frequency != play.Frequency {
		t.Errorf("Expected frequency %v, got %v", play.Frequency, p.Frequency)
	}

	if p.Amplitude != play.Amplitude {
		t.Errorf("Expected amplitude %v, got %v", play.Amplitude, p.Amplitude)
	}

	if p.Voice != play.Voice {
		t.Errorf("Expected voice %v, got %v", play.Voice, p.Voice)
	}
}
