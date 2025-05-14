package causalgraph

import (
"fmt"
"sort"
)

// CreateCG creates and returns a new, empty CausalGraph.
func CreateCG() *CausalGraph {
return &CausalGraph{
AgentToVersion: make(map[AgentID][]ClientEntry),
// Entries and Heads are initialized as empty slices by default.
// NextLV starts at 0.
}
}

// NextLV returns the next available local version (LV) in the graph.
// It's equivalent to the total number of versions assigned so far.
func NextLV(cg *CausalGraph) LV {
return cg.NextLV
}

// NextSeqForAgent returns the next sequence number for a given agent.
// If the agent is new, it returns 0.
func NextSeqForAgent(cg *CausalGraph, agent AgentID) int {
if entries, ok := cg.AgentToVersion[agent]; ok && len(entries) > 0 {
lastEntry := entries[len(entries)-1]
return lastEntry.SeqEnd // SeqEnd is exclusive, so it's the next seq
}
return 0 // First sequence number for this agent
}

// findEntryContainingRaw finds the CGEntry that contains the given RawVersion (agent, seq).
// It returns the entry, the offset of the RawVersion within that entry's sequence range, and a boolean indicating if found.
func findEntryContainingRaw(cg *CausalGraph, agent AgentID, seq int) (*CGEntry, int, bool) {
clientEntries, ok := cg.AgentToVersion[agent]
if !ok {
return nil, -1, false
}

idx := sort.Search(len(clientEntries), func(i int) bool {
return clientEntries[i].SeqEnd > seq
})

if idx < len(clientEntries) && clientEntries[idx].Seq <= seq {
entryLV := clientEntries[idx].Version
// Find the actual CGEntry in cg.Entries
for i := range cg.Entries {
if cg.Entries[i].Version == entryLV {
offset := seq - cg.Entries[i].Seq
// Check if seq is within the span of this specific CGEntry
if seq >= cg.Entries[i].Seq && seq < (cg.Entries[i].Seq+int(cg.Entries[i].VEnd-cg.Entries[i].Version)) {
return &cg.Entries[i], offset, true
}
}
}
}
return nil, -1, false
}

// findEntryContaining finds the CGEntry that contains the given LV.
// It returns the entry, the offset of the LV within that entry's version range, and a boolean indicating if found.
func findEntryContaining(cg *CausalGraph, v LV) (*CGEntry, int, bool) {
if v < 0 || v >= cg.NextLV {
return nil, -1, false
}

idx := sort.Search(len(cg.Entries), func(i int) bool {
return cg.Entries[i].VEnd > v
})

if idx < len(cg.Entries) && cg.Entries[idx].Version <= v {
entry := &cg.Entries[idx]
offset := int(v - entry.Version)
return entry, offset, true
}
return nil, -1, false
}

// LVToRaw converts an LV to its corresponding RawVersion (agent, seq).
// Returns the RawVersion and true if found, otherwise RawVersion{} and false.
func LVToRaw(cg *CausalGraph, v LV) (RawVersion, bool) {
entry, offset, found := findEntryContaining(cg, v)
if !found {
return RawVersion{}, false
}
return RawVersion{Agent: entry.Agent, Seq: entry.Seq + offset}, true
}

// LVToRawWithParents converts an LV to its RawVersion and also returns its parents.
// This is a helper, similar to LVToRaw but includes parent LVs.
func LVToRawWithParents(cg *CausalGraph, v LV) (AgentID, int, []LV, bool) {
entry, offset, found := findEntryContaining(cg, v)
if !found {
return "", -1, nil, false
}
var parents []LV
if offset == 0 {
parents = entry.Parents
} else {
parents = []LV{v - 1}
}
return entry.Agent, entry.Seq + offset, parents, true
}

