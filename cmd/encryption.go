package cmd

import (
	"fmt"
	"log"

	"github.com/glueops/apiary/internal/config"
	"github.com/glueops/apiary/internal/models/control"
	"github.com/glueops/apiary/internal/utils"
	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt",
	Short: "Manage encryption keys",
}

var generateMasterCmd = &cobra.Command{
	Use:   "generate-master",
	Short: "Generate a new random 32-byte master key (for initial setup)",
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := utils.RandomBytes(32)
		if err != nil {
			return err
		}
		fmt.Printf("ENCRYPTION_MASTER_KEY=%s\n", utils.EncodeB64(key))
		return nil
	},
}

var rotateMasterCmd = &cobra.Command{
	Use:   "rotate-master",
	Short: "Rotate the master encryption key (KEK)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		oldMasterKey, err := utils.MasterKeyFromBase64(cfg.EncryptionMasterKey)
		if err != nil {
			return fmt.Errorf("invalid current master key in config: %w", err)
		}

		oldDEKKey, oldDSNKey, err := tenantSubkeys(oldMasterKey)
		if err != nil {
			return err
		}

		log.Println("Generating new Master Key...")
		newMasterKey, err := utils.RandomBytes(32)
		if err != nil {
			return err
		}
		newDEKKey, newDSNKey, err := tenantSubkeys(newMasterKey)
		if err != nil {
			return err
		}

		controlDB, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
		if err != nil {
			return err
		}

		err = controlDB.Transaction(func(tx *gorm.DB) error {
			var orgs []control.Organization
			if err := tx.Find(&orgs).Error; err != nil {
				return err
			}

			log.Printf("Found %d organizations. Rotating wrapped keys...", len(orgs))

			for _, org := range orgs {
				if org.EncryptedTenantKey == "" {
					continue
				}

				ct, _ := utils.DecodeB64(org.EncryptedTenantKey)
				iv, _ := utils.DecodeB64(org.TenantKeyIV)
				tag, _ := utils.DecodeB64(org.TenantKeyTag)

				aad := []byte(org.ID.String())

				tenantKey, err := utils.DecryptAESGCMWithAAD(ct, oldDEKKey, iv, tag, aad)
				if err != nil {
					return fmt.Errorf("failed to decrypt tenant key for org %s: %w", org.ID, err)
				}

				newCt, newIv, newTag, err := utils.EncryptAESGCMWithAAD(tenantKey, newDEKKey, aad)
				if err != nil {
					return fmt.Errorf("failed to encrypt tenant key for org %s: %w", org.ID, err)
				}

				updates := map[string]interface{}{
					"encrypted_tenant_key": utils.EncodeB64(newCt),
					"tenant_key_iv":        utils.EncodeB64(newIv),
					"tenant_key_tag":       utils.EncodeB64(newTag),
				}

				if org.ConnectionStringIV != "" && org.ConnectionStringTag != "" {
					dsnCT, _ := utils.DecodeB64(org.ConnectionString)
					dsnIV, _ := utils.DecodeB64(org.ConnectionStringIV)
					dsnTag, _ := utils.DecodeB64(org.ConnectionStringTag)

					dsn, err := utils.DecryptAESGCMWithAAD(dsnCT, oldDSNKey, dsnIV, dsnTag, aad)
					if err != nil {
						return fmt.Errorf("failed to decrypt connection string for org %s: %w", org.ID, err)
					}

					newDSNCT, newDSNIV, newDSNTag, err := utils.EncryptAESGCMWithAAD(dsn, newDSNKey, aad)
					if err != nil {
						return fmt.Errorf("failed to encrypt connection string for org %s: %w", org.ID, err)
					}

					updates["connection_string"] = utils.EncodeB64(newDSNCT)
					updates["connection_string_iv"] = utils.EncodeB64(newDSNIV)
					updates["connection_string_tag"] = utils.EncodeB64(newDSNTag)
				} else if org.ConnectionString != "" {

					return fmt.Errorf(
						"org %s has an unencrypted connection_string; re-provision it before rotating",
						org.ID)
				}

				if err := tx.Model(&org).Updates(updates).Error; err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("rotation failed: %w", err)
		}

		fmt.Println("\n✅ Master key rotated successfully in the database.")
		fmt.Println("⚠️  IMPORTANT: Update your environment configuration IMMEDIATELY.")
		fmt.Println("Replace your current ENCRYPTION_MASTER_KEY in your .env with this new key:")
		fmt.Printf("ENCRYPTION_MASTER_KEY=%s\n\n", utils.EncodeB64(newMasterKey))
		fmt.Println("Then restart your API and Worker processes.")

		return nil
	},
}

func tenantSubkeys(master []byte) (dekKey, dsnKey []byte, err error) {
	dekKey, err = utils.DeriveKey(master, utils.KeyPurposeTenantDEK)
	if err != nil {
		return nil, nil, fmt.Errorf("derive tenant DEK key: %w", err)
	}
	dsnKey, err = utils.DeriveKey(master, utils.KeyPurposeTenantDSN)
	if err != nil {
		return nil, nil, fmt.Errorf("derive tenant DSN key: %w", err)
	}
	return dekKey, dsnKey, nil
}

func init() {
	encryptCmd.AddCommand(generateMasterCmd, rotateMasterCmd)
	rootCmd.AddCommand(encryptCmd)
}
