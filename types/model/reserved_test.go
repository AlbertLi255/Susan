package model

import "testing"

func TestIsReservedNamespaceExact(t *testing.T) {
	exact := []string{
		"library", "models", "search", "featured", "latest", "trending",
		"cloud", "local", "hf", "api", "admin", "account", "docs", "blog",
		"signin", "signup", "device", "download", "system", "susan", "official",
		"susanofficial", "pricing", "settings", "support", "connect",
		"susancloud", "chat",
	}

	if len(exact) != 28 {
		t.Fatalf("exact blacklist has %d entries, want 28", len(exact))
	}

	for _, ns := range exact {
		forms := []string{
			ns,
			toUpper(ns),
			titleCase(ns),
		}
		for _, form := range forms {
			if !IsReservedNamespace(form) {
				t.Errorf("IsReservedNamespace(%q) = false, want true", form)
			}
		}
	}
}

func TestIsReservedNamespacePrefixes(t *testing.T) {
	reserved := []string{
		"susanx",
		"susan-foo",
		"susan_cloud",
		"SuSaN-foo",
		"official-ai",
		"OFFICIAL_bar",
		"OfficialX",
	}

	for _, ns := range reserved {
		if !IsReservedNamespace(ns) {
			t.Errorf("IsReservedNamespace(%q) = false, want true", ns)
		}
	}
}

func TestIsReservedNamespaceNotReserved(t *testing.T) {
	notReserved := []string{
		"mysusan",
		"myofficial",
		"alice",
		"example",
		"namespace",
		"bob-255",
	}

	for _, ns := range notReserved {
		if IsReservedNamespace(ns) {
			t.Errorf("IsReservedNamespace(%q) = true, want false", ns)
		}
	}
}

func TestIsReservedNamespaceEmpty(t *testing.T) {
	if IsReservedNamespace("") {
		t.Error(`IsReservedNamespace("") = true, want false`)
	}
	if IsReservedNamespace("   ") {
		t.Error(`IsReservedNamespace("   ") = true, want false`)
	}
}

func toUpper(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 'a' - 'A'
		}
	}
	return string(b)
}

func titleCase(s string) string {
	b := []byte(s)
	if b[0] >= 'a' && b[0] <= 'z' {
		b[0] -= 'a' - 'A'
	}
	return string(b)
}
