package causalgraph

import (
"reflect"
"sort"
"testing"
)

// Helper function to check deep equality for slices of LV, as direct == doesn't work.
func compareLVSlices(a, b []LV) bool {
// Sort for canonical comparison if order doesn't strictly matter but content does.
// For FindDominators, the order of output LVs might not be guaranteed by the spec,
// but sorting makes tests stable. The function itself already sorts and dedups.
if len(a) == 0 && len(b) == 0 {
return true
}
// Create copies before sorting if the original slices should not be modified
acopy := append([]LV(nil), a...)
bcopy := append([]LV(nil), b...)
sort.Slice(acopy, func(i, j int) bool { return acopy[i] < acopy[j] })
sort.Slice(bcopy, func(i, j int) bool { return bcopy[i] < bcopy[j] })
return reflect.DeepEqual(acopy, bcopy)
}

// Helper function to check deep equality for VersionSummary.
func compareVersionSummaries(a, b VersionSummary) bool {
return reflect.DeepEqual(a, b)
}

// Helper function to check deep equality for slices of LVRange.
func compareLVRangeSlices(t *testing.T, got, want []LVRange) {
t.Helper()
if !reflect.DeepEqual(got, want) {
t.Errorf("LVRange slice mismatch:\ngot:  %v\nwant: %v", got, want)
}
}

// Helper function to check deep equality for slices of CGEntry.
// It sorts both slices and the Parents field within each CGEntry before comparison.
func compareCGEntrySlices(t *testing.T, got, want []CGEntry) {
	t.Helper()

	// Handle nil vs empty slice cases for reflect.DeepEqual
	if (got == nil && want != nil && len(want) == 0) || (want == nil && got != nil && len(got) == 0) {
		// Consider nil and empty slice as equal for this comparison
	} else if (got == nil && want == nil) || (len(got) == 0 && len(want) == 0) {
		// Both nil or both empty, they are equal
	} else {
		// Create copies to avoid modifying original slices
		gotCopy := make([]CGEntry, len(got))
		copy(gotCopy, got)
		wantCopy := make([]CGEntry, len(want))
		copy(wantCopy, want)

		// Sort Parents within each CGEntry
		for i := range gotCopy {
			sort.Slice(gotCopy[i].Parents, func(x, y int) bool { return gotCopy[i].Parents[x] < gotCopy[i].Parents[y] })
		}
		for i := range wantCopy {
			sort.Slice(wantCopy[i].Parents, func(x, y int) bool { return wantCopy[i].Parents[x] < wantCopy[i].Parents[y] })
		}

		// Sort the slices of CGEntry by Version, then Agent
		sort.Slice(gotCopy, func(i, j int) bool {
			if gotCopy[i].Version != gotCopy[j].Version {
				return gotCopy[i].Version < gotCopy[j].Version
			}
			return gotCopy[i].Agent < gotCopy[j].Agent
		})
		sort.Slice(wantCopy, func(i, j int) bool {
			if wantCopy[i].Version != wantCopy[j].Version {
				return wantCopy[i].Version < wantCopy[j].Version
			}
			return wantCopy[i].Agent < wantCopy[j].Agent
		})

		if !reflect.DeepEqual(gotCopy, wantCopy) {
			t.Errorf("CGEntry slice mismatch:\ngot:  %+v\nwant: %+v", gotCopy, wantCopy)
		}
		return // Already handled by DeepEqual or error
	}

	// If one is nil and the other is non-nil empty, or both are nil/empty, they are considered equal.
	// If reflect.DeepEqual was not called above, it means we are in one of these equivalent states.
	// If they were not equivalent, DeepEqual would have caught it.
	// This explicit check is mostly for clarity if DeepEqual's behavior with nil/empty is surprising.
	if (got == nil && (want == nil || len(want) == 0)) || ((got == nil || len(got) == 0) && want == nil) {
		return
	}
	if len(got) == 0 && len(want) == 0 { // Both empty non-nil
		return
	}

	// Fallback if one is nil and other is not empty, or vice-versa, which is a mismatch.
	// This path should ideally be caught by the main DeepEqual if not for the nil/empty special handling.
	if (got == nil && len(want) > 0) || (len(got) > 0 && want == nil) {
		t.Errorf("CGEntry slice mismatch (nil vs non-empty):\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestCreateCG(t *testing.T) {
cg := CreateCG()
if cg == nil {
t.Fatal("CreateCG returned nil")
}
if len(cg.Heads) != 0 {
t.Errorf("expected Heads to be empty, got %v", cg.Heads)
}
if len(cg.Entries) != 0 {
t.Errorf("expected Entries to be empty, got %v", cg.Entries)
}
if len(cg.AgentToVersion) != 0 {
t.Errorf("expected AgentToVersion to be empty, got %v", cg.AgentToVersion)
}
}

func TestAddRaw_SingleEntry(t *testing.T) {
cg := CreateCG()
agentAStr := "agentA"
agentA := AgentID(agentAStr)
idA0 := RawVersion{Agent: agentA, Seq: 0}

entry, err := AddRaw(cg, idA0, 1, nil) // Add agentA:0, len 1, parents: current heads (empty)
if err != nil {
t.Fatalf("AddRaw failed: %v", err)
}
if entry == nil {
t.Fatal("AddRaw returned nil entry")
}

// Check CGEntry
if entry.Agent != agentA || entry.Seq != 0 || entry.Version != 0 || entry.VEnd != 1 {
t.Errorf("unexpected entry fields: %+v", entry)
}
if len(entry.Parents) != 0 {
t.Errorf("expected empty parents for first entry, got %v", entry.Parents)
}

// Check CausalGraph state
if len(cg.Entries) != 1 {
t.Fatalf("expected 1 entry in cg.Entries, got %d", len(cg.Entries))
}
if !reflect.DeepEqual(cg.Entries[0], *entry) {
t.Errorf("cg.Entries[0] (%+v) does not match returned entry (%+v)", cg.Entries[0], *entry)
}

expectedHeads := []LV{0}
if !compareLVSlices(cg.Heads, expectedHeads) {
t.Errorf("expected Heads %v, got %v", expectedHeads, cg.Heads)
}

if NextLV(cg) != 1 {
t.Errorf("expected NextLV to be 1, got %d", NextLV(cg))
}
if NextSeqForAgent(cg, agentA) != 1 {
t.Errorf("expected NextSeqForAgent for %s to be 1, got %d", agentA, NextSeqForAgent(cg, agentA))
}

clientEntries, ok := cg.AgentToVersion[agentA]
if !ok {
t.Fatalf("agent %s not found in AgentToVersion", agentA)
}
if len(clientEntries) != 1 {
t.Fatalf("expected 1 clientEntry for agent %s, got %d", agentA, len(clientEntries))
}
expectedClientEntry := ClientEntry{Seq: 0, SeqEnd: 1, Version: 0}
if !reflect.DeepEqual(clientEntries[0], expectedClientEntry) {
t.Errorf("unexpected clientEntry: got %+v, want %+v", clientEntries[0], expectedClientEntry)
}
}

func TestAddRaw_AdvancedScenarios(t *testing.T) {
	agentA := AgentID("agentA")
	agentB := AgentID("agentB")
	agentC := AgentID("agentC")

	// Scenario 1: Attempting to add an overlapping operation (earlier sequence)
	t.Run("Overlap_EarlierSeq", func(t *testing.T) {
		cg := CreateCG()
		_, _ = AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 3, nil) // A0, A1, A2. NextSeq for A is 3.

		_, err := AddRaw(cg, RawVersion{Agent: agentA, Seq: 1}, 1, nil) // Try to add A1 again
		if err == nil {
			t.Errorf("Expected error when adding overlapping operation (A, seq 1) after (A, seq 0, len 3), but got nil")
		}
		// TODO: Check specific error message if it becomes part of the API contract, e.g., strings.Contains(err.Error(), "out of order")
	})

	// Scenario 2: Attempting to add a contained operation
	t.Run("Contained_Operation", func(t *testing.T) {
		cg := CreateCG()
		_, _ = AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 3, nil) // A0, A1, A2. NextSeq for A is 3.

		_, err := AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 1, nil) // Try to add A0 (subset)
		if err == nil {
			t.Errorf("Expected error when adding contained operation (A, seq 0, len 1) within (A, seq 0, len 3), but got nil")
}
}
}

