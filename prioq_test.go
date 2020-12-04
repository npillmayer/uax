package uax

import "testing"

// --- ad hoc type for testing purposes----------------------------------

type item struct { // will implement RuneSubscriber
	done bool
}

func (it *item) Done() bool {
	return it.done
}

func (it *item) Unsubscribed()                              {}
func (it *item) RuneEvent(r rune, codePointClass int) []int { return nil }
func (it *item) MatchLength() int                           { return 1 }

// ----------------------------------------------------------------------

func TestQueue1(t *testing.T) {
	pq := &DefaultRunePublisher{}
	if pq.PopDone() != nil {
		t.Error("Should not be able to PopDone() on empty Q")
	}
	pq.Push(&item{done: true})
	if pq.Len() != 1 {
		t.Error("Len() should be 1 for Q with 1 item")
	}
	if !pq.Top().Done() {
		t.Error("single item in Q is done; access by Top() does not reflect this")
	}
	if pq.gap != 0 {
		t.Errorf("gap calculation after Push() to Q is wrong, should be 0, is %d", pq.gap)
	}
	pq.Fix(0)
	if pq.gap != 0 {
		t.Errorf("gap calculation in Q after Fix() is wrong, should be 0, is %d", pq.gap)
	}
}

func TestQueue2(t *testing.T) {
	pq := &DefaultRunePublisher{}
	it := &item{done: true}
	pq.Push(it)
	pq.Push(&item{done: false})
	if pq.Len() != 2 {
		t.Error("Len() should be 2 for Q with 2 item")
	}
	if !pq.Top().Done() {
		t.Error("top item in Q is not done; should be")
	}
	if pq.gap != 1 {
		t.Errorf("gap calculation after Push()+Push() is wrong, should be 1, is %d", pq.gap)
	}
	it.done = false
	pq.Fix(1)
	if pq.Top().Done() {
		t.Error("top item in Q is done; should not be any more")
	}
	if pq.gap != 2 {
		t.Errorf("gap calculation after Fix() in Q with length 2 is wrong: %d", pq.gap)
	}
}

func TestQueue3(t *testing.T) {
	pq := &DefaultRunePublisher{}
	it := &item{done: false}
	pq.Push(it)
	pq.Push(&item{done: true})
	pq.Push(&item{done: false})
	if pq.Len() != 3 {
		t.Error("Len() should be 3 for Q with 3 item")
	}
	if !pq.Top().Done() {
		t.Error("top item in Q is not done; should be")
	}
	if pq.gap != 2 {
		t.Errorf("gap calculation after 3 x Push() is wrong: %d", pq.gap)
	}
	it.done = true
	pq.Fix(0)
	if pq.gap != 1 {
		t.Errorf("gap calculation after Fix() in Q with length 3 is wrong: %d", pq.gap)
	}
}

func TestQueue4(t *testing.T) {
	pq := &DefaultRunePublisher{}
	it1 := &item{done: false}
	pq.Push(it1)
	it2 := &item{done: false}
	pq.Push(it2)
	it3 := &item{done: false}
	pq.Push(it3)
	it4 := &item{done: false}
	pq.Push(it4)
	if pq.gap != 4 {
		t.Errorf("gap calculation after 4 x Push() is wrong: %d", pq.gap)
	}
	it3.done = true
	pq.Fix(2)
	it2.done = true
	pq.Fix(1)
	it1.done = true
	pq.Fix(0)
	if pq.gap != 1 {
		t.Errorf("gap calculation after Fix() is wrong: %d", pq.gap)
	}
	for j := 0; j < 3; j++ {
		if s := pq.PopDone(); s == nil {
			t.Error("top 3 items should have been done")
		}
	}
	if pq.Top().Done() {
		t.Error("top/only item in Q is done; should not be")
	}
	if pq.Len() != 1 {
		t.Error("Len() should be 1 after 3 pops")
	}
}

func TestQueue5(t *testing.T) {
	pq := &DefaultRunePublisher{}
	it1 := &item{done: false}
	pq.Push(it1)
	it2 := &item{done: false}
	pq.Push(it2)
	it3 := &item{done: false}
	pq.Push(it3)
	if pq.gap != 3 {
		t.Errorf("gap calculation after 3 x Push() is wrong: %d", pq.gap)
	}
	it2.done = true
	pq.Fix(1)
	if s := pq.PopDone(); s == nil {
		t.Error("top item should have been done")
	}
	if pq.Len() != 2 {
		t.Error("Len() should be 2 after 1 pop")
	}
	it4 := &item{done: false}
	pq.Push(it4)
	if s := pq.PopDone(); s != nil {
		t.Error("top item should not have been done")
	}
	it1.done = true
	pq.Fix(0)
	if s := pq.PopDone(); s == nil {
		t.Error("new top item should have been done")
	}
}
