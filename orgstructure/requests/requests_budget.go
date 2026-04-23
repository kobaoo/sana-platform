package requests

import (
	"context"
	"errors"

	"sana-platform/db/ent"

	uuid "encore.dev/types/uuid"
)

type BudgetService struct {
	client *ent.Client
}

func NewBudgetService(client *ent.Client) *BudgetService {
	return &BudgetService{client: client}
}

func (s *BudgetService) getContract(ctx context.Context, req *ent.Request) (*ent.ContractSupplier, error) {
	ext, err := s.client.ExternalTrainingEvent.
		Get(ctx, req.EntityID)

	if err != nil {
		return nil, err
	}

	return s.client.ContractSupplier.Get(ctx, ext.ContractID)
}

func (s *BudgetService) Reserve(ctx context.Context, reqID uuid.UUID, amount float64) error {
	req, err := s.client.Request.Get(ctx, reqID)
	if err != nil {
		return err
	}

	contract, err := s.getContract(ctx, req)
	if err != nil {
		return err
	}

	if contract.RemainingAmount < amount {
		return errors.New("not enough budget")
	}

	// списываем резерв
	_, err = s.client.ContractSupplier.
		UpdateOneID(contract.ID).
		SetRemainingAmount(contract.RemainingAmount - amount).
		Save(ctx)

	if err != nil {
		return err
	}

	// логируем
	_, err = s.client.RequestBudgetTransaction.
		Create().
		SetRequestID(reqID).
		SetContractID(contract.ID).
		SetAmount(amount).
		SetOperationType("RESERVE").
		Save(ctx)

	return err
}

func (s *BudgetService) WriteOff(ctx context.Context, reqID uuid.UUID, amount float64) error {
	req, err := s.client.Request.Get(ctx, reqID)
	if err != nil {
		return err
	}

	contract, err := s.getContract(ctx, req)
	if err != nil {
		return err
	}

	_, err = s.client.RequestBudgetTransaction.
		Create().
		SetRequestID(reqID).
		SetContractID(contract.ID).
		SetAmount(amount).
		SetOperationType("WRITE_OFF").
		Save(ctx)

	return err
}

func (s *BudgetService) Refund(ctx context.Context, reqID uuid.UUID, amount float64) error {
	req, err := s.client.Request.Get(ctx, reqID)
	if err != nil {
		return err
	}

	contract, err := s.getContract(ctx, req)
	if err != nil {
		return err
	}

	// возвращаем деньги
	_, err = s.client.ContractSupplier.
		UpdateOneID(contract.ID).
		SetRemainingAmount(contract.RemainingAmount + amount).
		Save(ctx)

	if err != nil {
		return err
	}

	_, err = s.client.RequestBudgetTransaction.
		Create().
		SetRequestID(reqID).
		SetContractID(contract.ID).
		SetAmount(amount).
		SetOperationType("REFUND").
		Save(ctx)

	return err
}

func (s *BudgetService) Release(ctx context.Context, reqID uuid.UUID, amount float64) error {
	return s.Refund(ctx, reqID, amount)
}
