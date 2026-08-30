package rag

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	chunkSize    = 800
	chunkOverlap = 120
	batchSize    = 20
)

type Service struct {
	db         *sql.DB
	httpClient *http.Client
	indexMu    sync.Mutex
}

func DefaultDatabasePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		return "", fmt.Errorf("无法定位用户配置目录")
	}
	return filepath.Join(configDir, "Q-Solver", "rag", "knowledge.db"), nil
}

func NewService(databasePath string) (*Service, error) {
	if databasePath == "" {
		var err error
		databasePath, err = DefaultDatabasePath()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(databasePath), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	service := &Service{db: db, httpClient: &http.Client{Timeout: 30 * time.Second}}
	if err := service.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return service, nil
}

func (s *Service) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Service) SetHTTPClient(client *http.Client) {
	if client != nil {
		s.httpClient = client
	}
}

func (s *Service) initSchema() error {
	statements := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA busy_timeout=5000`,
		`CREATE TABLE IF NOT EXISTS documents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL,
			content_hash TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'local',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS knowledge_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			document_id INTEGER REFERENCES documents(id) ON DELETE CASCADE,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			question TEXT NOT NULL DEFAULT '',
			answer TEXT NOT NULL DEFAULT '',
			search_text TEXT NOT NULL,
			vector BLOB,
			vector_model TEXT NOT NULL DEFAULT '',
			vector_dimensions INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_document ON knowledge_items(document_id)`,
		`CREATE INDEX IF NOT EXISTS idx_knowledge_vector ON knowledge_items(vector_model, vector_dimensions)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_fts USING fts5(title, search_text, content='knowledge_items', content_rowid='id', tokenize='unicode61')`,
		`CREATE TRIGGER IF NOT EXISTS knowledge_items_ai AFTER INSERT ON knowledge_items BEGIN
			INSERT INTO knowledge_fts(rowid, title, search_text) VALUES (new.id, new.title, new.search_text);
		END`,
		`CREATE TRIGGER IF NOT EXISTS knowledge_items_ad AFTER DELETE ON knowledge_items BEGIN
			INSERT INTO knowledge_fts(knowledge_fts, rowid, title, search_text) VALUES('delete', old.id, old.title, old.search_text);
		END`,
		`CREATE TRIGGER IF NOT EXISTS knowledge_items_au AFTER UPDATE ON knowledge_items BEGIN
			INSERT INTO knowledge_fts(knowledge_fts, rowid, title, search_text) VALUES('delete', old.id, old.title, old.search_text);
			INSERT INTO knowledge_fts(rowid, title, search_text) VALUES (new.id, new.title, new.search_text);
		END`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("初始化 RAG 数据库失败: %w", err)
		}
	}
	return nil
}

func (s *Service) ImportFile(ctx context.Context, path string, settings Settings) (ImportResult, error) {
	kind, content, err := extractFile(path)
	if err != nil {
		return ImportResult{Path: path}, err
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
	if existing, found, err := s.findDocumentByHash(ctx, hash); err != nil {
		return ImportResult{}, err
	} else if found {
		return ImportResult{Path: path, Document: existing, Duplicated: true}, nil
	}

	s.indexMu.Lock()
	defer s.indexMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ImportResult{}, err
	}
	defer tx.Rollback()
	now := time.Now().UnixMilli()
	status := "local"
	if settings.Mode != ModeLocal {
		status = "pending"
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO documents(name,path,kind,content_hash,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, filepath.Base(path), path, kind, hash, status, now, now)
	if err != nil {
		return ImportResult{}, err
	}
	documentID, _ := result.LastInsertId()
	chunks := splitText(content, chunkSize, chunkOverlap)
	for index, chunk := range chunks {
		title := fmt.Sprintf("%s（%d/%d）", filepath.Base(path), index+1, len(chunks))
		if _, err := tx.ExecContext(ctx, `INSERT INTO knowledge_items(document_id,kind,title,content,search_text,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, documentID, "document", title, chunk, searchableText(title+"\n"+chunk), now, now); err != nil {
			return ImportResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return ImportResult{}, err
	}
	document := Document{ID: documentID, Name: filepath.Base(path), Path: path, Kind: kind, Status: status, ChunkCount: len(chunks), CreatedAt: now, UpdatedAt: now}
	importResult := ImportResult{Path: path, Document: document}
	if settings.Mode != ModeLocal {
		if _, err := s.indexDocument(ctx, documentID, settings); err != nil {
			importResult.Warning = err.Error()
		} else {
			importResult.Document.Status = "ready"
		}
	}
	return importResult, nil
}

