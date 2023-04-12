//go:build integration
// +build integration

package database

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func testPostgresqlEnd() {
	_, _ = testSQLite.db.Exec("DROP TABLE IF EXISTS sessions")
	_ = testSQLite.CloseConnection(context.TODO())
}

func TestPostgresql_Add(t *testing.T) {
	validDate := time.Now().Add(5 * time.Minute).Round(time.Second)

	type fields struct {
		conn *pgx.Conn
	}

	type args struct {
		s SessionData
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "correct_data",
			fields: fields{conn: testPostgreSQL.conn},
			args:   args{s: SessionData{Subject: "user", ExpiresAt: validDate, IDToken: "id_token", RefreshToken: "refresh_token", AccessToken: "access_token"}},
		},
		{
			name:    "empty_subject",
			fields:  fields{conn: testPostgreSQL.conn},
			args:    args{s: SessionData{ExpiresAt: validDate, IDToken: "id_token", RefreshToken: "refresh_token", AccessToken: "access_token"}},
			wantErr: true,
		},
		{
			name:    "empty_id_token",
			fields:  fields{conn: testPostgreSQL.conn},
			args:    args{s: SessionData{Subject: "user", ExpiresAt: validDate, RefreshToken: "refresh_token", AccessToken: "access_token"}},
			wantErr: true,
		},
		{
			name:    "empty_refresh_token",
			fields:  fields{conn: testPostgreSQL.conn},
			args:    args{s: SessionData{Subject: "user", ExpiresAt: validDate, IDToken: "id_token", AccessToken: "access_token"}},
			wantErr: true,
		},
		{
			name:    "empty_access_token",
			fields:  fields{conn: testPostgreSQL.conn},
			args:    args{s: SessionData{Subject: "user", ExpiresAt: validDate, IDToken: "id_token", RefreshToken: "refresh_token"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &postgresql{
				conn: tt.fields.conn,
			}

			id, err := db.Add(context.TODO(), tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				verify, err := testPostgreSQL.Get(context.TODO(), id)
				if err != nil {
					t.Fatalf("Add() reading added data, fatal error = %v, ", err)
					return
				}

				if verify != tt.args.s {
					t.Errorf("Add() error = %v, want %v", verify, tt.args.s)
					return
				}
			}
		})
	}
}

func TestPostgresql_Delete(t *testing.T) {
	id, err := testPostgreSQL.Add(context.TODO(), SessionData{"subject_to_delete", "access_token", "refresh_token", "id_token", time.Now()})
	if err != nil {
		t.Fatalf("Add() reading added data, fatal error = %v, ", err)
		return
	}

	type fields struct {
		conn *pgx.Conn
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "valid_item",
			fields: fields{conn: testPostgreSQL.conn},
			args:   args{id: id},
		},
		{
			name:   "invalid_item",
			fields: fields{conn: testPostgreSQL.conn},
			args:   args{id: "NOT_VALID"}, // no error expected from SQL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &postgresql{
				conn: tt.fields.conn,
			}
			if err := db.Delete(context.TODO(), tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPostgresql_Get(t *testing.T) {
	validDate := time.Now().Add(5 * time.Minute).Round(time.Second)
	data := SessionData{"subject_to_get", "at_1234", "rt_1234", "it_1234", validDate}

	id, err := testPostgreSQL.Add(context.TODO(), data)
	if err != nil {
		t.Fatalf("Add() reading added data, fatal error = %v, ", err)
		return
	}

	type fields struct {
		conn *pgx.Conn
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    SessionData
		wantErr bool
	}{
		{
			name:   "found",
			fields: fields{conn: testPostgreSQL.conn},
			args:   args{id: id},
			want:   data,
		},
		{
			name:    "notfound",
			fields:  fields{conn: testPostgreSQL.conn},
			args:    args{id: "INVALID"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &postgresql{
				conn: tt.fields.conn,
			}
			got, err := db.Get(context.TODO(), tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostgresql_Update(t *testing.T) {
	validDate := time.Now().Add(5 * time.Minute).Round(time.Second)
	oldData := SessionData{"subject_to_update", "at_9999", "rt_9999", "it_9999", validDate}

	id, err := testPostgreSQL.Add(context.TODO(), oldData)
	if err != nil {
		t.Fatalf("Add() reading added data, fatal error = %v, ", err)
		return
	}

	newValidDate := time.Now().Add(10 * time.Minute).Round(time.Second)
	newData := SessionData{"subject_to_update", "at_1111", "rt_1111", "it_1111", newValidDate}
	newDataInvalidSubject := SessionData{"wrong_ubject", "at_1111", "rt_1111", "it_1111", newValidDate}

	type fields struct {
		conn *pgx.Conn
	}
	type args struct {
		id string
		s  SessionData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "valid",
			fields: fields{conn: testPostgreSQL.conn},
			args:   args{id: id, s: newData},
		},
		{
			name:    "invalid_id",
			fields:  fields{conn: testPostgreSQL.conn},
			args:    args{id: "NOT_VALID_ID", s: newData},
			wantErr: true,
		},
		{
			name:    "invalid_subject",
			fields:  fields{conn: testPostgreSQL.conn},
			args:    args{id: id, s: newDataInvalidSubject},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &postgresql{
				conn: tt.fields.conn,
			}
			if err := db.Update(context.TODO(), tt.args.id, tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				verify, err := testPostgreSQL.Get(context.TODO(), id)
				if err != nil {
					t.Fatalf("Add() reading added data, fatal error = %v, ", err)
					return
				}

				if verify != tt.args.s {
					t.Errorf("Add() error = %v, want %v", verify, tt.args.s)
					return
				}
			}
		})
	}
}

func TestPostgresql_Purge(t *testing.T) {
	t.Run("main", func(t *testing.T) {
		now := time.Now()
		expiredDate := now.Add(-5 * time.Minute).Round(time.Second)

		_, err := testPostgreSQL.Add(context.TODO(), SessionData{"exp_01", "at_9999", "rt_9999", "it_9999", expiredDate})
		if err != nil {
			t.Fatalf("Add() reading added data, fatal error = %v, ", err)
			return
		}
		_, err = testPostgreSQL.Add(context.TODO(), SessionData{"exp_02", "at_9999", "rt_9999", "it_9999", expiredDate})
		if err != nil {
			t.Fatalf("Add() reading added data, fatal error = %v, ", err)
			return
		}
		_, err = testPostgreSQL.Add(context.TODO(), SessionData{"exp_03", "at_9999", "rt_9999", "it_9999", expiredDate})
		if err != nil {
			t.Fatalf("Add() reading added data, fatal error = %v, ", err)
			return
		}

		db := &postgresql{
			conn: testPostgreSQL.conn,
		}

		var count int

		err = db.conn.QueryRow(context.TODO(), "SELECT count(*) FROM sessions WHERE expires_at < extract(epoch from now())").Scan(&count)
		if err != nil {
			t.Fatalf("Purge() getting expired sissions, fatal error = %v, ", err)
			return
		}

		if count < 3 {
			t.Errorf("Purge() epired count error = %d", count)
		}

		if err := db.Purge(context.TODO()); err != nil {
			t.Errorf("Purge() error = %v", err)
		}

		err = db.conn.QueryRow(context.TODO(), "SELECT COUNT(*) FROM SESSIONS WHERE expires_at < extract(epoch from now())").Scan(&count)
		if err != nil {
			t.Fatalf("Purge() getting expired sissions, fatal error = %v, ", err)
			return
		}

		if count > 0 {
			t.Errorf("Purge() epired count error = %d", count)
		}
	})
}
