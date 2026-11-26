package settlement

import "testing"

func TestAllocate(t *testing.T) {
	for _, direction := range []Direction{Receivable, Payable} {
		t.Run(string(direction), func(t *testing.T) {
			account := Account{PartyID: 1, Currency: "USD", Direction: direction}
			statement := Entry{Account: account, Remaining: 900}
			cash := Entry{Account: account, Remaining: 1_100}
			partial, err := Allocate(statement, cash, 400)
			if err != nil || partial != (Balances{500, 700}) {
				t.Fatalf("partial allocation = %+v, %v", partial, err)
			}

			statement.Remaining, cash.Remaining = partial.StatementRemaining, partial.CashRemaining
			full, err := Allocate(statement, cash, 500)
			if err != nil || full != (Balances{0, 200}) {
				t.Fatalf("full allocation = %+v, %v", full, err)
			}
		})
	}
}

func TestAllocateRejectsInvalidAmountsAndAccounts(t *testing.T) {
	account := Account{PartyID: 1, Currency: "USD", Direction: Receivable}
	statement := Entry{Account: account, Remaining: 900}
	cash := Entry{Account: account, Remaining: 600}
	for _, amount := range []int64{-1, 0, 601} {
		if _, err := Allocate(statement, cash, amount); err == nil {
			t.Errorf("Allocate accepted amount %d", amount)
		}
	}
	cash.Remaining = 1_000
	if _, err := Allocate(statement, cash, 901); err == nil {
		t.Error("Allocate accepted a statement over-allocation")
	}
	for _, mismatched := range []Account{
		{PartyID: 2, Currency: "USD", Direction: Receivable},
		{PartyID: 1, Currency: "EUR", Direction: Receivable},
		{PartyID: 1, Currency: "USD", Direction: Payable},
	} {
		cash.Account = mismatched
		if _, err := Allocate(statement, cash, 100); err == nil {
			t.Errorf("Allocate accepted account %+v", mismatched)
		}
	}
}
