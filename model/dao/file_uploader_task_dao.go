package dao

import (
	"fmt"
	"time"

	"meta-file-system/database"
	"meta-file-system/model"
)

// FileUploaderTaskDAO data access layer for upload tasks.
type FileUploaderTaskDAO struct{}

// NewFileUploaderTaskDAO creates a new DAO instance.
func NewFileUploaderTaskDAO() *FileUploaderTaskDAO {
	return &FileUploaderTaskDAO{}
}

// Create inserts a new task record.
func (dao *FileUploaderTaskDAO) Create(task *model.FileUploaderTask) error {
	return database.UploaderDB.Create(task).Error
}

// GetByTaskID fetches a task by task ID.
func (dao *FileUploaderTaskDAO) GetByTaskID(taskID string) (*model.FileUploaderTask, error) {
	var task model.FileUploaderTask
	err := database.UploaderDB.Where("task_id = ?", taskID).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByID fetches a task by primary key.
func (dao *FileUploaderTaskDAO) GetByID(id int64) (*model.FileUploaderTask, error) {
	var task model.FileUploaderTask
	err := database.UploaderDB.Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// Update persists task changes.
func (dao *FileUploaderTaskDAO) Update(task *model.FileUploaderTask) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}

	return database.UploaderDB.Model(&model.FileUploaderTask{}).
		Where("id = ?", task.ID).
		Select("*").
		Updates(task).Error
}

// GetPendingTasks returns pending tasks ordered by creation time ascending.
func (dao *FileUploaderTaskDAO) GetPendingTasks(limit int) ([]*model.FileUploaderTask, error) {
	var tasks []*model.FileUploaderTask
	err := database.UploaderDB.Where("status = ?", model.StatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

// GetProcessingTasks returns processing tasks ordered by creation time ascending.
func (dao *FileUploaderTaskDAO) GetProcessingTasks(limit int) ([]*model.FileUploaderTask, error) {
	var tasks []*model.FileUploaderTask
	err := database.UploaderDB.Where("status = ?", "processing").
		Order("created_at ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

// GetStalledProcessingTasks returns processing tasks that have not been updated before a specific time.
func (dao *FileUploaderTaskDAO) GetStalledProcessingTasks(updatedBefore time.Time, limit int) ([]*model.FileUploaderTask, error) {
	var tasks []*model.FileUploaderTask
	err := database.UploaderDB.
		Where("status = ? AND updated_at < ? AND progress < ?", "processing", updatedBefore, 100).
		Order("updated_at ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

// List returns tasks using pagination.
func (dao *FileUploaderTaskDAO) List(offset, limit int, status string) ([]*model.FileUploaderTask, error) {
	var tasks []*model.FileUploaderTask
	query := database.UploaderDB.Order("created_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Offset(offset).Limit(limit).Find(&tasks).Error
	return tasks, err
}

// Count returns total number of tasks (optional by status).
func (dao *FileUploaderTaskDAO) Count(status string) (int64, error) {
	var count int64
	query := database.UploaderDB.Model(&model.FileUploaderTask{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Count(&count).Error
	return count, err
}

// ListByAddressWithCursor returns tasks by address with cursor pagination (id desc).
// cursor: last task ID from previous page (0 for first page).
func (dao *FileUploaderTaskDAO) ListByAddressWithCursor(address string, cursor int64, size int) ([]*model.FileUploaderTask, int64, error) {
	if size <= 0 || size > 100 {
		size = 20
	}

	var tasks []*model.FileUploaderTask
	query := database.UploaderDB.Where("address = ?", address)
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}

	if err := query.Order("id DESC").Limit(size).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	var nextCursor int64
	if len(tasks) > 0 {
		nextCursor = tasks[len(tasks)-1].ID
	}

	return tasks, nextCursor, nil
}
