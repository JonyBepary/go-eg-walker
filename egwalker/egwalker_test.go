package egwalker

import (
"reflect"
"testing"

"github.com/JonyBepary/go-eg-walker/causalgraph"
)

// Helper to compare LV slices (could be moved to a common test utility if needed)
func compareLVSlices(t *testing.T, got, want []causalgraph.LV) {
t.Helper()
if !reflect.DeepEqual(got, want) {
t.Errorf("LV slice mismatch: got %v, want %v", got, want)
}
}

func TestNewWalker(t *testing.T) {
walker := NewWalker[string]()
if walker == nil {
t.Fatal("NewWalker returned nil")
}
if walker.Log == nil {
t.Error("walker.Log is nil")
}
if walker.Ctx == nil {
t.Error("walker.Ctx is nil")
}
if len(walker.Log.Ops) != 0 {
t.Errorf("expected empty Log.Ops, got %d", len(walker.Log.Ops))
}
if walker.Log.CG.NextLV != 0 { // Assuming NextLV is 0 for an empty graph
t.Errorf("expected empty CausalGraph (NextLV 0), got NextLV %d", walker.Log.CG.NextLV)
}
if len(walker.Ctx.CurVersion) != 0 {
t.Errorf("expected empty Ctx.CurVersion, got %v", walker.Ctx.CurVersion)
}
if len(walker.Ctx.Items) != 0 {
t.Errorf("expected empty Ctx.Items, got %d items", len(walker.Ctx.Items))
}
}

func TestWalker_LocalInsert_Single(t *testing.T) {
walker := NewWalker[string]()
agentStr := "agentA" // Keep as string for test input clarity
agent := causalgraph.AgentID(agentStr) // Convert for CG interactions if needed by lvToRaw directly
content := "hello"
pos := 0

lv, err := walker.LocalInsert(agentStr, pos, content)
if err != nil {
t.Fatalf("LocalInsert failed: %v", err)
}

if lv != 0 {
t.Errorf("expected LV 0 for first op, got %d", lv)
}

// Check Log.Ops
if len(walker.Log.Ops) != 1 {
t.Fatalf("expected 1 op in Log.Ops, got %d", len(walker.Log.Ops))
}
op := walker.Log.Ops[0]
if op.Type != ListOpTypeInsert || op.Pos != pos || op.Content != content {
t.Errorf("op mismatch: got %+v, want Type Ins, Pos %d, Content %s", op, pos, content)
}

// Check CausalGraph
if walker.Log.CG.NextLV != 1 {
t.Errorf("expected CG.NextLV to be 1, got %d", walker.Log.CG.NextLV)
}
rawV, found := causalgraph.LVToRaw(&walker.Log.CG, lv)
if !found || rawV.Agent != agent || rawV.Seq != 0 { // Compare with AgentID typed agent
t.Errorf("LVToRaw for LV %d: found %t, rawV %+v. Expected agent %s, seq 0", lv, found, rawV, agent)
}

// Check Ctx.CurVersion
expectedVersion := []causalgraph.LV{0}
compareLVSlices(t, walker.Ctx.CurVersion, expectedVersion)

// Check Ctx.Items and Ctx.ItemsByLV (relies on placeholder applyOp)
if len(walker.Ctx.Items) != 1 {
t.Fatalf("expected 1 item in Ctx.Items, got %d", len(walker.Ctx.Items))
}
item := walker.Ctx.Items[0]
if item.OpID != lv || item.CurState != Inserted {
t.Errorf("Ctx.Items[0] mismatch: OpID %d (want %d), CurState %d (want Inserted)", item.OpID, lv, item.CurState)
}
if _, ok := walker.Ctx.ItemsByLV[lv]; !ok {
t.Errorf("item LV %d not found in Ctx.ItemsByLV", lv)
}

// Check GetActiveItems
activeItems := walker.GetActiveItems()
if len(activeItems) != 1 || activeItems[0] != content {
t.Errorf("GetActiveItems: got %v, want [%s]", activeItems, content)
}
}

func TestWalker_LocalDelete_Simple(t *testing.T) {
walker := NewWalker[string]()
agent := "agentA"

// Insert an item first
content1 := "A"
lvIns, _ := walker.LocalInsert(agent, 0, content1)

// Delete the inserted item
posDel := 0
lvDel, err := walker.LocalDelete(agent, posDel)
if err != nil {
t.Fatalf("LocalDelete failed: %v", err)
}

if lvDel != 1 { // Second operation
t.Errorf("expected LV 1 for delete op, got %d", lvDel)
}

// Check Log.Ops
if len(walker.Log.Ops) != 2 {
t.Fatalf("expected 2 ops in Log.Ops, got %d", len(walker.Log.Ops))
}
opDel := walker.Log.Ops[lvDel]
if opDel.Type != ListOpTypeDelete || opDel.Pos != posDel {
t.Errorf("delete op mismatch: got %+v, want Type Del, Pos %d", opDel, posDel)
}

// Check Ctx.CurVersion
expectedVersion := []causalgraph.LV{lvDel}
compareLVSlices(t, walker.Ctx.CurVersion, expectedVersion)

// Check Ctx.Items state (relies on placeholder applyOp for delete)
itemInCtx, ok := walker.Ctx.ItemsByLV[lvIns]
if !ok {
t.Fatalf("original inserted item LV %d not found in Ctx.ItemsByLV after delete", lvIns)
}
if itemInCtx.CurState != Deleted {
t.Errorf("item LV %d CurState: got %d, want Deleted", lvIns, itemInCtx.CurState)
}
if target, ok := walker.Ctx.DelTargets[lvDel]; !ok || target != lvIns {
t.Errorf("DelTargets for LV %d: got target %d (found %t), want %d", lvDel, target, ok, lvIns)
}

// Check GetActiveItems
activeItems := walker.GetActiveItems()
if len(activeItems) != 0 {
t.Errorf("GetActiveItems after delete: got %v, want []", activeItems)
}
}

// TODO: Add more tests:
// - Integrate remote operations (rawParents != nil)
// - More complex sequences of local inserts and deletes
// - Merge scenarios (once merge and applyOp are more complete)
// - Checkout scenarios
// - Advance/Retreat
// - Edge cases for all functions
