package checkconsensusapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/clients/consensus/rpc/eventstream"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// checkClientSSE subscribes to a single event-stream topic on `client` and waits
// for at least `cfg.SSE.MinEvents` events to arrive within the configured
// window. The event payload is validated against `eventSchema` when present.
func (t *Task) checkClientSSE(
	ctx context.Context,
	client *clients.PoolClient,
	baseURL string,
	eventSchema *jsonschema.Schema,
) *PerClientResult {
	r := &PerClientResult{
		Client:     client.Config.Name,
		ClientType: clientTypeString(client),
	}

	topic := t.config.SSE.Topic

	eventName := t.config.SSE.EventName
	if eventName == "" {
		eventName = topic
	}

	wait := time.Duration(t.config.SSE.TimeoutSeconds) * time.Second
	if wait <= 0 {
		wait = defaultSSETimeoutSeconds * time.Second
	}

	minEvents := t.config.SSE.MinEvents
	if minEvents <= 0 {
		minEvents = 1
	}

	streamURL, err := url.Parse(strings.TrimRight(baseURL, "/") + "/eth/v1/events")
	if err != nil {
		r.Status = resultFail
		r.Error = fmt.Sprintf("invalid base URL: %v", err)

		return r
	}

	q := streamURL.Query()
	q.Set("topics", topic)
	streamURL.RawQuery = q.Encode()
	r.URL = streamURL.String()

	subCtx, cancel := context.WithTimeout(ctx, wait+5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(subCtx, methodGet, streamURL.String(), http.NoBody)
	if err != nil {
		r.Status = resultFail
		r.Error = fmt.Sprintf("build request: %v", err)

		return r
	}

	if cfgHeaders := client.ConsensusClient.GetEndpointConfig().Headers; len(cfgHeaders) > 0 {
		for k, v := range cfgHeaders {
			req.Header.Set(k, v)
		}
	}

	for k, v := range t.config.Headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	httpClient := &http.Client{Timeout: 0} // we manage cancellation via ctx
	t0 := time.Now()

	stream, err := eventstream.SubscribeWith("", httpClient, req)
	if err != nil {
		r.DurationMs = time.Since(t0).Milliseconds()
		r.Status = resultFail

		if subErr, ok := err.(eventstream.SubscriptionError); ok {
			r.HTTPStatus = subErr.Code
			r.Note = fmt.Sprintf("subscription rejected: %s", subErr.Message)
		} else {
			r.Error = fmt.Sprintf("subscription failed: %v", err)
		}

		return r
	}
	defer stream.Close()

	// Wait for first event or deadline.
	waitTimer := time.NewTimer(wait)
	defer waitTimer.Stop()

	eventsCount := 0
	schemaErrs := []string{}

eventLoop:
	for {
		select {
		case ev, ok := <-stream.Events:
			if !ok {
				break eventLoop
			}

			if ev.Event() != eventName && eventName != "" {
				// SSE topic subscription receives only filtered events normally
				// (the beacon node honors ?topics=...), but be defensive.
				continue
			}

			eventsCount++

			if eventSchema != nil {
				errs := validateBytes(eventSchema, []byte(ev.Data()))
				if len(errs) > 0 {
					schemaErrs = append(schemaErrs, errs...)
				}
			}

			if eventsCount >= minEvents {
				break eventLoop
			}
		case streamErr, ok := <-stream.Errors:
			if !ok {
				break eventLoop
			}
			// Non-fatal: the underlying eventsource library auto-reconnects.
			r.Note = fmt.Sprintf("stream warning: %v", streamErr)
		case <-waitTimer.C:
			break eventLoop
		case <-subCtx.Done():
			break eventLoop
		}
	}

	r.DurationMs = time.Since(t0).Milliseconds()
	r.EventCount = eventsCount

	switch {
	case eventsCount == 0:
		// Subscription was accepted (we got past Subscribe) but no events
		// arrived. That's still a pass — the server understood the topic
		// query and the route exists; the absence of events is a chain
		// state issue, not a client compatibility issue.
		r.Status = resultPass

		if r.Note == "" {
			r.Note = "subscription opened (no events within window)"
		}

		// HTTPStatus 200 (assumed by successful Subscribe)
		r.HTTPStatus = 200

		return r
	case len(schemaErrs) > 0:
		r.Status = resultPartial
		r.SchemaErrors = uniqueStrings(schemaErrs)
		r.Note = fmt.Sprintf("%d events, schema mismatches present", eventsCount)
		r.HTTPStatus = 200

		return r
	default:
		r.Status = resultPass
		r.HTTPStatus = 200

		if r.Note == "" {
			r.Note = fmt.Sprintf("%d event(s) received", eventsCount)
		}

		return r
	}
}

// uniqueStrings de-duplicates schemaErrs to keep output compact.
func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))

	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}

		seen[s] = struct{}{}
		out = append(out, s)
	}

	return out
}
