package egwalker

import (
	"fmt"
	"github.com/JonyBepary/go-eg-walker/causalgraph"
)

// newEditCtx creates and returns a new EditContext, initialized to an empty state.
func newEditCtx() *EditContext {
	return &EditContext{
		Items:      []Item{},
		DelTargets: make(map[causalgraph.LV]causalgraph.LV),
		ItemsByLV:  make(map[causalgraph.LV]*Item),
		CurVersion: []causalgraph.LV{}, // Starts at "root" or empty version
	}
}

// NewWalker creates a new Walker instance.
// It initializes an empty ListOpLog and EditContext.
func NewWalker[T any]() *Walker[T] {
	opLog := &ListOpLog[T]{
		Ops: []ListOp[T]{},
		CG:  *causalgraph.CreateCG(), // Dereference to store CausalGraph value
	}
	return &Walker[T]{
		Log: opLog,
		Ctx: newEditCtx(),
	}
}

// Integrate incorporates a new operation into the walker's log and context.
// op: The operation to integrate.
// agent: The agent ID creating this operation.
// rawParents: The RawVersion parents of this operation. If nil, current Ctx.CurVersion is used.
// Returns the LV of the integrated operation and an error if any.
func (w *Walker[T]) Integrate(op ListOp[T], agent string, rawParents []causalgraph.RawVersion) (causalgraph.LV, error) {
	// Tentative LV was previously len(w.Log.Ops), but actual LV comes from CG.
	cgAgentID := causalgraph.AgentID(agent)
	seq := causalgraph.NextSeqForAgent(&w.Log.CG, cgAgentID)
	w.Log.Ops = append(w.Log.Ops, op) // Op is appended, its index will be len(w.Log.Ops)-1

	var cgParents []causalgraph.RawVersion
	if rawParents != nil {
		cgParents = rawParents
	} else {
		var err error
		cgParents, err = causalgraph.LVToRawList(&w.Log.CG, w.Ctx.CurVersion)
		if err != nil {
			w.Log.Ops = w.Log.Ops[:len(w.Log.Ops)-1] // Rollback op append
			return -1, fmt.Errorf("failed to convert current version to raw parents: %w", err)
		}
	}

	id := causalgraph.RawVersion{Agent: cgAgentID, Seq: seq}
	cgEntry, err := causalgraph.AddRaw(&w.Log.CG, id, 1, cgParents)
	if err != nil {
		w.Log.Ops = w.Log.Ops[:len(w.Log.Ops)-1] // Rollback op append
		return -1, fmt.Errorf("failed to add to causal graph: %w", err)
	}
	if cgEntry == nil {
		w.Log.Ops = w.Log.Ops[:len(w.Log.Ops)-1] // Rollback op append
		return -1, fmt.Errorf("operation (%s, %d) already exists in causal graph or failed to add", agent, seq)
	}

	actualLV := cgEntry.Version
	// The op log index for this operation is len(w.Log.Ops)-1.
	// We need to ensure that applyOp and retreatOp can find the op if actualLV is not the same as this index.
	// For now, assume they are aligned or that applyOp/retreatOp use a mapping if necessary.
	// The current applyOp/retreatOp use LV as an index into w.Log.Ops. This implies actualLV must be len(w.Log.Ops)-1.
	// This holds if LVs are assigned sequentially starting from 0 and incrementing by 1 for each op.
	// causalgraph.AddRaw uses cg.NextLV, which is total ops length. So this should align.

	if rawParents == nil { // Implies local op, advance context version and apply op
		w.Ctx.CurVersion = []causalgraph.LV{actualLV}
		if errApply := w.applyOp(actualLV); errApply != nil {
			return actualLV, fmt.Errorf("op integrated (LV %d) but failed to apply to context: %w", actualLV, errApply)
		}
	}
	return actualLV, nil
}

