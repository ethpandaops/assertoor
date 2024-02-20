package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/donovanhide/eventsource"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus/rpc/eventstream"
)

const (
	StreamBlockEvent     uint16 = 0x01
	StreamHeadEvent      uint16 = 0x02
	StreamFinalizedEvent uint16 = 0x04
)

type BeaconStreamEvent struct {
	Event uint16
	Data  interface{}
}

type BeaconStream struct {
	ctx          context.Context
	ctxCancel    context.CancelFunc
	running      bool
	events       uint16
	client       *BeaconClient
	ReadyChan    chan bool
	EventChan    chan *BeaconStreamEvent
	lastHeadSeen time.Time
}

func (bc *BeaconClient) NewBlockStream(ctx context.Context, events uint16) *BeaconStream {
	streamCtx, ctxCancel := context.WithCancel(ctx)

	blockStream := BeaconStream{
		ctx:       streamCtx,
		ctxCancel: ctxCancel,
		running:   true,
		events:    events,
		client:    bc,
		ReadyChan: make(chan bool, 10),
		EventChan: make(chan *BeaconStreamEvent, 10),
	}
	go blockStream.startStream()

	return &blockStream
}

func (bs *BeaconStream) Close() {
	bs.ctxCancel()
}

func (bs *BeaconStream) startStream() {
	defer func() {
		bs.running = false
	}()

	stream := bs.subscribeStream(bs.client.endpoint, bs.events)
	if stream != nil {
		defer stream.Close()

		for {
			select {
			case <-bs.ctx.Done():
				return
			case evt := <-stream.Events:
				switch evt.Event() {
				case "block":
					bs.processBlockEvent(evt)
				case "head":
					bs.processHeadEvent(evt)
				case "finalized_checkpoint":
					bs.processFinalizedEvent(evt)
				}
			case <-stream.Ready:
				bs.ReadyChan <- true
			case err := <-stream.Errors:
				logger.WithField("client", bs.client.name).Warnf("beacon block stream error: %v", err)
				select {
				case bs.ReadyChan <- false:
				case <-bs.ctx.Done():
				}
			}
		}
	}
}

func (bs *BeaconStream) subscribeStream(endpoint string, events uint16) *eventstream.Stream {
	var topics strings.Builder

	topicsCount := 0

	if events&StreamBlockEvent > 0 {
		if topicsCount > 0 {
			fmt.Fprintf(&topics, ",")
		}

		fmt.Fprintf(&topics, "block")

		topicsCount++
	}

	if events&StreamHeadEvent > 0 {
		if topicsCount > 0 {
			fmt.Fprintf(&topics, ",")
		}

		fmt.Fprintf(&topics, "head")

		topicsCount++
	}

	if events&StreamFinalizedEvent > 0 {
		if topicsCount > 0 {
			fmt.Fprintf(&topics, ",")
		}

		fmt.Fprintf(&topics, "finalized_checkpoint")

		topicsCount++
	}

	if topicsCount == 0 {
		return nil
	}

	for {
		var stream *eventstream.Stream

		streamURL := fmt.Sprintf("%s/eth/v1/events?topics=%v", endpoint, topics.String())
		req, err := http.NewRequestWithContext(bs.ctx, "GET", streamURL, http.NoBody)

		if err == nil {
			for headerKey, headerVal := range bs.client.headers {
				req.Header.Set(headerKey, headerVal)
			}

			stream, err = eventstream.SubscribeWithRequest("", req)
		}

		if err != nil {
			logger.WithField("client", bs.client.name).Warnf("Error while subscribing beacon event stream %v: %v", getRedactedURL(streamURL), err)
			select {
			case <-bs.ctx.Done():
				return nil
			case <-time.After(10 * time.Second):
			}
		} else {
			return stream
		}
	}
}

func (bs *BeaconStream) processBlockEvent(evt eventsource.Event) {
	var parsed v1.BlockEvent

	err := json.Unmarshal([]byte(evt.Data()), &parsed)

	if err != nil {
		logger.WithField("client", bs.client.name).Warnf("beacon block stream failed to decode block event: %v", err)
		return
	}
	bs.EventChan <- &BeaconStreamEvent{
		Event: StreamBlockEvent,
		Data:  &parsed,
	}
}

func (bs *BeaconStream) processHeadEvent(evt eventsource.Event) {
	var parsed v1.HeadEvent

	err := json.Unmarshal([]byte(evt.Data()), &parsed)
	if err != nil {
		logger.WithField("client", bs.client.name).Warnf("beacon block stream failed to decode block event: %v", err)
		return
	}

	bs.lastHeadSeen = time.Now()
	bs.EventChan <- &BeaconStreamEvent{
		Event: StreamHeadEvent,
		Data:  &parsed,
	}
}

func (bs *BeaconStream) processFinalizedEvent(evt eventsource.Event) {
	var parsed v1.FinalizedCheckpointEvent

	err := json.Unmarshal([]byte(evt.Data()), &parsed)
	if err != nil {
		logger.WithField("client", bs.client.name).Warnf("beacon block stream failed to decode finalized_checkpoint event: %v", err)
		return
	}

	bs.EventChan <- &BeaconStreamEvent{
		Event: StreamFinalizedEvent,
		Data:  &parsed,
	}
}

func getRedactedURL(requrl string) string {
	var logurl string

	urlData, _ := url.Parse(requrl)
	if urlData != nil {
		logurl = urlData.Redacted()
	} else {
		logurl = requrl
	}

	return logurl
}
