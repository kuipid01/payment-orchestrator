package middleware

import "github.com/kuipid01/payconnect/internal/platform/gormx"

func gormxNewID() string { return gormx.NewID("log") }