// applyOp applies a single operation (specified by its LV) to the EditContext.
// It modifies Ctx.Items and Ctx.DelTargets. This is an internal method.
func (w *Walker[T]) applyOp(lv causalgraph.LV) error {
	// Assuming LV is the index in w.Log.Ops for this operation.
	opIndex := int(lv)
	if opIndex < 0 || opIndex >= len(w.Log.Ops) {
		return fmt.Errorf("applyOp: LV %d (index %d) is out of bounds for op log of length %d", lv, opIndex, len(w.Log.Ops))
	}
	op := w.Log.Ops[opIndex]

	switch op.Type {
	case ListOpTypeInsert:
		newItem := Item{
			OpID:        lv,
			CurState:    Inserted,
			EndState:    NotYetInserted,
			OriginLeft:  -1,
			RightParent: -1,
		}
		// The item is first created, then a pointer to it is stored in ItemsByLV.
		// Then, this item (a copy of it) is inserted into the Items slice.
		// To ensure ItemsByLV points to the item *in the slice*, we update it after insertion.

		insertAtIndex := op.Pos
		if insertAtIndex > len(w.Ctx.Items) {
			insertAtIndex = len(w.Ctx.Items)
		}
		// Insert a temporary copy, then get a pointer to the actual item in the slice.
		w.Ctx.Items = append(w.Ctx.Items[:insertAtIndex], append([]Item{newItem}, w.Ctx.Items[insertAtIndex:]...)...)
		w.Ctx.ItemsByLV[lv] = &w.Ctx.Items[insertAtIndex] // Point to the newly inserted item in the slice.

	case ListOpTypeDelete:
		targetLV := causalgraph.LV(-1)
		visibleCount := 0
		foundTarget := false
		for i := range w.Ctx.Items {
			item := &w.Ctx.Items[i]
			if item.CurState == Inserted {
				if visibleCount == op.Pos {
					targetLV = item.OpID
					item.CurState = Deleted // Update the item in the slice
					if itemInMap, ok := w.Ctx.ItemsByLV[targetLV]; ok {
						itemInMap.CurState = Deleted // Also update the map entry if it's distinct
					}
					w.Ctx.DelTargets[lv] = targetLV
					foundTarget = true
					break
				}
				visibleCount++
			}
		}
		if !foundTarget {
			w.Ctx.DelTargets[lv] = causalgraph.LV(-1) // No-op
		}
	}
	return nil
}

// retreatOp un-applies a single operation (specified by its LV) from the EditContext.
func (w *Walker[T]) retreatOp(lv causalgraph.LV) error {
	opIndex := int(lv)
	if opIndex < 0 || opIndex >= len(w.Log.Ops) {
		return fmt.Errorf("retreatOp: LV %d (index %d) is out of bounds for op log of length %d", lv, opIndex, len(w.Log.Ops))
	}
	op := w.Log.Ops[opIndex]
	itemToModifyInMap, itemExistsInMap := w.Ctx.ItemsByLV[lv]

	switch op.Type {
	case ListOpTypeInsert:
		if !itemExistsInMap {
			return fmt.Errorf("retreatOp: item for insert LV %d not found in ItemsByLV", lv)
		}
		itemToModifyInMap.CurState = NotYetInserted
		foundInSlice := false
		for i := range w.Ctx.Items {
			if w.Ctx.Items[i].OpID == lv {
				w.Ctx.Items[i].CurState = NotYetInserted
				foundInSlice = true
				break
			}
		}
		if !foundInSlice {
			return fmt.Errorf("retreatOp: item for insert LV %d found in ItemsByLV but not in Items slice", lv)
		}
	case ListOpTypeDelete:
		targetLV, wasDeleteRecorded := w.Ctx.DelTargets[lv]
		if !wasDeleteRecorded || targetLV == causalgraph.LV(-1) {
			return nil
		}
		targetItemInMap, targetExistsInMap := w.Ctx.ItemsByLV[targetLV]
		if !targetExistsInMap {
			return fmt.Errorf("retreatOp: target item LV %d for delete op LV %d not found in ItemsByLV", targetLV, lv)
		}
		targetItemInMap.CurState = Inserted
		foundInSlice := false
		for i := range w.Ctx.Items {
			if w.Ctx.Items[i].OpID == targetLV {
				w.Ctx.Items[i].CurState = Inserted
				foundInSlice = true
				break
			}
		}
		if !foundInSlice {
			return fmt.Errorf("retreatOp: target item LV %d for delete op LV %d found in ItemsByLV but not in Items slice", targetLV, lv)
		}
		delete(w.Ctx.DelTargets, lv)
	}
	return nil
}

