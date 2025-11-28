package upload_service

import (
	"log"
	"time"
)

// CleanupProcessor 清理过期上传的处理器
type CleanupProcessor struct {
	uploadService *UploadService
	stopChan      chan struct{}
	interval      time.Duration
	batchSize     int
	expiredBefore time.Duration // 清理多少时间之前过期的记录（例如：1小时前过期的）
}

// NewCleanupProcessor 创建清理处理器
func NewCleanupProcessor(uploadService *UploadService) *CleanupProcessor {
	return &CleanupProcessor{
		uploadService: uploadService,
		stopChan:      make(chan struct{}),
		interval:      10 * time.Minute, // 每10分钟执行一次清理
		batchSize:     100,              // 每次处理100条记录
		expiredBefore: 1 * time.Hour,    // 清理1小时前过期的记录（给一些缓冲时间）
	}
}

// Start 启动清理处理器
func (cp *CleanupProcessor) Start() {
	log.Println("Cleanup processor started")
	go cp.run()
}

// Stop 停止清理处理器
func (cp *CleanupProcessor) Stop() {
	log.Println("Stopping cleanup processor...")
	close(cp.stopChan)
}

// run 运行清理处理器主循环
func (cp *CleanupProcessor) run() {
	ticker := time.NewTicker(cp.interval)
	defer ticker.Stop()

	// 启动时立即执行一次清理
	cp.cleanupExpiredUploads()

	for {
		select {
		case <-cp.stopChan:
			log.Println("Cleanup processor stopped")
			return
		case <-ticker.C:
			cp.cleanupExpiredUploads()
		}
	}
}

// cleanupExpiredUploads 清理过期的上传记录
func (cp *CleanupProcessor) cleanupExpiredUploads() {
	// 计算清理时间点（当前时间减去expiredBefore，确保只清理已经过期一段时间的记录）
	beforeTime := time.Now().Add(-cp.expiredBefore)

	log.Printf("Starting cleanup expired uploads (before: %s)", beforeTime.Format(time.RFC3339))

	// 清理过期上传（从存储和数据库）
	cleanedCount, err := cp.uploadService.CleanupExpiredUploads(beforeTime, cp.batchSize)
	if err != nil {
		log.Printf("Failed to cleanup expired uploads: %v", err)
		return
	}

	if cleanedCount > 0 {
		log.Printf("Cleaned up %d expired upload records", cleanedCount)
	}

	// 删除已经标记为expired的记录（可选，可以定期删除更早的记录）
	// 这里删除7天前就标记为expired的记录
	deleteBeforeTime := time.Now().Add(-7 * 24 * time.Hour)
	deletedCount, err := cp.uploadService.DeleteExpiredUploadRecords(deleteBeforeTime)
	if err != nil {
		log.Printf("Failed to delete expired upload records: %v", err)
		return
	}

	if deletedCount > 0 {
		log.Printf("Deleted %d expired upload records from database", deletedCount)
	}
}
