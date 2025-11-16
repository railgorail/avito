package user_test

import (
	"context"
	"errors"
	"testing"

	"railgorail/avito/internal/entity"
	"railgorail/avito/internal/service/mocks"
	userservice "railgorail/avito/internal/service/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_SetIsActive_Success(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })

	mockPrRepo := mocks.NewPrProvider(t)
	mockUserRepo := mocks.NewUserChanger(t)
	mockTeamRepo := mocks.NewTeamIDProvider(t)

	userID := "employee-abc"
	isActive := true
	userEntity := &entity.User{
		ID:       userID,
		Name:     "Anna",
		TeamID:   420,
		IsActive: isActive,
	}
	teamName := "infra-squad"

	mockUserRepo.On("SetIsActive", ctx, userID, isActive).Return(nil).Once()
	mockUserRepo.On("GetById", ctx, userID).Return(userEntity, nil).Once()
	mockTeamRepo.On("GetTeamNameByID", ctx, 420).Return(teamName, nil).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			assert.NoError(t, fn(ctx))
		}).
		Return(nil).
		Once()

	userSvc := userservice.NewUserService(mockTx, mockPrRepo, mockUserRepo, mockTeamRepo)
	result, e := userSvc.SetIsActive(ctx, userID, isActive)

	assert.NoError(t, e)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, "Anna", result.Username)
	assert.Equal(t, teamName, result.TeamName)
	assert.True(t, result.IsActive)
}

func TestUserService_SetIsActive_SetIsActiveError(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })
	mockUserRepo := mocks.NewUserChanger(t)

	userID := "employee-abc"
	isActive := false
	databaseError := errors.New("update operation failed")

	mockUserRepo.On("SetIsActive", ctx, userID, isActive).Return(databaseError).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.ErrorIs(t, e, databaseError)
		}).
		Return(databaseError).
		Once()

	userSvc := userservice.NewUserService(mockTx, nil, mockUserRepo, nil)
	result, e := userSvc.SetIsActive(ctx, userID, isActive)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.ErrorIs(t, e, databaseError)
}

func TestUserService_SetIsActive_GetByIdError(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })
	mockUserRepo := mocks.NewUserChanger(t)

	userID := "employee-abc"
	isActive := true
	databaseError := errors.New("could not find user")

	mockUserRepo.On("SetIsActive", ctx, userID, isActive).Return(nil).Once()
	mockUserRepo.On("GetById", ctx, userID).Return((*entity.User)(nil), databaseError).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.ErrorIs(t, e, databaseError)
		}).
		Return(databaseError).
		Once()

	userSvc := userservice.NewUserService(mockTx, nil, mockUserRepo, nil)
	result, e := userSvc.SetIsActive(ctx, userID, isActive)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.ErrorIs(t, e, databaseError)
}

func TestUserService_SetIsActive_GetTeamNameByIDError(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })
	mockUserRepo := mocks.NewUserChanger(t)
	mockTeamRepo := mocks.NewTeamIDProvider(t)

	userID := "employee-abc"
	isActive := true
	teamID := 999
	databaseError := errors.New("could not find team")

	mockUserRepo.On("SetIsActive", ctx, userID, isActive).Return(nil).Once()
	mockUserRepo.On("GetById", ctx, userID).
		Return(&entity.User{ID: userID, Name: "Boris", TeamID: teamID, IsActive: isActive}, nil).
		Once()
	mockTeamRepo.On("GetTeamNameByID", ctx, teamID).Return("", databaseError).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.ErrorIs(t, e, databaseError)
		}).
		Return(databaseError).
		Once()

	userSvc := userservice.NewUserService(mockTx, nil, mockUserRepo, mockTeamRepo)
	result, e := userSvc.SetIsActive(ctx, userID, isActive)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.ErrorIs(t, e, databaseError)
}

