package contracts_dzo

import (
	"context"
	"fmt"
	"testing"
	"time"

	"encore.app/orgstructure/organizations"
	"encore.dev/beta/errs"
)

func testCtx() context.Context {
	return context.Background()
}

func makeDZO(t *testing.T) string {
	t.Helper()

	suffix := time.Now().UnixNano()
	resp, err := organizations.CreateOrg(testCtx(), &organizations.CreateOrgRequest{
		Name: fmt.Sprintf("DZO %d", suffix),
		Code: fmt.Sprintf("DZO-%d", suffix),
		Type: organizations.OrgTypeSubsidiary,
	})
	if err != nil {
		t.Fatalf("makeDZO: %v", err)
	}

	return resp.Organization.ID
}

func makeContract(t *testing.T, dzoID string) *ContractDZO {
	t.Helper()

	suffix := time.Now().UnixNano()
	resp, err := CreateContractDZO(testCtx(), &CreateContractDZORequest{
		DZOID:          dzoID,
		ContractNumber: fmt.Sprintf("CN-%d", suffix),
		Category:       "Рамочный",
		SignedDate:     "2026-01-30",
		AmountWithVAT:  1000,
	})
	if err != nil {
		t.Fatalf("makeContract: %v", err)
	}

	return &resp.Contract
}