// advance moves the EditContext forward from its current version to targetLV.
func (w *Walker[T]) advance(targetLV causalgraph.LV) error {
	opsToApply := []causalgraph.LV{}
	err := causalgraph.IterVersionsBetween(&w.Log.CG, w.Ctx.CurVersion, targetLV,
		func(v causalgraph.LV, isParentOfPrev bool, isMerge bool) (stop bool, err error) {
			opsToApply = append(opsToApply, v)
			return false, nil
		})
	if err != nil {
		return fmt.Errorf("advance: error iterating: %w", err)
	}
	for i := len(opsToApply) - 1; i >= 0; i-- {
		if err := w.applyOp(opsToApply[i]); err != nil {
			return fmt.Errorf("advance: failed to apply op LV %d: %w", opsToApply[i], err)
		}
	}
	w.Ctx.CurVersion = []causalgraph.LV{targetLV}
	return nil
}

// retreat moves the EditContext backward from its current version to targetLV.
func (w *Walker[T]) retreat(targetLV causalgraph.LV) error {
	isAncestor, err := causalgraph.VersionContainsLV(&w.Log.CG, w.Ctx.CurVersion, targetLV)
	if err != nil {
		return fmt.Errorf("retreat: error checking ancestry for targetLV %d: %w", targetLV, err)
	}
	// If CurVersion is empty, targetLV must be a root-like LV (-1 or 0) or it's an invalid retreat.
	if len(w.Ctx.CurVersion) == 0 {
		if targetLV == causalgraph.LV(-1) || (targetLV == 0 && causalgraph.NextLV(&w.Log.CG) > 0) { // Allow retreat to 0 if graph is not empty
			w.Ctx.CurVersion = []causalgraph.LV{targetLV}
			return nil
		} else if targetLV == 0 && causalgraph.NextLV(&w.Log.CG) == 0 { // Retreat to 0 on empty graph
			w.Ctx.CurVersion = []causalgraph.LV{} // Or []LV{0} depending on convention for root of empty graph
			return nil
		}
		return fmt.Errorf("retreat: cannot retreat from empty version to non-root LV %d", targetLV)
	}

	if !isAncestor && (len(w.Ctx.CurVersion) > 1 || w.Ctx.CurVersion[0] != targetLV) {
		return fmt.Errorf("retreat: targetLV %d is not an ancestor of current version %v", targetLV, w.Ctx.CurVersion)
	}

	if len(w.Ctx.CurVersion) != 1 {
		if targetLV == causalgraph.LV(-1) && len(w.Ctx.CurVersion) > 1 {
			*w.Ctx = *newEditCtx()
			w.Ctx.CurVersion = []causalgraph.LV{targetLV} // Represent root as [-1]
			return nil
		}
		// Fallthrough for more complex frontier retreats, currently simplified.
	}

	currentTip := w.Ctx.CurVersion[0] // Simplification: use first head

	if currentTip == targetLV {
		w.Ctx.CurVersion = []causalgraph.LV{targetLV} // Ensure it's set correctly even if no ops change
		return nil
	}

	err = causalgraph.IterVersionsBetween(&w.Log.CG, []causalgraph.LV{targetLV}, currentTip,
		func(v causalgraph.LV, isParentOfPrev bool, isMerge bool) (stop bool, err error) {
			if errRetreat := w.retreatOp(v); errRetreat != nil {
				return true, fmt.Errorf("retreat: failed to retreat op LV %d: %w", v, errRetreat)
			}
			return false, nil
		})
	if err != nil {
		return fmt.Errorf("retreat: error during iteration: %w", err)
	}
	w.Ctx.CurVersion = []causalgraph.LV{targetLV}
	return nil
}

