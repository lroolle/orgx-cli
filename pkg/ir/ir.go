package ir

type DocType string

const (
	DocTypeOrg      DocType = "org"
	DocTypeMarkdown DocType = "md"
)

type Document struct {
	Path    string            `json:"path"`
	SHA256  string            `json:"sha256"`
	DocType DocType           `json:"doc_type"`
	Meta    DocumentMeta      `json:"meta"`
	Nodes   []Node            `json:"nodes"`
}

type DocumentMeta struct {
	Title       string            `json:"title,omitempty"`
	Frontmatter map[string]any    `json:"frontmatter,omitempty"`
}

type NodeType string

const (
	NodeTypeHeading NodeType = "heading"
	NodeTypeTask    NodeType = "task"
	NodeTypeLink    NodeType = "link"
	NodeTypeBlock   NodeType = "block"
)

type Node interface {
	NodeType() NodeType
	GetRef() string
	GetSpan() Span
}

type Span struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

type Heading struct {
	Type      NodeType          `json:"type"`
	Ref       string            `json:"ref"`
	Level     int               `json:"level"`
	Title     string            `json:"title"`
	Todo      string            `json:"todo,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Props     map[string]string `json:"props,omitempty"`
	Scheduled string            `json:"scheduled,omitempty"`
	Deadline  string            `json:"deadline,omitempty"`
	Body      Body              `json:"body"`
	Children  []Node            `json:"children,omitempty"`
	Span      Span              `json:"span"`
}

func (h *Heading) NodeType() NodeType { return NodeTypeHeading }
func (h *Heading) GetRef() string     { return h.Ref }
func (h *Heading) GetSpan() Span      { return h.Span }

type Body struct {
	Raw    string  `json:"raw"`
	Blocks []Block `json:"blocks,omitempty"`
}

type TaskState string

const (
	TaskStateOpen TaskState = "open"
	TaskStateDone TaskState = "done"
)

type Task struct {
	Type      NodeType          `json:"type"`
	Ref       string            `json:"ref"`
	State     TaskState         `json:"state"`
	Text      string            `json:"text"`
	Tags      []string          `json:"tags,omitempty"`
	Scheduled string            `json:"scheduled,omitempty"`
	Deadline  string            `json:"deadline,omitempty"`
	Span      Span              `json:"span"`
}

func (t *Task) NodeType() NodeType { return NodeTypeTask }
func (t *Task) GetRef() string     { return t.Ref }
func (t *Task) GetSpan() Span      { return t.Span }

type LinkKind string

const (
	LinkKindHTTP LinkKind = "http"
	LinkKindFile LinkKind = "file"
	LinkKindID   LinkKind = "id"
	LinkKindRoam LinkKind = "roam"
)

type Link struct {
	Type   NodeType `json:"type"`
	Kind   LinkKind `json:"kind"`
	Target string   `json:"target"`
	Desc   string   `json:"desc,omitempty"`
	Span   Span     `json:"span"`
}

func (l *Link) NodeType() NodeType { return NodeTypeLink }
func (l *Link) GetRef() string     { return l.Target }
func (l *Link) GetSpan() Span      { return l.Span }

type BlockKind string

const (
	BlockKindCode  BlockKind = "code"
	BlockKindQuote BlockKind = "quote"
	BlockKindTable BlockKind = "table"
	BlockKindOther BlockKind = "other"
)

type Block struct {
	Type NodeType  `json:"type"`
	Kind BlockKind `json:"kind"`
	Raw  string    `json:"raw"`
	Span Span      `json:"span"`
}

func (b *Block) NodeType() NodeType { return NodeTypeBlock }
func (b *Block) GetRef() string     { return "" }
func (b *Block) GetSpan() Span      { return b.Span }
