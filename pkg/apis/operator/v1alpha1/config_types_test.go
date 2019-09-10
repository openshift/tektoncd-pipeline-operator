package v1alpha1

import (
	"testing"
)

func TestConfig_is_uptodate_and_ready(t *testing.T) {
	config := &Config{
		Spec: ConfigSpec{
			TargetNamespace: "test",
		},
		Status: ConfigStatus{
			Conditions: []ConfigCondition{
				{
					Version: "v0.6.0",
					Code:    ReadyStatus,
				},
			},
		},
	}

	actualStatus := config.IsUpToDateWith("v0.6.0")
	expectedStatus := true
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %v actual status %v", expectedStatus, actualStatus)
	}
}

func TestConfig_is_uptodate_error_state(t *testing.T) {
	config := &Config{
		Spec: ConfigSpec{
			TargetNamespace: "test",
		},
		Status: ConfigStatus{
			Conditions: []ConfigCondition{
				{
					Version: "v0.6.0",
					Code:    ErrorStatus,
				},
			},
		},
	}

	actualStatus := config.IsUpToDateWith("v0.6.0")
	expectedStatus := true
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %v actual status %v", expectedStatus, actualStatus)
	}
}

func TestConfig_is_not_uptodate(t *testing.T) {
	config := &Config{
		Spec: ConfigSpec{
			TargetNamespace: "test",
		},
		Status: ConfigStatus{
			Conditions: []ConfigCondition{
				{
					Version: "v0.6.0",
					Code:    ReadyStatus,
				},
			},
		},
	}

	actualStatus := config.IsUpToDateWith("v0.7.0")
	expectedStatus := false
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %v actual status %v", expectedStatus, actualStatus)
	}
}

func TestConfig_not_reconciled(t *testing.T) {
	config := &Config{
		Spec: ConfigSpec{
			TargetNamespace: "test",
		},
	}

	actualStatus := config.IsUpToDateWith("v0.7.0")
	expectedStatus := false
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %v actual status %v", expectedStatus, actualStatus)
	}
}

func TestConfig_is_installing(t *testing.T) {
	config := &Config{
		Spec: ConfigSpec{
			TargetNamespace: "test",
		},

		Status: ConfigStatus{
			Conditions: []ConfigCondition{
				{
					Version: "v0.6.0",
					Code:    InstallingStatus,
				},
			},
		},
	}

	actualStatus := config.IsInstalling("v0.6.0")
	expectedStatus := true
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %v actual status %v", expectedStatus, actualStatus)
	}
}

func TestConfig_is_installed(t *testing.T) {
	config := &Config{
		Spec: ConfigSpec{
			TargetNamespace: "test",
		},

		Status: ConfigStatus{
			Conditions: []ConfigCondition{
				{
					Version: "v0.6.0",
					Code:    InstalledStatus,
				},
				{
					Version: "v0.6.0",
					Code:    InstallingStatus,
				},
			},
		},
	}

	actualStatus := config.IsInstalled("v0.6.0")
	expectedStatus := true
	if actualStatus != expectedStatus {
		t.Fatalf("expected status %v actual status %v", expectedStatus, actualStatus)
	}
}
