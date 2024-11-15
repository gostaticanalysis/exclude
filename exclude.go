package exclude

import (
	"flag"
	"fmt"
	"io/ioutil"
	"regexp"
	"slices"
	"strings"

	"github.com/gostaticanalysis/analysisutil"
	"github.com/gostaticanalysis/comment"
	"github.com/gostaticanalysis/comment/passes/commentmap"
	"golang.org/x/tools/go/analysis"
)

// AllFuncs is all functions.
var AllFuncs = []Func{GeneratedFile, TestFile, FileWithPattern}

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

// All excludes the analyzers with the functions.
func All(as []*analysis.Analyzer, fs ...Func) []*analysis.Analyzer {
	excluded := make([]*analysis.Analyzer, len(as))
	for i := range as {
		excluded[i] = By(as[i], fs...)
	}
	return excluded
}

// Flags sets flags which name has "all-exclude" prefix to each analyzers.
// Flags returns remain arguments including flags which did not set to the analyzers.
func Flags(args []string, as ...*analysis.Analyzer) (remains []string, _ error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(ioutil.Discard)
	a := By(new(analysis.Analyzer), AllFuncs...)
	a.Flags.VisitAll(func(f *flag.Flag) {
		flags.Var(f.Value, "all-"+f.Name, f.Usage)
	})

	if err := flags.Parse(args); err != nil {
		return nil, err
	}

	var rerr error
	flags.Visit(func(f *flag.Flag) {
		if !strings.HasPrefix(f.Name, "all-exclude") {
			remains = append(remains, f.Name+"="+f.Value.String())
			return
		}

		for _, a := range as {
			// remove "all-" prefix and set each analyzer
			name := f.Name[4:]
			if a.Flags.Lookup(name) != nil {
				err := a.Flags.Set(f.Name[4:], f.Value.String())
				if err != nil {
					rerr = err
				}
			}
		}
	})

	if rerr != nil {
		return nil, rerr
	}

	remains = append(remains, flags.Args()...)

	if remains != nil {
		return remains, nil
	}

	return []string{}, nil
}

// GeneratedFile excludes auto generated files.
// Because it excludes only reporting diagnostics, analyzing would be excuted.
func GeneratedFile(a *analysis.Analyzer) *analysis.Analyzer {
	const flag = "exclude-generated"
	if a.Flags.Lookup(flag) != nil {
		panic("flag -" + flag + " has already been set")
	}
	var onoff bool
	a.Flags.BoolVar(&onoff, flag, true, "whether excludes generated files or not")
	if !onoff {
		// skip
		return a
	}

	orgRun := a.Run
	a.Run = func(pass *analysis.Pass) (interface{}, error) {
		pass.Report = ReportWithFilter(pass, func(d analysis.Diagnostic) bool {
			return !analysisutil.IsGeneratedFile(analysisutil.File(pass, d.Pos))
		})
		return orgRun(pass)
	}
	return a
}

// TestFile excludes test files.
// Because it excludes only reporting diagnostics, analyzing would be excuted.
func TestFile(a *analysis.Analyzer) *analysis.Analyzer {
	const flag = "exclude-testfile"
	if a.Flags.Lookup(flag) != nil {
		panic("flag -" + flag + " has already been set")
	}
	var onoff bool
	a.Flags.BoolVar(&onoff, flag, true, "whether excludes test files or not")
	if !onoff {
		// skip
		return a
	}

	orgRun := a.Run
	a.Run = func(pass *analysis.Pass) (interface{}, error) {
		pass.Report = ReportWithFilter(pass, func(d analysis.Diagnostic) bool {
			file := pass.Fset.File(d.Pos)
			return file != nil && !strings.HasSuffix(file.Name(), "_test.go")
		})
		return orgRun(pass)
	}
	return a
}

// FileWithPattern excludes files which matches the pattern given by exclude flag.
// Because it excludes only reporting diagnostics, analyzing would be excuted.
func FileWithPattern(a *analysis.Analyzer) *analysis.Analyzer {
	const flag = "exclude-file"
	if a.Flags.Lookup(flag) != nil {
		panic("flag -" + flag + " has already been set")
	}
	var pattern string
	a.Flags.StringVar(&pattern, flag, "", "a pattern of excluding file path")

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
			return file != nil && !excludeRegexp.MatchString(file.Name())
		})

		return orgRun(pass)
	}

	return a
}

// LintIgnoreComment excludes diagnostics with staticchek style //lint:ignore comment.
func LintIgnoreComment(a *analysis.Analyzer) *analysis.Analyzer {
	orgRun := a.Run

	if !slices.Contains(a.Requires, commentmap.Analyzer) {
		a.Requires = append(a.Requires, commentmap.Analyzer)
	}

	a.Run = func(pass *analysis.Pass) (interface{}, error) {
		pass.Report = ReportWithFilter(pass, func(d analysis.Diagnostic) bool {
			cmaps, _ := pass.ResultOf[commentmap.Analyzer].(comment.Maps)
			return !cmaps.IgnorePosLine(pass.Fset, d.Pos, a.Name)
		})
		return orgRun(pass)
	}
	return a
}
