package core

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

type fakeOverlay struct{ open bool }

func (*fakeOverlay) DrawOverlay(tcell.Screen) {}
func (f *fakeOverlay) OverlayActive() bool    { return f.open }

func TestOpenOverlay(t *testing.T) {
	a, b, c := &fakeOverlay{}, &fakeOverlay{open: true}, &fakeOverlay{open: true}

	if got, ok := OpenOverlay(a, b, c); !ok || got != b {
		t.Errorf("OpenOverlay(a, b, c) = %p, %v; want b (%p), true", got, ok, b)
	}
	if got, ok := OpenOverlay(a); ok || got != nil {
		t.Errorf("OpenOverlay(closed) = %p, %v; want nil, false", got, ok)
	}
	if got, ok := OpenOverlay[*fakeOverlay](); ok || got != nil {
		t.Errorf("OpenOverlay() = %p, %v; want nil, false", got, ok)
	}
	// An interface-typed list answers through the interface.
	if got, ok := OpenOverlay[Overlay](a, c); !ok || got != Overlay(c) {
		t.Errorf("OpenOverlay[Overlay](a, c) = %v, %v; want c, true", got, ok)
	}
}
