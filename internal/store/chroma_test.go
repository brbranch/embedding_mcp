package store

import (
	"testing"
	"time"

	"github.com/brbranch/embedding_mcp/internal/model"
)

// Note: これらのテストはChromaサーバー不要でテストできるようモック実装を使用する
// モック実装は後で追加予定

const (
	testProjectID  = "/test/project"
	testProjectID2 = "/test/project2"
	testGroupID    = "test-group"
	testGroupID2   = "test-group-2"
	testNamespace  = "openai:text-embedding-ada-002:1536"
)

// TestChromaStore_AddNote はノート追加をテスト
func TestChromaStore_AddNote(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	note := newTestNote("note-1", testProjectID, testGroupID, "This is a test note")
	embedding := dummyEmbedding(1536)

	err := store.AddNote(ctx, note, embedding)
	assertNoError(t, err)

	// 取得して検証
	retrieved, err := store.Get(ctx, note.ID)
	assertNoError(t, err)
	assertNoteEquals(t, note, retrieved)
}

// TestChromaStore_AddNote_AllFields は全フィールド含むノート追加をテスト
func TestChromaStore_AddNote_AllFields(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	note := newTestNoteAllFields("note-all", testProjectID, testGroupID)
	embedding := dummyEmbedding(1536)

	err := store.AddNote(ctx, note, embedding)
	assertNoError(t, err)

	// 取得して全フィールドを検証
	retrieved, err := store.Get(ctx, note.ID)
	assertNoError(t, err)
	assertNoteEquals(t, note, retrieved)

	// Metadata検証
	if retrieved.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
	if retrieved.Metadata["key1"] != "value1" {
		t.Errorf("Metadata[key1] mismatch: expected=%q, actual=%v", "value1", retrieved.Metadata["key1"])
	}
}

// TestChromaStore_Get はID指定でノート取得をテスト
func TestChromaStore_Get(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 準備
	note := newTestNote("note-get", testProjectID, testGroupID, "Get test")
	embedding := dummyEmbedding(1536)
	err := store.AddNote(ctx, note, embedding)
	assertNoError(t, err)

	// 実行
	retrieved, err := store.Get(ctx, note.ID)
	assertNoError(t, err)
	assertNoteEquals(t, note, retrieved)
}

// TestChromaStore_Get_NotFound は存在しないIDでErrNotFoundを返すことをテスト
func TestChromaStore_Get_NotFound(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	_, err := store.Get(ctx, "non-existent-id")
	assertErrorIs(t, err, ErrNotFound)
}

// TestChromaStore_Update はノート更新をテスト
func TestChromaStore_Update(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 準備
	note := newTestNote("note-update", testProjectID, testGroupID, "Original text")
	embedding := dummyEmbedding(1536)
	err := store.AddNote(ctx, note, embedding)
	assertNoError(t, err)

	// 更新
	note.Text = "Updated text"
	note.Tags = []string{"updated"}
	newEmbedding := dummyEmbedding(1536)
	newEmbedding[0] = 0.999 // 異なるembedding
	err = store.Update(ctx, note, newEmbedding)
	assertNoError(t, err)

	// 検証
	retrieved, err := store.Get(ctx, note.ID)
	assertNoError(t, err)
	assertNoteEquals(t, note, retrieved)
}

// TestChromaStore_Update_MetadataOnly はmetadataのみ更新をテスト
func TestChromaStore_Update_MetadataOnly(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 準備
	note := newTestNote("note-metadata", testProjectID, testGroupID, "Test text")
	note.Metadata = map[string]any{"version": 1}
	embedding := dummyEmbedding(1536)
	err := store.AddNote(ctx, note, embedding)
	assertNoError(t, err)

	// metadataのみ更新
	note.Metadata = map[string]any{"version": 2, "updated": true}
	err = store.Update(ctx, note, embedding) // 同じembedding
	assertNoError(t, err)

	// 検証
	retrieved, err := store.Get(ctx, note.ID)
	assertNoError(t, err)
	// JSONを通すとintがfloat64になるため、型に寛容に比較
	version, ok := retrieved.Metadata["version"]
	if !ok {
		t.Error("Metadata version should exist")
	}
	// versionが2または2.0であればOK
	switch v := version.(type) {
	case int:
		if v != 2 {
			t.Errorf("Metadata version mismatch: expected=2, actual=%v", v)
		}
	case float64:
		if v != 2.0 {
			t.Errorf("Metadata version mismatch: expected=2.0, actual=%v", v)
		}
	default:
		t.Errorf("Metadata version has unexpected type: %T", v)
	}
}

