package model

import (
	"testing"
)

// TestNote_Validate_Valid は有効なNoteがバリデーションを通過することをテスト
func TestNote_Validate_Valid(t *testing.T) {
	title := "Test Note"
	source := "test-source"
	createdAt := "2024-01-15T10:30:00Z"

	note := &Note{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		ProjectID: "/path/to/project",
		GroupID:   "test-group",
		Title:     &title,
		Text:      "This is a test note",
		Tags:      []string{"tag1", "tag2"},
		Source:    &source,
		CreatedAt: &createdAt,
		Metadata:  map[string]any{"key": "value"},
	}

	if err := note.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// TestNote_Validate_EmptyID は空のIDでエラーになることをテスト
func TestNote_Validate_EmptyID(t *testing.T) {
	note := &Note{
		ID:        "",
		ProjectID: "/path/to/project",
		GroupID:   "test-group",
		Text:      "This is a test note",
		Tags:      []string{},
	}

	if err := note.Validate(); err == nil {
		t.Error("expected error for empty ID, got nil")
	}
}

// TestNote_Validate_EmptyProjectID は空のProjectIDでエラーになることをテスト
func TestNote_Validate_EmptyProjectID(t *testing.T) {
	note := &Note{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		ProjectID: "",
		GroupID:   "test-group",
		Text:      "This is a test note",
		Tags:      []string{},
	}

	if err := note.Validate(); err == nil {
		t.Error("expected error for empty ProjectID, got nil")
	}
}

// TestNote_Validate_EmptyText は空のTextでエラーになることをテスト
func TestNote_Validate_EmptyText(t *testing.T) {
	note := &Note{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		ProjectID: "/path/to/project",
		GroupID:   "test-group",
		Text:      "",
		Tags:      []string{},
	}

	if err := note.Validate(); err == nil {
		t.Error("expected error for empty Text, got nil")
	}
}

// TestNote_Validate_InvalidGroupID は不正なGroupID（特殊文字含む）でエラーになることをテスト
func TestNote_Validate_InvalidGroupID(t *testing.T) {
	tests := []struct {
		name    string
		groupID string
	}{
		{"space", "test group"},
		{"special char @", "test@group"},
		{"special char #", "test#group"},
		{"japanese", "テストグループ"},
		{"dot", "test.group"},
		{"slash", "test/group"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := &Note{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				ProjectID: "/path/to/project",
				GroupID:   tt.groupID,
				Text:      "This is a test note",
				Tags:      []string{},
			}

			if err := note.Validate(); err == nil {
				t.Errorf("expected error for invalid GroupID %q, got nil", tt.groupID)
			}
		})
	}
}

// TestValidateGroupID_Valid は有効なGroupID（英数字、-、_）が通過することをテスト
func TestValidateGroupID_Valid(t *testing.T) {
	tests := []struct {
		name    string
		groupID string
	}{
		{"lowercase", "testgroup"},
		{"uppercase", "TESTGROUP"},
		{"mixed case", "TestGroup"},
		{"with hyphen", "test-group"},
		{"with underscore", "test_group"},
		{"alphanumeric", "test123"},
		{"hyphen and underscore", "test-group_123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateGroupID(tt.groupID); err != nil {
				t.Errorf("expected no error for GroupID %q, got %v", tt.groupID, err)
			}
		})
	}
}

// TestValidateGroupID_GlobalReserved は"global"が予約値として受理されることをテスト
func TestValidateGroupID_GlobalReserved(t *testing.T) {
	// "global" は正規表現を通過するので受理される
	if err := ValidateGroupID("global"); err != nil {
		t.Errorf("expected no error for GroupID 'global', got %v", err)
	}
}

// TestValidateGroupID_Invalid はスペース、日本語、その他特殊文字でエラーになることをテスト
func TestValidateGroupID_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		groupID string
	}{
		{"empty", ""},
		{"space", "test group"},
		{"special char @", "test@group"},
		{"special char #", "test#group"},
		{"japanese", "テストグループ"},
		{"dot", "test.group"},
		{"slash", "test/group"},
		{"backslash", "test\\group"},
		{"only space", " "},
		{"leading space", " testgroup"},
		{"trailing space", "testgroup "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateGroupID(tt.groupID); err == nil {
				t.Errorf("expected error for invalid GroupID %q, got nil", tt.groupID)
			}
		})
	}
}
