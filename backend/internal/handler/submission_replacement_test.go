package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type replacementFiles struct {
	mu         sync.Mutex
	data       map[string][]byte
	deletes    int
	failUpload bool
	onUpload   func()
}

func (s *replacementFiles) Upload(_ context.Context, data []byte, _ string) (string, error) {
	if s.onUpload != nil {
		s.onUpload()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failUpload {
		return "", errors.New("synthetic upload failure")
	}
	key := uuid.NewString()
	s.data[key] = bytes.Clone(data)
	return key, nil
}
func (s *replacementFiles) Download(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.data[key]
	if !ok {
		return nil, errors.New("missing file")
	}
	return bytes.Clone(data), nil
}
func (s *replacementFiles) ReadText(ctx context.Context, key string) (string, error) {
	data, err := s.Download(ctx, key)
	return string(data), err
}
func (s *replacementFiles) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deletes++
	delete(s.data, key)
	return nil
}

func replacementFile(t *testing.T, content string) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "replacement.py")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = io.WriteString(part, content); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("PUT", "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err = req.ParseMultipartForm(1024 * 1024); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = req.MultipartForm.RemoveAll() })
	return req.MultipartForm.File["file"][0]
}

func replacementState(t *testing.T, db *gorm.DB, sid string) string {
	t.Helper()
	var state string
	if err := db.Raw(`SELECT row_to_json(s)::text || COALESCE((SELECT json_agg(c ORDER BY id)::text FROM comments c WHERE c.submission_id=s.id),'[]') FROM submissions s WHERE s.id=?`, sid).Scan(&state).Error; err != nil {
		t.Fatal(err)
	}
	return state
}

func replacementFixture(t *testing.T, db *gorm.DB) (*submissionFixture, *replacementFiles) {
	t.Helper()
	f := newSubmissionFixture(t, db)
	files := &replacementFiles{data: f.files.data}
	storage.Provider = files
	mustExec(t, f.db, `UPDATE submissions SET point=0, graded_by='teacher' WHERE id=?`, f.own)
	return f, files
}

