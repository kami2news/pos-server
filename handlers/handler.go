package handlers

import "github.com/jmoiron/sqlx"

type Handler struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

// nilableString は空文字列をnilに変換（DB INSERT用）
func nilableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