// merge updates the EditContext to reflect the state at targetVersion.
func (w *Walker[T]) merge(targetVersion []causalgraph.LV) error {
	if len(targetVersion) == 0 {
		if len(w.Ctx.CurVersion) > 0 {
			*w.Ctx = *newEditCtx()
			w.Ctx.CurVersion = []causalgraph.LV{}
		}
		return nil
	}

	allVersions := make([]causalgraph.LV, 0, len(w.Ctx.CurVersion)+len(targetVersion))
	allVersions = append(allVersions, w.Ctx.CurVersion...)
	allVersions = append(allVersions, targetVersion...)

	commonAncestors, err := causalgraph.FindDominators(&w.Log.CG, allVersions)
	if err != nil {
		return fmt.Errorf("merge: failed to find common ancestors: %w", err)
	}

	if len(w.Ctx.CurVersion) > 0 {
		if len(commonAncestors) == 0 {
			*w.Ctx = *newEditCtx()
			w.Ctx.CurVersion = []causalgraph.LV{}
		} else {
			// Simplified retreat to the first common ancestor.
			if errRetreat := w.retreat(commonAncestors[0]); errRetreat != nil {
				fmt.Printf("Warning: merge retreat failed: %v. Resetting context and advancing from root to common ancestors.\n", errRetreat)
				*w.Ctx = *newEditCtx()
				// Advance from root to each common ancestor
				for _, caLV := range commonAncestors { // Use a different variable name here
					// Need to advance one by one if commonAncestors itself is a frontier.
					// This simplified advance assumes advancing to a single point, then another.
					// A proper merge would advance along multiple paths if commonAncestors is a frontier.
					// For now, let's assume advance can handle building up to a frontier.
					// This part needs careful thought for frontier-to-frontier state changes.
					// The current advance sets CurVersion to a single LV.
					// So, we effectively advance to commonAncestors[0], then commonAncestors[1], etc.
					// This is not a true "advance to frontier" but rather serial advance.
					if errAdvance := w.advance(caLV); errAdvance != nil { // This will set CurVersion to [caLV]
						return fmt.Errorf("merge: fallback advance to common ancestor %d failed: %w", caLV, errAdvance)
					}
				}
				w.Ctx.CurVersion = commonAncestors // After all advances, set CurVersion to the full frontier
			} else {
				w.Ctx.CurVersion = commonAncestors
			}
		}
	} else {
		w.Ctx.CurVersion = commonAncestors
	}

	for _, tv := range targetVersion {
		isCovered, err := causalgraph.VersionContainsLV(&w.Log.CG, w.Ctx.CurVersion, tv)
		if err != nil {
			return fmt.Errorf("merge: error checking coverage for target head %d: %w", tv, err)
		}
		if isCovered {
			continue
		}

		opsToApply := []causalgraph.LV{}
		iterErr := causalgraph.IterVersionsBetween(&w.Log.CG, w.Ctx.CurVersion, tv,
			func(v causalgraph.LV, isParentOfPrev bool, isMerge bool) (stop bool, err error) {
				opsToApply = append(opsToApply, v)
				return false, nil
			})
		if iterErr != nil {
			return fmt.Errorf("merge: error iterating from %v to target head %d: %w", w.Ctx.CurVersion, tv, iterErr)
		}
		for i := len(opsToApply) - 1; i >= 0; i-- {
			opToApplyLV := opsToApply[i]
			if errApply := w.applyOp(opToApplyLV); errApply != nil {
				return fmt.Errorf("merge: failed to apply op LV %d: %w", opToApplyLV, errApply)
			}
		}
		// After applying ops towards tv, the context's current version effectively includes tv.
		// We need to manage w.Ctx.CurVersion carefully if it's a frontier.
		// A simple approach: after processing one tv, update CurVersion to include it,
		// then subsequent tv's are diffed against this new CurVersion.
		// This is complex. A more robust way is to collect all diffs from commonAncestors to each targetVersion head,
		// then apply the union of these diffs.
		// For now, the loop implies serial application towards each target head.
		// The final w.Ctx.CurVersion = targetVersion will set it correctly.
	}
	w.Ctx.CurVersion = targetVersion
	return nil
}

