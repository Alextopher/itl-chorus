package shared

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

type Message struct {
	Pkt  Packet
	Addr *net.UDPAddr
}

func Send(conn *net.UDPConn, ch <-chan Message) {
	for msg := range ch {
		buf := new(bytes.Buffer)

		err := binary.Write(buf, binary.LittleEndian, uint32(msg.Pkt.Type()))
		if err != nil {
			fmt.Println("type:", err)
			continue
		}

		serialize := msg.Pkt.Serialize()
		buf.Write(serialize)

		_, err = conn.WriteToUDP(buf.Bytes(), msg.Addr)
		if err != nil {
			fmt.Println(err)
			continue
		}
	}
}

func Recv(conn *net.UDPConn, ch chan<- Message) {
	var buf [36]byte
	for {
		n, addr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			fmt.Println(err)
			close(ch)
			break
		}

		if n != 36 {
			fmt.Println("Invalid packet length", n, "bytes")
			continue
		}

		// Peak the first 4 bytes to determine the packet type
		var tp uint32

		err = binary.Read(bytes.NewReader(buf[0:4]), binary.LittleEndian, &tp)
		if err != nil {
			fmt.Println("type:", err)
			continue
		}

		// make tp a PacketType
		typ := PacketType(tp)

		var p Packet

		switch typ {
		case KA:
			p = &KA_Packet{}
		case PING:
			p = &PING_Packet{}
		case QUIT:
			p = &QUIT_Packet{}
		case PLAY:
			p = &PLAY_Packet{}
		case CAPS:
			p = &CAPS_Packet{}
		default:
			p = &UNKNOWN_Packet{}
		}

		if err = p.DeSerialize(buf[4:]); err != nil {
			fmt.Println("DeSerialize:", err)
			continue
		}

		ch <- Message{Pkt: p, Addr: addr}
	}
}
