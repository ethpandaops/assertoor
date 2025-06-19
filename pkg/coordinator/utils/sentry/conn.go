package sentry

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/erigontech/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/rlpx"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	pretty = spew.ConfigState{
		Indent:                  "  ",
		DisableCapacities:       true,
		DisablePointerAddresses: true,
		SortKeys:                true,
	}
	timeout = 2 * time.Second
)

// Conn represents an individual connection with a peer
type Conn struct {
	*rlpx.Conn
	ourKey                     *ecdsa.PrivateKey
	negotiatedProtoVersion     uint
	negotiatedSnapProtoVersion uint
	ourHighestProtoVersion     uint
	ourHighestSnapProtoVersion uint
	caps                       []p2p.Cap
}

// Read reads a packet from the connection.
func (c *Conn) Read() (code uint64, data []byte, err error) {
	err = c.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return 0, nil, err
	}

	code, data, _, err = c.Conn.Read()
	if err != nil {
		return 0, nil, err
	}

	return code, data, nil
}

// ReadMsg attempts to read a devp2p message with a specific code.
func (c *Conn) ReadMsg(proto Proto, code uint64, msg any) error {
	err := c.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return err
	}

	for {
		got, data, err := c.Read()
		if err != nil {
			return err
		}

		if protoOffset(proto)+code == got {
			return rlp.DecodeBytes(data, msg)
		}
	}
}

// Write writes a eth packet to the connection.
func (c *Conn) Write(proto Proto, code uint64, msg any) error {
	err := c.SetWriteDeadline(time.Now().Add(timeout))
	if err != nil {
		return err
	}

	payload, err := rlp.EncodeToBytes(msg)
	if err != nil {
		return err
	}

	_, err = c.Conn.Write(protoOffset(proto)+code, payload)

	return err
}

var errDisc error = fmt.Errorf("disconnect")

// ReadEth reads an Eth sub-protocol wire message.
func (c *Conn) ReadEth() (any, error) {
	err := c.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}

	for {
		code, data, _, err := c.Conn.Read()
		if code == discMsg {
			return nil, errDisc
		}

		if err != nil {
			return nil, err
		}

		if code == pingMsg {
			if err := c.Write(baseProto, pongMsg, []byte{}); err != nil {
				return nil, fmt.Errorf("failed to write pong: %v", err)
			}

			continue
		}

		if getProto(code) != ethProto {
			// Read until eth message.
			continue
		}

		code -= baseProtoLen

		var msg any

		if code > math.MaxInt {
			return nil, fmt.Errorf("message code too large: %d", code)
		}

		switch int(code) {
		case eth.StatusMsg:
			msg = new(eth.StatusPacket)
		case eth.GetBlockHeadersMsg:
			msg = new(eth.GetBlockHeadersPacket)
		case eth.BlockHeadersMsg:
			msg = new(eth.BlockHeadersPacket)
		case eth.GetBlockBodiesMsg:
			msg = new(eth.GetBlockBodiesPacket)
		case eth.BlockBodiesMsg:
			msg = new(eth.BlockBodiesPacket)
		case eth.NewBlockMsg:
			msg = new(eth.NewBlockPacket)
		case eth.NewBlockHashesMsg:
			msg = new(eth.NewBlockHashesPacket)
		case eth.TransactionsMsg:
			msg = new(eth.TransactionsPacket)
		case eth.NewPooledTransactionHashesMsg:
			msg = new(eth.NewPooledTransactionHashesPacket)
		case eth.GetPooledTransactionsMsg:
			msg = new(eth.GetPooledTransactionsPacket)
		case eth.PooledTransactionsMsg:
			msg = new(eth.PooledTransactionsPacket)
		default:
			panic(fmt.Sprintf("unhandled eth msg code %d", code))
		}

		if err := rlp.DecodeBytes(data, msg); err != nil {
			return nil, fmt.Errorf("unable to decode eth msg: %v", err)
		}

		return msg, nil
	}
}