func TestRawToLV_ErrorCases(t *testing.T) {
cg := setupTestGraphG1(t) // G1: A0(0) -> B0(1), A0(0) -> A1(2), (B0(1),A1(2)) -> C0(3)
agentA := AgentID("agentA")
// agentB := AgentID("agentB") // agentB has B0 (LV1, seq 0)
// agentC := AgentID("agentC") // agentC has C0 (LV3, seq 0)
unknownAgent := AgentID("unknownAgent")

tests := []struct {
name    string
agent   AgentID
seq     int
wantErr bool
// wantErrStr string // Optional: if we want to check specific error messages
}{
{
name:    "Agent_Not_In_Graph",
agent:   unknownAgent,
seq:     0,
wantErr: true,
// wantErrStr: "agent not found",
},
{
name:    "Seq_Out_Of_Bounds_For_AgentA_Positive",
agent:   agentA, // agentA has ops (A0, seq 0, len 1), (A1, seq 1, len 1)
seq:     5,      // Max seq for agentA is 1.
wantErr: true,
// wantErrStr: "sequence number out of bounds",
},
{
name:    "Seq_Negative_For_AgentA",
agent:   agentA,
seq:     -1,
wantErr: true,
// wantErrStr: "sequence number out of bounds", // or "invalid sequence number"
},
// Example of a sequence number that is valid for the agent but not the start of an op,
// RawToLV should still resolve it if it's within a span.
// G2: A0-2(0,1,2) -> B0-1(3,4)
// cgG2 := setupTestGraphG2(t)
// {
//  name:    "Seq_Within_Span_Not_Start_G2_AgentA_Seq1",
//  cgForTest: cgG2, // Special field for this test if needed, or separate test.
//  agent:   agentA, // agentA has A0-2 (seq 0, len 3) -> LVs 0,1,2
//  seq:     1,      // Seq 1 for agentA is LV 1
//  wantErr: false,
// },
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
// Use tt.cgForTest if provided, else default cg
currentCG := cg
// if tt.cgForTest != nil {
//  currentCG = tt.cgForTest
// }

_, err := RawToLV(currentCG, tt.agent, tt.seq)
if (err != nil) != tt.wantErr {
t.Errorf("RawToLV(%s, %d) error = %v, wantErr %v", tt.agent, tt.seq, err, tt.wantErr)
}
// if tt.wantErr && err != nil && tt.wantErrStr != "" && !strings.Contains(err.Error(), tt.wantErrStr) {
//  t.Errorf("RawToLV(%s, %d) error message = %q, want to contain %q", tt.agent, tt.seq, err.Error(), tt.wantErrStr)
// }
})
}
}

func TestSummarizeVersion(t *testing.T) {
cg := CreateCG()
		_, _ = AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 3, nil) // A0, A1, A2. NextSeq for A is 3.

		_, err := AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 3, nil) // Try to add A0-A2 again
		if err == nil {
			t.Errorf("Expected error when re-adding identical operation (A, seq 0, len 3), but got nil")
		}
	})

	// Scenario 4: Gap in sequence numbers
	t.Run("Gap_In_Sequence", func(t *testing.T) {
		cg := CreateCG()
		_, _ = AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 1, nil) // A0. NextSeq for A is 1.

		_, err := AddRaw(cg, RawVersion{Agent: agentA, Seq: 2}, 1, nil) // Try to add A2, skipping A1
		if err == nil {
			t.Errorf("Expected error when adding operation with a gap in sequence (A, seq 2 after A, seq 0), but got nil")
		}
		// TODO: Check specific error message, e.g., strings.Contains(err.Error(), "gap in sequence numbers")
	})

	// Scenario 5: Valid sequential add (control case)
	t.Run("Valid_Sequential_Add", func(t *testing.T) {
		cg := CreateCG()
		_, err := AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 1, nil) // A0. NextSeq for A is 1.
		if err != nil {
			t.Fatalf("Setup for valid sequential add failed: %v", err)
		}

		entry, err := AddRaw(cg, RawVersion{Agent: agentA, Seq: 1}, 1, []RawVersion{{Agent: agentA, Seq: 0}}) // Add A1
		if err != nil {
			t.Errorf("Expected no error for valid sequential add, but got %v", err)
		}
		if entry == nil {
			t.Fatal("Valid add returned nil entry")
		}
		if entry.Agent != agentA || entry.Seq != 1 || entry.Version != 1 { // LV0 was A0, so A1 is LV1
			t.Errorf("Unexpected entry fields for A1: %+v. Expected Agent: %s, Seq: 1, Version: 1", entry, agentA)
		}
		if NextSeqForAgent(cg, agentA) != 2 {
			t.Errorf("Expected NextSeqForAgent to be 2, got %d", NextSeqForAgent(cg, agentA))
		}
	})

	// Scenario 6: Add with multiple parents
	t.Run("Multiple_Parents", func(t *testing.T) {
		cg := CreateCG()
		entryA, errA := AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 1, nil) // A0, LV0
		if errA != nil {
			t.Fatalf("Failed to add entryA: %v", errA)
		}
		entryB, errB := AddRaw(cg, RawVersion{Agent: agentB, Seq: 0}, 1, []RawVersion{}) // B0, LV1 (independent)
		if errB != nil {
			t.Fatalf("Failed to add entryB: %v", errB)
		}

		parentsRaw := []RawVersion{{Agent: agentA, Seq: 0}, {Agent: agentB, Seq: 0}}
		entryC, errC := AddRaw(cg, RawVersion{Agent: agentC, Seq: 0}, 1, parentsRaw) // C0
		if errC != nil {
			t.Fatalf("Failed to add C0 with multiple parents: %v", errC)
		}
		if entryC == nil {
			t.Fatal("AddRaw with multiple parents returned nil entryC")
		}

		if entryC.Agent != agentC || entryC.Seq != 0 || entryC.Version != 2 { // LV0=A0, LV1=B0, so C0 is LV2
			t.Errorf("Unexpected entry fields for C0: %+v. Expected Agent: %s, Seq: 0, Version: 2", entryC, agentC)
		}

		expectedParentsLV := []LV{entryA.Version, entryB.Version}
		if !compareLVSlices(entryC.Parents, expectedParentsLV) {
			t.Errorf("C0 parents mismatch: got %v, want %v", entryC.Parents, expectedParentsLV)
		}

		expectedHeads := []LV{entryC.Version}
		if !compareLVSlices(cg.Heads, expectedHeads) {
			t.Errorf("Heads mismatch after adding C0: got %v, want %v", cg.Heads, expectedHeads)
		}
	})

	// Scenario 7: Adding an operation whose parent is not yet known (by RawVersion)
	t.Run("Parent_Not_Known_RawVersion", func(t *testing.T) {
		cg := CreateCG()
		_, _ = AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 1, nil) // A0

		parentsRaw := []RawVersion{{Agent: agentB, Seq: 0}} // B0 doesn't exist for agentB
		_, err := AddRaw(cg, RawVersion{Agent: agentC, Seq: 0}, 1, parentsRaw)
		if err == nil {
			t.Errorf("Expected error when adding operation with unknown raw parent, but got nil")
		}
		// TODO: Check specific error message if desired
	})

	// Scenario 8: Adding an operation with a non-existent agent in parent RawVersion
	t.Run("Parent_Agent_Not_Known", func(t *testing.T) {
		cg := CreateCG()
		// No ops added yet.

		parentsRaw := []RawVersion{{Agent: AgentID("nonExistentAgent"), Seq: 0}}
		_, err := AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 1, parentsRaw)
		if err == nil {
			t.Errorf("Expected error when adding operation with parent from non-existent agent, but got nil")
		}
	})

	// Scenario 9: Invalid length for AddRaw
	t.Run("Invalid_Length", func(t *testing.T) {
		cg := CreateCG()
		_, errZero := AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 0, nil)
		if errZero == nil {
			t.Errorf("Expected error for length 0, but got nil")
		}
		// TODO: Check specific error message if desired, e.g., strings.Contains(errZero.Error(), "length must be positive")

		_, errNegative := AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, -1, nil)
		if errNegative == nil {
			t.Errorf("Expected error for negative length, but got nil")
		}
		// TODO: Check specific error message if desired
	})
}

