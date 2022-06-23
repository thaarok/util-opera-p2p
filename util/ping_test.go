package util

import (
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	"github.com/ethereum/go-ethereum/rlp"
	"net"
	"testing"
)

func TestPing(t *testing.T) {
	n, err := enode.ParseV4("enode://a3b164a1b7c562e524353d40a2a1cb7e27f979773d0ade61de67cf268860f7d8b5d24cda03bfce22184624cf912d3f58905f63832155e5f44c965658d6ef9d8b@162.55.0.250:30358")
	if err != nil {
		panic(err)
	}

	fd, err := net.Dial("tcp", fmt.Sprintf("%v:%d", n.IP(), n.TCP()))
	if err != nil {
		panic(err)
	}

	fmt.Printf("dialed %v:%d\n", n.IP(), n.TCP())

	conn := rlpx.NewConn(fd, n.Pubkey())
	ourKey, _ := crypto.GenerateKey()
	_, err = conn.Handshake(ourKey)
	if err != nil {
		panic(fmt.Errorf("handshake failed: %s", err))
	}
	code, data, _, err := conn.Read()
	if err != nil {
		panic(err)
	}
	switch code {
	case 0:
		var h Hello
		if err := rlp.DecodeBytes(data, &h); err != nil {
			panic(err)
		}
		fmt.Printf("pinged - ProtoHandshake response:\n%+v\n", h)
	case 1:
		var msg []p2p.DiscReason
		if _ = rlp.DecodeBytes(data, &msg); len(msg) == 0 {
			panic("received invalid disconnect message")
		}
		panic(fmt.Errorf("received disconnect message: %v", msg[0]))
	default:
		panic(fmt.Errorf("invalid message code %d, expected handshake (code zero)", code))
	}
}
