// Package events defines the messages that services publish and consume.
// Producers and consumers both import these types, so the contract lives in one place.
package events

// RoutingKeyAccountCreated is the routing key of the account.created event.
const RoutingKeyAccountCreated = "account.created"

// AccountType tells whether an account buys from stores or runs a store.
type AccountType string

const (
	// AccountTypeSeller is an account that runs a store.
	AccountTypeSeller AccountType = "seller"
	// AccountTypeShopper is an account that buys from stores.
	AccountTypeShopper AccountType = "shopper"
)

// Valid reports whether t is one of the known account types.
func (t AccountType) Valid() bool {
	return t == AccountTypeSeller || t == AccountTypeShopper
}

// AccountCreated is the payload of the account.created event.
// It deliberately carries no email or other personal data.
type AccountCreated struct {
	AccountID   string      `json:"account_id"`
	AccountType AccountType `json:"account_type"`
}
