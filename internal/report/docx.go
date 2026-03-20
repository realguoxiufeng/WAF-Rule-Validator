package report

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/wallarm/gotestwaf/internal/db"
	"github.com/wallarm/gotestwaf/internal/version"
	"github.com/wallarm/gotestwaf/pkg/report"
)

// printFullReportToDocx prepares and saves a full report in DOCX format on a disk.
func printFullReportToDocx(
	s *db.Statistics, reportFile string, reportTime time.Time,
	wafName string, url string, openApiFile string, args []string,
	ignoreUnresolved bool, includePayloads bool,
) error {
	docxData := prepareDocxReportData(s, reportTime, wafName, url, openApiFile, args, ignoreUnresolved, includePayloads)

	buf, err := renderDocxReport(docxData)
	if err != nil {
		return errors.Wrap(err, "couldn't render DOCX report")
	}

	return writeDocxToFile(buf, reportFile)
}

// DocxReportData represents data for DOCX report rendering
type DocxReportData struct {
	WafName        string
	Url            string
	WafTestingDate string
	GtwVersion     string
	TestCasesFP    string
	OpenApiFile    string
	Args           []string

	OverallScore     *report.Grade
	ApiSecScore      *report.Grade
	AppSecScore      *report.Grade
	ApiSecTPositive  *report.Grade
	ApiSecTNegative  *report.Grade
	AppSecTPositive  *report.Grade
	AppSecTNegative  *report.Grade

	TotalSent                int
	BlockedRequestsNumber    int
	BypassedRequestsNumber   int
	UnresolvedRequestsNumber int
	FailedRequestsNumber     int

	TruePositiveTests DocxTestSummaryData
	TrueNegativeTests DocxTestSummaryData

	ScannedPaths db.ScannedPaths

	IgnoreUnresolved bool
	IncludePayloads  bool

	// Payload details for detailed report
	TpBypassed   map[string]map[string]map[int]*DocxTestDetails
	TpUnresolved map[string]map[int]*DocxTestDetails
	TpFailed     []*db.FailedDetails

	TnBlocked    map[string]map[int]*DocxTestDetails
	TnBypassed   map[string]map[int]*DocxTestDetails
	TnUnresolved map[string]map[int]*DocxTestDetails
	TnFailed     []*db.FailedDetails

	// Comparison data
	ComparisonTable []*report.ComparisonTableRow
	WallarmResult   *report.ComparisonTableRow

	// Chart data
	ApiSecIndicators []string
	ApiSecItems      []float64
	AppSecIndicators []string
	AppSecItems      []float64
}

// DocxTestDetails represents test details for DOCX report
type DocxTestDetails struct {
	TestCase     string
	Encoders     []string
	Placeholders []string
}

// DocxTestSummaryData represents test summary for DOCX report
type DocxTestSummaryData struct {
	Score      float64
	TotalSent  int
	Blocked    int
	Bypassed   int
	Unresolved int
	Failed     int
	TestSets   []DocxTestSetData
}

// DocxTestSetData represents test set summary for DOCX report
type DocxTestSetData struct {
	Name       string
	Percentage float64
	Sent       int
	Blocked    int
	Bypassed   int
	Unresolved int
	Failed     int
	TestCases  []DocxTestCaseData
}

// DocxTestCaseData represents test case summary for DOCX report
type DocxTestCaseData struct {
	Name       string
	Percentage float64
	Sent       int
	Blocked    int
	Bypassed   int
	Unresolved int
	Failed     int
}

