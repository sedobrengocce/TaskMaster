package db

import (
	"context"
	"database/sql"

	"github.com/stretchr/testify/mock"
)

type MockQuerier struct {
	mock.Mock
}

var _ Querier = (*MockQuerier)(nil)

func (m *MockQuerier) CreateProject(ctx context.Context, arg CreateProjectParams) (sql.Result, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *MockQuerier) CreateTask(ctx context.Context, arg CreateTaskParams) (sql.Result, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *MockQuerier) CreateUser(ctx context.Context, arg CreateUserParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) DeleteProject(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) DeleteTask(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) DeleteUser(ctx context.Context, id int32) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQuerier) GetClientByClientID(ctx context.Context, clientID string) (GetClientByClientIDRow, error) {
	args := m.Called(ctx, clientID)
	return args.Get(0).(GetClientByClientIDRow), args.Error(1)
}

func (m *MockQuerier) GetProjectById(ctx context.Context, id int32) (Project, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Project), args.Error(1)
}

func (m *MockQuerier) GetProjectsByUserId(ctx context.Context, arg GetProjectsByUserIdParams) ([]Project, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]Project), args.Error(1)
}

func (m *MockQuerier) GetRefreshToken(ctx context.Context, userID int32) (GetRefreshTokenRow, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(GetRefreshTokenRow), args.Error(1)
}

func (m *MockQuerier) GetTaskListByProjectId(ctx context.Context, projectID sql.NullInt32) ([]Task, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).([]Task), args.Error(1)
}

func (m *MockQuerier) GetTasksByUserId(ctx context.Context, arg GetTasksByUserIdParams) ([]Task, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]Task), args.Error(1)
}

func (m *MockQuerier) GetUnscheduledTasksByUserId(ctx context.Context, arg GetUnscheduledTasksByUserIdParams) ([]Task, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]Task), args.Error(1)
}

func (m *MockQuerier) GetUserByEmail(ctx context.Context, email string) (User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(User), args.Error(1)
}

func (m *MockQuerier) GetUserByEmailAndPassword(ctx context.Context, arg GetUserByEmailAndPasswordParams) (User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(User), args.Error(1)
}

func (m *MockQuerier) GetUserByID(ctx context.Context, id int32) (User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(User), args.Error(1)
}

func (m *MockQuerier) GetUserByTerm(ctx context.Context, concat interface{}) ([]User, error) {
	args := m.Called(ctx, concat)
	return args.Get(0).([]User), args.Error(1)
}

func (m *MockQuerier) InsertClient(ctx context.Context, arg InsertClientParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) InsertRefreshToken(ctx context.Context, arg InsertRefreshTokenParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) RevokeRefreshToken(ctx context.Context, userID int32) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockQuerier) ShareProjectWithUser(ctx context.Context, arg ShareProjectWithUserParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) ShareTaskWithUser(ctx context.Context, arg ShareTaskWithUserParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) UnshareProjectWithUser(ctx context.Context, arg UnshareProjectWithUserParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) UnshareTaskWithUser(ctx context.Context, arg UnshareTaskWithUserParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) UpdateProject(ctx context.Context, arg UpdateProjectParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) UpdateTask(ctx context.Context, arg UpdateTaskParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) UpdateUser(ctx context.Context, arg UpdateUserParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) CompleteTask(ctx context.Context, arg CompleteTaskParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) UncompleteTask(ctx context.Context, arg UncompleteTaskParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) GetTaskCompletions(ctx context.Context, taskID int32) ([]TaskLog, error) {
	args := m.Called(ctx, taskID)
	return args.Get(0).([]TaskLog), args.Error(1)
}

func (m *MockQuerier) GetCompletionsForWeek(ctx context.Context, arg GetCompletionsForWeekParams) ([]TaskLog, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]TaskLog), args.Error(1)
}

func (m *MockQuerier) IsProjectSharedWithUser(ctx context.Context, arg IsProjectSharedWithUserParams) (bool, error) {
	args := m.Called(ctx, arg)
	return args.Bool(0), args.Error(1)
}

func (m *MockQuerier) IsTaskSharedWithUser(ctx context.Context, arg IsTaskSharedWithUserParams) (bool, error) {
	args := m.Called(ctx, arg)
	return args.Bool(0), args.Error(1)
}

func (m *MockQuerier) GetTaskById(ctx context.Context, id int32) (Task, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Task), args.Error(1)
}

func (m *MockQuerier) UpdateTaskProject(ctx context.Context, arg UpdateTaskProjectParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) ScheduleTask(ctx context.Context, arg ScheduleTaskParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) UnscheduleTask(ctx context.Context, arg UnscheduleTaskParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockQuerier) GetTaskDates(ctx context.Context, taskID int32) ([]TaskDate, error) {
	args := m.Called(ctx, taskID)
	return args.Get(0).([]TaskDate), args.Error(1)
}

func (m *MockQuerier) GetScheduledTasksForDateRange(ctx context.Context, arg GetScheduledTasksForDateRangeParams) ([]TaskDate, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]TaskDate), args.Error(1)
}
