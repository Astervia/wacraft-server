package migrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
	"github.com/pterm/pterm"
)

func init() {
	goose.AddMigrationContext(upMigratePlanPrices, downMigratePlanPrices)
}

type legacyPlanRow struct {
	planID          string
	currency        string
	priceCents      int64
	stripePriceID   *string
	stripeProductID *string
}

// upMigratePlanPrices copies the legacy currency/price_cents/stripe fields from
// the plans table into the new plan_prices table. Each migrated row is marked as
// the default price (is_default=true) for its plan.
func upMigratePlanPrices(ctx context.Context, tx *sql.Tx) error {
	pterm.DefaultLogger.Info("Migrating plan prices from plans table to plan_prices table...")

	// Check that the legacy columns still exist before attempting the migration.
	var currencyExists bool
	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'plans' AND column_name = 'currency'
		)
	`).Scan(&currencyExists)
	if err != nil {
		return err
	}
	if !currencyExists {
		pterm.DefaultLogger.Info("Legacy currency column not found on plans, skipping migration")
		return nil
	}

	// Collect all rows first — the result set must be fully consumed and closed
	// before issuing further queries on the same transaction connection.
	rows, err := tx.QueryContext(ctx, `
		SELECT id, currency, price_cents, stripe_price_id, stripe_product_id
		FROM plans
		WHERE currency IS NOT NULL AND currency != ''
	`)
	if err != nil {
		return err
	}

	var plans []legacyPlanRow
	for rows.Next() {
		var r legacyPlanRow
		if err := rows.Scan(&r.planID, &r.currency, &r.priceCents, &r.stripePriceID, &r.stripeProductID); err != nil {
			rows.Close()
			return err
		}
		plans = append(plans, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// Now process each plan row with the result set closed.
	migrated := 0
	for _, r := range plans {
		// Skip if a price entry already exists for this plan (idempotent).
		var count int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM plan_prices WHERE plan_id = $1`, r.planID,
		).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO plan_prices (id, plan_id, currency, price_cents, is_default, stripe_price_id, stripe_product_id, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, true, $4, $5, NOW(), NOW())
		`, r.planID, r.currency, r.priceCents, r.stripePriceID, r.stripeProductID); err != nil {
			return err
		}
		migrated++
	}

	pterm.DefaultLogger.Info("Plan price migration complete", pterm.DefaultLogger.Args("migrated", migrated))
	return nil
}

// downMigratePlanPrices removes the plan_prices rows that were created by the up migration.
func downMigratePlanPrices(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM plan_prices`)
	return err
}