// prepareDocxReportData prepares data for DOCX report
func prepareDocxReportData(
	s *db.Statistics, reportTime time.Time, wafName string,
	url string, openApiFile string, args []string,
	ignoreUnresolved bool, includePayloads bool,
) *DocxReportData {
	data := &DocxReportData{
		WafName:         wafName,
		Url:             url,
		WafTestingDate:  reportTime.Format("2006年01月02日"),
		GtwVersion:      version.Version,
		TestCasesFP:     s.TestCasesFingerprint,
		OpenApiFile:     openApiFile,
		Args:            args,
		IgnoreUnresolved: ignoreUnresolved,
		IncludePayloads:  includePayloads,
		ComparisonTable:  comparisonTable,
		WallarmResult:    wallarmResult,
	}

	// Overall score
	data.OverallScore = getChineseGrade(s.Score.Average, s.Score.Average < 0)

	// API Security scores
	if s.Score.ApiSec.TruePositive < 0 {
		data.ApiSecTPositive = getChineseGrade(0.0, true)
	} else {
		data.ApiSecTPositive = getChineseGrade(s.Score.ApiSec.TruePositive, false)
	}

	if s.Score.ApiSec.TrueNegative < 0 {
		data.ApiSecTNegative = getChineseGrade(0.0, true)
	} else {
		data.ApiSecTNegative = getChineseGrade(s.Score.ApiSec.TrueNegative, false)
	}

	data.ApiSecScore = getChineseGrade(s.Score.ApiSec.Average, s.Score.ApiSec.Average < 0)

	// Application Security scores
	if s.Score.AppSec.TruePositive < 0 {
		data.AppSecTPositive = getChineseGrade(0.0, true)
	} else {
		data.AppSecTPositive = getChineseGrade(s.Score.AppSec.TruePositive, false)
	}

	if s.Score.AppSec.TrueNegative < 0 {
		data.AppSecTNegative = getChineseGrade(0.0, true)
	} else {
		data.AppSecTNegative = getChineseGrade(s.Score.AppSec.TrueNegative, false)
	}

	data.AppSecScore = getChineseGrade(s.Score.AppSec.Average, s.Score.AppSec.Average < 0)

	// Request statistics
	data.TotalSent = s.TruePositiveTests.ReqStats.AllRequestsNumber + s.TrueNegativeTests.ReqStats.AllRequestsNumber
	data.BlockedRequestsNumber = s.TruePositiveTests.ReqStats.BlockedRequestsNumber + s.TrueNegativeTests.ReqStats.BlockedRequestsNumber
	data.BypassedRequestsNumber = s.TruePositiveTests.ReqStats.BypassedRequestsNumber + s.TrueNegativeTests.ReqStats.BypassedRequestsNumber
	data.UnresolvedRequestsNumber = s.TruePositiveTests.ReqStats.UnresolvedRequestsNumber + s.TrueNegativeTests.ReqStats.UnresolvedRequestsNumber
	data.FailedRequestsNumber = s.TruePositiveTests.ReqStats.FailedRequestsNumber + s.TrueNegativeTests.ReqStats.FailedRequestsNumber

	// True Positive Tests
	data.TruePositiveTests = prepareDocxTestSummaryData(s.TruePositiveTests, true)

	// True Negative Tests
	data.TrueNegativeTests = prepareDocxTestSummaryData(s.TrueNegativeTests, false)

	// Scanned paths
	data.ScannedPaths = s.Paths

	// Chart data
	data.ApiSecIndicators, data.ApiSecItems, data.AppSecIndicators, data.AppSecItems = generateChartData(s)

	// Payload details
	if includePayloads {
		// True Positive Bypassed
		data.TpBypassed = make(map[string]map[string]map[int]*DocxTestDetails)
		for _, d := range s.TruePositiveTests.Bypasses {
			payload := truncatePayload(d.Payload)
			if data.TpBypassed[d.AdditionalInfo[0]] == nil {
				data.TpBypassed[d.AdditionalInfo[0]] = make(map[string]map[int]*DocxTestDetails)
			}
			if data.TpBypassed[d.AdditionalInfo[0]][payload] == nil {
				data.TpBypassed[d.AdditionalInfo[0]][payload] = make(map[int]*DocxTestDetails)
			}
			if _, ok := data.TpBypassed[d.AdditionalInfo[0]][payload][d.ResponseStatusCode]; !ok {
				data.TpBypassed[d.AdditionalInfo[0]][payload][d.ResponseStatusCode] = &DocxTestDetails{
					TestCase:     d.TestCase,
					Encoders:     []string{},
					Placeholders: []string{},
				}
			}
			details := data.TpBypassed[d.AdditionalInfo[0]][payload][d.ResponseStatusCode]
			details.Encoders = appendUnique(details.Encoders, d.Encoder)
			details.Placeholders = appendUnique(details.Placeholders, d.Placeholder)
		}

		// True Positive Unresolved
		data.TpUnresolved = make(map[string]map[int]*DocxTestDetails)
		for _, d := range s.TruePositiveTests.Unresolved {
			payload := truncatePayload(d.Payload)
			if data.TpUnresolved[payload] == nil {
				data.TpUnresolved[payload] = make(map[int]*DocxTestDetails)
			}
			if _, ok := data.TpUnresolved[payload][d.ResponseStatusCode]; !ok {
				data.TpUnresolved[payload][d.ResponseStatusCode] = &DocxTestDetails{
					TestCase:     d.TestCase,
					Encoders:     []string{},
					Placeholders: []string{},
				}
			}
			details := data.TpUnresolved[payload][d.ResponseStatusCode]
			details.Encoders = appendUnique(details.Encoders, d.Encoder)
			details.Placeholders = appendUnique(details.Placeholders, d.Placeholder)
		}

		data.TpFailed = s.TruePositiveTests.Failed

		// True Negative Blocked
		data.TnBlocked = make(map[string]map[int]*DocxTestDetails)
		for _, d := range s.TrueNegativeTests.Blocked {
			payload := truncatePayload(d.Payload)
			if data.TnBlocked[payload] == nil {
				data.TnBlocked[payload] = make(map[int]*DocxTestDetails)
			}
			if _, ok := data.TnBlocked[payload][d.ResponseStatusCode]; !ok {
				data.TnBlocked[payload][d.ResponseStatusCode] = &DocxTestDetails{
					TestCase:     d.TestCase,
					Encoders:     []string{},
					Placeholders: []string{},
				}
			}
			details := data.TnBlocked[payload][d.ResponseStatusCode]
			details.Encoders = appendUnique(details.Encoders, d.Encoder)
			details.Placeholders = appendUnique(details.Placeholders, d.Placeholder)
		}

		// True Negative Bypassed
		data.TnBypassed = make(map[string]map[int]*DocxTestDetails)
		for _, d := range s.TrueNegativeTests.Bypasses {
			payload := truncatePayload(d.Payload)
			if data.TnBypassed[payload] == nil {
				data.TnBypassed[payload] = make(map[int]*DocxTestDetails)
			}
			if _, ok := data.TnBypassed[payload][d.ResponseStatusCode]; !ok {
				data.TnBypassed[payload][d.ResponseStatusCode] = &DocxTestDetails{
					TestCase:     d.TestCase,
					Encoders:     []string{},
					Placeholders: []string{},
				}
			}
			details := data.TnBypassed[payload][d.ResponseStatusCode]
			details.Encoders = appendUnique(details.Encoders, d.Encoder)
			details.Placeholders = appendUnique(details.Placeholders, d.Placeholder)
		}

		// True Negative Unresolved
		data.TnUnresolved = make(map[string]map[int]*DocxTestDetails)
		for _, d := range s.TrueNegativeTests.Unresolved {
			payload := truncatePayload(d.Payload)
			if data.TnUnresolved[payload] == nil {
				data.TnUnresolved[payload] = make(map[int]*DocxTestDetails)
			}
			if _, ok := data.TnUnresolved[payload][d.ResponseStatusCode]; !ok {
				data.TnUnresolved[payload][d.ResponseStatusCode] = &DocxTestDetails{
					TestCase:     d.TestCase,
					Encoders:     []string{},
					Placeholders: []string{},
				}
			}
			details := data.TnUnresolved[payload][d.ResponseStatusCode]
			details.Encoders = appendUnique(details.Encoders, d.Encoder)
			details.Placeholders = appendUnique(details.Placeholders, d.Placeholder)
		}

		data.TnFailed = s.TrueNegativeTests.Failed
	}

	return data
}

// appendUnique appends a string to a slice if it doesn't already exist
func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

// prepareDocxTestSummaryData prepares test summary for DOCX report
func prepareDocxTestSummaryData(summary db.TestsSummary, isTruePositive bool) DocxTestSummaryData {
	result := DocxTestSummaryData{
		Score:      summary.ResolvedBlockedRequestsPercentage,
		TotalSent:  summary.ReqStats.AllRequestsNumber,
		Blocked:    summary.ReqStats.BlockedRequestsNumber,
		Bypassed:   summary.ReqStats.BypassedRequestsNumber,
		Unresolved: summary.ReqStats.UnresolvedRequestsNumber,
		Failed:     summary.ReqStats.FailedRequestsNumber,
	}

	// Group by test set
	testSetMap := make(map[string]*DocxTestSetData)
	for _, row := range summary.SummaryTable {
		if _, ok := testSetMap[row.TestSet]; !ok {
			testSetMap[row.TestSet] = &DocxTestSetData{
				Name: row.TestSet,
			}
		}

		testSet := testSetMap[row.TestSet]

		testSet.TestCases = append(testSet.TestCases, DocxTestCaseData{
			Name:       row.TestCase,
			Percentage: row.Percentage,
			Sent:       row.Sent,
			Blocked:    row.Blocked,
			Bypassed:   row.Bypassed,
			Unresolved: row.Unresolved,
			Failed:     row.Failed,
		})

		testSet.Sent += row.Sent
		testSet.Blocked += row.Blocked
		testSet.Bypassed += row.Bypassed
		testSet.Unresolved += row.Unresolved
		testSet.Failed += row.Failed
	}

	// Calculate test set percentages and convert to sorted slice
	for _, testSet := range testSetMap {
		resolved := testSet.Blocked + testSet.Bypassed
		if resolved > 0 {
			if isTruePositive {
				testSet.Percentage = db.Round(float64(testSet.Blocked) / float64(resolved) * 100)
			} else {
				testSet.Percentage = db.Round(float64(testSet.Bypassed) / float64(resolved) * 100)
			}
		}
		result.TestSets = append(result.TestSets, *testSet)
	}

	// Sort by name
	sort.Slice(result.TestSets, func(i, j int) bool {
		return result.TestSets[i].Name < result.TestSets[j].Name
	})

	return result
}

