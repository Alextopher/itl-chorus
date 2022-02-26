package generators

import (
	"errors"
	"math"

	"github.com/faiface/beep"
)

type sineGenerator struct {
	dt float64
	t  float64
}

func SineTone(sr beep.SampleRate, freq float64) (beep.Streamer, error) {
	dt := freq / float64(sr)

	if dt >= 1.0/2.0 {
		return nil, errors.New("faiface sin tone generator: samplerate must be at least 2 times grater then frequency")
	}

	return &sineGenerator{dt, 0}, nil
}

func (g *sineGenerator) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		v := math.Sin(g.t * 2.0 * math.Pi)
		samples[i][0] = v
		samples[i][1] = v
		_, g.t = math.Modf(g.t + g.dt)
	}

	return len(samples), true
}

func (*sineGenerator) Err() error {
	return nil
}
