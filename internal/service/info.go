package service

import (
	"context"
	"fmt"

	"github.com/linemk/avito-shop/internal/storage"
	"log/slog"
)

// InfoService определяет интерфейс для получения информации о пользователе.
type InfoService interface {
	GetInfo(ctx context.Context, userID int64) (*InfoResponse, error)
}

// infoService — конкретная реализация InfoService.
type infoService struct {
	log      *slog.Logger
	userRepo storage.UserStorage
	// Здесь можно добавить дополнительные репозитории для заказов, транзакций и инвентаря.
}

func NewInfoService(log *slog.Logger, userRepo storage.UserStorage) InfoService {
	return &infoService{
		log:      log,
		userRepo: userRepo,
	}
}

// InfoResponse — структура, возвращаемая сервисом, аналогична той, что в транспортном слое.
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
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		s.log.Error("failed to get user by id", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Получаем заказы пользователя
	orders, err := s.userRepo.GetOrdersByUserID(ctx, userID)
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

	// Для упрощения примера, инвентарь и история транзакций возвращаются пустыми.
	resp := &InfoResponse{
		Coins:       user.CoinBalance,
		Inventory:   []InventoryItem{},                                               // Здесь нужно собрать информацию о купленном мерче
		CoinHistory: CoinHistory{Received: []HistoryEntry{}, Sent: []HistoryEntry{}}, // Здесь - транзакции
	}
	return resp, nil
}
