package upload_service

import (
	"log"
	"time"

	"meta-file-system/model"
	"meta-file-system/model/dao"
)

// TaskProcessor 任务处理器
type TaskProcessor struct {
	uploadService    *UploadService
	taskDAO          *dao.FileUploaderTaskDAO
	stopChan         chan struct{}
	interval         time.Duration
	batchSize        int
	stalledThreshold time.Duration
}

// NewTaskProcessor 创建任务处理器
func NewTaskProcessor(uploadService *UploadService) *TaskProcessor {
	return &TaskProcessor{
		uploadService:    uploadService,
		taskDAO:          dao.NewFileUploaderTaskDAO(),
		stopChan:         make(chan struct{}),
		interval:         5 * time.Second, // 每5秒轮询一次
		batchSize:        5,               // 每次处理5个任务
		stalledThreshold: 2 * time.Minute, // processing 任务超过2分钟视为卡住
	}
}

// Start 启动任务处理器
func (tp *TaskProcessor) Start() {
	log.Println("Task processor started")
	go tp.run()
}

// Stop 停止任务处理器
func (tp *TaskProcessor) Stop() {
	log.Println("Stopping task processor...")
	close(tp.stopChan)
}

// run 运行任务处理器主循环
func (tp *TaskProcessor) run() {
	ticker := time.NewTicker(tp.interval)
	defer ticker.Stop()

	for {
		select {
		case <-tp.stopChan:
			log.Println("Task processor stopped")
			return
		case <-ticker.C:
			tp.processPendingTasks()
		}
	}
}

// processPendingTasks 处理待处理的任务
func (tp *TaskProcessor) processPendingTasks() {
	// 获取待处理的任务
	tasks, err := tp.taskDAO.GetPendingTasks(tp.batchSize)
	if err != nil {
		log.Printf("Failed to get pending tasks: %v", err)
		return
	}

	before := time.Now().Add(-tp.stalledThreshold)
	stalledTasks, err := tp.taskDAO.GetStalledProcessingTasks(before, tp.batchSize)
	if err != nil {
		log.Printf("Failed to get stalled processing tasks: %v", err)
	}

	uniqueTasks := make(map[int64]*model.FileUploaderTask)
	for _, task := range tasks {
		uniqueTasks[task.ID] = task
	}
	for _, task := range stalledTasks {
		if _, exists := uniqueTasks[task.ID]; exists {
			continue
		}
		uniqueTasks[task.ID] = task
	}

	if len(uniqueTasks) == 0 {
		return
	}

	log.Printf("Found %d pending tasks and %d stalled tasks, processing...", len(tasks), len(uniqueTasks)-len(tasks))

	// 处理每个任务
	for _, task := range uniqueTasks {
		// 使用 goroutine 异步处理每个任务，避免阻塞
		go tp.processTask(task)
	}
}

// processTask 处理单个任务
func (tp *TaskProcessor) processTask(task *model.FileUploaderTask) {
	log.Printf("Processing task: taskId=%s, fileId=%s", task.TaskId, task.FileId)

	// 处理任务
	if err := tp.uploadService.ProcessUploadTask(task); err != nil {
		log.Printf("Failed to process task %s: %v", task.TaskId, err)
	} else {
		log.Printf("Task %s processed successfully", task.TaskId)
	}
}
