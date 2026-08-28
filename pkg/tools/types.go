package tools

import "context"

type ToolContext struct{ Ctx context.Context }

type ToolResult struct {
	Text          string
	ImageData     []byte
	ImageMimeType string
	HasImage      bool
	Error         error
}

type Tool interface {
	Name() string
	Execute(ctx *ToolContext, toolID string) *ToolResult
}