// DocxBuilder builds DOCX file content
type DocxBuilder struct {
	documentContent strings.Builder
}

// NewDocxBuilder creates a new DOCX builder
func NewDocxBuilder() *DocxBuilder {
	return &DocxBuilder{}
}

// AddHeading adds a styled heading to the document
func (b *DocxBuilder) AddHeading(text string, level int) {
	styles := map[int]string{
		1: `font-size:32pt;font-weight:bold;color:#3942EA;margin-top:24pt;margin-bottom:12pt;`,
		2: `font-size:24pt;font-weight:bold;color:#000000;margin-top:18pt;margin-bottom:10pt;border-bottom:2pt solid #3942EA;padding-bottom:4pt;`,
		3: `font-size:16pt;font-weight:bold;color:#333333;margin-top:12pt;margin-bottom:6pt;`,
		4: `font-size:12pt;font-weight:bold;color:#555555;margin-top:8pt;margin-bottom:4pt;`,
	}
	style := styles[level]
	if style == "" {
		style = styles[3]
	}
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:p><w:pPr><w:spacing w:before="200" w:after="100"/><w:rPr><w:sz w:val="%d"/><w:b/><w:color w:val="%s"/></w:rPr></w:pPr><w:r><w:rPr><w:sz w:val="%d"/><w:b/><w:color w:val="%s"/></w:rPr><w:t>%s</w:t></w:r></w:p>`,
		getHeadingSize(level), getHeadingColor(level), getHeadingSize(level), getHeadingColor(level), escapeXML(text)))
}

func getHeadingSize(level int) int {
	sizes := map[int]int{1: 48, 2: 36, 3: 24, 4: 18}
	return sizes[level]
}

func getHeadingColor(level int) string {
	colors := map[int]string{1: "3942EA", 2: "000000", 3: "333333", 4: "555555"}
	return colors[level]
}

// AddParagraph adds a paragraph to the document
func (b *DocxBuilder) AddParagraph(text string) {
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:p><w:pPr><w:spacing w:before="60" w:after="60"/></w:pPr><w:r><w:rPr><w:sz w:val="20"/></w:rPr><w:t>%s</w:t></w:r></w:p>`,
		escapeXML(text)))
}

