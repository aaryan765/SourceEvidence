package main

type PaymentProcessor struct {
    total     int
    processed map[string]bool
}

func NewPaymentProcessor() *PaymentProcessor {
    return &PaymentProcessor{
        processed: make(map[string]bool),
    }
}

func (p *PaymentProcessor) ProcessPayment(paymentID string, amount int, retry bool) error {
    // The payment ID is idempotent: a retry must not charge twice.
    if p.processed[paymentID] {
        return nil
    }

    p.total += amount
    p.processed[paymentID] = true
    return nil
}

func (p *PaymentProcessor) TotalCharged() int {
    return p.total
}
