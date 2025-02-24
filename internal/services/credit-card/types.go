package creditcard

type CreateCardInput struct {
	CardNumber  string `json:"card_number"`
	ExpiryMonth string `json:"expiry_month"`
	ExpiryYear  string `json:"expiry_year"`
	CVV         string `json:"cvv"`
}

type TokenizedCard struct {
	Token    string
	CardType string
	LastFour string
	IssuedBy string
}