// TestChromaStore_Delete はノート削除をテスト
func TestChromaStore_Delete(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 準備
	note := newTestNote("note-delete", testProjectID, testGroupID, "Delete test")
	embedding := dummyEmbedding(1536)
	err := store.AddNote(ctx, note, embedding)
	assertNoError(t, err)

	// 削除
	err = store.Delete(ctx, note.ID)
	assertNoError(t, err)

	// 削除後に取得するとErrNotFoundになる
	_, err = store.Get(ctx, note.ID)
	assertErrorIs(t, err, ErrNotFound)
}

// TestChromaStore_Search_Basic は基本的なベクトル検索をテスト
func TestChromaStore_Search_Basic(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// テストデータ準備
	notes := []struct {
		note      *model.Note
		embedding []float32
	}{
		{newTestNote("s1", testProjectID, testGroupID, "Apple"), []float32{1.0, 0.0, 0.0}},
		{newTestNote("s2", testProjectID, testGroupID, "Banana"), []float32{0.8, 0.2, 0.0}},
		{newTestNote("s3", testProjectID, testGroupID, "Orange"), []float32{0.5, 0.5, 0.0}},
	}

	for _, n := range notes {
		err := store.AddNote(ctx, n.note, n.embedding)
		assertNoError(t, err)
	}

	// 検索
	queryEmbedding := []float32{0.9, 0.1, 0.0} // Appleに近い
	opts := SearchOptions{
		ProjectID: testProjectID,
		TopK:      2,
	}

	results, err := store.Search(ctx, queryEmbedding, opts)
	assertNoError(t, err)

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// スコア降順を確認
	assertSearchResultsOrdered(t, results)

	// 最も類似するのはAppleのはず
	if results[0].Note.ID != "s1" {
		t.Errorf("First result should be s1 (Apple), got %s", results[0].Note.ID)
	}
}

// TestChromaStore_Search_ProjectIDFilter はprojectIDでフィルタをテスト
func TestChromaStore_Search_ProjectIDFilter(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 異なるprojectIDのノートを追加
	embedding := dummyEmbedding(1536)
	note1 := newTestNote("p1", testProjectID, testGroupID, "Project 1 note")
	note2 := newTestNote("p2", testProjectID2, testGroupID, "Project 2 note")

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))

	// testProjectIDでフィルタ
	opts := SearchOptions{
		ProjectID: testProjectID,
		TopK:      10,
	}
	results, err := store.Search(ctx, embedding, opts)
	assertNoError(t, err)

	// testProjectIDのノートのみ返されること
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Note.ID != "p1" {
		t.Errorf("Expected note p1, got %s", results[0].Note.ID)
	}
}

// TestChromaStore_Search_GroupIDFilter はgroupIDでフィルタをテスト
func TestChromaStore_Search_GroupIDFilter(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 異なるgroupIDのノートを追加
	embedding := dummyEmbedding(1536)
	note1 := newTestNote("g1", testProjectID, testGroupID, "Group 1 note")
	note2 := newTestNote("g2", testProjectID, testGroupID2, "Group 2 note")

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))

	// testGroupIDでフィルタ
	opts := SearchOptions{
		ProjectID: testProjectID,
		GroupID:   stringPtr(testGroupID),
		TopK:      10,
	}
	results, err := store.Search(ctx, embedding, opts)
	assertNoError(t, err)

	// testGroupIDのノートのみ返されること
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	if results[0].Note.ID != "g1" {
		t.Errorf("Expected note g1, got %s", results[0].Note.ID)
	}
}

