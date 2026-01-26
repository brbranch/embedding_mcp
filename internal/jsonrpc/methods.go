package jsonrpc

import (
	"context"
	"encoding/json"
)

// handleAddNote は memory.add_note を処理
func (h *Handler) handleAddNote(ctx context.Context, params any) (any, error) {
	var p AddNoteParams
	if err := mapParams(params, &p); err != nil {
		return nil, err
	}

	resp, err := h.noteService.AddNote(ctx, p.ToRequest())
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":                 resp.ID,
		"namespace":          resp.Namespace,
		"canonicalProjectId": resp.CanonicalProjectID,
	}, nil
}

// handleSearch は memory.search を処理
func (h *Handler) handleSearch(ctx context.Context, params any) (any, error) {
	var p SearchParams
	if err := mapParams(params, &p); err != nil {
		return nil, err
	}

	resp, err := h.noteService.Search(ctx, p.ToRequest())
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, len(resp.Results))
	for i, r := range resp.Results {
		results[i] = map[string]any{
			"id":        r.ID,
			"projectId": r.ProjectID,
			"groupId":   r.GroupID,
			"title":     r.Title,
			"text":      r.Text,
			"tags":      r.Tags,
			"source":    r.Source,
			"createdAt": r.CreatedAt,
			"score":     r.Score,
			"metadata":  r.Metadata,
		}
	}

	return map[string]any{
		"namespace": resp.Namespace,
		"results":   results,
	}, nil
}

// handleGet は memory.get を処理
func (h *Handler) handleGet(ctx context.Context, params any) (any, error) {
	var p GetParams
	if err := mapParams(params, &p); err != nil {
		return nil, err
	}

	resp, err := h.noteService.Get(ctx, p.ID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":        resp.ID,
		"projectId": resp.ProjectID,
		"groupId":   resp.GroupID,
		"title":     resp.Title,
		"text":      resp.Text,
		"tags":      resp.Tags,
		"source":    resp.Source,
		"createdAt": resp.CreatedAt,
		"namespace": resp.Namespace,
		"metadata":  resp.Metadata,
	}, nil
}

// handleUpdate は memory.update を処理
func (h *Handler) handleUpdate(ctx context.Context, params any) (any, error) {
	var p UpdateParams
	if err := mapParams(params, &p); err != nil {
		return nil, err
	}

	req, err := p.ToRequest()
	if err != nil {
		return nil, err
	}

	if err := h.noteService.Update(ctx, req); err != nil {
		return nil, err
	}

	return map[string]any{
		"ok": true,
	}, nil
}

// handleListRecent は memory.list_recent を処理
func (h *Handler) handleListRecent(ctx context.Context, params any) (any, error) {
	var p ListRecentParams
	if err := mapParams(params, &p); err != nil {
		return nil, err
	}

	resp, err := h.noteService.ListRecent(ctx, p.ToRequest())
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, len(resp.Items))
	for i, item := range resp.Items {
		items[i] = map[string]any{
			"id":        item.ID,
			"projectId": item.ProjectID,
			"groupId":   item.GroupID,
			"title":     item.Title,
			"text":      item.Text,
			"tags":      item.Tags,
			"source":    item.Source,
			"createdAt": item.CreatedAt,
			"namespace": item.Namespace,
			"metadata":  item.Metadata,
		}
	}

	return map[string]any{
		"namespace": resp.Namespace,
		"items":     items,
	}, nil
}

// handleGetConfig は memory.get_config を処理
func (h *Handler) handleGetConfig(ctx context.Context) (any, error) {
	resp, err := h.configService.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"transportDefaults": map[string]any{
			"defaultTransport": resp.TransportDefaults.DefaultTransport,
		},
		"embedder": map[string]any{
			"provider": resp.Embedder.Provider,
			"model":    resp.Embedder.Model,
			"dim":      resp.Embedder.Dim,
			"baseUrl":  resp.Embedder.BaseURL,
		},
		"store": map[string]any{
			"type": resp.Store.Type,
			"path": resp.Store.Path,
			"url":  resp.Store.URL,
		},
		"paths": map[string]any{
			"configPath": resp.Paths.ConfigPath,
			"dataDir":    resp.Paths.DataDir,
		},
	}, nil
}

// handleSetConfig は memory.set_config を処理
func (h *Handler) handleSetConfig(ctx context.Context, params any) (any, error) {
	var p SetConfigParams
	if err := mapParams(params, &p); err != nil {
		return nil, err
	}

	resp, err := h.configService.SetConfig(ctx, p.ToRequest())
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"ok":                 resp.OK,
		"effectiveNamespace": resp.EffectiveNamespace,
	}, nil
}

// handleUpsertGlobal は memory.upsert_global を処理
func (h *Handler) handleUpsertGlobal(ctx context.Context, params any) (any, error) {
	var p UpsertGlobalParams
	if err := mapParams(params, &p); err != nil {
		return nil, err
	}

	resp, err := h.globalService.UpsertGlobal(ctx, p.ToRequest())
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"ok":        resp.OK,
		"id":        resp.ID,
		"namespace": resp.Namespace,
	}, nil
}

// handleGetGlobal は memory.get_global を処理
func (h *Handler) handleGetGlobal(ctx context.Context, params any) (any, error) {
	var p GetGlobalParams
	if err := mapParams(params, &p); err != nil {
		return nil, err
	}

	// key必須チェック（Handler側で実施）
	if p.Key == "" {
		return nil, errKeyRequired
	}

	resp, err := h.globalService.GetGlobal(ctx, p.ProjectID, p.Key)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"namespace": resp.Namespace,
		"found":     resp.Found,
	}
	if resp.Found {
		result["id"] = resp.ID
		result["value"] = resp.Value
		result["updatedAt"] = resp.UpdatedAt
	}

	return result, nil
}

// mapParams はanyをターゲット構造体にマッピング
func mapParams(params any, target any) error {
	if params == nil {
		return nil
	}

	// anyをJSONに変換してから構造体にアンマーシャル
	b, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
