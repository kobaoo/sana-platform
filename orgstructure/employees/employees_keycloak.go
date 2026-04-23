package employees

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"encore.dev/rlog"
)

var secrets struct {
	KeycloakIssuerURL         string
	KeycloakAdminClientID     string
	KeycloakAdminClientSecret string
}

var kcAdmin = &keycloakAdminClient{
	httpClient: &http.Client{Timeout: 10 * time.Second},
}

type keycloakAdminClient struct {
	httpClient  *http.Client
	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

func (k *keycloakAdminClient) baseURL() string {
	u, err := url.Parse(secrets.KeycloakIssuerURL)
	if err != nil || secrets.KeycloakIssuerURL == "" {
		return ""
	}
	parts := strings.SplitN(u.Path, "/realms/", 2)
	u.Path = parts[0]
	return u.String()
}

func (k *keycloakAdminClient) realm() string {
	parts := strings.SplitN(secrets.KeycloakIssuerURL, "/realms/", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSuffix(parts[1], "/")
}

func (k *keycloakAdminClient) adminToken() (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.cachedToken != "" && time.Now().Before(k.tokenExpiry) {
		return k.cachedToken, nil
	}

	tokenURL := fmt.Sprintf("%s/protocol/openid-connect/token", secrets.KeycloakIssuerURL)
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {secrets.KeycloakAdminClientID},
		"client_secret": {secrets.KeycloakAdminClientSecret},
	}
	resp, err := k.httpClient.Post(tokenURL, "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("keycloak admin token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("keycloak admin token: status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("keycloak admin token decode: %w", err)
	}

	k.cachedToken = result.AccessToken
	k.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-30) * time.Second)
	return k.cachedToken, nil
}

type kcCreateUserRequest struct {
	Username    string                `json:"username"`
	Email       string                `json:"email"`
	FirstName   string                `json:"firstName"`
	LastName    string                `json:"lastName,omitempty"`
	Enabled     bool                  `json:"enabled"`
	Attributes  map[string][]string   `json:"attributes,omitempty"`
	Credentials []kcCredentialRequest `json:"credentials,omitempty"`
}

type kcCredentialRequest struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	Temporary bool   `json:"temporary"`
}

// createKeycloakUser создаёт пользователя в keycloak и возвращает его id
func createKeycloakUser(ctx context.Context, email string, fullName string, companyID string, dzoID string) (string, error) {
	if secrets.KeycloakIssuerURL == "" || secrets.KeycloakAdminClientID == "" {
		rlog.Warn("keycloak not configured, using stub kcUserID")
		return fmt.Sprintf("stub-%s", email), nil
	}

	token, err := kcAdmin.adminToken()
	if err != nil {
		return "", fmt.Errorf("keycloak: get admin token: %w", err)
	}

	usersURL := fmt.Sprintf("%s/admin/realms/%s/users", kcAdmin.baseURL(), kcAdmin.realm())

	firstName, lastName := splitFullName(fullName)

	payload := kcCreateUserRequest{
		Username:  email,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Enabled:   true,
		Attributes: map[string][]string{
			"companyId": {companyID},
			"dzoId":     {dzoID},
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, usersURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := kcAdmin.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("keycloak: create user request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return "", fmt.Errorf("keycloak: user with email %q already exists", email)
	}
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("keycloak: create user: status %d: %s", resp.StatusCode, respBody)
	}

	location := resp.Header.Get("Location")
	parts := strings.Split(location, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("keycloak: empty Location header after user creation")
	}
	return parts[len(parts)-1], nil
}

// deleteKeycloakUser удаляет пользователя из keycloak во время ошибок
func deleteKeycloakUser(ctx context.Context, kcUserID string) {
	if secrets.KeycloakIssuerURL == "" || kcUserID == "" || strings.HasPrefix(kcUserID, "stub-") {
		return
	}

	token, err := kcAdmin.adminToken()
	if err != nil {
		rlog.Error("compensation: failed to get admin token", "err", err.Error())
		return
	}

	userURL := fmt.Sprintf("%s/admin/realms/%s/users/%s",
		kcAdmin.baseURL(), kcAdmin.realm(), kcUserID)

	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, userURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := kcAdmin.httpClient.Do(req)
	if err != nil {
		rlog.Error("compensation: failed to delete keycloak user",
			"kcUserID", kcUserID, "err", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		rlog.Error("compensation: unexpected status on keycloak delete",
			"kcUserID", kcUserID, "status", resp.StatusCode, "body", string(body))
		return
	}
	rlog.Info("compensation: keycloak user deleted", "kcUserID", kcUserID)
}

func splitFullName(fullName string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(fullName))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}