// RawToLV converts a RawVersion (agent, seq) to its corresponding LV.
// Returns the LV and an error if not found or invalid.
func RawToLV(cg *CausalGraph, agent AgentID, seq int) (LV, error) {
entry, offset, found := findEntryContainingRaw(cg, agent, seq)
if !found || entry == nil {
return -1, fmt.Errorf("raw version %s:%d not found in causal graph", agent, seq)
}
return entry.Version + LV(offset), nil
}

// LVToRawList converts a list of LVs to a list of RawVersions.
// If any LV is not found, it returns an error.
func LVToRawList(cg *CausalGraph, lvs []LV) ([]RawVersion, error) {
if len(lvs) == 0 {
return nil, nil
}
raws := make([]RawVersion, len(lvs))
for i, lv := range lvs {
rv, found := LVToRaw(cg, lv)
if !found {
return nil, fmt.Errorf("failed to convert LV %d to RawVersion: not found", lv)
}
raws[i] = rv
}
return raws, nil
}

// AddRaw adds a new version span to the causal graph.
func AddRaw(cg *CausalGraph, id RawVersion, length int, rawParents []RawVersion) (*CGEntry, error) {
if length <= 0 {
return nil, fmt.Errorf("length must be positive")
}

if _, err := RawToLV(cg, id.Agent, id.Seq); err == nil {
    return nil, nil // Duplicate
}

var parentLVs []LV
if rawParents == nil { // If nil, use current graph heads
    parentLVs = make([]LV, len(cg.Heads))
    copy(parentLVs, cg.Heads)
} else { // If not nil (could be empty slice or have elements), process them
    parentLVs = make([]LV, 0, len(rawParents))
    for _, rp := range rawParents {
        lv, err := RawToLV(cg, rp.Agent, rp.Seq)
        if err != nil {
            return nil, fmt.Errorf("parent %s:%d not found: %w", rp.Agent, rp.Seq, err)
        }
        parentLVs = append(parentLVs, lv)
    }
}
parentLVs = sortLVsAndDedup(parentLVs)

startLV := cg.NextLV
endLV := startLV + LV(length)

newEntry := CGEntry{
Agent:   id.Agent,
Seq:     id.Seq,
Version: startLV,
VEnd:    endLV,
Parents: parentLVs,
}
cg.Entries = append(cg.Entries, newEntry)
sort.Slice(cg.Entries, func(i, j int) bool {
    return cg.Entries[i].Version < cg.Entries[j].Version
})

cg.NextLV = endLV

clientEntries, _ := cg.AgentToVersion[id.Agent]
clientEntries = append(clientEntries, ClientEntry{
Seq:     id.Seq,
SeqEnd:  id.Seq + length,
Version: startLV,
})
sort.Slice(clientEntries, func(i, j int) bool {
return clientEntries[i].Seq < clientEntries[j].Seq
})
cg.AgentToVersion[id.Agent] = clientEntries

newHeads := make([]LV, 0, len(cg.Heads)+length) // Max capacity
for _, h := range cg.Heads {
isParent := false
for _, p := range parentLVs {
if h == p {
isParent = true
break
}
}
if !isParent {
newHeads = append(newHeads, h)
}
}
for i := 0; i < length; i++ {
    newHeads = append(newHeads, startLV+LV(i))
}
cg.Heads = sortLVsAndDedup(newHeads)

idx := sort.Search(len(cg.Entries), func(i int) bool {
    return cg.Entries[i].Version >= startLV
})
if idx < len(cg.Entries) && cg.Entries[idx].Version == startLV && cg.Entries[idx].Agent == id.Agent {
    return &cg.Entries[idx], nil
}

return nil, fmt.Errorf("internal error: added entry not found after sorting (target LV %d)", startLV)
}

// sortLVsAndDedup sorts a slice of LVs and removes duplicates, returning the new slice.
func sortLVsAndDedup(lvs []LV) []LV {
    if len(lvs) <= 1 {
        return lvs
    }
    sort.Slice(lvs, func(i, j int) bool { return lvs[i] < lvs[j] })

    j := 1
    for i := 1; i < len(lvs); i++ {
        if lvs[i] != lvs[i-1] {
            lvs[j] = lvs[i]
            j++
        }
    }
    return lvs[:j]
}