func (s *Service) AddQA(ctx context.Context, question, answer string, settings Settings) (QAEntry, error) {
	question = strings.TrimSpace(question)
	answer = strings.TrimSpace(answer)
	if question == "" || answer == "" {
		return QAEntry{}, fmt.Errorf("问题和参考答案不能为空")
	}
	now := time.Now().UnixMilli()
	content := "问题：" + question + "\n参考答案：" + answer
	status := "local"
	if settings.Mode != ModeLocal {
		status = "pending"
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO knowledge_items(kind,title,content,question,answer,search_text,created_at,updated_at) VALUES('qa',?,?,?,?,?,?,?)`, question, content, question, answer, searchableText(content), now, now)
	if err != nil {
		return QAEntry{}, err
	}
	id, _ := result.LastInsertId()
	entry := QAEntry{ID: id, Question: question, Answer: answer, Status: status, CreatedAt: now, UpdatedAt: now}
	if settings.Mode != ModeLocal {
		if err := s.indexItems(ctx, []int64{id}, []string{content}, settings); err != nil {
			entry.Warning = err.Error()
			return entry, nil
		}
		entry.Status = "ready"
	}
	return entry, nil
}

func (s *Service) UpdateQA(ctx context.Context, id int64, question, answer string, settings Settings) error {
	question = strings.TrimSpace(question)
	answer = strings.TrimSpace(answer)
	if question == "" || answer == "" {
		return fmt.Errorf("问题和参考答案不能为空")
	}
	content := "问题：" + question + "\n参考答案：" + answer
	now := time.Now().UnixMilli()
	result, err := s.db.ExecContext(ctx, `UPDATE knowledge_items SET title=?,content=?,question=?,answer=?,search_text=?,vector=NULL,vector_model='',vector_dimensions=0,updated_at=? WHERE id=? AND kind='qa'`, question, content, question, answer, searchableText(content), now, id)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return fmt.Errorf("未找到问答条目")
	}
	if settings.Mode != ModeLocal {
		return s.indexItems(ctx, []int64{id}, []string{content}, settings)
	}
	return nil
}

func (s *Service) DeleteQA(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM knowledge_items WHERE id=? AND kind='qa'`, id)
	return err
}

func (s *Service) DeleteDocument(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM knowledge_items WHERE document_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM documents WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) List(ctx context.Context, settings Settings) (Snapshot, error) {
	snapshot := Snapshot{Documents: []Document{}, QAEntries: []QAEntry{}}
	rows, err := s.db.QueryContext(ctx, `SELECT d.id,d.name,d.path,d.kind,d.status,d.created_at,d.updated_at,COUNT(i.id) FROM documents d LEFT JOIN knowledge_items i ON i.document_id=d.id GROUP BY d.id ORDER BY d.updated_at DESC`)
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var doc Document
		var created, updated int64
		if err := rows.Scan(&doc.ID, &doc.Name, &doc.Path, &doc.Kind, &doc.Status, &created, &updated, &doc.ChunkCount); err != nil {
			rows.Close()
			return snapshot, err
		}
		doc.CreatedAt, doc.UpdatedAt = created, updated
		if doc.Status == "ready" {
			var missing int
			_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM knowledge_items WHERE document_id=? AND (vector_model<>? OR vector_dimensions<>? OR vector IS NULL)`, doc.ID, settings.EmbeddingModel, settings.Dimensions).Scan(&missing)
			if missing > 0 && settings.Mode != ModeLocal {
				doc.Status = "pending"
			}
		}
		snapshot.Documents = append(snapshot.Documents, doc)
	}
	rows.Close()
	qaRows, err := s.db.QueryContext(ctx, `SELECT id,question,answer,vector_model,vector_dimensions,vector IS NOT NULL,created_at,updated_at FROM knowledge_items WHERE kind='qa' ORDER BY updated_at DESC`)
	if err != nil {
		return snapshot, err
	}
	defer qaRows.Close()
	for qaRows.Next() {
		var entry QAEntry
		var model string
		var dimensions int
		var hasVector bool
		var created, updated int64
		if err := qaRows.Scan(&entry.ID, &entry.Question, &entry.Answer, &model, &dimensions, &hasVector, &created, &updated); err != nil {
			return snapshot, err
		}
		entry.Status = "local"
		if settings.Mode != ModeLocal {
			if hasVector && model == settings.EmbeddingModel && dimensions == settings.Dimensions {
				entry.Status = "ready"
			} else {
				entry.Status = "pending"
			}
		}
		entry.CreatedAt, entry.UpdatedAt = created, updated
		snapshot.QAEntries = append(snapshot.QAEntries, entry)
	}
	return snapshot, nil
}

func (s *Service) RebuildIndex(ctx context.Context, settings Settings) (IndexResult, error) {
	if settings.Mode == ModeLocal {
		return IndexResult{}, fmt.Errorf("本地检索模式不需要生成向量")
	}
	s.indexMu.Lock()
	defer s.indexMu.Unlock()
	rows, err := s.db.QueryContext(ctx, `SELECT id,content FROM knowledge_items ORDER BY id`)
	if err != nil {
		return IndexResult{}, err
	}
	defer rows.Close()
	var ids []int64
	var contents []string
	for rows.Next() {
		var id int64
		var content string
		if err := rows.Scan(&id, &content); err != nil {
			return IndexResult{}, err
		}
		ids = append(ids, id)
		contents = append(contents, content)
	}
	result := IndexResult{Total: len(ids)}
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		if err := s.indexItems(ctx, ids[start:end], contents[start:end], settings); err != nil {
			result.Warning = err.Error()
			return result, err
		}
		result.Indexed += end - start
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE documents SET status='ready',updated_at=?`, time.Now().UnixMilli())
	return result, nil
}

