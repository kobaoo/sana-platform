package notifutil

import "testing"

func TestCertificateExpiryEvent(t *testing.T) {
	e := CertificateExpiryEvent{
		EmployeeID: "emp-1",
		CertID:     "cert-1",
		Title:      "Safety",
		ExpiryDate: "2026-10-01",
	}

	req := e.ToNotifyRequest()

	if req.UserID != "emp-1" {
		t.Errorf("want UserID=emp-1, got %s", req.UserID)
	}
	if req.Type != TypeCertExpiring {
		t.Errorf("want type=CERT_EXPIRING, got %s", req.Type)
	}
	if req.EntityType != EntityCertificate {
		t.Errorf("want entity=CERTIFICATE, got %s", req.EntityType)
	}
	if req.EntityID != "cert-1" {
		t.Errorf("want entityID=cert-1, got %s", req.EntityID)
	}
}

func TestRequestCreatedEvent(t *testing.T) {
	e := RequestCreatedEvent{
		InitiatorID: "emp-1",
		ApproverID:  "hr-1",
		RequestID:   "req-1",
	}

	req := e.ToNotifyRequest()

	// Уведомляем апрувера, не инициатора
	if req.UserID != "hr-1" {
		t.Errorf("want UserID=hr-1, got %s", req.UserID)
	}
	if req.Type != TypeRequestCreated {
		t.Errorf("want type=REQUEST_CREATED, got %s", req.Type)
	}
	if req.EntityType != EntityRequest {
		t.Errorf("want entity=REQUEST, got %s", req.EntityType)
	}
}

func TestRequestStepUpdatedEvent(t *testing.T) {
	e := RequestStepUpdatedEvent{
		ApproverID: "adm-1",
		RequestID:  "req-2",
		Step:       2,
	}

	req := e.ToNotifyRequest()

	if req.UserID != "adm-1" {
		t.Errorf("want UserID=adm-1, got %s", req.UserID)
	}
	if req.Type != TypeRequestStepUpdated {
		t.Errorf("want type=REQUEST_STEP_UPDATED, got %s", req.Type)
	}
}

func TestRequestApprovedEvent(t *testing.T) {
	e := RequestApprovedEvent{
		InitiatorID: "emp-2",
		RequestID:   "req-3",
	}

	req := e.ToNotifyRequest()

	// Уведомляем инициатора об одобрении
	if req.UserID != "emp-2" {
		t.Errorf("want UserID=emp-2, got %s", req.UserID)
	}
	if req.Type != TypeRequestApproved {
		t.Errorf("want type=REQUEST_APPROVED, got %s", req.Type)
	}
}

func TestRequestCancelledEvent(t *testing.T) {
	e := RequestCancelledEvent{
		InitiatorID: "emp-3",
		RequestID:   "req-4",
	}

	req := e.ToNotifyRequest()

	if req.UserID != "emp-3" {
		t.Errorf("want UserID=emp-3, got %s", req.UserID)
	}
	if req.Type != TypeRequestCancelled {
		t.Errorf("want type=REQUEST_CANCELLED, got %s", req.Type)
	}
}
