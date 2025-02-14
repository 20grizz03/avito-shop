package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const baseURL = "http://localhost:8080"

// AuthResponse структура ответа при аутентификации
type AuthResponse struct {
	Token string `json:"token"`
}

// SendCoinRequest структура запроса на отправку монет
type SendCoinRequest struct {
	ToUser string `json:"toUser"`
	Amount int    `json:"amount"`
}

var Token string

func TestAuth(t *testing.T) {
	requestBody := []byte(`{"username": "testuser@gmail.com", "password": "testpass"}`)
	resp, err := http.Post(baseURL+"/api/auth", "application/json", bytes.NewBuffer(requestBody))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var authResp AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	assert.NoError(t, err)
	assert.NotEmpty(t, authResp.Token)

	Token = authResp.Token
}

func TestAuthInvalid(t *testing.T) {
	requestBody := []byte(`{"username": "", "password": ""}`)
	resp, err := http.Post(baseURL+"/api/auth", "application/json", bytes.NewBuffer(requestBody))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestGetInfo(t *testing.T) {
	token := Token
	req, err := http.NewRequest("GET", baseURL+"/api/info", nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetInfoUnauthorized(t *testing.T) {
	req, err := http.NewRequest("GET", baseURL+"/api/info", nil)
	assert.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestSendCoin(t *testing.T) {
	requestAuth := []byte(`{"username": "testuser2@gmail.com", "password": "testpass"}`)
	_, err := http.Post(baseURL+"/api/auth", "application/json", bytes.NewBuffer(requestAuth))

	token := Token
	requestBody := SendCoinRequest{ToUser: "testuser2@gmail.com", Amount: 10}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", baseURL+"/api/sendCoin", bytes.NewBuffer(jsonBody))
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSendCoinInvalid(t *testing.T) {
	token := Token
	requestBody := SendCoinRequest{ToUser: "", Amount: -5}
	jsonBody, _ := json.Marshal(requestBody)

	req, err := http.NewRequest("POST", baseURL+"/api/sendCoin", bytes.NewBuffer(jsonBody))
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestBuyItem(t *testing.T) {
	token := Token
	item := "t-shirt"
	req, err := http.NewRequest("GET", baseURL+"/api/buy/"+item, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBuyItemNotFound(t *testing.T) {
	token := Token
	item := "nonexistent_item"
	req, err := http.NewRequest("GET", baseURL+"/api/buy/"+item, nil)
	assert.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
