package auth

import (
	"context"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// Manager handles Azure device code authentication lifecycle.
// In-memory; restart clears state.
type Manager struct {
    mu sync.RWMutex
    deviceCred *azidentity.DeviceCodeCredential
    lastToken *azcore.AccessToken
    authenticating bool
    authenticated bool
    lastMsg string
    userCode string
    verificationURL string
}

func NewManager() *Manager { return &Manager{} }

// StartDeviceLogin begins device code flow.
func (m *Manager) StartDeviceLogin(ctx context.Context, tenantID string) (code, verificationURL, message string, err error) {
    m.mu.Lock()
    if m.authenticated {
        code, verificationURL, message = m.userCode, m.verificationURL, "Already authenticated"
        m.mu.Unlock()
        return
    }
    if m.authenticating {
        code, verificationURL, message = m.userCode, m.verificationURL, m.lastMsg
        m.mu.Unlock()
        return
    }
    m.authenticating = true
    m.mu.Unlock()

    cred, err := azidentity.NewDeviceCodeCredential(&azidentity.DeviceCodeCredentialOptions{
        TenantID: tenantID,
        UserPrompt: func(ctx context.Context, dc azidentity.DeviceCodeMessage) error {
            m.mu.Lock()
            m.lastMsg = dc.Message
            m.userCode = dc.UserCode
            m.verificationURL = dc.VerificationURL
            m.mu.Unlock()
            return nil
        },
    })
    if err != nil {
        m.mu.Lock(); m.authenticating = false; m.lastMsg = err.Error(); m.mu.Unlock()
        return "", "", "", err
    }
    go func() {
        ctx2, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
        defer cancel()
        tok, tErr := cred.GetToken(ctx2, policy.TokenRequestOptions{Scopes: []string{"https://management.azure.com/.default"}})
        m.mu.Lock()
        defer m.mu.Unlock()
        if tErr != nil {
            m.lastMsg = "authentication failed: " + tErr.Error()
            m.authenticating = false
            return
        }
        m.deviceCred = cred
        m.lastToken = &tok
        m.authenticated = true
        m.authenticating = false
        m.lastMsg = "authenticated"
    }()
    time.Sleep(50 * time.Millisecond)
    m.mu.RLock(); defer m.mu.RUnlock()
    return m.userCode, m.verificationURL, m.lastMsg, nil
}

func (m *Manager) Status() (authenticating, authenticated bool, msg, code, url string) {
    m.mu.RLock(); defer m.mu.RUnlock()
    return m.authenticating, m.authenticated, m.lastMsg, m.userCode, m.verificationURL
}

func (m *Manager) IsAuthenticated() bool { m.mu.RLock(); defer m.mu.RUnlock(); return m.authenticated }
