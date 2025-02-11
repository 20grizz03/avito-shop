package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/linemk/avito-shop/internal/domain/models"
)

var ErrUserNotFound = errors.New("user not found")

type UserStorage interface {
	// Получить пользователя по email
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	// Создать нового пользователя
	CreateUser(ctx context.Context, user *models.User) (*models.User, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *userRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	row := r.db.QueryRowContext(ctx, "SELECT id, email, pass_hash, coin_balance FROM users WHERE email = $1", email)
	if err := row.Scan(&user.ID, &user.Email, &user.PassHash, &user.CoinBalance); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *userRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		"INSERT INTO users (email, pass_hash, coin_balance) VALUES ($1, $2, $3) RETURNING id",
		user.Email, user.PassHash, user.CoinBalance,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	user.ID = id
	return user, nil
}