// TestChromaStore_Search_GroupIDNil はgroupID=nilで全group検索をテスト
func TestChromaStore_Search_GroupIDNil(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 異なるgroupIDのノートを追加
	embedding := dummyEmbedding(1536)
	note1 := newTestNote("gn1", testProjectID, testGroupID, "Group 1 note")
	note2 := newTestNote("gn2", testProjectID, testGroupID2, "Group 2 note")

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))

	// groupID=nilで全group検索
	opts := SearchOptions{
		ProjectID: testProjectID,
		GroupID:   nil, // 全group対象
		TopK:      10,
	}
	results, err := store.Search(ctx, embedding, opts)
	assertNoError(t, err)

	// 両方のgroupのノートが返されること
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
}

// TestChromaStore_Search_TopK はTopK指定で件数制限をテスト
func TestChromaStore_Search_TopK(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 5件のノートを追加
	embedding := dummyEmbedding(1536)
	for i := 1; i <= 5; i++ {
		id := "topk-" + string(rune('0'+i))
		text := "Note " + string(rune('0'+i))
		note := newTestNote(id, testProjectID, testGroupID, text)
		assertNoError(t, store.AddNote(ctx, note, embedding))
	}

	// TopK=3で検索
	opts := SearchOptions{
		ProjectID: testProjectID,
		TopK:      3,
	}
	results, err := store.Search(ctx, embedding, opts)
	assertNoError(t, err)

	if len(results) != 3 {
		t.Fatalf("Expected 3 results (TopK=3), got %d", len(results))
	}
}

// TestChromaStore_Search_ScoreOrder はスコア降順でソートをテスト
func TestChromaStore_Search_ScoreOrder(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 異なるembeddingのノートを追加
	notes := []struct {
		id        string
		embedding []float32
	}{
		{"score1", []float32{1.0, 0.0, 0.0}},
		{"score2", []float32{0.5, 0.5, 0.0}},
		{"score3", []float32{0.0, 1.0, 0.0}},
	}

	for _, n := range notes {
		note := newTestNote(n.id, testProjectID, testGroupID, "Score test")
		assertNoError(t, store.AddNote(ctx, note, n.embedding))
	}

	// クエリ
	queryEmbedding := []float32{0.9, 0.1, 0.0}
	opts := SearchOptions{
		ProjectID: testProjectID,
		TopK:      10,
	}
	results, err := store.Search(ctx, queryEmbedding, opts)
	assertNoError(t, err)

	// スコア降順を確認
	assertSearchResultsOrdered(t, results)
}

// TestChromaStore_Search_TagsFilter_AND はtags AND検索をテスト
func TestChromaStore_Search_TagsFilter_AND(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 異なるタグのノートを追加
	embedding := dummyEmbedding(1536)
	note1 := newTestNoteWithTags("tag1", testProjectID, testGroupID, "Note 1", []string{"go", "test"})
	note2 := newTestNoteWithTags("tag2", testProjectID, testGroupID, "Note 2", []string{"go", "prod"})
	note3 := newTestNoteWithTags("tag3", testProjectID, testGroupID, "Note 3", []string{"go", "test", "prod"})

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))
	assertNoError(t, store.AddNote(ctx, note3, embedding))

	// tags: ["go", "test"] でAND検索
	opts := SearchOptions{
		ProjectID: testProjectID,
		Tags:      []string{"go", "test"},
		TopK:      10,
	}
	results, err := store.Search(ctx, embedding, opts)
	assertNoError(t, err)

	// note1とnote3のみが返されること
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if !containsAllTags(r.Note.Tags, []string{"go", "test"}) {
			t.Errorf("Result should contain both 'go' and 'test' tags, got %v", r.Note.Tags)
		}
	}
}

