package controls

import (
	"strconv"
	"testing"
)

// A host re-delivering a large folder's children in place (SetChildren with
// the nodes it already holds) runs dropExcept on the UI goroutine. It used to
// scan the new children once per old child: 400M comparisons for 20k nodes.
func BenchmarkTreeModelSetChildrenRedeliver20k(b *testing.B) {
	m := newTMTree().m
	root := &tmNode{name: "r"}
	m.AddRoot(root)
	kids := make([]*tmNode, 20_000)
	for i := range kids {
		kids[i] = &tmNode{name: strconv.Itoa(i)}
	}
	m.SetChildren(root, kids)
	for b.Loop() {
		m.SetChildren(root, kids)
	}
}
