package domain

import "testing"

func TestIsValidStatus(t *testing.T) {
	valid := []string{"pending", "approved", "active", "suspended", "blocked", "terminated"}
	for _, s := range valid {
		if !IsValidStatus(s) {
			t.Errorf("expected %q to be a valid status", s)
		}
	}
}

func TestIsValidStatus_RejectsUnknownValues(t *testing.T) {
	invalid := []string{
		"",
		"PENDING",       // case-sensitive: only lowercase canonical values are valid
		"deleted",       // not a real status
		"'; DROP TABLE", // garbage / injection-shaped input
		"<script>",
	}
	for _, s := range invalid {
		if IsValidStatus(s) {
			t.Errorf("expected %q to be an invalid status", s)
		}
	}
}
