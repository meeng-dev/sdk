package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every language writes its client, a service per resource, the models and the
// webhooks, and says nothing of what the contract does not have.
func TestEveryLanguageIsWritten(t *testing.T) {
	ir := loadContract(t)
	want := map[string][]string{
		"go":         {"go.mod", "client.go", "services.go", "models.go", "webhooks.go"},
		"typescript": {"package.json", "src/client.ts", "src/services.ts", "src/models.ts", "src/webhooks.ts"},
		"python":     {"pyproject.toml", "src/meeng/_client.py", "src/meeng/_services.py", "src/meeng/models.py", "src/meeng/webhooks.py"},
		"php":        {"composer.json", "src/Meeng.php", "src/Services/EmailsService.php", "src/Webhooks.php"},
		"ruby":       {"meeng-sdk.gemspec", "lib/meeng/client.rb", "lib/meeng/services.rb", "lib/meeng/webhooks.rb"},
		"java":       {"pom.xml", "src/main/java/dev/meeng/Meeng.java", "src/main/java/dev/meeng/EmailsService.java", "src/main/java/dev/meeng/Domain.java", "src/main/java/dev/meeng/EmailWebhookEvent.java"},
		"csharp":     {"Meeng.slnx", "src/Meeng/MeengClient.cs", "src/Meeng/Services/EmailsService.cs", "src/Meeng/Models/Domain.cs", "src/Meeng/Webhooks.cs"},
		"elixir":     {"mix.exs", "lib/meeng.ex", "lib/meeng/services.ex", "lib/meeng/webhooks.ex"},
		"dart":       {"pubspec.yaml", "lib/src/client.dart", "lib/src/services.dart", "lib/src/models.dart", "lib/src/webhooks.dart"},
		"rust":       {"Cargo.toml", "src/client.rs", "src/services.rs", "src/models.rs", "src/webhooks.rs"},
		"swift":      {"Package.swift", "Sources/Meeng/MeengClient.swift", "Sources/Meeng/Services.swift", "Sources/Meeng/Models.swift", "Sources/Meeng/Webhooks.swift"},
	}
	if len(Languages) != len(want) {
		t.Errorf("%d languages are registered, want %d", len(Languages), len(want))
	}
	for language, files := range want {
		out := t.TempDir()
		if err := Generate(ir, language, out); err != nil {
			t.Errorf("%s: %v", language, err)
			continue
		}
		for _, file := range files {
			if _, err := os.Stat(filepath.Join(out, file)); err != nil {
				t.Errorf("%s: %v", language, err)
			}
		}
	}
}

// What the generator writes does not depend on the run.
func TestGenerationIsDeterministic(t *testing.T) {
	ir := loadContract(t)
	for language := range Languages {
		first, second := t.TempDir(), t.TempDir()
		if err := Generate(ir, language, first); err != nil {
			t.Fatal(err)
		}
		if err := Generate(ir, language, second); err != nil {
			t.Fatal(err)
		}
		_ = filepath.WalkDir(first, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(first, path)
			a, _ := os.ReadFile(path)
			b, _ := os.ReadFile(filepath.Join(second, rel))
			if string(a) != string(b) {
				t.Errorf("%s: %s differs between two runs", language, rel)
			}
			return nil
		})
	}
}

// No SDK carries the product the generator was taken from, or a name nobody
// confirmed as published.
func TestNoLanguageCarriesAForeignName(t *testing.T) {
	ir := loadContract(t)
	for language := range Languages {
		out := t.TempDir()
		if err := Generate(ir, language, out); err != nil {
			t.Fatal(err)
		}
		_ = filepath.WalkDir(out, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			raw, _ := os.ReadFile(path)
			lower := strings.ToLower(string(raw))
			for _, foreign := range []string{"peráta", "perata", "prk_test", "prk_live", "showcase_id", "pus_live"} {
				if strings.Contains(lower, foreign) {
					t.Errorf("%s: %s mentions %q", language, path, foreign)
				}
			}
			return nil
		})
	}
}

// Every SDK is written under one version.
func TestEverySDKCarriesTheSameVersion(t *testing.T) {
	ir := loadContract(t)
	for language := range Languages {
		out := t.TempDir()
		if err := Generate(ir, language, out); err != nil {
			t.Fatal(err)
		}
		found := false
		_ = filepath.WalkDir(out, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			raw, _ := os.ReadFile(path)
			if strings.Contains(string(raw), `"`+ir.Version+`"`) || strings.Contains(string(raw), `'`+ir.Version+`'`) || strings.Contains(string(raw), "version: "+ir.Version) || strings.Contains(string(raw), "<Version>"+ir.Version) {
				found = true
			}
			return nil
		})
		if !found {
			t.Errorf("%s never says its version %s", language, ir.Version)
		}
	}
}
