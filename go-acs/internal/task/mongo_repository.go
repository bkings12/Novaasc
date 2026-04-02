package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const collectionName = "tasks"
const defaultTimeoutNs = int64(5 * time.Minute)

// MongoRepository implements Repository using MongoDB.
type MongoRepository struct {
	col         *mongo.Collection
	log         *zap.Logger
	resultChans sync.Map // taskID -> chan Result
}

// NewMongoRepository returns a new MongoDB task repository.
func NewMongoRepository(db *mongo.Database, log *zap.Logger) *MongoRepository {
	return &MongoRepository{col: db.Collection(collectionName), log: log}
}

// EnsureIndexes creates required indexes.
func (r *MongoRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{Keys: bson.D{
			{Key: "tenant_id", Value: 1},
			{Key: "device_serial", Value: 1},
			{Key: "status", Value: 1},
			{Key: "priority", Value: -1},
			{Key: "created_at", Value: 1},
		}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "status", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "device_serial", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("task indexes: %w", err)
	}
	return nil
}

func (r *MongoRepository) Enqueue(ctx context.Context, t *Task) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	t.Status = StatusPending
	t.CreatedAt = time.Now()
	if t.Timeout == 0 {
		t.Timeout = defaultTimeoutNs
	}
	if t.ResultChan != nil {
		r.resultChans.Store(t.ID, t.ResultChan)
	}
	doc := taskToDoc(t)
	_, err := r.col.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("task enqueue: %w", err)
	}
	return nil
}

