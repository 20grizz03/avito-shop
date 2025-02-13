package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/linemk/avito-shop/internal/storage"
)

// InfoService определяет интерфейс для получения информации о пользователе.
type InfoService interface {
	GetInfo(ctx context.Context, userID int64) (*InfoResponse, error)
}

// infoService — конкретная реализация InfoService.
type infoService struct {
	log        *slog.Logger
	userRepo   storage.UserStorage
	orderRepo  storage.OrderStorage
	coinTxRepo storage.CoinTransactionStorage
}

func NewInfoService(log *slog.Logger, userRepo storage.UserStorage, orderRepo storage.OrderStorage, coinTxRepo storage.CoinTransactionStorage) InfoService {
	return &infoService{
		log:        log,
		userRepo:   userRepo,
		orderRepo:  orderRepo,
		coinTxRepo: coinTxRepo,
	}
}

// InfoResponse — структура, возвращаемая сервисом, аналогична той, что в транспортном слое
type InfoResponse struct {
	Coins       int             `json:"coins"`
	Inventory   []InventoryItem `json:"inventory"`
	CoinHistory CoinHistory     `json:"coinHistory"`
}

type InventoryItem struct {
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
}

type CoinHistory struct {
	Received []HistoryEntry `json:"received"`
	Sent     []HistoryEntry `json:"sent"`
}

type HistoryEntry struct {
	FromUser string `json:"fromUser,omitempty"`
	ToUser   string `json:"toUser,omitempty"`
	Amount   int    `json:"amount"`
}

// GetInfo собирает информацию о пользователе, например, баланс, инвентарь и историю транзакций.
// Здесь для примера мы просто возвращаем баланс из таблицы пользователей. В реальной реализации
// необходимо обращаться к соответствующим репозиториям для инвентаря и транзакций.
func (s *infoService) GetInfo(ctx context.Context, userID int64) (*InfoResponse, error) {
	const op = "service.InfoService.GetInfo"
	s.log.Info("getting info", slog.String("op", op), slog.Int64("userID", userID))

	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.log.Error("failed to get user by id", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Получаем заказы пользователя
	orders, err := s.orderRepo.GetOrdersByUserID(ctx, userID)
	if err != nil {
		s.log.Error("failed to get orders", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	// Группируем заказы по типу мерча
	inventoryMap := make(map[string]int)
	for _, order := range orders {
		inventoryMap[order.MerchName] += order.Quantity
	}

	// Преобразуем результат в массив InventoryItem
	var inventory []InventoryItem
	for merch, quantity := range inventoryMap {
		inventory = append(inventory, InventoryItem{
			Type:     merch,
			Quantity: quantity,
		})
	}

	// Получаем историю транзакций пользователя через CoinTransactionStorage
	transactions, err := s.coinTxRepo.GetTransactionsByUserID(ctx, userID)
	var received []HistoryEntry
	var sent []HistoryEntry
	if err != nil {
		s.log.Error("failed to get coin transactions", slog.Any("error", err))
		// Если ошибка получения транзакций, можно продолжить с пустой историей
	} else {
		for _, tx := range transactions {
			switch tx.Type {
			case "transfer_received":
				received = append(received, HistoryEntry{
					FromUser: "", // при необходимости можно получить имя отправителя
					Amount:   tx.Amount,
				})
			case "transfer_sent":
				sent = append(sent, HistoryEntry{
					ToUser: "",        // при необходимости можно получить имя получателя
					Amount: tx.Amount, // предполагаем, что tx.Amount хранится как положительное значение, если нет, можно умножить на -1
				})
			}
		}
	}

	// Для упрощения примера, инвентарь и история транзакций возвращаются пустыми.
	resp := &InfoResponse{
		Coins:     user.CoinBalance,
		Inventory: inventory,
		// TODO Здесь нужно собрать информацию об историях перевода
		CoinHistory: CoinHistory{Received: received, Sent: sent}, // Здесь - транзакции
	}
	return resp, nil
}
