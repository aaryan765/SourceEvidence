package main

type PaymentService struct{ processor *PaymentProcessor }

func NewPaymentService() *PaymentService { return &PaymentService{processor: NewPaymentProcessor()} }
func (s *PaymentService) Charge(paymentID string, amount int, retry bool) error {
	return s.processor.ProcessPayment(paymentID, amount, retry)
}
func (s *PaymentService) TotalCharged() int { return s.processor.TotalCharged() }
