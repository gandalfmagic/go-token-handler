package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/gandalfmagic/go-token-handler/opentelemetry"

	"github.com/gandalfmagic/encryption"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	querySQLiteCreate = `CREATE TABLE IF NOT EXISTS sessions (
		session_id TEXT NOT NULL PRIMARY KEY,
		subject TEXT NOT NULL CHECK(subject != ''),
		access_token TEXT NOT NULL CHECK(access_token != ''),
		refresh_token TEXT NOT NULL CHECK(refresh_token != ''),
		id_token TEXT NOT NULL CHECK(id_token != ''),
		expires_at INTEGER NOT NULL);
		CREATE INDEX IF NOT EXISTS session_subject ON sessions (subject);
		CREATE INDEX IF NOT EXISTS session_expires_at ON sessions (expires_at);`
	querySQLiteDelete = `DELETE FROM sessions WHERE session_id = ?`
	querySQLiteInsert = `INSERT INTO sessions (session_id, subject, access_token, refresh_token, id_token, expires_at) VALUES (?, ?, ?, ?, ?, ?)`
	querySQLiteSelect = `SELECT subject, access_token, refresh_token, id_token, expires_at FROM sessions WHERE session_id = ?`
	querySQLiteUpdate = `UPDATE sessions SET subject = ?, access_token = ?, refresh_token = ?, id_token = ?, expires_at = ? WHERE session_id = ?`
	querySQLitePurge  = `DELETE FROM sessions WHERE expires_at < ?`
)

type sqlite struct {
	db     *sql.DB
	cipher encryption.HexCipher
}

var (
	sqliteInstance *sqlite
	sqliteOnce     sync.Once
)

func NewSQLiteSessionImpl(_ context.Context, cipher encryption.HexCipher, database string) (SessionImpl, error) {
	var db *sql.DB
	var err error

	sqliteOnce.Do(func() {
		db, err = sql.Open("sqlite3", fmt.Sprintf("file:%s", database))
		if err != nil {
			return
		}

		_, err = db.Exec(querySQLiteCreate)
		if err != nil {
			return
		}

		sqliteInstance = &sqlite{db: db, cipher: cipher}
	})
	if err != nil {
		return nil, err
	}

	return sqliteInstance, err
}

func (db *sqlite) CloseConnection(_ context.Context) error {
	return db.db.Close()
}

func (db *sqlite) Add(ctx context.Context, s SessionData) (string, error) {
	_, span := opentelemetry.NewSpanFromContext(ctx, "session.postgresql: INSERT")
	if span != nil {
		defer span.End()
	}

	id := uuid.New().String()

	if span != nil {
		span.SetAttributes(
			attribute.String("db.table", "sessions"),
			attribute.String("db.table.id", id),
			attribute.String("db.table.subject", s.Subject),
			attribute.String("db.table.expires_at", s.ExpiresAt.String()))
	}

	enc, err := encryptArgs(db.cipher, s)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.sqlite: INSERT -> encryptArgs")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}

		return "", err
	}

	if _, err := db.db.Exec(querySQLiteInsert, id, enc.Subject, enc.AccessToken, enc.RefreshToken, enc.IDToken, enc.ExpiresAt.Unix()); err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.sqlite: INSERT -> db.db.Exec")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}

		return "", err
	}

	return id, nil
}

func (db *sqlite) Delete(ctx context.Context, id string) error {
	_, span := opentelemetry.NewSpanFromContext(ctx, "session.postgresql: INSERT")
	if span != nil {
		defer span.End()

		span.SetAttributes(
			attribute.String("db.table", "sessions"),
			attribute.String("db.table.id", id))
	}

	_, err := db.db.Exec(querySQLiteDelete, id)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.sqlite: DELETE -> db.conn.Exec")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}

		return err
	}

	return nil
}

func (db *sqlite) Get(ctx context.Context, id string) (SessionData, error) {
	_, span := opentelemetry.NewSpanFromContext(ctx, "session.postgresql: INSERT")
	if span != nil {
		defer span.End()

		span.SetAttributes(
			attribute.String("db.table", "sessions"),
			attribute.String("db.table.id", id))
	}

	s := SessionData{}
	var expiresAt int64

	if err := db.db.QueryRow(querySQLiteSelect, id).Scan(&s.Subject, &s.AccessToken, &s.RefreshToken, &s.IDToken, &expiresAt); err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.sqlite: INSERT -> db.conn.QueryRow")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}

		return SessionData{}, err
	}

	s.ExpiresAt = time.Unix(expiresAt, 0)

	return decryptArgs(db.cipher, s)
}

func (db *sqlite) Update(ctx context.Context, id string, s SessionData) error {
	_, span := opentelemetry.NewSpanFromContext(ctx, "session.postgresql: INSERT")
	if span != nil {
		defer span.End()

		span.SetAttributes(
			attribute.String("db.table", "sessions"),
			attribute.String("db.table.id", id),
			attribute.String("db.table.subject", s.Subject),
			attribute.String("db.table.expires_at", s.ExpiresAt.String()))
	}

	oldSession, err := db.Get(context.TODO(), id)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.sqlite: UPDATE -> Get")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}

		return err
	}

	if s.Subject != oldSession.Subject {
		if span != nil {
			span.SetStatus(codes.Error, "session.sqlite: subject mismatch")
		}

		return ErrSessionsMismatch
	}

	enc, err := encryptArgs(db.cipher, s)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.sqlite: UPDATE -> encryptArgs")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}

		return err
	}

	_, err = db.db.Exec(querySQLiteUpdate, enc.Subject, enc.AccessToken, enc.RefreshToken, enc.IDToken, enc.ExpiresAt.Unix(), id)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.sqlite: UPDATE -> db.conn.Exec")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}

		return err
	}

	return nil
}

func (db *sqlite) Purge(_ context.Context) error {
	now := time.Now().Unix()

	if _, err := db.db.Exec(querySQLitePurge, now); err != nil {
		return err
	}

	return nil
}
