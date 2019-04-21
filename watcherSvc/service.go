package watcherSvc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DenysNahurnyi/deal/pb/generated/pb"
	"github.com/go-kit/kit/log"

	"github.com/mongodb/mongo-go-driver/mongo"
)

type service struct {
	envType       string
	queueTable    *mongo.Collection
	dataSvcClient *pb.DataServiceClient
	dT            *DealTimer
}

type DealTimer struct {
	timer        *time.Timer
	turnOffTimer chan bool
	m            sync.Mutex
	currentDeal  string
}

// NewService creates new service of watchSvc that allows to call it's functions to handle watcherSvc domain
func NewService(ctx context.Context, logger log.Logger, mgc *mongo.Client, dataSvcClient *pb.DataServiceClient) (Service, error) {
	// If this service falled, run timer on the first deal on start
	s := &service{
		envType:       "test",
		dataSvcClient: dataSvcClient,
		queueTable:    mgc.Database("travel").Collection("dealWaitingQueue"),
		dT: &DealTimer{
			timer: nil,
			m:     sync.Mutex{},
		},
	}
	deal, err := GetFirstDeal(ctx, s.queueTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to get the first deal from the queue: ", err)
		return nil, err
	}
	if deal != nil {
		fmt.Println("{DEBUG}", "Service got first deal "+deal.ID.Hex()+" from queue and will create timer for that")
		err = UpdateStatus(ctx, deal.ID.Hex(), "WATCHING", s.queueTable)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to update deal status: ", err)
			return nil, err
		}
		go s.runTimer(ctx, deal.Timeout, deal.ID.Hex())
	}
	return s, nil
}

func isClosed(ch <-chan bool) bool {
	select {
	case <-ch:
		return true
	default:
	}

	return false
}

func (s *service) runTimer(ctx context.Context, timeout time.Time, dealQueueID string) {
	// Check if another goroutine running timer
	if s.dT.turnOffTimer != nil && !isClosed(s.dT.turnOffTimer) {
		if len(s.dT.currentDeal) != 0 {
			err := UpdateStatus(ctx, s.dT.currentDeal, "BACK_TO_QUEUE", s.queueTable)
			if err != nil {
				// Normal case
				fmt.Println("[LOG]:", "Failed to update deal status: ", err)
				return
			}
		}
		fmt.Println("{DEBUG}", "Service closed timer")
		// If another goroutine running timer, close it
		close(s.dT.turnOffTimer)
	}
	s.dT.turnOffTimer = make(chan bool)
	s.dT.timer = time.NewTimer(time.Until(timeout))
	s.dT.currentDeal = dealQueueID
	fmt.Println("{DEBUG}", "Service created new timer for ", time.Until(timeout))

	select {
	case <-s.dT.timer.C:
		fmt.Println("{DEBUG}", "timer expired ")
		s.dT.m.Lock()
		defer s.dT.m.Unlock()
		// Since this goroutine will be closed, {turnOffTimer} is not more possible to use
		close(s.dT.turnOffTimer)
		// Get first deal from DB
		deal, err := GetFirstDeal(ctx, s.queueTable)
		if err != nil {
			fmt.Println("[LOG]:", "Can't get the first deal from queue: ", err)
			return
		}
		if deal == nil {
			// Normal case
			fmt.Println("[LOG]:", "No deal in queue")
			return
		}
		fmt.Println("{DEBUG}", "Get first deal on timer expire: ", deal.ID.Hex())
		fmt.Println("{DEBUG}", "Deal timeout: ", deal.Timeout)
		fmt.Println("{DEBUG}", "Timer timeout: ", timeout)
		if deal.Timeout == timeout {
			fmt.Println("[LOG]:", "Deal "+deal.DealID+" timeout happened")
			// Call dataSvc to update deal status
			// In another case this timer is obsolete
			// Start new timer:
			//--Update last deal status
			err = UpdateStatus(ctx, deal.ID.Hex(), "PROCESSED", s.queueTable)
			if err != nil {
				// Normal case
				fmt.Println("[LOG]:", "Failed to update deal status: ", err)
				return
			}
		}
		//--Create timer for the new one
		deal, err = GetFirstDeal(ctx, s.queueTable)
		if err != nil {
			fmt.Println("[LOG]:", "Can't get the first deal from queue: ", err)
			return
		}
		if deal != nil {
			fmt.Println("[LOG]:", "Create timer for the next deal")
			err = UpdateStatus(ctx, deal.ID.Hex(), "WATCHING", s.queueTable)
			if err != nil {
				fmt.Println("[LOG]:", "Failed to update deal status: ", err)
				return
			}
			go s.runTimer(ctx, deal.Timeout, deal.ID.Hex())
		} else {
			fmt.Println("[LOG]:", "No deals left in queue")
		}

	case <-s.dT.turnOffTimer:
		fmt.Println("[LOG]:", "Timer has to be recreated")
	}
}

type Service interface {
	HoldAndWatch(ctx context.Context, dealID, timeout string) error
}

func (s *service) HoldAndWatch(ctx context.Context, dealID, timeoutStr string) error {
	// Check if timeout valid
	LAYOUT := "2006-01-02T15:04:05.000Z"
	timeout, err := time.Parse(LAYOUT, timeoutStr)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to parse timeout for deal "+dealID+": ", err)
		return err
	}
	deal := &DealDB{
		DealID:  dealID,
		Timeout: timeout,
		Status:  "RECEIVER_FROM_DATASVC",
	}
	updateTimer, err := PutDealToQueue(ctx, deal, s.queueTable)
	if err != nil {
		fmt.Println("[LOG]:", "Failed to add deal "+dealID+" to the queue: ", err)
		return err
	}
	fmt.Println("{DEBUG}", "Deal placed in queue, need to update timer: ", updateTimer)
	if updateTimer {
		fmt.Println("{DEBUG}", "Update timer with timeout: ", deal.Timeout)
		err = UpdateStatus(ctx, deal.ID.Hex(), "WATCHING", s.queueTable)
		if err != nil {
			fmt.Println("[LOG]:", "Failed to update deal status: ", err)
			return err
		}
		go s.runTimer(ctx, deal.Timeout, deal.ID.Hex())
	}
	return nil
}
