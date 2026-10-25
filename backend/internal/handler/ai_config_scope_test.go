package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	apperrors "github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type aiConfigFixture struct {
	*submissionFixture
	target, sibling, otherCredentialSibling, foreignConfig string
	credential, secondCredential, foreignCredential        string
}

func aiConfigTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	mustExec(t, db, `CREATE TABLE ai_credentials (id uuid PRIMARY KEY, user_id uuid, name text, provider text, base_url text, encrypted_api_key text, created_at timestamptz, updated_at timestamptz)`)
	mustExec(t, db, `CREATE TABLE ai_configs (id uuid PRIMARY KEY, user_id uuid, name text, ai_credential_id uuid, model text, temperature real, created_at timestamptz, updated_at timestamptz)`)
}

func newAIConfigFixture(t *testing.T, db *gorm.DB) *aiConfigFixture {
	t.Helper()
	f := &aiConfigFixture{submissionFixture: newSubmissionFixture(t, db)}
	credential := func(owner string) string {
		id := uuid.NewString()
		mustExec(t, f.db, `INSERT INTO ai_credentials (id,user_id,name,provider,encrypted_api_key,created_at,updated_at) VALUES (?,?,'fixture','custom','synthetic',now(),now())`, id, owner)
		return id
	}
	f.credential, f.secondCredential, f.foreignCredential = credential(f.teacher), credential(f.teacher), credential(f.teacherAsStudent)
	config := func(owner, credential string) string {
		id := uuid.NewString()
		mustExec(t, f.db, `INSERT INTO ai_configs (id,user_id,name,ai_credential_id,model,temperature,created_at,updated_at) VALUES (?,?,'original',?,'original-model',0.4,now(),now())`, id, owner, credential)
		return id
	}
	f.target = config(f.teacher, f.credential)
	f.sibling = config(f.teacher, f.credential)
	f.otherCredentialSibling = config(f.teacher, f.secondCredential)
	f.foreignConfig = config(f.teacherAsStudent, f.foreignCredential)
	return f
}

func (f *aiConfigFixture) snapshot(t *testing.T, exclude string) string {
	t.Helper()
	var result string
	if err := f.db.Raw(`SELECT COALESCE((SELECT json_agg(c ORDER BY id)::text FROM ai_configs c WHERE id::text<>?),'[]') || COALESCE((SELECT json_agg(a ORDER BY id)::text FROM ai_credentials a),'[]')`, exclude).Scan(&result).Error; err != nil {
		t.Fatal(err)
	}
	return result
}

