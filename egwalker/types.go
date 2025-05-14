package egwalker

import "github.com/JonyBepary/go-eg-walker/causalgraph"

// ListOpType defines the type of a list operation.
type ListOpType string

const (
	ListOpTypeInsert ListOpType = "ins"
	ListOpTypeDelete ListOpType = "del"
)

// ListOp represents a single operation on a list.
// The generic type T represents the type of content being inserted.
type ListOp[T any] struct {
	Type    ListOpType
	Pos     int
	Content T // Only used for insert operations.
}

// ListOpLog holds the sequence of operations and their causal relationships.
// The generic type T represents the type of content in the operations.
type ListOpLog[T any] struct {
	// Ops stores the actual operations. The LV for each op is its index in this slice.
	Ops []ListOp[T]
	// CG stores the Causal Graph for these operations.
	CG causalgraph.CausalGraph
}

// ItemState represents the state of an item during merging.
type ItemState int

const (
	// NotYetInserted means the item has not been processed or has been retreated.
	NotYetInserted ItemState = -1
	// Inserted means the item is currently considered part of the document.
	Inserted ItemState = 0
	// Deleted means the item is currently considered deleted from the document.
	// Can be > 1 if deleted concurrently multiple times.
	Deleted ItemState = 1
)

// Item represents an element in the document during state reconstruction.
// It's an internal structure used by the merging logic.
type Item struct {
	OpID causalgraph.LV // The LV of the insert operation that created this item.

	// CurState is the item's state at the current point in the merge traversal.
	CurState ItemState
	// EndState is the item's state when all operations up to the target version are merged.
	EndState ItemState

	// OriginLeft is the LV of the item to the left of this item when it was inserted.
	// -1 means the start of the document.
	OriginLeft causalgraph.LV
	// RightParent is the LV of the item to the right of this item that was
	// used as a tie-breaker for concurrent inserts at the same position.
	// -1 means no specific right parent (e.g., end of document or no conflict).
	RightParent causalgraph.LV
}

// EditContext holds the state during the traversal and application of operations.
// It's an internal structure.
type EditContext struct {
	// Items stores all known items in document order according to Fugue/YjsMod rules.
	// This list grows and items are spliced in as needed.
	Items []Item
	// DelTargets maps the LV of a delete operation to the LV of the item it deletes.
	// delTarget[delLV] = targetLV.
	DelTargets map[causalgraph.LV]causalgraph.LV // Using a map for sparse LVs
	// ItemsByLV provides quick access to items by their OpID (LV).
	ItemsByLV map[causalgraph.LV]*Item // Using a map for sparse LVs
	// CurVersion is the current version (frontier) of the EditContext.
	CurVersion []causalgraph.LV
}

// Branch represents a checked-out snapshot of the document at a specific version.
// The generic type T represents the type of content in the snapshot.
type Branch[T any] struct {
	Snapshot []T
	Version  []causalgraph.LV
}

// Walker encapsulates the state of an eg-walker instance.
// It holds the operation log and the current edit context for merging.
type Walker[T any] struct {
	Log *ListOpLog[T]
	Ctx *EditContext
	// TODO: Add other fields as needed, e.g., for caching or specific algorithms.
}
