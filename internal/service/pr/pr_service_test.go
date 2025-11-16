package pr_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"railgorail/avito/internal/entity"
	"railgorail/avito/internal/repo"
	"railgorail/avito/internal/service/mocks"
	"railgorail/avito/internal/service/pr"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPullRequestService_Create_Success_NoAvailableCandidates(t *testing.T) {
	ctx := context.Background()
	prID := "pr-alpha"
	prName := "feat: initial commit"
	authorID := "author-10"
	teamID := 100

	mockPr := mocks.NewPrController(t)
	mockUser := mocks.NewUserGetter(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	authorUser := &entity.User{ID: authorID, TeamID: teamID}
	activeUserIDs := []string{authorID}

	mockPr.On("Create", ctx, mock.AnythingOfType("*entity.PullRequest")).Return(prID, nil).Once()
	mockUser.On("GetById", ctx, authorID).Return(authorUser, nil).Once()
	mockUser.On("GetActiveUsersIDInTeam", ctx, teamID).Return(activeUserIDs, nil).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			assert.NoError(t, fn(ctx))
		}).Return(nil).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, mockUser)
	result, e := service.Create(ctx, prID, prName, authorID)

	assert.NoError(t, e)
	assert.NotNil(t, result)
	assert.Equal(t, prID, result.ID)
	assert.Equal(t, prName, result.Name)
	assert.Equal(t, authorID, result.AuthorID)
	assert.Equal(t, pr.StatusOpen, result.Status)
	assert.Empty(t, result.AssignedReviewers)
}

func TestPullRequestService_Create_AssignReviewerFailsOnSecond(t *testing.T) {
	ctx := context.Background()
	prID := "pr-beta"
	prName := "fix: critical bug"
	authorID := "author-10"
	teamID := 101

	mockPr := mocks.NewPrController(t)
	mockUser := mocks.NewUserGetter(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	authorUser := &entity.User{ID: authorID, TeamID: teamID}
	activeUserIDs := []string{"rev-20", "rev-30", "rev-40"}

	mockPr.On("Create", ctx, mock.AnythingOfType("*entity.PullRequest")).Return(prID, nil).Once()
	mockUser.On("GetById", ctx, authorID).Return(authorUser, nil).Once()
	mockUser.On("GetActiveUsersIDInTeam", ctx, teamID).Return(activeUserIDs, nil).Once()

	assignError := errors.New("failed to assign")
	mockReviewer.
		On("AssignReviewer", ctx, prID, mock.AnythingOfType("string")).
		Return(nil).Once()
	mockReviewer.
		On("AssignReviewer", ctx, prID, mock.AnythingOfType("string")).
		Return(assignError).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.Equal(t, assignError, e)
		}).Return(assignError).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, mockUser)
	result, e := service.Create(ctx, prID, prName, authorID)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.Equal(t, assignError, e)
}

func TestPullRequestService_Create_GetActiveUsersError(t *testing.T) {
	ctx := context.Background()
	prID := "pr-gamma"
	prName := "bug: user service down"
	authorID := "author-10"
	teamID := 102

	mockPr := mocks.NewPrController(t)
	mockUser := mocks.NewUserGetter(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	authorUser := &entity.User{ID: authorID, TeamID: teamID}
	activeError := errors.New("user service unavailable")

	mockUser.On("GetById", ctx, authorID).Return(authorUser, nil).Once()
	mockUser.On("GetActiveUsersIDInTeam", ctx, teamID).Return(([]string)(nil), activeError).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.Equal(t, activeError, e)
		}).Return(activeError).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, mockUser)
	result, e := service.Create(ctx, prID, prName, authorID)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.Equal(t, activeError, e)
	mockPr.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	mockReviewer.AssertNotCalled(t, "AssignReviewer", mock.Anything, mock.Anything, mock.Anything)
}

func TestPullRequestService_Merge_Success_FromOpen(t *testing.T) {
	ctx := context.Background()
	prID := "merge-req-1"

	mockPr := mocks.NewPrController(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	now := time.Now()
	openPR := &entity.PullRequest{ID: prID, Title: "docs: update README", AuthorId: "author-a", Status: pr.StatusOpen}
	mergedPR := &entity.PullRequest{ID: prID, Title: "docs: update README", AuthorId: "author-a", Status: pr.StatusMerged, MergedAt: &now}
	reviewerIDs := []string{"reviewer-r1", "reviewer-r2"}

	mockPr.On("GetById", ctx, prID).Return(openPR, nil).Once()
	mockPr.On("MarkAsMerged", ctx, prID).Return(nil).Once()
	mockPr.On("GetById", ctx, prID).Return(mergedPR, nil).Once()
	mockReviewer.On("GetPrReviewers", ctx, prID).Return(reviewerIDs, nil).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			assert.NoError(t, fn(ctx))
		}).Return(nil).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, nil)
	result, e := service.Merge(ctx, prID)

	assert.NoError(t, e)
	assert.NotNil(t, result)
	assert.Equal(t, prID, result.ID)
	assert.Equal(t, "docs: update README", result.Name)
	assert.Equal(t, pr.StatusMerged, result.Status)
	assert.Equal(t, reviewerIDs, result.AssignedReviewers)
	assert.NotNil(t, result.MergedAt)
}