func (s *Service) TestEmbedding(ctx context.Context, settings Settings) error {
	_, err := NewEmbeddingClient(settings, s.httpClient).Embed(ctx, []string{"Q-Solver 面试知识库连接测试"})
	return err
}

func (s *Service) Search(ctx context.Context, query string, settings Settings) (SearchResult, error) {
	started := time.Now()
	query = strings.TrimSpace(query)
	if query == "" {
		return SearchResult{Mode: settings.Mode}, fmt.Errorf("检索问题不能为空")
	}
	var result SearchResult
	var err error
	switch settings.Mode {
	case ModeLocal:
		result, err = s.searchLocal(ctx, query, settings.TopK)
	case ModeAPI:
		result, err = s.searchVector(ctx, query, settings)
	case ModeHybrid:
		result, err = s.searchHybrid(ctx, query, settings)
	default:
		err = fmt.Errorf("不支持的 RAG 检索模式")
	}
	result.Mode = settings.Mode
	result.DurationMs = time.Since(started).Milliseconds()
	return result, err
}

func (s *Service) TestSearch(ctx context.Context, query string, settings Settings) (SearchTestResult, error) {
	local, localErr := s.searchLocal(ctx, query, settings.TopK)
	api, apiErr := s.searchVector(ctx, query, settings)
	hybrid := fuseResults(local, api, settings.TopK)
	if localErr != nil {
		local.Warning = localErr.Error()
	}
	if apiErr != nil {
		api.Warning = apiErr.Error()
	}
	if localErr != nil && apiErr != nil {
		return SearchTestResult{Local: local, API: api, Hybrid: hybrid}, fmt.Errorf("本地和 API 检索均失败")
	}
	return SearchTestResult{Local: local, API: api, Hybrid: hybrid}, nil
}

func (s *Service) searchLocal(ctx context.Context, query string, topK int) (SearchResult, error) {
	result := SearchResult{Mode: ModeLocal, Hits: []SearchHit{}}
	match := ftsQuery(query)
	if match == "" {
		return result, nil
	}
	limit := topK * 3
	rows, err := s.db.QueryContext(ctx, `SELECT i.id,i.kind,i.title,i.content,COALESCE(d.name,''),bm25(knowledge_fts,4.0,1.0) FROM knowledge_fts JOIN knowledge_items i ON i.id=knowledge_fts.rowid LEFT JOIN documents d ON d.id=i.document_id WHERE knowledge_fts MATCH ? ORDER BY bm25(knowledge_fts,4.0,1.0) LIMIT ?`, match, limit)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	queryNormalized := strings.TrimSpace(strings.ToLower(query))
	for rows.Next() {
		var hit SearchHit
		var rank float64
		if err := rows.Scan(&hit.ID, &hit.Kind, &hit.Title, &hit.Content, &hit.Source, &rank); err != nil {
			return result, err
		}
		hit.Score = 1 / (1 + math.Abs(rank))
		if hit.Kind == "qa" {
			hit.Source = "手动问答"
			if strings.Contains(strings.ToLower(hit.Title), queryNormalized) || strings.Contains(queryNormalized, strings.ToLower(hit.Title)) {
				hit.Score += 1
			}
		}
		result.Hits = append(result.Hits, hit)
	}
	sort.SliceStable(result.Hits, func(i, j int) bool { return result.Hits[i].Score > result.Hits[j].Score })
	if len(result.Hits) > topK {
		result.Hits = result.Hits[:topK]
	}
	for index := range result.Hits {
		result.Hits[index].LocalRank = index + 1
	}
	return result, rows.Err()
}

