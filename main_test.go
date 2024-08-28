package main

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/gorm"
)

// GORM_REPO: https://github.com/go-gorm/gorm.git
// GORM_BRANCH: master
// TEST_DRIVERS: sqlite, mysql, postgres, sqlserver

func TestGORM(t *testing.T) {
	var user User
	err := DB.First(&user, "name = ?", "jinzhu").Error
	// ensure we start with no user
	if err == nil {
		t.Errorf("User %d already exists", user.ID)
	}

	// Create a wrapping transaction
	err = Transaction(context.Background(), DB, func(ctx context.Context, tx *gorm.DB) error {
		// Within this, create a transaction that retrieves a user, and then
		// creates another transaction that retrieves the account, but errors,
		// and bubble that error up.
		err := Transaction(ctx, tx, func(ctx context.Context, tx *gorm.DB) error {
			user := User{Name: "jinzhu"}
			var account Account
			err := tx.Create(&user).Error
			if err != nil {
				return err
			}
			fmt.Printf("User created: %d", user.ID)

			// Since we propagate the error, we'll now also try to
			// rollback this nested transaction, using the same
			// savepoint ID, since the `Transaction` helper
			// creates a single closure that always has the
			// same address.
			return Transaction(ctx, tx, func(ctx context.Context, tx *gorm.DB) error {
				// We haven't created an account, so we return an error, which does
				// a rollback of the inner transaction using a savepoint.
				err := tx.First(&account, 1).Error
				if err != nil {
					return err
				}

				return nil
			})
		})
		if err == nil {
			t.Errorf("Expected an error, got none")
		}
		// We discard the inner transaction error, which allows us to commit this outer
		// transaction (which does nothing), even if the inner fails.
		return nil
	})

	// Since we have rolled back the inner transaction, we expect that
	// no user was created, since we should have rolled that back.
	err = DB.First(&user, "name = ?", "jinzhu").Error
	if err == nil {
		t.Errorf("User %d was created, despite erroring inside the transaction", user.ID)
	}
}
