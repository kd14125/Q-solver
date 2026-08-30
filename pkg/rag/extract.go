package rag

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	pdf "github.com/ledongthuc/pdf"
)

func extractFile(path string) (kind, content string, err error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".txt", ".md", ".markdown":
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return "", "", readErr
		}
		content, readErr = decodeText(data)
		kind = strings.TrimPrefix(ext, ".")
		err = readErr
	case ".docx":
		content, err = extractDOCX(path)
		kind = "docx"
	case ".pdf":
		content, err = extractPDF(path)
		kind = "pdf"
	default:
		return "", "", fmt.Errorf("不支持的知识库文件类型: %s", ext)
	}
	content = strings.TrimSpace(content)
	if err == nil && content == "" {
		err = fmt.Errorf("文件没有可提取的文本；扫描版 PDF 请先转换为可复制文本")
	}
	return kind, content, err
}

func decodeText(data []byte) (string, error) {
	if utf8.Valid(data) {
		return strings.TrimPrefix(string(data), "\ufeff"), nil
	}
	if len(data) >= 2 && ((data[0] == 0xff && data[1] == 0xfe) || (data[0] == 0xfe && data[1] == 0xff)) {
		little := data[0] == 0xff
		words := make([]uint16, 0, (len(data)-2)/2)
		for i := 2; i+1 < len(data); i += 2 {
			if little {
				words = append(words, uint16(data[i])|uint16(data[i+1])<<8)
			} else {
				words = append(words, uint16(data[i])<<8|uint16(data[i+1]))
			}
		}
		return string(utf16.Decode(words)), nil
	}
	return "", fmt.Errorf("文本文件不是有效的 UTF-8 或 UTF-16 编码")
}

func extractPDF(path string) (string, error) {
	file, reader, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	plain, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(plain)
	return string(data), err
}

func extractDOCX(path string) (string, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer archive.Close()
	for _, file := range archive.File {
		if file.Name != "word/document.xml" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return "", err
		}
		defer reader.Close()
		decoder := xml.NewDecoder(reader)
		var out bytes.Buffer
		inText := false
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			switch value := token.(type) {
			case xml.StartElement:
				if value.Name.Local == "t" {
					inText = true
				}
			case xml.CharData:
				if inText {
					out.Write([]byte(value))
				}
			case xml.EndElement:
				switch value.Name.Local {
				case "t":
					inText = false
				case "p", "tr":
					out.WriteByte('\n')
				case "tab":
					out.WriteByte('\t')
				}
			}
		}
		return out.String(), nil
	}
	return "", fmt.Errorf("DOCX 中缺少 word/document.xml")
}
