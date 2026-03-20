package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Test case data structure
type TestCase struct {
	TestSet    string
	TestCase   string
	Percentage float64
	Blocked    int
	Bypassed   int
	Unresolved int
	Sent       int
	Failed     int
}

// Report data structure
type ReportData struct {
	WafName        string
	Url            string
	TestingDate    string
	GtwVersion     string
	TestCasesFP    string
	Args           string
	OverallGrade   string
	OverallScore   float64
	TotalSent      int
	Blocked        int
	Bypassed       int
	Unresolved     int
	Failed         int
	TestCases      []TestCase
}

func main() {
	// Read the HTML file
	htmlContent, err := os.ReadFile("D:/WAFtest/gotestwaf/gotestwaf-0.5.8/reports/waf-evaluation-report-2026-March-19-20-07-38.html")
	if err != nil {
		fmt.Printf("Error reading HTML file: %v\n", err)
		return
	}

	// Parse HTML
	data := parseHTML(string(htmlContent))

	// Generate DOCX
	docxContent := renderDocxReport(data)

	// Write to file
	outputFile := "D:/WAFtest/gotestwaf/gotestwaf-0.5.8/reports/waf-evaluation-report-2026-March-19-20-07-38.docx"
	err = os.WriteFile(outputFile, docxContent, 0644)
	if err != nil {
		fmt.Printf("Error writing DOCX file: %v\n", err)
		return
	}

	fmt.Printf("Successfully converted HTML to DOCX: %s\n", outputFile)
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

func parseHTML(html string) *ReportData {
	data := &ReportData{}

	// Extract project name
	if match := regexp.MustCompile(`<span class="row__content">([^<]+)</span>`).FindStringSubmatch(html); len(match) > 1 {
		data.WafName = strings.TrimSpace(match[1])
	}

	// Extract URL
	if match := regexp.MustCompile(`URL</span>\s*:\s*<span class="row__content">([^<]+)</span>`).FindStringSubmatch(html); len(match) > 1 {
		data.Url = strings.TrimSpace(match[1])
	}

	// Extract testing date
	if match := regexp.MustCompile(`Testing Date</span>\s*:\s*<span class="row__content">([^<]+)</span>`).FindStringSubmatch(html); len(match) > 1 {
		data.TestingDate = strings.TrimSpace(match[1])
	}

	// Extract GoTestWAF version
	if match := regexp.MustCompile(`GoTestWAF version</span>\s*:\s*<span class="row__content mono">([^<]+)</span>`).FindStringSubmatch(html); len(match) > 1 {
		data.GtwVersion = strings.TrimSpace(match[1])
	}

	// Extract test cases fingerprint
	if match := regexp.MustCompile(`Test cases fingerprint</span>\s*:\s*<span class="row__content mono">([^<]+)</span>`).FindStringSubmatch(html); len(match) > 1 {
		data.TestCasesFP = strings.TrimSpace(match[1])
	}

	// Extract arguments
	if match := regexp.MustCompile(`Used arguments</span>\s*:\s*<span class="row__args mono">([^<]+)</span>`).FindStringSubmatch(html); len(match) > 1 {
		data.Args = strings.TrimSpace(match[1])
	}

	// Extract overall grade
	if match := regexp.MustCompile(`<span class="grade__info-grade">([^<]+)</span>`).FindStringSubmatch(html); len(match) > 1 {
		data.OverallGrade = strings.TrimSpace(match[1])
	}

	// Extract overall score
	if match := regexp.MustCompile(`<span class="grade__info-ratio">([\d.]+)\s*/\s*100</span>`).FindStringSubmatch(html); len(match) > 1 {
		data.OverallScore, _ = strconv.ParseFloat(match[1], 64)
	}

	// Extract summary statistics
	if match := regexp.MustCompile(`Total requests sent:\s*(\d+)`).FindStringSubmatch(html); len(match) > 1 {
		data.TotalSent, _ = strconv.Atoi(match[1])
	}
	if match := regexp.MustCompile(`Number of blocked requests:\s*(\d+)`).FindStringSubmatch(html); len(match) > 1 {
		data.Blocked, _ = strconv.Atoi(match[1])
	}
	if match := regexp.MustCompile(`Number of passed requests:\s*(\d+)`).FindStringSubmatch(html); len(match) > 1 {
		data.Bypassed, _ = strconv.Atoi(match[1])
	}
	if match := regexp.MustCompile(`Number of unresolved requests:\s*(\d+)`).FindStringSubmatch(html); len(match) > 1 {
		data.Unresolved, _ = strconv.Atoi(match[1])
	}
	if match := regexp.MustCompile(`Number of failed requests:\s*(\d+)`).FindStringSubmatch(html); len(match) > 1 {
		data.Failed, _ = strconv.Atoi(match[1])
	}

	// Extract test cases
	testCaseRegex := regexp.MustCompile(`<div class="summary__grid--row">\s*<div class="summary__grid--row-item">([^<]+)</div>\s*<div class="summary__grid--row-item">([^<]+)</div>\s*<div class="summary__grid--row-item">([\d.]+)%</div>\s*<div class="summary__grid--row-item">(\d+)</div>\s*<div class="summary__grid--row-item">(\d+)</div>\s*<div class="summary__grid--row-item">(\d+)</div>\s*<div class="summary__grid--row-item">(\d+)</div>\s*<div class="summary__grid--row-item">(\d+)</div>`)
	matches := testCaseRegex.FindAllStringSubmatch(html, -1)
	for _, match := range matches {
		tc := TestCase{
			TestSet:    strings.TrimSpace(match[1]),
			TestCase:   strings.TrimSpace(match[2]),
			Percentage: parseFloat(match[3]),
			Blocked:    parseInt(match[4]),
			Bypassed:   parseInt(match[5]),
			Unresolved: parseInt(match[6]),
			Sent:       parseInt(match[7]),
			Failed:     parseInt(match[8]),
		}
		data.TestCases = append(data.TestCases, tc)
	}

	return data
}

func renderDocxReport(data *ReportData) []byte {
	builder := NewDocxBuilder()

	// Title
	builder.AddHeading("GoTestWAF 测试报告", 1)
	builder.AddParagraph("")

	// Summary Section
	builder.AddHeading("测试概要", 2)
	builder.AddBoldParagraph("WAF名称", data.WafName)
	builder.AddBoldParagraph("目标URL", data.Url)
	builder.AddBoldParagraph("测试日期", data.TestingDate)
	builder.AddBoldParagraph("GoTestWAF版本", data.GtwVersion)
	builder.AddBoldParagraph("测试用例指纹", data.TestCasesFP)
	builder.AddBoldParagraph("命令行参数", data.Args)
	builder.AddParagraph("")

	// Overall Score
	builder.AddHeading("综合得分", 2)
	gradeCN := getChineseGrade(data.OverallScore)
	builder.AddBoldParagraph("得分", fmt.Sprintf("%.1f%% (%s)", data.OverallScore, gradeCN))
	builder.AddParagraph("")

	// Score Details Table
	builder.AddHeading("得分详情", 3)
	scoreHeaders := []string{"类别", "真正例", "真负例", "综合"}
	scoreRows := [][]string{
		{"API安全", "不适用", "不适用", "不适用"},
		{"应用安全", fmt.Sprintf("%.1f%%", data.OverallScore), "不适用", fmt.Sprintf("%.1f%%", data.OverallScore)},
		{"总计", fmt.Sprintf("%.1f%%", data.OverallScore), "不适用", fmt.Sprintf("%.1f%%", data.OverallScore)},
	}
	builder.AddTable(scoreHeaders, scoreRows)

	// Request Statistics
	builder.AddHeading("请求统计", 3)
	statsHeaders := []string{"指标", "数值"}
	statsRows := [][]string{
		{"发送请求总数", fmt.Sprintf("%d", data.TotalSent)},
		{"已拦截", fmt.Sprintf("%d", data.Blocked)},
		{"已绕过", fmt.Sprintf("%d", data.Bypassed)},
		{"未确定", fmt.Sprintf("%d", data.Unresolved)},
		{"失败", fmt.Sprintf("%d", data.Failed)},
	}
	builder.AddTable(statsHeaders, statsRows)

	// True Positive Tests
	builder.AddHeading("真正例测试", 2)
	builder.AddParagraph("应被WAF拦截的恶意请求测试。")
	builder.AddBoldParagraph("得分", fmt.Sprintf("%.1f%%", data.OverallScore))
	builder.AddParagraph("")

	// Group test cases by test set
	testSetMap := make(map[string][]TestCase)
	for _, tc := range data.TestCases {
		testSetMap[tc.TestSet] = append(testSetMap[tc.TestSet], tc)
	}

	for testSet, testCases := range testSetMap {
		builder.AddHeading(testSet, 3)

		headers := []string{"测试用例", "拦截率%", "发送数", "已拦截", "已绕过", "未确定", "失败"}
		var rows [][]string
		for _, tc := range testCases {
			rows = append(rows, []string{
				tc.TestCase,
				fmt.Sprintf("%.2f", tc.Percentage),
				fmt.Sprintf("%d", tc.Sent),
				fmt.Sprintf("%d", tc.Blocked),
				fmt.Sprintf("%d", tc.Bypassed),
				fmt.Sprintf("%d", tc.Unresolved),
				fmt.Sprintf("%d", tc.Failed),
			})
		}
		builder.AddTable(headers, rows)
	}

	// Risk Level Description
	builder.AddHeading("风险等级说明", 2)
	riskHeaders := []string{"等级", "分数范围", "风险描述"}
	riskRows := [][]string{
		{"优秀", "≥90%", "WAF配置优秀，能有效防护各类攻击"},
		{"良好", "75-89%", "WAF配置良好，建议优化个别规则"},
		{"中等", "60-74%", "WAF存在一定防护盲区，需要调优"},
		{"及格", "40-59%", "WAF配置存在明显问题，需要重点优化"},
		{"不及格", "<40%", "WAF几乎无防护能力，需要重新评估"},
	}
	builder.AddTable(riskHeaders, riskRows)

	// Report Footer
	builder.AddParagraph("")
	builder.AddParagraph("报告由 GoTestWAF 自动生成")
	builder.AddParagraph(fmt.Sprintf("生成时间: %s", time.Now().Format("2006年01月02日 15:04:05")))

	// Build the complete DOCX file
	buf, _ := buildDocxFile(builder.GetContent())
	return buf.Bytes()
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

// AddBoldParagraph adds a paragraph with bold label
func (b *DocxBuilder) AddBoldParagraph(label, value string) {
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:p><w:r><w:rPr><w:b/></w:rPr><w:t>%s</w:t></w:r><w:r><w:t>: %s</w:t></w:r></w:p>`,
		escapeXML(label), escapeXML(value)))
}

// AddTable adds a table to the document
func (b *DocxBuilder) AddTable(headers []string, rows [][]string) {
	b.documentContent.WriteString(`<w:tbl>`)
	b.documentContent.WriteString(`<w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="0" w:type="auto"/></w:tblPr>`)

	// Header row
	b.documentContent.WriteString(`<w:tr>`)
	for _, header := range headers {
		b.documentContent.WriteString(fmt.Sprintf(
			`<w:tc><w:tcPr><w:shd w:val="clear" w:color="auto" w:fill="D9D9D9"/></w:tcPr><w:p><w:r><w:rPr><w:b/></w:rPr><w:t>%s</w:t></w:r></w:p></w:tc>`,
			escapeXML(header)))
	}
	b.documentContent.WriteString(`</w:tr>`)

	// Data rows
	for _, row := range rows {
		b.documentContent.WriteString(`<w:tr>`)
		for _, cell := range row {
			b.documentContent.WriteString(fmt.Sprintf(
				`<w:tc><w:p><w:r><w:t>%s</w:t></w:r></w:p></w:tc>`,
				escapeXML(cell)))
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

// buildDocxFile creates a valid DOCX file (which is a ZIP archive with specific structure)
func buildDocxFile(documentContent string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// [Content_Types].xml
	contentTypes := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

	if err := addFileToZip(zipWriter, "[Content_Types].xml", contentTypes); err != nil {
		return nil, err
	}

	// _rels/.rels
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

	if err := addFileToZip(zipWriter, "_rels/.rels", rels); err != nil {
		return nil, err
	}

	// word/_rels/document.xml.rels
	documentRels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`

	if err := addFileToZip(zipWriter, "word/_rels/document.xml.rels", documentRels); err != nil {
		return nil, err
	}

	// word/document.xml
	documentXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
%s
<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>
</w:body>
</w:document>`, documentContent)

	if err := addFileToZip(zipWriter, "word/document.xml", documentXML); err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf, nil
}

// addFileToZip adds a file to the ZIP archive
func addFileToZip(zipWriter *zip.Writer, filename, content string) error {
	writer, err := zipWriter.Create(filename)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(content))
	return err
}

// escapeXML escapes special characters for XML
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// getChineseGrade returns a Chinese grade string based on percentage
func getChineseGrade(percentage float64) string {
	switch {
	case percentage >= 97.0:
		return "优秀+"
	case percentage >= 93.0:
		return "优秀"
	case percentage >= 90.0:
		return "优秀-"
	case percentage >= 87.0:
		return "良好+"
	case percentage >= 83.0:
		return "良好"
	case percentage >= 80.0:
		return "良好-"
	case percentage >= 77.0:
		return "中等+"
	case percentage >= 73.0:
		return "中等"
	case percentage >= 70.0:
		return "中等-"
	case percentage >= 67.0:
		return "及格+"
	case percentage >= 63.0:
		return "及格"
	case percentage >= 60.0:
		return "及格-"
	default:
		return "不及格"
	}
}