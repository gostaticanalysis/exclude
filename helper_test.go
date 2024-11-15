package exclude_test

import (
	"testing"

	"github.com/gostaticanalysis/exclude"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

type reportRecoder struct {
	reports []analysis.Diagnostic
}

func (r *reportRecoder) new(f exclude.Func) *analysis.Analyzer {
	a := exclude.By(&analysis.Analyzer{
		Name: "TestAnalyzer",
		Doc: "document",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			if pass.Pkg.Name() == "main" ||
				len(pass.Files) == 0 {
				return nil, nil
			}

			pass.Reportf(pass.Files[0].Pos(), "hello")
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
