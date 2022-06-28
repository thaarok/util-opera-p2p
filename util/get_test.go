package util

import (
	"fmt"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/status-im/keycard-go/hexutils"
	"net"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	n, err := enode.ParseV4("enode://e11ceab68af8b2cc3f9e452cb103728eb294c06e3e4841ef21201420cc5f18fa51f913429b1c66a023f7e09773019a79c188795be97feedadf4e2d0df7924a7e@127.0.0.1:5050")
	if err != nil {
		t.Fatalf("ParseV4 failed: %v", err)
	}
	fd, err := net.Dial("tcp", fmt.Sprintf("%v:%d", n.IP(), n.TCP()))
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	conn := rlpx.NewConn(fd, n.Pubkey())

	// do encHandshake
	ourKey, _ := crypto.GenerateKey()
	_, err = conn.Handshake(ourKey)
	if err != nil {
		conn.Close()
		t.Fatalf("RLPx Handshake failed: %v", err)
	}

	// write hello to client
	pub0 := crypto.FromECDSAPub(&ourKey.PublicKey)[1:]
	ourHello := &Hello{
		Version: 5,
		Caps: []p2p.Cap{
			{Name: "opera", Version: 62},
		},
		ID: pub0,
	}
	payload, err := rlp.EncodeToBytes(ourHello)
	if err != nil {
		t.Fatalf("EncodeToBytes failed: %v", err)
	}
	_, err = conn.Write(uint64(HelloMsg), payload)

	err = receiveMsg(conn)
	if err != nil {
		t.Fatalf("receiveMsg failed: %v", err)
	}

	err = receiveMsg(conn)
	if err != nil {
		t.Fatalf("receiveMsg failed: %v", err)
	}

	err = receiveMsg(conn)
	if err != nil {
		t.Fatalf("receiveMsg failed: %v", err)
	}
}

func receiveMsg(conn *rlpx.Conn) (err error) {
	err = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	if err != nil {
		return fmt.Errorf("unable to SetReadDeadline: %v", err)
	}
	code, rawData, _, err := conn.Read()
	if err != nil {
		return fmt.Errorf("could not read: %v", err)
	}
	rawData = common.CopyBytes(rawData)
	fmt.Printf("received code: %d size: %x hex data: %s\n", code, len(rawData), hexutils.BytesToHex(rawData))

	if code == HelloMsg {
		helloMsg := new(Hello)
		if err = rlp.DecodeBytes(rawData, helloMsg); err != nil {
			return fmt.Errorf("could not rlp decode Hello message: %v", err)
		}

		fmt.Printf("peer version name: %s\n", helloMsg.Name)
		hasOpera := false
		for _, c := range helloMsg.Caps {
			fmt.Printf("capability: %s\n", c.String())
			if c.Name == "opera" && c.Version == 62 {
				hasOpera = true
			}
		}
		fmt.Printf("peer has opera protocol: %t\n", hasOpera)
	}

	if code == DisconnectMsg {
		var disc Disconnect
		if err = rlp.DecodeBytes(rawData, &disc); err != nil {
			return fmt.Errorf("could not rlp decode Disconnect message: %v", err)
		}
		return fmt.Errorf("disconnected! reason: %s\n", disc.Reason.String())
	}

	if code == HandshakeMsg {
		var data handshakeData // expected len: 0x24
		if err = rlp.DecodeBytes(rawData[2:], &data); err != nil {
			return fmt.Errorf("could not rlp decode handshakeData: %v", err)
		}
		fmt.Printf("NetworkID: %X\n", data.NetworkID)
		fmt.Printf("Genesis: %s\n", data.Genesis.String())
	}

	if code == ProgressMsg {
		var progress PeerProgress // expected size: 0x2A
		if err = rlp.DecodeBytes(rawData[2:], &progress); err != nil {
			return fmt.Errorf("could not rlp decode ProgressMsg: %v", err)
		}
		fmt.Printf("Progress - Last Block: %d\n", progress.LastBlockIdx)
		fmt.Printf("Progress - Epoch: %d\n", progress.Epoch)
	}

	return nil
}

const HelloMsg = 0x00

type Hello struct {
	Version    uint64
	Name       string
	Caps       []p2p.Cap
	ListenPort uint64
	ID         []byte // secp256k1 public key

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

const DisconnectMsg = 0x01

// Disconnect is the RLP structure for a disconnect message.
type Disconnect struct {
	Reason p2p.DiscReason
}

const baseProtocolLength = uint64(16)

const HandshakeMsg = baseProtocolLength + 0

type handshakeData struct { // HandshakeMsg
	ProtocolVersion uint32
	NetworkID       uint64
	Genesis         common.Hash
}

const ProgressMsg = baseProtocolLength + 1

type PeerProgress struct {
	Epoch            idx.Epoch
	LastBlockIdx     idx.Block
	LastBlockAtropos hash.Event
	// Currently unused
	HighestLamport idx.Lamport
}
