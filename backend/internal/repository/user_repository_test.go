package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapUserUniqueViolation(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		want       error
		wantMapped bool
	}{
		{
			name:       "phone",
			err:        &pgconn.PgError{Code: "23505", ConstraintName: "users_phone_key"},
			want:       ErrPhoneExists,
			wantMapped: true,
		},
		{
			name:       "email",
			err:        &pgconn.PgError{Code: "23505", ConstraintName: "users_email_key"},
			want:       ErrEmailExists,
			wantMapped: true,
		},
		{
			name: "other unique constraint",
			err:  &pgconn.PgError{Code: "23505", ConstraintName: "some_other_key"},
		},
		{
			name: "different postgres error",
			err:  &pgconn.PgError{Code: "23503", ConstraintName: "users_phone_key"},
		},
		{
			name: "non-postgres error",
			err:  errors.New("network unavailable"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := mapUserUniqueViolation(test.err)
			if test.wantMapped {
				if !errors.Is(got, test.want) {
					t.Fatalf("mapped error = %v, want %v", got, test.want)
				}
				return
			}
			if got != test.err {
				t.Fatalf("error = %v, want original %v", got, test.err)
			}
		})
	}
}

func TestCreateRejectsNilUser(t *testing.T) {
	repo := NewUserRepository(nil)

	if _, err := repo.Create(nil); err == nil {
		t.Fatal("expected nil user to be rejected")
	}
}
