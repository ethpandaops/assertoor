package consensus

import "testing"

// TestDispatcherUnsubscribeRemovesSubscription verifies that Unsubscribe actually
// removes the subscription from the dispatcher. A regression here means every
// subscription leaks and Fire keeps delivering to and iterating over dead entries.
func TestDispatcherUnsubscribeRemovesSubscription(t *testing.T) {
	d := &Dispatcher[int]{}

	subs := make([]*Subscription[int], 5)
	for i := range subs {
		subs[i] = d.Subscribe(1)
	}

	if got := len(d.subscriptions); got != 5 {
		t.Fatalf("after subscribing 5: got %d subscriptions, want 5", got)
	}

	// remove two, including one that is not the last element
	subs[1].Unsubscribe()
	subs[3].Unsubscribe()

	if got := len(d.subscriptions); got != 3 {
		t.Fatalf("after unsubscribing 2: got %d subscriptions, want 3", got)
	}

	// unsubscribing again must be a safe no-op
	subs[1].Unsubscribe()

	if got := len(d.subscriptions); got != 3 {
		t.Fatalf("after double unsubscribe: got %d subscriptions, want 3", got)
	}

	// Fire must only reach the still-subscribed channels
	d.Fire(42)

	for i, s := range subs {
		want := 0
		if i == 0 || i == 2 || i == 4 {
			want = 1
		}

		if got := len(s.channel); got != want {
			t.Fatalf("subscription %d: got %d buffered events, want %d", i, got, want)
		}
	}

	subs[0].Unsubscribe()
	subs[2].Unsubscribe()
	subs[4].Unsubscribe()

	if got := len(d.subscriptions); got != 0 {
		t.Fatalf("after unsubscribing all: got %d subscriptions, want 0", got)
	}
}
