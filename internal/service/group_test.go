package service

import (
	"context"
	"testing"

	"github.com/brbranch/embedding_mcp/internal/store"
)

func setupGroupTestService(t *testing.T) (GroupService, store.Store) {
	t.Helper()

	st := store.NewMemoryStore()
	namespace := "test:mock:128"
	if err := st.Initialize(context.Background(), namespace); err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}

	svc := NewGroupService(st, namespace)
	return svc, st
}

func TestGroupService_CreateGroup(t *testing.T) {
	ctx := context.Background()
	svc, _ := setupGroupTestService(t)

	tests := []struct {
		name    string
		req     *CreateGroupRequest
		wantErr bool
		errType error
	}{
		{
			name: "valid group",
			req: &CreateGroupRequest{
				ProjectID:   "/path/to/project",
				GroupKey:    "feature-1",
				Title:       "Feature 1",
				Description: "Description",
			},
			wantErr: false,
		},
		{
			name: "missing projectId",
			req: &CreateGroupRequest{
				ProjectID: "",
				GroupKey:  "feature-1",
				Title:     "Feature 1",
			},
			wantErr: true,
			errType: ErrProjectIDRequired,
		},
		{
			name: "missing groupKey",
			req: &CreateGroupRequest{
				ProjectID: "/path/to/project",
				GroupKey:  "",
				Title:     "Feature 1",
			},
			wantErr: true,
			errType: ErrGroupKeyRequired,
		},
		{
			name: "missing title",
			req: &CreateGroupRequest{
				ProjectID: "/path/to/project",
				GroupKey:  "feature-1",
				Title:     "",
			},
			wantErr: true,
			errType: ErrTitleRequired,
		},
		{
			name: "reserved groupKey global",
			req: &CreateGroupRequest{
				ProjectID: "/path/to/project",
				GroupKey:  "global",
				Title:     "Global Group",
			},
			wantErr: true,
			errType: ErrInvalidGroupKey,
		},
		{
			name: "invalid groupKey with space",
			req: &CreateGroupRequest{
				ProjectID: "/path/to/project",
				GroupKey:  "my group",
				Title:     "My Group",
			},
			wantErr: true,
			errType: ErrInvalidGroupKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.CreateGroup(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateGroup() expected error but got nil")
					return
				}
			} else {
				if err != nil {
					t.Errorf("CreateGroup() unexpected error: %v", err)
					return
				}
				if resp.ID == "" {
					t.Error("CreateGroup() returned empty ID")
				}
			}
		})
	}
}

func TestGroupService_CreateGroup_DuplicateKey(t *testing.T) {
	ctx := context.Background()
	svc, _ := setupGroupTestService(t)

	// 最初のグループを作成
	req := &CreateGroupRequest{
		ProjectID: "/path/to/project",
		GroupKey:  "feature-1",
		Title:     "Feature 1",
	}
	_, err := svc.CreateGroup(ctx, req)
	if err != nil {
		t.Fatalf("first CreateGroup() failed: %v", err)
	}

	// 同じgroupKeyで作成を試みる
	_, err = svc.CreateGroup(ctx, req)
	if err == nil {
		t.Error("CreateGroup() expected duplicate key error but got nil")
	}
}

func TestGroupService_GetGroup(t *testing.T) {
	ctx := context.Background()
	svc, _ := setupGroupTestService(t)

	// グループを作成
	createResp, err := svc.CreateGroup(ctx, &CreateGroupRequest{
		ProjectID:   "/path/to/project",
		GroupKey:    "feature-1",
		Title:       "Feature 1",
		Description: "Test description",
	})
	if err != nil {
		t.Fatalf("CreateGroup() failed: %v", err)
	}

	// 取得
	resp, err := svc.GetGroup(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("GetGroup() failed: %v", err)
	}

	if resp.ID != createResp.ID {
		t.Errorf("GetGroup() ID = %v, want %v", resp.ID, createResp.ID)
	}
	if resp.GroupKey != "feature-1" {
		t.Errorf("GetGroup() GroupKey = %v, want %v", resp.GroupKey, "feature-1")
	}
	if resp.Title != "Feature 1" {
		t.Errorf("GetGroup() Title = %v, want %v", resp.Title, "Feature 1")
	}
	if resp.Description != "Test description" {
		t.Errorf("GetGroup() Description = %v, want %v", resp.Description, "Test description")
	}
}

