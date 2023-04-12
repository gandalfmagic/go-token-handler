package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gandalfmagic/go-token-handler/opentelemetry"

	"github.com/gandalfmagic/encryption"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	queryPostgresqlCreate = `CREATE TABLE IF NOT EXISTS public.sessions (
		session_id varchar(36) NOT NULL,
		subject varchar NOT NULL,
		access_token varchar NOT NULL,
		refresh_token varchar NOT NULL,
		id_token varchar NOT NULL,
		expires_at integer NOT NULL,
		CONSTRAINT sessions_pkey PRIMARY KEY (session_id),
		CONSTRAINT sessions_subject_check CHECK (subject != ''),
		CONSTRAINT sessions_access_token_check CHECK (access_token != ''),
		CONSTRAINT sessions_refresh_token_check CHECK (refresh_token != ''),
		CONSTRAINT sessions_id_token CHECK (id_token != ''));`
	queryPostgresqlDelete = `DELETE FROM sessions WHERE session_id = $1`
	queryPostgresqlInsert = `INSERT INTO sessions (session_id, subject, access_token, refresh_token, id_token, expires_at) VALUES ($1, $2, $3, $4, $5, $6)`
	queryPostgresqlSelect = `SELECT subject, access_token, refresh_token, id_token, expires_at FROM sessions WHERE session_id = $1`
	queryPostgresqlUpdate = `UPDATE sessions SET subject = $1, access_token = $2, refresh_token = $3, id_token = $4, expires_at = $5 WHERE session_id = $6`
	queryPostgresqlPurge  = `DELETE FROM sessions WHERE expires_at < $1`
)

type postgresql struct {
	conn   *pgx.Conn
	cipher encryption.HexCipher
}

var (
	pgInstance *postgresql
	pgOnce     sync.Once
)

func NewPostgresqlSessionImpl(ctx context.Context, cipher encryption.HexCipher, host, database, username, password string) (SessionImpl, error) {
	var conn *pgx.Conn
	var err error

	pgOnce.Do(func() {
		connStr := fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=disable", username, password, host, database)
		conn, err = pgx.Connect(ctx, connStr)
		if err != nil {
			return
		}

		_, err = conn.Exec(context.TODO(), queryPostgresqlCreate)
		if err != nil {
			return
		}

		pgInstance = &postgresql{conn: conn, cipher: cipher}
	})
	if err != nil {
		return nil, err
	}

	return pgInstance, err
}

func (db *postgresql) CloseConnection(ctx context.Context) error {
	return db.conn.Close(ctx)
}

func (db *postgresql) Add(ctx context.Context, s SessionData) (string, error) {
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
			span.SetStatus(codes.Error, "session.postgresql: INSERT -> encryptArgs")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}
		return "", err
	}

	if _, err = db.conn.Exec(ctx, queryPostgresqlInsert, id, enc.Subject, enc.AccessToken, enc.RefreshToken, enc.IDToken, enc.ExpiresAt.Unix()); err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.postgresql: INSERT -> db.conn.Exec")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}
		return "", err
	}

	return id, nil
}

func (db *postgresql) Delete(ctx context.Context, id string) error {
	_, span := opentelemetry.NewSpanFromContext(ctx, "session.postgresql: INSERT")
	if span != nil {
		defer span.End()

		span.SetAttributes(
			attribute.String("db.table", "sessions"),
			attribute.String("db.table.id", id))
	}

	_, err := db.conn.Exec(ctx, queryPostgresqlDelete, id)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.postgresql: DELETE -> db.conn.Exec")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}

		return err
	}

	return nil
}

func (db *postgresql) Get(ctx context.Context, id string) (SessionData, error) {
	_, span := opentelemetry.NewSpanFromContext(ctx, "session.postgresql: INSERT")
	if span != nil {
		defer span.End()

		span.SetAttributes(
			attribute.String("db.table", "sessions"),
			attribute.String("db.table.id", id))
	}

	s := SessionData{}
	var expiresAt int64

	if err := db.conn.QueryRow(ctx, queryPostgresqlSelect, id).Scan(&s.Subject, &s.AccessToken, &s.RefreshToken, &s.IDToken, &expiresAt); err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.postgresql: INSERT -> db.conn.QueryRow")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}
		return SessionData{}, err
	}

	s.ExpiresAt = time.Unix(expiresAt, 0)

	return decryptArgs(db.cipher, s)
}

func (db *postgresql) Update(ctx context.Context, id string, s SessionData) error {
	_, span := opentelemetry.NewSpanFromContext(ctx, "session.postgresql: INSERT")
	if span != nil {
		defer span.End()

		span.SetAttributes(
			attribute.String("db.table", "sessions"),
			attribute.String("db.table.id", id),
			attribute.String("db.table.subject", s.Subject),
			attribute.String("db.table.expires_at", s.ExpiresAt.String()))
	}

	oldSession, err := db.Get(ctx, id)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.postgresql: UPDATE -> Get")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}

		return err
	}

	if s.Subject != oldSession.Subject {
		if span != nil {
			span.SetStatus(codes.Error, "session.postgresql: subject mismatch")
		}

		return ErrSessionsMismatch
	}

	enc, err := encryptArgs(db.cipher, s)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.postgresql: UPDATE -> encryptArgs")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}

		return err
	}

	_, err = db.conn.Exec(ctx, queryPostgresqlUpdate, enc.Subject, enc.AccessToken, enc.RefreshToken, enc.IDToken, enc.ExpiresAt.Unix(), id)
	if err != nil {
		if span != nil {
			span.SetStatus(codes.Error, "session.postgresql: UPDATE -> db.conn.Exec")
			span.SetAttributes(attribute.String("error.message", err.Error()))
		}

		return err
	}

	return nil
}

func (db *postgresql) Purge(ctx context.Context) error {
	now := time.Now().Unix()

	if _, err := db.conn.Exec(ctx, queryPostgresqlPurge, now); err != nil {
		return err
	}

	return nil
}
