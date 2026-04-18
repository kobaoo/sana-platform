package applications

import (
	"testing"

	"encore.app/auth/authhandler"
)

func ptrStr(v string) *string {
	return &v
}

func TestInitialStatusForKind(t *testing.T) {
	if got := initialStatusForKind(ApplicationKindRegular); got != ApplicationStatusDraft {
		t.Errorf("expected draft, got %q", got)
	}
	if got := initialStatusForKind(ApplicationKindClosed); got != ApplicationStatusCompleted {
		t.Errorf("expected completed, got %q", got)
	}
	if got := initialStatusForKind(ApplicationKindArchived); got != ApplicationStatusCompleted {
		t.Errorf("expected completed, got %q", got)
	}
}

func TestCanCreateApplication(t *testing.T) {
	if !CanCreateApplication(authhandler.RoleHR, ApplicationKindRegular) {
		t.Error("HR should be able to create regular applications")
	}
	if CanCreateApplication(authhandler.RoleHR, ApplicationKindArchived) {
		t.Error("HR should not be able to create archived applications")
	}
	if !CanCreateApplication(authhandler.RoleADM, ApplicationKindArchived) {
		t.Error("ADM should be able to create archived applications")
	}
	if CanCreateApplication(authhandler.RoleEMP, ApplicationKindRegular) {
		t.Error("EMP should not be able to create applications")
	}
}

func TestCanAccessApplication(t *testing.T) {
	app := &Application{
		DzoID:           ptrStr("dzo-1"),
		CreatedByUserID: ptrStr("user-1"),
	}

	if !CanAccessApplication(&Caller{Role: authhandler.RoleSA}, app) {
		t.Error("SA should access any application")
	}
	if !CanAccessApplication(&Caller{Role: authhandler.RoleHR, DzoID: ptrStr("dzo-1")}, app) {
		t.Error("HR should access application inside own DZO")
	}
	if CanAccessApplication(&Caller{Role: authhandler.RoleHR, DzoID: ptrStr("dzo-2")}, app) {
		t.Error("HR should not access application from another DZO")
	}
	if !CanAccessApplication(&Caller{Role: authhandler.RoleHR, UserID: "user-1"}, app) {
		t.Error("creator should access own application")
	}
}

func TestValidateStatusTransition_Success(t *testing.T) {
	cases := []struct {
		kind    ApplicationKind
		current ApplicationStatus
		next    ApplicationStatus
	}{
		{ApplicationKindRegular, ApplicationStatusDraft, ApplicationStatusSubmitted},
		{ApplicationKindRegular, ApplicationStatusDraft, ApplicationStatusCancelled},
		{ApplicationKindRegular, ApplicationStatusSubmitted, ApplicationStatusInProcess},
		{ApplicationKindRegular, ApplicationStatusSubmitted, ApplicationStatusCompleted},
		{ApplicationKindRegular, ApplicationStatusInProcess, ApplicationStatusCompleted},
	}

	for _, tc := range cases {
		if !isAllowedStatusTransition(tc.kind, tc.current, tc.next) {
			t.Errorf("expected transition %q -> %q to be allowed", tc.current, tc.next)
		}
	}
}

func TestValidateStatusTransition_FailsForInvalidMove(t *testing.T) {
	if isAllowedStatusTransition(ApplicationKindRegular, ApplicationStatusCompleted, ApplicationStatusSubmitted) {
		t.Error("completed application should not transition back to submitted")
	}
}

func TestValidateStatusTransition_FailsForArchivedKinds(t *testing.T) {
	if isAllowedStatusTransition(ApplicationKindArchived, ApplicationStatusCompleted, ApplicationStatusCancelled) {
		t.Error("archived application should remain final")
	}
}

func TestCanTransitionApplication(t *testing.T) {
	app := &Application{
		Kind:   ApplicationKindRegular,
		Status: ApplicationStatusSubmitted,
		DzoID:  ptrStr("dzo-1"),
	}

	if !CanTransitionApplication(&Caller{Role: authhandler.RoleADM, DzoID: ptrStr("dzo-1")}, app, ApplicationStatusInProcess) {
		t.Error("ADM should move submitted application to in_process")
	}
	if CanTransitionApplication(&Caller{Role: authhandler.RoleHR, DzoID: ptrStr("dzo-1")}, app, ApplicationStatusInProcess) {
		t.Error("HR should not move application to in_process")
	}
	if !CanTransitionApplication(&Caller{Role: authhandler.RoleHR, DzoID: ptrStr("dzo-1")}, app, ApplicationStatusCancelled) {
		t.Error("HR should be able to cancel application in own DZO")
	}
}