func TestLVToRawAndRawToLV(t *testing.T) {
cg := CreateCG()
agentA := AgentID("agentA")
agentB := AgentID("agentB")

_, err := AddRaw(cg, RawVersion{Agent: agentA, Seq: 0}, 3, nil)
if err != nil {
t.Fatalf("AddRaw(agentA) failed: %v", err)
}
_, err = AddRaw(cg, RawVersion{Agent: agentB, Seq: 0}, 2, []RawVersion{{Agent: agentA, Seq: 2}})
if err != nil {
t.Fatalf("AddRaw(agentB) failed: %v", err)
}

tests := []struct {
name    string
lv      LV
wantRV  RawVersion
wantErr bool
}{
{"agentA_0", 0, RawVersion{Agent: agentA, Seq: 0}, false},
{"agentA_1", 1, RawVersion{Agent: agentA, Seq: 1}, false},
{"agentA_2", 2, RawVersion{Agent: agentA, Seq: 2}, false},
{"agentB_0", 3, RawVersion{Agent: agentB, Seq: 0}, false},
{"agentB_1", 4, RawVersion{Agent: agentB, Seq: 1}, false},
{"non_existent_lv", 5, RawVersion{}, true},
{"negative_lv", -1, RawVersion{}, true},
}

for _, tt := range tests {
t.Run("LVToRaw_"+tt.name, func(t *testing.T) {
gotRV, found := LVToRaw(cg, tt.lv)
if tt.wantErr {
if found {
t.Errorf("LVToRaw(%d) expected not found, but got %+v", tt.lv, gotRV)
}
} else {
if !found {
t.Errorf("LVToRaw(%d) expected found, but was not", tt.lv)
}
if !reflect.DeepEqual(gotRV, tt.wantRV) {
t.Errorf("LVToRaw(%d) = %+v, want %+v", tt.lv, gotRV, tt.wantRV)
}
}
})

if !tt.wantErr {
t.Run("RawToLV_"+tt.name, func(t *testing.T) {
gotLV, err := RawToLV(cg, tt.wantRV.Agent, tt.wantRV.Seq)
if err != nil {
t.Errorf("RawToLV(%s, %d) failed: %v", tt.wantRV.Agent, tt.wantRV.Seq, err)
}
if gotLV != tt.lv {
t.Errorf("RawToLV(%s, %d) = %d, want %d", tt.wantRV.Agent, tt.wantRV.Seq, gotLV, tt.lv)
}
})
}
}
}

func TestRawToLV_ErrorCases(t *testing.T) {
	cg := setupTestGraphG1(t) // G1: A0(0) -> B0(1), A0(0) -> A1(2), (B0(1),A1(2)) -> C0(3)
	agentA := AgentID("agentA")
	// agentB := AgentID("agentB") // agentB has B0 (LV1, seq 0)
	// agentC := AgentID("agentC") // agentC has C0 (LV3, seq 0)
	unknownAgent := AgentID("unknownAgent")

	tests := []struct {
		name    string
		agent   AgentID
		seq     int
		wantErr bool
		// wantErrStr string // Optional: if we want to check specific error messages
	}{
		{
			name:    "Agent_Not_In_Graph",
			agent:   unknownAgent,
			seq:     0,
			wantErr: true,
			// wantErrStr: "agent not found",
		},
		{
			name:    "Seq_Out_Of_Bounds_For_AgentA_Positive",
			agent:   agentA, // agentA has ops (A0, seq 0, len 1), (A1, seq 1, len 1)
			seq:     5,      // Max seq for agentA is 1.
			wantErr: true,
			// wantErrStr: "sequence number out of bounds",
		},
		{
			name:    "Seq_Negative_For_AgentA",
			agent:   agentA,
			seq:     -1,
			wantErr: true,
			// wantErrStr: "sequence number out of bounds", // or "invalid sequence number"
		},
		// Example of a sequence number that is valid for the agent but not the start of an op,
		// RawToLV should still resolve it if it's within a span.
		// G2: A0-2(0,1,2) -> B0-1(3,4)
		// cgG2 := setupTestGraphG2(t)
		// {
		//  name:    "Seq_Within_Span_Not_Start_G2_AgentA_Seq1",
		//  cgForTest: cgG2, // Special field for this test if needed, or separate test.
		//  agent:   agentA, // agentA has A0-2 (seq 0, len 3) -> LVs 0,1,2
		//  seq:     1,      // Seq 1 for agentA is LV 1
		//  wantErr: false,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use tt.cgForTest if provided, else default cg
			currentCG := cg
			// if tt.cgForTest != nil {
			//  currentCG = tt.cgForTest
			// }

			_, err := RawToLV(currentCG, tt.agent, tt.seq)
			if (err != nil) != tt.wantErr {
				t.Errorf("RawToLV(%s, %d) error = %v, wantErr %v", tt.agent, tt.seq, err, tt.wantErr)
			}
			// if tt.wantErr && err != nil && tt.wantErrStr != "" && !strings.Contains(err.Error(), tt.wantErrStr) {
			//  t.Errorf("RawToLV(%s, %d) error message = %q, want to contain %q", tt.agent, tt.seq, err.Error(), tt.wantErrStr)
			// }
		})
	}
}

func TestSummarizeVersion(t *testing.T) {
cg := CreateCG()
agentA := AgentID("agentA")
agentB := AgentID("agentB")
agentC := AgentID("agentC")

_, _ = AddRaw(cg, RawVersion{agentA, 0}, 1, nil)
_, _ = AddRaw(cg, RawVersion{agentB, 0}, 1, []RawVersion{{agentA, 0}})
_, _ = AddRaw(cg, RawVersion{agentA, 1}, 1, []RawVersion{{agentA, 0}})
_, _ = AddRaw(cg, RawVersion{agentC, 0}, 1, []RawVersion{{agentB, 0}, {agentA, 1}})

frontier1 := []LV{1, 2}
wantSummary1 := VersionSummary{
agentA: [][2]int{{0, 1}, {1, 2}},
agentB: [][2]int{{0, 1}},
}
summary1, err := SummarizeVersion(cg, frontier1)
if err != nil {
t.Fatalf("SummarizeVersion for frontier1 failed: %v", err)
}
if !compareVersionSummaries(summary1, wantSummary1) {
t.Errorf("SummarizeVersion(%v) = %v, want %v", frontier1, summary1, wantSummary1)
}

frontier2 := cg.Heads
wantSummary2 := VersionSummary{
agentA: [][2]int{{0, 1}, {1, 2}},
agentB: [][2]int{{0, 1}},
agentC: [][2]int{{0, 1}},
}
summary2, err := SummarizeVersion(cg, frontier2)
if err != nil {
t.Fatalf("SummarizeVersion for frontier2 failed: %v", err)
}
if !compareVersionSummaries(summary2, wantSummary2) {
t.Errorf("SummarizeVersion(%v) = %v, want %v", frontier2, summary2, wantSummary2)
}

summaryEmpty, err := SummarizeVersion(cg, []LV{})
if err != nil {
t.Fatalf("SummarizeVersion for empty frontier failed: %v", err)
}
if len(summaryEmpty) != 0 {
t.Errorf("SummarizeVersion([]) expected empty summary, got %v", summaryEmpty)
}
}

