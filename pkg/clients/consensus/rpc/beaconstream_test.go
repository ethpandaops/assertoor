package rpc

import (
	"testing"

	v1 "github.com/ethpandaops/go-eth2-client/api/v1"
)

type testStreamEvent struct {
	event string
	data  string
}

func (e testStreamEvent) Id() string    { return "" } //nolint:revive // name required by eventsource.Event
func (e testStreamEvent) Event() string { return e.event }
func (e testStreamEvent) Data() string  { return e.data }

func TestProcessExecutionPayloadEvent(t *testing.T) {
	bs := &BeaconStream{EventChan: make(chan *BeaconStreamEvent, 1)}

	bs.processExecutionPayloadEvent(testStreamEvent{
		event: "execution_payload",
		data:  `{"slot":"10","builder_index":"42","block_hash":"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef","block_root":"0x9a2fefd2fdb57f74993c7780ea5b9030d2897b615b89f808011ca5aebed54eaf","execution_optimistic":false}`,
	})

	select {
	case evt := <-bs.EventChan:
		if evt.Event != StreamExecutionPayloadEvent {
			t.Fatalf("unexpected event type %v", evt.Event)
		}

		payloadEvent, ok := evt.Data.(*v1.ExecutionPayloadEvent)
		if !ok {
			t.Fatalf("unexpected event data %T", evt.Data)
		}

		if payloadEvent.Slot != 10 {
			t.Fatalf("unexpected slot %v", payloadEvent.Slot)
		}

		if payloadEvent.BlockRoot.String() != "0x9a2fefd2fdb57f74993c7780ea5b9030d2897b615b89f808011ca5aebed54eaf" {
			t.Fatalf("unexpected block root %v", payloadEvent.BlockRoot)
		}
	default:
		t.Fatal("no event forwarded")
	}
}
