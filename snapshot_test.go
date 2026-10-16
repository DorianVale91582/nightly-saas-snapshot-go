package main

import (
	"testing"
	"time"
)

func TestSnapshotDecision(t *testing.T) {
	now := time.Date(2026, 9, 6, 1, 0, 0, 0, time.UTC)
	cases := []struct {
		name, tenant string
		want         bool
	}{
		{"active tenant", "acme", true}, {"closed tenant", "closed", false}, {"unknown tenant", "missing", false},
	}
	accounts := []Account{{TenantID: "acme", Active: true}, {TenantID: "closed", Active: false}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, got := snapshotFor(tc.tenant, accounts, now)
			if got != tc.want {
				t.Fatalf("snapshotFor(%q)=%v, want %v", tc.tenant, got, tc.want)
			}
		})
	}
}
