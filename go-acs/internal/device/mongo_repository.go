package device

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

var ErrNotFound = errors.New("device not found")

const collectionName = "devices"

// MongoRepository implements Repository using MongoDB.
type MongoRepository struct {
	col *mongo.Collection
	log *zap.Logger
}

// NewMongoRepository returns a new MongoDB device repository.
func NewMongoRepository(db *mongo.Database, log *zap.Logger) *MongoRepository {
	return &MongoRepository{col: db.Collection(collectionName), log: log}
}

// EnsureIndexes creates required indexes (idempotent).
func (r *MongoRepository) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "tenant_id", Value: 1}, {Key: "serial_number", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "online", Value: 1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "manufacturer", Value: 1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "last_inform", Value: -1}}},
		{Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "tags", Value: 1}}},
	}
	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("create indexes: %w", err)
	}
	return nil
}

// deviceDoc is the MongoDB document (with ObjectID _id).
type deviceDoc struct {
	ID                        primitive.ObjectID `bson:"_id,omitempty"`
	TenantID                  string             `bson:"tenant_id"`
	SerialNumber              string             `bson:"serial_number"`
	Manufacturer              string             `bson:"manufacturer"`
	OUI                       string             `bson:"oui"`
	ProductClass              string             `bson:"product_class"`
	ModelName                 string             `bson:"model_name"`
	SoftwareVersion           string             `bson:"software_version"`
	HardwareVersion           string             `bson:"hardware_version"`
	IPAddress                 string             `bson:"ip_address"`
	MACAddress                string             `bson:"mac_address"`
	CWMPURL                   string             `bson:"cwmp_url"`
	ConnectionRequestURL      string             `bson:"connection_request_url"`
	ConnectionRequestUsername string             `bson:"connection_request_username"`
	ConnectionRequestPassword string             `bson:"connection_request_password"`
	Online                    bool               `bson:"online"`
	LastInform                primitive.DateTime `bson:"last_inform"`
	LastBoot                  primitive.DateTime `bson:"last_boot"`
	FirstSeen                 primitive.DateTime `bson:"first_seen"`
	BootCount                 int                `bson:"boot_count"`
	LastEvents                []string           `bson:"last_events"`
	Parameters                map[string]string  `bson:"parameters"`
	PON                       *PONInfo           `bson:"pon,omitempty"`
	Tags                      []string           `bson:"tags"`
}

func docToDevice(d *deviceDoc) *Device {
	if d == nil {
		return nil
	}
	dev := &Device{
		TenantID:                  d.TenantID,
		SerialNumber:              d.SerialNumber,
		Manufacturer:              d.Manufacturer,
		OUI:                       d.OUI,
		ProductClass:              d.ProductClass,
		ModelName:                 d.ModelName,
		SoftwareVersion:           d.SoftwareVersion,
		HardwareVersion:           d.HardwareVersion,
		IPAddress:                 d.IPAddress,
		MACAddress:                d.MACAddress,
		CWMPURL:                   d.CWMPURL,
		ConnectionRequestURL:      d.ConnectionRequestURL,
		ConnectionRequestUsername: d.ConnectionRequestUsername,
		ConnectionRequestPassword: d.ConnectionRequestPassword,
		Online:                    d.Online,
		LastInform:                d.LastInform.Time(),
		LastBoot:                  d.LastBoot.Time(),
		FirstSeen:                 d.FirstSeen.Time(),
		BootCount:                 d.BootCount,
		LastEvents:                d.LastEvents,
		Parameters:                d.Parameters,
		PON:                       d.PON,
		Tags:                      d.Tags,
	}
	if d.ID != primitive.NilObjectID {
		dev.ID = d.ID.Hex()
	}
	return dev
}

func deviceToDoc(d *Device) *deviceDoc {
	if d == nil {
		return nil
	}
	doc := &deviceDoc{
		TenantID:                  d.TenantID,
		SerialNumber:              d.SerialNumber,
		Manufacturer:              d.Manufacturer,
		OUI:                       d.OUI,
		ProductClass:              d.ProductClass,
		ModelName:                 d.ModelName,
		SoftwareVersion:           d.SoftwareVersion,
		HardwareVersion:           d.HardwareVersion,
		IPAddress:                 d.IPAddress,
		MACAddress:                d.MACAddress,
		CWMPURL:                   d.CWMPURL,
		ConnectionRequestURL:      d.ConnectionRequestURL,
		ConnectionRequestUsername: d.ConnectionRequestUsername,
		ConnectionRequestPassword: d.ConnectionRequestPassword,
		Online:                    d.Online,
		LastInform:                primitive.NewDateTimeFromTime(d.LastInform),
		LastBoot:                  primitive.NewDateTimeFromTime(d.LastBoot),
		FirstSeen:                 primitive.NewDateTimeFromTime(d.FirstSeen),
		BootCount:                 d.BootCount,
		LastEvents:                d.LastEvents,
		Parameters:                d.Parameters,
		PON:                       d.PON,
		Tags:                      d.Tags,
	}
	if d.ID != "" {
		if oid, err := primitive.ObjectIDFromHex(d.ID); err == nil {
			doc.ID = oid
		}
	}
	return doc
}

