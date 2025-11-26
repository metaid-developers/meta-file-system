package respond

import (
	"encoding/json"
	"log"
	"time"

	"meta-file-system/model"
)

// ChunkedUploadTaskResponse describes the response after creating an async chunked upload task.
type ChunkedUploadTaskResponse struct {
	TaskId  string `json:"taskId" example:"task_123" description:"Unique task ID"`
	Status  string `json:"status" example:"pending" description:"Task status"`
	Message string `json:"message" example:"task created" description:"Additional message"`
}

// UploadTask represents the public view of a file upload task.
type UploadTask struct {
	TaskId          string     `json:"taskId"`
	MetaId          string     `json:"metaId"`
	Address         string     `json:"address"`
	FileName        string     `json:"fileName"`
	FileHash        string     `json:"fileHash"`
	FileMd5         string     `json:"fileMd5"`
	FileSize        int64      `json:"fileSize"`
	ContentType     string     `json:"contentType"`
	Path            string     `json:"path"`
	Operation       string     `json:"operation"`
	Status          string     `json:"status"`
	Progress        int        `json:"progress"`
	TotalChunks     int        `json:"totalChunks"`
	ProcessedChunks int        `json:"processedChunks"`
	CurrentStep     string     `json:"currentStep"`
	Stage           string     `json:"stage"`
	FileId          string     `json:"fileId"`
	ChunkFundingTx  string     `json:"chunkFundingTx"`
	ChunkTxIds      []string   `json:"chunkTxIds"`
	IndexTxId       string     `json:"indexTxId"`
	ErrorMessage    string     `json:"errorMessage"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	StartedAt       *time.Time `json:"startedAt"`
	FinishedAt      *time.Time `json:"finishedAt"`
}

// UploadTaskDetailResponse wraps a single upload task.
type UploadTaskDetailResponse struct {
	Task *UploadTask `json:"task"`
}

// UploadTaskListResponse describes a paginated upload task list.
type UploadTaskListResponse struct {
	Tasks      []*UploadTask `json:"tasks"`
	NextCursor int64         `json:"nextCursor" example:"123" description:"Cursor for the next page"`
	HasMore    bool          `json:"hasMore" example:"true" description:"Whether there are more records"`
}

// ToUploadTask converts a model.FileUploaderTask into a public response struct.
func ToUploadTask(task *model.FileUploaderTask) *UploadTask {
	if task == nil {
		return nil
	}

	var chunkTxIds []string
	if task.ChunkTxIds != "" {
		if err := json.Unmarshal([]byte(task.ChunkTxIds), &chunkTxIds); err != nil {
			log.Printf("Failed to unmarshal chunkTxIds for task %s: %v", task.TaskId, err)
		}
	}

	return &UploadTask{
		TaskId:          task.TaskId,
		MetaId:          task.MetaId,
		Address:         task.Address,
		FileName:        task.FileName,
		FileHash:        task.FileHash,
		FileMd5:         task.FileMd5,
		FileSize:        task.FileSize,
		ContentType:     task.ContentType,
		Path:            task.Path,
		Operation:       task.Operation,
		Status:          string(task.Status),
		Progress:        task.Progress,
		TotalChunks:     task.TotalChunks,
		ProcessedChunks: task.ProcessedChunks,
		CurrentStep:     task.CurrentStep,
		Stage:           string(task.Stage),
		FileId:          task.FileId,
		ChunkFundingTx:  task.ChunkFundingTx,
		ChunkTxIds:      chunkTxIds,
		IndexTxId:       task.IndexTxId,
		ErrorMessage:    task.ErrorMessage,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
		StartedAt:       task.StartedAt,
		FinishedAt:      task.FinishedAt,
	}
}

// ToUploadTaskList converts a slice of model tasks to response structs.
func ToUploadTaskList(tasks []*model.FileUploaderTask) []*UploadTask {
	result := make([]*UploadTask, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, ToUploadTask(t))
	}
	return result
}