// AddBoldParagraph adds a paragraph with bold label
func (b *DocxBuilder) AddBoldParagraph(label, value string) {
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:p><w:pPr><w:spacing w:before="60" w:after="60"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="20"/></w:rPr><w:t>%s：</w:t></w:r><w:r><w:rPr><w:sz w:val="20"/></w:rPr><w:t>%s</w:t></w:r></w:p>`,
		escapeXML(label), escapeXML(value)))
}

// AddKeyValueParagraph adds a paragraph with label and value
func (b *DocxBuilder) AddKeyValueParagraph(label, value string, bold bool) {
	if bold {
		b.documentContent.WriteString(fmt.Sprintf(
			`<w:p><w:pPr><w:spacing w:before="80" w:after="40"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="22"/><w:color w:val="333333"/></w:rPr><w:t>%s：</w:t></w:r><w:r><w:rPr><w:sz w:val="22"/></w:rPr><w:t>%s</w:t></w:r></w:p>`,
			escapeXML(label), escapeXML(value)))
	} else {
		b.documentContent.WriteString(fmt.Sprintf(
			`<w:p><w:pPr><w:spacing w:before="60" w:after="40"/></w:pPr><w:r><w:rPr><w:sz w:val="20"/></w:rPr><w:t>%s：%s</w:t></w:r></w:p>`,
			escapeXML(label), escapeXML(value)))
	}
}

// AddSpacer adds a spacer paragraph
func (b *DocxBuilder) AddSpacer() {
	b.documentContent.WriteString(`<w:p><w:pPr><w:spacing w:before="120" w:after="0"/></w:pPr></w:p>`)
}

// AddHorizontalLine adds a horizontal line
func (b *DocxBuilder) AddHorizontalLine() {
	b.documentContent.WriteString(`<w:p><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="12" w:space="1" w:color="3942EA"/></w:pBdr><w:spacing w:before="100" w:after="100"/></w:pPr></w:p>`)
}

// GradeColors defines color scheme for grades
var GradeColors = map[string]string{
	"a":  "56CC54", // Green
	"b":  "FDBE10", // Yellow
	"c":  "FC7303", // Orange
	"d":  "F26344", // Orange-Red
	"f":  "F24444", // Red
	"na": "CCCCCC", // Grey
}

// GradeBgColors defines background colors for grades
var GradeBgColors = map[string]string{
	"a":  "E1F9D9", // Light Green
	"b":  "FEF2B9", // Light Yellow
	"c":  "FEE1B4", // Light Orange
	"d":  "f8e6df", // Light Orange-Red
	"f":  "f8d2c4", // Light Red
	"na": "ECECEC", // Light Grey
}

// AddGradeTable adds a styled grade table
func (b *DocxBuilder) AddGradeTable(data *DocxReportData) {
	b.documentContent.WriteString(`<w:tbl>`)
	b.documentContent.WriteString(`<w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="9000" w:type="dxa"/><w:tblBorders><w:top w:val="single" w:sz="4" w:color="CCCCCC"/><w:left w:val="single" w:sz="4" w:color="CCCCCC"/><w:bottom w:val="single" w:sz="4" w:color="CCCCCC"/><w:right w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideH w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideV w:val="single" w:sz="4" w:color="CCCCCC"/></w:tblBorders></w:tblPr>`)

	// Header row
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableHeaderCell("类别", 2000)
	b.addTableHeaderCell("真正例", 2000)
	b.addTableHeaderCell("真负例", 2000)
	b.addTableHeaderCell("综合评分", 3000)
	b.documentContent.WriteString(`</w:tr>`)

	// API Security row
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableCell("API安全", 2000, false, "")
	b.addGradeCell(data.ApiSecTPositive, 2000)
	b.addGradeCell(data.ApiSecTNegative, 2000)
	b.addGradeCell(data.ApiSecScore, 3000)
	b.documentContent.WriteString(`</w:tr>`)

	// Application Security row
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableCell("应用安全", 2000, false, "")
	b.addGradeCell(data.AppSecTPositive, 2000)
	b.addGradeCell(data.AppSecTNegative, 2000)
	b.addGradeCell(data.AppSecScore, 3000)
	b.documentContent.WriteString(`</w:tr>`)

	// Total row
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableCell("综合得分", 2000, true, "")
	avgTP := (data.ApiSecTPositive.Percentage + data.AppSecTPositive.Percentage) / 2
	avgTN := (data.ApiSecTNegative.Percentage + data.AppSecTNegative.Percentage) / 2
	b.addTableCell(fmt.Sprintf("%.2f%%", avgTP), 2000, true, "")
	b.addTableCell(fmt.Sprintf("%.2f%%", avgTN), 2000, true, "")
	b.addGradeCell(data.OverallScore, 3000)
	b.documentContent.WriteString(`</w:tr>`)

	b.documentContent.WriteString(`</w:tbl>`)
	b.AddSpacer()
}

// AddBenchmarkTable adds benchmark comparison table
func (b *DocxBuilder) AddBenchmarkTable(data *DocxReportData) {
	b.documentContent.WriteString(`<w:tbl>`)
	b.documentContent.WriteString(`<w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="9000" w:type="dxa"/><w:tblBorders><w:top w:val="single" w:sz="4" w:color="CCCCCC"/><w:left w:val="single" w:sz="4" w:color="CCCCCC"/><w:bottom w:val="single" w:sz="4" w:color="CCCCCC"/><w:right w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideH w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideV w:val="single" w:sz="4" w:color="CCCCCC"/></w:tblBorders></w:tblPr>`)

	// Header row
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableHeaderCell("解决方案", 3000)
	b.addTableHeaderCell("API安全", 2000)
	b.addTableHeaderCell("应用安全", 2000)
	b.addTableHeaderCell("综合评分", 2000)
	b.documentContent.WriteString(`</w:tr>`)

	// Comparison rows
	for _, row := range data.ComparisonTable {
		b.documentContent.WriteString(`<w:tr>`)
		b.addTableCell(row.Name, 3000, false, "")
		b.addGradeCell(row.ApiSec, 2000)
		b.addGradeCell(row.AppSec, 2000)
		b.addGradeCell(row.OverallScore, 2000)
		b.documentContent.WriteString(`</w:tr>`)
	}

	// Wallarm row
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableCell("Wallarm", 3000, true, "")
	b.addGradeCell(data.WallarmResult.ApiSec, 2000)
	b.addGradeCell(data.WallarmResult.AppSec, 2000)
	b.addGradeCell(data.WallarmResult.OverallScore, 2000)
	b.documentContent.WriteString(`</w:tr>`)

	// Your project row (highlighted)
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableCell("您的项目", 3000, true, "DEE0FC")
	b.addGradeCellWithBg(data.ApiSecScore, 2000, "DEE0FC")
	b.addGradeCellWithBg(data.AppSecScore, 2000, "DEE0FC")
	b.addGradeCellWithBg(data.OverallScore, 2000, "DEE0FC")
	b.documentContent.WriteString(`</w:tr>`)

	b.documentContent.WriteString(`</w:tbl>`)
	b.AddSpacer()
}

// AddSummaryTable adds a summary statistics table
func (b *DocxBuilder) AddSummaryTable(data *DocxReportData) {
	b.documentContent.WriteString(`<w:tbl>`)
	b.documentContent.WriteString(`<w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="9000" w:type="dxa"/><w:tblBorders><w:top w:val="single" w:sz="4" w:color="CCCCCC"/><w:left w:val="single" w:sz="4" w:color="CCCCCC"/><w:bottom w:val="single" w:sz="4" w:color="CCCCCC"/><w:right w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideH w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideV w:val="single" w:sz="4" w:color="CCCCCC"/></w:tblBorders></w:tblPr>`)

	// Header row
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableHeaderCell("统计指标", 5000)
	b.addTableHeaderCell("数值", 4000)
	b.documentContent.WriteString(`</w:tr>`)

	// Data rows
	stats := []struct {
		label string
		value int
		color string
	}{
		{"发送请求总数", data.TotalSent, ""},
		{"已拦截请求数", data.BlockedRequestsNumber, "E1F9D9"},
		{"已绕过请求数", data.BypassedRequestsNumber, "f8d2c4"},
		{"未确定请求数", data.UnresolvedRequestsNumber, "FEF2B9"},
		{"失败请求数", data.FailedRequestsNumber, "FEE1B4"},
	}

	for _, stat := range stats {
		b.documentContent.WriteString(`<w:tr>`)
		b.addTableCell(stat.label, 5000, false, "")
		b.addTableCell(fmt.Sprintf("%d", stat.value), 4000, true, stat.color)
		b.documentContent.WriteString(`</w:tr>`)
	}

	b.documentContent.WriteString(`</w:tbl>`)
	b.AddSpacer()
}

// AddTestSetTable adds a test set summary table
func (b *DocxBuilder) AddTestSetTable(testSets []DocxTestSetData, isTruePositive bool, ignoreUnresolved bool) {
	for _, testSet := range testSets {
		b.AddHeading(testSet.Name, 3)

		b.documentContent.WriteString(`<w:tbl>`)
		b.documentContent.WriteString(`<w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="9000" w:type="dxa"/><w:tblBorders><w:top w:val="single" w:sz="4" w:color="CCCCCC"/><w:left w:val="single" w:sz="4" w:color="CCCCCC"/><w:bottom w:val="single" w:sz="4" w:color="CCCCCC"/><w:right w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideH w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideV w:val="single" w:sz="4" w:color="CCCCCC"/></w:tblBorders></w:tblPr>`)

		// Header row
		b.documentContent.WriteString(`<w:tr>`)
		b.addTableHeaderCell("测试用例", 2500)
		if isTruePositive {
			b.addTableHeaderCell("拦截率", 1200)
		} else {
			b.addTableHeaderCell("通过率", 1200)
		}
		b.addTableHeaderCell("发送", 1000)
		b.addTableHeaderCell("拦截", 1000)
		b.addTableHeaderCell("绕过", 1000)
		if !ignoreUnresolved {
			b.addTableHeaderCell("未确定", 1100)
		}
		b.addTableHeaderCell("失败", 1200)
		b.documentContent.WriteString(`</w:tr>`)

		// Test case rows
		for _, tc := range testSet.TestCases {
			b.documentContent.WriteString(`<w:tr>`)
			b.addTableCell(tc.Name, 2500, false, "")
			percentageColor := getPercentageColor(tc.Percentage)
			b.addTableCell(fmt.Sprintf("%.2f%%", tc.Percentage), 1200, true, percentageColor)
			b.addTableCell(fmt.Sprintf("%d", tc.Sent), 1000, false, "")
			b.addTableCell(fmt.Sprintf("%d", tc.Blocked), 1000, false, "")
			b.addTableCell(fmt.Sprintf("%d", tc.Bypassed), 1000, false, "")
			if !ignoreUnresolved {
				b.addTableCell(fmt.Sprintf("%d", tc.Unresolved), 1100, false, "")
			}
			b.addTableCell(fmt.Sprintf("%d", tc.Failed), 1200, false, "")
			b.documentContent.WriteString(`</w:tr>`)
		}

		// Summary row
		b.documentContent.WriteString(`<w:tr>`)
		b.addTableCell("合计", 2500, true, "D9D9D9")
		b.addTableCell(fmt.Sprintf("%.2f%%", testSet.Percentage), 1200, true, "D9D9D9")
		b.addTableCell(fmt.Sprintf("%d", testSet.Sent), 1000, true, "D9D9D9")
		b.addTableCell(fmt.Sprintf("%d", testSet.Blocked), 1000, true, "D9D9D9")
		b.addTableCell(fmt.Sprintf("%d", testSet.Bypassed), 1000, true, "D9D9D9")
		if !ignoreUnresolved {
			b.addTableCell(fmt.Sprintf("%d", testSet.Unresolved), 1100, true, "D9D9D9")
		}
		b.addTableCell(fmt.Sprintf("%d", testSet.Failed), 1200, true, "D9D9D9")
		b.documentContent.WriteString(`</w:tr>`)

		b.documentContent.WriteString(`</w:tbl>`)
		b.AddSpacer()
	}
}

// AddPayloadDetailsTable adds a table showing payload details
func (b *DocxBuilder) AddPayloadDetailsTable(title string, payloads map[string]map[int]*DocxTestDetails, isTruePositive bool) {
	if len(payloads) == 0 {
		return
	}

	b.AddHeading(title, 4)

	b.documentContent.WriteString(`<w:tbl>`)
	b.documentContent.WriteString(`<w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="9000" w:type="dxa"/><w:tblBorders><w:top w:val="single" w:sz="4" w:color="CCCCCC"/><w:left w:val="single" w:sz="4" w:color="CCCCCC"/><w:bottom w:val="single" w:sz="4" w:color="CCCCCC"/><w:right w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideH w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideV w:val="single" w:sz="4" w:color="CCCCCC"/></w:tblBorders></w:tblPr>`)

	// Header row
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableHeaderCell("Payload", 3000)
	b.addTableHeaderCell("测试用例", 1500)
	b.addTableHeaderCell("编码器", 1500)
	b.addTableHeaderCell("占位符", 1500)
	b.addTableHeaderCell("状态码", 1500)
	b.documentContent.WriteString(`</w:tr>`)

	// Data rows
	for payload, codeMap := range payloads {
		for code, details := range codeMap {
			b.documentContent.WriteString(`<w:tr>`)
			b.addTableCellMono(truncatePayloadForTable(payload), 3000)
			b.addTableCell(details.TestCase, 1500, false, "")
			b.addTableCell(strings.Join(details.Encoders, ", "), 1500, false, "")
			b.addTableCell(strings.Join(details.Placeholders, ", "), 1500, false, "")
			b.addTableCell(fmt.Sprintf("%d", code), 1500, false, "")
			b.documentContent.WriteString(`</w:tr>`)
		}
	}

	b.documentContent.WriteString(`</w:tbl>`)
	b.AddSpacer()
}

// AddFailedDetailsTable adds a table showing failed test details
func (b *DocxBuilder) AddFailedDetailsTable(title string, failed []*db.FailedDetails) {
	if len(failed) == 0 {
		return
	}

	b.AddHeading(title, 4)

	b.documentContent.WriteString(`<w:tbl>`)
	b.documentContent.WriteString(`<w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="9000" w:type="dxa"/><w:tblBorders><w:top w:val="single" w:sz="4" w:color="CCCCCC"/><w:left w:val="single" w:sz="4" w:color="CCCCCC"/><w:bottom w:val="single" w:sz="4" w:color="CCCCCC"/><w:right w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideH w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideV w:val="single" w:sz="4" w:color="CCCCCC"/></w:tblBorders></w:tblPr>`)

	// Header row
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableHeaderCell("Payload", 2500)
	b.addTableHeaderCell("测试用例", 1500)
	b.addTableHeaderCell("编码器", 1500)
	b.addTableHeaderCell("占位符", 1500)
	b.addTableHeaderCell("失败原因", 2000)
	b.documentContent.WriteString(`</w:tr>`)

	// Data rows
	for _, f := range failed {
		b.documentContent.WriteString(`<w:tr>`)
		b.addTableCellMono(truncatePayloadForTable(f.Payload), 2500)
		b.addTableCell(f.TestCase, 1500, false, "")
		b.addTableCell(f.Encoder, 1500, false, "")
		b.addTableCell(f.Placeholder, 1500, false, "")
		b.addTableCell(strings.Join(f.Reason, "; "), 2000, false, "")
		b.documentContent.WriteString(`</w:tr>`)
	}

	b.documentContent.WriteString(`</w:tbl>`)
	b.AddSpacer()
}

// AddScannedPathsTable adds a table showing scanned paths
func (b *DocxBuilder) AddScannedPathsTable(paths db.ScannedPaths) {
	if len(paths) == 0 {
		return
	}

	b.AddHeading("扫描路径", 3)
	b.AddParagraph(fmt.Sprintf("共扫描 %d 个端点", len(paths)))

	b.documentContent.WriteString(`<w:tbl>`)
	b.documentContent.WriteString(`<w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="9000" w:type="dxa"/><w:tblBorders><w:top w:val="single" w:sz="4" w:color="CCCCCC"/><w:left w:val="single" w:sz="4" w:color="CCCCCC"/><w:bottom w:val="single" w:sz="4" w:color="CCCCCC"/><w:right w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideH w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideV w:val="single" w:sz="4" w:color="CCCCCC"/></w:tblBorders></w:tblPr>`)

	// Header row
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableHeaderCell("方法", 1500)
	b.addTableHeaderCell("路径", 7500)
	b.documentContent.WriteString(`</w:tr>`)

	// Data rows
	for _, path := range paths {
		b.documentContent.WriteString(`<w:tr>`)
		b.addTableCell(path.Method, 1500, true, "")
		b.addTableCellMono(path.Path, 7500)
		b.documentContent.WriteString(`</w:tr>`)
	}

	b.documentContent.WriteString(`</w:tbl>`)
	b.AddSpacer()
}

// AddRiskLevelTable adds a risk level description table
func (b *DocxBuilder) AddRiskLevelTable() {
	b.AddHeading("风险等级说明", 3)

	b.documentContent.WriteString(`<w:tbl>`)
	b.documentContent.WriteString(`<w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="9000" w:type="dxa"/><w:tblBorders><w:top w:val="single" w:sz="4" w:color="CCCCCC"/><w:left w:val="single" w:sz="4" w:color="CCCCCC"/><w:bottom w:val="single" w:sz="4" w:color="CCCCCC"/><w:right w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideH w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideV w:val="single" w:sz="4" w:color="CCCCCC"/></w:tblBorders></w:tblPr>`)

	// Header row
	b.documentContent.WriteString(`<w:tr>`)
	b.addTableHeaderCell("等级", 1500)
	b.addTableHeaderCell("分数范围", 2000)
	b.addTableHeaderCell("风险描述", 5500)
	b.documentContent.WriteString(`</w:tr>`)

	// Risk level rows
	risks := []struct {
		level  string
		range_ string
		desc   string
		color  string
	}{
		{"优秀", "≥97%", "WAF配置优秀，能有效防护各类攻击，安全风险极低", "56CC54"},
		{"良好", "80-96%", "WAF配置良好，建议优化个别规则以提升防护效果", "FDBE10"},
		{"中等", "70-79%", "WAF存在一定防护盲区，需要针对性调优规则", "FC7303"},
		{"及格", "60-69%", "WAF配置存在明显问题，需要重点优化核心规则", "F26344"},
		{"不及格", "<60%", "WAF几乎无防护能力，建议重新评估安全策略", "F24444"},
	}

	for _, risk := range risks {
		b.documentContent.WriteString(`<w:tr>`)
		b.addTableCell(risk.level, 1500, true, "")
		b.addTableCell(risk.range_, 2000, false, "")
		b.addTableCell(risk.desc, 5500, false, "")
		b.documentContent.WriteString(`</w:tr>`)
	}

	b.documentContent.WriteString(`</w:tbl>`)
	b.AddSpacer()
}

func getPercentageColor(percentage float64) string {
	switch {
	case percentage >= 90:
		return "E1F9D9" // Light green
	case percentage >= 70:
		return "FEF2B9" // Light yellow
	case percentage >= 50:
		return "FEE1B4" // Light orange
	default:
		return "f8d2c4" // Light red
	}
}

func (b *DocxBuilder) addTableHeaderCell(text string, width int) {
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/><w:shd w:val="clear" w:color="auto" w:fill="3942EA"/></w:tcPr><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="20"/><w:color w:val="FFFFFF"/></w:rPr><w:t>%s</w:t></w:r></w:p></w:tc>`,
		width*10, escapeXML(text)))
}

