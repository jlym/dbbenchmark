package postgres

import "fmt"

var DevConnStringOptions = &ConnStringOptions{
	Host:     "localhost",
	Port:     5432,
	UserName: "postgres",
	Password: "password",
}

type ConnStringOptions struct {
	Host     string
	Port     int
	UserName string
	Password string
}

func (opt *ConnStringOptions) GetConnString(dbName string) string {
	return opt.getConnString(dbName, false)
}

func (opt *ConnStringOptions) GetDebugConnString(dbName string) string {
	return opt.getConnString(dbName, true)
}

func (opt *ConnStringOptions) getConnString(dbName string, hidePassword bool) string {
	password := opt.Password
	if hidePassword && opt.Password != "" {
		password = "***"
	}

	return fmt.Sprintf(
		"postgres://%s:%d/%s?user=postgres&password=%s",
		opt.Host,
		opt.Port,
		dbName,
		password,
	)
}