// peer performs both the protocol handshake and the status message
// exchange with the node in order to peer with it.
func (c *Conn) Peer(chainID *big.Int, genesisHash, headHash common.Hash, forkID forkid.ID, status *eth.StatusPacket) error {
	if err := c.handshake(); err != nil {
		return fmt.Errorf("handshake failed: %v", err)
	}

	if err := c.statusExchange(chainID, genesisHash, headHash, forkID, status); err != nil {
		return fmt.Errorf("status exchange failed: %v", err)
	}

	return nil
}

// handshake performs a protocol handshake with the node.
func (c *Conn) handshake() error {
	// Write hello to client.
	pub0 := crypto.FromECDSAPub(&c.ourKey.PublicKey)[1:]

	ourHandshake := &protoHandshake{
		Version: 5,
		Caps:    c.caps,
		ID:      pub0,
	}
	if err := c.Write(baseProto, handshakeMsg, ourHandshake); err != nil {
		return fmt.Errorf("write to connection failed: %v", err)
	}
	// Read hello from client.
	code, data, err := c.Read()
	if err != nil {
		return fmt.Errorf("erroring reading handshake: %v", err)
	}

	switch code {
	case handshakeMsg:
		msg := new(protoHandshake)
		if err := rlp.DecodeBytes(data, &msg); err != nil {
			return fmt.Errorf("error decoding handshake msg: %v", err)
		}

		fmt.Println("handshake msg", msg)
		// Set snappy if version is at least 5.
		if msg.Version >= 5 {
			c.SetSnappy(true)
		}

		c.negotiateEthProtocol(msg.Caps)

		if c.negotiatedProtoVersion == 0 {
			return fmt.Errorf("could not negotiate eth protocol (remote caps: %v, local eth version: %v)", msg.Caps, c.ourHighestProtoVersion)
		}
		// If we require snap, verify that it was negotiated.
		if c.ourHighestSnapProtoVersion != c.negotiatedSnapProtoVersion {
			return fmt.Errorf("could not negotiate snap protocol (remote caps: %v, local snap version: %v)", msg.Caps, c.ourHighestSnapProtoVersion)
		}

		return nil
	default:
		return fmt.Errorf("bad handshake: got msg code %d", code)
	}
}

// negotiateEthProtocol sets the Conn's eth protocol version to highest
// advertised capability from peer.
func (c *Conn) negotiateEthProtocol(caps []p2p.Cap) {
	var highestEthVersion uint

	var highestSnapVersion uint

	for _, capability := range caps {
		switch capability.Name {
		case "eth":
			if capability.Version > highestEthVersion && capability.Version <= c.ourHighestProtoVersion {
				highestEthVersion = capability.Version
			}
		case "snap":
			if capability.Version > highestSnapVersion && capability.Version <= c.ourHighestSnapProtoVersion {
				highestSnapVersion = capability.Version
			}
		}
	}

	c.negotiatedProtoVersion = highestEthVersion
	c.negotiatedSnapProtoVersion = highestSnapVersion
}

// statusExchange performs a `Status` message exchange with the given node.
func (c *Conn) statusExchange(chainID *big.Int, genesisHash, headHash common.Hash, forkID forkid.ID, status *eth.StatusPacket) error {
loop:
	for {
		code, data, err := c.Read()
		if err != nil {
			return fmt.Errorf("failed to read from connection: %w", err)
		}
		switch code {
		case eth.StatusMsg + protoOffset(ethProto):
			msg := new(eth.StatusPacket)
			if err := rlp.DecodeBytes(data, &msg); err != nil {
				return fmt.Errorf("error decoding status packet: %w", err)
			}
			if c.ourHighestProtoVersion > math.MaxUint32 {
				return fmt.Errorf("protocol version too large: %d", c.ourHighestProtoVersion)
			}
			if have, want := msg.ProtocolVersion, uint32(c.ourHighestProtoVersion); have != want {
				return fmt.Errorf("wrong protocol version: have %v, want %v", have, want)
			}
			fmt.Println("status msg", msg)
			break loop
		case discMsg:
			var msg []p2p.DiscReason
			if err := rlp.DecodeBytes(data, &msg); err != nil {
				return fmt.Errorf("failed to decode disconnect message: %v", err)
			}
			if len(msg) == 0 {
				return errors.New("invalid disconnect message")
			}
			return fmt.Errorf("disconnect received: %v", pretty.Sdump(msg))
		case pingMsg:
			// TODO (renaynay): in the future, this should be an error
			// (PINGs should not be a response upon fresh connection)
			if err := c.Write(baseProto, pongMsg, nil); err != nil {
				return fmt.Errorf("failed to write pong: %v", err)
			}
		default:
			return fmt.Errorf("bad status message: code %d", code)
		}
	}
	// make sure eth protocol version is set for negotiation
	if c.negotiatedProtoVersion == 0 {
		return errors.New("eth protocol version must be set in Conn")
	}

	if status == nil {
		if c.negotiatedProtoVersion > math.MaxUint32 {
			return fmt.Errorf("negotiated protocol version too large: %d", c.negotiatedProtoVersion)
		}
		// default status message
		status = &eth.StatusPacket{
			ProtocolVersion: uint32(c.negotiatedProtoVersion),
			NetworkID:       chainID.Uint64(),
			TD:              new(big.Int).SetUint64(0),
			Head:            headHash,
			Genesis:         genesisHash,
			ForkID:          forkID,
		}
	}

	if err := c.Write(ethProto, eth.StatusMsg, status); err != nil {
		return fmt.Errorf("write to connection failed: %v", err)
	}

	return nil
}