// VersionContainsLV checks if targetLV is an ancestor of (or equal to) any LV in frontier.
func VersionContainsLV(cg *CausalGraph, frontier []LV, targetLV LV) (bool, error) {
if targetLV < 0 || targetLV >= cg.NextLV {
    // Allow targetLV == cg.NextLV if cg.NextLV is 0 (empty graph), effectively targetLV is 0.
    // But if cg.NextLV > 0, then targetLV >= cg.NextLV is out of bounds.
    // A simpler check: if targetLV is negative, it's invalid. If positive or zero,
    // it must be < cg.NextLV unless cg.NextLV is 0.
    // If cg.NextLV is 0, any non-negative targetLV is out of bounds.
    // If targetLV is negative, it's always out of bounds.
    if targetLV < 0 || (cg.NextLV == 0 && targetLV >= 0) || (cg.NextLV > 0 && targetLV >= cg.NextLV) {
        return false, fmt.Errorf("targetLV %d is out of bounds for graph with %d LVs", targetLV, cg.NextLV)
    }
}

for _, fv := range frontier {
    if fv < 0 || fv >= cg.NextLV {
        return false, fmt.Errorf("frontier LV %d is out of bounds for graph with %d LVs", fv, cg.NextLV)
    }
    if fv == targetLV {
        return true, nil
    }
}
// If targetLV was valid but not found directly in frontier, and frontier is empty, it cannot be an ancestor.
if len(frontier) == 0 {
    return false, nil
}


queue := make([]LV, len(frontier))
copy(queue, frontier)
visited := make(map[LV]struct{})

for len(queue) > 0 {
curr := queue[0]
queue = queue[1:]

if _, ok := visited[curr]; ok {
continue
}
visited[curr] = struct{}{}

if curr < 0 {
continue
}
if curr == targetLV {
return true, nil
}

entry, offset, found := findEntryContaining(cg, curr)
if !found {
return false, fmt.Errorf("LV %d in frontier not found in graph during VersionContainsLV", curr)
}

var parents []LV
if offset == 0 {
parents = entry.Parents
} else {
parents = []LV{curr - 1}
}

for _, p := range parents {
if p == targetLV {
return true, nil
}
if _, vstd := visited[p]; !vstd && p >= 0 {
queue = append(queue, p)
}
}
}
return false, nil
}

// SummarizeVersion creates a VersionSummary for a given frontier.
// Each LV in the history of the frontier contributes a [seq, seq+1) range.
func SummarizeVersion(cg *CausalGraph, frontier []LV) (VersionSummary, error) {
summary := make(VersionSummary)
if len(frontier) == 0 {
return summary, nil
}

for _, fv := range frontier {
    if fv < 0 || fv >= cg.NextLV {
        return nil, fmt.Errorf("frontier LV %d is out of bounds for graph with %d LVs", fv, cg.NextLV)
    }
}

allHistoryLVs := make(map[LV]struct{})
queue := make([]LV, len(frontier))
copy(queue, frontier)
visited := make(map[LV]struct{})

for len(queue) > 0 {
curr := queue[0]
queue = queue[1:]

if _, ok := visited[curr]; ok {
continue
}
visited[curr] = struct{}{}

if curr < 0 {
continue
}
allHistoryLVs[curr] = struct{}{}

entry, offset, found := findEntryContaining(cg, curr)
if !found {
return nil, fmt.Errorf("LV %d in frontier/history not found in graph during SummarizeVersion", curr)
}

var parents []LV
if offset == 0 {
parents = entry.Parents
} else {
parents = []LV{curr - 1}
}
for _, p := range parents {
if _, vstd := visited[p]; !vstd && p >= 0 {
queue = append(queue, p)
}
}
}

agentSeqPairs := make(map[AgentID][]int)
for lv := range allHistoryLVs {
raw, found := LVToRaw(cg, lv)
if !found {
return nil, fmt.Errorf("failed to convert LV %d to RawVersion during SummarizeVersion", lv)
}
agentSeqPairs[raw.Agent] = append(agentSeqPairs[raw.Agent], raw.Seq)
}

for agent, seqs := range agentSeqPairs {
if len(seqs) == 0 {
continue
}
sort.Ints(seqs)

ranges := make([][2]int, 0, len(seqs))
for _, s := range seqs {
ranges = append(ranges, [2]int{s, s + 1})
}
summary[agent] = ranges
}

return summary, nil
}

