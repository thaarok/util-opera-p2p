package util

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"reflect"
	"testing"
)

var chain = Chain{
	genesis: core.Genesis{
		Config:     nil,
		Nonce:      0,
		Timestamp:  0,
		ExtraData:  nil,
		GasLimit:   0,
		Difficulty: nil,
		Mixhash:    common.Hash{},
		Coinbase:   common.Address{},
		Alloc:      nil,
		Number:     0,
		GasUsed:    0,
		ParentHash: common.Hash{},
		BaseFee:    nil,
	},
	blocks: []*types.Block{},
	chainConfig: &params.ChainConfig{
		ChainID:             big.NewInt(0xFA),
		HomesteadBlock:      nil,
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.Hash{},
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		CatalystBlock:       big.NewInt(0),
		Ethash:              nil,
		Clique:              nil,
	},
}

var (
	eth66 = true  // indicates whether suite should negotiate eth66 connection
	eth65 = false // indicates whether suite should negotiate eth65 connection or below.
)

func TestGetBlockHeaders(t *testing.T) {
	conn, err := Dial("enode://a3b164a1b7c562e524353d40a2a1cb7e27f979773d0ade61de67cf268860f7d8b5d24cda03bfce22184624cf912d3f58905f63832155e5f44c965658d6ef9d8b@162.55.0.250:30358")
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	if err := conn.peer(&chain, nil); err != nil {
		t.Fatalf("peer failed: %v", err)
	}

	// write request
	req := &GetBlockHeaders{
		Origin: eth.HashOrNumber{
			Hash: chain.blocks[1].Hash(),
		},
		Amount:  2,
		Skip:    1,
		Reverse: false,
	}
	headers, err := conn.headersRequest(req, &chain, eth65, 0)
	if err != nil {
		t.Fatalf("GetBlockHeaders request failed: %v", err)
	}
	// check for correct headers
	expected, err := chain.GetHeaders(*req)
	if err != nil {
		t.Fatalf("failed to get headers for given request: %v", err)
	}
	if !headersMatch(expected, headers) {
		t.Fatalf("header mismatch: \nexpected %v \ngot %v", expected, headers)
	}
}

// headersMatch returns whether the received headers match the given request
func headersMatch(expected BlockHeaders, headers BlockHeaders) bool {
	return reflect.DeepEqual(expected, headers)
}
