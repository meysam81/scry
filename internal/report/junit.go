package report

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/meysam81/scry/core/model"
)

// JUnitReporter writes the CrawlResult issues as JUnit XML.
type JUnitReporter struct{}

// Name returns "junit".
func (r *JUnitReporter) Name() string { return "junit" }

// junitTestSuites is the top-level <testsuites> element.
type junitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Name       string           `xml:"name,attr"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Time       string           `xml:"time,attr"`
	TestSuites []junitTestSuite `xml:"testsuite"`
}

// junitTestSuite is a single <testsuite> element grouping tests by category.
type junitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

// junitTestCase is a single <testcase> element.
type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

// junitFailure is the <failure> child of a testcase.
type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

// issueCategory extracts the category prefix from a check name.
// For "seo/missing-title" it returns "seo". If there is no slash, the
// full check name is the category.
func issueCategory(checkName string) string {
	if idx := strings.IndexByte(checkName, '/'); idx >= 0 {
		return checkName[:idx]
	}
	return checkName
}

// Write renders result as JUnit XML and writes it to w.
func (r *JUnitReporter) Write(_ context.Context, result *model.CrawlResult, w io.Writer) error {
	if result == nil {
		return nil
	}

	// Group issues by category, preserving encounter order.
	type suiteData struct {
		name  string
		cases []junitTestCase
	}
	seen := make(map[string]int) // category -> index into suites slice
	var suites []suiteData

	for _, iss := range result.Issues {
		cat := issueCategory(iss.CheckName)
		idx, ok := seen[cat]
		if !ok {
			idx = len(suites)
			seen[cat] = idx
			suites = append(suites, suiteData{name: cat})
		}

		tc := junitTestCase{
			Name:      iss.CheckName,
			ClassName: iss.URL,
			Failure: &junitFailure{
				Message: iss.Message,
				Type:    string(iss.Severity),
				Body:    iss.Detail,
			},
		}
		suites[idx].cases = append(suites[idx].cases, tc)
	}

	totalTests := 0
	totalFailures := 0

	xmlSuites := make([]junitTestSuite, 0, len(suites))
	for _, sd := range suites {
		failures := len(sd.cases)
		xmlSuites = append(xmlSuites, junitTestSuite{
			Name:      sd.name,
			Tests:     len(sd.cases),
			Failures:  failures,
			TestCases: sd.cases,
		})
		totalTests += len(sd.cases)
		totalFailures += failures
	}

	doc := junitTestSuites{
		Name:       "scry",
		Tests:      totalTests,
		Failures:   totalFailures,
		Time:       fmt.Sprintf("%.3f", result.Duration.Seconds()),
		TestSuites: xmlSuites,
	}

	if _, err := io.WriteString(w, xml.Header); err != nil {
		return fmt.Errorf("writing XML header: %w", err)
	}

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("encoding JUnit XML: %w", err)
	}

	if _, err := io.WriteString(w, "\n"); err != nil {
		return fmt.Errorf("writing trailing newline: %w", err)
	}

	return nil
}
