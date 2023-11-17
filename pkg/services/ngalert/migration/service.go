package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/serverlock"
	migrationStore "github.com/grafana/grafana/pkg/services/ngalert/migration/store"
	"github.com/grafana/grafana/pkg/services/secrets"
	"github.com/grafana/grafana/pkg/setting"
)

// actionName is the unique row-level lock name for serverlock.ServerLockService.
const actionName = "alerting migration"

const anyOrg = 0

type UpgradeService interface {
	Run(ctx context.Context) error
}

type migrationService struct {
	lock           *serverlock.ServerLockService
	cfg            *setting.Cfg
	log            log.Logger
	store          db.DB
	migrationStore migrationStore.Store

	encryptionService secrets.Service
}

func ProvideService(
	lock *serverlock.ServerLockService,
	cfg *setting.Cfg,
	store db.DB,
	migrationStore migrationStore.Store,
	encryptionService secrets.Service,
) (UpgradeService, error) {
	return &migrationService{
		lock:              lock,
		log:               log.New("ngalert.migration"),
		cfg:               cfg,
		store:             store,
		migrationStore:    migrationStore,
		encryptionService: encryptionService,
	}, nil
}

// Run starts the migration, any migration issues will throw an error.
// If we are moving from legacy->UA:
//   - All orgs without their migration status set to true in the kvstore will be migrated.
//   - If CleanUpgrade=true, then UA will be reverted first. So, all orgs will be migrated from scratch.
//
// If we are moving from UA->legacy:
//   - No-op except to set a kvstore flag with orgId=0 that lets us determine when we move from legacy->UA. No UA resources are deleted or reverted.
func (ms *migrationService) Run(ctx context.Context) error {
	var errMigration error
	errLock := ms.lock.LockExecuteAndRelease(ctx, actionName, time.Minute*10, func(ctx context.Context) {
		ms.log.Info("Starting")
		errMigration = ms.store.InTransaction(ctx, func(ctx context.Context) error {
			migrated, err := ms.migrationStore.IsMigrated(ctx, anyOrg)
			if err != nil {
				return fmt.Errorf("getting migration status: %w", err)
			}

			if !ms.cfg.UnifiedAlerting.IsEnabled() {
				// Set status to false so that next time UA is enabled, we run the migration again. That is when
				// CleanUpgrade will be checked to determine if revert should happen.
				err = ms.migrationStore.SetMigrated(ctx, anyOrg, false)
				if err != nil {
					return fmt.Errorf("setting migration status: %w", err)
				}
				return nil
			}

			if migrated {
				ms.log.Info("Migration already run")
				return nil
			}

			// Safeguard to prevent data loss.
			if ms.cfg.UnifiedAlerting.Upgrade.CleanUpgrade {
				ms.log.Info("CleanUpgrade enabled, reverting and migrating orgs from scratch")
				// Revert migration
				ms.log.Info("Reverting unified alerting data")
				err := ms.migrationStore.RevertAllOrgs(ctx)
				if err != nil {
					return fmt.Errorf("reverting: %w", err)
				}
				ms.log.Info("Unified alerting data reverted")
			}

			ms.log.Info("Starting legacy migration")
			err = ms.migrateAllOrgs(ctx)
			if err != nil {
				return fmt.Errorf("executing migration: %w", err)
			}

			err = ms.migrationStore.SetMigrated(ctx, anyOrg, true)
			if err != nil {
				return fmt.Errorf("setting migration status: %w", err)
			}

			ms.log.Info("Completed legacy migration")
			return nil
		})
	})
	if errLock != nil {
		ms.log.Warn("Server lock for alerting migration already exists")
		return nil
	}
	if errMigration != nil {
		return fmt.Errorf("migration failed: %w", errMigration)
	}
	return nil
}

// migrateAllOrgs executes the migration for all orgs.
func (ms *migrationService) migrateAllOrgs(ctx context.Context) error {
	orgs, err := ms.migrationStore.GetAllOrgs(ctx)
	if err != nil {
		return fmt.Errorf("get orgs: %w", err)
	}

	for _, o := range orgs {
		om := ms.newOrgMigration(o.ID)
		migrated, err := ms.migrationStore.IsMigrated(ctx, o.ID)
		if err != nil {
			return fmt.Errorf("getting migration status for org %d: %w", o.ID, err)
		}
		if migrated {
			om.log.Info("Org already migrated, skipping")
			continue
		}

		if err := om.migrateOrg(ctx); err != nil {
			return fmt.Errorf("migrate org %d: %w", o.ID, err)
		}

		err = om.migrationStore.SetOrgMigrationState(ctx, o.ID, om.state)
		if err != nil {
			return fmt.Errorf("set org migration state: %w", err)
		}

		err = ms.migrationStore.SetMigrated(ctx, o.ID, true)
		if err != nil {
			return fmt.Errorf("setting migration status: %w", err)
		}
	}
	return nil
}
