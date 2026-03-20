package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/wallarm/gotestwaf/internal/db"
)

// TestDocxReportGeneration tests the DOCX report generation
func TestDocxReportGeneration(t *testing.T) {
	// Create mock statistics
	s := &db.Statistics{
		TestCasesFingerprint: "test-fingerprint-123",
		TruePositiveTests: db.TestsSummary{
			SummaryTable: []*db.SummaryTableRow{
				{
					TestSet:    "owasp",
					TestCase:   "sql-injection",
					Percentage: 85.5,
					Sent:       100,
					Blocked:    85,
					Bypassed:   15,
					Unresolved: 0,
					Failed:     0,
				},
			},
			ReqStats: db.RequestStats{
				AllRequestsNumber:      100,
				BlockedRequestsNumber:  85,
				BypassedRequestsNumber: 15,
			},
		},
		TrueNegativeTests: db.TestsSummary{
			SummaryTable: []*db.SummaryTableRow{
				{
					TestSet:    "false-pos",
					TestCase:   "texts",
					Percentage: 90.0,
					Sent:       50,
					Blocked:    5,
					Bypassed:   45,
					Unresolved: 0,
					Failed:     0,
				},
			},
			ReqStats: db.RequestStats{
				AllRequestsNumber:      50,
				BlockedRequestsNumber:  5,
				BypassedRequestsNumber: 45,
			},
		},
	}

	// Calculate score percentages
	s.TruePositiveTests.ResolvedBlockedRequestsPercentage = 85.0
	s.TrueNegativeTests.ResolvedBypassedRequestsPercentage = 90.0
	s.Score.Average = 87.5
	s.Score.ApiSec.Average = 85.0
	s.Score.ApiSec.TruePositive = 85.0
	s.Score.ApiSec.TrueNegative = -1.0
	s.Score.AppSec.Average = 87.5
	s.Score.AppSec.TruePositive = 87.5
	s.Score.AppSec.TrueNegative = 90.0

	reportTime := time.Now()

	// Test prepareDocxReportData
	data := prepareDocxReportData(s, reportTime, "测试WAF", "http://test.com", "", []string{"--url=http://test.com"}, false, true)

	if data == nil {
		t.Fatal("prepareDocxReportData returned nil")
	}

	if data.WafName != "测试WAF" {
		t.Errorf("Expected WafName '测试WAF', got '%s'", data.WafName)
	}

	if data.Url != "http://test.com" {
		t.Errorf("Expected URL 'http://test.com', got '%s'", data.Url)
	}

	// Check date format contains Chinese characters
	if !strings.Contains(data.WafTestingDate, "年") {
		t.Errorf("Expected Chinese date format, got '%s'", data.WafTestingDate)
	}

	// Test renderDocxReport
	buf, err := renderDocxReport(data)
	if err != nil {
		t.Fatalf("renderDocxReport failed: %v", err)
	}

	if buf == nil {
		t.Fatal("renderDocxReport returned nil buffer")
	}

	if buf.Len() == 0 {
		t.Fatal("renderDocxReport returned empty buffer")
	}

	// Check if it starts with PK (ZIP signature)
	bytes := buf.Bytes()
	if len(bytes) < 4 {
		t.Fatal("DOCX buffer too small")
	}

	if string(bytes[0:2]) != "PK" {
		t.Errorf("Expected DOCX to start with 'PK' (ZIP signature), got '%s'", string(bytes[0:2]))
	}
}

