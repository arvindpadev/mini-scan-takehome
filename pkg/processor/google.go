package processor

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"regexp"

	"cloud.google.com/go/bigtable"

	"github.com/censys/scan-takehome/pkg/shared"
)

// NOTE: This is my first time working with google cloud
// I have not had a chance to look into the details of
// connection pooling and possibly some other details
// related to bigtable. Hence this implementation is
// going to focus on the meeting the basic requirements
// BigTable has been chosen since it fits nicely for the
// requirements to be able to handle a large number of
// concurrent writes, and it can handle the load from
// horizontally scaling the services
type bigTableStore struct {
	projectId  string
	instanceId string
}

func (b *bigTableStore) StoreScan(ctx context.Context, scan *StorableScan) error {
	key := fmt.Sprintf("%s %d %s", scan.ip, scan.port, scan.service)
	client, errClient := bigtable.NewClient(ctx, b.projectId, b.instanceId)
	if errClient != nil {
		log.Fatalf("Failed to create Bigtable client: %v", errClient)
	}

	defer client.Close()

	table := client.Open(shared.TableName)
	timestampFilter := bigtable.FamilyFilter(shared.TimestampColumn)
	timestampReadOption := bigtable.RowFilter(timestampFilter)
	return optimisticLockingLikeMutateWithRetry(ctx, table, key, timestampFilter, timestampReadOption, scan)
}

func NewBigTableStore(projectId *string, instanceId *string) Store {
	return &bigTableStore{
		projectId:  *projectId,
		instanceId: *instanceId,
	}
}

/***
* Since BigTable does not support optimistic locking, we attempt to
* use the timestamp column for optimistic locking purposes. The
* attempts are not for the purposes of a retry strategy per se. If
* the optimistic lock condition fails on the off chance that an
* older timestamp than the current one, but newer than the one read
* in the ReadRow invocation caused this, it would be nice to quickly
* reattempt instead of placing an avoidable upstream burden by NACK-ing
* the message from the pubsub based on the error encountered. The more
* common case is that the second ReadRow invocation finds that this change
* is no longer needed. This minor optimization should help when there are
* a high number of updates coming in for the same key, likely when overall
* traffic is bursty
 */
func optimisticLockingLikeMutateWithRetry(ctx context.Context, table *bigtable.Table, key string, timestampFilter bigtable.Filter, timestampReadOption bigtable.ReadOption, scan *StorableScan) error {
	var previousAttemptError error
	for attempt := 1; attempt <= 2; attempt = attempt + 1 {
		row, errRow := table.ReadRow(ctx, key, timestampReadOption)
		if errRow != nil {
			return errRow
		}

		if row == nil && attempt == 1 {
			shouldRetry, err := writeNewRow(ctx, table, key, timestampFilter, scan)
			previousAttemptError = err
			switch {
			case err != nil && !shouldRetry:
				return err
			case err == nil:
				break
			}
		} else if row == nil && attempt > 1 {
			// The first attempt failed because a row existed during creation - this should never happen
			return previousAttemptError
		} else {
			currentTimestamp := int64(binary.BigEndian.Uint64(row[shared.TimestampColumn][0].Value))
			if currentTimestamp < scan.timestamp {
				shouldRetry, err := updateExistingRow(ctx, table, key, timestampFilter, scan, row[shared.TimestampColumn][0].Value)
				previousAttemptError = err
				switch {
				case err != nil && (attempt > 1 || !shouldRetry):
					return err
				case err == nil:
					break
				}
			} else {
				break
			}
		}
	}

	return nil
}

func writeNewRow(ctx context.Context, table *bigtable.Table, key string, timestampFilter bigtable.Filter, scan *StorableScan) (bool, error) {
	creationMut := newScanMutation(scan)
	mut := bigtable.NewCondMutation(timestampFilter, nil, creationMut)

	var conditionMatched bool
	err := table.Apply(ctx, key, mut, bigtable.GetCondMutationResult(&conditionMatched))
	if err != nil {
		return false, err
	}

	if conditionMatched {
		return true, fmt.Errorf("scan %v could not be created due to condition match failure. A scan with the same key %s already exists", *scan, key)
	}

	return false, nil
}

func updateExistingRow(ctx context.Context, table *bigtable.Table, key string, timestampFilter bigtable.Filter, scan *StorableScan, currentTimestamp []byte) (bool, error) {
	timestampPattern := regexp.QuoteMeta(string(currentTimestamp))
	timestampValueFilter := bigtable.ValueFilter(timestampPattern)
	filter := bigtable.ChainFilters(timestampFilter, timestampValueFilter)
	updateMut := newUpdateScanMutation(scan)
	mut := bigtable.NewCondMutation(filter, updateMut, nil)
	var conditionMatched bool
	err := table.Apply(ctx, key, mut, bigtable.GetCondMutationResult(&conditionMatched))
	if err != nil {
		return false, err
	}

	if !conditionMatched {
		return true, fmt.Errorf("scan %v could not be updated due to condition match failure on filters %v and %v", *scan, timestampFilter, timestampValueFilter)
	}

	return false, nil
}

func newScanMutation(scan *StorableScan) *bigtable.Mutation {
	mut := newUpdateScanMutation(scan)
	mut.Set(shared.IpColumn, shared.IpColumn, bigtable.ServerTime, []byte(scan.ip))
	mut.Set(shared.ServiceColumn, shared.ServiceColumn, bigtable.ServerTime, []byte(scan.service))

	portBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(portBytes, scan.port)
	mut.Set(shared.PortColumn, shared.PortColumn, bigtable.ServerTime, portBytes)
	return mut
}

func newUpdateScanMutation(scan *StorableScan) *bigtable.Mutation {
	mut := bigtable.NewMutation()
	mut.Set(shared.DataColumn, shared.DataColumn, bigtable.ServerTime, []byte(scan.data))

	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(scan.timestamp))
	mut.Set(shared.TimestampColumn, shared.TimestampColumn, bigtable.ServerTime, timestampBytes)
	return mut
}
