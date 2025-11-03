package repository

import (
	"app/config"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type Repository interface {
	categories
	pets
}

func NewRepository(cfg config.Database) (Repository, error) {
	result := &repository{
		cfg: cfg,
	}
	return result, result.open()
}

type repository struct {
	cfg config.Database
	db  *sql.DB
}

func (r *repository) open() (err error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true&multiStatements=true",
		r.cfg.Username, r.cfg.Password, r.cfg.Host, r.cfg.Port, r.cfg.Name)
	if r.db, err = sql.Open("mysql", dsn); err == nil {
		err = r.db.Ping()
	}
	return err
}