// TestChromaStore_Search_TagsFilter_Empty は空配列はフィルタなしをテスト
func TestChromaStore_Search_TagsFilter_Empty(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// タグ付きノートを追加
	embedding := dummyEmbedding(1536)
	note1 := newTestNoteWithTags("empty1", testProjectID, testGroupID, "Note 1", []string{"tag1"})
	note2 := newTestNoteWithTags("empty2", testProjectID, testGroupID, "Note 2", []string{"tag2"})

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))

	// tags: [] で検索（フィルタなし）
	opts := SearchOptions{
		ProjectID: testProjectID,
		Tags:      []string{}, // 空配列
		TopK:      10,
	}
	results, err := store.Search(ctx, embedding, opts)
	assertNoError(t, err)

	// 全てのノートが返されること
	if len(results) != 2 {
		t.Fatalf("Expected 2 results (no filter), got %d", len(results))
	}
}

// TestChromaStore_Search_TagsNil はtags=nilでフィルタなしをテスト
func TestChromaStore_Search_TagsNil(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// タグ付きノートを追加
	embedding := dummyEmbedding(1536)
	note1 := newTestNoteWithTags("nil1", testProjectID, testGroupID, "Note 1", []string{"tag1"})
	note2 := newTestNoteWithTags("nil2", testProjectID, testGroupID, "Note 2", []string{"tag2"})

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))

	// tags: nil で検索（フィルタなし）
	opts := SearchOptions{
		ProjectID: testProjectID,
		Tags:      nil, // nil
		TopK:      10,
	}
	results, err := store.Search(ctx, embedding, opts)
	assertNoError(t, err)

	// 全てのノートが返されること
	if len(results) != 2 {
		t.Fatalf("Expected 2 results (no filter), got %d", len(results))
	}
}

// TestChromaStore_Search_TagsFilter_CaseSensitive は大小文字区別をテスト
func TestChromaStore_Search_TagsFilter_CaseSensitive(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 大小文字が異なるタグのノートを追加
	embedding := dummyEmbedding(1536)
	note1 := newTestNoteWithTags("case1", testProjectID, testGroupID, "Note 1", []string{"Go", "Test"})
	note2 := newTestNoteWithTags("case2", testProjectID, testGroupID, "Note 2", []string{"go", "test"})

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))

	// tags: ["go", "test"] で検索（小文字）
	opts := SearchOptions{
		ProjectID: testProjectID,
		Tags:      []string{"go", "test"},
		TopK:      10,
	}
	results, err := store.Search(ctx, embedding, opts)
	assertNoError(t, err)

	// note2のみが返されること（大小文字区別）
	if len(results) != 1 {
		t.Fatalf("Expected 1 result (case-sensitive), got %d", len(results))
	}
	if results[0].Note.ID != "case2" {
		t.Errorf("Expected note case2, got %s", results[0].Note.ID)
	}
}

// TestChromaStore_Search_SinceFilter はsince <= createdAtをテスト
func TestChromaStore_Search_SinceFilter(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 異なる作成日時のノートを追加
	embedding := dummyEmbedding(1536)
	time1 := time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	time3 := time.Date(2024, 1, 20, 10, 0, 0, 0, time.UTC)

	note1 := newTestNoteWithCreatedAt("since1", testProjectID, testGroupID, "Note 1", time1)
	note2 := newTestNoteWithCreatedAt("since2", testProjectID, testGroupID, "Note 2", time2)
	note3 := newTestNoteWithCreatedAt("since3", testProjectID, testGroupID, "Note 3", time3)

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))
	assertNoError(t, store.AddNote(ctx, note3, embedding))

	// since: 2024-01-15T10:00:00Z で検索
	since := time2
	opts := SearchOptions{
		ProjectID: testProjectID,
		Since:     &since,
		TopK:      10,
	}
	results, err := store.Search(ctx, embedding, opts)
	assertNoError(t, err)

	// note2とnote3のみが返されること（境界条件: since <= createdAt）
	if len(results) != 2 {
		t.Fatalf("Expected 2 results (since filter), got %d", len(results))
	}

	for _, r := range results {
		createdAt, _ := time.Parse(time.RFC3339, *r.Note.CreatedAt)
		if createdAt.Before(since) {
			t.Errorf("Result createdAt %s is before since %s", createdAt, since)
		}
	}
}