func (f *aiConfigFixture) patch(t *testing.T, actor, target, body string) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest("PATCH", "/api/ai/configs/"+target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if actor != "" {
		token, err := service.GenerateToken(actor)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := f.app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, data
}

func TestAIConfigUpdateScope(t *testing.T) {
	db := submissionTestDB(t)
	aiConfigTables(t, db)
	for _, scenario := range []string{"target-only", "zero-and-empty-fields", "missing-config", "foreign-config", "missing-credential", "foreign-credential", "omitted-credential", "student", "global-ta", "admin", "no-auth", "malformed", "update-failure", "owner-changed", "target-deleted"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAIConfigFixture(t, db)
			actor, target, credential, want := f.teacher, f.target, f.secondCredential, 200
			name, model, temp := "changed", "new-model", float32(0.7)
			var injected bool
			cleanup := func() { f.db.Callback().Update().Remove("r7_update") }
			t.Cleanup(cleanup)
			switch scenario {
			case "zero-and-empty-fields":
				name, model, temp = "", "", 0
			case "missing-config":
				target = uuid.NewString()
				want = 404
			case "foreign-config":
				target = f.foreignConfig
				want = 404
			case "missing-credential":
				credential = uuid.NewString()
				want = 404
			case "foreign-credential":
				credential = f.foreignCredential
				want = 404
			case "omitted-credential":
				credential = ""
				want = 500 // Existing empty UUID validation is separate R9 work.
			case "student":
				actor = f.student
				want = 403
			case "global-ta":
				mustExec(t, f.db, `UPDATE users SET role='ta' WHERE id=?`, actor)
				want = 403
			case "admin":
				actor = f.admin
				want = 403
			case "no-auth":
				actor = ""
				want = 401
			case "malformed":
				want = 400
			case "update-failure":
				want = 500
			case "owner-changed", "target-deleted":
				want = 404
			}
			if scenario == "update-failure" || scenario == "owner-changed" || scenario == "target-deleted" {
				f.db.Callback().Update().Before("gorm:update").Register("r7_update", func(tx *gorm.DB) {
					if tx.Statement.Table != "ai_configs" {
						return
					}
					injected = true
					switch scenario {
					case "update-failure":
						tx.AddError(errors.New("synthetic update failure"))
					case "owner-changed":
						tx.Session(&gorm.Session{NewDB: true}).Exec(`UPDATE ai_configs SET user_id=? WHERE id=?`, f.teacherAsStudent, f.target)
					case "target-deleted":
						tx.Session(&gorm.Session{NewDB: true}).Exec(`DELETE FROM ai_configs WHERE id=?`, f.target)
					}
				})
			}
			exclude := ""
			if want == 200 || scenario == "owner-changed" || scenario == "target-deleted" {
				exclude = f.target
			}
			before := f.snapshot(t, exclude)
			body, _ := json.Marshal(map[string]any{"name": name, "model": model, "temperature": temp, "credential_id": credential})
			if scenario == "malformed" {
				body = []byte("{")
			}
			if scenario == "omitted-credential" {
				// A malformed UUID aborts the outer fixture transaction; production
				// runs this lookup in autocommit. Isolate that existing failure.
				if err := f.db.SavePoint("r7_empty_credential").Error; err != nil {
					t.Fatal(err)
				}
			}
			status, data := f.patch(t, actor, target, string(body))
			if scenario == "omitted-credential" {
				if err := f.db.RollbackTo("r7_empty_credential").Error; err != nil {
					t.Fatal(err)
				}
			}
			cleanup()
			if status != want {
				t.Fatalf("status=%d want=%d: %s", status, want, data)
			}
			if f.snapshot(t, exclude) != before {
				t.Fatal("update changed another config or credential")
			}
			if scenario == "update-failure" || scenario == "owner-changed" || scenario == "target-deleted" {
				if !injected {
					t.Fatal("write-time race/failure was not exercised")
				}
			}
			if want == 200 {
				var current models.AIConfig
				if err := f.db.First(&current, "id=?", f.target).Error; err != nil {
					t.Fatal(err)
				}
				if name == "" {
					name = "original"
				}
				if model == "" {
					model = "original-model"
				}
				if current.Name != name || current.Model != model || current.Temperature != temp || current.AICredentialID != credential {
					t.Fatal("target fields incorrect")
				}
				var response models.ResponseAIConfig
				if err := json.Unmarshal(data, &response); err != nil {
					t.Fatal(err)
				}
				if response.ID != f.target || response.ConfigName != name || response.CredentialID != credential || response.Model != model || response.Temperature != temp {
					t.Fatal("response does not describe target config")
				}
			}
		})
	}
}

func TestAIConfigRepositoryScope(t *testing.T) {
	db := submissionTestDB(t)
	aiConfigTables(t, db)
	for _, scenario := range []string{"owned", "foreign", "missing"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAIConfigFixture(t, db)
			target, exclude := f.target, f.target
			if scenario == "foreign" {
				target = f.foreignConfig
				exclude = ""
			}
			if scenario == "missing" {
				target = uuid.NewString()
				exclude = ""
			}
			before := f.snapshot(t, exclude)
			err := repository.UpdateAIConfig(f.teacher, target, map[string]any{"name": "direct update", "temperature": float32(0)})
			if scenario == "owned" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, apperrors.ErrAIConfigNotFound) {
				t.Fatalf("expected not found, got %v", err)
			}
			if f.snapshot(t, exclude) != before {
				t.Fatal("direct update changed an unrelated record")
			}
		})
	}
}
