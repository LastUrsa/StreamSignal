package runtimeenv

import "testing"

func TestProductionEnvironmentUsesExistingNames(t *testing.T) {
	t.Setenv(EnvironmentVariable, "")

	if Name() != "production" {
		t.Fatalf("expected production environment, got %q", Name())
	}
	if DataDirName() != "StreamSignal" {
		t.Fatalf("expected existing data dir name, got %q", DataDirName())
	}
	if SecretNamespace() != "streamsignal" {
		t.Fatalf("expected existing secret namespace, got %q", SecretNamespace())
	}
}

func TestNonProductionEnvironmentIsNamespaced(t *testing.T) {
	t.Setenv(EnvironmentVariable, " Test Profile ")

	if Name() != "test-profile" {
		t.Fatalf("expected sanitized environment name, got %q", Name())
	}
	if DataDirName() != "StreamSignal-test-profile" {
		t.Fatalf("expected namespaced data dir, got %q", DataDirName())
	}
	if SecretNamespace() != "streamsignal-test-profile" {
		t.Fatalf("expected namespaced secret store, got %q", SecretNamespace())
	}
}
