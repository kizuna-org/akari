package emotion

// Kind identifies one of the basic emotions a persona can feel.
//
// The vocabulary follows the human basic emotions named in the design docs
// (docs/02-emotion.md): four pleasant kinds and five unpleasant ones.
// Unpleasant kinds are first-class here on purpose: a persona that can only
// feel good is not a persona.
type Kind int

const (
	// Joy is the pleasant reaction to something good happening.
	Joy Kind = iota
	// Fun is the pleasant reaction to an engaging moment.
	Fun
	// Relief is the pleasant reaction to a worry lifting.
	Relief
	// Anticipation is the pleasant pull of something good ahead.
	Anticipation
	// Anger is the unpleasant reaction to being obstructed or wronged.
	Anger
	// Sadness is the unpleasant reaction to a loss.
	Sadness
	// Anxiety is the unpleasant reaction to a threat that may be coming.
	Anxiety
	// Boredom is the unpleasant flatness of nothing worth attending to.
	Boredom
	// Surprise is the jolt of the unexpected, pleasant or not.
	Surprise
)

// kindCount is the number of distinct emotion kinds.
const kindCount = int(Surprise) + 1

// Names used when reporting a kind or tendency, kept as constants so the
// implementation and its tests cannot drift apart.
const (
	NameJoy          = "joy"
	NameFun          = "fun"
	NameRelief       = "relief"
	NameAnticipation = "anticipation"
	NameAnger        = "anger"
	NameSadness      = "sadness"
	NameAnxiety      = "anxiety"
	NameBoredom      = "boredom"
	NameSurprise     = "surprise"

	NameNone   = "none"
	NameFreeze = "freeze"
	NameAvoid  = "avoid"

	NameUnknown = "unknown"
)

// Tendency is a readiness to act that an emotion produces by itself, without
// waiting on interest. It is the fast direct path of docs/02-emotion.md 2.1.
type Tendency int

const (
	// TendencyNone means the emotion produces no reflex of its own.
	TendencyNone Tendency = iota
	// TendencyFreeze is stopping mid-action to take stock.
	TendencyFreeze
	// TendencyAvoid is pulling back from whatever caused the feeling.
	TendencyAvoid
)

// String returns a stable lowercase name, for logs and debugging.
func (k Kind) String() string {
	if !k.valid() {
		return NameUnknown
	}

	names := [kindCount]string{
		NameJoy,
		NameFun,
		NameRelief,
		NameAnticipation,
		NameAnger,
		NameSadness,
		NameAnxiety,
		NameBoredom,
		NameSurprise,
	}

	return names[k]
}

// String returns a stable lowercase name, for logs and debugging.
func (t Tendency) String() string {
	switch t {
	case TendencyNone:
		return NameNone
	case TendencyFreeze:
		return NameFreeze
	case TendencyAvoid:
		return NameAvoid
	default:
		return NameUnknown
	}
}

// Valence reports whether the emotion feels good (+1), bad (-1) or neither (0).
// Mood accumulates along this axis.
func (k Kind) Valence() float64 {
	switch k {
	case Joy, Fun, Relief, Anticipation:
		return 1
	case Anger, Sadness, Anxiety, Boredom:
		return -1
	case Surprise:
		return 0
	default:
		return 0
	}
}

// IsReflexive reports whether the emotion can drive action on its own.
//
// Only the fast, protective feelings do. Everything else reaches behaviour the
// slow way, by tilting interest, because that is the path the design puts most
// of everyday behaviour on (docs/02-emotion.md 2.1).
func (k Kind) IsReflexive() bool {
	return k.Tendency() != TendencyNone
}

// Tendency reports the readiness to act that this emotion produces by itself.
func (k Kind) Tendency() Tendency {
	switch k {
	case Surprise:
		return TendencyFreeze
	case Anxiety:
		return TendencyAvoid
	case Joy, Fun, Relief, Anticipation, Anger, Sadness, Boredom:
		return TendencyNone
	default:
		return TendencyNone
	}
}

// valid reports whether the kind is one of the defined emotions, so callers
// cannot index internal state out of range.
func (k Kind) valid() bool {
	return k >= Joy && int(k) < kindCount
}