// TestChromaStore_Search_UntilFilter はcreatedAt < untilをテスト
func TestChromaStore_Search_UntilFilter(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 異なる作成日時のノートを追加
	embedding := dummyEmbedding(1536)
	time1 := time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	time3 := time.Date(2024, 1, 20, 10, 0, 0, 0, time.UTC)

	note1 := newTestNoteWithCreatedAt("until1", testProjectID, testGroupID, "Note 1", time1)
	note2 := newTestNoteWithCreatedAt("until2", testProjectID, testGroupID, "Note 2", time2)
	note3 := newTestNoteWithCreatedAt("until3", testProjectID, testGroupID, "Note 3", time3)

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))
	assertNoError(t, store.AddNote(ctx, note3, embedding))

	// until: 2024-01-15T10:00:00Z で検索
	until := time2
	opts := SearchOptions{
		ProjectID: testProjectID,
		Until:     &until,
		TopK:      10,
	}
	results, err := store.Search(ctx, embedding, opts)
	assertNoError(t, err)

	// note1のみが返されること（境界条件: createdAt < until）
	if len(results) != 1 {
		t.Fatalf("Expected 1 result (until filter), got %d", len(results))
	}
	if results[0].Note.ID != "until1" {
		t.Errorf("Expected note until1, got %s", results[0].Note.ID)
	}
}

// TestChromaStore_Search_BoundaryCondition は境界条件テスト
func TestChromaStore_Search_BoundaryCondition(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 同一時刻のノートを追加
	embedding := dummyEmbedding(1536)
	exactTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	note := newTestNoteWithCreatedAt("boundary", testProjectID, testGroupID, "Boundary test", exactTime)

	assertNoError(t, store.AddNote(ctx, note, embedding))

	// since: 境界値で検索（since <= createdAt なので含まれる）
	opts1 := SearchOptions{
		ProjectID: testProjectID,
		Since:     &exactTime,
		TopK:      10,
	}
	results1, err := store.Search(ctx, embedding, opts1)
	assertNoError(t, err)
	if len(results1) != 1 {
		t.Errorf("Expected 1 result with since=exactTime (since <= createdAt), got %d", len(results1))
	}

	// until: 境界値で検索（createdAt < until なので含まれない）
	opts2 := SearchOptions{
		ProjectID: testProjectID,
		Until:     &exactTime,
		TopK:      10,
	}
	results2, err := store.Search(ctx, embedding, opts2)
	assertNoError(t, err)
	if len(results2) != 0 {
		t.Errorf("Expected 0 results with until=exactTime (createdAt < until), got %d", len(results2))
	}
}

// TestChromaStore_ListRecent_Basic はcreatedAt降順で取得をテスト
func TestChromaStore_ListRecent_Basic(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 異なる作成日時のノートを追加
	embedding := dummyEmbedding(1536)
	time1 := time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	time3 := time.Date(2024, 1, 20, 10, 0, 0, 0, time.UTC)

	note1 := newTestNoteWithCreatedAt("recent1", testProjectID, testGroupID, "Note 1", time1)
	note2 := newTestNoteWithCreatedAt("recent2", testProjectID, testGroupID, "Note 2", time2)
	note3 := newTestNoteWithCreatedAt("recent3", testProjectID, testGroupID, "Note 3", time3)

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))
	assertNoError(t, store.AddNote(ctx, note3, embedding))

	// ListRecent実行
	opts := ListOptions{
		ProjectID: testProjectID,
		Limit:     10,
	}
	notes, err := store.ListRecent(ctx, opts)
	assertNoError(t, err)

	if len(notes) != 3 {
		t.Fatalf("Expected 3 notes, got %d", len(notes))
	}

	// createdAt降順を確認
	assertNotesOrderedByCreatedAt(t, notes)

	// 最新のノートがnote3であることを確認
	if notes[0].ID != "recent3" {
		t.Errorf("First note should be recent3 (latest), got %s", notes[0].ID)
	}
}