func TestUserService_GetReview_Success_WithPRs(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })
	mockPrRepo := mocks.NewPrProvider(t)
	mockUserRepo := mocks.NewUserChanger(t)

	userID := "employee-def"
	pullRequests := []*entity.PullRequest{
		{
			ID:       "pr-alpha-1",
			Title:    "feat: implement login form",
			AuthorId: userID,
			Status:   "OPEN",
		},
		{
			ID:       "pr-beta-2",
			Title:    "test: cover login service",
			AuthorId: userID,
			Status:   "MERGED",
		},
	}

	mockUserRepo.On("GetById", ctx, userID).Return(&entity.User{ID: userID}, nil).Once()
	mockPrRepo.On("GetUserReviews", ctx, userID).Return(pullRequests, nil).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			assert.NoError(t, fn(ctx))
		}).
		Return(nil).
		Once()

	userSvc := userservice.NewUserService(mockTx, mockPrRepo, mockUserRepo, nil)
	result, e := userSvc.GetReview(ctx, userID)

	assert.NoError(t, e)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)
	assert.Len(t, result.PullRequests, 2)
	assert.Equal(t, "pr-alpha-1", result.PullRequests[0].ID)
	assert.Equal(t, "feat: implement login form", result.PullRequests[0].Name)
	assert.Equal(t, userID, result.PullRequests[0].AuthorID)
	assert.Equal(t, "OPEN", result.PullRequests[0].Status)
	assert.Equal(t, "pr-beta-2", result.PullRequests[1].ID)
	assert.Equal(t, "test: cover login service", result.PullRequests[1].Name)
	assert.Equal(t, "MERGED", result.PullRequests[1].Status)
}

func TestUserService_GetReview_Success_NoPRs(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })
	mockPrRepo := mocks.NewPrProvider(t)
	mockUserRepo := mocks.NewUserChanger(t)

	userID := "employee-ghi"

	mockUserRepo.On("GetById", ctx, userID).Return(&entity.User{ID: userID}, nil).Once()
	mockPrRepo.On("GetUserReviews", ctx, userID).Return([]*entity.PullRequest{}, nil).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			assert.NoError(t, fn(ctx))
		}).
		Return(nil).
		Once()

	userSvc := userservice.NewUserService(mockTx, mockPrRepo, mockUserRepo, nil)
	result, e := userSvc.GetReview(ctx, userID)

	assert.NoError(t, e)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)
	assert.Empty(t, result.PullRequests)
}

func TestUserService_GetReview_GetByIdError(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })
	mockPrRepo := mocks.NewPrProvider(t)
	mockUserRepo := mocks.NewUserChanger(t)

	userID := "employee-jkl"
	databaseError := errors.New("could not find user")

	mockUserRepo.On("GetById", ctx, userID).Return((*entity.User)(nil), databaseError).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.ErrorIs(t, e, databaseError)
		}).
		Return(databaseError).
		Once()

	userSvc := userservice.NewUserService(mockTx, mockPrRepo, mockUserRepo, nil)
	result, e := userSvc.GetReview(ctx, userID)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.ErrorIs(t, e, databaseError)
}

func TestUserService_GetReview_PrProviderError(t *testing.T) {
	ctx := context.Background()
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })
	mockPrRepo := mocks.NewPrProvider(t)
	mockUserRepo := mocks.NewUserChanger(t)

	userID := "employee-mno"
	prError := errors.New("pr service is down")

	mockUserRepo.On("GetById", ctx, userID).Return(&entity.User{ID: userID}, nil).Once()
	mockPrRepo.On("GetUserReviews", ctx, userID).Return(([]*entity.PullRequest)(nil), prError).Once()

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.ErrorIs(t, e, prError)
		}).
		Return(prError).
		Once()

	userSvc := userservice.NewUserService(mockTx, mockPrRepo, mockUserRepo, nil)
	result, e := userSvc.GetReview(ctx, userID)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.ErrorIs(t, e, prError)
}