func (s *Service) searchVector(ctx context.Context, query string, settings Settings) (SearchResult, error) {
	result := SearchResult{Mode: ModeAPI, Hits: []SearchHit{}}
	vectors, err := NewEmbeddingClient(settings, s.httpClient).Embed(ctx, []string{query})
	if err != nil {
		return result, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT i.id,i.kind,i.title,i.content,COALESCE(d.name,''),i.vector FROM knowledge_items i LEFT JOIN documents d ON d.id=i.document_id WHERE i.vector IS NOT NULL AND i.vector_model=? AND i.vector_dimensions=?`, settings.EmbeddingModel, settings.Dimensions)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var hit SearchHit
		var data []byte
		if err := rows.Scan(&hit.ID, &hit.Kind, &hit.Title, &hit.Content, &hit.Source, &data); err != nil {
			return result, err
		}
		vector, err := decodeVector(data, settings.Dimensions)
		if err != nil {
			continue
		}
		hit.Score = cosineSimilarity(vectors[0], vector)
		if hit.Score < 0.2 {
			continue
		}
		if hit.Kind == "qa" {
			hit.Score += 0.03
			hit.Source = "手动问答"
		}
		result.Hits = append(result.Hits, hit)
	}
	sort.SliceStable(result.Hits, func(i, j int) bool { return result.Hits[i].Score > result.Hits[j].Score })
	if len(result.Hits) > settings.TopK {
		result.Hits = result.Hits[:settings.TopK]
	}
	for index := range result.Hits {
		result.Hits[index].VectorRank = index + 1
	}
	return result, rows.Err()
}

func (s *Service) searchHybrid(ctx context.Context, query string, settings Settings) (SearchResult, error) {
	var local SearchResult
	var vector SearchResult
	var localErr, vectorErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		local, localErr = s.searchLocal(ctx, query, settings.TopK)
	}()
	go func() {
		defer wg.Done()
		vector, vectorErr = s.searchVector(ctx, query, settings)
	}()
	wg.Wait()
	if localErr != nil && vectorErr != nil {
		return SearchResult{Mode: ModeHybrid}, fmt.Errorf("本地检索和 API 向量检索均失败")
	}
	result := fuseResults(local, vector, settings.TopK)
	if localErr != nil {
		result.Warning = "本地检索失败，已使用 API 向量结果"
	} else if vectorErr != nil {
		result.Warning = "API 向量检索失败，已使用本地结果"
	}
	return result, nil
}

func fuseResults(local, vector SearchResult, topK int) SearchResult {
	const rrfK = 60.0
	byID := make(map[int64]SearchHit)
	scores := make(map[int64]float64)
	for index, hit := range local.Hits {
		hit.LocalRank = index + 1
		byID[hit.ID] = hit
		scores[hit.ID] += 0.4 / (rrfK + float64(index+1))
	}
	for index, hit := range vector.Hits {
		existing, found := byID[hit.ID]
		if found {
			existing.VectorRank = index + 1
			byID[hit.ID] = existing
		} else {
			hit.VectorRank = index + 1
			byID[hit.ID] = hit
		}
		scores[hit.ID] += 0.6 / (rrfK + float64(index+1))
	}
	hits := make([]SearchHit, 0, len(byID))
	for id, hit := range byID {
		hit.Score = scores[id]
		if hit.Kind == "qa" {
			hit.Score += 0.002
		}
		hits = append(hits, hit)
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return SearchResult{Mode: ModeHybrid, Hits: hits}
}

func (s *Service) FormatContext(result SearchResult, maxChars int) string {
	if len(result.Hits) == 0 {
		return "知识库未找到可靠的相关资料。请使用通用知识回答，但不得编造候选人的个人经历、公司、项目、职责或数据。"
	}
	var out strings.Builder
	usedRunes := 0
	header := "以下是当前问题从候选人知识库检索到的资料。优先依据这些资料回答；资料未覆盖的部分可以补充通用知识，但不得编造个人事实。\n"
	out.WriteString(header)
	usedRunes += len([]rune(header))
	for index, hit := range result.Hits {
		block := fmt.Sprintf("\n[%d] 来源：%s｜%s\n%s\n", index+1, hit.Source, hit.Title, hit.Content)
		blockRunes := len([]rune(block))
		if maxChars > 0 && usedRunes+blockRunes > maxChars {
			break
		}
		out.WriteString(block)
		usedRunes += blockRunes
	}
	if result.Warning != "" {
		out.WriteString("\n检索提示：" + result.Warning)
	}
	return out.String()
}

func (s *Service) indexDocument(ctx context.Context, documentID int64, settings Settings) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,content FROM knowledge_items WHERE document_id=? ORDER BY id`, documentID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var ids []int64
	var contents []string
	for rows.Next() {
		var id int64
		var content string
		if err := rows.Scan(&id, &content); err != nil {
			return 0, err
		}
		ids = append(ids, id)
		contents = append(contents, content)
	}
	indexed := 0
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		if err := s.indexItems(ctx, ids[start:end], contents[start:end], settings); err != nil {
			return indexed, err
		}
		indexed += end - start
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE documents SET status='ready',updated_at=? WHERE id=?`, time.Now().UnixMilli(), documentID)
	return indexed, nil
}

func (s *Service) indexItems(ctx context.Context, ids []int64, contents []string, settings Settings) error {
	if len(ids) != len(contents) {
		return errors.New("索引条目数量不匹配")
	}
	vectors, err := NewEmbeddingClient(settings, s.httpClient).Embed(ctx, contents)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for index, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE knowledge_items SET vector=?,vector_model=?,vector_dimensions=?,updated_at=? WHERE id=?`, encodeVector(vectors[index]), settings.EmbeddingModel, settings.Dimensions, time.Now().UnixMilli(), id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Service) findDocumentByHash(ctx context.Context, hash string) (Document, bool, error) {
	var doc Document
	var created, updated int64
	err := s.db.QueryRowContext(ctx, `SELECT d.id,d.name,d.path,d.kind,d.status,d.created_at,d.updated_at,(SELECT COUNT(*) FROM knowledge_items i WHERE i.document_id=d.id) FROM documents d WHERE d.content_hash=?`, hash).Scan(&doc.ID, &doc.Name, &doc.Path, &doc.Kind, &doc.Status, &created, &updated, &doc.ChunkCount)
	if errors.Is(err, sql.ErrNoRows) {
		return Document{}, false, nil
	}
	if err != nil {
		return Document{}, false, err
	}
	doc.CreatedAt, doc.UpdatedAt = created, updated
	return doc, true, nil
}

func splitText(text string, size, overlap int) []string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 {
		return nil
	}
	if overlap >= size {
		overlap = 0
	}
	var chunks []string
	for start := 0; start < len(runes); {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		if end < len(runes) {
			for candidate := end; candidate > start+size/2; candidate-- {
				if runes[candidate-1] == '\n' || runes[candidate-1] == '。' || runes[candidate-1] == '；' {
					end = candidate
					break
				}
			}
		}
		chunks = append(chunks, strings.TrimSpace(string(runes[start:end])))
		if end == len(runes) {
			break
		}
		start = end - overlap
	}
	return chunks
}

func encodeVector(vector []float32) []byte {
	data := make([]byte, len(vector)*4)
	for index, value := range vector {
		binary.LittleEndian.PutUint32(data[index*4:], math.Float32bits(value))
	}
	return data
}

func decodeVector(data []byte, dimensions int) ([]float32, error) {
	if len(data) != dimensions*4 {
		return nil, fmt.Errorf("向量维度不匹配")
	}
	vector := make([]float32, dimensions)
	for index := range vector {
		vector[index] = math.Float32frombits(binary.LittleEndian.Uint32(data[index*4:]))
	}
	return vector, nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return -1
	}
	var dot, normA, normB float64
	for index := range a {
		av, bv := float64(a[index]), float64(b[index])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}
	if normA == 0 || normB == 0 {
		return -1
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