// Diff calculates the versions in `from` that are not ancestors of or equal to `to`.
func Diff(cg *CausalGraph, from []LV, to VersionSummary) ([]LVRange, error) {
result := []LVRange{}
visitedForTraversal := make(map[LV]struct{})
initialQueue := make([]LV, 0, len(from))
tempVisitedForQueue := make(map[LV]struct{})

for _, v := range from {
    if _, seen := tempVisitedForQueue[v]; !seen {
        initialQueue = append(initialQueue, v)
        tempVisitedForQueue[v] = struct{}{}
    }
}
queue := sortLVsAndDedup(initialQueue)

processedInQueue := make(map[LV]struct{})

for len(queue) > 0 {
v := queue[0]
queue = queue[1:]

if _, ok := visitedForTraversal[v]; ok {
continue
}

entry, _, found := findEntryContaining(cg, v)
if !found {
return nil, fmt.Errorf("LV %d in 'from' or its history not found in graph during Diff", v)
}

for lvInEntry := entry.Version; lvInEntry < entry.VEnd; lvInEntry++ {
visitedForTraversal[lvInEntry] = struct{}{}
}

isEntireEntryCoveredByTo := true
currentRunStartLV := LV(-1)

for lvIter := entry.Version; lvIter < entry.VEnd; lvIter++ {
seqIter := entry.Seq + int(lvIter-entry.Version)
isLVCoveredByTo := false
if ranges, ok := to[entry.Agent]; ok {
for _, r := range ranges {
if seqIter >= r[0] && seqIter < r[1] {
isLVCoveredByTo = true
break
}
}
}

if !isLVCoveredByTo {
isEntireEntryCoveredByTo = false
if currentRunStartLV == -1 {
currentRunStartLV = lvIter
}
} else {
if currentRunStartLV != -1 {
result = append(result, LVRange{Start: currentRunStartLV, End: lvIter})
currentRunStartLV = -1
}
}
}
if currentRunStartLV != -1 {
result = append(result, LVRange{Start: currentRunStartLV, End: entry.VEnd})
}

if !isEntireEntryCoveredByTo {
for _, p := range entry.Parents {
if _, qProc := processedInQueue[p]; !qProc && p >= 0 {
pIsCoveredByTo := false
pRaw, pFound := LVToRaw(cg, p)
if pFound {
if ranges, ok := to[pRaw.Agent]; ok {
for _, r := range ranges {
if pRaw.Seq >= r[0] && pRaw.Seq < r[1] {
pIsCoveredByTo = true
break
}
}
}
}
if !pIsCoveredByTo {
queue = append(queue, p)
processedInQueue[p] = struct{}{}
}
}
}
}
}

if len(result) == 0 {
return result, nil
}
sort.Slice(result, func(i, j int) bool {
return result[i].Start < result[j].Start
})

merged := []LVRange{result[0]}
for i := 1; i < len(result); i++ {
last := &merged[len(merged)-1]
current := result[i]
if current.Start == last.End {
last.End = current.End
} else if current.Start < last.End {
if current.End > last.End {
last.End = current.End
}
} else {
merged = append(merged, current)
}
}
return merged, nil
}

