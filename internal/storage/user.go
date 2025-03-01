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
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
	GetUserByIDtx(ctx context.Context, tx *sql.Tx, id int64) (*models.User, error)
	UpdateUserBalance(ctx context.Context, tx *sql.Tx, id int64, newBalance int) error
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *userRepository {
	return &userRepository{db: db}
}

// получение уже существующего пользователя
func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	row := r.db.QueryRowContext(ctx, "SELECT id, username, pass_hash, coin_balance FROM users WHERE username = $1", email)
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
		"INSERT INTO users (username, pass_hash, coin_balance) VALUES ($1, $2, $3) RETURNING id",
		user.Email, user.PassHash, user.CoinBalance,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	user.ID = id
	return user, nil
}

func (r *userRepository) UpdateUserBalance(ctx context.Context, tx *sql.Tx, id int64, newBalance int) error {
	res, err := tx.ExecContext(ctx, "UPDATE users SET coin_balance = $1 WHERE id = $2", newBalance, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *userRepository) GetUserByIDtx(ctx context.Context, tx *sql.Tx, id int64) (*models.User, error) {
	user := &models.User{}
	row := tx.QueryRowContext(ctx, "SELECT id, username, pass_hash, coin_balance FROM users WHERE id = $1", id)
	if err := row.Scan(&user.ID, &user.Email, &user.PassHash, &user.CoinBalance); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}