// TestEscapeXML tests the XML escaping function
func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"hello & world", "hello &amp; world"},
		{"<script>", "&lt;script&gt;"},
		{"test\"quote", "test&quot;quote"},
		{"test'apostrophe", "test&apos;apostrophe"},
		{"中文测试", "中文测试"},
	}

	for _, test := range tests {
		result := escapeXML(test.input)
		if result != test.expected {
			t.Errorf("escapeXML(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

// TestGetChineseGrade tests the Chinese grade function
func TestGetChineseGrade(t *testing.T) {
	tests := []struct {
		percentage float64
		na         bool
		expected   string
	}{
		{97.5, false, "优秀+"},
		{95.0, false, "优秀"},
		{91.0, false, "优秀-"},
		{88.0, false, "良好+"},
		{85.0, false, "良好"},
		{81.0, false, "良好-"},
		{78.0, false, "中等+"},
		{75.0, false, "中等"},
		{71.0, false, "中等-"},
		{68.0, false, "及格+"},
		{65.0, false, "及格"},
		{61.0, false, "及格-"},
		{55.0, false, "不及格"},
		{0.0, true, "不适用"},
	}

	for _, test := range tests {
		grade := getChineseGrade(test.percentage, test.na)
		if grade.Mark != test.expected {
			t.Errorf("getChineseGrade(%.1f, %v) = %s, expected %s", test.percentage, test.na, grade.Mark, test.expected)
		}
	}
}

// TestBuildDocxFile tests the DOCX file building
func TestBuildDocxFile(t *testing.T) {
	content := `<w:p><w:r><w:t>测试内容</w:t></w:r></w:p>`

	buf, err := buildDocxFile(content)
	if err != nil {
		t.Fatalf("buildDocxFile failed: %v", err)
	}

	if buf == nil {
		t.Fatal("buildDocxFile returned nil buffer")
	}

	// Verify it's a valid ZIP file (DOCX is a ZIP archive)
	bytes := buf.Bytes()
	if len(bytes) < 4 {
		t.Fatal("Generated DOCX is too small")
	}

	// ZIP files start with "PK"
	if string(bytes[0:2]) != "PK" {
		t.Errorf("Generated file does not appear to be a valid ZIP/DOCX file")
	}
}

// TestDocxBuilder tests the DocxBuilder methods
func TestDocxBuilder(t *testing.T) {
	builder := NewDocxBuilder()

	builder.AddHeading("测试标题", 1)
	builder.AddParagraph("测试段落")
	builder.AddBoldParagraph("标签", "值")
	builder.AddSpacer()

	content := builder.GetContent()

	if content == "" {
		t.Fatal("GetContent returned empty string")
	}

	// Check for expected elements
	if !bytes.Contains([]byte(content), []byte("测试标题")) {
		t.Error("Content missing heading text")
	}

	if !bytes.Contains([]byte(content), []byte("测试段落")) {
		t.Error("Content missing paragraph text")
	}
}

// TestAppendUnique tests the appendUnique function
func TestAppendUnique(t *testing.T) {
	slice := []string{"a", "b"}

	result := appendUnique(slice, "c")
	if len(result) != 3 {
		t.Errorf("Expected length 3, got %d", len(result))
	}

	result = appendUnique(slice, "a")
	if len(result) != 2 {
		t.Errorf("Expected length 2 (no duplicate), got %d", len(result))
	}
}

// TestGetPercentageColor tests the percentage color function
func TestGetPercentageColor(t *testing.T) {
	tests := []struct {
		percentage float64
		expected   string
	}{
		{95.0, "E1F9D9"},  // Light green
		{80.0, "FEF2B9"},  // Light yellow
		{65.0, "FEE1B4"},  // Light orange
		{40.0, "f8d2c4"},  // Light red
	}

	for _, test := range tests {
		result := getPercentageColor(test.percentage)
		if result != test.expected {
			t.Errorf("getPercentageColor(%.1f) = %s, expected %s", test.percentage, result, test.expected)
		}
	}
}

// TestTruncatePayloadForTable tests the payload truncation
func TestTruncatePayloadForTable(t *testing.T) {
	shortPayload := "short"
	result := truncatePayloadForTable(shortPayload)
	if result != shortPayload {
		t.Errorf("Expected %s, got %s", shortPayload, result)
	}

	longPayload := strings.Repeat("a", 150)
	result = truncatePayloadForTable(longPayload)
	if len(result) > 103 {
		t.Errorf("Expected truncated payload <= 103 chars, got %d", len(result))
	}
}