// FindDominators finds the "head" versions within the common ancestors of the specified versions.
// A version is a head if it's a common ancestor and no other common ancestor is its descendant.
func FindDominators(cg *CausalGraph, versions []LV) ([]LV, error) {
if len(versions) == 0 {
return []LV{}, nil
}
uniqueVersions := sortLVsAndDedup(append([]LV(nil), versions...))

if len(uniqueVersions) == 1 {
v := uniqueVersions[0]
if v < 0 || v >= cg.NextLV {
return nil, fmt.Errorf("version %d not found in graph", v)
}
return []LV{v}, nil
}

ancestorSets := make([]map[LV]struct{}, len(uniqueVersions))
for i, v := range uniqueVersions {
if v < 0 || v >= cg.NextLV {
return nil, fmt.Errorf("version %d not found in graph or invalid", v)
}
set := make(map[LV]struct{})
q := []LV{v}
visitedInSet := make(map[LV]struct{})

for len(q) > 0 {
curr := q[0]
q = q[1:]

if _, ok := visitedInSet[curr]; ok {
continue
}
visitedInSet[curr] = struct{}{}
set[curr] = struct{}{}

entry, offset, found := findEntryContaining(cg, curr)
if !found {
return nil, fmt.Errorf("LV %d in history not found during FindDominators", curr)
}
var parents []LV
if offset == 0 {
parents = entry.Parents
} else {
parents = []LV{curr - 1}
}
for _, p := range parents {
if _, vstd := visitedInSet[p]; !vstd && p >= 0 {
q = append(q, p)
}
}
}
ancestorSets[i] = set
}

if len(ancestorSets) == 0 {
return []LV{}, nil
}
common := make(map[LV]struct{})
if len(ancestorSets) > 0 {
    for lv := range ancestorSets[0] {
        common[lv] = struct{}{}
    }
}

for i := 1; i < len(ancestorSets); i++ {
currentSet := ancestorSets[i]
nextCommon := make(map[LV]struct{})
for lv := range currentSet {
if _, ok := common[lv]; ok {
nextCommon[lv] = struct{}{}
}
}
common = nextCommon
if len(common) == 0 {
return []LV{}, nil
}
}

// Filter for "heads": ca is a head if it's not an ancestor of any otherCa in common.
dominators := make([]LV, 0, len(common))
for ca := range common {
    isAncestorOfAnotherCommon := false
    for otherCa := range common {
        if ca == otherCa {
            continue
        }
        // Check if ca is an ancestor of otherCa.
        // VersionContainsLV(cg, frontier_is_otherCa, target_is_ca)
        caIsAncestor, err := VersionContainsLV(cg, []LV{otherCa}, ca)
        if err != nil {
            return nil, fmt.Errorf("error checking ancestry for dominator filtering: %w", err)
        }
        if caIsAncestor {
            isAncestorOfAnotherCommon = true
            break
        }
    }
    if !isAncestorOfAnotherCommon {
        dominators = append(dominators, ca)
    }
}
return sortLVsAndDedup(dominators), nil
}

// FindConflicting returns operations in `versions` that are not descendants of `commonAncestors`.
func FindConflicting(cg *CausalGraph, versions []LV, commonAncestors []LV) ([]LVRange, error) {
summary, err := SummarizeVersion(cg, commonAncestors)
if err != nil {
return nil, fmt.Errorf("FindConflicting: could not summarize commonAncestors: %w", err)
}
return Diff(cg, versions, summary)
}

// Relation defines the relationship between two versions.
type Relation string

const (
RelationEqual      Relation = "eq"
RelationAncestor   Relation = "ancestor"
RelationDescendant Relation = "descendant"
RelationConcurrent Relation = "concurrent"
)

