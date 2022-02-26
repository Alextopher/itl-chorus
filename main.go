package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Alextopher/itl-ghorus/generators"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

func main() {
	sr := beep.SampleRate(48000)
	speaker.Init(sr, sr.N(time.Second)/10)

	g, err := generators.SawTone(sr, 440)
	// g, err := generators.SawToneReversed(sr, 440)
	// g, err := generators.TriangleTone(sr, 440)
	// g, err := generators.SquareTone(sr, 440)
	// g, err := generators.SinTone(sr, 440)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	speaker.Play(g)

	select {}
}
