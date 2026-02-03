//go:build e2e

package e2e

import (
	"testing"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// TestE2E_Group_CRUD はグループのフルフローをテスト
// create → get → list → update → delete → 削除確認
func TestE2E_Group_CRUD(t *testing.T) {
	h := setupTestHandler(t)

	projectID := "/test/group-crud"
	groupKey := "test-group"
	title := "Test Group"
	description := "Test Description"

	// 1. create
	t.Run("create", func(t *testing.T) {
		result := callGroupCreate(t, h, projectID, groupKey, title, description)

		if result.ID == "" {
			t.Error("expected non-empty ID")
		}
		if result.Namespace != "test:mock:128" {
			t.Errorf("expected namespace 'test:mock:128', got %s", result.Namespace)
		}
	})

	// 2. list → 作成したグループが存在することを確認
	var groupID string
	t.Run("list after create", func(t *testing.T) {
		result := callGroupList(t, h, projectID)

		if len(result.Groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(result.Groups))
		}

		group := result.Groups[0]
		groupID = group.ID

		if group.GroupKey != groupKey {
			t.Errorf("expected groupKey %q, got %q", groupKey, group.GroupKey)
		}
		if group.Title != title {
			t.Errorf("expected title %q, got %q", title, group.Title)
		}
		if group.Description != description {
			t.Errorf("expected description %q, got %q", description, group.Description)
		}
	})

	// 3. get
	t.Run("get", func(t *testing.T) {
		result := callGroupGet(t, h, groupID)

		if result.ID != groupID {
			t.Errorf("expected ID %q, got %q", groupID, result.ID)
		}
		if result.GroupKey != groupKey {
			t.Errorf("expected groupKey %q, got %q", groupKey, result.GroupKey)
		}
		if result.Title != title {
			t.Errorf("expected title %q, got %q", title, result.Title)
		}
		if result.Description != description {
			t.Errorf("expected description %q, got %q", description, result.Description)
		}
	})

	// 4. update
	t.Run("update", func(t *testing.T) {
		newTitle := "Updated Title"
		newDescription := "Updated Description"

		result := callGroupUpdate(t, h, groupID, &newTitle, &newDescription)

		if !result.OK {
			t.Error("expected OK to be true")
		}

		// 更新後の確認
		updatedGroup := callGroupGet(t, h, groupID)
		if updatedGroup.Title != newTitle {
			t.Errorf("expected updated title %q, got %q", newTitle, updatedGroup.Title)
		}
		if updatedGroup.Description != newDescription {
			t.Errorf("expected updated description %q, got %q", newDescription, updatedGroup.Description)
		}
	})

	// 5. delete
	t.Run("delete", func(t *testing.T) {
		result := callGroupDelete(t, h, groupID)

		if !result.OK {
			t.Error("expected OK to be true")
		}
	})

	// 6. 削除確認（get → エラー）
	t.Run("get after delete", func(t *testing.T) {
		resp := callGroupGetRaw(t, h, groupID)

		if resp.Error == nil {
			t.Error("expected error after delete")
		}

		if resp.Error.Code != model.ErrCodeNotFound {
			t.Errorf("expected error code %d, got %d", model.ErrCodeNotFound, resp.Error.Code)
		}
	})

	// 7. 削除確認（list → 空）
	t.Run("list after delete", func(t *testing.T) {
		result := callGroupList(t, h, projectID)

		if len(result.Groups) != 0 {
			t.Errorf("expected 0 groups after delete, got %d", len(result.Groups))
		}
	})
}