// TestChromaStore_ListRecent_Limit はLimit指定で件数制限をテスト
func TestChromaStore_ListRecent_Limit(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 5件のノートを追加
	embedding := dummyEmbedding(1536)
	for i := 1; i <= 5; i++ {
		note := newTestNote("limit-"+string(rune('0'+i)), testProjectID, testGroupID, "Note "+string(rune('0'+i)))
		assertNoError(t, store.AddNote(ctx, note, embedding))
	}

	// Limit=3で取得
	opts := ListOptions{
		ProjectID: testProjectID,
		Limit:     3,
	}
	notes, err := store.ListRecent(ctx, opts)
	assertNoError(t, err)

	if len(notes) != 3 {
		t.Fatalf("Expected 3 notes (Limit=3), got %d", len(notes))
	}
}

// TestChromaStore_ListRecent_Filters はフィルタ適用をテスト
func TestChromaStore_ListRecent_Filters(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// タグ付きノートを追加
	embedding := dummyEmbedding(1536)
	note1 := newTestNoteWithTags("filter1", testProjectID, testGroupID, "Note 1", []string{"go", "test"})
	note2 := newTestNoteWithTags("filter2", testProjectID, testGroupID, "Note 2", []string{"go"})
	note3 := newTestNoteWithTags("filter3", testProjectID, testGroupID, "Note 3", []string{"python"})

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))
	assertNoError(t, store.AddNote(ctx, note3, embedding))

	// tags: ["go"] でフィルタ
	opts := ListOptions{
		ProjectID: testProjectID,
		Tags:      []string{"go"},
		Limit:     10,
	}
	notes, err := store.ListRecent(ctx, opts)
	assertNoError(t, err)

	// note1とnote2のみが返されること
	if len(notes) != 2 {
		t.Fatalf("Expected 2 notes with 'go' tag, got %d", len(notes))
	}

	for _, n := range notes {
		if !containsTag(n.Tags, "go") {
			t.Errorf("Result should contain 'go' tag, got %v", n.Tags)
		}
	}
}

// TestChromaStore_ListRecent_GroupIDNil はgroupID=nilで全group取得をテスト
func TestChromaStore_ListRecent_GroupIDNil(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 異なるgroupIDのノートを追加
	embedding := dummyEmbedding(1536)
	note1 := newTestNote("lgn1", testProjectID, testGroupID, "Group 1 note")
	note2 := newTestNote("lgn2", testProjectID, testGroupID2, "Group 2 note")

	assertNoError(t, store.AddNote(ctx, note1, embedding))
	assertNoError(t, store.AddNote(ctx, note2, embedding))

	// groupID=nilで全group取得
	opts := ListOptions{
		ProjectID: testProjectID,
		GroupID:   nil, // 全group対象
		Limit:     10,
	}
	notes, err := store.ListRecent(ctx, opts)
	assertNoError(t, err)

	// 両方のgroupのノートが返されること
	if len(notes) != 2 {
		t.Fatalf("Expected 2 notes (all groups), got %d", len(notes))
	}
}

// TestChromaStore_UpsertGlobal_Insert は新規追加をテスト
func TestChromaStore_UpsertGlobal_Insert(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	config := newTestGlobalConfig(testProjectID, "global.test-key", "test-value")

	err := store.UpsertGlobal(ctx, config)
	assertNoError(t, err)

	// 取得して検証
	retrieved, found, err := store.GetGlobal(ctx, testProjectID, "global.test-key")
	assertNoError(t, err)
	if !found {
		t.Fatal("Config should be found")
	}
	if retrieved.Value != "test-value" {
		t.Errorf("Value mismatch: expected=%q, actual=%q", "test-value", retrieved.Value)
	}
}