// setupTestGraphG1 creates a predefined causal graph.
// G1: A0(0) -> B0(1) -> C0(3)
//          \-> A1(2) /
// Heads: [3]
func setupTestGraphG1(t *testing.T) *CausalGraph {
t.Helper()
cg := CreateCG()
agentA := AgentID("agentA")
agentB := AgentID("agentB")
agentC := AgentID("agentC")

_, err := AddRaw(cg, RawVersion{agentA, 0}, 1, nil) // LV0
if err != nil {
t.Fatalf("G1 setup: AddRaw(A0) failed: %v", err)
}
_, err = AddRaw(cg, RawVersion{agentB, 0}, 1, []RawVersion{{agentA, 0}}) // LV1
if err != nil {
t.Fatalf("G1 setup: AddRaw(B0) failed: %v", err)
}
_, err = AddRaw(cg, RawVersion{agentA, 1}, 1, []RawVersion{{agentA, 0}}) // LV2
if err != nil {
t.Fatalf("G1 setup: AddRaw(A1) failed: %v", err)
}
_, err = AddRaw(cg, RawVersion{agentC, 0}, 1, []RawVersion{{agentB, 0}, {agentA, 1}}) // LV3
if err != nil {
t.Fatalf("G1 setup: AddRaw(C0) failed: %v", err)
}
return cg
}

// setupTestGraphG2 creates another predefined causal graph.
// G2: A0-2(0,1,2) -> B0-1(3,4)
// Heads: [4]
func setupTestGraphG2(t *testing.T) *CausalGraph {
t.Helper()
cg := CreateCG()
agentA := AgentID("agentA")
agentB := AgentID("agentB")

_, err := AddRaw(cg, RawVersion{agentA, 0}, 3, nil) // LVs 0,1,2
if err != nil {
t.Fatalf("G2 setup: AddRaw(A0-2) failed: %v", err)
}
_, err = AddRaw(cg, RawVersion{agentB, 0}, 2, []RawVersion{{agentA, 2}}) // LVs 3,4
if err != nil {
t.Fatalf("G2 setup: AddRaw(B0-1) failed: %v", err)
}
return cg
}

// setupTestGraphG3 creates a linear causal graph.
// G3: A0(0) -> A1(1) -> A2(2)
// Heads: [2]
func setupTestGraphG3(t *testing.T) *CausalGraph {
t.Helper()
cg := CreateCG()
agentA := AgentID("agentA")
_, err := AddRaw(cg, RawVersion{agentA, 0}, 1, nil) // LV0
if err != nil {
t.Fatalf("G3 setup: AddRaw(A0) failed: %v", err)
}
_, err = AddRaw(cg, RawVersion{agentA, 1}, 1, []RawVersion{{agentA, 0}}) // LV1
if err != nil {
t.Fatalf("G3 setup: AddRaw(A1) failed: %v", err)
}
_, err = AddRaw(cg, RawVersion{agentA, 2}, 1, []RawVersion{{agentA, 1}}) // LV2
if err != nil {
t.Fatalf("G3 setup: AddRaw(A2) failed: %v", err)
}
return cg
}

// setupTestGraphG4 creates a graph with independent branches.
// G4: A0(0)
//     B0(1)
// Heads: [0, 1]
func setupTestGraphG4(t *testing.T) *CausalGraph {
t.Helper()
cg := CreateCG()
agentA := AgentID("agentA")
agentB := AgentID("agentB")
// Pass []RawVersion{} to indicate explicit empty parents, not default to heads.
_, err := AddRaw(cg, RawVersion{agentA, 0}, 1, []RawVersion{}) // LV0
if err != nil {
t.Fatalf("G4 setup: AddRaw(A0) failed: %v", err)
}
_, err = AddRaw(cg, RawVersion{agentB, 0}, 1, []RawVersion{}) // LV1
if err != nil {
t.Fatalf("G4 setup: AddRaw(B0) failed: %v", err)
}
return cg
}

func TestDiff(t *testing.T) {
g1 := setupTestGraphG1(t)
g2 := setupTestGraphG2(t)

agentA := AgentID("agentA")
agentB := AgentID("agentB")

tests := []struct {
name        string
cg          *CausalGraph
from        []LV
toSummary   VersionSummary
wantDiff    []LVRange
wantErr     bool
}{
{
name:      "FromG1_Fully_Covered",
cg:        g1,
from:      []LV{0}, // A0
toSummary: VersionSummary{agentA: [][2]int{{0, 1}}},
wantDiff:  []LVRange{},
wantErr:   false,
},
{
name:      "FromG1_One_Item_Not_In_To",
cg:        g1,
from:      []LV{1}, // B0
toSummary: VersionSummary{agentA: [][2]int{{0, 1}}}, // Knows A0
wantDiff:  []LVRange{{Start: 1, End: 2}},           // B0 (LV1)
wantErr:   false,
},
{
name: "FromG1_Complex_Diff_C0_vs_A0",
cg:   g1,
from: []LV{3}, // C0
toSummary: VersionSummary{agentA: [][2]int{{0, 1}}}, // Knows A0
wantDiff: []LVRange{{Start: 1, End: 4}},
wantErr:  false,
},
{
name: "FromG1_Frontier_From_vs_A0_B0",
cg:   g1,
from: []LV{1, 2}, // B0, A1
toSummary: VersionSummary{
agentA: [][2]int{{0, 1}},
agentB: [][2]int{{0, 1}},
},
wantDiff: []LVRange{{Start: 2, End: 3}}, // A1 (LV2)
wantErr:  false,
},
{
name:      "FromG1_Empty_To_Summary",
cg:        g1,
from:      []LV{0}, // A0
toSummary: VersionSummary{},
wantDiff:  []LVRange{{Start: 0, End: 1}},
wantErr:   false,
},
{
name:      "FromG1_Empty_From_Frontier",
cg:        g1,
from:      []LV{},
toSummary: VersionSummary{agentA: [][2]int{{0, 1}}},
wantDiff:  []LVRange{},
wantErr:   false,
},
{
name:     "FromG1_From_Not_In_Graph",
cg:       g1,
from:     []LV{100},
toSummary: VersionSummary{},
wantDiff: nil,
wantErr:  true,
},
{
name: "FromG2_Longer_Entries_Diff",
cg:   g2,
from: []LV{4},
toSummary: VersionSummary{agentA: [][2]int{{0, 2}}},
wantDiff: []LVRange{{Start: 2, End: 5}},
wantErr:  false,
},
{
name: "FromG2_To_Covers_All",
cg:   g2,
from: []LV{4},
toSummary: VersionSummary{
agentA: [][2]int{{0, 3}},
agentB: [][2]int{{0, 2}},
},
wantDiff: []LVRange{},
wantErr:  false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
gotDiff, err := Diff(tt.cg, tt.from, tt.toSummary)
if (err != nil) != tt.wantErr {
t.Errorf("Diff() error = %v, wantErr %v", err, tt.wantErr)
return
}
if !tt.wantErr {
sort.Slice(gotDiff, func(i, j int) bool {
return gotDiff[i].Start < gotDiff[j].Start
})
sort.Slice(tt.wantDiff, func(i, j int) bool {
return tt.wantDiff[i].Start < tt.wantDiff[j].Start
})
compareLVRangeSlices(t, gotDiff, tt.wantDiff)
}
})
}
}

