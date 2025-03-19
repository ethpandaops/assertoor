package txpoolcheck

import (
	"net"
	"bytes"
	"errors"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"net/url"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p/discover/v4wire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "tx_pool_check"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks the throughput and latency of transactions in the Ethereum TxPool",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config
	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	executionClients := clientPool.GetExecutionPool().GetReadyEndpoints(true)

	if len(executionClients) == 0 {
		t.logger.Error("No execution clients available")
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	t.logger.Infof("Testing TxPool with %d transactions", t.config.TxCount)

	chainID, err := executionClients[0].GetRPCClient().GetEthClient().ChainID(ctx)
	if err != nil {
		t.logger.Errorf("Failed to fetch chain ID: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	t.logger.Infof("Chain ID: %d", chainID)

	privKey, err := crypto.HexToECDSA(t.config.PrivateKey)
	if err != nil {
		t.logger.Errorf("Failed to generate private key: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	// get last nonce
	latestBlock, err := executionClients[0].GetRPCClient().GetLatestBlock(ctx)
	if err != nil {
		t.logger.Errorf("Failed to fetch latest block: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	nonce, err := executionClients[0].GetRPCClient().GetEthClient().NonceAt(ctx, crypto.PubkeyToAddress(privKey.PublicKey), latestBlock.Number())
	if err != nil {
		t.logger.Errorf("Failed to fetch nonce: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	if t.config.Nonce != nil {
		t.logger.Infof("Using custom nonce: %d", *t.config.Nonce)
		nonce = *t.config.Nonce
	}

	t.logger.Infof("Starting nonce: %d", nonce)
	clientIndex := rand.Intn(len(executionClients))
	client := executionClients[clientIndex]

	t.logger.Infof("Using client: %s", client.GetName())

	// Extract hostname from client URL, removing protocol and port
	clientURL := client.GetEndpointConfig().URL
	if parsedURL, err := url.Parse(clientURL); err == nil {
		hostname := parsedURL.Hostname()
		BasicPing(hostname, t.logger)
	} else {
		t.logger.Errorf("Failed to parse client URL: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	var totalLatency time.Duration
	retryCount := 0

	for i := 0; i < t.config.TxCount; i++ {
		tx, err := createDummyTransaction(nonce, chainID, privKey)
		if err != nil {
			t.logger.Errorf("Failed to create transaction: %v", err)
			t.ctx.SetResult(types.TaskResultFailure)
			return nil
		}

		startTx := time.Now()

		err = client.GetRPCClient().SendTransaction(ctx, tx)

		if err != nil {
			t.logger.Errorf("Failed to send transaction: %v. Nonce: %d. ", err, nonce)

			// retry increasing the nonce
			nonce++
			i--
			retryCount++

			if retryCount > 1000 {
				t.logger.Errorf("Too many retries")
				t.ctx.SetResult(types.TaskResultFailure)
				return nil
			}

			continue
		}

		retryCount = 0

		// wait for tx to be confirmed
		// for stop := false; !stop; {
		// 	select {
		// 		case msg := <-gotTxCh:
		// 			if msg.MessageID == sentryproto.MessageId_TRANSACTIONS_66 {
		// 				stop = true
		// 			}
		// 		case err := <-errCh:
		// 			t.logger.Errorf("Error receiving tx: %v", err)
		// 	}
		// }

		nonce++

		latency := time.Since(startTx)
		totalLatency += latency

		if (i+1)%t.config.MeasureInterval == 0 {
			avgSoFar := totalLatency.Milliseconds() / int64(i+1)
			t.logger.Infof("Processed %d transactions, current avg latency: %dms", i+1, avgSoFar)
		}
	}

	// todo: change, cause the average latency isn't measured that way. It's a test only for the future percentiles measurement
	avgLatency := totalLatency / time.Duration(t.config.TxCount)
	t.logger.Infof("Average transaction latency: %dms", avgLatency.Milliseconds())

	if t.config.FailOnHighLatency && avgLatency.Milliseconds() > t.config.ExpectedLatency {
		t.logger.Errorf("Transaction latency too high: %dms (expected <= %dms)", avgLatency.Milliseconds(), t.config.ExpectedLatency)
		t.ctx.SetResult(types.TaskResultFailure)
	} else {
		t.ctx.Outputs.SetVar("tx_count", t.config.TxCount)
		t.ctx.Outputs.SetVar("avg_latency_ms", avgLatency.Milliseconds())
	}

	// select random client, not the first
	client = executionClients[(clientIndex+1)%len(executionClients)]
	t.logger.Infof("Using second random client: %s", client.GetName())

	startTime := time.Now()
	sentTxCount := 0

	var lastTransaction *ethtypes.Transaction

	go func ()  {
		for i := 0; i < t.config.TxCount; i++ {
			// generate and sign tx
			tx, err := createDummyTransaction(nonce, chainID, privKey)
			if err != nil {
				t.logger.Errorf("Failed to create transaction: %v", err)
				t.ctx.SetResult(types.TaskResultFailure)
				return
			}

			err = client.GetRPCClient().SendTransaction(ctx, tx)

			if err != nil {
				t.logger.WithField("client", client.GetName()).Errorf("Failed to send transaction: %v", err)
				t.ctx.SetResult(types.TaskResultFailure)
				return
			}

			sentTxCount++
			nonce++

			if sentTxCount%t.config.MeasureInterval == 0 {
				elapsed := time.Since(startTime)
				t.logger.Infof("Sent %d transactions in %.2fs", sentTxCount, elapsed.Seconds())
			}

			if i == t.config.TxCount-1 {
				lastTransaction = tx
			}
		}
	}()

	t.logger.Infof("Waiting for tx confirmation for the last tx: %s", lastTransaction.Hash().Hex())

	

	// lastMeasureTime := time.Now()
	// gotTx := 0

	// for gotTx < t.config.TxCount {
	// 	select {
	// 	case msg := <-gotTxCh:
	// 		if msg.MessageID != sentryproto.MessageId_TRANSACTIONS_66 {
	// 			continue
	// 		}

	// 		gotTx += 1

	// 		if gotTx%t.config.MeasureInterval != 0 {
	// 			continue
	// 		}

	// 		t.logger.Infof("Got %d transactions", gotTx)
	// 		t.logger.Infof("Tx/s: (%d txs processed): %.2f / s \n", t.config.MeasureInterval, float64(t.config.MeasureInterval)*float64(time.Second)/float64(time.Since(lastMeasureTime)))

	// 		lastMeasureTime = time.Now()
	// 	case err := <-errCh:
	// 		t.logger.Errorf("Error receiving tx: %v", err)
	// 	}
	// }

	totalTime := time.Since(startTime)
	t.logger.Infof("Total time for %d transactions: %.2fs", sentTxCount, totalTime.Seconds())
	t.ctx.Outputs.SetVar("total_time_ms", totalTime.Milliseconds())
	t.ctx.SetResult(types.TaskResultSuccess)

	return nil
}

func createDummyTransaction(nonce uint64, chainID *big.Int, privateKey *ecdsa.PrivateKey) (*ethtypes.Transaction, error) {
	// create a dummy transaction, we don't care about the actual data
	toAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	tx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    nonce,
		To:       &toAddress,
		Value:    big.NewInt(100),
		Gas:      21000,
		GasPrice: big.NewInt(1),
		// random data + nonce to hex
		// Data: 		[]byte(fmt.Sprintf("0xdeadbeef%v", nonce)),
	})

	signer := ethtypes.LatestSignerForChainID(chainID)
	signedTx, err := ethtypes.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, err
	}

	return signedTx, nil
}

func BasicPing(remoteAddress string, logger logrus.FieldLogger) {
	te := ConnectToP2p(remoteAddress, logger)
	defer te.close()

	pingHash := te.send(&v4wire.Ping{
		Version:    4,
		From:       te.localEndpoint(),
		To:         te.remoteEndpoint(),
		Expiration: uint64(time.Now().Add(20 * time.Second).Unix()),
	})
	if err := te.checkPingPong(pingHash); err != nil {
		logger.Errorf("PingPong failed: %v", err)
	}
}

type sentryenv struct {
	endpoint   net.PacketConn
	key        *ecdsa.PrivateKey
	remote     *enode.Node
	remoteAddr *net.UDPAddr
}

const waitTime = 300 * time.Millisecond

func (te *sentryenv) localEndpoint() v4wire.Endpoint {
	addr := te.endpoint.LocalAddr().(*net.UDPAddr)
	return v4wire.Endpoint{
		IP:  addr.IP.To4(),
		UDP: uint16(addr.Port),
		TCP: 0,
	}
}

func (te *sentryenv) remoteEndpoint() v4wire.Endpoint {
	return v4wire.NewEndpoint(te.remoteAddr.AddrPort(), 0)
}

func (env *sentryenv) close() {
	env.endpoint.Close()
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
	pong := reply.(*v4wire.Pong)
	if !bytes.Equal(pong.ReplyTok, pingHash) {
		return fmt.Errorf("PONG reply token mismatch: got %x, want %x", pong.ReplyTok, pingHash)
	}
	if want := te.localEndpoint(); !want.IP.Equal(pong.To.IP) || want.UDP != pong.To.UDP {
		return fmt.Errorf("PONG 'to' endpoint mismatch: got %+v, want %+v", pong.To, want)
	}
	if v4wire.Expired(pong.Expiration) {
		return fmt.Errorf("PONG is expired (%v)", pong.Expiration)
	}
	return nil
}

func ConnectToP2p(remoteAddress string, logger logrus.FieldLogger) (*sentryenv) {
	endpoint, err := net.ListenPacket("udp", fmt.Sprintf("%v:0", remoteAddress))
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
	return &sentryenv{endpoint, key, node, addr}
}