func TestPullRequestService_Merge_Success_AlreadyMerged_NoMarkCall(t *testing.T) {
	ctx := context.Background()
	prID := "merge-req-2"

	mockPr := mocks.NewPrController(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	now := time.Now()
	mergedPR := &entity.PullRequest{
		ID:       prID,
		Title:    "chore: release v1.1",
		AuthorId: "author-b",
		Status:   pr.StatusMerged,
		MergedAt: &now,
	}
	reviewerIDs := []string{"reviewer-r9"}

	mockPr.On("GetById", ctx, prID).Return(mergedPR, nil).Once()
	mockPr.On("GetById", ctx, prID).Return(mergedPR, nil).Once()
	mockReviewer.On("GetPrReviewers", ctx, prID).Return(reviewerIDs, nil).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			assert.NoError(t, fn(ctx))
		}).Return(nil).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, nil)
	result, e := service.Merge(ctx, prID)

	assert.NoError(t, e)
	assert.NotNil(t, result)
	assert.Equal(t, pr.StatusMerged, result.Status)
	assert.Equal(t, reviewerIDs, result.AssignedReviewers)
	mockPr.AssertNotCalled(t, "MarkAsMerged", ctx, prID)
}

func TestPullRequestService_Merge_Error_FirstGetById(t *testing.T) {
	ctx := context.Background()
	prID := "merge-fail-1"

	mockPr := mocks.NewPrController(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	getError := errors.New("db lookup failed")
	mockPr.On("GetById", ctx, prID).Return((*entity.PullRequest)(nil), getError).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.Equal(t, getError, e)
		}).Return(getError).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, nil)
	result, e := service.Merge(ctx, prID)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.Equal(t, getError, e)
}

func TestPullRequestService_Merge_Error_SecondGetById(t *testing.T) {
	ctx := context.Background()
	prID := "merge-fail-2"

	mockPr := mocks.NewPrController(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	openPR := &entity.PullRequest{ID: prID, Title: "Some Title", AuthorId: "author-a", Status: pr.StatusOpen}
	secondError := errors.New("db lookup failed again")

	mockPr.On("GetById", ctx, prID).Return(openPR, nil).Once()
	mockPr.On("MarkAsMerged", ctx, prID).Return(nil).Once()
	mockPr.On("GetById", ctx, prID).Return((*entity.PullRequest)(nil), secondError).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.Equal(t, secondError, e)
		}).Return(secondError).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, nil)
	result, e := service.Merge(ctx, prID)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.Equal(t, secondError, e)
}

func TestPullRequestService_Merge_Error_GetReviewers(t *testing.T) {
	ctx := context.Background()
	prID := "merge-fail-3"

	mockPr := mocks.NewPrController(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	mergedPR := &entity.PullRequest{ID: prID, Title: "Some Title", AuthorId: "author-a", Status: pr.StatusMerged}
	reviewerError := errors.New("failed to get reviewers")

	mockPr.On("GetById", ctx, prID).Return(mergedPR, nil).Once()
	mockPr.On("GetById", ctx, prID).Return(mergedPR, nil).Once()
	mockReviewer.On("GetPrReviewers", ctx, prID).Return(([]string)(nil), reviewerError).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.Equal(t, reviewerError, e)
		}).Return(reviewerError).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, nil)
	result, e := service.Merge(ctx, prID)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.Equal(t, reviewerError, e)
}

