package stats_test

import (
	"context"
	"errors"
	"testing"

	"railgorail/avito/internal/entity"
	"railgorail/avito/internal/service/mocks"
	"railgorail/avito/internal/service/stats"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStatsService_GetStatistics_Success(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })
	mockStatsRepo := mocks.NewStatsProvider(t)

	sort := "desc"
	userStatistics := []*entity.UserStatistics{
		{UserID: "dev-a", Username: "Alex", AssignmentCount: 25},
		{UserID: "dev-b", Username: "Boris", AssignmentCount: 20},
		{UserID: "dev-c", Username: "Charles", AssignmentCount: 15},
	}
	prStatistics := &entity.PrStatistics{
		PrCount:   200,
		OpenPrs:   60,
		MergedPrs: 140,
	}

	mockStatsRepo.On("GetAssignmentsCountStats", ctx, sort).Return(userStatistics, nil).Once()
	mockStatsRepo.On("GetPrStats", ctx).Return(prStatistics, nil).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			assert.NoError(t, fn(ctx))
		}).
		Return(nil).
		Once()

	statsSvc := stats.NewStatsService(mockTx, mockStatsRepo)
	result, e := statsSvc.GetStatistics(ctx, sort)

	assert.NoError(t, e)
	assert.NotNil(t, result)
	assert.Len(t, result.User, 3)

	assert.Equal(t, "dev-a", result.User[0].UserID)
	assert.Equal(t, "Alex", result.User[0].Username)
	assert.Equal(t, 25, result.User[0].AssignmentCount)

	assert.Equal(t, "dev-b", result.User[1].UserID)
	assert.Equal(t, "Boris", result.User[1].Username)
	assert.Equal(t, 20, result.User[1].AssignmentCount)

	assert.Equal(t, "dev-c", result.User[2].UserID)
	assert.Equal(t, "Charles", result.User[2].Username)
	assert.Equal(t, 15, result.User[2].AssignmentCount)

	assert.Equal(t, 200, result.Pr.PrCount)
	assert.Equal(t, 60, result.Pr.OpenPrs)
	assert.Equal(t, 140, result.Pr.MergedPrs)
}

func TestStatsService_GetStatistics_EmptyUserStats(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })
	mockStatsRepo := mocks.NewStatsProvider(t)

	sort := "asc"
	userStatistics := []*entity.UserStatistics{}
	prStatistics := &entity.PrStatistics{
		PrCount:   0,
		OpenPrs:   0,
		MergedPrs: 0,
	}

	mockStatsRepo.On("GetAssignmentsCountStats", ctx, sort).Return(userStatistics, nil).Once()
	mockStatsRepo.On("GetPrStats", ctx).Return(prStatistics, nil).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			assert.NoError(t, fn(ctx))
		}).
		Return(nil).
		Once()

	statsSvc := stats.NewStatsService(mockTx, mockStatsRepo)
	result, e := statsSvc.GetStatistics(ctx, sort)

	assert.NoError(t, e)
	assert.NotNil(t, result)
	assert.Empty(t, result.User)
	assert.Equal(t, 0, result.Pr.PrCount)
	assert.Equal(t, 0, result.Pr.OpenPrs)
	assert.Equal(t, 0, result.Pr.MergedPrs)
}

func TestStatsService_GetStatistics_GetAssignmentsCountStatsError(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })
	mockStatsRepo := mocks.NewStatsProvider(t)

	sort := "desc"
	databaseError := errors.New("db connection lost")

	mockStatsRepo.On("GetAssignmentsCountStats", ctx, sort).Return(([]*entity.UserStatistics)(nil), databaseError).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.ErrorIs(t, e, databaseError)
		}).
		Return(databaseError).
		Once()

	statsSvc := stats.NewStatsService(mockTx, mockStatsRepo)
	result, e := statsSvc.GetStatistics(ctx, sort)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.ErrorIs(t, e, databaseError)
}

func TestStatsService_GetStatistics_GetPrStatsError(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })
	mockStatsRepo := mocks.NewStatsProvider(t)

	sort := "asc"
	databaseError := errors.New("pr stats unavailable")
	userStatistics := []*entity.UserStatistics{
		{UserID: "dev-a", Username: "David", AssignmentCount: 9},
	}

	mockStatsRepo.On("GetAssignmentsCountStats", ctx, sort).Return(userStatistics, nil).Once()
	mockStatsRepo.On("GetPrStats", ctx).Return((*entity.PrStatistics)(nil), databaseError).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.ErrorIs(t, e, databaseError)
		}).
		Return(databaseError).
		Once()

	statsSvc := stats.NewStatsService(mockTx, mockStatsRepo)
	result, e := statsSvc.GetStatistics(ctx, sort)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.ErrorIs(t, e, databaseError)
}
