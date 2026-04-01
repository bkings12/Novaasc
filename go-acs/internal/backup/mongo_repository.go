package backup

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type MongoRepository struct {
	backups  *mongo.Collection
	restores *mongo.Collection
	log      *zap.Logger
}

func NewMongoRepository(db *mongo.Database, log *zap.Logger) *MongoRepository {
	return &MongoRepository{
		backups:  db.Collection("backups"),
		restores: db.Collection("restore_jobs"),
		log:      log,
	}
}

func (r *MongoRepository) EnsureIndexes(ctx context.Context) error {
	backupIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "device_serial", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err := r.backups.Indexes().CreateMany(ctx, backupIndexes); err != nil {
		return err
	}
	restoreIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "device_serial", Value: 1}, {Key: "created_at", Value: -1}}},
	}
	if _, err := r.restores.Indexes().CreateMany(ctx, restoreIndexes); err != nil {
		return err
	}
	return nil
}

func (r *MongoRepository) Create(ctx context.Context, b *Backup) error {
	if b.ID == "" {
		b.ID = uuid.New().String()
	}
	b.CreatedAt = time.Now()
	b.ParameterCount = len(b.Parameters)
	_, err := r.backups.InsertOne(ctx, b)
	if err != nil {
		return err
	}
	return nil
}

func (r *MongoRepository) GetByID(ctx context.Context, tenantID, id string) (*Backup, error) {
	filter := bson.M{"_id": id, "tenant_id": tenantID}
	var b Backup
	err := r.backups.FindOne(ctx, filter).Decode(&b)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

func (r *MongoRepository) ListForDevice(ctx context.Context, tenantID, serial string, limit int64) ([]*Backup, error) {
	filter := bson.M{"tenant_id": tenantID, "device_serial": serial}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetProjection(bson.M{"parameters": 0})
	cursor, err := r.backups.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var list []*Backup
	if err := cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *MongoRepository) Delete(ctx context.Context, tenantID, id string) error {
	filter := bson.M{"_id": id, "tenant_id": tenantID}
	res, err := r.backups.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MongoRepository) CreateRestoreJob(ctx context.Context, job *RestoreJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	job.CreatedAt = time.Now()
	_, err := r.restores.InsertOne(ctx, job)
	return err
}

func (r *MongoRepository) GetRestoreJob(ctx context.Context, tenantID, id string) (*RestoreJob, error) {
	filter := bson.M{"_id": id, "tenant_id": tenantID}
	var job RestoreJob
	err := r.restores.FindOne(ctx, filter).Decode(&job)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &job, nil
}

func (r *MongoRepository) UpdateRestoreJob(ctx context.Context, job *RestoreJob) error {
	filter := bson.M{"_id": job.ID, "tenant_id": job.TenantID}
	_, err := r.restores.ReplaceOne(ctx, filter, job)
	return err
}
