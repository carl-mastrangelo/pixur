package main

import (
	"log"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/tasks"
	"pixur.org/pixur/tools/batch"
)

func main() {
	err := batch.ForEachPic(func(p *schema.Pic, sc *batch.ServerConfig, err error) error {
		if err != nil {
			return err
		}
		now := time.Now()
		// No deletion info
		if p.DeletionStatus == nil {
			return nil
		}
		// Some deletion info, but it isn't on the chopping block.
		if p.DeletionStatus.PendingDeletedTs == nil {
			return nil
		}
		// It was already hard deleted, ignore it
		if p.DeletionStatus.ActualDeletedTs != nil {
			return nil
		}

		pendingTime := schema.ToTime(p.DeletionStatus.PendingDeletedTs)
		// It is pending deletion, just not yet.
		if !now.After(pendingTime) {
			return nil
		}

		log.Println("Preparing to delete", p.GetVarPicID(), pendingTime)
		var task = &tasks.HardDeletePicTask{
			DB:      sc.DB,
			PixPath: sc.PixPath,
			PicID:   p.PicId,
		}
		runner := new(tasks.TaskRunner)
		if err := runner.Run(task); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Println(err)
	}
}
