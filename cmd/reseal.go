package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/models/tenant"
	"github.com/glueops/apiary/internal/utils"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// The key purposes as they were BEFORE the 2026-08-23 rename. They exist here
// and nowhere else, deliberately: they are migration input, not a derivation
// anything new should reach for. Putting them in internal/utils beside the live
// constants would make picking the wrong one a plausible mistake.
//
// These strings are HKDF info parameters, so they ARE the old keys. Do not
// "tidy" them.
const (
	legacyPurposeTOTPSeed  utils.KeyPurpose = "eyrie/mfa/totp-seed/v1"
	legacyPurposeTenantDEK utils.KeyPurpose = "eyrie/tenant/dek/v1"
	legacyPurposeTenantDSN utils.KeyPurpose = "eyrie/tenant/dsn/v1"
)

var (
	resealApply bool
)

var resealCmd = &cobra.Command{
	Use:   "reseal-keys",
	Short: "Re-wrap stored secrets after a key-purpose change",
	Long: `Decrypts under the previous key purposes and re-encrypts under the current ones.

Renaming a KeyPurpose re-derives every key in the system, because the purpose
string is the HKDF info parameter. Nothing errors when that happens — the data
simply stops opening. This walks the three places affected:

  organizations.connection_string   the tenant DSN        (master -> dsn key)
  organizations.encrypted_tenant_key the wrapped tenant key (master -> dek key)
  <tenant>.credentials              sealed provider secrets (tenant key -> dek)

It is IDEMPOTENT and safe to re-run: each value is tried under the old
derivation first, and a value that only opens under the new one is already
migrated and left alone. It therefore also works as a checker.

Dry run by default. Pass --apply to write.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		cfg, err := config.Load()
		if err != nil {
			return err
		}
		master, err := utils.MasterKeyFromBase64(cfg.EncryptionMasterKey)
		if err != nil {
			return fmt.Errorf("reseal: %w", err)
		}

		keys, err := deriveResealKeys(master)
		if err != nil {
			return err
		}

		db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return fmt.Errorf("reseal: open control database: %w", err)
		}

		var orgs []control.Organization
		if err := db.WithContext(ctx).Find(&orgs).Error; err != nil {
			return fmt.Errorf("reseal: list organizations: %w", err)
		}

		if !resealApply {
			cmd.Println("DRY RUN — nothing will be written. Re-run with --apply to migrate.")
		}
		cmd.Printf("%d organization(s) to inspect\n\n", len(orgs))

		var summary resealSummary
		for i := range orgs {
			if err := resealOrg(ctx, cmd, db, &orgs[i], keys, &summary); err != nil {
				// One organization failing must not abandon the rest: a partial
				// migration that reports precisely which tenants are still broken
				// is far more useful than an abort at the first bad row.
				cmd.Printf("  %s: FAILED — %v\n", orgs[i].Name, err)
				summary.failed++
			}
		}

		cmd.Printf("\ndsn: %d rewrapped, %d already current\n", summary.dsnRewrapped, summary.dsnCurrent)
		cmd.Printf("tenant keys: %d rewrapped, %d already current\n", summary.keyRewrapped, summary.keyCurrent)
		cmd.Printf("credentials: %d resealed, %d already current\n", summary.credRewrapped, summary.credCurrent)
		if summary.failed > 0 {
			return fmt.Errorf("%d organization(s) could not be migrated", summary.failed)
		}
		return nil
	},
}

type resealKeys struct {
	oldDSN, newDSN []byte
	oldDEK, newDEK []byte
}

func deriveResealKeys(master []byte) (resealKeys, error) {
	var k resealKeys
	var err error
	if k.oldDSN, err = utils.DeriveKey(master, legacyPurposeTenantDSN); err != nil {
		return k, err
	}
	if k.newDSN, err = utils.DeriveKey(master, utils.KeyPurposeTenantDSN); err != nil {
		return k, err
	}
	if k.oldDEK, err = utils.DeriveKey(master, legacyPurposeTenantDEK); err != nil {
		return k, err
	}
	if k.newDEK, err = utils.DeriveKey(master, utils.KeyPurposeTenantDEK); err != nil {
		return k, err
	}
	return k, nil
}

type resealSummary struct {
	dsnRewrapped, dsnCurrent   int
	keyRewrapped, keyCurrent   int
	credRewrapped, credCurrent int
	failed                     int
}

// openEither decrypts under the old key, falling back to the new one.
//
// The fallback is what makes this idempotent AND a checker: a value that only
// opens under the new derivation has already been migrated, and one that opens
// under neither is genuinely lost rather than merely stale — a distinction worth
// reporting differently, because the second means the master key changed too.
func openEither(ct, iv, tag, oldKey, newKey, aad []byte) (plain []byte, alreadyCurrent bool, err error) {
	if p, e := utils.DecryptAESGCMWithAAD(ct, oldKey, iv, tag, aad); e == nil {
		return p, false, nil
	}
	if p, e := utils.DecryptAESGCMWithAAD(ct, newKey, iv, tag, aad); e == nil {
		return p, true, nil
	}
	return nil, false, fmt.Errorf("opens under neither the previous nor the current derivation; " +
		"the master key itself may have changed, and this value cannot be recovered from here")
}

func resealOrg(
	ctx context.Context, cmd *cobra.Command, db *gorm.DB,
	org *control.Organization, keys resealKeys, sum *resealSummary,
) error {
	aad := []byte(org.ID.String())

	// --- the wrapped tenant key -------------------------------------------
	if org.EncryptedTenantKey == "" {
		cmd.Printf("  %s: no tenant key; never provisioned, skipping\n", org.Name)
		return nil
	}
	ct, iv, tag, err := decodeTriplet(org.EncryptedTenantKey, org.TenantKeyIV, org.TenantKeyTag)
	if err != nil {
		return fmt.Errorf("tenant key: %w", err)
	}
	tenantKey, keyCurrent, err := openEither(ct, iv, tag, keys.oldDEK, keys.newDEK, aad)
	if err != nil {
		return fmt.Errorf("tenant key: %w", err)
	}

	// --- the DSN -----------------------------------------------------------
	dsnCT, dsnIV, dsnTag, err := decodeTriplet(org.ConnectionString, org.ConnectionStringIV, org.ConnectionStringTag)
	if err != nil {
		return fmt.Errorf("connection string: %w", err)
	}
	dsn, dsnCurrent, err := openEither(dsnCT, dsnIV, dsnTag, keys.oldDSN, keys.newDSN, aad)
	if err != nil {
		return fmt.Errorf("connection string: %w", err)
	}

	updates := map[string]any{}
	if !keyCurrent {
		nct, niv, ntag, err := utils.EncryptAESGCMWithAAD(tenantKey, keys.newDEK, aad)
		if err != nil {
			return fmt.Errorf("re-wrap tenant key: %w", err)
		}
		updates["encrypted_tenant_key"] = utils.EncodeB64(nct)
		updates["tenant_key_iv"] = utils.EncodeB64(niv)
		updates["tenant_key_tag"] = utils.EncodeB64(ntag)
		sum.keyRewrapped++
	} else {
		sum.keyCurrent++
	}
	if !dsnCurrent {
		nct, niv, ntag, err := utils.EncryptAESGCMWithAAD(dsn, keys.newDSN, aad)
		if err != nil {
			return fmt.Errorf("re-wrap connection string: %w", err)
		}
		updates["connection_string"] = utils.EncodeB64(nct)
		updates["connection_string_iv"] = utils.EncodeB64(niv)
		updates["connection_string_tag"] = utils.EncodeB64(ntag)
		sum.dsnRewrapped++
	} else {
		sum.dsnCurrent++
	}

	if len(updates) > 0 && resealApply {
		if err := db.WithContext(ctx).Model(&control.Organization{}).
			Where("id = ?", org.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("write re-wrapped org secrets: %w", err)
		}
	}

	// --- the tenant's own credentials --------------------------------------
	//
	// The tenant key itself is unchanged — it is a random value, only its
	// wrapping moved. But the DEK derived FROM it uses the purpose too, so every
	// credential sealed under it has to be re-sealed as well.
	oldDEK, err := utils.DeriveKey(tenantKey, legacyPurposeTenantDEK)
	if err != nil {
		return fmt.Errorf("derive previous tenant dek: %w", err)
	}
	newDEK, err := utils.DeriveKey(tenantKey, utils.KeyPurposeTenantDEK)
	if err != nil {
		return fmt.Errorf("derive current tenant dek: %w", err)
	}

	n, current, err := resealCredentials(ctx, string(dsn), org.ID, oldDEK, newDEK)
	if err != nil {
		return fmt.Errorf("credentials: %w", err)
	}
	sum.credRewrapped += n
	sum.credCurrent += current

	cmd.Printf("  %s: dsn=%s tenant-key=%s credentials=%d resealed, %d current\n",
		org.Name, state(dsnCurrent), state(keyCurrent), n, current)
	return nil
}

func state(alreadyCurrent bool) string {
	if alreadyCurrent {
		return "current"
	}
	return "rewrapped"
}

func resealCredentials(ctx context.Context, dsn string, orgID uuid.UUID, oldDEK, newDEK []byte) (int, int, error) {
	tdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return 0, 0, fmt.Errorf("connect to tenant database: %w", err)
	}
	defer func() {
		if sqlDB, err := tdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	var creds []tenant.Credential
	if err := tdb.WithContext(ctx).Find(&creds).Error; err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			// Provisioned but never migrated. Nothing sealed, nothing to do.
			return 0, 0, nil
		}
		return 0, 0, err
	}

	rewrapped, current := 0, 0
	for i := range creds {
		c := &creds[i]
		aad := utils.AAD("credentials", orgID, c.ID)
		ct, iv, tag, err := decodeTriplet(c.Ciphertext, c.IV, c.Tag)
		if err != nil {
			return rewrapped, current, fmt.Errorf("credential %s: %w", c.ID, err)
		}
		plain, alreadyCurrent, err := openEither(ct, iv, tag, oldDEK, newDEK, aad)
		if err != nil {
			return rewrapped, current, fmt.Errorf("credential %s: %w", c.ID, err)
		}
		if alreadyCurrent {
			current++
			continue
		}
		if !resealApply {
			rewrapped++
			continue
		}

		sealed, err := utils.Seal(plain, c.KeyID, newDEK, aad)
		if err != nil {
			return rewrapped, current, fmt.Errorf("credential %s: %w", c.ID, err)
		}
		if err := tdb.WithContext(ctx).Model(&tenant.Credential{}).
			Where("id = ?", c.ID).
			Updates(map[string]any{
				"ciphertext": sealed.Ciphertext,
				"iv":         sealed.IV,
				"tag":        sealed.Tag,
				"updated_at": time.Now().UTC(),
			}).Error; err != nil {
			return rewrapped, current, fmt.Errorf("credential %s: %w", c.ID, err)
		}
		rewrapped++
	}
	return rewrapped, current, nil
}

func decodeTriplet(ctB64, ivB64, tagB64 string) (ct, iv, tag []byte, err error) {
	var e1, e2, e3 error
	ct, e1 = utils.DecodeB64(ctB64)
	iv, e2 = utils.DecodeB64(ivB64)
	tag, e3 = utils.DecodeB64(tagB64)
	if e1 != nil || e2 != nil || e3 != nil {
		return nil, nil, nil, fmt.Errorf("stored value is not valid base64")
	}
	return ct, iv, tag, nil
}

func init() {
	resealCmd.Flags().BoolVar(&resealApply, "apply", false,
		"Actually write. Without it the command reports what it would do and changes nothing.")
	rootCmd.AddCommand(resealCmd)
}