func TestPullRequestService_Merge_Ignores_MarkAsMergedError(t *testing.T) {
	ctx := context.Background()
	prID := "merge-ignore-1"

	mockPr := mocks.NewPrController(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	openPR := &entity.PullRequest{ID: prID, Title: "Some Title", AuthorId: "author-a", Status: pr.StatusOpen}
	mergedPR := &entity.PullRequest{ID: prID, Title: "Some Title", AuthorId: "author-a", Status: pr.StatusMerged}
	reviewerIDs := []string{"reviewer-r1"}
	mergeError := errors.New("could not mark as merged")

	mockPr.On("GetById", ctx, prID).Return(openPR, nil).Once()
	mockPr.On("MarkAsMerged", ctx, prID).Return(mergeError).Once()
	mockPr.On("GetById", ctx, prID).Return(mergedPR, nil).Once()
	mockReviewer.On("GetPrReviewers", ctx, prID).Return(reviewerIDs, nil).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			assert.NoError(t, fn(ctx))
		}).Return(nil).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, nil)
	result, e := service.Merge(ctx, prID)

	assert.NoError(t, e)
	assert.NotNil(t, result)
	assert.Equal(t, pr.StatusMerged, result.Status)
	assert.Equal(t, reviewerIDs, result.AssignedReviewers)
}

func TestPullRequestService_Reassign_Success(t *testing.T) {
	ctx := context.Background()
	prID := "reassign-1"
	oldRev := "reviewer-r1"

	mockPr := mocks.NewPrController(t)
	mockUser := mocks.NewUserGetter(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	currentPR := &entity.PullRequest{ID: prID, Title: "refactor: improve performance", AuthorId: "author-a", Status: pr.StatusOpen}
	authorUser := &entity.User{ID: "author-a", TeamID: 777}
	activeIDs := []string{"author-a", "reviewer-r1", "reviewer-r2", "reviewer-r3"}
	assignedIDs := []string{"reviewer-r1", "reviewer-r2"}
	finalIDs := []string{"reviewer-r2", "reviewer-r3"}

	mockPr.On("GetById", ctx, prID).Return(currentPR, nil).Twice()
	mockUser.On("GetById", ctx, "author-a").Return(authorUser, nil).Once()
	mockUser.On("GetActiveUsersIDInTeam", ctx, 777).Return(activeIDs, nil).Once()
	mockReviewer.On("GetPrReviewers", ctx, prID).Return(assignedIDs, nil).Once()
	mockReviewer.On("ReassignReviewer", ctx, prID, oldRev, mock.AnythingOfType("string")).Return(nil).Once()
	mockReviewer.On("GetPrReviewers", ctx, prID).Return(finalIDs, nil).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			assert.NoError(t, fn(ctx))
		}).Return(nil).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, mockUser)
	result, e := service.Reassign(ctx, prID, oldRev)

	assert.NoError(t, e)
	assert.NotNil(t, result)
	assert.Equal(t, prID, result.PullRequest.ID)
	assert.Equal(t, "refactor: improve performance", result.PullRequest.Name)
	assert.Equal(t, pr.StatusOpen, result.PullRequest.Status)
	assert.Len(t, result.PullRequest.AssignedReviewers, 2)
	assert.NotEmpty(t, result.ReplacedBy)
	assert.NotEqual(t, oldRev, result.ReplacedBy)
	assert.NotContains(t, result.PullRequest.AssignedReviewers, oldRev)
}

func TestPullRequestService_Reassign_NoCandidate(t *testing.T) {
	ctx := context.Background()
	prID := "reassign-2"
	oldRev := "busy-reviewer"

	mockPr := mocks.NewPrController(t)
	mockUser := mocks.NewUserGetter(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	currentPR := &entity.PullRequest{ID: prID, Title: "fix: alignment issue", AuthorId: "author-a", Status: pr.StatusOpen}
	authorUser := &entity.User{ID: "author-a", TeamID: 55}
	activeIDs := []string{"author-a", "busy-reviewer"}
	assignedIDs := []string{"busy-reviewer"}

	mockPr.On("GetById", ctx, prID).Return(currentPR, nil).Once()
	mockUser.On("GetById", ctx, "author-a").Return(authorUser, nil).Once()
	mockUser.On("GetActiveUsersIDInTeam", ctx, 55).Return(activeIDs, nil).Once()
	mockReviewer.On("GetPrReviewers", ctx, prID).Return(assignedIDs, nil).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.Equal(t, repo.ErrNoCandidate, e)
		}).Return(repo.ErrNoCandidate).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, mockUser)
	result, e := service.Reassign(ctx, prID, oldRev)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.Equal(t, repo.ErrNoCandidate, e)
	mockReviewer.AssertNotCalled(t, "ReassignReviewer", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestPullRequestService_Reassign_ReassignReviewerError(t *testing.T) {
	ctx := context.Background()
	prID := "reassign-3"
	oldRev := "reviewer-r1"

	mockPr := mocks.NewPrController(t)
	mockUser := mocks.NewUserGetter(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	currentPR := &entity.PullRequest{ID: prID, Title: "hotfix: critical security patch", AuthorId: "author-a", Status: pr.StatusOpen}
	authorUser := &entity.User{ID: "author-a", TeamID: 33}
	activeIDs := []string{"author-a", "reviewer-r1", "reviewer-r2"}
	assignedIDs := []string{"reviewer-r1"}
	reassignError := errors.New("could not reassign")

	mockPr.On("GetById", ctx, prID).Return(currentPR, nil).Once()
	mockUser.On("GetById", ctx, "author-a").Return(authorUser, nil).Once()
	mockUser.On("GetActiveUsersIDInTeam", ctx, 33).Return(activeIDs, nil).Once()
	mockReviewer.On("GetPrReviewers", ctx, prID).Return(assignedIDs, nil).Once()
	mockReviewer.On("ReassignReviewer", ctx, prID, oldRev, mock.AnythingOfType("string")).Return(reassignError).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.Equal(t, reassignError, e)
		}).Return(reassignError).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, mockUser)
	result, e := service.Reassign(ctx, prID, oldRev)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.Equal(t, reassignError, e)
}

