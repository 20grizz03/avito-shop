// internal/service/auth_test.go
package service_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/linemk/avito-shop/internal/domain/models"
	"github.com/linemk/avito-shop/internal/service"
	"github.com/linemk/avito-shop/internal/storage"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
)

// fakeUserRepo реализует интерфейс UserStorage для тестирования.
type fakeUserRepo struct {
	users map[string]*models.User
}

var _ storage.UserStorage = (*fakeUserRepo)(nil)

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: make(map[string]*models.User)}
}

func (f *fakeUserRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, ok := f.users[email]
	if !ok {
		return nil, storage.ErrUserNotFound
	}
	return user, nil
}

func (f *fakeUserRepo) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	// Присвоим ID на основе количества пользователей
	user.ID = int64(len(f.users) + 1)
	f.users[user.Email] = user
	return user, nil
}

func (f *fakeUserRepo) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, storage.ErrUserNotFound
}

func TestAuthService_Login_NewUser(t *testing.T) {
	// Настраиваем переменную окружения для JWT-секрета
	os.Setenv("JWT_SECRET", "testsecret")
	defer os.Unsetenv("JWT_SECRET")

	fakeRepo := newFakeUserRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// Создаем AuthService с TTL токена 60 минут
	authSvc := service.NewAuthService(logger, fakeRepo, 60*time.Minute)
	ctx := context.Background()

	email := "newuser@example.com"
	password := "password123"

	token, err := authSvc.Login(ctx, email, password)
	assert.NoError(t, err, "login should succeed for a new user")
	assert.NotEmpty(t, token, "token should not be empty")

	user, err := fakeRepo.GetUserByEmail(ctx, email)
	assert.NoError(t, err, "user should exist after creation")
	assert.Equal(t, 1000, user.CoinBalance, "initial coin balance should be 1000")
	// Проверяем, что пароль хэширован (не равен исходному паролю)
	assert.NotEqual(t, password, string(user.PassHash), "password should be hashed")
}

func TestAuthService_Login_ExistingUser_CorrectPassword(t *testing.T) {
	os.Setenv("JWT_SECRET", "testsecret")
	defer os.Unsetenv("JWT_SECRET")

	fakeRepo := newFakeUserRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authSvc := service.NewAuthService(logger, fakeRepo, 60*time.Minute)
	ctx := context.Background()

	email := "existing@example.com"
	password := "password123"
	// Создаем пользователя вручную с хэшированием пароля
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)
	user := &models.User{
		Email:       email,
		PassHash:    hashed,
		CoinBalance: 1000,
	}
	_, err = fakeRepo.CreateUser(ctx, user)
	assert.NoError(t, err)

	// Попытка логина с корректным паролем
	token, err := authSvc.Login(ctx, email, password)
	assert.NoError(t, err, "login should succeed with correct password")
	assert.NotEmpty(t, token, "token should be returned")
}

func TestAuthService_Login_ExistingUser_WrongPassword(t *testing.T) {
	os.Setenv("JWT_SECRET", "testsecret")
	defer os.Unsetenv("JWT_SECRET")

	fakeRepo := newFakeUserRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authSvc := service.NewAuthService(logger, fakeRepo, 60*time.Minute)
	ctx := context.Background()

	email := "existing@example.com"
	password := "password123"
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)
	user := &models.User{
		Email:       email,
		PassHash:    hashed,
		CoinBalance: 1000,
	}
	_, err = fakeRepo.CreateUser(ctx, user)
	assert.NoError(t, err)

	// Попытка логина с неверным паролем
	token, err := authSvc.Login(ctx, email, "wrongpassword")
	assert.Error(t, err, "Login should fail with incorrect password")
	assert.Empty(t, token, "Token should be empty on failed login")
}