func TestFindDominators(t *testing.T) {
g1 := setupTestGraphG1(t)
g3 := setupTestGraphG3(t)
g4 := setupTestGraphG4(t)

tests := []struct {
name            string
cg              *CausalGraph
versions        []LV
wantDominators  []LV
wantErr         bool
}{
{
name:           "G1_Single_Version_A0",
cg:             g1,
versions:       []LV{0}, // A0
wantDominators: []LV{0},
wantErr:        false,
},
{
name:           "G1_Two_Versions_B0_A1_Common_A0", // Common ancestors {0}, heads of common {0}
cg:             g1,
versions:       []LV{1, 2}, // B0, A1
wantDominators: []LV{0},    // A0
wantErr:        false,
},
{
name:           "G1_Single_Head_C0",
cg:             g1,
versions:       []LV{3}, // C0
wantDominators: []LV{3},
wantErr:        false,
},
{
name:           "G1_C0_and_parent_B0", // Common ancestors {1,0}, heads of common {1}
cg:             g1,
versions:       []LV{3, 1}, // C0, B0
wantDominators: []LV{1},    // B0
wantErr:        false,
},
{
name:           "G1_All_LVs", // Common ancestors {0}, heads of common {0}
cg:             g1,
versions:       []LV{0, 1, 2, 3},
wantDominators: []LV{0}, // A0
wantErr:        false,
},
{
name:           "G1_Empty_Versions_Input",
cg:             g1,
versions:       []LV{},
wantDominators: []LV{},
wantErr:        false,
},
{
name:           "G1_Version_Not_In_Graph",
cg:             g1,
versions:       []LV{0, 100},
wantDominators: nil,
wantErr:        true,
},
{
name:           "G1_Duplicate_Versions_Input_B0_A0", // Common ancestors {0}, heads of common {0}
cg:             g1,
versions:       []LV{1, 1, 0}, // B0, B0, A0
wantDominators: []LV{0},       // A0
wantErr:        false,
},
{
name:           "G3_Linear_Chain_All", // Common {0}, heads {0}
cg:             g3,
versions:       []LV{0, 1, 2},
wantDominators: []LV{0},
wantErr:        false,
},
{
name:           "G3_Tip_of_Linear_Chain",
cg:             g3,
versions:       []LV{2},
wantDominators: []LV{2},
wantErr:        false,
},
{
name:           "G3_Mid_and_Root_of_Chain", // Common {0}, heads {0}
cg:             g3,
versions:       []LV{1, 0},
wantDominators: []LV{0},
wantErr:        false,
},
{
name:           "G4_Independent_Branches_A0_B0", // Common {}, heads {}
cg:             g4,
versions:       []LV{0, 1},
wantDominators: []LV{},
wantErr:        false,
},
{
name:           "G4_Single_from_Independent_A0",
cg:             g4,
versions:       []LV{0},
wantDominators: []LV{0},
wantErr:        false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
gotDominators, err := FindDominators(tt.cg, tt.versions)
if (err != nil) != tt.wantErr {
t.Errorf("FindDominators() error = %v, wantErr %v", err, tt.wantErr)
return
}
if !compareLVSlices(gotDominators, tt.wantDominators) {
t.Errorf("FindDominators() = %v, want %v", gotDominators, tt.wantDominators)
}
})
}
}

func TestFindConflicting(t *testing.T) {
g1 := setupTestGraphG1(t)
// agentA := AgentID("agentA") // Not directly used in table, but good for context
// agentB := AgentID("agentB")
// agentC := AgentID("agentC")

tests := []struct {
name              string
cg                *CausalGraph
versions          []LV
commonAncestors   []LV
wantConflictDiff  []LVRange
wantErr           bool
}{
{
name:            "G1_B0_A1_vs_A0",
cg:              g1,
versions:        []LV{1, 2}, // B0, A1
commonAncestors: []LV{0},    // A0
// Diff([1,2], Summary([0])) -> Diff([1,2], {A0}) -> [1,2] (B0,A1)
// LVRange should be [{1,3}] after merging
wantConflictDiff: []LVRange{{Start: 1, End: 3}},
wantErr:         false,
},
{
name:            "G1_C0_vs_C0_no_conflict",
cg:              g1,
versions:        []LV{3}, // C0
commonAncestors: []LV{3}, // C0
// Diff([3], Summary([3])) -> []
wantConflictDiff: []LVRange{},
wantErr:         false,
},
{
name:            "G1_C0_vs_A0",
cg:              g1,
versions:        []LV{3}, // C0
commonAncestors: []LV{0}, // A0
// Diff([3], Summary([0])) -> Diff([3], {A0}) -> [1,2,3] (B0,A1,C0)
// LVRange should be [{1,4}]
wantConflictDiff: []LVRange{{Start: 1, End: 4}},
wantErr:         false,
},
{
name:            "G1_B0_vs_A1", // B0 is version, A1 is commonAncestor
cg:              g1,
versions:        []LV{1}, // B0
commonAncestors: []LV{2}, // A1
// Diff([1], Summary([2])) -> Diff([1], {A1,A0}) -> [1] (B0)
// LVRange should be [{1,2}]
wantConflictDiff: []LVRange{{Start: 1, End: 2}},
wantErr:         false,
},
{
name:            "G1_Empty_Versions",
cg:              g1,
versions:        []LV{},
commonAncestors: []LV{0},
wantConflictDiff: []LVRange{},
wantErr:         false,
},
{
name:            "G1_Empty_Common_Ancestors",
cg:              g1,
versions:        []LV{1, 2}, // B0, A1
commonAncestors: []LV{},
// Diff([1,2], Summary([])) -> Diff([1,2], {}) -> [0,1,2] (A0,B0,A1)
// LVRange should be [{0,3}]
wantConflictDiff: []LVRange{{Start: 0, End: 3}},
wantErr:         false,
},
{
name:            "G1_Version_Not_In_Graph",
cg:              g1,
versions:        []LV{100},
commonAncestors: []LV{0},
wantConflictDiff: nil, // Diff will error
wantErr:         true,
},
{
name:            "G1_Ancestor_Not_In_Graph",
cg:              g1,
versions:        []LV{1},
commonAncestors: []LV{100}, // SummarizeVersion will error
wantConflictDiff: nil,
wantErr:         true,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
gotDiff, err := FindConflicting(tt.cg, tt.versions, tt.commonAncestors)
if (err != nil) != tt.wantErr {
t.Errorf("FindConflicting() error = %v, wantErr %v", err, tt.wantErr)
return
}
if !tt.wantErr {
// Diff results are sorted and merged by the Diff function itself.
// So, direct comparison should be fine if wantConflictDiff is also sorted and merged.
compareLVRangeSlices(t, gotDiff, tt.wantConflictDiff)
}
})
}
}

func TestCompareVersions(t *testing.T) {
g1 := setupTestGraphG1(t)
g4 := setupTestGraphG4(t)

tests := []struct {
name         string
cg           *CausalGraph
a, b         LV
wantRelation Relation
wantErr      bool
}{
{name: "G1_Equal_B0_B0", cg: g1, a: 1, b: 1, wantRelation: RelationEqual},
{name: "G1_Ancestor_A0_B0", cg: g1, a: 0, b: 1, wantRelation: RelationAncestor},
{name: "G1_Ancestor_A0_C0", cg: g1, a: 0, b: 3, wantRelation: RelationAncestor},
{name: "G1_Ancestor_B0_C0", cg: g1, a: 1, b: 3, wantRelation: RelationAncestor},
{name: "G1_Descendant_B0_A0", cg: g1, a: 1, b: 0, wantRelation: RelationDescendant},
{name: "G1_Descendant_C0_A0", cg: g1, a: 3, b: 0, wantRelation: RelationDescendant},
{name: "G1_Descendant_C0_B0", cg: g1, a: 3, b: 1, wantRelation: RelationDescendant},
{name: "G1_Concurrent_B0_A1", cg: g1, a: 1, b: 2, wantRelation: RelationConcurrent},
{name: "G4_Concurrent_A0_B0", cg: g4, a: 0, b: 1, wantRelation: RelationConcurrent},
{name: "G1_LV_A_Not_In_Graph", cg: g1, a: 100, b: 0, wantErr: true},
{name: "G1_LV_B_Not_In_Graph", cg: g1, a: 0, b: 100, wantErr: true},
{name: "G1_Both_LVs_Not_In_Graph", cg: g1, a: 100, b: 101, wantErr: true},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
gotRelation, err := CompareVersions(tt.cg, tt.a, tt.b)
if (err != nil) != tt.wantErr {
t.Errorf("CompareVersions(%d, %d) error = %v, wantErr %v", tt.a, tt.b, err, tt.wantErr)
return
}
if !tt.wantErr && gotRelation != tt.wantRelation {
t.Errorf("CompareVersions(%d, %d) = %v, want %v", tt.a, tt.b, gotRelation, tt.wantRelation)
}
})
}
}

type IteratedItem struct {
LV             LV
IsParentOfPrev bool
IsMerge        bool
}

