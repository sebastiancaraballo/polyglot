package menu

import (
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/art"
)

func TestMenuStepRestsThenSpins(t *testing.T) {
	m := newTestMenu()

	// While resting, the globe holds frame 0 for restHold ticks.
	for i := 0; i < restHold; i++ {
		m.step()
		if m.frame != 0 {
			t.Fatalf("frame left 0 after %d rest ticks, want it to hold", i+1)
		}
	}

	// The next tick begins the turn, and the globe wraps back to frame 0.
	m.step()
	if m.frame != 1 {
		t.Fatalf("frame after rest = %d, want 1", m.frame)
	}
	for i := 0; i < len(art.GlobeFrames)-1; i++ {
		m.step()
	}
	if m.frame != 0 {
		t.Fatalf("frame after a full turn = %d, want 0", m.frame)
	}
}

func TestMenuTickAdvancesAndReschedules(t *testing.T) {
	m := newTestMenu()
	m.animate = true
	m.holding = restHold // poised to advance on the next tick

	next, cmd := m.Update(tickMsg{id: m.animID})
	if got := next.(Model).frame; got != 1 {
		t.Fatalf("frame after tick = %d, want 1", got)
	}
	if cmd == nil {
		t.Fatal("a live tick should reschedule the next tick")
	}
}

func TestMenuStaleTickIgnored(t *testing.T) {
	m := newTestMenu()
	m.animate = true
	m.holding = restHold

	next, cmd := m.Update(tickMsg{id: m.animID + 1})
	if got := next.(Model).frame; got != 0 {
		t.Fatalf("stale tick advanced frame to %d, want 0", got)
	}
	if cmd != nil {
		t.Fatal("a stale tick must not reschedule, so its chain dies")
	}
}

func TestMenuInitTicksWhenAnimated(t *testing.T) {
	m := newTestMenu()
	m.animate = true
	if m.Init() == nil {
		t.Fatal("Init should start the animation when animate is true")
	}

	m.animate = false
	if m.Init() != nil {
		t.Fatal("Init should be inert when animation is disabled")
	}
}
