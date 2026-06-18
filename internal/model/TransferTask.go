package model

import "search-gin/pkg/types"

type TransferTaskModel = types.TransferTaskModel

const (
	TaskTypeCut   = types.TaskTypeCut
	TaskTypeMerge = types.TaskTypeMerge
	TaskTypeTrans = types.TaskTypeTrans
)

const (
	StatusPending   = types.StatusPending
	StatusExecuting = types.StatusExecuting
	StatusCompleted = types.StatusCompleted
	StatusFailed    = types.StatusFailed
	StatusCancelled = types.StatusCancelled
)

const UndefinedStr = types.UndefinedStr

var NewMergeTask = types.NewMergeTask
var NewTask = types.NewTask
var NewCutTask = types.NewCutTask
