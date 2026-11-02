package xray

import (
	"testing"

	"github.com/xtls/xray-core/proxy/socks"
)

// The modified core's mixed inbound is a SOCKS server, so an account built
// from the panel's email/password must be a socks.Account or AddUser is a no-op.
func TestBuildUserAccountMixed(t *testing.T) {
	tests := []struct {
		name     string
		user     map[string]any
		username string
	}{
		{"panel client uses email as username", map[string]any{"email": "ivy@example.test", "password": "pw-ivy"}, "ivy@example.test"},
		{"explicit username wins", map[string]any{"email": "ivy@example.test", "username": "ivy", "password": "pw-ivy"}, "ivy"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			typed, err := buildUserAccount("mixed", tc.user)
			if err != nil {
				t.Fatalf("buildUserAccount: %v", err)
			}
			if typed == nil {
				t.Fatal("buildUserAccount returned no account")
			}
			instance, err := typed.GetInstance()
			if err != nil {
				t.Fatalf("GetInstance: %v", err)
			}
			account, ok := instance.(*socks.Account)
			if !ok {
				t.Fatalf("account = %T, want *socks.Account", instance)
			}
			if account.Username != tc.username || account.Password != "pw-ivy" {
				t.Fatalf("account = %#v, want username %q password pw-ivy", account, tc.username)
			}
		})
	}
}

func TestBuildUserAccountMixedRequiresPassword(t *testing.T) {
	for name, user := range map[string]map[string]any{
		"missing": {"email": "ivy@example.test"},
		"empty":   {"email": "ivy@example.test", "password": ""},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := buildUserAccount("mixed", user); err == nil {
				t.Fatal("a mixed user without a password must be refused, not added unauthenticatable")
			}
		})
	}
}
