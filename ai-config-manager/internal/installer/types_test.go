package installer

import "testing"

func TestItemGroup(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		exists bool
		want   Group
	}{
		{"unchanged and existing is up to date", StatusUnchanged, true, GroupUpToDate},
		{"unchanged and missing is still up to date", StatusUnchanged, false, GroupUpToDate},
		{"updated and existing is to update", StatusUpdated, true, GroupToUpdate},
		{"updated and missing is to install", StatusUpdated, false, GroupToInstall},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := Item{Status: tt.status, Exists: tt.exists}
			if got := item.Group(); got != tt.want {
				t.Errorf("Group() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupString(t *testing.T) {
	tests := []struct {
		name string
		g    Group
		want string
	}{
		{"to update", GroupToUpdate, "to update"},
		{"to install", GroupToInstall, "to install"},
		{"up to date", GroupUpToDate, "up to date"},
		{"out of range", Group(99), "Group(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.g.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
