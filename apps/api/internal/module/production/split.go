package production

import "errors"

// Quantities are integers in thousandths of the sales line's unit.
type OrderLine struct {
	ID       int64
	Ordered  int64
	Assigned int64
}

type WorkOrderDraft struct {
	OrderLineID int64
	Quantity    int64
}

// Split plans positive work order quantities against the unassigned balance of one
// sales line. A partial split leaves the remaining quantity available for later work.
func Split(line OrderLine, batches []int64) ([]WorkOrderDraft, error) {
	if line.ID <= 0 || line.Ordered <= 0 || line.Assigned < 0 || line.Assigned > line.Ordered {
		return nil, errors.New("invalid sales line quantities or identifier")
	}
	if len(batches) == 0 {
		return nil, errors.New("at least one batch is required")
	}
	remaining := line.Ordered - line.Assigned
	drafts := make([]WorkOrderDraft, 0, len(batches))
	for _, quantity := range batches {
		if quantity <= 0 {
			return nil, errors.New("batch quantity must be positive")
		}
		if quantity > remaining {
			return nil, errors.New("batch quantity exceeds the unassigned quantity")
		}
		drafts = append(drafts, WorkOrderDraft{OrderLineID: line.ID, Quantity: quantity})
		remaining -= quantity
	}
	return drafts, nil
}
