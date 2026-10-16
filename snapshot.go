package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Account struct {
	TenantID string `json:"tenant_id"`
	Active   bool   `json:"active"`
	Admin    bool   `json:"admin"`
}
type Snapshot struct {
	TenantID string    `json:"tenant_id"`
	TakenAt  time.Time `json:"taken_at"`
	Accounts []Account `json:"accounts"`
}

func snapshotFor(tenant string, accounts []Account, now time.Time) (Snapshot, bool) {
	for _, a := range accounts {
		if a.TenantID == tenant && a.Active {
			return Snapshot{tenant, now.UTC(), accounts}, true
		}
	}
	return Snapshot{}, false
}

func runSnapshot(c *InfraiClient, bucket, tenant string, accounts []Account, now time.Time) (string, error) {
	s, ok := snapshotFor(tenant, accounts, now)
	if !ok {
		return "", fmt.Errorf("tenant is inactive")
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	key := tenant + "/" + now.UTC().Format("20060102") + ".json"
	if err := c.BucketCreate(bucket); err != nil {
		return "", err
	}
	if err := c.ObjectPut(bucket, key, b); err != nil {
		return "", err
	}
	return key, nil
}

func serve(c *InfraiClient, bucket string, accounts []Account) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/snapshot" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		tenant := strings.TrimSpace(r.URL.Query().Get("tenant"))
		if tenant == "" {
			http.Error(w, "tenant is required", http.StatusBadRequest)
			return
		}
		key, err := runSnapshot(c, bucket, tenant, accounts, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"object_key": key})
	})
}
