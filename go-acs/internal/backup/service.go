package backup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/novaacs/go-acs/internal/device"
	"github.com/novaacs/go-acs/internal/task"
	"go.uber.org/zap"
)

const MaxParamsPerChunk = 50

var ReadOnlyPrefixes = []string{
	"Device.DeviceInfo.Uptime",
	"Device.DeviceInfo.MemoryStatus",
	"Device.DeviceInfo.ProcessStatus",
	"Device.WiFi.SSID.1.Stats",
	"Device.WiFi.Radio.1.Stats",
	"Device.WiFi.AccessPoint.1.AssociatedDevice",
	"Device.IP.Interface.1.Stats",
	"Device.Ethernet.Interface",
	".Stats.",
	".Status",
	"NumberOfEntries",
	"Device.ManagementServer.ConnectionRequestURL",
}

type Service struct {
	repo       Repository
	deviceRepo device.Repository
	taskRepo   task.Repository
	log        *zap.Logger
}

func NewService(
	repo Repository,
	deviceRepo device.Repository,
	taskRepo task.Repository,
	log *zap.Logger,
) *Service {
	return &Service{
		repo:       repo,
		deviceRepo: deviceRepo,
		taskRepo:   taskRepo,
		log:        log,
	}
}

func (s *Service) TakeBackup(ctx context.Context, tenantID, serial, trigger, createdBy string) (*Backup, error) {
	dev, err := s.deviceRepo.GetBySerial(ctx, tenantID, serial)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	if len(dev.Parameters) == 0 {
		return nil, fmt.Errorf("device has no parameters stored — run GetParameterValues first")
	}
	b := &Backup{
		TenantID:        tenantID,
		DeviceSerial:    serial,
		Trigger:         trigger,
		Label:           fmt.Sprintf("%s snapshot", trigger),
		Parameters:      dev.Parameters,
		ParameterCount:  len(dev.Parameters),
		SoftwareVersion: dev.SoftwareVersion,
		IPAddress:       dev.IPAddress,
		CreatedAt:       time.Now(),
		CreatedBy:       createdBy,
	}
	if err := s.repo.Create(ctx, b); err != nil {
		return nil, fmt.Errorf("store backup: %w", err)
	}
	s.log.Info("backup created",
		zap.String("serial", serial),
		zap.String("id", b.ID),
		zap.Int("params", b.ParameterCount),
		zap.String("trigger", trigger),
	)
	return b, nil
}

func (s *Service) StartRestore(ctx context.Context, tenantID, backupID, createdBy string) (*RestoreJob, error) {
	b, err := s.repo.GetByID(ctx, tenantID, backupID)
	if err != nil {
		return nil, fmt.Errorf("backup not found: %w", err)
	}
	writable := s.filterWritable(b.Parameters)
	s.log.Info("starting restore",
		zap.String("backup_id", backupID),
		zap.String("serial", b.DeviceSerial),
		zap.Int("total_params", len(b.Parameters)),
		zap.Int("writable_params", len(writable)),
	)
	chunks := chunkParams(writable, MaxParamsPerChunk)
	job := &RestoreJob{
		TenantID:     tenantID,
		BackupID:     backupID,
		DeviceSerial: b.DeviceSerial,
		Status:       "pending",
		TotalChunks:  len(chunks),
		DoneChunks:   0,
		CreatedAt:    time.Now(),
		CreatedBy:    createdBy,
	}
	if err := s.repo.CreateRestoreJob(ctx, job); err != nil {
		return nil, fmt.Errorf("create restore job: %w", err)
	}
	for i, chunk := range chunks {
		t := &task.Task{
			TenantID:        tenantID,
			DeviceSerial:    b.DeviceSerial,
			Type:            task.TypeSetParameterValues,
			Status:          task.StatusPending,
			Priority:        100 + i,
			ParameterValues: chunk,
			CreatedAt:       time.Now(),
			Timeout:         int64(10 * time.Minute),
			CreatedBy:       fmt.Sprintf("restore:%s", job.ID),
		}
		if err := s.taskRepo.Enqueue(ctx, t); err != nil {
			s.log.Error("enqueue restore chunk", zap.Int("chunk", i), zap.Error(err))
		} else {
			job.TaskIDs = append(job.TaskIDs, t.ID)
		}
	}
	job.Status = "running"
	_ = s.repo.UpdateRestoreJob(ctx, job)
	s.log.Info("restore job started",
		zap.String("job_id", job.ID),
		zap.String("serial", b.DeviceSerial),
		zap.Int("chunks", len(chunks)),
		zap.Int("tasks", len(job.TaskIDs)),
	)
	return job, nil
}

func (s *Service) filterWritable(params map[string]string) map[string]string {
	result := make(map[string]string)
outer:
	for k, v := range params {
		for _, prefix := range ReadOnlyPrefixes {
			if strings.Contains(k, prefix) {
				continue outer
			}
		}
		result[k] = v
	}
	return result
}

func chunkParams(params map[string]string, n int) []map[string]string {
	var chunks []map[string]string
	current := make(map[string]string)
	i := 0
	for k, v := range params {
		current[k] = v
		i++
		if i >= n {
			chunks = append(chunks, current)
			current = make(map[string]string)
			i = 0
		}
	}
	if len(current) > 0 {
		chunks = append(chunks, current)
	}
	return chunks
}
