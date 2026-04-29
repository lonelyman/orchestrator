package auth

import (
	"crypto/tls"
	"fmt"
	"log/slog"

	"github.com/enterprise-ai/orchestrator/internal/domain/models"
	"github.com/go-ldap/ldap/v3"
)

type LDAPAdapter struct {
	server string
	port   int
	baseDN string
	domain string
}

func NewLDAPAdapter(server string, port int, baseDN, domain string) *LDAPAdapter {
	return &LDAPAdapter{server: server, port: port, baseDN: baseDN, domain: domain}
}

func (l *LDAPAdapter) Authenticate(username, password string) (*models.User, error) {
	// เชื่อมต่อ LDAP
	ldapURL := fmt.Sprintf("ldap://%s:%d", l.server, l.port)
	conn, err := ldap.DialURL(ldapURL)
	if err != nil {
		return nil, fmt.Errorf("connect AD: %w", err)
	}
	defer conn.Close()

	// StartTLS
	if err := conn.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		slog.Warn("startTLS failed, continuing without TLS", "error", err)
	}

	// Bind ด้วย username@domain
	userDN := fmt.Sprintf("%s@%s", username, l.domain)
	if err := conn.Bind(userDN, password); err != nil {
		slog.Warn("authentication failed", "username", username, "error", err.Error())
		return nil, fmt.Errorf("invalid credentials")
	}

	// ดึงข้อมูล user จาก AD
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
		// ถ้า search ไม่ได้ ใช้ข้อมูลจาก username แทน
		slog.Warn("search AD failed, using basic user info", "error", err)
		return &models.User{
			ID:          username,
			Username:    username,
			Email:       userDN,
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
