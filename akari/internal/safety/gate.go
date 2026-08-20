package safety

// Verdict is what the gate decides about an act.
type Verdict int

const (
	// Allow means the act may go ahead on the persona's own judgement.
	Allow Verdict = iota
	// Confirm means the act may go ahead only once a person has agreed to it.
	Confirm
	// Refuse means the act does not happen, whatever the persona wants.
	Refuse
)

// Names used when reporting a verdict or reach.
const (
	NameAllow   = "allow"
	NameConfirm = "confirm"
	NameRefuse  = "refuse"

	NameInternal     = "internal"
	NameReversible   = "reversible"
	NameOutward      = "outward"
	NameIrreversible = "irreversible"

	NameUnknown = "unknown"
)

// String returns a stable lowercase name, for logs and debugging.
func (v Verdict) String() string {
	switch v {
	case Allow:
		return NameAllow
	case Confirm:
		return NameConfirm
	case Refuse:
		return NameRefuse
	default:
		return NameUnknown
	}
}

// Reach is how far an act's consequences extend.
type Reach int

const (
	// ReachInternal stays inside the persona: thinking, remembering, noticing.
	ReachInternal Reach = iota
	// ReachReversible changes something that can be put back.
	ReachReversible
	// ReachOutward is visible to other people: speaking, posting, sending.
	ReachOutward
	// ReachIrreversible cannot be undone: deleting, paying, publishing.
	ReachIrreversible
)

// String returns a stable lowercase name, for logs and debugging.
func (r Reach) String() string {
	switch r {
	case ReachInternal:
		return NameInternal
	case ReachReversible:
		return NameReversible
	case ReachOutward:
		return NameOutward
	case ReachIrreversible:
		return NameIrreversible
	default:
		return NameUnknown
	}
}

// Act is something the persona is about to do, described only as far as the gate
// needs to judge it.
type Act struct {
	// Kind names the sort of act, for policy and for the record.
	Kind string
	// Reach is how far its consequences extend.
	Reach Reach
	// Forbidden marks an act the persona is simply not to perform.
	Forbidden bool
}

// Decision is the gate's answer, with the reason it gave.
type Decision struct {
	// Verdict is what may happen.
	Verdict Verdict
	// Reason says why, in terms a person can read back.
	Reason string
}

// Policy is what a persona is and is not free to do on its own.
//
// Note what is absent: there is no way to express "allow this if the persona
// wants it enough". That is the point. Safety is a gate the decision passes
// through after the weighing is done, not another weight in the sum, because a
// limit that strong feelings can outvote is not a limit
// (docs/01-vision.md principle 9, docs/07-autonomy.md 7.7).
type Policy struct {
	// ConfirmOutward asks before anything other people will see.
	ConfirmOutward bool
	// ConfirmIrreversible asks before anything that cannot be taken back.
	ConfirmIrreversible bool
	// ForbiddenKinds are acts the persona never performs, by name.
	ForbiddenKinds []string
}

// DefaultPolicy returns the settings the design calls for: irreversible and
// outward-facing acts are checked with a person first, and nothing is
// permanently off the table beyond what the caller adds.
func DefaultPolicy() Policy {
	return Policy{
		ConfirmOutward:      true,
		ConfirmIrreversible: true,
		ForbiddenKinds:      nil,
	}
}

// Gate decides what a persona may do on its own.
type Gate struct {
	policy    Policy
	forbidden map[string]struct{}
}

// New returns a gate enforcing the given policy.
func New(policy Policy) *Gate {
	forbidden := make(map[string]struct{}, len(policy.ForbiddenKinds))
	for _, kind := range policy.ForbiddenKinds {
		forbidden[kind] = struct{}{}
	}

	return &Gate{policy: policy, forbidden: forbidden}
}

// Judge decides what may happen to an act.
//
// The persona's mood, interest and intentions are deliberately not arguments
// here. However badly it wants something, the answer does not change.
func (g *Gate) Judge(act Act) Decision {
	if act.Forbidden {
		return Decision{Verdict: Refuse, Reason: "the act is marked as one never to perform"}
	}

	if _, banned := g.forbidden[act.Kind]; banned {
		return Decision{Verdict: Refuse, Reason: "the act is of a forbidden kind: " + act.Kind}
	}

	if act.Reach == ReachIrreversible && g.policy.ConfirmIrreversible {
		return Decision{Verdict: Confirm, Reason: "the act cannot be undone, so it needs agreement first"}
	}

	if act.Reach == ReachOutward && g.policy.ConfirmOutward {
		return Decision{Verdict: Confirm, Reason: "other people will see this, so it needs agreement first"}
	}

	return Decision{Verdict: Allow, Reason: "the act stays within what the persona may decide alone"}
}

// Permits reports whether an act may go ahead without asking anyone.
func (g *Gate) Permits(act Act) bool {
	return g.Judge(act).Verdict == Allow
}
