package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/linemk/avito-shop/internal/jwtNew/jwtmiddleware"
	"github.com/linemk/avito-shop/internal/service"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/linemk/avito-shop/internal/app/handlers"
	"github.com/stretchr/testify/assert"
	"log/slog"
)

// fakeAuthService — фиктивная реализация для тестирования.
type fakeAuthService struct {
	token string
	err   error
}

func (f *fakeAuthService) Login(ctx context.Context, username, password string) (string, error) {
	return f.token, f.err
}

type fakeInfoService struct {
	resp *service.InfoResponse
	err  error
}

func (f *fakeInfoService) GetInfo(ctx context.Context, userID int64) (*service.InfoResponse, error) {
	return f.resp, f.err
}

func TestAuthHandler_Success(t *testing.T) {
	// Фиктивный сервис возвращает корректный токен.
	fakeSvc := &fakeAuthService{token: "test-token", err: nil}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := handlers.AuthHandler(logger, fakeSvc)

	reqBody := `{"username": "test@example.com", "password": "password123"}`
	req := httptest.NewRequest("POST", "/api/auth", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code, "Expected status 200 OK")

	var resp struct {
		Token string `json:"token"`
	}
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err, "Response decoding should succeed")
	assert.Equal(t, "test-token", resp.Token, "Returned token should match fake token")
}

func TestAuthHandler_InvalidJSON(t *testing.T) {
	fakeSvc := &fakeAuthService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := handlers.AuthHandler(logger, fakeSvc)

	reqBody := `{"username": "test@example.com", "password":`
	req := httptest.NewRequest("POST", "/api/auth", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected status 400 for invalid JSON")
}

func TestAuthHandler_ValidationError(t *testing.T) {
	fakeSvc := &fakeAuthService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := handlers.AuthHandler(logger, fakeSvc)

	reqBody := `{"username": "test@example.com", "password": "short"}`
	req := httptest.NewRequest("POST", "/api/auth", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code, "Expected status 400 for validation error")
}

func TestAuthHandler_LoginError(t *testing.T) {
	fakeSvc := &fakeAuthService{token: "", err: assert.AnError}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := handlers.AuthHandler(logger, fakeSvc)

	reqBody := `{"username": "test@example.com", "password": "password123"}`
	req := httptest.NewRequest("POST", "/api/auth", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code, "Expected status 401 for login error")
}

func TestInfoHandler_Success(t *testing.T) {
	// Подготовка фиктивного ответа от сервиса.
	fakeResp := &service.InfoResponse{
		Coins: 920,
		Inventory: []service.InventoryItem{
			{Type: "t-shirt", Quantity: 2},
		},
		CoinHistory: service.CoinHistory{
			Received: []service.HistoryEntry{
				{FromUser: "userA", Amount: 80},
			},
			Sent: []service.HistoryEntry{
				{ToUser: "userB", Amount: 80},
			},
		},
	}
	fakeSvc := &fakeInfoService{resp: fakeResp, err: nil}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := handlers.InfoHandler(logger, fakeSvc)

	// Создаем HTTP-запрос.
	req := httptest.NewRequest("GET", "/api/info", nil)
	req.Header.Set("Content-Type", "application/json")

	// Эмулируем наличие userID в контексте через jwtmiddleware.
	// Предполагаем, что jwtmiddleware.FromContext ищет значение по ключу jwtmiddleware.UserIDKey.
	ctx := context.WithValue(req.Context(), jwtmiddleware.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Проверяем, что статус 200 OK.
	assert.Equal(t, http.StatusOK, rr.Code, "Expected status 200 OK")

	// Декодируем и проверяем ответ.
	var resp service.InfoResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err, "Response decoding should succeed")
	assert.Equal(t, fakeResp.Coins, resp.Coins, "Coins should match")
	assert.Len(t, resp.Inventory, 1, "Expected one inventory item")
	assert.Equal(t, "t-shirt", resp.Inventory[0].Type, "Inventory item type should be 't-shirt'")
	assert.Equal(t, 2, resp.Inventory[0].Quantity, "Inventory quantity should be 2")
	assert.Len(t, resp.CoinHistory.Received, 1, "Expected one received transaction")
	assert.Len(t, resp.CoinHistory.Sent, 1, "Expected one sent transaction")
}

func TestInfoHandler_Unauthorized(t *testing.T) {
	// Если в контексте нет userID, должен вернуть 401.
	fakeSvc := &fakeInfoService{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := handlers.InfoHandler(logger, fakeSvc)

	req := httptest.NewRequest("GET", "/api/info", nil)
	req.Header.Set("Content-Type", "application/json")
	// Не добавляем userID в контекст.
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code, "Expected status 401 when userID is missing")
}

func TestInfoHandler_ServiceError(t *testing.T) {
	// Если сервис возвращает ошибку, обработчик должен вернуть 500.
	fakeSvc := &fakeInfoService{
		resp: nil,
		err:  assert.AnError,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := handlers.InfoHandler(logger, fakeSvc)

	req := httptest.NewRequest("GET", "/api/info", nil)
	req.Header.Set("Content-Type", "application/json")
	// Эмулируем наличие userID в контексте.
	ctx := context.WithValue(req.Context(), jwtmiddleware.UserIDKey, int64(1))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code, "Expected status 500 when service returns error")
}
