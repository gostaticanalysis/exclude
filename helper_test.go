package exclude_test

import (
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/gostaticanalysis/exclude"
)

type reportRecoder struct {
	reports []analysis.Diagnostic
}

func (r *reportRecoder) new(f exclude.Func, line int) *analysis.Analyzer {
	a := exclude.By(&analysis.Analyzer{
		Name: "TestAnalyzer",
		Doc:  "document",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			if pass.Pkg.Name() == "main" ||
				len(pass.Files) == 0 {
				return nil, nil
			}

			var pos token.Pos
			if line >= 0 {
				file := pass.Fset.File(pass.Files[0].Pos())
				pos = file.LineStart(line)
			} else {
				pos = pass.Files[0].Pos()
			}

			pass.Reportf(pos, "hello")
			return nil, nil
		},
	}, f)

	orgRun := a.Run
	a.Run = func(pass *analysis.Pass) (interface{}, error) {
		pass.Report = func(d analysis.Diagnostic) {
			r.reports = append(r.reports, d)
		}
		return orgRun(pass)
	}

	return a
}

func (r *reportRecoder) isReported() bool {
	return len(r.reports) != 0
}

func writeFiles(t *testing.T, filemap map[string]string) string {
	t.Helper()
	dir, clean, err := analysistest.WriteFiles(filemap)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	t.Cleanup(clean)
	return dir
}