func TestIterVersionsBetween(t *testing.T) {
g1 := setupTestGraphG1(t) // A0(0) -> B0(1), A0(0) -> A1(2), (B0(1),A1(2)) -> C0(3)
g2 := setupTestGraphG2(t) // A0-2(0,1,2) -> B0-1(3,4)

tests := []struct {
name          string
cg            *CausalGraph
from          []LV
to            LV
stopAtLV      LV // LV at which the callback should return stop=true, or -1 to not stop
wantIteration []IteratedItem
wantErr       bool
}{
{
name:          "G1_Simple_A0_to_B0",
cg:            g1,
from:          []LV{0}, // A0
to:            1,       // B0
stopAtLV:      -1,
wantIteration: []IteratedItem{{LV: 1, IsParentOfPrev: false, IsMerge: false}},
},
{
name:     "G1_A0_to_C0_full_history",
cg:       g1,
from:     []LV{0}, // A0
to:       3,       // C0
stopAtLV: -1,
// C0(3) parents [1,2]. Traversal visits parents in reverse: 2 then 1.
// fn(3, false, true) -> queue (2,false), (1,true)
// fn(1, true, false) -> parent 0 (in from)
// fn(2, false, false) -> parent 0 (in from)
wantIteration: []IteratedItem{
{LV: 3, IsParentOfPrev: false, IsMerge: true},
{LV: 1, IsParentOfPrev: true, IsMerge: false},
{LV: 2, IsParentOfPrev: false, IsMerge: false},
},
},
{
name:     "G1_B0_to_C0",
cg:       g1,
from:     []LV{1}, // B0
to:       3,       // C0
stopAtLV: -1,
// fn(3,f,T) -> parents [1,2]. 1 is in from. queue (2,f)
// fn(2,f,F) -> parent 0.
// Is 0 covered by from=[1]? No. So 0 is visited.
// Wait, IterVersionsBetweenBP adds `from` to `visited`.
// So, fn(3,f,T), parents [1,2]. 1 is in `from`/`visited`. Queue (2,f).
// fn(2,f,F), parent 0. Is 0 covered by `from`? No.
// The `from` check is `if _, ok := visited[v]; ok { continue }`
// And `for _, fv := range from { visited[fv] = struct{}{} }`
// So, if 0 is parent of 2, and 0 is not in `from`=[1], it will be visited.
// Expected: C0(3) -> A1(2) -> A0(0)
// Let's trace:
// iter(from=[1], to=3)
// visited = {1: {}}
// queue = [(3,f)]
// item = (3,f). fn(3,f,T). visited[3]={}. parents of 3 are [1,2].
//   parent 2: not in visited. queue = [(2,f)]
//   parent 1: in visited. skip.
// item = (2,f). fn(2,f,F). visited[2]={}. parents of 2 are [0].
//   parent 0: not in visited. queue = [(0,t)]
// item = (0,t). fn(0,t,F). visited[0]={}. parents of 0 are [].
// Result: [(3,f,T), (2,f,F), (0,t,F)]
wantIteration: []IteratedItem{
{LV: 3, IsParentOfPrev: false, IsMerge: true},
{LV: 2, IsParentOfPrev: false, IsMerge: false}, // A1 is the 2nd parent of C0, B0(1) is first.
{LV: 0, IsParentOfPrev: true, IsMerge: false},
},
},
{
name:     "G1_A1_to_C0",
cg:       g1,
from:     []LV{2}, // A1
to:       3,       // C0
stopAtLV: -1,
// iter(from=[2], to=3)
// visited = {2: {}}
// queue = [(3,f)]
// item = (3,f). fn(3,f,T). visited[3]={}. parents of 3 are [1,2].
//   parent 2: in visited. skip.
//   parent 1: not in visited. queue = [(1,t)]
// item = (1,t). fn(1,t,F). visited[1]={}. parents of 1 are [0].
//   parent 0: not in visited. queue = [(0,t)]
// item = (0,t). fn(0,t,F). visited[0]={}. parents of 0 are [].
// Result: [(3,f,T), (1,t,F), (0,t,F)]
wantIteration: []IteratedItem{
{LV: 3, IsParentOfPrev: false, IsMerge: true},
{LV: 1, IsParentOfPrev: true, IsMerge: false},
{LV: 0, IsParentOfPrev: true, IsMerge: false},
},
},
{
name:          "G1_Multiple_from_B0A1_to_C0",
cg:            g1,
from:          []LV{1, 2}, // B0, A1
to:            3,          // C0
stopAtLV:      -1,
wantIteration: []IteratedItem{{LV: 3, IsParentOfPrev: false, IsMerge: true}},
},
{
name:          "G1_from_equals_to",
cg:            g1,
from:          []LV{0},
to:            0,
stopAtLV:      -1,
wantIteration: nil, // Changed from []IteratedItem{}
},
{
name:          "G1_to_is_ancestor_of_from",
cg:            g1,
from:          []LV{1}, // B0
to:            0,       // A0
stopAtLV:      -1,
wantIteration: nil, // Changed from []IteratedItem{}, IterVersionsBetween has a check for this
},
{
name:     "G1_Empty_from_to_B0",
cg:       g1,
from:     []LV{},
to:       1, // B0
stopAtLV: -1,
wantIteration: []IteratedItem{
{LV: 1, IsParentOfPrev: false, IsMerge: false},
{LV: 0, IsParentOfPrev: true, IsMerge: false},
},
},
{
name:          "G2_A2_to_B0", // from=[2], to=3
cg:            g2,
from:          []LV{2}, // A2 (part of A0-2 span)
to:            3,       // B0 (part of B0-1 span)
stopAtLV:      -1,
wantIteration: []IteratedItem{{LV: 3, IsParentOfPrev: false, IsMerge: false}},
},
{
name:     "G2_A2_to_B1", // from=[2], to=4
cg:       g2,
from:     []LV{2}, // A2
to:       4,       // B1
stopAtLV: -1,
// iter(from=[2], to=4)
// visited={2}
// q=[(4,f)]
// item=(4,f). fn(4,f,F). v[4]. p=[3]. q=[(3,t)]
// item=(3,t). fn(3,t,F). v[3]. p=[2]. 2 is in visited.
// Result: [(4,f,F), (3,t,F)]
wantIteration: []IteratedItem{
{LV: 4, IsParentOfPrev: false, IsMerge: false},
{LV: 3, IsParentOfPrev: true, IsMerge: false},
},
},
{
name:     "G2_A0_to_B1", // from=[0], to=4
cg:       g2,
from:     []LV{0}, // A0
to:       4,       // B1
stopAtLV: -1,
// iter(from=[0], to=4)
// visited={0}
// q=[(4,f)]
// item=(4,f). fn(4,f,F). v[4]. p=[3]. q=[(3,t)]
// item=(3,t). fn(3,t,F). v[3]. p=[2]. q=[(2,t),(3,t)] -> q=[(2,t)]
// item=(2,t). fn(2,t,F). v[2]. p=[1]. q=[(1,t)]
// item=(1,t). fn(1,t,F). v[1]. p=[0]. 0 is in visited.
// Result: [(4,f,F), (3,t,F), (2,t,F), (1,t,F)]
wantIteration: []IteratedItem{
{LV: 4, IsParentOfPrev: false, IsMerge: false},
{LV: 3, IsParentOfPrev: true, IsMerge: false},
{LV: 2, IsParentOfPrev: true, IsMerge: false},
{LV: 1, IsParentOfPrev: true, IsMerge: false},
},
},
{
name:          "G1_Stop_Iteration_Early_At_C0",
cg:            g1,
from:          []LV{0},
to:            3,
stopAtLV:      3, // Stop at C0 itself
wantIteration: []IteratedItem{{LV: 3, IsParentOfPrev: false, IsMerge: true}},
},
{
name:    "G1_to_LV_Not_In_Graph",
cg:      g1,
from:    []LV{0},
to:      100,
wantErr: true,
},
{
name:    "G1_from_LV_Not_In_Graph",
cg:      g1,
from:    []LV{100},
to:      1,
wantErr: true, // VersionContainsLV in IterVersionsBetween will fail
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
var iteratedItems []IteratedItem
callback := func(v LV, isParentOfPrev bool, isMerge bool) (stop bool, err error) {
iteratedItems = append(iteratedItems, IteratedItem{v, isParentOfPrev, isMerge})
if tt.stopAtLV != -1 && v == tt.stopAtLV {
return true, nil
}
return false, nil
}

err := IterVersionsBetween(tt.cg, tt.from, tt.to, callback)

if (err != nil) != tt.wantErr {
t.Errorf("IterVersionsBetween() error = %v, wantErr %v", err, tt.wantErr)
return
}
if !tt.wantErr {
if !reflect.DeepEqual(iteratedItems, tt.wantIteration) {
t.Errorf("IterVersionsBetween() iteration order mismatch:\ngot:  %+v\nwant: %+v", iteratedItems, tt.wantIteration)
}
}
})
}
}