func (b *DocxBuilder) addTableCell(text string, width int, bold bool, bgColor string) {
	bgColorAttr := ""
	if bgColor != "" {
		bgColorAttr = fmt.Sprintf(`<w:shd w:val="clear" w:color="auto" w:fill="%s"/>`, bgColor)
	}
	boldAttr := ""
	if bold {
		boldAttr = `<w:b/>`
	}
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/>%s</w:tcPr><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr>%s<w:sz w:val="20"/></w:rPr><w:t>%s</w:t></w:r></w:p></w:tc>`,
		width*10, bgColorAttr, boldAttr, escapeXML(text)))
}

func (b *DocxBuilder) addTableCellMono(text string, width int) {
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/></w:tcPr><w:p><w:pPr><w:jc w:val="left"/></w:pPr><w:r><w:rPr><w:rFonts w:ascii="Courier New" w:hAnsi="Courier New" w:cs="Courier New"/><w:sz w:val="18"/></w:rPr><w:t>%s</w:t></w:r></w:p></w:tc>`,
		width*10, escapeXML(text)))
}

func (b *DocxBuilder) addGradeCell(grade *report.Grade, width int) {
	color := GradeColors[grade.CSSClassSuffix]
	bgColor := GradeBgColors[grade.CSSClassSuffix]
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/><w:shd w:val="clear" w:color="auto" w:fill="%s"/></w:tcPr><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="20"/><w:color w:val="%s"/></w:rPr><w:t>%s (%.1f%%)</w:t></w:r></w:p></w:tc>`,
		width*10, bgColor, color, grade.Mark, grade.Percentage))
}

func (b *DocxBuilder) addGradeCellWithBg(grade *report.Grade, width int, bgColor string) {
	color := GradeColors[grade.CSSClassSuffix]
	b.documentContent.WriteString(fmt.Sprintf(
		`<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/><w:shd w:val="clear" w:color="auto" w:fill="%s"/></w:tcPr><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="20"/><w:color w:val="%s"/></w:rPr><w:t>%s (%.1f%%)</w:t></w:r></w:p></w:tc>`,
		width*10, bgColor, color, grade.Mark, grade.Percentage))
}

func truncatePayloadForTable(payload string) string {
	if len(payload) > 100 {
		return payload[:97] + "..."
	}
	return payload
}

// GetContent returns the document content
func (b *DocxBuilder) GetContent() string {
	return b.documentContent.String()
}

// renderDocxReport renders the DOCX report
func renderDocxReport(data *DocxReportData) (*bytes.Buffer, error) {
	builder := NewDocxBuilder()

	// ==================== 标题部分 ====================
	builder.AddHeading("GoTestWAF 安全测试报告", 1)
	builder.AddParagraph("")
	builder.AddParagraph("API / Application Security Testing Results")
	builder.AddSpacer()

	// ==================== 综合评分卡片 ====================
	builder.AddHeading("综合评分", 2)

	// Overall score highlight
	overallColor := GradeColors[data.OverallScore.CSSClassSuffix]
	overallBg := GradeBgColors[data.OverallScore.CSSClassSuffix]
	builder.documentContent.WriteString(fmt.Sprintf(
		`<w:tbl><w:tblPr><w:tblW w:w="9000" w:type="dxa"/><w:tblBorders><w:top w:val="nil"/><w:left w:val="nil"/><w:bottom w:val="nil"/><w:right w:val="nil"/></w:tblBorders></w:tblPr><w:tr><w:tc><w:tcPr><w:tcW w:w="3000" w:type="dxa"/><w:shd w:val="clear" w:fill="%s"/></w:tcPr><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="72"/><w:color w:val="%s"/></w:rPr><w:t>%s</w:t></w:r></w:p><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:sz w:val="28"/><w:color w:val="%s"/></w:rPr><w:t>%.1f / 100</w:t></w:r></w:p></w:tc><w:tc><w:tcPr><w:tcW w:w="6000" w:type="dxa"/></w:tcPr>`,
		overallBg, overallColor, data.OverallScore.Mark, overallColor, data.OverallScore.Percentage))

	// Project info
	infoContent := fmt.Sprintf(`<w:p><w:pPr><w:spacing w:before="60" w:after="60"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="22"/></w:rPr><w:t>项目名称：</w:t></w:r><w:r><w:rPr><w:sz w:val="22"/></w:rPr><w:t>%s</w:t></w:r></w:p>`, escapeXML(data.WafName))
	infoContent += fmt.Sprintf(`<w:p><w:pPr><w:spacing w:before="40" w:after="40"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="22"/></w:rPr><w:t>目标URL：</w:t></w:r><w:r><w:rPr><w:sz w:val="22"/></w:rPr><w:t>%s</w:t></w:r></w:p>`, escapeXML(data.Url))
	infoContent += fmt.Sprintf(`<w:p><w:pPr><w:spacing w:before="40" w:after="40"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="22"/></w:rPr><w:t>测试日期：</w:t></w:r><w:r><w:rPr><w:sz w:val="22"/></w:rPr><w:t>%s</w:t></w:r></w:p>`, escapeXML(data.WafTestingDate))
	infoContent += fmt.Sprintf(`<w:p><w:pPr><w:spacing w:before="40" w:after="40"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="22"/></w:rPr><w:t>工具版本：</w:t></w:r><w:r><w:rPr><w:sz w:val="22"/></w:rPr><w:t>%s</w:t></w:r></w:p>`, escapeXML(data.GtwVersion))
	infoContent += fmt.Sprintf(`<w:p><w:pPr><w:spacing w:before="40" w:after="40"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="22"/></w:rPr><w:t>测试用例指纹：</w:t></w:r><w:r><w:rPr><w:sz w:val="22"/></w:rPr><w:t>%s</w:t></w:r></w:p>`, escapeXML(data.TestCasesFP))
	if data.OpenApiFile != "" {
		infoContent += fmt.Sprintf(`<w:p><w:pPr><w:spacing w:before="40" w:after="40"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="22"/></w:rPr><w:t>OpenAPI文件：</w:t></w:r><w:r><w:rPr><w:sz w:val="22"/></w:rPr><w:t>%s</w:t></w:r></w:p>`, escapeXML(data.OpenApiFile))
	}
	if len(data.Args) > 0 {
		infoContent += fmt.Sprintf(`<w:p><w:pPr><w:spacing w:before="40" w:after="40"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="22"/></w:rPr><w:t>命令行参数：</w:t></w:r><w:r><w:rPr><w:sz w:val="20"/></w:rPr><w:t>%s</w:t></w:r></w:p>`, escapeXML(strings.Join(data.Args, " ")))
	}
	builder.documentContent.WriteString(infoContent)
	builder.documentContent.WriteString(`</w:tc></w:tr></w:tbl>`)
	builder.AddSpacer()

	// ==================== 评分详情 ====================
	builder.AddHeading("评分详情", 3)
	builder.AddGradeTable(data)

	// ==================== 基准对比 ====================
	builder.AddHeading("与其他解决方案对比", 2)
	builder.AddParagraph("以下是您的WAF与其他常见安全解决方案的性能对比：")
	builder.AddSpacer()
	builder.AddBenchmarkTable(data)

	// ==================== 请求统计 ====================
	builder.AddHeading("请求统计", 2)
	builder.AddSummaryTable(data)

	// ==================== 真正例测试 ====================
	builder.AddHeading("真正例测试（True-Positive Tests）", 2)
	builder.AddParagraph("真正例测试是指恶意请求，应该被WAF拦截。")
	builder.AddKeyValueParagraph("总体得分", fmt.Sprintf("%.2f%%", data.TruePositiveTests.Score), true)
	builder.AddKeyValueParagraph("发送请求数", fmt.Sprintf("%d", data.TruePositiveTests.TotalSent), false)
	builder.AddKeyValueParagraph("已拦截", fmt.Sprintf("%d", data.TruePositiveTests.Blocked), false)
	builder.AddKeyValueParagraph("已绕过", fmt.Sprintf("%d", data.TruePositiveTests.Bypassed), false)
	if !data.IgnoreUnresolved {
		builder.AddKeyValueParagraph("未确定", fmt.Sprintf("%d", data.TruePositiveTests.Unresolved), false)
	}
	builder.AddKeyValueParagraph("失败", fmt.Sprintf("%d", data.TruePositiveTests.Failed), false)
	builder.AddSpacer()

	if len(data.TruePositiveTests.TestSets) > 0 {
		builder.AddTestSetTable(data.TruePositiveTests.TestSets, true, data.IgnoreUnresolved)
	}

	// ==================== 真负例测试 ====================
	builder.AddHeading("真负例测试（True-Negative Tests）", 2)
	builder.AddParagraph("真负例测试是指正常请求，应该通过WAF而不被拦截。")
	builder.AddKeyValueParagraph("总体得分", fmt.Sprintf("%.2f%%", data.TrueNegativeTests.Score), true)
	builder.AddKeyValueParagraph("发送请求数", fmt.Sprintf("%d", data.TrueNegativeTests.TotalSent), false)
	builder.AddKeyValueParagraph("已通过", fmt.Sprintf("%d", data.TrueNegativeTests.Bypassed), false)
	builder.AddKeyValueParagraph("已拦截（误报）", fmt.Sprintf("%d", data.TrueNegativeTests.Blocked), false)
	if !data.IgnoreUnresolved {
		builder.AddKeyValueParagraph("未确定", fmt.Sprintf("%d", data.TrueNegativeTests.Unresolved), false)
	}
	builder.AddKeyValueParagraph("失败", fmt.Sprintf("%d", data.TrueNegativeTests.Failed), false)
	builder.AddSpacer()

	if len(data.TrueNegativeTests.TestSets) > 0 {
		builder.AddTestSetTable(data.TrueNegativeTests.TestSets, false, data.IgnoreUnresolved)
	}

	// ==================== 扫描路径 ====================
	if len(data.ScannedPaths) > 0 {
		builder.AddScannedPathsTable(data.ScannedPaths)
	}

	// ==================== Payload详情 ====================
	if data.IncludePayloads {
		builder.AddHorizontalLine()
		builder.AddHeading("详细测试结果", 2)

		// True Negative - Blocked (误报)
		if len(data.TnBlocked) > 0 {
			builder.AddHeading("真负例中被拦截的请求（误报）", 3)
			builder.AddParagraph(fmt.Sprintf("共 %d 个正常请求被错误拦截", len(data.TnBlocked)))
			builder.AddPayloadDetailsTable("误报详情", data.TnBlocked, false)
		}

		// True Negative - Unresolved
		if len(data.TnUnresolved) > 0 && !data.IgnoreUnresolved {
			builder.AddHeading("真负例中未确定的请求", 3)
			builder.AddPayloadDetailsTable("未确定详情", data.TnUnresolved, false)
		}

		// True Negative - Failed
		if len(data.TnFailed) > 0 {
			builder.AddFailedDetailsTable("真负例失败的请求", data.TnFailed)
		}

		// True Positive - Bypassed (绕过)
		if len(data.TpBypassed) > 0 {
			builder.AddHeading("真正例中绕过的请求（漏报）", 3)
			builder.AddParagraph("以下恶意请求成功绕过了WAF，这是安全风险。")
			for path, payloadMap := range data.TpBypassed {
				if path != "" {
					builder.AddParagraph(fmt.Sprintf("路径: %s", path))
				}
				builder.AddPayloadDetailsTable("绕过详情", payloadMap, true)
			}
		}

		// True Positive - Unresolved
		if len(data.TpUnresolved) > 0 && !data.IgnoreUnresolved {
			builder.AddHeading("真正例中未确定的请求", 3)
			builder.AddPayloadDetailsTable("未确定详情", data.TpUnresolved, true)
		}

		// True Positive - Failed
		if len(data.TpFailed) > 0 {
			builder.AddFailedDetailsTable("真正例失败的请求", data.TpFailed)
		}
	}

	// ==================== 风险等级说明 ====================
	builder.AddHorizontalLine()
	builder.AddRiskLevelTable()

	// ==================== 报告页脚 ====================
	builder.AddHorizontalLine()
	builder.AddParagraph("")
	builder.AddParagraph("报告由 GoTestWAF 自动生成")
	builder.AddParagraph(fmt.Sprintf("生成时间: %s", time.Now().Format("2006年01月02日 15:04:05")))
	builder.AddParagraph("")
	builder.AddParagraph("更多信息请访问: https://github.com/wallarm/gotestwaf")

	// Build the complete DOCX file
	return buildDocxFile(builder.GetContent())
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
<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
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
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`

	if err := addFileToZip(zipWriter, "word/_rels/document.xml.rels", documentRels); err != nil {
		return nil, err
	}

	// word/styles.xml - Professional styles
	stylesXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:docDefaults>
<w:rPrDefault><w:rPr><w:rFonts w:ascii="Arial" w:eastAsia="微软雅黑" w:hAnsi="Arial" w:cs="Arial"/><w:sz w:val="22"/></w:rPr></w:rPrDefault>
<w:pPrDefault><w:pPr><w:spacing w:before="100" w:after="100"/></w:pPr></w:pPrDefault>
</w:docDefaults>
<w:style w:type="paragraph" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style>
<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="Heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:before="240" w:after="120"/><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:rFonts w:ascii="Arial" w:eastAsia="微软雅黑" w:hAnsi="Arial"/><w:b/><w:sz w:val="48"/><w:color w:val="3942EA"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="Heading 2"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:before="200" w:after="100"/><w:outlineLvl w:val="1"/></w:pPr><w:rPr><w:rFonts w:ascii="Arial" w:eastAsia="微软雅黑" w:hAnsi="Arial"/><w:b/><w:sz w:val="36"/><w:color w:val="000000"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading3"><w:name w:val="Heading 3"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:before="160" w:after="80"/><w:outlineLvl w:val="2"/></w:pPr><w:rPr><w:rFonts w:ascii="Arial" w:eastAsia="微软雅黑" w:hAnsi="Arial"/><w:b/><w:sz w:val="28"/><w:color w:val="333333"/></w:rPr></w:style>
<w:style w:type="table" w:styleId="TableGrid"><w:name w:val="Table Grid"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="0" w:after="0"/></w:pPr><w:tblPr><w:tblBorders><w:top w:val="single" w:sz="4" w:color="CCCCCC"/><w:left w:val="single" w:sz="4" w:color="CCCCCC"/><w:bottom w:val="single" w:sz="4" w:color="CCCCCC"/><w:right w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideH w:val="single" w:sz="4" w:color="CCCCCC"/><w:insideV w:val="single" w:sz="4" w:color="CCCCCC"/></w:tblBorders></w:tblPr></w:style>
</w:styles>`

	if err := addFileToZip(zipWriter, "word/styles.xml", stylesXML); err != nil {
		return nil, err
	}

	// word/document.xml
	documentXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<w:body>
%s
<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1080" w:bottom="1440" w:left="1080" w:header="720" w:footer="720"/><w:cols w:space="720"/></w:sectPr>
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

// getChineseGrade returns a grade with Chinese mark
func getChineseGrade(percentage float64, na bool) *report.Grade {
	g := &report.Grade{
		Percentage:     0.0,
		Mark:           "不适用",
		CSSClassSuffix: "na",
	}

	if na {
		return g
	}

	g.Percentage = percentage
	if g.Percentage <= 1 {
		g.Percentage *= 100
	}

	switch {
	case g.Percentage >= 97.0:
		g.Mark = "优秀+"
		g.CSSClassSuffix = "a"
	case g.Percentage >= 93.0:
		g.Mark = "优秀"
		g.CSSClassSuffix = "a"
	case g.Percentage >= 90.0:
		g.Mark = "优秀-"
		g.CSSClassSuffix = "a"
	case g.Percentage >= 87.0:
		g.Mark = "良好+"
		g.CSSClassSuffix = "b"
	case g.Percentage >= 83.0:
		g.Mark = "良好"
		g.CSSClassSuffix = "b"
	case g.Percentage >= 80.0:
		g.Mark = "良好-"
		g.CSSClassSuffix = "b"
	case g.Percentage >= 77.0:
		g.Mark = "中等+"
		g.CSSClassSuffix = "c"
	case g.Percentage >= 73.0:
		g.Mark = "中等"
		g.CSSClassSuffix = "c"
	case g.Percentage >= 70.0:
		g.Mark = "中等-"
		g.CSSClassSuffix = "c"
	case g.Percentage >= 67.0:
		g.Mark = "及格+"
		g.CSSClassSuffix = "d"
	case g.Percentage >= 63.0:
		g.Mark = "及格"
		g.CSSClassSuffix = "d"
	case g.Percentage >= 60.0:
		g.Mark = "及格-"
		g.CSSClassSuffix = "d"
	case g.Percentage < 60.0:
		g.Mark = "不及格"
		g.CSSClassSuffix = "f"
	}

	return g
}

// writeDocxToFile writes the buffer content to a file
func writeDocxToFile(buf *bytes.Buffer, filename string) error {
	data := buf.Bytes()

	// Validate it's a valid ZIP file (DOCX is a ZIP archive)
	if len(data) < 4 {
		return errors.New("generated content is empty")
	}

	return os.WriteFile(filename, data, 0644)
}

// GenerateDocxReport generates DOCX report bytes
func GenerateDocxReport(
	s *db.Statistics, reportTime time.Time, wafName string,
	url string, openApiFile string, args []string,
	ignoreUnresolved bool, includePayloads bool,
) ([]byte, error) {
	data := prepareDocxReportData(s, reportTime, wafName, url, openApiFile, args, ignoreUnresolved, includePayloads)

	buf, err := renderDocxReport(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}