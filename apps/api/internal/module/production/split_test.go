package production

import (
	"math"
	"reflect"
	"testing"
)

func TestSplit(t *testing.T) {
	line := OrderLine{ID: 12, Ordered: 10_000, Assigned: 3_000}
	drafts, err := Split(line, []int64{2_000, 5_000})
	want := []WorkOrderDraft{{OrderLineID: 12, Quantity: 2_000}, {OrderLineID: 12, Quantity: 5_000}}
	if err != nil || !reflect.DeepEqual(drafts, want) {
		t.Fatalf("Split = %#v, %v; want %#v", drafts, err, want)
	}
	if _, err := Split(line, []int64{1_000}); err != nil {
		t.Fatalf("partial split: %v", err)
	}
}

func TestSplitRejectsInvalidBatchesWithoutDrafts(t *testing.T) {
	line := OrderLine{ID: 1, Ordered: math.MaxInt64}
	for _, batches := range [][]int64{nil, {1, 0}, {-1}, {math.MaxInt64, 1}} {
		if drafts, err := Split(line, batches); err == nil || drafts != nil {
			t.Errorf("Split(%v) = %v, %v; want no drafts and an error", batches, drafts, err)
		}
	}
}
