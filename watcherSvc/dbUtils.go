package watcherSvc

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/mongo"
)

type DealDB struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	DealID  string             `bson:"deal_id,omitempty"`
	Timeout time.Time          `bson:"timeout,omitempty"`
	Status  string             `bson:"status,omitempty"`
}

func (d *DealDB) toMongoFormat() bson.D {
	es := []bson.E{}
	if len(d.ID) > 0 {
		es = append(es, bson.E{Key: "_id", Value: d.ID})
	}
	if len(d.DealID) > 0 {
		es = append(es, bson.E{Key: "deal_id", Value: d.DealID})
	}
	if len(d.Timeout.String()) != 0 {
		es = append(es, bson.E{Key: "timeout", Value: d.Timeout.String()})
	}
	if len(d.Status) > 0 {
		es = append(es, bson.E{Key: "status", Value: d.Status})
	}
	return es
}

// Put deal to the queue and return {needToUpdateTimer}
func PutDealToQueue(ctx context.Context, deal *DealDB, table *mongo.Collection) (needToUpdateTimer bool, err error) {
	firstDeal, err := GetFirstDeal(ctx, table)
	if err != nil {
		fmt.Println("Error getting first deal from mongo: ", err)
		return false, err
	}
	if firstDeal != nil {
		needToUpdateTimer = firstDeal.Timeout.Before(deal.Timeout)
	} else {
		needToUpdateTimer = true
	}
	// Add new deal
	_, err = table.InsertOne(ctx, deal)
	if err != nil {
		fmt.Println("Error adding new deal to the queue in mongo: ", err)
		return false, err
	}
	return needToUpdateTimer, err
}

func GetFirstDeal(ctx context.Context, table *mongo.Collection) (*DealDB, error) {
	deals := []*DealDB{}
	// Get WATCHING deal
	watchingDeal := &DealDB{}

	err := table.FindOne(ctx, bson.D{{Key: "status", Value: "WATCHING"}}).Decode(watchingDeal)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			// it's ok
		} else {
			fmt.Println("Error getting user from mongo: ", err)
			return nil, err
		}
	}
	if len(watchingDeal.DealID) != 0 {
		return watchingDeal, nil
	}
	// Get all deals is no watching deals
	cursor, err := table.Find(ctx, bson.D{{Key: "status", Value: "RECEIVER_FROM_DATASVC"}})
	if err != nil {
		fmt.Println("Error getting deals from mongo: ", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		d := &DealDB{}
		if err := cursor.Decode(d); err != nil {
			fmt.Println("Error getting deals from mongo: ", err)
			return nil, err
		}
		deals = append(deals, d)
	}
	fmt.Println("Deals in DB: ", len(deals))
	// Sort in the right order and get {needToUpdateTimer} value
	if len(deals) != 0 {
		sort.Slice(deals, func(i, j int) bool {
			return deals[i].Timeout.Before(deals[i].Timeout)
		})
		fmt.Println("reutrn deal: ", deals[0])
		return deals[0], nil
	} else {
		return nil, nil
	}
}

// UpdateStatus updates deal status
func UpdateStatus(ctx context.Context, dealID, status string, table *mongo.Collection) error {
	id, err := primitive.ObjectIDFromHex(dealID)
	if err != nil {
		fmt.Println("Error creating object id to get user: ", err)
		return err
	}
	_, err = table.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: id}},
		bson.D{{"$set", bson.D{{Key: "status", Value: status}}}},
	)
	if err != nil {
		fmt.Println("Error updating user in mongo: ", err)
	}
	return err
}
