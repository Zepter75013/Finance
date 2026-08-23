package user

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"finance/backend/internal/authctx"
	"finance/backend/internal/database"

	"golang.org/x/crypto/bcrypt"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.OpenSQLite(":memory:")
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return db
}

func hashPassword(t *testing.T) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("test-pass-1234"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword: %v", err)
	}

	return string(hash)
}

func requestAs(method, path string, callerID uint64, payload any) *http.Request {
	var body *bytes.Buffer
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewBuffer(b)
	} else {
		body = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, body)
	req = req.WithContext(authctx.WithUserID(req.Context(), callerID))

	return req
}

func TestCreate_RequiresAdmin(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)
	ctx := t.Context()

	nonAdmin, err := repo.Create(ctx, "nonadmin@test.local", "Non", "Admin", nil, hashPassword(t), nil, false)
	if err != nil {
		t.Fatalf("Create nonAdmin: %v", err)
	}

	req := requestAs(http.MethodPost, "/users", nonAdmin.ID, map[string]any{
		"username": "blocked@test.local", "first_name": "X", "last_name": "Y", "password": "abcd1234",
	})
	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Create par un non-admin: statut = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCreate_AllowedForAdmin(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)
	ctx := t.Context()

	admin, err := repo.Create(ctx, "admin@test.local", "Admin", "Test", nil, hashPassword(t), nil, true)
	if err != nil {
		t.Fatalf("Create admin: %v", err)
	}

	req := requestAs(http.MethodPost, "/users", admin.ID, map[string]any{
		"username": "created@test.local", "first_name": "X", "last_name": "Y", "password": "abcd1234",
	})
	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Create par un admin: statut = %d, want %d (body=%s)", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestUpdate_SelfEditIgnoresPrivilegeEscalation(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)
	ctx := t.Context()

	nonAdmin, err := repo.Create(ctx, "self@test.local", "Self", "Edit", nil, hashPassword(t), nil, false)
	if err != nil {
		t.Fatalf("Create nonAdmin: %v", err)
	}

	req := requestAs(http.MethodPut, "/users/"+itoa(nonAdmin.ID), nonAdmin.ID, map[string]any{
		"username": nonAdmin.Username, "first_name": "Renamed", "last_name": "Self",
		"is_admin": true, "account_ids": []uint64{1},
	})
	w := httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Update auto-édition: statut = %d, want %d (body=%s)", w.Code, http.StatusOK, w.Body.String())
	}

	var updated User
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("décodage réponse: %v", err)
	}

	if updated.IsAdmin {
		t.Error("un non-admin ne doit jamais pouvoir s'auto-élever admin en éditant son propre profil")
	}
}

func TestUpdate_OtherUserRequiresAdmin(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)
	ctx := t.Context()

	nonAdmin, err := repo.Create(ctx, "nonadmin2@test.local", "Non", "Admin", nil, hashPassword(t), nil, false)
	if err != nil {
		t.Fatalf("Create nonAdmin: %v", err)
	}

	other, err := repo.Create(ctx, "other@test.local", "Other", "User", nil, hashPassword(t), nil, false)
	if err != nil {
		t.Fatalf("Create other: %v", err)
	}

	req := requestAs(http.MethodPut, "/users/"+itoa(other.ID), nonAdmin.ID, map[string]any{
		"username": other.Username, "first_name": "Hacked", "last_name": "User",
	})
	w := httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Update d'un autre utilisateur par un non-admin: statut = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestDelete_LastAdminIsProtected(t *testing.T) {
	db := newTestDB(t)
	repo := NewRepository(db)
	handler := NewHandler(repo)
	ctx := t.Context()

	admin, err := repo.Create(ctx, "onlyadmin@test.local", "Only", "Admin", nil, hashPassword(t), nil, true)
	if err != nil {
		t.Fatalf("Create admin: %v", err)
	}

	// Un second utilisateur (non-admin) pour ne pas se heurter à la garde
	// "impossible de supprimer le dernier compte" et isoler la garde
	// "dernier admin".
	if _, err := repo.Create(ctx, "second@test.local", "Second", "User", nil, hashPassword(t), nil, false); err != nil {
		t.Fatalf("Create second: %v", err)
	}

	req := requestAs(http.MethodDelete, "/users/"+itoa(admin.ID), admin.ID, nil)
	w := httptest.NewRecorder()
	handler.Delete(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Delete du dernier admin: statut = %d, want %d (body=%s)", w.Code, http.StatusConflict, w.Body.String())
	}
}

func itoa(id uint64) string {
	return strconv.FormatUint(id, 10)
}
