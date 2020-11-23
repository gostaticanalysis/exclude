package exclude

import "golang.org/x/tools/go/analysis"

type DiagnosticFilter func(d analysis.Diagnostic) bool

func ReportWithFilter(pass *analysis.Pass, filters ...DiagnosticFilter) func(analysis.Diagnostic) {
	// original reporter
	report := pass.Report

	for _, filter := range filters {
		filter := filter
		orgReport := report
		report = func(d analysis.Diagnostic) {
			if filter(d) {
				orgReport(d)
			}
		}
	}

	return report
}