// TestE2E_Group_CreateGlobalKeyRejected はgroupKey "global" の作成を拒否することをテスト
func TestE2E_Group_CreateGlobalKeyRejected(t *testing.T) {
	h := setupTestHandler(t)

	projectID := "/test/group-global"
	groupKey := "global"
	title := "Global Group"
	description := ""

	resp := callGroupCreateRaw(t, h, projectID, groupKey, title, description)

	if resp.Error == nil {
		t.Fatal("expected error for groupKey 'global'")
	}

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected error code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

// TestE2E_Group_CreateDuplicateKeyRejected は同じprojectId+groupKeyで2回作成→2回目でエラー
func TestE2E_Group_CreateDuplicateKeyRejected(t *testing.T) {
	h := setupTestHandler(t)

	projectID := "/test/group-duplicate"
	groupKey := "duplicate-key"
	title := "Duplicate Test"
	description := ""

	// 1回目は成功
	result := callGroupCreate(t, h, projectID, groupKey, title, description)
	if result.ID == "" {
		t.Fatal("expected successful create on first attempt")
	}

	// 2回目はエラー
	resp := callGroupCreateRaw(t, h, projectID, groupKey, title, description)
	if resp.Error == nil {
		t.Fatal("expected error on duplicate create")
	}

	if resp.Error.Code != model.ErrCodeConflict {
		t.Errorf("expected error code %d, got %d", model.ErrCodeConflict, resp.Error.Code)
	}
}

// TestE2E_Group_CreateInvalidKeyFormat は無効なgroupKey（スペース含む）→エラー
func TestE2E_Group_CreateInvalidKeyFormat(t *testing.T) {
	h := setupTestHandler(t)

	projectID := "/test/group-invalid"
	groupKey := "invalid key" // スペース含む
	title := "Invalid Key Test"
	description := ""

	resp := callGroupCreateRaw(t, h, projectID, groupKey, title, description)

	if resp.Error == nil {
		t.Fatal("expected error for invalid groupKey format")
	}

	if resp.Error.Code != model.ErrCodeInvalidParams {
		t.Errorf("expected error code %d, got %d", model.ErrCodeInvalidParams, resp.Error.Code)
	}
}

// TestE2E_Group_GetNotFound は存在しないID→エラー
func TestE2E_Group_GetNotFound(t *testing.T) {
	h := setupTestHandler(t)

	nonexistentID := "00000000-0000-0000-0000-000000000000"

	resp := callGroupGetRaw(t, h, nonexistentID)

	if resp.Error == nil {
		t.Fatal("expected error for nonexistent group")
	}

	if resp.Error.Code != model.ErrCodeNotFound {
		t.Errorf("expected error code %d, got %d", model.ErrCodeNotFound, resp.Error.Code)
	}
}

// TestE2E_Group_ListByProject は異なるprojectIdにグループ作成、特定projectIdでlist
func TestE2E_Group_ListByProject(t *testing.T) {
	h := setupTestHandler(t)

	projectA := "/test/group-list-a"
	projectB := "/test/group-list-b"

	// projectA に2つのグループ作成
	callGroupCreate(t, h, projectA, "group-a1", "Group A1", "")
	callGroupCreate(t, h, projectA, "group-a2", "Group A2", "")

	// projectB に1つのグループ作成
	callGroupCreate(t, h, projectB, "group-b1", "Group B1", "")

	// projectA のグループを取得
	resultA := callGroupList(t, h, projectA)
	if len(resultA.Groups) != 2 {
		t.Errorf("expected 2 groups in projectA, got %d", len(resultA.Groups))
	}

	// projectB のグループを取得
	resultB := callGroupList(t, h, projectB)
	if len(resultB.Groups) != 1 {
		t.Errorf("expected 1 group in projectB, got %d", len(resultB.Groups))
	}

	// projectA のグループが正しいことを確認
	foundA1 := false
	foundA2 := false
	for _, group := range resultA.Groups {
		if group.GroupKey == "group-a1" {
			foundA1 = true
		}
		if group.GroupKey == "group-a2" {
			foundA2 = true
		}
	}
	if !foundA1 || !foundA2 {
		t.Error("expected to find both group-a1 and group-a2 in projectA")
	}

	// projectB のグループが正しいことを確認
	if resultB.Groups[0].GroupKey != "group-b1" {
		t.Errorf("expected groupKey 'group-b1' in projectB, got %q", resultB.Groups[0].GroupKey)
	}
}
