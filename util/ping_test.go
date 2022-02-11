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
	n, err := enode.ParseV4("enode://07e7bd4a528fae2b57c4e86ce1da50f7530f156a2bfb2349deae38201343abd6b4a4cf9cec03fb611a92e24c824089caa32e6ed2288c147b814e0841202a1989@188.40.141.251:5050")
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
		var h ProtoHandshake
		if err := rlp.DecodeBytes(data, &h); err != nil {
			panic(err)
		}
		fmt.Printf("pinged - ProtoHandshake response:\n%+v\n", h)
	case 1:
		var msg []p2p.DiscReason
		if _ = rlp.DecodeBytes(data, &msg); len(msg) == 0 {
			panic("invalid disconnect message")
		}
		panic(fmt.Errorf("received disconnect message: %v", msg[0]))
	default:
		panic(fmt.Errorf("invalid message code %d, expected handshake (code zero)", code))
	}
}

// ProtoHandshake is the RLP structure of the protocol handshake.
type ProtoHandshake struct {
	Version    uint64
	Name       string
	Caps       []p2p.Cap
	ListenPort uint64
	ID         []byte // secp256k1 public key

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

func (h ProtoHandshake) Code() int { return 0x00 }
