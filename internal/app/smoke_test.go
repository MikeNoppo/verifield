package app_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"verifield-be/internal/common/apperror"
	"verifield-be/internal/common/middleware"
	"verifield-be/internal/common/pagination"
	"verifield-be/internal/common/response"
	"verifield-be/internal/common/validation"
	"verifield-be/internal/modules/auth"
	"verifield-be/internal/modules/user"
	userdto "verifield-be/internal/modules/user/dto"
	"verifield-be/internal/schema"
	"verifield-be/internal/shared/hash"
	"verifield-be/internal/shared/jwtx"
)

// stubUserService menggantikan modul user tanpa database.
type stubUserService struct{ user *schema.User }

func (s *stubUserService) Create(context.Context, userdto.CreateUserDTO) (*userdto.UserResponse, error) {
	res := userdto.ToUserResponse(s.user)
	return &res, nil
}
func (s *stubUserService) FindAll(context.Context, pagination.Query) ([]userdto.UserResponse, response.Meta, error) {
	return nil, response.Meta{}, nil
}
func (s *stubUserService) FindByID(_ context.Context, id string) (*userdto.UserResponse, error) {
	if id != s.user.ID.String() {
		return nil, apperror.NotFound("User tidak ditemukan")
	}
	res := userdto.ToUserResponse(s.user)
	return &res, nil
}
func (s *stubUserService) Update(context.Context, string, userdto.UpdateUserDTO) (*userdto.UserResponse, error) {
	return nil, nil
}
func (s *stubUserService) Remove(context.Context, string) error { return nil }
func (s *stubUserService) CreateEntity(context.Context, userdto.CreateUserDTO) (*schema.User, error) {
	return s.user, nil
}
func (s *stubUserService) FindEntityByEmail(_ context.Context, email string) (*schema.User, error) {
	if !strings.EqualFold(email, s.user.Email) {
		return nil, apperror.NotFound("User tidak ditemukan")
	}
	return s.user, nil
}
func (s *stubUserService) FindEntityByID(_ context.Context, id string) (*schema.User, error) {
	if id != s.user.ID.String() {
		return nil, apperror.NotFound("User tidak ditemukan")
	}
	return s.user, nil
}

var _ user.Service = (*stubUserService)(nil)

func newTestEngine(t *testing.T) (*gin.Engine, *schema.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	validation.Init()

	hashed, err := hash.Password("rahasia123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	testUser := &schema.User{
		ID:       uuid.New(),
		Name:     "Siti Rahma",
		Email:    "siti@verifield.id",
		Password: hashed,
		Role:     schema.RoleClient,
		IsActive: true,
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	manager := jwtx.NewManager("secret-uji-yang-cukup-panjang-32ch", "verifield-be", 15*time.Minute, time.Hour)

	engine := gin.New()
	engine.Use(
		middleware.RequestID(),
		middleware.Recovery(log),
		middleware.ErrorHandler(log),
	)

	authGuard := middleware.JWTAuth(manager)
	adminOnly := middleware.RequireRoles(string(schema.RoleAdmin))

	api := engine.Group("/api/v1")
	auth.NewModule(manager, &stubUserService{user: testUser}).RegisterRoutes(api, authGuard)
	api.GET("/admin-only", authGuard, adminOnly, func(c *gin.Context) {
		response.OK(c, "ok", nil)
	})

	engine.NoRoute(func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "Endpoint tidak ditemukan", nil)
	})

	return engine, testUser
}

func do(t *testing.T, engine *gin.Engine, method, path, body, token string) (int, response.Envelope) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	var envelope response.Envelope
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("response bukan envelope JSON: %v (body=%s)", err, rec.Body.String())
		}
	}
	return rec.Code, envelope
}

func login(t *testing.T, engine *gin.Engine) string {
	t.Helper()

	status, env := do(t, engine, http.MethodPost, "/api/v1/auth/login",
		`{"email":"siti@verifield.id","password":"rahasia123"}`, "")
	if status != http.StatusOK {
		t.Fatalf("login gagal: status=%d body=%+v", status, env)
	}

	data, _ := env.Data.(map[string]any)
	token, _ := data["token"].(map[string]any)
	accessToken, _ := token["access_token"].(string)
	if accessToken == "" {
		t.Fatalf("access_token kosong: %+v", env.Data)
	}
	return accessToken
}

func TestLoginSukses(t *testing.T) {
	engine, _ := newTestEngine(t)
	if token := login(t, engine); token == "" {
		t.Fatal("token kosong")
	}
}

func TestLoginPasswordSalah(t *testing.T) {
	engine, _ := newTestEngine(t)

	status, env := do(t, engine, http.MethodPost, "/api/v1/auth/login",
		`{"email":"siti@verifield.id","password":"passwordsalah"}`, "")

	if status != http.StatusUnauthorized {
		t.Fatalf("mau 401, dapat %d", status)
	}
	if env.Success || env.Code != "UNAUTHORIZED" {
		t.Fatalf("envelope error tidak sesuai: %+v", env)
	}
}

func TestValidasiDTOGagal(t *testing.T) {
	engine, _ := newTestEngine(t)

	status, env := do(t, engine, http.MethodPost, "/api/v1/auth/register",
		`{"name":"Si","email":"bukan-email","password":"123"}`, "")

	if status != http.StatusUnprocessableEntity {
		t.Fatalf("mau 422, dapat %d", status)
	}
	if len(env.Errors) != 3 {
		t.Fatalf("mau 3 field error, dapat %d: %+v", len(env.Errors), env.Errors)
	}
	for _, fieldErr := range env.Errors {
		switch fieldErr.Field {
		case "name", "email", "password":
		default:
			t.Fatalf("nama field tidak memakai nama JSON: %q", fieldErr.Field)
		}
	}
}

func TestMeTanpaToken(t *testing.T) {
	engine, _ := newTestEngine(t)

	status, env := do(t, engine, http.MethodGet, "/api/v1/auth/me", "", "")

	if status != http.StatusUnauthorized {
		t.Fatalf("mau 401, dapat %d", status)
	}
	if env.Success {
		t.Fatalf("mau success=false, dapat %+v", env)
	}
}

func TestMeDenganToken(t *testing.T) {
	engine, testUser := newTestEngine(t)
	token := login(t, engine)

	status, env := do(t, engine, http.MethodGet, "/api/v1/auth/me", "", token)

	if status != http.StatusOK {
		t.Fatalf("mau 200, dapat %d (%+v)", status, env)
	}

	data, _ := env.Data.(map[string]any)
	if data["email"] != testUser.Email {
		t.Fatalf("email tidak cocok: %+v", data)
	}
	if _, leaked := data["password"]; leaked {
		t.Fatal("password bocor ke response")
	}
}

func TestRoleGuardMenolakNonAdmin(t *testing.T) {
	engine, _ := newTestEngine(t)
	token := login(t, engine)

	status, env := do(t, engine, http.MethodGet, "/api/v1/admin-only", "", token)

	if status != http.StatusForbidden {
		t.Fatalf("mau 403, dapat %d", status)
	}
	if env.Code != "FORBIDDEN" {
		t.Fatalf("mau code FORBIDDEN, dapat %q", env.Code)
	}
}

func TestRouteTidakDikenal(t *testing.T) {
	engine, _ := newTestEngine(t)

	status, env := do(t, engine, http.MethodGet, "/api/v1/tidak-ada", "", "")

	if status != http.StatusNotFound || env.Success {
		t.Fatalf("mau 404 envelope error, dapat %d %+v", status, env)
	}
}