func TestSubmissionReplacement(t *testing.T) {
	db := submissionTestDB(t)
	for _, scenario := range []string{"success", "fileless", "first-file", "shared-attachment", "shared-key", "repeated", "concurrent-grade", "open-failure", "upload-failure", "attachment-insert-failure", "reference-update-failure", "response-reload-failure", "missing-at-write", "too-large", "hidden", "closed"} {
		t.Run(scenario, func(t *testing.T) {
			f, files := replacementFixture(t, db)
			var original models.Submission
			if err := f.db.Preload("Attachment").First(&original, "id=?", f.own).Error; err != nil {
				t.Fatal(err)
			}
			oldKey, oldID := original.Attachment.FileKey, *original.AttachmentID
			oldBytes := bytes.Clone(files.data[oldKey])
			file := replacementFile(t, "replacement answer")
			fail := false
			var injected bool
			removeCallbacks := func() {
				f.db.Callback().Create().Remove("r6_create")
				f.db.Callback().Update().Remove("r6_update")
				f.db.Callback().Query().Remove("r6_query")
			}
			t.Cleanup(removeCallbacks)
			switch scenario {
			case "fileless":
				file = nil
			case "first-file":
				mustExec(t, f.db, `UPDATE submissions SET attachment_id=NULL WHERE id=?`, f.own)
			case "shared-attachment":
				mustExec(t, f.db, `UPDATE submissions SET attachment_id=? WHERE id=?`, oldID, f.peer)
			case "shared-key":
				mustExec(t, f.db, `UPDATE attachments SET file_key=? WHERE id=(SELECT attachment_id FROM submissions WHERE id=?)`, oldKey, f.peer)
			case "concurrent-grade":
				files.onUpload = func() { mustExec(t, f.db, `UPDATE submissions SET point=9, graded_by='teacher' WHERE id=?`, f.own) }
			case "open-failure":
				file = &multipart.FileHeader{Filename: "unreadable.py"}
				fail = true
			case "upload-failure":
				files.failUpload = true
				fail = true
			case "attachment-insert-failure":
				fail = true
				f.db.Callback().Create().Before("gorm:create").Register("r6_create", func(tx *gorm.DB) {
					if tx.Statement.Table == "attachments" {
						injected = true
						tx.AddError(errors.New("synthetic insert failure"))
					}
				})
			case "reference-update-failure":
				fail = true
				f.db.Callback().Update().Before("gorm:update").Register("r6_update", func(tx *gorm.DB) {
					if tx.Statement.Table == "submissions" {
						injected = true
						tx.AddError(errors.New("synthetic update failure"))
					}
				})
			case "response-reload-failure":
				fail = true
				f.db.Callback().Update().After("gorm:update").Register("r6_update", func(tx *gorm.DB) {
					if tx.Statement.Table == "submissions" {
						injected = true
					}
				})
				f.db.Callback().Query().Before("gorm:query").Register("r6_query", func(tx *gorm.DB) {
					if injected && tx.Statement.Table == "submissions" {
						tx.AddError(errors.New("synthetic reload failure"))
					}
				})
			case "missing-at-write":
				fail = true
				f.db.Callback().Create().Before("gorm:create").Register("r6_create", func(tx *gorm.DB) {
					if tx.Statement.Table == "attachments" {
						injected = true
						tx.Exec("DELETE FROM submissions WHERE id=?", f.own)
					}
				})
			case "too-large":
				file.Size = 5*1024*1024 + 1
				fail = true
			case "hidden":
				mustExec(t, f.db, `UPDATE assignments SET visible=false WHERE id=?`, f.a1)
				fail = true
			case "closed":
				mustExec(t, f.db, `UPDATE assignments SET close_date=now()-interval '1 hour' WHERE id=?`, f.a1)
				fail = true
			}
			before := replacementState(t, f.db, f.own)
			var commentsBefore string
			if err := f.db.Raw(`SELECT COALESCE(json_agg(c ORDER BY id)::text,'[]') FROM comments c WHERE submission_id=?`, f.own).Scan(&commentsBefore).Error; err != nil {
				t.Fatal(err)
			}
			var countBefore int64
			f.db.Model(&models.Attachment{}).Count(&countBefore)
			_, err := service.UpdateSubmission(f.own, f.student, file)
			removeCallbacks()
			var commentsAfter string
			if err := f.db.Raw(`SELECT COALESCE(json_agg(c ORDER BY id)::text,'[]') FROM comments c WHERE submission_id=?`, f.own).Scan(&commentsAfter).Error; err != nil {
				t.Fatal(err)
			}
			if commentsAfter != commentsBefore {
				t.Fatal("replacement changed feedback")
			}
			if fail && err == nil {
				t.Fatal("failure was not reported")
			}
			if !fail && err != nil {
				t.Fatal(err)
			}
			if files.deletes != 0 || !bytes.Equal(files.data[oldKey], oldBytes) {
				t.Fatal("replacement deleted or changed original bytes")
			}
			var old models.Attachment
			if err := f.db.First(&old, "id=?", oldID).Error; err != nil || old.FileKey != oldKey {
				t.Fatal("original attachment record lost")
			}
			if fail {
				if replacementState(t, f.db, f.own) != before {
					t.Fatal("failed replacement changed reference, grade, or comments")
				}
				var countAfter int64
				f.db.Model(&models.Attachment{}).Count(&countAfter)
				if countAfter != countBefore {
					t.Fatal("failed transaction retained attachment row")
				}
				if scenario == "attachment-insert-failure" || scenario == "reference-update-failure" || scenario == "response-reload-failure" || scenario == "missing-at-write" {
					if !injected {
						t.Fatal("failure injection did not run")
					}
				}
				return
			}
			var current models.Submission
			if err := f.db.Preload("Attachment").First(&current, "id=?", f.own).Error; err != nil {
				t.Fatal(err)
			}
			if scenario == "fileless" {
				if current.AttachmentID != nil {
					t.Fatal("fileless update did not detach")
				}
			} else if current.Attachment == nil || string(files.data[current.Attachment.FileKey]) != "replacement answer" {
				t.Fatal("new reference does not resolve to uploaded bytes")
			}
			want := int16(0)
			if scenario == "concurrent-grade" {
				want = 9
			}
			if current.Point == nil || *current.Point != want || current.GradedBy == nil || *current.GradedBy != "teacher" {
				t.Fatal("replacement altered grade")
			}
			if scenario == "repeated" {
				first := current.Attachment.FileKey
				if _, err := service.UpdateSubmission(f.own, f.student, replacementFile(t, "next answer")); err != nil {
					t.Fatal(err)
				}
				if string(files.data[first]) != "replacement answer" {
					t.Fatal("repeat deleted previous upload")
				}
			}
			if scenario == "shared-attachment" || scenario == "shared-key" {
				var peer models.Submission
				if err := f.db.Preload("Attachment").First(&peer, "id=?", f.peer).Error; err != nil {
					t.Fatal(err)
				}
				if peer.Attachment == nil || !bytes.Equal(files.data[peer.Attachment.FileKey], oldBytes) {
					t.Fatal("shared file changed")
				}
			}
		})
	}
}

func TestConcurrentSubmissionReplacements(t *testing.T) {
	db := submissionTestDB(t)
	for _, detach := range []bool{false, true} {
		t.Run(map[bool]string{false: "two-uploads", true: "upload-and-detach"}[detach], func(t *testing.T) {
			f, files := replacementFixture(t, db)
			if err := f.db.Commit().Error; err != nil {
				t.Fatal(err)
			}
			f.db = db
			database.DB = db
			first, second := replacementFile(t, "first"), replacementFile(t, "second")
			if detach {
				second = nil
			}
			done := make(chan error, 2)
			start := make(chan struct{})
			for _, file := range []*multipart.FileHeader{first, second} {
				go func(file *multipart.FileHeader) {
					<-start
					_, err := service.UpdateSubmission(f.own, f.student, file)
					done <- err
				}(file)
			}
			close(start)
			for range 2 {
				if err := <-done; err != nil {
					t.Fatal(err)
				}
			}
			var current models.Submission
			if err := db.Preload("Attachment").First(&current, "id=?", f.own).Error; err != nil {
				t.Fatal(err)
			}
			if current.Attachment != nil {
				if _, err := files.Download(context.Background(), current.Attachment.FileKey); err != nil {
					t.Fatal("current file missing")
				}
			} else if !detach {
				t.Fatal("upload unexpectedly detached")
			}
			files.mu.Lock()
			deletes := files.deletes
			data, _ := json.Marshal(files.data)
			files.mu.Unlock()
			if deletes != 0 || len(data) == 0 {
				t.Fatal("concurrent request deleted a file")
			}
		})
	}
}