func strPtr(v string) *string {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

func TestCreateContractDZO_Success(t *testing.T) {
	dzoID := makeDZO(t)

	resp, err := CreateContractDZO(testCtx(), &CreateContractDZORequest{
		DZOID:          dzoID,
		ContractNumber: fmt.Sprintf("21/-----%d", time.Now().UnixNano()),
		Category:       "Рамочный",
		SignedDate:     "2026-01-30",
		AmountWithVAT:  243600000.00,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Contract.ID == "" {
		t.Error("expected non-empty id")
	}
	if resp.Contract.TotalAmount != 243600000.00 {
		t.Errorf("expected total_amount 243600000.00, got %f", resp.Contract.TotalAmount)
	}
	if resp.Contract.SpentAmount != 0 {
		t.Errorf("expected spent_amount 0, got %f", resp.Contract.SpentAmount)
	}
	if resp.Contract.RemainingAmount != resp.Contract.TotalAmount {
		t.Errorf("expected remaining_amount = total_amount, got %f", resp.Contract.RemainingAmount)
	}
}

func TestCreateContractDZO_InvalidDZOID(t *testing.T) {
	_, err := CreateContractDZO(testCtx(), &CreateContractDZORequest{
		DZOID:          "bad-uuid",
		ContractNumber: fmt.Sprintf("CN-%d", time.Now().UnixNano()),
		Category:       "Рамочный",
		SignedDate:     "2026-01-30",
		AmountWithVAT:  100,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateContractDZO_InvalidSignedDate(t *testing.T) {
	dzoID := makeDZO(t)

	_, err := CreateContractDZO(testCtx(), &CreateContractDZORequest{
		DZOID:          dzoID,
		ContractNumber: fmt.Sprintf("CN-%d", time.Now().UnixNano()),
		Category:       "Рамочный",
		SignedDate:     "30-01-2026",
		AmountWithVAT:  100,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestCreateContractDZO_AmountMustBePositive(t *testing.T) {
	dzoID := makeDZO(t)

	_, err := CreateContractDZO(testCtx(), &CreateContractDZORequest{
		DZOID:          dzoID,
		ContractNumber: fmt.Sprintf("CN-%d", time.Now().UnixNano()),
		Category:       "Рамочный",
		SignedDate:     "2026-01-30",
		AmountWithVAT:  0,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestGetContractDZO_Success(t *testing.T) {
	dzoID := makeDZO(t)
	created := makeContract(t, dzoID)

	resp, err := GetContractDZO(testCtx(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Contract.ID != created.ID {
		t.Errorf("expected id %q, got %q", created.ID, resp.Contract.ID)
	}
}

func TestGetContractDZO_NotFound(t *testing.T) {
	_, err := GetContractDZO(testCtx(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestListContractsDZO_FilterByStatus(t *testing.T) {
	dzoID := makeDZO(t)
	active := makeContract(t, dzoID)
	toDelete := makeContract(t, dzoID)

	if _, err := DeleteContractDZO(testCtx(), toDelete.ID); err != nil {
		t.Fatalf("failed to delete contract: %v", err)
	}

	resp, err := ListContractsDZO(testCtx(), &ListContractsDZORequest{IsActive: "true"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundActive := false
	for _, c := range resp.Contracts {
		if c.ID == toDelete.ID {
			t.Errorf("deleted contract should not appear in active list")
		}
		if c.ID == active.ID {
			foundActive = true
		}
	}
	if !foundActive {
		t.Error("expected active contract in list")
	}
}

func TestAddAmendmentAndSpendBudget_Success(t *testing.T) {
	dzoID := makeDZO(t)
	contract := makeContract(t, dzoID)

	amendResp, err := AddContractDZOAmendment(testCtx(), contract.ID, &AddAmendmentRequest{
		AmendmentNumber: strPtr("ДС-1"),
		AmendmentDate:   strPtr("2026-02-15"),
		AmendmentAmount: 500,
	})
	if err != nil {
		t.Fatalf("unexpected amendment error: %v", err)
	}
	if amendResp.Contract.TotalAmount != 1500 {
		t.Errorf("expected total_amount 1500, got %f", amendResp.Contract.TotalAmount)
	}

	spendResp, err := SpendContractDZOBudget(testCtx(), contract.ID, &SpendContractBudgetRequest{Amount: 300})
	if err != nil {
		t.Fatalf("unexpected spend error: %v", err)
	}
	if spendResp.Contract.SpentAmount != 300 {
		t.Errorf("expected spent_amount 300, got %f", spendResp.Contract.SpentAmount)
	}
	if spendResp.Contract.RemainingAmount != 1200 {
		t.Errorf("expected remaining_amount 1200, got %f", spendResp.Contract.RemainingAmount)
	}

	analyticsResp, err := GetContractDZOAnalytics(testCtx(), contract.ID)
	if err != nil {
		t.Fatalf("unexpected analytics error: %v", err)
	}
	if analyticsResp.Analytics.UtilizationPercent != 20 {
		t.Errorf("expected utilization 20, got %f", analyticsResp.Analytics.UtilizationPercent)
	}
}

func TestSpendContractDZOBudget_InsufficientAmount(t *testing.T) {
	dzoID := makeDZO(t)
	contract := makeContract(t, dzoID)

	_, err := SpendContractDZOBudget(testCtx(), contract.ID, &SpendContractBudgetRequest{Amount: 1001})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestListContractsDZO_FilterByDZOAndRemainingAmount(t *testing.T) {
	firstDZO := makeDZO(t)
	secondDZO := makeDZO(t)

	first := makeContract(t, firstDZO)
	second := makeContract(t, secondDZO)

	if _, err := SpendContractDZOBudget(testCtx(), first.ID, &SpendContractBudgetRequest{Amount: 300}); err != nil {
		t.Fatalf("failed to spend budget for first contract: %v", err)
	}

	resp, err := ListContractsDZO(testCtx(), &ListContractsDZORequest{
		DZOID:             firstDZO,
		RemainingAmountLT: "800",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, c := range resp.Contracts {
		if c.ID == second.ID {
			t.Errorf("contract from another dzo should not appear")
		}
	}
}

func TestListContractsDZO_InvalidIsActive(t *testing.T) {
	_, err := ListContractsDZO(testCtx(), &ListContractsDZORequest{IsActive: "not-bool"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestListContractsDZO_InvalidDZOID(t *testing.T) {
	_, err := ListContractsDZO(testCtx(), &ListContractsDZORequest{DZOID: "bad-uuid"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestListContractsDZO_InvalidRemainingAmountLT(t *testing.T) {
	_, err := ListContractsDZO(testCtx(), &ListContractsDZORequest{RemainingAmountLT: "abc"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateContractDZO_Success(t *testing.T) {
	dzoID := makeDZO(t)
	contract := makeContract(t, dzoID)

	if _, err := SpendContractDZOBudget(testCtx(), contract.ID, &SpendContractBudgetRequest{Amount: 200}); err != nil {
		t.Fatalf("failed to spend budget before update: %v", err)
	}

	newCategory := "Обычный"
	newAmount := 1500.0

	resp, err := UpdateContractDZO(testCtx(), contract.ID, &UpdateContractDZORequest{
		Category:      &newCategory,
		AmountWithVAT: float64Ptr(newAmount),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Contract.Category != newCategory {
		t.Errorf("expected category %q, got %q", newCategory, resp.Contract.Category)
	}
	if resp.Contract.TotalAmount != 1500 {
		t.Errorf("expected total_amount 1500, got %f", resp.Contract.TotalAmount)
	}
	if resp.Contract.RemainingAmount != 1300 {
		t.Errorf("expected remaining_amount 1300, got %f", resp.Contract.RemainingAmount)
	}
}

func TestUpdateContractDZO_InvalidSignedDate(t *testing.T) {
	dzoID := makeDZO(t)
	contract := makeContract(t, dzoID)
	badDate := "01-30-2026"

	_, err := UpdateContractDZO(testCtx(), contract.ID, &UpdateContractDZORequest{SignedDate: &badDate})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestUpdateContractDZO_NotFound(t *testing.T) {
	newCategory := "Ghost"

	_, err := UpdateContractDZO(testCtx(), "00000000-0000-0000-0000-000000000000", &UpdateContractDZORequest{Category: &newCategory})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestDeleteContractDZO_NotFoundWhenAlreadyDeleted(t *testing.T) {
	dzoID := makeDZO(t)
	contract := makeContract(t, dzoID)

	if _, err := DeleteContractDZO(testCtx(), contract.ID); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}

	_, err := DeleteContractDZO(testCtx(), contract.ID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestAddContractDZOAmendment_InvalidAmount(t *testing.T) {
	dzoID := makeDZO(t)
	contract := makeContract(t, dzoID)

	_, err := AddContractDZOAmendment(testCtx(), contract.ID, &AddAmendmentRequest{AmendmentAmount: -1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestAddContractDZOAmendment_InvalidDate(t *testing.T) {
	dzoID := makeDZO(t)
	contract := makeContract(t, dzoID)

	_, err := AddContractDZOAmendment(testCtx(), contract.ID, &AddAmendmentRequest{
		AmendmentDate:   strPtr("15-02-2026"),
		AmendmentAmount: 100,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestGetContractDZOAnalytics_NotFound(t *testing.T) {
	_, err := GetContractDZOAnalytics(testCtx(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.NotFound {
		t.Errorf("expected NotFound, got %v", errs.Code(err))
	}
}

func TestSpendContractDZOBudget_InvalidAmount(t *testing.T) {
	dzoID := makeDZO(t)
	contract := makeContract(t, dzoID)

	_, err := SpendContractDZOBudget(testCtx(), contract.ID, &SpendContractBudgetRequest{Amount: 0})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}

func TestSpendContractDZOBudget_InactiveContract(t *testing.T) {
	dzoID := makeDZO(t)
	contract := makeContract(t, dzoID)

	if _, err := UpdateContractDZO(testCtx(), contract.ID, &UpdateContractDZORequest{IsActive: boolPtr(false)}); err != nil {
		t.Fatalf("failed to deactivate contract: %v", err)
	}

	_, err := SpendContractDZOBudget(testCtx(), contract.ID, &SpendContractBudgetRequest{Amount: 1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errs.Code(err) != errs.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", errs.Code(err))
	}
}
