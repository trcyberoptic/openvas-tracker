package service

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"

	"github.com/cyberoptic/openvas-tracker/internal/config"
)

type LDAPUser struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	DN          string `json:"dn"`
}

type LDAPService struct{}

func NewLDAPService() *LDAPService {
	return &LDAPService{}
}

// Authenticate checks user credentials against AD and verifies group membership.
func (s *LDAPService) Authenticate(cfg config.LDAPConfig, username, password string) (*LDAPUser, error) {
	conn, err := s.connect(cfg)
	if err != nil {
		return nil, fmt.Errorf("ldap connect: %w", err)
	}
	defer conn.Close()

	// Bind with service account to search for user
	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return nil, fmt.Errorf("ldap bind: %w", err)
	}

	// Search for user
	filter := fmt.Sprintf(cfg.UserFilter, ldap.EscapeFilter(username))
	sr, err := conn.Search(ldap.NewSearchRequest(
		cfg.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 10, false,
		filter, []string{"dn", "sAMAccountName", "displayName", "mail", "memberOf"}, nil,
	))
	if err != nil || len(sr.Entries) == 0 {
		return nil, fmt.Errorf("user not found in directory")
	}

	entry := sr.Entries[0]

	// Check group membership
	if cfg.GroupDN != "" {
		inGroup := false
		for _, g := range entry.GetAttributeValues("memberOf") {
			if strings.EqualFold(g, cfg.GroupDN) {
				inGroup = true
				break
			}
		}
		if !inGroup {
			return nil, fmt.Errorf("user is not a member of the required group")
		}
	}

	// Verify user password by re-binding
	if err := conn.Bind(entry.DN, password); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return &LDAPUser{
		Username:    entry.GetAttributeValue("sAMAccountName"),
		DisplayName: entry.GetAttributeValue("displayName"),
		Email:       entry.GetAttributeValue("mail"),
		DN:          entry.DN,
	}, nil
}

// ListGroupMembers returns all users in the configured LDAP group.
func (s *LDAPService) ListGroupMembers(cfg config.LDAPConfig) ([]LDAPUser, error) {
	if cfg.GroupDN == "" {
		return nil, nil
	}

	conn, err := s.connect(cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return nil, fmt.Errorf("ldap bind: %w", err)
	}

	// Search for group members
	sr, err := conn.Search(ldap.NewSearchRequest(
		cfg.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 30, false,
		fmt.Sprintf("(&(objectClass=user)(memberOf=%s))", ldap.EscapeFilter(cfg.GroupDN)),
		[]string{"sAMAccountName", "displayName", "mail"}, nil,
	))
	if err != nil {
		return nil, fmt.Errorf("ldap search: %w", err)
	}

	var users []LDAPUser
	for _, e := range sr.Entries {
		users = append(users, LDAPUser{
			Username:    e.GetAttributeValue("sAMAccountName"),
			DisplayName: e.GetAttributeValue("displayName"),
			Email:       e.GetAttributeValue("mail"),
			DN:          e.DN,
		})
	}
	return users, nil
}

// TestConnection verifies LDAP connectivity and bind credentials.
func (s *LDAPService) TestConnection(cfg config.LDAPConfig) error {
	conn, err := s.connect(cfg)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer conn.Close()

	if err := conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return fmt.Errorf("bind failed: %w", err)
	}
	return nil
}

func (s *LDAPService) connect(cfg config.LDAPConfig) (*ldap.Conn, error) {
	if strings.HasPrefix(cfg.URL, "ldaps://") {
		return ldap.DialURL(cfg.URL, ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	}
	return ldap.DialURL(cfg.URL)
}