// CompareVersions determines the relationship between two LVs, a and b.
func CompareVersions(cg *CausalGraph, a, b LV) (Relation, error) {
if a == b {
return RelationEqual, nil
}
aIsAncestor, err := VersionContainsLV(cg, []LV{b}, a)
if err != nil {
return "", fmt.Errorf("error checking if %d is ancestor of %d: %w", a, b, err)
}
if aIsAncestor {
return RelationAncestor, nil
}
bIsAncestor, err := VersionContainsLV(cg, []LV{a}, b)
if err != nil {
return "", fmt.Errorf("error checking if %d is ancestor of %d: %w", b, a, err)
}
if bIsAncestor {
return RelationDescendant, nil
}
return RelationConcurrent, nil
}

// iterVersionsBetweenBP is a helper for IterVersionsBetween.
func iterVersionsBetweenBP(cg *CausalGraph, from []LV, to LV,
fn func(v LV, isParentOfPrev bool, isMerge bool) (stop bool, err error)) error {
queue := []struct {
v              LV
isParentOfPrev bool
}{{v: to, isParentOfPrev: false}}
visited := make(map[LV]struct{})

for _, fv := range from {
visited[fv] = struct{}{}
}

for _, fv := range from {
    if fv == to {
        return nil
    }
}

for len(queue) > 0 {
item := queue[len(queue)-1]
queue = queue[:len(queue)-1]
v := item.v
isParentOfPrev := item.isParentOfPrev

if _, ok := visited[v]; ok {
continue
}

entry, offset, found := findEntryContaining(cg, v)
if !found {
return fmt.Errorf("iterVersionsBetweenBP: LV %d not found in CG", v)
}

stop, err := fn(v, isParentOfPrev, isMergeFlag(entry, offset))
if err != nil {
return fmt.Errorf("iterVersionsBetweenBP: callback error at LV %d: %w", v, err)
}
if stop {
return nil
}
visited[v] = struct{}{}

var parentsToVisit []LV
if offset == 0 {
parentsToVisit = entry.Parents
} else {
parentsToVisit = []LV{v - 1}
}

for i := len(parentsToVisit) - 1; i >= 0; i-- {
p := parentsToVisit[i]
if _, stopAtParent := visited[p]; !stopAtParent && p >= 0 {
queue = append(queue, struct{v LV; isParentOfPrev bool}{p, i == 0 && len(parentsToVisit) > 0})
}
}
}
return nil
}

func isMergeFlag(entry *CGEntry, offset int) bool {
if offset == 0 {
return len(entry.Parents) > 1
}
return false
}

// IterVersionsBetween iterates over LVs in the range (from, to].
func IterVersionsBetween(cg *CausalGraph, from []LV, to LV,
fn func(v LV, isParentOfPrev bool, isMerge bool) (stop bool, err error)) error {

    if to < 0 || (cg.NextLV > 0 && to >= cg.NextLV) || (cg.NextLV == 0 && to != 0) {
        // Allow to == 0 if cg.NextLV == 0 (empty graph, iterating towards nothing from nothing)
        // but if cg.NextLV > 0, then to >= cg.NextLV is out of bounds.
        // If cg.NextLV == 0 and to is not 0, it's out of bounds.
        if !(cg.NextLV == 0 && to == 0) { // Special case: to=0 is valid for empty graph if from is also empty or contains 0
             return fmt.Errorf("IterVersionsBetween: 'to' LV %d is out of bounds for graph with %d LVs", to, cg.NextLV)
        }
    }


for _, fv := range from {
    if fv < 0 || (cg.NextLV > 0 && fv >= cg.NextLV) || (cg.NextLV == 0 && fv != 0) {
        if !(cg.NextLV == 0 && fv == 0) {
            return fmt.Errorf("IterVersionsBetween: 'from' LV %d is out of bounds for graph with %d LVs", fv, cg.NextLV)
        }
    }
if fv == to { return nil } // If any 'from' is 'to', the range is empty or invalid in one interpretation.
    isToAncestorOfFrom, err := VersionContainsLV(cg, []LV{fv}, to) // VersionContainsLV now has its own checks
    if err != nil {
        return fmt.Errorf("IterVersionsBetween: error checking ancestry for 'from' LV %d: %w", fv, err)
    }
    if isToAncestorOfFrom { // If 'to' is an ancestor of any 'from', the range is invalid.
        return nil
    }
}
return iterVersionsBetweenBP(cg, from, to, fn)
}

