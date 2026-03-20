package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func main() {
	docDir := "D:/WAFtest/gotestwaf/gotestwaf-0.5.8/docs"

	files := []struct {
		input  string
		output string
		title  string
	}{
		{"GoTestWAF使用手册.md", "GoTestWAF使用手册.docx", "GoTestWAF 使用手册"},
		{"WAF安全验证方案.md", "WAF安全验证方案.docx", "WAF 安全验证方案"},
	}

	for _, f := range files {
		inputPath := filepath.Join(docDir, f.input)
		outputPath := filepath.Join(docDir, f.output)

		content, err := os.ReadFile(inputPath)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", inputPath, err)
			continue
		}

		docxContent := convertMarkdownToDocx(string(content), f.title)
		err = os.WriteFile(outputPath, docxContent, 0644)
		if err != nil {
			fmt.Printf("Error writing %s: %v\n", outputPath, err)
			continue
		}

		fmt.Printf("Converted: %s -> %s\n", f.input, f.output)
	}
}

func convertMarkdownToDocx(mdContent, title string) []byte {
	builder := NewDocxBuilder()

	// 添加标题
	builder.AddHeading(title, 1)
	builder.AddParagraph("")

	// 解析 Markdown
	lines := strings.Split(mdContent, "\n")

	inCodeBlock := false
	inTable := false
	tableHeaders := []string{}
	tableRows := [][]string{}
	inList := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// 处理代码块
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				inCodeBlock = false
				builder.AddParagraph("")
			} else {
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			// 代码块内容
			builder.AddCodeLine(line)
			continue
		}

		// 处理表格
		if strings.Contains(line, "|") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			if !inTable {
				inTable = true
				tableHeaders = []string{}
				tableRows = [][]string{}
			}

			// 解析表格行
			cells := parseTableRow(line)
			if len(cells) > 0 {
				// 检查是否是分隔行
				if isTableSeparator(line) {
					continue
				}
				if len(tableHeaders) == 0 {
					tableHeaders = cells
				} else {
					tableRows = append(tableRows, cells)
				}
			}
			continue
		} else if inTable {
			// 表格结束，输出表格
			if len(tableHeaders) > 0 && len(tableRows) > 0 {
				builder.AddTable(tableHeaders, tableRows)
			}
			inTable = false
			tableHeaders = []string{}
			tableRows = [][]string{}
		}

		// 处理标题
		if strings.HasPrefix(line, "# ") {
			builder.AddHeading(strings.TrimPrefix(line, "# "), 1)
			continue
		}
		if strings.HasPrefix(line, "## ") {
			builder.AddHeading(strings.TrimPrefix(line, "## "), 2)
			continue
		}
		if strings.HasPrefix(line, "### ") {
			builder.AddHeading(strings.TrimPrefix(line, "### "), 3)
			continue
		}
		if strings.HasPrefix(line, "#### ") {
			builder.AddHeading(strings.TrimPrefix(line, "#### "), 4)
			continue
		}

		// 处理列表
		if strings.HasPrefix(strings.TrimSpace(line), "- ") || strings.HasPrefix(strings.TrimSpace(line), "* ") {
			inList = true
			text := strings.TrimPrefix(strings.TrimSpace(line), "- ")
			text = strings.TrimPrefix(text, "* ")
			text = processMarkdownInline(text)
			builder.AddBulletItem(text)
			continue
		}

		// 数字列表
		if matched, _ := regexp.MatchString(`^\d+\.\s`, strings.TrimSpace(line)); matched {
			re := regexp.MustCompile(`^\d+\.\s*`)
			text := re.ReplaceAllString(strings.TrimSpace(line), "")
			text = processMarkdownInline(text)
			builder.AddNumberedItem(text)
			continue
		}

		// 空行
		if strings.TrimSpace(line) == "" {
			if inList {
				inList = false
			}
			builder.AddParagraph("")
			continue
		}

		// 普通段落
		text := processMarkdownInline(line)
		builder.AddParagraph(text)
	}

	// 处理最后的表格
	if inTable && len(tableHeaders) > 0 && len(tableRows) > 0 {
		builder.AddTable(tableHeaders, tableRows)
	}

	// 添加页脚
	builder.AddParagraph("")
	builder.AddParagraph("")
	builder.AddParagraph(fmt.Sprintf("文档生成时间: %s", time.Now().Format("2006年01月02日 15:04:05")))

	// 生成 DOCX
	buf, _ := buildDocxFile(builder.GetContent())
	return buf.Bytes()
}

