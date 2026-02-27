package indexer_service

import (
	"log"

	"meta-file-system/database"
	"meta-file-system/model"
	common_service "meta-file-system/service/common_service"
)

// LatestSchemaVersion 当前最新 schema 版本，新增 migrate 时递增
const LatestSchemaVersion = 1

// MigrateService 负责 indexer 启动时根据版本号执行 migrate
type MigrateService struct{}

// NewMigrateService 创建 MigrateService
func NewMigrateService() *MigrateService {
	return &MigrateService{}
}

// Run 检查当前版本与最新版本，不匹配则按版本号依次执行 migrate
func (s *MigrateService) Run() error {
	if database.DB == nil {
		return nil
	}
	// 仅 Pebble 需要 migrate（MySQL 无 collectionLatestFileInfo 等）
	if database.GetDBType() != database.DBTypePebble {
		return nil
	}

	current, err := database.DB.GetIndexerSchemaVersion()
	if err != nil {
		return err
	}
	if current >= LatestSchemaVersion {
		log.Printf("[Migrate] Schema version is up to date: current=%d, latest=%d", current, LatestSchemaVersion)
		return nil
	}

	log.Printf("[Migrate] Schema version outdated: current=%d, latest=%d, running migrations...", current, LatestSchemaVersion)
	for v := current + 1; v <= LatestSchemaVersion; v++ {
		if err := s.runMigrate(v); err != nil {
			return err
		}
		if err := database.DB.SetIndexerSchemaVersion(v); err != nil {
			return err
		}
		log.Printf("[Migrate] Schema version updated to %d", v)
	}
	log.Printf("[Migrate] All migrations completed, version=%d", LatestSchemaVersion)
	return nil
}

// runMigrate 执行指定版本的 migrate
func (s *MigrateService) runMigrate(version int) error {
	switch version {
	case 1:
		return s.migrateV1()
	default:
		log.Printf("[Migrate] No migration defined for version %d", version)
		return nil
	}
}

// migrateV1 遍历 collectionLatestFileInfo，回填 collectionFileGlobalMetaID、collectionFileExtensionTimestamp、collectionGlobalMetaIDFileExtensionTimestamp
func (s *MigrateService) migrateV1() error {
	log.Println("[Migrate] V1: Backfilling file_global_meta, file_extension_timestamp, global_meta_id_file_extension_timestamp from latest_file_info...")
	var count int
	err := database.DB.IterateLatestFileInfo(func(file *model.IndexerFile) error {
		if file.CreatorGlobalMetaId == "" && file.CreatorAddress != "" {
			file.CreatorGlobalMetaId = common_service.ConvertToGlobalMetaId(file.CreatorAddress)
		}
		if err := database.DB.WriteFileToExtensionAndGlobalMetaIndexes(file); err != nil {
			return err
		}
		count++
		if count%1000 == 0 {
			log.Printf("[Migrate] V1: processed %d files...", count)
		}
		return nil
	})
	if err != nil {
		return err
	}
	log.Printf("[Migrate] V1: completed, total %d files backfilled", count)
	return nil
}
