package brain

// Principal is an opaque authenticated subject supplied by an outer
// authentication boundary. The Brain Core does not interpret its contents.
type Principal string

// BrainID, RootID and LayerID are stable identities. Names and hierarchy
// position are deliberately not used as identifiers.
type BrainID string
type RootID string
type LayerID string

// LayerVersion is the optimistic-concurrency version observed by a caller.
type LayerVersion uint64

type NodeKind string

const (
	NodeRoot  NodeKind = "root"
	NodeLayer NodeKind = "layer"
)

// NodeRef identifies either the structural root or a visible layer as a
// parent. The Brain Core never treats a display name or path as an identity.
type NodeRef struct {
	Kind NodeKind
	ID   string
}

func RootRef(id RootID) NodeRef {
	return NodeRef{Kind: NodeRoot, ID: string(id)}
}

func LayerRef(id LayerID) NodeRef {
	return NodeRef{Kind: NodeLayer, ID: string(id)}
}

type Root struct {
	ID      RootID
	BrainID BrainID
}

type Brain struct {
	ID    BrainID
	Owner Principal
	Root  Root
}

type Layer struct {
	ID      LayerID
	BrainID BrainID
	Parent  NodeRef
	Name    string
	Version LayerVersion
}

type Hierarchy struct {
	Brain  Brain
	Root   Root
	Layers []Layer
}
