package detect

import (
	"os"
	"path/filepath"
)

type Result struct {
	Type  string
	Build string
	Test  string
	Run   string
	Lint  string
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasGlob(dir, pattern string) bool {
	matches, _ := filepath.Glob(filepath.Join(dir, pattern))
	return len(matches) > 0
}

// Detect scans dir for known project markers, mirroring
// scripts/detect-project.sh's detection order (first match wins).
func Detect(dir string) Result {
	has := func(name string) bool { return exists(filepath.Join(dir, name)) }

	switch {
	case has("sdkconfig") || has("idf_component.yml"):
		return Result{Type: "esp-idf",
			Build: ". ~/esp/esp-idf/export.sh && idf.py build",
			Test:  ". ~/esp/esp-idf/export.sh && idf.py build 2>&1 | tail -5",
			Run:   ". ~/esp/esp-idf/export.sh && idf.py flash monitor"}
	case has("Cargo.toml"):
		return Result{Type: "rust", Build: "cargo build", Test: "cargo test",
			Run: "cargo run", Lint: "cargo clippy -- -D warnings"}
	case has("package.json"):
		pm := "npm"
		if has("pnpm-lock.yaml") {
			pm = "pnpm"
		} else if has("yarn.lock") {
			pm = "yarn"
		}
		return Result{Type: "nodejs", Build: pm + " run build", Test: pm + " test",
			Run: pm + " start", Lint: pm + " run lint"}
	case has("pyproject.toml") || has("setup.py") || has("requirements.txt"):
		return Result{Type: "python", Build: "pip install -e .", Test: "pytest -x",
			Run: "python main.py", Lint: "ruff check ."}
	case has("go.mod"):
		return Result{Type: "go", Build: "go build ./...", Test: "go test ./...",
			Run: "go run .", Lint: "golangci-lint run"}
	case hasGlob(dir, "*.sln"):
		return Result{Type: "csharp", Build: "dotnet build", Test: "dotnet test",
			Lint: "dotnet format --verify-no-changes"}
	case hasGlob(dir, "*.csproj"):
		return Result{Type: "csharp", Build: "dotnet build", Test: "dotnet test",
			Lint: "dotnet format --verify-no-changes", Run: "dotnet run"}
	case has("CMakeLists.txt"):
		return Result{Type: "c-cpp", Build: "cmake -B build && cmake --build build",
			Test: "ctest --test-dir build"}
	case has("Makefile") || has("makefile"):
		return Result{Type: "make", Build: "make", Test: "make test", Run: "make run"}
	default:
		return Result{Type: "unknown"}
	}
}
