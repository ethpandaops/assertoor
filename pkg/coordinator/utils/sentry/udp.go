package sentry

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover/v4wire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/sirupsen/logrus"
)

type (
	MessageID int32

	sentryenv struct {
		endpoint   net.PacketConn
		key        *ecdsa.PrivateKey
		remote     *enode.Node
		remoteAddr *net.UDPAddr
		adminRPC   string
	}

	Transaction struct {
		Version uint
	}
)

const (
	waitTime = 300 * time.Millisecond
)

func BasicPing(remoteAddress, rpcAdmin string, logger logrus.FieldLogger) {
	te := connectToP2p(remoteAddress, rpcAdmin, logger)
	defer te.close()

	expiration := time.Now().Add(20 * time.Second).Unix()
	if expiration < 0 {
		logger.Errorf("Invalid expiration time: %d", expiration)
		return
	}

	pingHash := te.send(&v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(),
		To:         te.remoteEndpoint(),
		Expiration: uint64(expiration),
	})
	if err := te.checkPingPong(pingHash); err != nil {
		logger.Errorf("PingPong failed: %v", err)
	}
}

func (te *sentryenv) localEndpoint() v4wire.Endpoint {
	addr, ok := te.endpoint.LocalAddr().(*net.UDPAddr)
	if !ok {
		panic("expected UDP address")
	}

	if addr.Port < 0 || addr.Port > math.MaxUint16 {
		panic(fmt.Sprintf("port out of range: %d", addr.Port))
	}

	return v4wire.Endpoint{
		IP:  addr.IP.To4(),
		UDP: uint16(addr.Port),
		TCP: 0,
	}
}

func (te *sentryenv) remoteEndpoint() v4wire.Endpoint {
	return v4wire.NewEndpoint(te.remoteAddr.AddrPort(), 0)
}

func (te *sentryenv) close() {
	te.endpoint.Close()
}

func (te *sentryenv) send(req v4wire.Packet) []byte {
	packet, hash, err := v4wire.Encode(te.key, req)
	if err != nil {
		panic(fmt.Errorf("can't encode %v packet: %v", req.Name(), err))
	}

	if _, err := te.endpoint.WriteTo(packet, te.remoteAddr); err != nil {
		panic(fmt.Errorf("can't send %v: %v", req.Name(), err))
	}

	return hash
}

func (te *sentryenv) read() (v4wire.Packet, []byte, error) {
	buf := make([]byte, 2048)

	if err := te.endpoint.SetReadDeadline(time.Now().Add(waitTime)); err != nil {
		return nil, nil, err
	}

	n, _, err := te.endpoint.ReadFrom(buf)
	if err != nil {
		return nil, nil, err
	}

	p, _, hash, err := v4wire.Decode(buf[:n])

	return p, hash, err
}

// checkPingPong verifies that the remote side sends both a PONG with the
// correct hash, and a PING.
// The two packets do not have to be in any particular order.
func (te *sentryenv) checkPingPong(pingHash []byte) error {
	var (
		pings int
		pongs int
	)

	for i := 0; i < 2; i++ {
		reply, _, err := te.read()
		if err != nil {
			return err
		}

		switch reply.Kind() {
		case v4wire.PongPacket:
			if err := te.checkPong(reply, pingHash); err != nil {
				return err
			}

			pongs++
		case v4wire.PingPacket:
			pings++
		default:
			return fmt.Errorf("expected PING or PONG, got %v %v", reply.Name(), reply)
		}
	}

	if pongs == 1 && pings == 1 {
		return nil
	}

	return fmt.Errorf("expected 1 PING  (got %d) and 1 PONG (got %d)", pings, pongs)
}

// checkPong verifies that reply is a valid PONG matching the given ping hash,
// and a PING. The two packets do not have to be in any particular order.
func (te *sentryenv) checkPong(reply v4wire.Packet, pingHash []byte) error {
	if reply == nil {
		return errors.New("expected PONG reply, got nil")
	}

	if reply.Kind() != v4wire.PongPacket {
		return fmt.Errorf("expected PONG reply, got %v %v", reply.Name(), reply)
	}

	pong, ok := reply.(*v4wire.Pong)
	if !ok {
		return errors.New("failed to cast reply to *v4wire.Pong")
	}

	if !bytes.Equal(pong.ReplyTok, pingHash) {
		return fmt.Errorf("PONG reply token mismatch: got %x, want %x", pong.ReplyTok, pingHash)
	}

	want := te.localEndpoint()
	if want.IP == nil || want.IP.Equal(net.IPv4zero) {
		want.IP = pong.To.IP
	}

	if !want.IP.Equal(pong.To.IP) || want.UDP != pong.To.UDP {
		return fmt.Errorf("PONG 'to' endpoint mismatch: got %+v, want %+v", pong.To, want)
	}

	if v4wire.Expired(pong.Expiration) {
		return fmt.Errorf("PONG is expired (%v)", pong.Expiration)
	}

	return nil
}

// func (p *sentryenv) notifyWhenReady() (<-chan struct{}, error) {
// 	ready := make(chan struct{})

// 	r, err := http.Post(p.adminRPC, "application/json", strings.NewReader(
// 		`{"jsonrpc":"2.0","method":"admin_peers","params":[],"id":1}`,
// 	))
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer r.Body.Close()

// 	var resp struct {
// 		Result []struct {
// 			Enode string `json:"enode"`
// 		} `json:"result"`
// 	}

// 	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
// 		return nil, err
// 	}

// 	numConn := len(resp.Result)

// 	go func() {
// 		for {
// 			time.Sleep(100 * time.Millisecond)

// 			r, err := http.Post(p.adminRPC, "application/json", strings.NewReader(
// 				`{"jsonrpc":"2.0","method":"admin_peers","params":[],"id":1}`,
// 			))
// 			if err != nil {
// 				continue
// 			}
// 			defer r.Body.Close()

// 			if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
// 				continue
// 			}

// 			if len(resp.Result) > numConn {
// 				ready <- struct{}{}
// 				return
// 			}
// 		}
// 	}()

// 	return ready, nil
// }

func connectToP2p(remoteAddress, rpcAdmin string, logger logrus.FieldLogger) *sentryenv {
	endpoint, err := net.ListenPacket("udp", "0.0.0.0:0")
	if err != nil {
		logger.Errorf("Failed to listen: %v", err)
		return nil
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		logger.Errorf("Failed to generate key: %v", err)
		return nil
	}

	node, err := enode.Parse(enode.ValidSchemes, remoteAddress)
	if err != nil {
		logger.Errorf("Failed to parse node: %v", err)
		return nil
	}

	if !node.IPAddr().IsValid() || node.UDP() == 0 {
		var ip net.IP

		var tcpPort, udpPort int

		if node.IPAddr().IsValid() {
			ip = node.IPAddr().AsSlice()
		} else {
			ip = net.ParseIP("127.0.0.1")
		}

		if tcpPort = node.TCP(); tcpPort == 0 {
			tcpPort = 30303
		}

		if udpPort = node.UDP(); udpPort == 0 {
			udpPort = 30303
		}

		node = enode.NewV4(node.Pubkey(), ip, tcpPort, udpPort)
	}

	addr := &net.UDPAddr{IP: node.IP(), Port: node.UDP()}

	return &sentryenv{endpoint, key, node, addr, rpcAdmin}
}
