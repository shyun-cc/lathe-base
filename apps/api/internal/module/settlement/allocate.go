package settlement

import "errors"

type Direction string

const (
	Receivable Direction = "receivable"
	Payable    Direction = "payable"
)

type Account struct {
	PartyID   int64
	Currency  string
	Direction Direction
}

// Entry represents a confirmed statement or a posted cash entry. In a receivable
// account the cash entry is a receipt. In a payable account it is a disbursement.
// Remaining is expressed in the currency's minor units.
type Entry struct {
	Account
	Remaining int64
}

type Balances struct {
	StatementRemaining int64
	CashRemaining      int64
}

// Allocate calculates the balances after applying a positive amount from a cash
// entry to a statement. The caller owns the transaction and persists the allocation.
func Allocate(statement, cash Entry, amount int64) (Balances, error) {
	account := statement.Account
	if account.PartyID <= 0 || !isCurrencyCode(account.Currency) ||
		(account.Direction != Receivable && account.Direction != Payable) {
		return Balances{}, errors.New("invalid statement account")
	}
	if cash.Account != account {
		return Balances{}, errors.New("statement and cash entry must share a party, currency and direction")
	}
	if amount <= 0 || amount > statement.Remaining || amount > cash.Remaining {
		return Balances{}, errors.New("allocation must be positive and within both available balances")
	}
	return Balances{
		StatementRemaining: statement.Remaining - amount,
		CashRemaining:      cash.Remaining - amount,
	}, nil
}

func isCurrencyCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, letter := range code {
		if letter < 'A' || letter > 'Z' {
			return false
		}
	}
	return true
}
