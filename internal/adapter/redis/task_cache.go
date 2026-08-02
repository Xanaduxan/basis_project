package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	redislib "github.com/redis/go-redis/v9"

	"basisProject/internal/domain"
)

type TaskCache struct {
	client *redislib.Client
	ttl    time.Duration
}

func NewTaskCache(
	client *redislib.Client,
	ttl time.Duration,
) *TaskCache {
	return &TaskCache{
		client: client,
		ttl:    ttl,
	}
}

func (c *TaskCache) Get(
	ctx context.Context,
	filter domain.TaskFilter,
) (
	tasks []domain.Task,
	hit bool,
	version int64,
	err error,
) {
	version, err = c.currentVersion(
		ctx,
		filter.TeamID,
	)
	if err != nil {
		return nil, false, 0, fmt.Errorf(
			"get task cache version: %w",
			err,
		)
	}

	key := taskListKey(filter, version)

	value, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redislib.Nil) {
			return nil, false, version, nil
		}

		return nil, false, version, fmt.Errorf(
			"get cached tasks: %w",
			err,
		)
	}

	if err := json.Unmarshal(value, &tasks); err != nil {
		return nil, false, version, fmt.Errorf(
			"decode cached tasks: %w",
			err,
		)
	}

	if tasks == nil {
		tasks = make([]domain.Task, 0)
	}

	return tasks, true, version, nil
}

func (c *TaskCache) Set(
	ctx context.Context,
	filter domain.TaskFilter,
	version int64,
	tasks []domain.Task,
) error {
	value, err := json.Marshal(tasks)
	if err != nil {
		return fmt.Errorf(
			"encode tasks for cache: %w",
			err,
		)
	}

	key := taskListKey(filter, version)

	if err := c.client.Set(
		ctx,
		key,
		value,
		c.ttl,
	).Err(); err != nil {
		return fmt.Errorf(
			"store tasks in cache: %w",
			err,
		)
	}

	return nil
}

func (c *TaskCache) InvalidateTeam(
	ctx context.Context,
	teamID int64,
) error {
	if err := c.client.Incr(
		ctx,
		taskVersionKey(teamID),
	).Err(); err != nil {
		return fmt.Errorf(
			"increment task cache version: %w",
			err,
		)
	}

	return nil
}

func (c *TaskCache) currentVersion(
	ctx context.Context,
	teamID int64,
) (int64, error) {
	version, err := c.client.Get(
		ctx,
		taskVersionKey(teamID),
	).Int64()
	if err != nil {
		if errors.Is(err, redislib.Nil) {
			return 0, nil
		}

		return 0, err
	}

	return version, nil
}

func taskVersionKey(teamID int64) string {
	return fmt.Sprintf(
		"tasks:team:%d:version",
		teamID,
	)
}

func taskListKey(
	filter domain.TaskFilter,
	version int64,
) string {
	status := "all"

	if filter.Status != nil {
		status = string(*filter.Status)
	}

	assignee := "all"

	if filter.AssigneeID != nil {
		assignee = strconv.FormatInt(
			*filter.AssigneeID,
			10,
		)
	}

	return fmt.Sprintf(
		"tasks:team:%d:v:%d:status:%s:assignee:%s:limit:%d:offset:%d",
		filter.TeamID,
		version,
		status,
		assignee,
		filter.Limit,
		filter.Offset,
	)
}
