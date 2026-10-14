package config

import (
	"fmt"
	"net/url"
	"strings"
)

// DatabaseConfig describes the Postgres connection.
type DatabaseConfig struct {
	URL      string
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN returns DATABASE_URL when set, otherwise assembles one from parts.
func (d DatabaseConfig) DSN() string {
	if strings.TrimSpace(d.URL) != "" {
		return d.URL
	}
	ssl := d.SSLMode
	if ssl == "" {
		ssl = "disable"
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		d.Host, d.User, d.Password, d.Name, d.Port, ssl)
}

// Redacted returns a log-safe description of the target database.
func (d DatabaseConfig) Redacted() string {
	if d.URL != "" {
		if u, err := url.Parse(d.URL); err == nil && u.Host != "" {
			return u.Host + "/" + strings.TrimPrefix(u.Path, "/")
		}
		return "(DATABASE_URL)"
	}
	return d.Host + ":" + d.Port + "/" + d.Name
}