func (r *MongoRepository) Upsert(ctx context.Context, d *Device) error {
	filter := bson.M{"tenant_id": d.TenantID, "serial_number": d.SerialNumber}
	doc := deviceToDoc(d)
	now := primitive.NewDateTimeFromTime(time.Now())
	update := bson.M{
		"$set": bson.M{
			"tenant_id":                   doc.TenantID,
			"serial_number":               doc.SerialNumber,
			"manufacturer":                doc.Manufacturer,
			"oui":                         doc.OUI,
			"product_class":               doc.ProductClass,
			"model_name":                  doc.ModelName,
			"software_version":            doc.SoftwareVersion,
			"hardware_version":            doc.HardwareVersion,
			"ip_address":                  doc.IPAddress,
			"mac_address":                 doc.MACAddress,
			"cwmp_url":                    doc.CWMPURL,
			"connection_request_url":      doc.ConnectionRequestURL,
			"connection_request_username": doc.ConnectionRequestUsername,
			"connection_request_password": doc.ConnectionRequestPassword,
			"online":                      doc.Online,
			"last_inform":                 doc.LastInform,
			"last_boot":                   doc.LastBoot,
			"boot_count":                  doc.BootCount,
			"last_events":                 doc.LastEvents,
			"pon":                         doc.PON,
			"tags":                        doc.Tags,
		},
		"$setOnInsert": bson.M{"first_seen": now},
	}
	opts := options.Update().SetUpsert(true)
	res, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("device upsert: %w", err)
	}
	if res.UpsertedCount > 0 {
		d.ID = res.UpsertedID.(primitive.ObjectID).Hex()
		d.FirstSeen = d.LastInform
	}
	return nil
}

func (r *MongoRepository) GetBySerial(ctx context.Context, tenantID, serial string) (*Device, error) {
	filter := bson.M{"tenant_id": tenantID, "serial_number": serial}
	var doc deviceDoc
	err := r.col.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return docToDevice(&doc), nil
}

func (r *MongoRepository) GetByID(ctx context.Context, tenantID, id string) (*Device, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}
	filter := bson.M{"_id": oid, "tenant_id": tenantID}
	var doc deviceDoc
	err = r.col.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return docToDevice(&doc), nil
}

func (r *MongoRepository) List(ctx context.Context, tenantID string, filter DeviceFilter) ([]*Device, int64, error) {
	f := bson.M{"tenant_id": tenantID}
	if filter.Online != nil {
		f["online"] = *filter.Online
	}
	if filter.Manufacturer != "" {
		f["manufacturer"] = filter.Manufacturer
	}
	if filter.ProductClass != "" {
		f["product_class"] = filter.ProductClass
	}
	if len(filter.Tags) > 0 {
		f["tags"] = bson.M{"$all": filter.Tags}
	}
	if filter.Search != "" {
		f["$or"] = []bson.M{
			{"serial_number": bson.M{"$regex": filter.Search, "$options": "i"}},
			{"ip_address": bson.M{"$regex": filter.Search, "$options": "i"}},
			{"model_name": bson.M{"$regex": filter.Search, "$options": "i"}},
		}
	}

	total, err := r.col.CountDocuments(ctx, f)
	if err != nil {
		return nil, 0, err
	}

	limit, offset := filter.Limit, filter.Offset
	if limit <= 0 {
		limit = 50
	}
	opts := options.Find().SetSort(bson.D{{Key: "last_inform", Value: -1}}).SetSkip(offset).SetLimit(limit)
	cursor, err := r.col.Find(ctx, f, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var list []*Device
	for cursor.Next(ctx) {
		var doc deviceDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, err
		}
		list = append(list, docToDevice(&doc))
	}
	return list, total, cursor.Err()
}

func (r *MongoRepository) Delete(ctx context.Context, tenantID, serial string) error {
	filter := bson.M{"tenant_id": tenantID, "serial_number": serial}
	res, err := r.col.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MongoRepository) SetOnline(ctx context.Context, tenantID, serial string, online bool) error {
	filter := bson.M{"tenant_id": tenantID, "serial_number": serial}
	update := bson.M{"$set": bson.M{"online": online, "last_inform": primitive.NewDateTimeFromTime(time.Now())}}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MongoRepository) UpdateParameters(ctx context.Context, tenantID, serial string, params map[string]string) error {
	if len(params) == 0 {
		return nil
	}
	filter := bson.M{"tenant_id": tenantID, "serial_number": serial}
	// Get current parameters and merge. Using dot notation in $set (e.g. "parameters.Device.DeviceInfo.X")
	// would create nested documents; we need flat keys, so we replace the whole "parameters" object.
	var doc struct {
		Parameters map[string]interface{} `bson:"parameters"`
	}
	err := r.col.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrNotFound
		}
		return err
	}
	merged := make(map[string]string)
	for k, v := range doc.Parameters {
		if s, ok := v.(string); ok {
			merged[k] = s
		}
	}
	for k, v := range params {
		merged[k] = v
	}
	update := bson.M{"$set": bson.M{"parameters": merged}}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MongoRepository) UpdateConnectionRequest(ctx context.Context, tenantID, serial string, fields map[string]string) error {
	if len(fields) == 0 {
		return nil
	}
	filter := bson.M{"tenant_id": tenantID, "serial_number": serial}
	set := bson.M{}
	for k, v := range fields {
		set[k] = v
	}
	res, err := r.col.UpdateOne(ctx, filter, bson.M{"$set": set})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}
