package main

import "testing"

func TestFirstPaymentIsCharged(t *testing.T) {
	processor := NewPaymentProcessor()
	if err := processor.ProcessPayment("pay-1001", 1000, false); err != nil {
		t.Fatal(err)
	}
	if got := processor.TotalCharged(); got != 1000 {
		t.Fatalf("expected total 1000, got %d", got)
	}
}

func TestRetryDoesNotDuplicateCharge(t *testing.T) {
	processor := NewPaymentProcessor()
	if err := processor.ProcessPayment("pay-1001", 1000, false); err != nil {
		t.Fatal(err)
	}
	if err := processor.ProcessPayment("pay-1001", 1000, true); err != nil {
		t.Fatal(err)
	}
	if got := processor.TotalCharged(); got != 1000 {
		t.Fatalf("duplicate charge: expected total 1000, got %d", got)
	}
}