// readUntil reads eth protocol messages until a message of the target type is
// received.  It returns an error if there is a disconnect, timeout expires,
// or if the context is cancelled before a message of the desired type can be read.
func readUntil[T any](ctx context.Context, conn *Conn) (*T, error) {
	resultCh := make(chan *T, 1)
	errCh := make(chan error, 1)

	go func() {
		defer close(resultCh)
		defer close(errCh)

		for {
			received, err := conn.ReadEth()
			if err != nil {
				if err == errDisc {
					errCh <- errDisc
					return
				}

				continue
			}

			if res, ok := received.(*T); ok {
				resultCh <- res
				return
			}
		}
	}()

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("timeoutExpired")
	}
}

// readTransactionMessages reads transaction messages from the connection.
// The timeout parameter is optional - if provided and > 0, the function will timeout after the specified duration.
func (c *Conn) ReadTransactionMessages(timeout ...time.Duration) (*eth.TransactionsPacket, error) {
	ctx := context.Background()

	if len(timeout) > 0 && timeout[0] > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout[0])
		defer cancel()
	}

	return readUntil[eth.TransactionsPacket](ctx, c)
}

// dialAs attempts to dial a given node and perform a handshake using the generated
// private key.
func dialAs(remoteAddress string) (*Conn, error) {
	key, _ := crypto.GenerateKey()

	node, err := enode.Parse(enode.ValidSchemes, remoteAddress)
	if err != nil {
		return nil, err
	}

	fd, err := net.Dial("tcp", node.IPAddr().String()+":"+strconv.FormatInt(int64(node.TCP()), 10))
	if err != nil {
		return nil, err
	}

	conn := Conn{Conn: rlpx.NewConn(fd, node.Pubkey())}
	conn.ourKey = key
	_, err = conn.Handshake(conn.ourKey)

	if err != nil {
		conn.Close()
		return nil, err
	}

	conn.caps = []p2p.Cap{
		{Name: "eth", Version: 67},
		{Name: "eth", Version: 68},
	}
	conn.ourHighestProtoVersion = 68

	return &conn, nil
}

// GetTCPConn dials the TCP wire eth connection to the given client retrieving the
// node information using the `admin_nodeInfo` method.
func GetTCPConn(client *execution.Client) (*Conn, error) {
	r, err := http.Post(client.GetEndpointConfig().URL, "application/json", strings.NewReader(
		`{"jsonrpc":"2.0","method":"admin_nodeInfo","params":[],"id":1}`,
	))

	if err != nil {
		return nil, err
	}

	defer r.Body.Close()

	var resp struct {
		Result struct {
			Enode     string `json:"enode"`
			Protocols struct {
				Eth struct {
					Genesis    string `json:"genesis"`
					Network    int    `json:"network"`
					Difficulty int    `json:"difficulty"`
				} `json:"eth"`
			} `json:"protocols"`
		} `json:"result"`
	}

	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return nil, err
	}

	return dialAs(resp.Result.Enode)
}
