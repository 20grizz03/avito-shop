package service_test

import (
	"context"
	"database/sql"
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

type fakeUserRepo struct {
	users map[string]*models.User // ключ — email
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

func (f *fakeUserRepo) GetUserByIDtx(ctx context.Context, tx *sql.Tx, id int64) (*models.User, error) {
	return f.GetUserByID(ctx, id)
}

func (f *fakeUserRepo) UpdateUserBalance(ctx context.Context, tx *sql.Tx, id int64, newBalance int) error {
	for _, u := range f.users {
		if u.ID == id {
			u.CoinBalance = newBalance
			return nil
		}
	}
	return storage.ErrUserNotFound
}

type fakeOrderRepo struct {
	orders map[int64][]*models.Order // ключ: userID
}

var _ storage.OrderStorage = (*fakeOrderRepo)(nil)

func newFakeOrderRepo() *fakeOrderRepo {
	return &fakeOrderRepo{orders: make(map[int64][]*models.Order)}
}

func (f *fakeOrderRepo) GetOrdersByUserID(ctx context.Context, userID int64) ([]*models.Order, error) {
	if orders, ok := f.orders[userID]; ok {
		return orders, nil
	}
	return []*models.Order{}, nil
}

func (f *fakeOrderRepo) CreateOrder(ctx context.Context, tx *sql.Tx, userID int64, merchID int64, quantity int, totalPrice int) error {
	// Не требуется для теста InfoService
	return nil
}

type fakeCoinTxRepo struct {
	transactions map[int64][]*models.CoinTransaction // ключ: userID
}

var _ storage.CoinTransactionStorage = (*fakeCoinTxRepo)(nil)

func newFakeCoinTxRepo() *fakeCoinTxRepo {
	return &fakeCoinTxRepo{transactions: make(map[int64][]*models.CoinTransaction)}
}

func (f *fakeCoinTxRepo) GetTransactionsByUserID(ctx context.Context, userID int64) ([]*models.CoinTransaction, error) {
	if txs, ok := f.transactions[userID]; ok {
		return txs, nil
	}
	return []*models.CoinTransaction{}, nil
}

func (f *fakeCoinTxRepo) CreateTransaction(ctx context.Context, tx *sql.Tx, userID int64, amount int, txType string, relatedUserID *int64) error {
	// Не требуется для теста InfoService
	return nil
}

func TestAuthService_Login_NewUser(t *testing.T) {
	os.Setenv("JWT_SECRET", "testsecret")
	defer os.Unsetenv("JWT_SECRET")

	fakeRepo := newFakeUserRepo()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authSvc := service.NewAuthService(logger, fakeRepo, 60*time.Minute)
	ctx := context.Background()

	email := "newuser@example.com"
	password := "password123"

	token, err := authSvc.Login(ctx, email, password)
	assert.NoError(t, err, "Login should succeed for a new user")
	assert.NotEmpty(t, token, "Token should not be empty")

	user, err := fakeRepo.GetUserByEmail(ctx, email)
	assert.NoError(t, err, "User should exist after creation")
	assert.Equal(t, 1000, user.CoinBalance, "Initial coin balance should be 1000")
	// Проверяем, что пароль хэширован (не равен исходному паролю)
	assert.NotEqual(t, password, string(user.PassHash), "Password should be hashed")
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
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)
	user := &models.User{
		Email:       email,
		PassHash:    hashed,
		CoinBalance: 1000,
	}
	_, err = fakeRepo.CreateUser(ctx, user)
	assert.NoError(t, err)

	token, err := authSvc.Login(ctx, email, password)
	assert.NoError(t, err, "Login should succeed with correct password")
	assert.NotEmpty(t, token, "Token should be returned")
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

	token, err := authSvc.Login(ctx, email, "wrongpassword")
	assert.Error(t, err, "Login should fail with incorrect password")
	assert.Empty(t, token, "Token should be empty on failed login")
}

// ================= Тесты для InfoService =================

func TestInfoService_GetInfo_Success(t *testing.T) {
	// Создаем фиктивные репозитории.
	userRepo := newFakeUserRepo()
	orderRepo := newFakeOrderRepo()
	coinTxRepo := newFakeCoinTxRepo()

	// Добавляем пользователя с балансом 920, ID=1.
	user := &models.User{
		ID:          1,
		Email:       "test@example.com",
		PassHash:    []byte("hashed-password"),
		CoinBalance: 920,
	}

	userRepo.users[user.Email] = user

	// Добавляем пару заказов для пользователя (например, куплена футболка: 2 единицы)
	orderRepo.orders[user.ID] = []*models.Order{
		{
			ID:         1,
			UserID:     user.ID,
			MerchID:    1,
			MerchName:  "t-shirt",
			Quantity:   1,
			TotalPrice: 80,
			CreatedAt:  time.Now().Add(-time.Hour),
		},
		{
			ID:         2,
			UserID:     user.ID,
			MerchID:    1,
			MerchName:  "t-shirt",
			Quantity:   1,
			TotalPrice: 80,
			CreatedAt:  time.Now().Add(-30 * time.Minute),
		},
	}

	// Добавляем транзакции для пользователя
	coinTxRepo.transactions[user.ID] = []*models.CoinTransaction{
		{
			ID:            1,
			UserID:        user.ID,
			Amount:        80,
			Type:          "transfer_received",
			RelatedUserID: nil,
			CreatedAt:     time.Now().Add(-45 * time.Minute),
		},
		{
			ID:            2,
			UserID:        user.ID,
			Amount:        80,
			Type:          "transfer_sent",
			RelatedUserID: nil,
			CreatedAt:     time.Now().Add(-50 * time.Minute),
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	infoSvc := service.NewInfoService(logger, userRepo, orderRepo, coinTxRepo)

	ctx := context.Background()
	infoResp, err := infoSvc.GetInfo(ctx, user.ID)
	assert.NoError(t, err, "GetInfo should succeed")
	assert.Equal(t, 920, infoResp.Coins, "User coin balance should match")

	// Проверяем инвентарь: два заказа футболок должны сгруппироваться в один элемент с количеством 2.
	assert.Len(t, infoResp.Inventory, 1, "There should be one inventory item")
	if len(infoResp.Inventory) > 0 {
		assert.Equal(t, "t-shirt", infoResp.Inventory[0].Type, "Inventory item type should be t-shirt")
		assert.Equal(t, 2, infoResp.Inventory[0].Quantity, "Inventory quantity should be 2")
	}

	// Проверяем транзакции: ожидаем одну запись для каждого типа
	assert.Len(t, infoResp.CoinHistory.Received, 1, "There should be one received transaction")
	assert.Len(t, infoResp.CoinHistory.Sent, 1, "There should be one sent transaction")
}

func TestInfoService_GetInfo_UserNotFound(t *testing.T) {
	userRepo := newFakeUserRepo()
	orderRepo := newFakeOrderRepo()
	coinTxRepo := newFakeCoinTxRepo()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	infoSvc := service.NewInfoService(logger, userRepo, orderRepo, coinTxRepo)

	ctx := context.Background()
	_, err := infoSvc.GetInfo(ctx, 999) // Пользователь с таким ID не существует
	assert.Error(t, err, "Expected error for non-existing user")
}
