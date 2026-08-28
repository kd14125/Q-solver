package tools

import (
	imageutil "Q-Solver/pkg/ImageUtil"
	"fmt"

	"github.com/kbinani/screenshot"
)

type ScreenshotTool struct{}

func NewScreenshotTool() *ScreenshotTool { return &ScreenshotTool{} }
func (t *ScreenshotTool) Name() string   { return "get_exam_question" }
func (t *ScreenshotTool) Execute(_ *ToolContext, _ string) *ToolResult {
	data, err := CaptureFullScreen()
	if err != nil {
		return &ToolResult{Text: err.Error(), Error: err}
	}
	return &ToolResult{ImageData: data, ImageMimeType: "image/jpeg", HasImage: true}
}

func CaptureFullScreen() ([]byte, error) {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.Capture(bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy())
	if err != nil {
		return nil, fmt.Errorf("截图失败: %w", err)
	}
	data, err := imageutil.CompressForOCRWithMaxSize(img, 85, 0.5, false, 1280)
	if err != nil {
		return nil, fmt.Errorf("图片处理失败: %w", err)
	}
	return data, nil
}

func init() { Register(NewScreenshotTool()) }
