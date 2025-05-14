package causalgraph

// AgentID is a type alias for agent identifiers.
type AgentID string

// RawVersion represents a version identifier as a [agent, seq] pair.
type RawVersion struct {
Agent AgentID
Seq   int
}

// LV (Local Version) is a local, auto-incremented ID per known version.
type LV int

// LVRange represents a local version range [Start, End).
type LVRange struct {
Start LV
End   LV
}

// CGEntry stores metadata for a run of versions in the causal graph.
type CGEntry struct {
Version LV      // Starting LV of this entry.
VEnd    LV      // Ending LV (exclusive) of this entry.
Agent   AgentID // Agent ID for this run.
Seq     int     // Starting sequence number for this run.
Parents []LV    // Parent LVs for the first version in this entry.
}

// ClientEntry stores metadata for a run of versions by a specific client.
type ClientEntry struct {
Seq     int // Starting sequence number of this run.
SeqEnd  int // Ending sequence number (exclusive) of this run.
Version LV  // LV of the first item in this run.
}

// CausalGraph holds the entire causal graph structure.
type CausalGraph struct {
// Heads stores the current global version frontier as LVs.
Heads []LV
// Entries maps local versions to their raw version and parent information.
// Stored in runs, sorted by LV.
Entries []CGEntry
// AgentToVersion maps an agent ID to a list of ClientEntry runs by that agent.
// Sorted by sequence number.
AgentToVersion map[AgentID][]ClientEntry
// NextLV is the next available local version to assign.
NextLV LV
}

// VersionSummary is a map from agent ID to a list of [start_seq, end_seq) ranges.
type VersionSummary map[AgentID][][2]int
