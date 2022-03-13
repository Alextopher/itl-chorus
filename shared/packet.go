package shared

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"
)

type PacketType uint32

const (
	KA PacketType = iota // Keep Alive
	PING
	QUIT
	PLAY    // [0] uint duration seconds [1] uint nanoseconds offest [2] frequency [3] Amplitude [4] Voice
	CAPS    // [0] name [1] number of voices [2-7] identity
	UNKNOWN = 0xFFFFFFFF
)

type Packet interface {
	fmt.Stringer
	Type() PacketType
	Serialize() []byte
	DeSerialize(data []byte) error
}

// Keep Alive Packet (KA)
// [0-31] unused
type KA_Packet struct{}

func (*KA_Packet) Type() PacketType {
	return KA
}

func (*KA_Packet) Serialize() []byte {
	b := make([]byte, 32)
	return b
}

func (*KA_Packet) DeSerialize(data []byte) error {
	if len(data) != 32 {
		return fmt.Errorf("invalid PING_Packet data length %d byte", len(data))
	}

	return nil
}

func (*KA_Packet) String() string {
	return "KA"
}

// Ping Packet (PING)
// [0-31] bytes to be echoed back
type PING_Packet []byte

func RandomPing() PING_Packet {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	return PING_Packet(b)
}

func (PING_Packet) Type() PacketType {
	return PING
}

func (p PING_Packet) Serialize() []byte {
	return p
}

func (p PING_Packet) DeSerialize(data []byte) error {
	if len(data) != 32 {
		return fmt.Errorf("invalid PING_Packet data length %d byte", len(data))
	}

	p = data
	return nil
}

func (p PING_Packet) String() string {
	s := "PING("
	for _, b := range p {
		s += fmt.Sprintf("%02x", b)
	}
	s += ")"
	return s
}

// Quit Packet (QUIT)
// [0-31] unused
type QUIT_Packet struct{}

func (*QUIT_Packet) Type() PacketType {
	return QUIT
}

func (*QUIT_Packet) Serialize() []byte {
	b := make([]byte, 32)
	return b
}

func (*QUIT_Packet) DeSerialize(data []byte) error {
	return nil
}

func (QUIT_Packet) String() string {
	return "QUIT_Packet"
}

// Play Packet (PLAY)
// [0-3] uint32 duration in seconds
// [4-7] uint32 duration in nanoseconds
// [8-11] uint32 frequency
// [12-15] float32 amplitude
// [16-19] uint32 voice id
// [20-31] unused
type PLAY_Packet struct {
	Duration  time.Duration
	Frequency uint32
	Amplitude float32
	Voice     uint32
}

var padding []byte = make([]byte, 12)

func (*PLAY_Packet) Type() PacketType {
	return PLAY
}

func (p *PLAY_Packet) Serialize() []byte {
	// Create a buffer
	buf := bytes.Buffer{}

	// Write the duration
	binary.Write(&buf, binary.BigEndian, uint32(p.Duration/time.Second))
	binary.Write(&buf, binary.BigEndian, uint32(p.Duration%time.Second))

	// Write the frequency
	binary.Write(&buf, binary.BigEndian, p.Frequency)

	// Write the amplitude
	binary.Write(&buf, binary.BigEndian, p.Amplitude)

	// Write the voice
	binary.Write(&buf, binary.BigEndian, p.Voice)

	// Write 12 bytes of padding
	buf.Write(padding)

	// Return the buffer
	return buf.Bytes()
}

func (p *PLAY_Packet) DeSerialize(data []byte) error {
	if len(data) != 32 {
		return fmt.Errorf("invalid PLAY_Packet data length %d byte", len(data))
	}

	// Create a buffer
	buf := bytes.Buffer{}

	// Write the data
	buf.Write(data)

	// Read the seconds part of the duration
	var seconds, nanoseconds uint32
	binary.Read(&buf, binary.BigEndian, &seconds)
	binary.Read(&buf, binary.BigEndian, &nanoseconds)

	p.Duration = time.Duration(seconds)*time.Second + time.Duration(nanoseconds)

	// Read the frequency
	binary.Read(&buf, binary.BigEndian, &p.Frequency)

	// Read the amplitude
	binary.Read(&buf, binary.BigEndian, &p.Amplitude)

	// Read the voice
	binary.Read(&buf, binary.BigEndian, &p.Voice)

	return nil
}

func (p *PLAY_Packet) String() string {
	return fmt.Sprintf("PLAY(%d, %d, %f, %d)", p.Duration, p.Frequency, p.Amplitude, p.Voice)
}

// Caps Packet (CAPS)
// [0-3] name
// [4-7] uint32 number of voices
// [8-31] identity
type CAPS_Packet struct {
	Name      string
	NumVoices uint32
	Identity  [24]byte
}

func (*CAPS_Packet) Type() PacketType {
	return CAPS
}

func (p *CAPS_Packet) Serialize() []byte {
	// Create a buffer
	buf := bytes.Buffer{}

	// Write the name
	buf.WriteString(p.Name)

	// Write the number of voices
	binary.Write(&buf, binary.LittleEndian, p.NumVoices)

	// Write the identity
	buf.Write(p.Identity[:])

	// Return the buffer
	return buf.Bytes()
}

func (p *CAPS_Packet) DeSerialize(data []byte) error {
	if len(data) != 32 {
		return fmt.Errorf("invalid CAP_Packet data length %d byte", len(data))
	}

	// Create a buffer
	buf := bytes.Buffer{}

	// Write the data
	buf.Write(data)

	// Read the first 4 bytes as the name
	p.Name = string(buf.Next(4))

	// Read the number of voices
	var voices uint32
	err := binary.Read(&buf, binary.LittleEndian, &voices)
	if err != nil {
		return err
	}
	p.NumVoices = voices

	// Read the remaining bytes as the identity
	_, err = buf.Read(p.Identity[:])
	if err != nil {
		return err
	}

	// Return the buffer
	return nil
}

func (p *CAPS_Packet) String() string {
	return fmt.Sprintf("CAPS(%q, %d, %s)", p.Name, p.NumVoices, hex.EncodeToString(p.Identity[:]))
}

type UNKNOWN_Packet []byte

func (UNKNOWN_Packet) Type() PacketType {
	return UNKNOWN
}

func (p UNKNOWN_Packet) Serialize() []byte {
	return p
}

func (p UNKNOWN_Packet) DeSerialize(data []byte) error {
	if len(data) != 32 {
		return fmt.Errorf("invalid UNKNOWN_Packet data length %d byte", len(data))
	}

	p = data
	return nil
}

func (p UNKNOWN_Packet) String() string {
	s := "PING("
	for _, b := range p {
		s += fmt.Sprintf("%02x", b)
	}
	s += ")"
	return s
}
