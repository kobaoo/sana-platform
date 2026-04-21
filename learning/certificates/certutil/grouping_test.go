package certutil

import (
	"testing"
	"time"
)

func TestGroupByEmployee(t *testing.T) {
	now := time.Now()

	certs := []Certificate{
		{ID: "cert-1", EmployeeID: "emp-a", Title: "Safety Training", IssuedDate: now},
		{ID: "cert-2", EmployeeID: "emp-b", Title: "First Aid", IssuedDate: now},
		{ID: "cert-3", EmployeeID: "emp-a", Title: "Fire Safety", IssuedDate: now},
		{ID: "cert-4", EmployeeID: "emp-c", Title: "ISO 9001", IssuedDate: now},
		{ID: "cert-5", EmployeeID: "emp-b", Title: "CPR Certification", IssuedDate: now},
		{ID: "cert-6", EmployeeID: "emp-a", Title: "SCORM Completion", IssuedDate: now},
	}

	grouped := GroupByEmployee(certs)

	if got := len(grouped); got != 3 {
		t.Fatalf("expected 3 groups, got %d", got)
	}

	cases := []struct {
		id   string
		want int
	}{
		{"emp-a", 3},
		{"emp-b", 2},
		{"emp-c", 1},
	}

	for _, tc := range cases {
		got, ok := grouped[tc.id]
		if !ok {
			t.Errorf("group %q not found", tc.id)
			continue
		}
		if len(got) != tc.want {
			t.Errorf("emp %q: want %d certs, got %d", tc.id, tc.want, len(got))
		}
	}
}

func TestGroupByEmployee_Empty(t *testing.T) {
	if got := len(GroupByEmployee(nil)); got != 0 {
		t.Errorf("expected 0 groups, got %d", got)
	}
}

func TestGroupByEmployee_SingleEmployee(t *testing.T) {
	now := time.Now()
	certs := []Certificate{
		{ID: "cert-1", EmployeeID: "emp-solo", Title: "Cert A", IssuedDate: now},
		{ID: "cert-2", EmployeeID: "emp-solo", Title: "Cert B", IssuedDate: now},
	}

	grouped := GroupByEmployee(certs)

	if len(grouped) != 1 {
		t.Fatalf("expected 1 group, got %d", len(grouped))
	}
	if len(grouped["emp-solo"]) != 2 {
		t.Errorf("want 2 certs, got %d", len(grouped["emp-solo"]))
	}
}
