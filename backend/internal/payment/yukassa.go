package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const yukassaAPI = "https://api.yookassa.ru/v3"

type YukassaClient struct {
	shopID    string
	secretKey string
	client    *http.Client
}

func NewYukassaClient(shopID, secretKey string) *YukassaClient {
	return &YukassaClient{shopID: shopID, secretKey: secretKey, client: &http.Client{}}
}

type PaymentResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Confirmation struct {
		ConfirmationURL string `json:"confirmation_url"`
	} `json:"confirmation"`
}

func (c *YukassaClient) CreatePayment(ctx context.Context, amountKopecks int, orderID, returnURL, idempotencyKey string) (*PaymentResponse, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"amount": map[string]string{
			"value":    fmt.Sprintf("%.2f", float64(amountKopecks)/100),
			"currency": "RUB",
		},
		"confirmation": map[string]string{
			"type":       "redirect",
			"return_url": returnURL,
		},
		"capture":     true,
		"description": fmt.Sprintf("Заказ #%s", orderID[:8]),
		"metadata":    map[string]string{"order_id": orderID},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, yukassaAPI+"/payments", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.shopID, c.secretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", idempotencyKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yukassa error: %s", resp.Status)
	}

	var p PaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}