// TODO: Add more tests for:
// - IntersectWithSummary / IntersectWithSummaryFull (more edge cases, e.g., empty graph, summary with unknown agents/out-of-bounds seqs)
// - Edge cases for other functions (empty inputs, invalid inputs where appropriate)

func TestIntersectWithSummaryFull(t *testing.T) {
agentA := AgentID("agentA")
	agentB := AgentID("agentB")
	agentC := AgentID("agentC")

	g1 := setupTestGraphG1(t) // A0(0) -> B0(1), A0(0) -> A1(2), (B0(1),A1(2)) -> C0(3). Heads: [3]

	tests := []struct {
		name    string
		cg      *CausalGraph
		summary VersionSummary
		want    []CGEntry
		wantErr bool
	}{
		{
			name:    "G1_Empty_Summary",
			cg:      g1,
			summary: VersionSummary{},
			want: []CGEntry{
				{Agent: agentA, Seq: 0, Version: 0, VEnd: 1, Parents: []LV{}},
				{Agent: agentB, Seq: 0, Version: 1, VEnd: 2, Parents: []LV{0}},
				{Agent: agentA, Seq: 1, Version: 2, VEnd: 3, Parents: []LV{0}},
{Agent: agentC, Seq: 0, Version: 3, VEnd: 4, Parents: []LV{1, 2}},
},
wantErr: false,
},
{
name: "G1_Full_Summary",
cg:   g1,
summary: VersionSummary{
agentA: [][2]int{{0, 2}}, // Covers A0 (seq 0), A1 (seq 1)
agentB: [][2]int{{0, 1}}, // Covers B0 (seq 0)
agentC: [][2]int{{0, 1}}, // Covers C0 (seq 0)
},
want:    []CGEntry{},
wantErr: false,
},
{
name: "G1_Partial_Summary_Covers_A0",
cg:   g1,
summary: VersionSummary{
agentA: [][2]int{{0, 1}}, // Covers A0 (seq 0)
},
want: []CGEntry{
// A0 is covered by summary.
// B0 depends on A0.
// A1 depends on A0.
// C0 depends on B0, A1.
// Expected: B0, A1, C0
{Agent: agentB, Seq: 0, Version: 1, VEnd: 2, Parents: []LV{0}},
{Agent: agentA, Seq: 1, Version: 2, VEnd: 3, Parents: []LV{0}},
{Agent: agentC, Seq: 0, Version: 3, VEnd: 4, Parents: []LV{1, 2}},
},
wantErr: false,
},
{
name:    "G1_Empty_Graph_IntersectFull",
cg:      CreateCG(),
summary: VersionSummary{agentA: [][2]int{{0, 1}}},
want:    []CGEntry{},
wantErr: false, // Should not error, just return no entries
},
{
name: "G1_Summary_With_Unknown_Agent_IntersectFull",
cg:   g1,
summary: VersionSummary{
AgentID("unknownAgent"): [][2]int{{0, 1}},
agentA:                  [][2]int{{0, 1}}, // Covers A0
},
// Should ignore unknownAgent and process agentA.
// A0 is covered. B0, A1, C0 remain.
want: []CGEntry{
{Agent: agentB, Seq: 0, Version: 1, VEnd: 2, Parents: []LV{0}},
{Agent: agentA, Seq: 1, Version: 2, VEnd: 3, Parents: []LV{0}},
{Agent: agentC, Seq: 0, Version: 3, VEnd: 4, Parents: []LV{1, 2}},
},
wantErr: false,
},
{
name: "G1_Summary_With_OutOfBounds_Seq_IntersectFull",
cg:   g1,
summary: VersionSummary{
agentA: [][2]int{{0, 1}, {5, 10}}, // Covers A0, then an out-of-bounds range
},
// Should process the valid range for A0. Out-of-bounds should be ignored or handled gracefully.
// A0 is covered. B0, A1, C0 remain.
want: []CGEntry{
{Agent: agentB, Seq: 0, Version: 1, VEnd: 2, Parents: []LV{0}},
{Agent: agentA, Seq: 1, Version: 2, VEnd: 3, Parents: []LV{0}},
{Agent: agentC, Seq: 0, Version: 3, VEnd: 4, Parents: []LV{1, 2}},
},
wantErr: false,
},
}

g2 := setupTestGraphG2(t) // A0-2(0,1,2) -> B0-1(3,4). Heads: [4]
testsG2 := []struct {
name    string
cg      *CausalGraph
summary VersionSummary
want    []CGEntry
wantErr bool
}{
{
name:    "G2_Empty_Summary",
cg:      g2,
summary: VersionSummary{},
want: []CGEntry{
{Agent: agentA, Seq: 0, Version: 0, VEnd: 3, Parents: []LV{}}, // A0-2
{Agent: agentB, Seq: 0, Version: 3, VEnd: 5, Parents: []LV{2}}, // B0-1
},
wantErr: false,
},
{
name: "G2_Full_Summary",
cg:   g2,
summary: VersionSummary{
agentA: [][2]int{{0, 3}}, // Covers A0-2
agentB: [][2]int{{0, 2}}, // Covers B0-1
},
want:    []CGEntry{},
wantErr: false,
},
{
name: "G2_Partial_Summary_Covers_A0_A1",
cg:   g2,
summary: VersionSummary{
agentA: [][2]int{{0, 2}}, // Covers A0, A1 (seq 0, 1 of agentA)
},
want: []CGEntry{
// A0-2 is (A0,A1,A2). Summary covers A0,A1. A2 (LV 2) remains.
{Agent: agentA, Seq: 2, Version: 2, VEnd: 3, Parents: []LV{}},
// B0-1 depends on A2 (LV 2). Since A2 is not covered, B0-1 is included.
{Agent: agentB, Seq: 0, Version: 3, VEnd: 5, Parents: []LV{2}},
},
wantErr: false,
},
{
name: "G2_Partial_Summary_Covers_A2",
cg:   g2,
summary: VersionSummary{
agentA: [][2]int{{2, 3}}, // Covers A2 (seq 2 of agentA)
},
want: []CGEntry{
// A0-2 is (A0,A1,A2). Summary covers A2. A0,A1 (LVs 0,1) remain.
{Agent: agentA, Seq: 0, Version: 0, VEnd: 2, Parents: []LV{}},
// B0-1 is not covered by summary for agentB.
{Agent: agentB, Seq: 0, Version: 3, VEnd: 5, Parents: []LV{2}},
},
wantErr: false,
},
{
name: "G2_Partial_Summary_Covers_B0",
cg:   g2,
summary: VersionSummary{
agentB: [][2]int{{0, 1}}, // Covers B0 (seq 0 of agentB)
},
want: []CGEntry{
// A0-2 is not covered by summary for agentA.
{Agent: agentA, Seq: 0, Version: 0, VEnd: 3, Parents: []LV{}},
// B0-1 is (B0,B1). Summary covers B0. B1 (LV 4) remains.
// Original B0-1: Agent B, Seq 0, V3, VEnd 5, Parents [2]
// Remaining B1: Agent B, Seq 1, V4, VEnd 5, Parents [2]
{Agent: agentB, Seq: 1, Version: 4, VEnd: 5, Parents: []LV{2}},
},
wantErr: false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
got, err := IntersectWithSummaryFull(tt.cg, tt.summary)
if (err != nil) != tt.wantErr {
t.Errorf("IntersectWithSummaryFull() error = %v, wantErr %v", err, tt.wantErr)
return
}
compareCGEntrySlices(t, got, tt.want)
})
}

