package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/clients/consensus/rpc"
	v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

func TestClientTracksHeadEvents(t *testing.T) {
	pool := newTestPool(t)
	pool.blockCache.specs = &ChainSpec{SlotsPerEpoch: 8}
	parent := cacheTestBlock(t, pool, 12, phase0.Root{1}, phase0.Root{})
	oldHead := cacheTestBlock(t, pool, 13, phase0.Root{2}, parent.Root)
	newHead := cacheTestBlock(t, pool, 13, phase0.Root{3}, parent.Root)
	events := make(chan *v1.HeadEvent)
	topics := make(chan []string, 1)
	payloadReady := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data any

		switch r.URL.Path {
		case "/eth/v1/node/syncing":
			data = &v1.SyncState{HeadSlot: oldHead.Slot}
		case "/eth/v1/node/version":
			data = map[string]string{"version": "Lighthouse/test"}
		case "/eth/v1/beacon/headers/head":
			data = &v1.BeaconBlockHeader{Root: oldHead.Root, Canonical: true, Header: oldHead.GetHeader()}
		case "/eth/v1/beacon/states/head/finality_checkpoints":
			data = &v1.Finality{
				PreviousJustified: &phase0.Checkpoint{},
				Justified:         &phase0.Checkpoint{},
				Finalized:         &phase0.Checkpoint{},
			}
		case "/eth/v1/events":
			requested := strings.Split(r.URL.Query().Get("topics"), ",")

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			controller := http.NewResponseController(w)
			if err := controller.Flush(); err != nil {
				t.Error(err)
				return
			}

			if slices.Contains(requested, "execution_payload_available") {
				close(payloadReady)
				<-r.Context().Done()

				return
			}

			topics <- requested

			for {
				select {
				case event := <-events:
					encoded, err := json.Marshal(event)
					if err != nil {
						t.Error(err)
						return
					}

					if _, err := fmt.Fprintf(w, "event: head\ndata: %s\n\n", encoded); err != nil {
						t.Error(err)
						return
					}

					if err := controller.Flush(); err != nil {
						t.Error(err)
						return
					}
				case <-r.Context().Done():
					return
				}
			}
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			http.NotFound(w, r)

			return
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(map[string]any{"data": data}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(cancel)

	rpcClient, err := rpc.NewBeaconClient("test", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := rpcClient.Initialize(ctx); err != nil {
		t.Fatal(err)
	}

	client := &Client{pool: pool, clientCtx: ctx, rpcClient: rpcClient, logger: pool.logger.WithField("client", "test")}
	subscription := client.SubscribeBlockEvent(4)
	done := make(chan error, 1)

	go func() { done <- client.runClientLogic() }()

	t.Cleanup(func() {
		cancel()

		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Error("client loop did not stop")
		}
	})

	select {
	case requested := <-topics:
		if !slices.Contains(requested, "head") {
			t.Fatalf("client subscribes to %v, want head events", requested)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("client did not subscribe to events")
	}

	awaitHead(t, subscription, oldHead)

	select {
	case <-payloadReady:
	case <-time.After(5 * time.Second):
		t.Fatal("client did not subscribe to payload events")
	}

	select {
	case events <- &v1.HeadEvent{Slot: newHead.Slot, Block: newHead.Root}:
	case <-time.After(5 * time.Second):
		t.Fatal("could not send head event")
	}

	awaitHead(t, subscription, newHead)

	if slot, root := client.GetLastHead(); slot != newHead.Slot || root != newHead.Root {
		t.Fatalf("head is %d [%s], want %d [%s]", slot, root, newHead.Slot, newHead.Root)
	}
}

func TestBlockEventDoesNotReplaceHead(t *testing.T) {
	pool := newTestPool(t)
	parent := cacheTestBlock(t, pool, 12, phase0.Root{1}, phase0.Root{})
	head := cacheTestBlock(t, pool, 13, phase0.Root{2}, parent.Root)
	side := cacheTestBlock(t, pool, 13, phase0.Root{3}, parent.Root)
	client := &Client{
		pool:      pool,
		logger:    pool.logger.WithField("client", "test"),
		headSlot:  head.Slot,
		headRoot:  head.Root,
		isOnline:  true,
		clientCtx: t.Context(),
	}
	peer := &Client{pool: pool, clientIdx: 1, headSlot: head.Slot, headRoot: head.Root, isOnline: true}
	pool.clients = []*Client{client, peer}

	if err := client.processBlockEvent(&v1.BlockEvent{Slot: side.Slot, Block: side.Root}); err != nil {
		t.Fatal(err)
	}

	if slot, root := client.GetLastHead(); slot != head.Slot || root != head.Root {
		t.Fatalf("block import changed head to %d [%s], want %d [%s]", slot, root, head.Slot, head.Root)
	}

	if seen := side.GetSeenBy(); len(seen) != 1 || seen[0] != client {
		t.Fatal("side block was not recorded as seen by the client")
	}

	if forks := pool.GetHeadForks(1); len(forks) != 1 {
		t.Fatalf("got %d forks after a side block import, want 1", len(forks))
	}
}

func awaitHead(t *testing.T, subscription *Subscription[*Block], want *Block) {
	t.Helper()

	select {
	case block := <-subscription.Channel():
		if block.Root != want.Root || block.Slot != want.Slot {
			t.Fatalf("published head is %d [%s], want %d [%s]", block.Slot, block.Root, want.Slot, want.Root)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("head %d [%s] was not published", want.Slot, want.Root)
	}
}

func newTestPool(t *testing.T) *Pool {
	t.Helper()

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	pool, err := NewPool(t.Context(), &PoolConfig{FollowDistance: 64}, logger)
	if err != nil {
		t.Fatal(err)
	}

	return pool
}

func cacheTestBlock(t *testing.T, pool *Pool, slot phase0.Slot, root, parent phase0.Root) *Block {
	t.Helper()

	block, _ := pool.blockCache.AddBlock(root, slot)
	if err := block.EnsureHeader(func() (*phase0.SignedBeaconBlockHeader, error) {
		return &phase0.SignedBeaconBlockHeader{Message: &phase0.BeaconBlockHeader{Slot: slot, ParentRoot: parent}}, nil
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := block.EnsureBlock(func() (*spec.VersionedSignedBeaconBlock, error) {
		return &spec.VersionedSignedBeaconBlock{Version: spec.DataVersionPhase0}, nil
	}); err != nil {
		t.Fatal(err)
	}

	return block
}