// Checkout computes and returns the document snapshot at a given targetVersion.
func (w *Walker[T]) Checkout(targetVersion []causalgraph.LV) (*Branch[T], error) {
	// Operate on a temporary walker with a fresh context to avoid mutating the main walker's state.
	tempWalker := &Walker[T]{
		Log: w.Log,        // Share the log (read-only for this operation)
		Ctx: newEditCtx(), // Start with a fresh context
	}

	err := tempWalker.merge(targetVersion)
	if err != nil {
		return nil, fmt.Errorf("checkout: failed to merge to targetVersion %v on temp context: %w", targetVersion, err)
	}

	snapshot := make([]T, 0)
	for _, item := range tempWalker.Ctx.Items {
		if item.CurState == Inserted {
			opIndex := int(item.OpID)
			if opIndex < len(tempWalker.Log.Ops) && tempWalker.Log.Ops[opIndex].Type == ListOpTypeInsert {
				snapshot = append(snapshot, tempWalker.Log.Ops[opIndex].Content)
			} else {
				return nil, fmt.Errorf("checkout: inconsistent item state for OpID %d in temp context (op log len %d)", item.OpID, len(tempWalker.Log.Ops))
			}
		}
	}

	return &Branch[T]{
		Snapshot: snapshot,
		Version:  targetVersion,
	}, nil
}

// LocalInsert creates a new local insert operation and integrates it.
func (w *Walker[T]) LocalInsert(agent string, pos int, content T) (causalgraph.LV, error) {
	op := ListOp[T]{
		Type:    ListOpTypeInsert,
		Pos:     pos,
		Content: content,
	}
	lv, err := w.Integrate(op, agent, nil)
	if err != nil {
		return -1, fmt.Errorf("localInsert: failed to integrate op: %w", err)
	}
	return lv, nil
}

// LocalDelete creates a new local delete operation and integrates it.
func (w *Walker[T]) LocalDelete(agent string, pos int) (causalgraph.LV, error) {
	op := ListOp[T]{
		Type: ListOpTypeDelete,
		Pos:  pos,
	}
	lv, err := w.Integrate(op, agent, nil)
	if err != nil {
		return -1, fmt.Errorf("localDelete: failed to integrate op: %w", err)
	}
	return lv, nil
}

// GetVersion returns the current version (frontier) of the walker's EditContext.
func (w *Walker[T]) GetVersion() []causalgraph.LV {
	v := make([]causalgraph.LV, len(w.Ctx.CurVersion))
	copy(v, w.Ctx.CurVersion)
	return v
}

// GetOps returns all operations in the log.
func (w *Walker[T]) GetOps() []ListOp[T] {
	ops := make([]ListOp[T], len(w.Log.Ops))
	copy(ops, w.Log.Ops)
	return ops
}

// GetCG returns a pointer to the causal graph.
func (w *Walker[T]) GetCG() *causalgraph.CausalGraph {
	return &w.Log.CG
}

// GetActiveItems returns the current snapshot of content based on Ctx.Items.
// This reflects the state at w.Ctx.CurVersion.
func (w *Walker[T]) GetActiveItems() []T {
	snapshot := make([]T, 0)
	for _, item := range w.Ctx.Items {
		if item.CurState == Inserted {
			opIndex := int(item.OpID)
			if opIndex < len(w.Log.Ops) && w.Log.Ops[opIndex].Type == ListOpTypeInsert {
				snapshot = append(snapshot, w.Log.Ops[opIndex].Content)
			} else {
				fmt.Printf("Warning: GetActiveItems found an item with OpID %d marked Inserted but Op is missing or not an Insert (op log len %d)\n", item.OpID, len(w.Log.Ops))
			}
		}
	}
	return snapshot
}

// TODO: Port utility functions opToPretty, opToCompact