// IntersectWithSummaryFull finds versions in cg.Heads not covered by summary.
func IntersectWithSummaryFull(cg *CausalGraph, summary VersionSummary) ([]CGEntry, error) {
result := []CGEntry{}
visitedLVs := make(map[LV]struct{})

queue := make([]LV, len(cg.Heads))
copy(queue, cg.Heads)
queue = sortLVsAndDedup(queue)

processedEntries := make(map[LV]struct{})
for len(queue) > 0 {
v := queue[len(queue)-1]
queue = queue[:len(queue)-1]

if v < 0 { continue }
if _, ok := visitedLVs[v]; ok {
continue
}

entry, _, found := findEntryContaining(cg, v)
if !found {
return nil, fmt.Errorf("IntersectWithSummaryFull: LV %d (from queue) not found in CG", v)
}

if _, ok := processedEntries[entry.Version]; ok {
continue
}

currentRunStartLV := LV(-1)
var currentRunParents []LV

for lvIter := entry.VEnd - 1; lvIter >= entry.Version; lvIter-- {
if _, ok := visitedLVs[lvIter]; ok {
if currentRunStartLV != -1 {
startSeq := entry.Seq + int((lvIter+1)-entry.Version)
result = append(result, CGEntry{
Agent:   entry.Agent,
Seq:     startSeq,
Version: lvIter + 1,
VEnd:    currentRunStartLV + 1,
Parents: currentRunParents,
})
currentRunStartLV = -1
}
continue
}

seqIter := entry.Seq + int(lvIter-entry.Version)
isCovered := false
if ranges, ok := summary[entry.Agent]; ok {
for _, r := range ranges {
if seqIter >= r[0] && seqIter < r[1] {
isCovered = true
break
}
}
}

if !isCovered {
if currentRunStartLV == -1 {
currentRunStartLV = lvIter
}
if lvIter == entry.Version {
currentRunParents = entry.Parents
} else {
currentRunParents = []LV{lvIter - 1}
}
} else {
if currentRunStartLV != -1 {
startSeq := entry.Seq + int((lvIter+1)-entry.Version)
result = append(result, CGEntry{
Agent:   entry.Agent,
Seq:     startSeq,
Version: lvIter + 1,
VEnd:    currentRunStartLV + 1,
Parents: currentRunParents,
})
currentRunStartLV = -1
}
visitedLVs[lvIter] = struct{}{}
}
}

if currentRunStartLV != -1 {
startSeq := entry.Seq
result = append(result, CGEntry{
Agent:   entry.Agent,
Seq:     startSeq,
Version: entry.Version,
VEnd:    currentRunStartLV + 1,
Parents: entry.Parents,
})
}

processedEntries[entry.Version] = struct{}{}

for _, p := range entry.Parents {
if p >= 0 {
if _, parentVisited := visitedLVs[p]; !parentVisited {
queue = append(queue, p)
}
}
}
}

for _, rEntry := range result {
    for v := rEntry.Version; v < rEntry.VEnd; v++ {
        visitedLVs[v] = struct{}{}
    }
}

sort.Slice(result, func(i, j int) bool {
if result[i].Version != result[j].Version {
return result[i].Version < result[j].Version
}
return result[i].Agent < result[j].Agent
})
return result, nil
}

// IntersectWithSummary is a simpler version of IntersectWithSummaryFull.
func IntersectWithSummary(cg *CausalGraph, summary VersionSummary) ([]LV, error) {
entries, err := IntersectWithSummaryFull(cg, summary)
if err != nil {
return nil, err
}
var lvs []LV
for _, entry := range entries {
for v := entry.Version; v < entry.VEnd; v++ {
lvs = append(lvs, v)
}
}
return sortLVsAndDedup(lvs), nil
}
