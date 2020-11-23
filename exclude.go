package exclude

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
)

// Func excludes reporting diagnostics or analyzing.
type Func func(a *analysis.Analyzer) *analysis.Analyzer

// By excludes with the functions.
func By(a *analysis.Analyzer, fs ...Func) *analysis.Analyzer {
	analyzer := a
	for _, f := range fs {
		analyzer = f(analyzer)
	}
	return analyzer
}

// GeneratedFile excludes auto generated files.
// Because it excludes only reporting diagnostics, analyzing would be excuted.
func GeneratedFile(a *analysis.Analyzer) *analysis.Analyzer {
	orgRun := a.Run
	a.Run = func(pass *analysis.Pass) (interface{}, error) {
		pass.Report = ReportWithFilter(pass, func(d analysis.Diagnostic) bool {
			return analysisutil.IsGeneratedFile(analysisutil.File(pass, d.Pos))
		})
		return orgRun(pass)
	}
	return a
}

// TestFile excludes test files.
// Because it excludes only reporting diagnostics, analyzing would be excuted.
func TestFile(a *analysis.Analyzer) *analysis.Analyzer {
	orgRun := a.Run
	a.Run = func(pass *analysis.Pass) (interface{}, error) {
		pass.Report = ReportWithFilter(pass, func(d analysis.Diagnostic) bool {
			file := pass.Fset.File(d.Pos)
			return strings.HasSuffix(file.Name(), "_test.go")
		})
		return orgRun(pass)
	}
	return a
}

// FilePattern excludes files which matches the pattern given by exclude flag.
// Because it excludes only reporting diagnostics, analyzing would be excuted.
func FilePattern(a *analysis.Analyzer) *analysis.Analyzer {
	if a.Flags.Lookup("exclude") != nil {
		panic("flag -exclude has already been set")
	}
	var pattern string
	a.Flags.StringVar(&pattern, "exclude", "", "a pattern of excluding file path")

	orgRun := a.Run
	a.Run = func(pass *analysis.Pass) (interface{}, error) {
		if pattern == "" {
			return orgRun(pass)
		}

		excludeRegexp, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("exclude path: %w", err)
		}

		pass.Report = ReportWithFilter(pass, func(d analysis.Diagnostic) bool {
			file := pass.Fset.File(d.Pos)
			return !excludeRegexp.MatchString(file.Name())
		})

		return orgRun(pass)
	}

	return a
}