func TestPullRequestService_Reassign_GetByIdError(t *testing.T) {
	ctx := context.Background()
	prID := "reassign-4"
	oldRev := "reviewer-r1"

	mockPr := mocks.NewPrController(t)
	mockUser := mocks.NewUserGetter(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	getError := errors.New("pr not found in db")
	mockPr.On("GetById", ctx, prID).Return((*entity.PullRequest)(nil), getError).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			e := fn(ctx)
			assert.Error(t, e)
			assert.Equal(t, getError, e)
		}).Return(getError).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, mockUser)
	result, e := service.Reassign(ctx, prID, oldRev)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.Equal(t, getError, e)
}

func TestPullRequestService_Reassign_GetActiveUsersError(t *testing.T) {
	ctx := context.Background()
	prID := "reassign-5"
	oldRev := "reviewer-r1"

	mockPr := mocks.NewPrController(t)
	mockUser := mocks.NewUserGetter(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	currentPR := &entity.PullRequest{ID: prID, Title: "bug: active user lookup", AuthorId: "author-a", Status: pr.StatusOpen}
	authorUser := &entity.User{ID: "author-a", TeamID: 113}
	activeUsersError := errors.New("user service is down")

	mockPr.On("GetById", ctx, prID).Return(currentPR, nil).Once()
	mockUser.On("GetById", ctx, "author-a").Return(authorUser, nil).Once()
	mockUser.On("GetActiveUsersIDInTeam", ctx, 113).Return(([]string)(nil), activeUsersError).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context) error)
		e := fn(ctx)
		assert.Error(t, e)
		assert.Equal(t, activeUsersError, e)
	}).Return(activeUsersError).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, mockUser)
	result, e := service.Reassign(ctx, prID, oldRev)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.Equal(t, activeUsersError, e)
}

func TestPullRequestService_Reassign_GetReviewersError(t *testing.T) {
	ctx := context.Background()
	prID := "reassign-6"
	oldRev := "reviewer-r1"

	mockPr := mocks.NewPrController(t)
	mockUser := mocks.NewUserGetter(t)
	mockReviewer := mocks.NewReviewerProvider(t)
	mockTxManager := &mocks.MockManager{}
	mockTxManager.Test(t)
	t.Cleanup(func() { mockTxManager.AssertExpectations(t) })

	currentPR := &entity.PullRequest{ID: prID, Title: "bug: reviewer lookup", AuthorId: "author-a", Status: pr.StatusOpen}
	authorUser := &entity.User{ID: "author-a", TeamID: 111}
	reviewerError := errors.New("reviewer service is down")

	mockPr.On("GetById", ctx, prID).Return(currentPR, nil).Once()
	mockUser.On("GetById", ctx, "author-a").Return(authorUser, nil).Once()
	mockUser.On("GetActiveUsersIDInTeam", ctx, 111).Return([]string{"author-a", "reviewer-r1", "reviewer-r2"}, nil).Once()
	mockReviewer.On("GetPrReviewers", ctx, prID).Return(([]string)(nil), reviewerError).Once()

	mockTxManager.On("Do", ctx, mock.AnythingOfType("func(context.Context) error")).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context) error)
		e := fn(ctx)
		assert.Error(t, e)
		assert.Equal(t, reviewerError, e)
	}).Return(reviewerError).Once()

	service := pr.NewPullRequestService(mockTxManager, mockPr, mockReviewer, mockUser)
	result, e := service.Reassign(ctx, prID, oldRev)

	assert.Nil(t, result)
	assert.Error(t, e)
	assert.Equal(t, reviewerError, e)
}
