package shipping

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const cdekAPI = "https://api.cdek.ru/v2"

type CDEKClient struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	mu           sync.Mutex
	token        string
	tokenExpiry  time.Time
}

func NewCDEKClient(clientID, clientSecret string) *CDEKClient {
	return &CDEKClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

type ShippingRate struct {
	TariffCode int    `json:"tariff_code"`
	TariffName string `json:"tariff_name"`
	PriceRub   int    `json:"price_rub"`
	MinDays    int    `json:"min_days"`
	MaxDays    int    `json:"max_days"`
}

func (c *CDEKClient) token_(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return c.token, nil
	}
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cdekAPI+"/oauth/token", bytes.NewBufferString(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	c.token = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	return c.token, nil
}

func (c *CDEKClient) CalculateRates(ctx context.Context, fromCity, toCity, weightGrams int) ([]ShippingRate, error) {
	token, err := c.token_(ctx)
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]interface{}{
		"tariff_code":   136,
		"from_location": map[string]int{"code": fromCity},
		"to_location":   map[string]int{"code": toCity},
		"packages":      []map[string]interface{}{{"weight": weightGrams, "length": 20, "width": 20, "height": 10}},
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cdekAPI+"/calculator/tariff", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		TariffCode  int     `json:"tariff_code"`
		DeliverySum float64 `json:"delivery_sum"`
		Period      struct {
			Min int `json:"min"`
			Max int `json:"max"`
		} `json:"period"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return []ShippingRate{{
		TariffCode: result.TariffCode,
		TariffName: "СДЭК",
		PriceRub:   int(result.DeliverySum),
		MinDays:    result.Period.Min,
		MaxDays:    result.Period.Max,
	}}, nil
}
