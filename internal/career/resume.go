package career

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const MaxResumeBytes = 10 << 20
const MaxResumeTextBytes = 1 << 20
const maxDocumentXML = 4 << 20

// ExtractResume extracts local TXT, DOCX and text-based PDF resumes. It never
// uploads document content. PDF uses the pdftotext binary bundled in the image;
// image-only PDFs return a clear OCR limitation instead of pretending to parse.
func ExtractResume(data []byte, filename string) (string, error) {
	if len(data) == 0 {
		return "", errors.New("파일이 비어 있습니다")
	}
	if len(data) > MaxResumeBytes {
		return "", errors.New("이력서 파일은 10MB 이하여야 합니다")
	}
	var text string
	var err error
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".txt":
		if !utf8.Valid(data) {
			return "", errors.New("텍스트 파일은 UTF-8 인코딩으로 저장해 주세요")
		}
		if len(data) > MaxResumeTextBytes {
			return "", errors.New("텍스트 내용은 1MB 이하여야 합니다")
		}
		text = string(data)
	case ".docx":
		text, err = extractDOCX(data)
	case ".pdf":
		text, err = extractPDF(data)
	default:
		return "", errors.New("PDF, DOCX, TXT 파일만 업로드할 수 있습니다")
	}
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, text))
	if text == "" {
		return "", errors.New("추출할 텍스트가 없습니다. 스캔 PDF는 OCR 후 텍스트 PDF 또는 TXT로 업로드해 주세요")
	}
	if len(text) > MaxResumeTextBytes {
		return "", errors.New("추출한 내용이 1MB를 초과했습니다")
	}
	return text, nil
}

func extractDOCX(data []byte) (string, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", errors.New("올바른 DOCX 파일이 아닙니다")
	}
	if len(archive.File) > 2048 {
		return "", errors.New("DOCX 내부 항목이 너무 많습니다")
	}
	var doc *zip.File
	for _, f := range archive.File {
		if f.Name == "word/document.xml" {
			doc = f
			break
		}
	}
	if doc == nil {
		return "", errors.New("DOCX 본문을 찾을 수 없습니다")
	}
	if doc.UncompressedSize64 > maxDocumentXML {
		return "", errors.New("DOCX 본문이 허용 크기를 초과했습니다")
	}
	r, err := doc.Open()
	if err != nil {
		return "", errors.New("DOCX 본문을 열 수 없습니다")
	}
	defer r.Close()
	raw, err := io.ReadAll(io.LimitReader(r, maxDocumentXML+1))
	if err != nil {
		return "", errors.New("DOCX 압축을 읽을 수 없습니다")
	}
	if len(raw) > maxDocumentXML {
		return "", errors.New("DOCX 압축 해제 크기가 너무 큽니다")
	}
	decoder := xml.NewDecoder(bytes.NewReader(raw))
	var out strings.Builder
	inText := false
	deleted := 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", errors.New("DOCX 본문 XML을 해석할 수 없습니다")
		}
		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "del":
				deleted++
			case "t":
				inText = true
			case "tab":
				if deleted == 0 {
					out.WriteByte('\t')
				}
			case "br", "cr":
				if deleted == 0 {
					out.WriteByte('\n')
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "del":
				if deleted > 0 {
					deleted--
				}
			case "t":
				inText = false
			case "p", "tr":
				if deleted == 0 {
					out.WriteByte('\n')
				}
			case "tc":
				if deleted == 0 {
					out.WriteByte('\t')
				}
			}
		case xml.CharData:
			if inText && deleted == 0 {
				out.Write(t)
			}
		}
		if out.Len() > MaxResumeTextBytes {
			return "", errors.New("DOCX 텍스트가 1MB를 초과했습니다")
		}
	}
	return out.String(), nil
}

type boundedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.Len() {
		b.exceeded = true
		return 0, errors.New("output limit exceeded")
	}
	return b.Buffer.Write(p)
}

func extractPDF(data []byte) (string, error) {
	header := data
	if len(header) > 1024 {
		header = header[:1024]
	}
	if !bytes.Contains(header, []byte("%PDF-")) {
		return "", errors.New("올바른 PDF 파일이 아닙니다")
	}
	binary, err := exec.LookPath("pdftotext")
	if err != nil {
		return "", errors.New("서버에 PDF 추출기가 없습니다. DOCX 또는 TXT로 업로드해 주세요")
	}
	dir, err := os.MkdirTemp("", "nextrole-resume-")
	if err != nil {
		return "", errors.New("임시 저장소를 사용할 수 없습니다")
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "resume.pdf")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", errors.New("PDF 임시 파일을 저장할 수 없습니다")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "-enc", "UTF-8", "-nopgbrk", "-f", "1", "-l", "100", path, "-")
	command.WaitDelay = time.Second
	output := &boundedBuffer{limit: MaxResumeTextBytes}
	stderr := &boundedBuffer{limit: 4096}
	command.Stdout = output
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if output.exceeded {
			return "", errors.New("PDF 텍스트가 1MB를 초과했습니다")
		}
		if ctx.Err() != nil {
			return "", errors.New("PDF 추출 시간이 초과되었습니다. 파일을 줄여 주세요")
		}
		return "", fmt.Errorf("PDF를 해석할 수 없습니다. 암호화·손상 여부를 확인해 주세요")
	}
	if !utf8.Valid(output.Bytes()) {
		return "", errors.New("PDF에서 유효한 UTF-8 텍스트를 추출할 수 없습니다")
	}
	return output.String(), nil
}
