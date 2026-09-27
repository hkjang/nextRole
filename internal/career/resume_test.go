package career

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func docxForTest(t *testing.T, xml string) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	w, err := z.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Write([]byte(xml)); err != nil {
		t.Fatal(err)
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestExtractDOCX(t *testing.T) {
	data := docxForTest(t, `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Java 개발 경력 10년</w:t></w:r></w:p><w:p><w:r><w:t>Python</w:t></w:r><w:r><w:tab/><w:t>중급</w:t></w:r></w:p><w:del><w:r><w:t>삭제된 과거 내용</w:t></w:r></w:del></w:body></w:document>`)
	got, err := ExtractResume(data, "resume.docx")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Java 개발 경력 10년\nPython\t중급") {
		t.Fatalf("lost text structure %q", got)
	}
	if strings.Contains(got, "삭제") {
		t.Fatal("tracked deleted text included")
	}
}
func TestResumeRejectsMalformedAndHugeData(t *testing.T) {
	cases := []struct {
		data []byte
		name string
	}{{nil, "x.txt"}, {[]byte("content"), "x.exe"}, {[]byte{0xff}, "x.txt"}, {[]byte("bad"), "x.docx"}, {[]byte("bad"), "x.pdf"}, {bytes.Repeat([]byte("x"), MaxResumeBytes+1), "x.txt"}, {docxForTest(t, `<bad`), "x.docx"}, {docxForTest(t, `<root><t>`+strings.Repeat("x", maxDocumentXML)+`</t></root>`), "x.docx"}}
	for _, tc := range cases {
		if _, err := ExtractResume(tc.data, tc.name); err == nil {
			t.Fatalf("accepted invalid %s", tc.name)
		}
	}
}
func TestExtractUTF8Text(t *testing.T) {
	got, err := ExtractResume([]byte("\x00  Java 백엔드\nPython \t"), "PROFILE.TXT")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Java 백엔드\nPython" {
		t.Fatalf("unexpected text %q", got)
	}
}
func TestExtractRealPDF(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext binary unavailable")
	}
	stream := "BT /F1 18 Tf 50 700 Td (Java developer 10 years) Tj ET"
	objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>", "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>", "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>", fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream)}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, off := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	got, err := ExtractResume(b.Bytes(), "resume.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Java developer 10 years") {
		t.Fatalf("PDF text lost: %q", got)
	}
}