for _, tt := range testsG2 {
t.Run(tt.name, func(t *testing.T) {
got, err := IntersectWithSummaryFull(tt.cg, tt.summary)
if (err != nil) != tt.wantErr {
t.Errorf("IntersectWithSummaryFull() error = %v, wantErr %v", err, tt.wantErr)
return
}
compareCGEntrySlices(t, got, tt.want)
})
}
}
t.Run(tt.name, func(t *testing.T) {
got, err := IntersectWithSummaryFull(tt.cg, tt.summary)
if (err != nil) != tt.wantErr {
t.Errorf("IntersectWithSummaryFull() error = %v, wantErr %v", err, tt.wantErr)
return
}
compareCGEntrySlices(t, got, tt.want)
})
}
}

func TestIntersectWithSummary(t *testing.T) {
agentA := AgentID("agentA")
agentB := AgentID("agentB")
agentC := AgentID("agentC")

g1 := setupTestGraphG1(t) // A0(0) -> B0(1), A0(0) -> A1(2), (B0(1),A1(2)) -> C0(3). Heads: [3]
// Entries: A0(LV0), B0(LV1), A1(LV2), C0(LV3)

testsG1 := []struct {
name    string
cg      *CausalGraph
summary VersionSummary
want    []LVRange
wantErr bool
}{
{
name:    "G1_Empty_Summary_Intersect",
cg:      g1,
summary: VersionSummary{},
want:    []LVRange{{Start: 0, End: 4}}, // All LVs: 0,1,2,3
wantErr: false,
},
{
name: "G1_Full_Summary_Intersect",
cg:   g1,
summary: VersionSummary{
agentA: [][2]int{{0, 2}}, // Covers A0 (seq 0, LV0), A1 (seq 1, LV2)
agentB: [][2]int{{0, 1}}, // Covers B0 (seq 0, LV1)
agentC: [][2]int{{0, 1}}, // Covers C0 (seq 0, LV3)
},
want:    []LVRange{},
wantErr: false,
},
{
name: "G1_Partial_Summary_Covers_A0_Intersect",
cg:   g1,
summary: VersionSummary{
agentA: [][2]int{{0, 1}}, // Covers A0 (seq 0, LV0)
},
// A0(LV0) covered. B0(LV1), A1(LV2), C0(LV3) not covered.
want:    []LVRange{{Start: 1, End: 4}}, // LVs 1,2,3
wantErr: false,
},
{
name: "G1_Partial_Summary_Covers_B0_Intersect",
cg:   g1,
summary: VersionSummary{
agentB: [][2]int{{0, 1}}, // Covers B0 (seq 0, LV1)
},
// A0(LV0) not covered. B0(LV1) covered. A1(LV2) not covered. C0(LV3) not covered.
// Remaining LVs: 0, 2, 3
want:    []LVRange{{Start: 0, End: 1}, {Start: 2, End: 4}},
wantErr: false,
},
{
name: "G1_Partial_Summary_Covers_A0_B0_Intersect",
cg:   g1,
summary: VersionSummary{
agentA: [][2]int{{0, 1}}, // Covers A0 (seq 0, LV0)
agentB: [][2]int{{0, 1}}, // Covers B0 (seq 0, LV1)
},
// A0(LV0) covered. B0(LV1) covered. A1(LV2) not covered. C0(LV3) not covered.
// Remaining LVs: 2, 3
want:    []LVRange{{Start: 2, End: 4}},
wantErr: false,
},
{
name:    "G1_Empty_Graph_Intersect",
cg:      CreateCG(),
summary: VersionSummary{agentA: [][2]int{{0, 1}}},
want:    []LVRange{},
wantErr: false,
},
{
name: "G1_Summary_With_Unknown_Agent_Intersect",
cg:   g1,
summary: VersionSummary{
AgentID("unknownAgent"): [][2]int{{0, 1}},
agentA:                  [][2]int{{0, 1}}, // Covers A0 (LV0)
},
// Should ignore unknownAgent. A0 covered. B0(LV1), A1(LV2), C0(LV3) not covered.
want:    []LVRange{{Start: 1, End: 4}}, // LVs 1,2,3
wantErr: false,
},
{
name: "G1_Summary_With_OutOfBounds_Seq_Intersect",
cg:   g1,
summary: VersionSummary{
agentA: [][2]int{{0, 1}, {5, 10}}, // Covers A0 (LV0), then an out-of-bounds range
},
// Should process valid range for A0. A0 covered. B0(LV1), A1(LV2), C0(LV3) not covered.
want:    []LVRange{{Start: 1, End: 4}}, // LVs 1,2,3
wantErr: false,
},
}

for _, tt := range testsG1 {
t.Run(tt.name, func(t *testing.T) {
got, err := IntersectWithSummary(tt.cg, tt.summary)
if (err != nil) != tt.wantErr {
t.Errorf("IntersectWithSummary() error = %v, wantErr %v", err, tt.wantErr)
return
}
compareLVRangeSlices(t, got, tt.want)
})
}

g2 := setupTestGraphG2(t) // A0-2(0,1,2) -> B0-1(3,4). Heads: [4]
// Entries: A0-2 (LVs 0,1,2), B0-1 (LVs 3,4)

testsG2 := []struct {
name    string
cg      *CausalGraph
summary VersionSummary
want    []LVRange
wantErr bool
}{
{
name:    "G2_Empty_Summary_Intersect",
cg:      g2,
summary: VersionSummary{},
want:    []LVRange{{Start: 0, End: 5}}, // All LVs: 0,1,2,3,4
wantErr: false,
},
{
name: "G2_Full_Summary_Intersect",
cg:   g2,
summary: VersionSummary{
agentA: [][2]int{{0, 3}}, // Covers A0-2 (LVs 0,1,2)
agentB: [][2]int{{0, 2}}, // Covers B0-1 (LVs 3,4)
},
want:    []LVRange{},
wantErr: false,
},
{
name: "G2_Partial_Summary_Covers_A0_A1_Intersect",
cg:   g2,
summary: VersionSummary{
agentA: [][2]int{{0, 2}}, // Covers A0,A1 (LVs 0,1 which are seq 0,1 for agentA)
},
// A0-2 (LVs 0,1,2): LVs 0,1 covered. LV 2 (seq 2 for agentA) not.
// B0-1 (LVs 3,4): Not covered.
// Remaining: LV 2, LVs 3,4
want:    []LVRange{{Start: 2, End: 5}},
wantErr: false,
},
{
name: "G2_Partial_Summary_Covers_A2_Intersect",
cg:   g2,
summary: VersionSummary{
agentA: [][2]int{{2, 3}}, // Covers A2 (LV 2 which is seq 2 for agentA)
},
// A0-2 (LVs 0,1,2): LV 2 covered. LVs 0,1 (seq 0,1 for agentA) not.
// B0-1 (LVs 3,4): Not covered.
// Remaining: LVs 0,1 and LVs 3,4
want:    []LVRange{{Start: 0, End: 2}, {Start: 3, End: 5}},
wantErr: false,
},
{
name: "G2_Partial_Summary_Covers_B0_Intersect",
cg:   g2,
summary: VersionSummary{
agentB: [][2]int{{0, 1}}, // Covers B0 (LV 3 which is seq 0 for agentB)
},
// A0-2 (LVs 0,1,2): Not covered.
// B0-1 (LVs 3,4): LV 3 covered. LV 4 (seq 1 for agentB) not.
// Remaining: LVs 0,1,2 and LV 4
want:    []LVRange{{Start: 0, End: 3}, {Start: 4, End: 5}},
wantErr: false,
},
}

for _, tt := range testsG2 {
t.Run(tt.name, func(t *testing.T) {
got, err := IntersectWithSummary(tt.cg, tt.summary)
if (err != nil) != tt.wantErr {
t.Errorf("IntersectWithSummary() error = %v, wantErr %v", err, tt.wantErr)
return
}
compareLVRangeSlices(t, got, tt.want)
})
}
}
