package creditcard

// CreateCardInput represents the input for creating a new card
type CreateCardInput struct {
	CardNumber  string `json:"card_number"`
	ExpiryMonth string `json:"expiry_month"`
	ExpiryYear  string `json:"expiry_year"`
}

// TokenizedCard represents a tokenized credit card
type TokenizedCard struct {
	Token    string
	CardType string
	LastFour string
	IssuedBy string
}
