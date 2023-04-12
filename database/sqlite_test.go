//go:build integration
// +build integration

package database

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func testSQLiteEnd() {
	_, _ = testPostgreSQL.conn.Exec(context.TODO(), "DROP TABLE IF EXISTS sessions")
	_ = testPostgreSQL.CloseConnection(context.TODO())
}

func TestDB_Add(t *testing.T) {
	validDate := time.Now().Add(5 * time.Minute).Round(time.Second)

	type fields struct {
		db *sql.DB
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
			fields: fields{db: testSQLite.db},
			args:   args{s: SessionData{Subject: "user", ExpiresAt: validDate, IDToken: "id_token", RefreshToken: "refresh_token", AccessToken: "access_token"}},
		},
		{
			name:    "empty_subject",
			fields:  fields{db: testSQLite.db},
			args:    args{s: SessionData{ExpiresAt: validDate, IDToken: "id_token", RefreshToken: "refresh_token", AccessToken: "access_token"}},
			wantErr: true,
		},
		{
			name:    "empty_id_token",
			fields:  fields{db: testSQLite.db},
			args:    args{s: SessionData{Subject: "user", ExpiresAt: validDate, RefreshToken: "refresh_token", AccessToken: "access_token"}},
			wantErr: true,
		},
		{
			name:    "empty_refresh_token",
			fields:  fields{db: testSQLite.db},
			args:    args{s: SessionData{Subject: "user", ExpiresAt: validDate, IDToken: "id_token", AccessToken: "access_token"}},
			wantErr: true,
		},
		{
			name:    "empty_access_token",
			fields:  fields{db: testSQLite.db},
			args:    args{s: SessionData{Subject: "user", ExpiresAt: validDate, IDToken: "id_token", RefreshToken: "refresh_token"}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &sqlite{
				db: tt.fields.db,
			}

			id, err := db.Add(context.TODO(), tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				verify, err := testSQLite.Get(context.TODO(), id)
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

func TestDB_Delete(t *testing.T) {
	id, err := testSQLite.Add(context.TODO(), SessionData{"subject_to_delete", "access_token", "refresh_token", "id_token", time.Now()})
	if err != nil {
		t.Fatalf("Add() reading added data, fatal error = %v, ", err)
		return
	}

	type fields struct {
		db *sql.DB
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
			fields: fields{db: testSQLite.db},
			args:   args{id: id},
		},
		{
			name:   "invalid_item",
			fields: fields{db: testSQLite.db},
			args:   args{id: "NOT_VALID"}, // no error expected from SQL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &sqlite{
				db: tt.fields.db,
			}
			if err := db.Delete(context.TODO(), tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDB_Get(t *testing.T) {
	validDate := time.Now().Add(5 * time.Minute).Round(time.Second)
	data := SessionData{"subject_to_get", "at_1234", "rt_1234", "it_1234", validDate}

	id, err := testSQLite.Add(context.TODO(), data)
	if err != nil {
		t.Fatalf("Add() reading added data, fatal error = %v, ", err)
		return
	}

	type fields struct {
		db *sql.DB
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
			fields: fields{db: testSQLite.db},
			args:   args{id: id},
			want:   data,
		},
		{
			name:    "notfound",
			fields:  fields{db: testSQLite.db},
			args:    args{id: "INVALID"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &sqlite{
				db: tt.fields.db,
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

func TestDB_Update(t *testing.T) {
	validDate := time.Now().Add(5 * time.Minute).Round(time.Second)
	oldData := SessionData{"subject_to_update", "at_9999", "rt_9999", "it_9999", validDate}

	id, err := testSQLite.Add(context.TODO(), oldData)
	if err != nil {
		t.Fatalf("Add() reading added data, fatal error = %v, ", err)
		return
	}

	newValidDate := time.Now().Add(10 * time.Minute).Round(time.Second)
	newData := SessionData{"subject_to_update", "at_1111", "rt_1111", "it_1111", newValidDate}
	newDataInvalidSubject := SessionData{"wrong_ubject", "at_1111", "rt_1111", "it_1111", newValidDate}

	type fields struct {
		db *sql.DB
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
			fields: fields{db: testSQLite.db},
			args:   args{id: id, s: newData},
		},
		{
			name:    "invalid_id",
			fields:  fields{db: testSQLite.db},
			args:    args{id: "NOT_VALID_ID", s: newData},
			wantErr: true,
		},
		{
			name:    "invalid_subject",
			fields:  fields{db: testSQLite.db},
			args:    args{id: id, s: newDataInvalidSubject},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &sqlite{
				db: tt.fields.db,
			}
			if err := db.Update(context.TODO(), tt.args.id, tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				verify, err := testSQLite.Get(context.TODO(), id)
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

func TestDB_Purge(t *testing.T) {
	t.Run("main", func(t *testing.T) {
		now := time.Now()
		expiredDate := now.Add(-5 * time.Minute).Round(time.Second)

		_, err := testSQLite.Add(context.TODO(), SessionData{"exp_01", "at_9999", "rt_9999", "it_9999", expiredDate})
		if err != nil {
			t.Fatalf("Add() reading added data, fatal error = %v, ", err)
			return
		}
		_, err = testSQLite.Add(context.TODO(), SessionData{"exp_02", "at_9999", "rt_9999", "it_9999", expiredDate})
		if err != nil {
			t.Fatalf("Add() reading added data, fatal error = %v, ", err)
			return
		}
		_, err = testSQLite.Add(context.TODO(), SessionData{"exp_03", "at_9999", "rt_9999", "it_9999", expiredDate})
		if err != nil {
			t.Fatalf("Add() reading added data, fatal error = %v, ", err)
			return
		}

		db := &sqlite{
			db: testSQLite.db,
		}

		var count int

		err = db.db.QueryRow("SELECT count(*) FROM sessions WHERE expires_at < unixepoch()").Scan(&count)
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

		err = db.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE expires_at < unixepoch()").Scan(&count)
		if err != nil {
			t.Fatalf("Purge() getting expired sissions, fatal error = %v, ", err)
			return
		}

		if count > 0 {
			t.Errorf("Purge() epired count error = %d", count)
		}
	})
}
