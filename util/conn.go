// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package util

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	"github.com/ethereum/go-ethereum/rlp"
	"net"
	"reflect"
	"time"
)

var timeout = 20 * time.Second

// Conn represents an individual connection with a peer
type Conn struct {
	*rlpx.Conn
	ourKey                 *ecdsa.PrivateKey
	negotiatedProtoVersion uint
	ourHighestProtoVersion uint
	caps                   []p2p.Cap
}

// Read reads an eth packet from the connection.
func (c *Conn) Read() Message {
	code, rawData, _, err := c.Conn.Read()
	if err != nil {
		return errorf("could not read from connection: %v", err)
	}

	var msg Message
	switch int(code) {
	case (Hello{}).Code():
		msg = new(Hello)
	case (Ping{}).Code():
		msg = new(Ping)
	case (Pong{}).Code():
		msg = new(Pong)
	case (Disconnect{}).Code():
		msg = new(Disconnect)
	case (Status{}).Code():
		msg = new(Status)
	case (GetBlockHeaders{}).Code():
		msg = new(GetBlockHeaders)
	case (BlockHeaders{}).Code():
		msg = new(BlockHeaders)
	case (GetBlockBodies{}).Code():
		msg = new(GetBlockBodies)
	case (BlockBodies{}).Code():
		msg = new(BlockBodies)
	case (NewBlock{}).Code():
		msg = new(NewBlock)
	case (NewBlockHashes{}).Code():
		msg = new(NewBlockHashes)
	case (Transactions{}).Code():
		msg = new(Transactions)
	case (NewPooledTransactionHashes{}).Code():
		msg = new(NewPooledTransactionHashes)
	case (GetPooledTransactions{}.Code()):
		msg = new(GetPooledTransactions)
	case (PooledTransactions{}.Code()):
		msg = new(PooledTransactions)
	default:
		return errorf("invalid message code: %d", code)
	}
	// if message is devp2p, decode here
	if err := rlp.DecodeBytes(rawData, msg); err != nil {
		return errorf("could not rlp decode message: %v", err)
	}
	return msg
}

