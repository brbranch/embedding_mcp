package model

import (
	"testing"
	"time"
)

func TestGroup_Validate(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name    string
		group   *Group
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid group",
			group: &Group{
				ID:          "test-id",
				ProjectID:   "/path/to/project",
				GroupKey:    "my-group",
				Title:       "Test Group",
				Description: "A test group",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantErr: false,
		},
		{
			name: "valid group with minimal fields",
			group: &Group{
				ID:        "test-id",
				ProjectID: "/path/to/project",
				GroupKey:  "group1",
				Title:     "Title",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: false,
		},
		{
			name: "empty ID",
			group: &Group{
				ID:        "",
				ProjectID: "/path/to/project",
				GroupKey:  "my-group",
				Title:     "Title",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "ID must not be empty",
		},
		{
			name: "empty ProjectID",
			group: &Group{
				ID:        "test-id",
				ProjectID: "",
				GroupKey:  "my-group",
				Title:     "Title",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "ProjectID must not be empty",
		},
		{
			name: "empty GroupKey",
			group: &Group{
				ID:        "test-id",
				ProjectID: "/path/to/project",
				GroupKey:  "",
				Title:     "Title",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "GroupKey must not be empty",
		},
		{
			name: "empty Title",
			group: &Group{
				ID:        "test-id",
				ProjectID: "/path/to/project",
				GroupKey:  "my-group",
				Title:     "",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "Title must not be empty",
		},
		{
			name: "reserved groupKey 'global'",
			group: &Group{
				ID:        "test-id",
				ProjectID: "/path/to/project",
				GroupKey:  "global",
				Title:     "Title",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "GroupKey 'global' is reserved",
		},
		{
			name: "invalid groupKey with space",
			group: &Group{
				ID:        "test-id",
				ProjectID: "/path/to/project",
				GroupKey:  "my group",
				Title:     "Title",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "GroupKey must match pattern",
		},
		{
			name: "invalid groupKey with special chars",
			group: &Group{
				ID:        "test-id",
				ProjectID: "/path/to/project",
				GroupKey:  "my@group",
				Title:     "Title",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: true,
			errMsg:  "GroupKey must match pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.group.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateGroupKeyForCreate(t *testing.T) {
	tests := []struct {
		name     string
		groupKey string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid key alphanumeric",
			groupKey: "myGroup123",
			wantErr:  false,
		},
		{
			name:     "valid key with hyphen",
			groupKey: "my-group",
			wantErr:  false,
		},
		{
			name:     "valid key with underscore",
			groupKey: "my_group",
			wantErr:  false,
		},
		{
			name:     "valid key mixed",
			groupKey: "My_Group-123",
			wantErr:  false,
		},
		{
			name:     "empty key",
			groupKey: "",
			wantErr:  true,
			errMsg:   "GroupKey must not be empty",
		},
		{
			name:     "reserved 'global'",
			groupKey: "global",
			wantErr:  true,
			errMsg:   "GroupKey 'global' is reserved",
		},
		{
			name:     "invalid with space",
			groupKey: "my group",
			wantErr:  true,
			errMsg:   "GroupKey must match pattern",
		},
		{
			name:     "invalid with dot",
			groupKey: "my.group",
			wantErr:  true,
			errMsg:   "GroupKey must match pattern",
		},
		{
			name:     "invalid with slash",
			groupKey: "my/group",
			wantErr:  true,
			errMsg:   "GroupKey must match pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGroupKeyForCreate(tt.groupKey)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateGroupKeyForCreate() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateGroupKeyForCreate() error = %v, want contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateGroupKeyForCreate() unexpected error: %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
