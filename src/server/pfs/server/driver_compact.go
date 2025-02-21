package server

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pachyderm/pachyderm/v2/src/internal/backoff"
	"github.com/pachyderm/pachyderm/v2/src/internal/errors"
	"github.com/pachyderm/pachyderm/v2/src/internal/storage/fileset"
	"github.com/pachyderm/pachyderm/v2/src/internal/storage/fileset/index"
	"github.com/pachyderm/pachyderm/v2/src/internal/work"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func (d *driver) compact(ctx context.Context, ids []fileset.ID) (*fileset.ID, error) {
	return d.storage.CompactLevelBased(ctx, ids, defaultTTL, func(ctx context.Context, ids []fileset.ID, ttl time.Duration) (*fileset.ID, error) {
		var id *fileset.ID
		if err := d.compactionQueue.RunTaskBlock(ctx, func(master *work.Master) error {
			workerFunc := func(ctx context.Context, tasks []fileset.CompactionTask) ([]fileset.ID, error) {
				workTasks := make([]*work.Task, len(tasks))
				for i, task := range tasks {
					serInputs := make([]string, len(task.Inputs))
					for i := range task.Inputs {
						serInputs[i] = task.Inputs[i].HexString()
					}
					any, err := serializeCompactionTask(&CompactionTask{
						Inputs: serInputs,
						Range: &PathRange{
							Lower: task.PathRange.Lower,
							Upper: task.PathRange.Upper,
						},
					})
					if err != nil {
						return nil, err
					}
					workTasks[i] = &work.Task{Data: any}
				}
				var results []fileset.ID
				if err := master.RunSubtasks(workTasks, func(_ context.Context, taskInfo *work.TaskInfo) error {
					if taskInfo.Result == nil {
						return errors.Errorf("no result set for compaction work.TaskInfo")
					}
					res, err := deserializeCompactionResult(taskInfo.Result)
					if err != nil {
						return err
					}
					id, err := fileset.ParseID(res.Id)
					if err != nil {
						return err
					}
					results = append(results, *id)
					return nil
				}); err != nil {
					return nil, err
				}
				return results, nil
			}
			dc := fileset.NewDistributedCompactor(d.storage, d.env.Config().StorageCompactionMaxFanIn, workerFunc)
			var err error
			id, err = dc.Compact(master.Ctx(), ids, ttl)
			return err
		}); err != nil {
			return nil, err
		}
		return id, nil
	})
}

func (d *driver) compactionWorker() {
	ctx := context.Background()
	w := work.NewWorker(d.etcdClient, d.prefix, storageTaskNamespace)
	err := backoff.RetryNotify(func() error {
		return w.Run(ctx, func(ctx context.Context, subtask *work.Task) (*types.Any, error) {
			task, err := deserializeCompactionTask(subtask.Data)
			if err != nil {
				return nil, err
			}
			ids := []fileset.ID{}
			for _, input := range task.Inputs {
				id, err := fileset.ParseID(input)
				if err != nil {
					return nil, err
				}
				ids = append(ids, *id)
			}
			pathRange := &index.PathRange{
				Lower: task.Range.Lower,
				Upper: task.Range.Upper,
			}
			id, err := d.storage.Compact(ctx, ids, defaultTTL, index.WithRange(pathRange))
			if err != nil {
				return nil, err
			}
			return serializeCompactionResult(&CompactionTaskResult{
				Id: id.HexString(),
			})
		})
	}, backoff.NewInfiniteBackOff(), func(err error, _ time.Duration) error {
		log.Printf("error in compaction worker: %v", err)
		return nil
	})
	// Never ending backoff should prevent us from getting here.
	panic(err)
}

func serializeCompactionTask(task *CompactionTask) (*types.Any, error) {
	data, err := proto.Marshal(task)
	if err != nil {
		return nil, err
	}
	return &types.Any{
		TypeUrl: "/" + proto.MessageName(task),
		Value:   data,
	}, nil
}

func deserializeCompactionTask(taskAny *types.Any) (*CompactionTask, error) {
	task := &CompactionTask{}
	if err := types.UnmarshalAny(taskAny, task); err != nil {
		return nil, err
	}
	return task, nil
}

func serializeCompactionResult(res *CompactionTaskResult) (*types.Any, error) {
	data, err := proto.Marshal(res)
	if err != nil {
		return nil, err
	}
	return &types.Any{
		TypeUrl: "/" + proto.MessageName(res),
		Value:   data,
	}, nil
}

func deserializeCompactionResult(any *types.Any) (*CompactionTaskResult, error) {
	res := &CompactionTaskResult{}
	if err := types.UnmarshalAny(any, res); err != nil {
		return nil, err
	}
	return res, nil
}