// Read66 reads an eth66 packet from the connection.
func (c *Conn) Read66() (uint64, Message) {
	code, rawData, _, err := c.Conn.Read()
	if err != nil {
		return 0, errorf("could not read from connection: %v", err)
	}

	var msg Message
	switch int(code) {
	case (Hello{}).Code():
		msg = new(Hello)
	case (Ping{}).Code():
		msg = new(Ping)
	case (Pong{}).Code():
		msg = new(Pong)
	case (Disconnect{}).Code():
		msg = new(Disconnect)
	case (Status{}).Code():
		msg = new(Status)
	case (GetBlockHeaders{}).Code():
		ethMsg := new(eth.GetBlockHeadersPacket66)
		if err := rlp.DecodeBytes(rawData, ethMsg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return ethMsg.RequestId, GetBlockHeaders(*ethMsg.GetBlockHeadersPacket)
	case (BlockHeaders{}).Code():
		ethMsg := new(eth.BlockHeadersPacket66)
		if err := rlp.DecodeBytes(rawData, ethMsg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return ethMsg.RequestId, BlockHeaders(ethMsg.BlockHeadersPacket)
	case (GetBlockBodies{}).Code():
		ethMsg := new(eth.GetBlockBodiesPacket66)
		if err := rlp.DecodeBytes(rawData, ethMsg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return ethMsg.RequestId, GetBlockBodies(ethMsg.GetBlockBodiesPacket)
	case (BlockBodies{}).Code():
		ethMsg := new(eth.BlockBodiesPacket66)
		if err := rlp.DecodeBytes(rawData, ethMsg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return ethMsg.RequestId, BlockBodies(ethMsg.BlockBodiesPacket)
	case (NewBlock{}).Code():
		msg = new(NewBlock)
	case (NewBlockHashes{}).Code():
		msg = new(NewBlockHashes)
	case (Transactions{}).Code():
		msg = new(Transactions)
	case (NewPooledTransactionHashes{}).Code():
		msg = new(NewPooledTransactionHashes)
	case (GetPooledTransactions{}.Code()):
		ethMsg := new(eth.GetPooledTransactionsPacket66)
		if err := rlp.DecodeBytes(rawData, ethMsg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return ethMsg.RequestId, GetPooledTransactions(ethMsg.GetPooledTransactionsPacket)
	case (PooledTransactions{}.Code()):
		ethMsg := new(eth.PooledTransactionsPacket66)
		if err := rlp.DecodeBytes(rawData, ethMsg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return ethMsg.RequestId, PooledTransactions(ethMsg.PooledTransactionsPacket)
	default:
		msg = errorf("invalid message code: %d", code)
	}

	if msg != nil {
		if err := rlp.DecodeBytes(rawData, msg); err != nil {
			return 0, errorf("could not rlp decode message: %v", err)
		}
		return 0, msg
	}
	return 0, errorf("invalid message: %s", string(rawData))
}

// Write writes a eth packet to the connection.
func (c *Conn) Write(msg Message) error {
	// check if message is eth protocol message
	var (
		payload []byte
		err     error
	)
	payload, err = rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(uint64(msg.Code()), payload)
	return err
}

// Write66 writes an eth66 packet to the connection.
func (c *Conn) Write66(req eth.Packet, code int) error {
	payload, err := rlp.EncodeToBytes(req)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(uint64(code), payload)
	return err
}

// peer performs both the protocol handshake and the status message
// exchange with the node in order to peer with it.
func (c *Conn) peer(chain *Chain, status *Status) error {
	if err := c.handshake(); err != nil {
		return fmt.Errorf("handshake failed: %v", err)
	}
	if _, err := c.statusExchange(chain, status); err != nil {
		return fmt.Errorf("status exchange failed: %v", err)
	}
	return nil
}

// handshake performs a protocol handshake with the node.
func (c *Conn) handshake() error {
	defer c.SetDeadline(time.Time{})
	c.SetDeadline(time.Now().Add(10 * time.Second))
	// write hello to client
	pub0 := crypto.FromECDSAPub(&c.ourKey.PublicKey)[1:]
	ourHandshake := &Hello{
		Version: 5,
		Caps:    c.caps,
		ID:      pub0,
	}
	if err := c.Write(ourHandshake); err != nil {
		return fmt.Errorf("write to connection failed: %v", err)
	}
	// read hello from client
	switch msg := c.Read().(type) {
	case *Hello:
		// set snappy if version is at least 5
		if msg.Version >= 5 {
			c.SetSnappy(true)
		}
		c.negotiateEthProtocol(msg.Caps)
		if c.negotiatedProtoVersion == 0 {
			return fmt.Errorf("unexpected eth protocol version")
		}
		return nil
	default:
		return fmt.Errorf("bad handshake: %#v", msg)
	}
}

// negotiateEthProtocol sets the Conn's eth protocol version to highest
// advertised capability from peer.
func (c *Conn) negotiateEthProtocol(caps []p2p.Cap) {
	var highestEthVersion uint
	for _, capability := range caps {
		if capability.Name != "eth" {
			continue
		}
		if capability.Version > highestEthVersion && capability.Version <= c.ourHighestProtoVersion {
			highestEthVersion = capability.Version
		}
	}
	c.negotiatedProtoVersion = highestEthVersion
}

// statusExchange performs a `Status` message exchange with the given node.
func (c *Conn) statusExchange(chain *Chain, status *Status) (Message, error) {
	defer c.SetDeadline(time.Time{})
	c.SetDeadline(time.Now().Add(20 * time.Second))

	// read status message from client
	var message Message
loop:
	for {
		switch msg := c.Read().(type) {
		case *Status:
			if have, want := msg.Head, chain.blocks[chain.Len()-1].Hash(); have != want {
				return nil, fmt.Errorf("wrong head block in status, want:  %#x (block %d) have %#x",
					want, chain.blocks[chain.Len()-1].NumberU64(), have)
			}
			if have, want := msg.TD.Cmp(chain.TD()), 0; have != want {
				return nil, fmt.Errorf("wrong TD in status: have %v want %v", have, want)
			}
			if have, want := msg.ForkID, chain.ForkID(); !reflect.DeepEqual(have, want) {
				return nil, fmt.Errorf("wrong fork ID in status: have %v, want %v", have, want)
			}
			if have, want := msg.ProtocolVersion, c.ourHighestProtoVersion; have != uint32(want) {
				return nil, fmt.Errorf("wrong protocol version: have %v, want %v", have, want)
			}
			message = msg
			break loop
		case *Disconnect:
			return nil, fmt.Errorf("disconnect received: %v", msg.Reason)
		case *Ping:
			c.Write(&Pong{}) // TODO (renaynay): in the future, this should be an error
			// (PINGs should not be a response upon fresh connection)
		default:
			return nil, fmt.Errorf("bad status message: %s", msg)
		}
	}
	// make sure eth protocol version is set for negotiation
	if c.negotiatedProtoVersion == 0 {
		return nil, fmt.Errorf("eth protocol version must be set in Conn")
	}
	if status == nil {
		// default status message
		status = &Status{
			ProtocolVersion: uint32(c.negotiatedProtoVersion),
			NetworkID:       chain.chainConfig.ChainID.Uint64(),
			TD:              chain.TD(),
			Head:            chain.blocks[chain.Len()-1].Hash(),
			Genesis:         chain.blocks[0].Hash(),
			ForkID:          chain.ForkID(),
		}
	}
	if err := c.Write(status); err != nil {
		return nil, fmt.Errorf("write to connection failed: %v", err)
	}
	return message, nil
}

// Dial attempts to dial the given node and perform a handshake,
// returning the created Conn if successful.
func Dial(nodeUri string) (*Conn, error) {
	// dial
	n, err := enode.ParseV4(nodeUri)
	if err != nil {
		return nil, err
	}
	fd, err := net.Dial("tcp", fmt.Sprintf("%v:%d", n.IP(), n.TCP()))
	if err != nil {
		return nil, err
	}
	conn := Conn{Conn: rlpx.NewConn(fd, n.Pubkey())}
	// do encHandshake
	conn.ourKey, _ = crypto.GenerateKey()
	_, err = conn.Handshake(conn.ourKey)
	if err != nil {
		conn.Close()
		return nil, err
	}
	// set default p2p capabilities
	conn.caps = []p2p.Cap{
		{Name: "eth", Version: 64},
		{Name: "eth", Version: 65},
	}
	conn.ourHighestProtoVersion = 65
	return &conn, nil
}

// Dial66 attempts to dial the given node and perform a handshake,
// returning the created Conn with additional eth66 capabilities if
// successful
func Dial66(nodeUri string) (*Conn, error) {
	conn, err := Dial(nodeUri)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %v", err)
	}
	conn.caps = append(conn.caps, p2p.Cap{Name: "eth", Version: 66})
	conn.ourHighestProtoVersion = 66
	return conn, nil
}

// headersRequest executes the given `GetBlockHeaders` request.
func (c *Conn) headersRequest(request *GetBlockHeaders, chain *Chain, isEth66 bool, reqID uint64) (BlockHeaders, error) {
	defer c.SetReadDeadline(time.Time{})
	c.SetReadDeadline(time.Now().Add(20 * time.Second))
	// if on eth66 connection, perform eth66 GetBlockHeaders request
	if isEth66 {
		return getBlockHeaders66(chain, c, request, reqID)
	}
	if err := c.Write(request); err != nil {
		return nil, err
	}
	switch msg := c.readAndServe(chain, timeout).(type) {
	case *BlockHeaders:
		return *msg, nil
	default:
		return nil, fmt.Errorf("invalid message: %s", msg)
	}
}

// getBlockHeaders66 executes the given `GetBlockHeaders` request over the eth66 protocol.
func getBlockHeaders66(chain *Chain, conn *Conn, request *GetBlockHeaders, id uint64) (BlockHeaders, error) {
	// write request
	packet := eth.GetBlockHeadersPacket(*request)
	req := &eth.GetBlockHeadersPacket66{
		RequestId:             id,
		GetBlockHeadersPacket: &packet,
	}
	if err := conn.Write66(req, GetBlockHeaders{}.Code()); err != nil {
		return nil, fmt.Errorf("could not write to connection: %v", err)
	}
	// wait for response
	msg := conn.waitForResponse(chain, timeout, req.RequestId)
	headers, ok := msg.(BlockHeaders)
	if !ok {
		return nil, fmt.Errorf("unexpected message received: %s", msg)
	}
	return headers, nil
}

// readAndServe serves GetBlockHeaders requests while waiting
// on another message from the node.
func (c *Conn) readAndServe(chain *Chain, timeout time.Duration) Message {
	start := time.Now()
	for time.Since(start) < timeout {
		c.SetReadDeadline(time.Now().Add(5 * time.Second))
		switch msg := c.Read().(type) {
		case *Ping:
			c.Write(&Pong{})
		case *GetBlockHeaders:
			req := *msg
			headers, err := chain.GetHeaders(req)
			if err != nil {
				return errorf("could not get headers for inbound header request: %v", err)
			}
			if err := c.Write(headers); err != nil {
				return errorf("could not write to connection: %v", err)
			}
		default:
			return msg
		}
	}
	return errorf("no message received within %v", timeout)
}

// readAndServe66 serves eth66 GetBlockHeaders requests while waiting
// on another message from the node.
func (c *Conn) readAndServe66(chain *Chain, timeout time.Duration) (uint64, Message) {
	start := time.Now()
	for time.Since(start) < timeout {
		c.SetReadDeadline(time.Now().Add(10 * time.Second))

		reqID, msg := c.Read66()

		switch msg := msg.(type) {
		case *Ping:
			c.Write(&Pong{})
		case *GetBlockHeaders:
			headers, err := chain.GetHeaders(*msg)
			if err != nil {
				return 0, errorf("could not get headers for inbound header request: %v", err)
			}
			resp := &eth.BlockHeadersPacket66{
				RequestId:          reqID,
				BlockHeadersPacket: eth.BlockHeadersPacket(headers),
			}
			if err := c.Write66(resp, BlockHeaders{}.Code()); err != nil {
				return 0, errorf("could not write to connection: %v", err)
			}
		default:
			return reqID, msg
		}
	}
	return 0, errorf("no message received within %v", timeout)
}

// waitForResponse reads from the connection until a response with the expected
// request ID is received.
func (c *Conn) waitForResponse(chain *Chain, timeout time.Duration, requestID uint64) Message {
	for {
		id, msg := c.readAndServe66(chain, timeout)
		if id == requestID {
			return msg
		}
	}
}
