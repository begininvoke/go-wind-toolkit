package extract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newPlanTestExtractor 搭一个含 role 模型三文件(admin)的最小项目,
// 目标服务 svc2 不存在。
func newPlanTestExtractor(t *testing.T, keepSource bool) (*Extractor, *Plan) {
	t.Helper()
	root := t.TempDir()

	src := filepath.Join(root, "app", "admin", "service")
	for _, rel := range []string{
		filepath.Join("internal", "data", "ent", "schema", "role.go"),
		filepath.Join("internal", "data", "role_repo.go"),
		filepath.Join("internal", "service", "role_service.go"),
	} {
		p := filepath.Join(src, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	e := NewExtractor(Options{
		RootPath:      root,
		ModulePath:    "example.com/ext",
		SourceService: "admin",
		TargetService: "svc2",
		Models:        []string{"role"},
		OrmType:       "ent",
		KeepSource:    keepSource,
	})

	plan, err := e.Plan()
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	return e, plan
}

func TestPlanCopiesAndDeletes(t *testing.T) {
	_, plan := newPlanTestExtractor(t, false)

	if !plan.TargetWillBeCreated {
		t.Fatal("target svc2 does not exist, plan should mark scaffold creation")
	}
	if len(plan.CopyFiles) != 3 {
		t.Fatalf("expected 3 copy actions, got %d: %+v", len(plan.CopyFiles), plan.CopyFiles)
	}
	for _, c := range plan.CopyFiles {
		if c.Overwrite {
			t.Fatalf("fresh target must not be marked overwrite: %s", c.Dst)
		}
		if !strings.Contains(c.Src, "admin") || !strings.Contains(c.Dst, "svc2") {
			t.Fatalf("copy pair crosses wrong services: %+v", c)
		}
	}
	if len(plan.DeletedFiles) != 3 {
		t.Fatalf("expected 3 deletions (KeepSource=false), got %d", len(plan.DeletedFiles))
	}
	// 删除清单必须等于复制清单的源端。
	srcSet := map[string]bool{}
	for _, c := range plan.CopyFiles {
		srcSet[c.Src] = true
	}
	for _, d := range plan.DeletedFiles {
		if !srcSet[d] {
			t.Fatalf("deletion %s is not among copy sources", d)
		}
	}
	if len(plan.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", plan.Warnings)
	}
}

func TestPlanKeepSourceHasNoDeletions(t *testing.T) {
	_, plan := newPlanTestExtractor(t, true)

	if len(plan.DeletedFiles) != 0 {
		t.Fatalf("KeepSource plan must have no deletions, got %d", len(plan.DeletedFiles))
	}
	if len(plan.CopyFiles) != 3 {
		t.Fatalf("expected 3 copy actions, got %d", len(plan.CopyFiles))
	}
}

func TestPlanWarnsOnMissingSource(t *testing.T) {
	// 空项目:源服务不存在,全部源文件缺失 → 警告且无复制动作。
	e := NewExtractor(Options{
		RootPath:      t.TempDir(),
		ModulePath:    "example.com/ext",
		SourceService: "nope",
		TargetService: "svc2",
		Models:        []string{"ghost"},
		OrmType:       "ent",
	})
	plan, err := e.Plan()
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Warnings) == 0 {
		t.Fatal("missing source files must produce warnings")
	}
	if len(plan.CopyFiles) != 0 {
		t.Fatalf("missing sources must not produce copy actions, got %+v", plan.CopyFiles)
	}
}

func TestDedupeSorted(t *testing.T) {
	got := dedupeSorted([]string{"b", "a", "b", "c", "a"})
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("dedupeSorted = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dedupeSorted = %v, want %v", got, want)
		}
	}
}
