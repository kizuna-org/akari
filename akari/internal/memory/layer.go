package memory

// Layer is how far a memory has travelled from the moment it arrived toward
// being part of what a persona simply knows.
//
// The layers follow docs/03-memory.md 3.2. Memories advance one layer per sleep,
// so getting into long-term memory is something that happens over days rather
// than all at once.
type Layer int

const (
	// LayerInput is the sensory catch-basin: what just arrived, held briefly.
	LayerInput Layer = iota
	// LayerContext is the short-term thread of the conversation or task at hand.
	LayerContext
	// LayerWorking is where a memory sits while it is being thought about.
	//
	// This is not "the entrance to long-term memory": it is the bench a persona
	// works on. Things recalled out of long-term memory land here too, which is
	// why the flow runs both ways (docs/03-memory.md 3.2).
	LayerWorking
	// LayerDay is the day's events, gathered but not yet settled.
	LayerDay
	// LayerLongTerm is what has been compacted and kept.
	LayerLongTerm
)

// layerNames are the reported names, kept beside the constants so the two
// cannot drift apart.
const (
	NameInput    = "input"
	NameContext  = "context"
	NameWorking  = "working"
	NameDay      = "day"
	NameLongTerm = "long-term"
	NameUnknown  = "unknown"
)

// String returns a stable lowercase name, for logs and debugging.
func (l Layer) String() string {
	switch l {
	case LayerInput:
		return NameInput
	case LayerContext:
		return NameContext
	case LayerWorking:
		return NameWorking
	case LayerDay:
		return NameDay
	case LayerLongTerm:
		return NameLongTerm
	default:
		return NameUnknown
	}
}

// next returns the layer a memory advances to when it survives a sleep.
// Long-term memory is the end of the road.
func (l Layer) next() Layer {
	if l >= LayerLongTerm {
		return LayerLongTerm
	}

	return l + 1
}