func TestGroupService_GetGroup_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, _ := setupGroupTestService(t)

	_, err := svc.GetGroup(ctx, "non-existent-id")
	if err != ErrGroupNotFound {
		t.Errorf("GetGroup() expected ErrGroupNotFound, got %v", err)
	}
}

func TestGroupService_UpdateGroup(t *testing.T) {
	ctx := context.Background()
	svc, _ := setupGroupTestService(t)

	// グループを作成
	createResp, err := svc.CreateGroup(ctx, &CreateGroupRequest{
		ProjectID:   "/path/to/project",
		GroupKey:    "feature-1",
		Title:       "Feature 1",
		Description: "Original",
	})
	if err != nil {
		t.Fatalf("CreateGroup() failed: %v", err)
	}

	// 更新
	newTitle := "Updated Feature 1"
	newDesc := "Updated description"
	err = svc.UpdateGroup(ctx, &UpdateGroupRequest{
		ID: createResp.ID,
		Patch: GroupPatch{
			Title:       &newTitle,
			Description: &newDesc,
		},
	})
	if err != nil {
		t.Fatalf("UpdateGroup() failed: %v", err)
	}

	// 確認
	resp, err := svc.GetGroup(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("GetGroup() failed: %v", err)
	}

	if resp.Title != newTitle {
		t.Errorf("UpdateGroup() Title = %v, want %v", resp.Title, newTitle)
	}
	if resp.Description != newDesc {
		t.Errorf("UpdateGroup() Description = %v, want %v", resp.Description, newDesc)
	}
}

func TestGroupService_DeleteGroup(t *testing.T) {
	ctx := context.Background()
	svc, _ := setupGroupTestService(t)

	// グループを作成
	createResp, err := svc.CreateGroup(ctx, &CreateGroupRequest{
		ProjectID: "/path/to/project",
		GroupKey:  "feature-1",
		Title:     "Feature 1",
	})
	if err != nil {
		t.Fatalf("CreateGroup() failed: %v", err)
	}

	// 削除
	err = svc.DeleteGroup(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("DeleteGroup() failed: %v", err)
	}

	// 存在しないことを確認
	_, err = svc.GetGroup(ctx, createResp.ID)
	if err != ErrGroupNotFound {
		t.Errorf("GetGroup() after delete expected ErrGroupNotFound, got %v", err)
	}
}

func TestGroupService_ListGroups(t *testing.T) {
	ctx := context.Background()
	svc, _ := setupGroupTestService(t)

	projectID := "/path/to/project"

	// 複数グループを作成
	_, err := svc.CreateGroup(ctx, &CreateGroupRequest{
		ProjectID: projectID,
		GroupKey:  "feature-1",
		Title:     "Feature 1",
	})
	if err != nil {
		t.Fatalf("CreateGroup() failed: %v", err)
	}

	_, err = svc.CreateGroup(ctx, &CreateGroupRequest{
		ProjectID: projectID,
		GroupKey:  "feature-2",
		Title:     "Feature 2",
	})
	if err != nil {
		t.Fatalf("CreateGroup() failed: %v", err)
	}

	// 別プロジェクトのグループ
	_, err = svc.CreateGroup(ctx, &CreateGroupRequest{
		ProjectID: "/path/to/other",
		GroupKey:  "feature-3",
		Title:     "Feature 3",
	})
	if err != nil {
		t.Fatalf("CreateGroup() failed: %v", err)
	}

	// リスト取得
	resp, err := svc.ListGroups(ctx, projectID)
	if err != nil {
		t.Fatalf("ListGroups() failed: %v", err)
	}

	if len(resp.Groups) != 2 {
		t.Errorf("ListGroups() returned %d groups, want 2", len(resp.Groups))
	}
}