func parseTableRow(line string) []string {
	// 去除首尾的 |
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")

	// 分割单元格
	cells := strings.Split(line, "|")
	result := []string{}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		cell = processMarkdownInline(cell)
		result = append(result, cell)
	}
	return result
}

func isTableSeparator(line string) bool {
	// 检查是否是表格分隔行（如 |---|---|）
	re := regexp.MustCompile(`^[\|\s\-:]+$`)
	return re.MatchString(strings.TrimSpace(line))
}

func processMarkdownInline(text string) string {
	// 处理行内代码 `code`
	re := regexp.MustCompile("`([^`]+)`")
	text = re.ReplaceAllString(text, "【$1】")

	// 处理加粗 **text** 或 __text__
	re = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	text = re.ReplaceAllString(text, "$1")
	re = regexp.MustCompile(`__([^_]+)__`)
	text = re.ReplaceAllString(text, "$1")

	// 处理斜体 *text* 或 _text_
	re = regexp.MustCompile(`\*([^*]+)\*`)
	text = re.ReplaceAllString(text, "$1")
	re = regexp.MustCompile(`_([^_]+)_`)
	text = re.ReplaceAllString(text, "$1")

	// 处理链接 [text](url)
	re = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	text = re.ReplaceAllString(text, "$1 ($2)")

	return text
}

// DocxBuilder builds DOCX file content
type DocxBuilder struct {
	documentContent strings.Builder
}

// NewDocxBuilder creates a new DOCX builder
func NewDocxBuilder() *DocxBuilder {
	return &DocxBuilder{}
}

// AddHeading adds a heading to the document
func (b *DocxBuilder) AddHeading(text string, level int) {
	style := fmt.Sprintf("Heading%d", level)
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:p><w:pPr><w:pStyle w:val="%s"/></w:pPr><w:r><w:t>%s</w:t></w:r></w:p>`,
		style, escapeXML(text)))
}

// AddParagraph adds a paragraph to the document
func (b *DocxBuilder) AddParagraph(text string) {
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:p><w:r><w:t>%s</w:t></w:r></w:p>`,
		escapeXML(text)))
}

// AddCodeLine adds a code line with monospace font
func (b *DocxBuilder) AddCodeLine(text string) {
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:p><w:r><w:rPr><w:rFonts w:ascii="Courier New" w:hAnsi="Courier New"/></w:rPr><w:t>%s</w:t></w:r></w:p>`,
		escapeXML(text)))
}

// AddBulletItem adds a bullet list item
func (b *DocxBuilder) AddBulletItem(text string) {
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:ilfo w:val="1"/></w:numPr></w:pPr><w:r><w:t>• %s</w:t></w:r></w:p>`,
		escapeXML(text)))
}

