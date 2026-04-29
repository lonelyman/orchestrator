package auth

import (
	"crypto/tls"
	"fmt"
	"log/slog"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/go-ldap/ldap/v3"
)

type LDAPAdapter struct {
	server      string
	port        int
	baseDN      string
	domain      string
	devMode     bool
	devUsername string
	devPassword string
}

func NewLDAPAdapter(server string, port int, baseDN, domain string, devMode bool, devUsername, devPassword string) *LDAPAdapter {
	return &LDAPAdapter{
		server:      server,
		port:        port,
		baseDN:      baseDN,
		domain:      domain,
		devMode:     devMode,
		devUsername: devUsername,
		devPassword: devPassword,
	}
}

func (l *LDAPAdapter) Authenticate(username, password string) (*models.User, error) {
	// Dev Mode — ข้าม AD
	if l.devMode {
		slog.Warn("DEV MODE: bypassing AD authentication")
		if username == l.devUsername && password == l.devPassword {
			return &models.User{
				ID:          username,
				Username:    username,
				Email:       username + "@nutritionprofess.com",
				DisplayName: username,
				Department:  "Development",
				Role:        "admin",
			}, nil
		}
		return nil, fmt.Errorf("invalid credentials")
	}

	// Production — ใช้ AD จริง
	ldapURL := fmt.Sprintf("ldap://%s:%d", l.server, l.port)
	conn, err := ldap.DialURL(ldapURL)
	if err != nil {
		return nil, fmt.Errorf("connect AD: %w", err)
	}
	defer conn.Close()

	if err := conn.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		slog.Warn("startTLS failed", "error", err)
	}

	userDN := fmt.Sprintf("%s@%s", username, l.domain)
	if err := conn.Bind(userDN, password); err != nil {
		slog.Warn("authentication failed", "username", username, "error", err.Error())
		return nil, fmt.Errorf("invalid credentials")
	}

	searchRequest := ldap.NewSearchRequest(
		l.baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf("(sAMAccountName=%s)", ldap.EscapeFilter(username)),
		[]string{"sAMAccountName", "mail", "displayName", "department"},
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil || len(result.Entries) == 0 {
		slog.Warn("search AD failed, using basic info", "error", err)
		return &models.User{
			ID:          username,
			Username:    username,
			Email:       fmt.Sprintf("%s@%s", username, l.domain),
			DisplayName: username,
			Role:        "employee",
		}, nil
	}

	entry := result.Entries[0]
	user := &models.User{
		ID:          entry.GetAttributeValue("sAMAccountName"),
		Username:    entry.GetAttributeValue("sAMAccountName"),
		Email:       entry.GetAttributeValue("mail"),
		DisplayName: entry.GetAttributeValue("displayName"),
		Department:  entry.GetAttributeValue("department"),
		Role:        "employee",
	}

	slog.Info("authentication success", "username", username, "department", user.Department)
	return user, nil
}
