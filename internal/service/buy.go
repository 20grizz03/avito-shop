// internal/service/buy.go
package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/linemk/avito-shop/internal/storage"
	"log/slog"
)

type BuyService interface {
	Buy(ctx context.Context, userID int64, item string) error
}

type buyService struct {
	log       *slog.Logger
	userRepo  storage.UserStorage
	merchRepo storage.MerchStorage
	orderRepo storage.OrderStorage
	db        *sql.DB
}

func NewBuyService(log *slog.Logger, db *sql.DB, userRepo storage.UserStorage, merchRepo storage.MerchStorage, orderRepo storage.OrderStorage) BuyService {
	return &buyService{
		log:       log,
		db:        db,
		userRepo:  userRepo,
		merchRepo: merchRepo,
		orderRepo: orderRepo,
	}
}

// Buy осуществляет покупку товара:
// 1. Запускается транзакция.
// 2. Получается мерч по названию.
// 3. Получается пользователь.
// 4. Проверяется, достаточно ли средств у пользователя.
// 5. Обновляется баланс пользователя.
// 6. Создается заказ.
// Если что-то идет не так, транзакция откатывается.
func (s *buyService) Buy(ctx context.Context, userID int64, item string) error {
	const op = "service.BuyService.Buy"
	logger := s.log.With(slog.String("op", op), slog.Int64("userID", userID), slog.String("item", item))
	logger.Info("starting purchase transaction")

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", slog.Any("error", err))
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}

	// Получаем мерч по названию
	merch, err := s.merchRepo.GetMerchByName(ctx, tx, item)
	if err != nil {
		tx.Rollback()
		logger.Error("failed to get merch", slog.Any("error", err))
		return fmt.Errorf("%s: failed to get merch: %w", op, err)
	}

	// Получаем пользователя
	user, err := s.userRepo.GetUserByIDtx(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		logger.Error("failed to get user", slog.Any("error", err))
		return fmt.Errorf("%s: failed to get user: %w", op, err)
	}

	// Проверяем, достаточно ли средств
	if user.CoinBalance < merch.Price {
		tx.Rollback()
		logger.Warn("insufficient funds", slog.Int("balance", user.CoinBalance), slog.Int("price", merch.Price))
		return fmt.Errorf("%s: insufficient funds", op)
	}

	// Обновляем баланс пользователя
	newBalance := user.CoinBalance - merch.Price
	if err := s.userRepo.UpdateUserBalance(ctx, tx, userID, newBalance); err != nil {
		tx.Rollback()
		logger.Error("failed to update user balance", slog.Any("error", err))
		return fmt.Errorf("%s: failed to update user balance: %w", op, err)
	}

	// Создаем заказ
	if err := s.orderRepo.CreateOrder(ctx, tx, userID, merch.ID, 1, merch.Price); err != nil {
		tx.Rollback()
		logger.Error("failed to create order", slog.Any("error", err))
		return fmt.Errorf("%s: failed to create order: %w", op, err)
	}

	// Коммит транзакции
	if err := tx.Commit(); err != nil {
		logger.Error("failed to commit transaction", slog.Any("error", err))
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	logger.Info("purchase completed successfully")
	return nil
}