// TestChromaStore_UpsertGlobal_Update は更新をテスト
func TestChromaStore_UpsertGlobal_Update(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 新規追加
	config := newTestGlobalConfig(testProjectID, "global.update-key", "initial-value")
	err := store.UpsertGlobal(ctx, config)
	assertNoError(t, err)

	// 更新
	config.Value = "updated-value"
	err = store.UpsertGlobal(ctx, config)
	assertNoError(t, err)

	// 取得して検証
	retrieved, found, err := store.GetGlobal(ctx, testProjectID, "global.update-key")
	assertNoError(t, err)
	if !found {
		t.Fatal("Config should be found")
	}
	if retrieved.Value != "updated-value" {
		t.Errorf("Value mismatch: expected=%q, actual=%q", "updated-value", retrieved.Value)
	}
}

// TestChromaStore_GetGlobal_Found は取得成功をテスト
func TestChromaStore_GetGlobal_Found(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	// 準備
	config := newTestGlobalConfig(testProjectID, "global.found-key", "found-value")
	err := store.UpsertGlobal(ctx, config)
	assertNoError(t, err)

	// 取得
	retrieved, found, err := store.GetGlobal(ctx, testProjectID, "global.found-key")
	assertNoError(t, err)
	if !found {
		t.Fatal("Config should be found")
	}
	if retrieved.ProjectID != testProjectID {
		t.Errorf("ProjectID mismatch: expected=%q, actual=%q", testProjectID, retrieved.ProjectID)
	}
	if retrieved.Key != "global.found-key" {
		t.Errorf("Key mismatch: expected=%q, actual=%q", "global.found-key", retrieved.Key)
	}
	if retrieved.Value != "found-value" {
		t.Errorf("Value mismatch: expected=%q, actual=%q", "found-value", retrieved.Value)
	}
}

// TestChromaStore_GetGlobal_NotFound はfound=falseをテスト
func TestChromaStore_GetGlobal_NotFound(t *testing.T) {

	ctx := setupTestContext()
	store := setupTestStore(t)
	defer store.Close()

	_, found, err := store.GetGlobal(ctx, testProjectID, "global.non-existent-key")
	assertNoError(t, err)
	if found {
		t.Error("Config should not be found")
	}
}

// TestChromaStore_Namespace_Isolation は異なるnamespaceは別コレクションをテスト
func TestChromaStore_Namespace_Isolation(t *testing.T) {

	ctx := setupTestContext()

	// 異なるnamespaceのストアを作成
	store1 := setupTestStoreWithNamespace(t, "openai:text-embedding-ada-002:1536")
	store2 := setupTestStoreWithNamespace(t, "openai:text-embedding-3-small:1536")
	defer store1.Close()
	defer store2.Close()

	// store1にノート追加
	embedding := dummyEmbedding(1536)
	note1 := newTestNote("ns1", testProjectID, testGroupID, "Namespace 1 note")
	err := store1.AddNote(ctx, note1, embedding)
	assertNoError(t, err)

	// store2からは取得できない（namespace分離）
	_, err = store2.Get(ctx, "ns1")
	assertErrorIs(t, err, ErrNotFound)

	// store2にノート追加
	note2 := newTestNote("ns2", testProjectID, testGroupID, "Namespace 2 note")
	err = store2.AddNote(ctx, note2, embedding)
	assertNoError(t, err)

	// store1からは取得できない（namespace分離）
	_, err = store1.Get(ctx, "ns2")
	assertErrorIs(t, err, ErrNotFound)
}

// Helper functions

// setupTestStore はテスト用のChromaStoreを初期化
func setupTestStore(t *testing.T) Store {
	t.Helper()
	return setupTestStoreWithNamespace(t, testNamespace)
}

// setupTestStoreWithNamespace はnamespace指定でテスト用のChromaStoreを初期化
func setupTestStoreWithNamespace(t *testing.T, namespace string) Store {
	t.Helper()

	// MemoryStoreを使用（Chromaサーバー不要）
	store := NewMemoryStore()
	ctx := setupTestContext()

	err := store.Initialize(ctx, namespace)
	if err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}

	return store
}
