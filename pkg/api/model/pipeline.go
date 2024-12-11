package model

// Node represents a single node in the pipeline
type Node struct {
	ID         string   `json:"id"`
	Data       NodeData `json:"data"`
	Position   Position `json:"position"`
	Type       string   `json:"type,omitempty"`       // Optional field
	SequenceId string   `json:"sequenceId,omitempty"` // Optional field to update that the node is a pipeline
}

// NodeData represents the data within a node
type NodeData struct {
	Label  string `json:"label"`
	FaasID string `json:"faasId,omitempty"` // Optional field
}

// Position represents the position of a node
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Edge represents an edge between two nodes
type Edge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// PipelinePayload represents the full payload for the /pipeline endpoint
type PipelinePayload struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}
