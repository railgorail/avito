package team_test

import (
	"context"
	"errors"
	"testing"

	"railgorail/avito/internal/entity"
	"railgorail/avito/internal/repo"
	"railgorail/avito/internal/service/mocks"
	"railgorail/avito/internal/service/team"
	"railgorail/avito/internal/transport/http/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTeamService_Add_Success(t *testing.T) {
	ctx := context.Background()
	mockTeamRepo := mocks.NewTeamProvider(t)
	mockUserRepo := mocks.NewUserProvider(t)
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })

	teamName := "backend-guild"
	users := []dto.TeamMember{
		{UserID: "usr-a-1", Username: "Anton", IsActive: true},
		{UserID: "usr-b-2", Username: "Stepan", IsActive: false},
	}
	teamID := 123

	mockTeamRepo.On("Create", ctx, teamName).Return(teamID, nil)
	mockUserRepo.On("Save", ctx, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == "usr-a-1" && u.Name == "Anton" && u.TeamID == teamID && u.IsActive
	})).Return("", nil)
	mockUserRepo.On("Save", ctx, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == "usr-b-2" && u.Name == "Stepan" && u.TeamID == teamID && !u.IsActive
	})).Return("", nil)

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			assert.NoError(t, fn(ctx))
		}).
		Return(nil).Once()

	teamSvc := team.NewTeamService(mockTx, mockTeamRepo, mockUserRepo, nil)
	result, e := teamSvc.Add(ctx, teamName, users)

	assert.NoError(t, e)
	assert.Equal(t, teamName, result.TeamName)
	assert.Len(t, result.Members, 2)
	assert.Equal(t, "Anton", result.Members[0].Username)
	assert.True(t, result.Members[0].IsActive)
}

func TestTeamService_Add_TeamExists(t *testing.T) {
	ctx := context.Background()
	mockTeamRepo := mocks.NewTeamProvider(t)
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })

	teamName := "backend-guild"
	users := []dto.TeamMember{{UserID: "usr-a-1", Username: "Fedor", IsActive: true}}

	mockTeamRepo.On("Create", ctx, teamName).Return(0, repo.ErrTeamExists)

	mockTx.On("Do", ctx, mock.Anything).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.ErrorIs(t, e, repo.ErrTeamExists)
		}).
		Return(repo.ErrTeamExists).
		Once()

	teamSvc := team.NewTeamService(mockTx, mockTeamRepo, nil, nil)
	result, e := teamSvc.Add(ctx, teamName, users)

	assert.Nil(t, result)
	assert.ErrorIs(t, e, repo.ErrTeamExists)
}

func TestTeamService_Get_Success(t *testing.T) {
	ctx := context.Background()
	mockTeamRepo := mocks.NewTeamProvider(t)
	mockUserRepo := mocks.NewUserProvider(t)
	teamName := "frontend-guild"
	teamEntity := &entity.Team{ID: 99, Name: teamName}
	users := []*entity.User{
		{ID: "frontend-lead", Name: "Leonid", TeamID: 99, IsActive: true},
		{ID: "senior-frontend", Name: "Olga", TeamID: 99, IsActive: false},
	}

	mockTeamRepo.On("GetByTeamName", ctx, teamName).Return(teamEntity, nil)
	mockUserRepo.On("GetUsersInTeam", ctx, teamName).Return(users, nil)

	teamSvc := team.NewTeamService(nil, mockTeamRepo, mockUserRepo, nil)
	result, e := teamSvc.Get(ctx, teamName)

	assert.NoError(t, e)
	assert.Equal(t, teamName, result.TeamName)
	assert.Len(t, result.Members, 2)
}

func TestTeamService_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	mockTeamRepo := mocks.NewTeamProvider(t)
	teamName := "qa-team"

	mockTeamRepo.On("GetByTeamName", ctx, teamName).Return((*entity.Team)(nil), repo.ErrNotFound)

	teamSvc := team.NewTeamService(nil, mockTeamRepo, nil, nil)
	result, e := teamSvc.Get(ctx, teamName)

	assert.Nil(t, result)
	assert.ErrorIs(t, e, repo.ErrNotFound)
}

func TestTeamService_Add_SaveError(t *testing.T) {
	ctx := context.Background()
	mockTeamRepo := mocks.NewTeamProvider(t)
	mockUserRepo := mocks.NewUserProvider(t)
	mockTx := &mocks.MockManager{}
	mockTx.Test(t)
	t.Cleanup(func() { mockTx.AssertExpectations(t) })

	teamName := "sre-team"
	users := []dto.TeamMember{
		{UserID: "usr-a-1", Username: "Boris", IsActive: true},
		{UserID: "usr-b-2", Username: "Konstantin", IsActive: true},
	}
	teamID := 77
	storageError := errors.New("storage error")

	mockTeamRepo.On("Create", ctx, teamName).Return(teamID, nil)
	mockUserRepo.On("Save", ctx, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == "usr-a-1" && u.Name == "Boris" && u.TeamID == teamID && u.IsActive
	})).Return("", nil)
	mockUserRepo.On("Save", ctx, mock.MatchedBy(func(u *entity.User) bool {
		return u.ID == "usr-b-2" && u.Name == "Konstantin" && u.TeamID == teamID && u.IsActive
	})).Return("", storageError)

	mockTx.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.ErrorIs(t, e, storageError)
		}).
		Return(storageError).
		Once()

	teamSvc := team.NewTeamService(mockTx, mockTeamRepo, mockUserRepo, nil)
	result, e := teamSvc.Add(ctx, teamName, users)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.ErrorIs(t, e, storageError)
}

func TestTeamService_Get_GetUsersError(t *testing.T) {
	ctx := context.Background()
	mockTeamRepo := mocks.NewTeamProvider(t)
	mockUserRepo := mocks.NewUserProvider(t)
	teamName := "frontend-guild-err"
	teamEntity := &entity.Team{ID: 99, Name: teamName}
	fetchError := errors.New("could not fetch users")

	mockTeamRepo.On("GetByTeamName", ctx, teamName).Return(teamEntity, nil)
	mockUserRepo.On("GetUsersInTeam", ctx, teamName).Return(([]*entity.User)(nil), fetchError)

	teamSvc := team.NewTeamService(nil, mockTeamRepo, mockUserRepo, nil)
	result, e := teamSvc.Get(ctx, teamName)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.ErrorIs(t, e, fetchError)
}