func (r *MongoRepository) HasPendingCreatedBy(ctx context.Context, tenantID, serial, createdBy string) (bool, error) {
	n, err := r.col.CountDocuments(ctx, bson.M{
		"tenant_id":     tenantID,
		"device_serial": serial,
		"status":        StatusPending,
		"created_by":    createdBy,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *MongoRepository) NextForDevice(ctx context.Context, tenantID, serial string) (*Task, error) {
	filter := bson.M{
		"tenant_id":     tenantID,
		"device_serial": serial,
		"status":        StatusPending,
	}
	now := primitive.NewDateTimeFromTime(time.Now())
	update := bson.M{
		"$set": bson.M{
			"status":        StatusDispatched,
			"dispatched_at": now,
		},
	}
	opts := options.FindOneAndUpdate().
		SetSort(bson.D{{Key: "priority", Value: -1}, {Key: "created_at", Value: 1}}).
		SetReturnDocument(options.After)
	var doc taskDoc
	err := r.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return docToTask(&doc), nil
}

func (r *MongoRepository) Complete(ctx context.Context, tenantID, taskID string, result Result) error {
	result.CompletedAt = time.Now()
	filter := bson.M{"_id": taskID, "tenant_id": tenantID}
	update := bson.M{
		"$set": bson.M{
			"status":       StatusComplete,
			"result":       result,
			"completed_at": primitive.NewDateTimeFromTime(result.CompletedAt),
		},
	}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	if v, ok := r.resultChans.LoadAndDelete(taskID); ok {
		if ch, ok := v.(chan Result); ok {
			select {
			case ch <- result:
			default:
			}
		}
	}
	return nil
}

func (r *MongoRepository) Fail(ctx context.Context, tenantID, taskID string, reason string) error {
	result := Result{
		Success:     false,
		CompletedAt: time.Now(),
		Fault:       &FaultResult{Message: reason},
	}
	filter := bson.M{"_id": taskID, "tenant_id": tenantID}
	update := bson.M{
		"$set": bson.M{
			"status":       StatusFailed,
			"result":       result,
			"completed_at": primitive.NewDateTimeFromTime(result.CompletedAt),
		},
	}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	if v, ok := r.resultChans.LoadAndDelete(taskID); ok {
		if ch, ok := v.(chan Result); ok {
			select {
			case ch <- result:
			default:
			}
		}
	}
	return nil
}

func (r *MongoRepository) Cancel(ctx context.Context, tenantID, taskID string) error {
	filter := bson.M{"_id": taskID, "tenant_id": tenantID, "status": StatusPending}
	res, err := r.col.UpdateOne(ctx, filter, bson.M{"$set": bson.M{"status": StatusCancelled}})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MongoRepository) GetByID(ctx context.Context, tenantID, taskID string) (*Task, error) {
	filter := bson.M{"_id": taskID, "tenant_id": tenantID}
	var doc taskDoc
	err := r.col.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return docToTask(&doc), nil
}

func (r *MongoRepository) ListForDevice(ctx context.Context, tenantID, serial string, limit int64) ([]*Task, error) {
	if limit <= 0 {
		limit = 50
	}
	filter := bson.M{"tenant_id": tenantID, "device_serial": serial}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)
	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var list []*Task
	for cursor.Next(ctx) {
		var doc taskDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		list = append(list, docToTask(&doc))
	}
	return list, cursor.Err()
}

func (r *MongoRepository) ListForTenant(ctx context.Context, tenantID string, filter Filter) ([]*Task, int64, error) {
	f := bson.M{"tenant_id": tenantID}
	if filter.Status != "" {
		f["status"] = filter.Status
	}
	if filter.DeviceSerial != "" {
		f["device_serial"] = filter.DeviceSerial
	}
	total, err := r.col.CountDocuments(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	limit, offset := filter.Limit, filter.Offset
	if limit <= 0 {
		limit = 50
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetSkip(offset).SetLimit(limit)
	cursor, err := r.col.Find(ctx, f, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	var list []*Task
	for cursor.Next(ctx) {
		var doc taskDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, err
		}
		list = append(list, docToTask(&doc))
	}
	return list, total, cursor.Err()
}

func (r *MongoRepository) TimeoutStale(ctx context.Context, cutoff time.Time) (int64, error) {
	filter := bson.M{
		"status":        StatusDispatched,
		"dispatched_at": bson.M{"$lt": primitive.NewDateTimeFromTime(cutoff)},
	}
	update := bson.M{"$set": bson.M{"status": StatusTimeout}}
	res, err := r.col.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

type taskDoc struct {
	ID              string             `bson:"_id"`
	TenantID        string             `bson:"tenant_id"`
	DeviceSerial    string             `bson:"device_serial"`
	Type            Type               `bson:"type"`
	Status          Status             `bson:"status"`
	Priority        int                `bson:"priority"`
	ParameterNames  []string           `bson:"parameter_names,omitempty"`
	ParameterValues map[string]string  `bson:"parameter_values,omitempty"`
	Download        *DownloadArgs      `bson:"download,omitempty"`
	Result          *Result            `bson:"result,omitempty"`
	CreatedAt       primitive.DateTime `bson:"created_at"`
	DispatchedAt    primitive.DateTime `bson:"dispatched_at,omitempty"`
	CompletedAt     primitive.DateTime `bson:"completed_at,omitempty"`
	Timeout         int64              `bson:"timeout"`
	CreatedBy       string             `bson:"created_by"`
}

func taskToDoc(t *Task) *taskDoc {
	doc := &taskDoc{
		ID:              t.ID,
		TenantID:        t.TenantID,
		DeviceSerial:    t.DeviceSerial,
		Type:            t.Type,
		Status:          t.Status,
		Priority:        t.Priority,
		ParameterNames:  t.ParameterNames,
		ParameterValues: t.ParameterValues,
		Download:        t.Download,
		Result:          t.Result,
		CreatedAt:       primitive.NewDateTimeFromTime(t.CreatedAt),
		Timeout:         t.Timeout,
		CreatedBy:       t.CreatedBy,
	}
	if t.DispatchedAt != nil {
		doc.DispatchedAt = primitive.NewDateTimeFromTime(*t.DispatchedAt)
	}
	if t.CompletedAt != nil {
		doc.CompletedAt = primitive.NewDateTimeFromTime(*t.CompletedAt)
	}
	return doc
}

func docToTask(d *taskDoc) *Task {
	if d == nil {
		return nil
	}
	t := &Task{
		ID:              d.ID,
		TenantID:        d.TenantID,
		DeviceSerial:    d.DeviceSerial,
		Type:            d.Type,
		Status:          d.Status,
		Priority:        d.Priority,
		ParameterNames:  d.ParameterNames,
		ParameterValues: d.ParameterValues,
		Download:        d.Download,
		Result:          d.Result,
		CreatedAt:       d.CreatedAt.Time(),
		Timeout:         d.Timeout,
		CreatedBy:       d.CreatedBy,
	}
	if !d.DispatchedAt.Time().IsZero() {
		dt := d.DispatchedAt.Time()
		t.DispatchedAt = &dt
	}
	if !d.CompletedAt.Time().IsZero() {
		ct := d.CompletedAt.Time()
		t.CompletedAt = &ct
	}
	return t
}