// AddNumberedItem adds a numbered list item
func (b *DocxBuilder) AddNumberedItem(text string) {
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:p><w:r><w:t>  %s</w:t></w:r></w:p>`,
		escapeXML(text)))
}

// AddTable adds a table to the document
func (b *DocxBuilder) AddTable(headers []string, rows [][]string) {
	b.documentContent.WriteString(`<w:tbl>`)
	b.documentContent.WriteString(`<w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="0" w:type="auto"/></w:tblPr>`)

	// Header row
	b.documentContent.WriteString(`<w:tr>`)
	for _, header := range headers {
		b.documentContent.WriteString(fmt.Sprintf(
			`<w:tc><w:tcPr><w:shd w:val="clear" w:color="auto" w:fill="4472C4"/></w:tcPr><w:p><w:r><w:rPr><w:b/><w:color w:val="FFFFFF"/></w:rPr><w:t>%s</w:t></w:r></w:p></w:tc>`,
			escapeXML(header)))
	}
	b.documentContent.WriteString(`</w:tr>`)

	// Data rows
	for i, row := range rows {
		b.documentContent.WriteString(`<w:tr>`)
		fillColor := "FFFFFF"
		if i%2 == 1 {
			fillColor = "E8F4FD"
		}
		for _, cell := range row {
			b.documentContent.WriteString(fmt.Sprintf(
				`<w:tc><w:tcPr><w:shd w:val="clear" w:color="auto" w:fill="%s"/></w:tcPr><w:p><w:r><w:t>%s</w:t></w:r></w:p></w:tc>`,
				fillColor, escapeXML(cell)))
		}
		b.documentContent.WriteString(`</w:tr>`)
	}

	b.documentContent.WriteString(`</w:tbl>`)
	b.documentContent.WriteString(`<w:p/>`) // Empty paragraph after table
}

// GetContent returns the document content
func (b *DocxBuilder) GetContent() string {
	return b.documentContent.String()
}

// buildDocxFile creates a valid DOCX file
func buildDocxFile(documentContent string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
<Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>
</Types>`

	addFileToZip(zipWriter, "[Content_Types].xml", contentTypes)

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

	addFileToZip(zipWriter, "_rels/.rels", rels)

	// word/_rels/document.xml.rels
	documentRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>
</Relationships>`

	addFileToZip(zipWriter, "word/_rels/document.xml.rels", documentRels)

	// word/styles.xml
	styles := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri" w:eastAsia="宋体"/></w:rPr></w:rPrDefault></w:docDefaults>
<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="Heading 1"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="240" w:after="120"/><w:jc w:val="center"/></w:pPr><w:rPr><w:b/><w:sz w:val="36"/><w:color w:val="2F5496"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="Heading 2"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="200" w:after="100"/></w:pPr><w:rPr><w:b/><w:sz w:val="28"/><w:color w:val="2F5496"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading3"><w:name w:val="Heading 3"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="160" w:after="80"/></w:pPr><w:rPr><w:b/><w:sz w:val="24"/><w:color w:val="1F3763"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading4"><w:name w:val="Heading 4"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="120" w:after="60"/></w:pPr><w:rPr><w:b/><w:sz w:val="22"/></w:rPr></w:style>
<w:style w:type="table" w:styleId="TableGrid"><w:name w:val="Table Grid"/><w:basedOn w:val="Normal"/><w:tblPr><w:tblBorders><w:top w:val="single" w:sz="4"/><w:left w:val="single" w:sz="4"/><w:bottom w:val="single" w:sz="4"/><w:right w:val="single" w:sz="4"/><w:insideH w:val="single" w:sz="4"/><w:insideV w:val="single" w:sz="4"/></w:tblBorders></w:tblPr></w:style>
</w:styles>`

	addFileToZip(zipWriter, "word/styles.xml", styles)

	// word/numbering.xml
	numbering := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:abstractNum w:abstractNumId="0">
<w:lvl w:ilvl="0"><w:start w:val="1"/><w:numFmt w:val="bullet"/><w:lvlText w:val="•"/><w:lvlJc w:val="left"/></w:lvl>
</w:abstractNum>
<w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>
</w:numbering>`

	addFileToZip(zipWriter, "word/numbering.xml", numbering)

	// word/document.xml
	documentXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
%s
<w:sectPr><w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>
</w:body>
</w:document>`, documentContent)

	addFileToZip(zipWriter, "word/document.xml", documentXML)

	zipWriter.Close()
	return buf, nil
}

func addFileToZip(zipWriter *zip.Writer, filename, content string) error {
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(content))
	return err
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}