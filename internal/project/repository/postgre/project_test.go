package postgre

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"project-srv/internal/model"
	"project-srv/internal/project/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/paginator"
	"github.com/stretchr/testify/require"
)

func newTestRepo(t *testing.T) (*implRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	l := log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
	return New(db, l).(*implRepository), mockDB, func() { _ = db.Close() }
}

func projectColumns() []string {
	return []string{"id", "campaign_id", "name", "description", "brand", "entity_type", "entity_name", "domain_type_code", "status", "config_status", "favorite_user_ids", "created_by", "created_at", "updated_at"}
}

func projectRowValues() []driver.Value {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return []driver.Value{"project-1", "campaign-1", "Project A", "Desc", "Brand", "product", "VF8", "ev", "PENDING", "DRAFT", pq.StringArray{"user-1"}, "user-1", now, now}
}

func sqlboilerProjectColumns() []string {
	return []string{"id", "campaign_id", "name", "description", "brand", "entity_type", "entity_name", "domain_type_code", "status", "config_status", "created_by", "favorite_user_ids", "created_at", "updated_at", "deleted_at"}
}

func sqlboilerProjectRows(id string) *sqlmock.Rows {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return sqlmock.NewRows(sqlboilerProjectColumns()).AddRow(id, "campaign-1", "Project A", "Desc", "Brand", "product", "VF8", "ev", "PENDING", "DRAFT", "user-1", pq.StringArray{"user-1"}, now, now, nil)
}

func TestNew(t *testing.T) {
	tcs := map[string]struct {
		input  struct{}
		mock   struct{}
		output bool
		err    error
	}{
		"success": {output: true},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, _, done := newTestRepo(t)
			defer done()

			require.Equal(t, tc.output, r != nil)
		})
	}
}

func TestToProjectModel(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		input  projectRow
		mock   struct{}
		output model.Project
		err    error
	}{
		"success": {
			input: projectRow{
				ID:              "project-1",
				CampaignID:      "campaign-1",
				Name:            "Project A",
				Description:     sql.NullString{String: "Desc", Valid: true},
				Brand:           sql.NullString{String: "Brand", Valid: true},
				EntityType:      "product",
				EntityName:      "VF8",
				DomainTypeCode:  "ev",
				Status:          "PENDING",
				ConfigStatus:    sql.NullString{String: "DRAFT", Valid: true},
				FavoriteUserIDs: pq.StringArray{"user-1"},
				CreatedBy:       "user-1",
				CreatedAt:       sql.NullTime{Time: now, Valid: true},
				UpdatedAt:       sql.NullTime{Time: now, Valid: true},
			},
			output: model.Project{ID: "project-1", CampaignID: "campaign-1", Name: "Project A", Description: "Desc", Brand: "Brand", EntityType: model.EntityTypeProduct, EntityName: "VF8", DomainTypeCode: "ev", Status: model.ProjectStatusPending, ConfigStatus: model.ConfigStatusDraft, FavoriteUserIDs: []string{"user-1"}, CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now},
		},
		"nullable_fields": {
			input:  projectRow{ID: "project-1", Name: "Project A", EntityType: "topic", Status: "ACTIVE"},
			output: model.Project{ID: "project-1", Name: "Project A", EntityType: model.EntityTypeTopic, Status: model.ProjectStatusActive},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := toProjectModel(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestBuildProjectFilters(t *testing.T) {
	tcs := map[string]struct {
		input  repository.GetOptions
		mock   struct{}
		output string
		err    error
	}{
		"default": {
			output: "deleted_at IS NULL",
		},
		"all_filters": {
			input:  repository.GetOptions{CampaignID: "campaign-1", Status: "PENDING", Name: "Project", Brand: "Brand", EntityType: "product", FavoriteOnly: true, CurrentUserID: "user-1"},
			output: "deleted_at IS NULL AND campaign_id = $1 AND status = $2 AND name ILIKE $3 AND brand ILIKE $4 AND entity_type = $5 AND favorite_user_ids @> $6::uuid[]",
		},
		"favorite_without_user": {
			input:  repository.GetOptions{FavoriteOnly: true},
			output: "deleted_at IS NULL AND 1 = 0",
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, _, done := newTestRepo(t)
			defer done()

			output, _ := r.buildProjectFilters(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestBuildProjectOrderBy(t *testing.T) {
	tcs := map[string]struct {
		input  repository.GetOptions
		mock   struct{}
		output string
		err    error
	}{
		"default": {
			output: "created_at DESC",
		},
		"favorite_desc": {
			input:  repository.GetOptions{Sort: "favorite_desc", CurrentUserID: "user-1"},
			output: "CASE WHEN favorite_user_ids @> $1::uuid[] THEN 0 ELSE 1 END, created_at DESC",
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, _, done := newTestRepo(t)
			defer done()
			args := []any{}

			output := r.buildProjectOrderBy(tc.input, &args)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestDetail(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input  string
		mock   struct{ err error }
		output model.Project
		err    error
	}{
		"success": {
			input:  "project-1",
			output: model.Project{ID: "project-1"},
		},
		"not_found": {
			input: "project-1",
			mock:  struct{ err error }{err: sql.ErrNoRows},
			err:   repository.ErrNotFound,
		},
		"query_error": {
			input: "project-1",
			mock:  struct{ err error }{err: errors.New("db error")},
			err:   repository.ErrFailedToGet,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			query := `SELECT id, campaign_id, name, description, brand, entity_type, entity_name, domain_type_code, status, config_status, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.projects\s+WHERE id = \$1 AND deleted_at IS NULL`
			if tc.mock.err != nil {
				mockDB.ExpectQuery(query).WithArgs(tc.input).WillReturnError(tc.mock.err)
			} else {
				mockDB.ExpectQuery(query).WithArgs(tc.input).WillReturnRows(sqlmock.NewRows(projectColumns()).AddRow(projectRowValues()...))
			}

			output, err := r.Detail(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				require.NoError(t, mockDB.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output.ID, output.ID)
			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input repository.CreateOptions
		mock  struct {
			insertErr error
			fetchErr  error
		}
		output model.Project
		err    error
	}{
		"success": {
			input:  repository.CreateOptions{CampaignID: "campaign-1", Name: "Project A", EntityType: "product", EntityName: "VF8", DomainTypeCode: "ev", CreatedBy: "user-1"},
			output: model.Project{ID: "project-1"},
		},
		"insert_error": {
			input: repository.CreateOptions{CampaignID: "campaign-1", Name: "Project A"},
			mock: struct {
				insertErr error
				fetchErr  error
			}{insertErr: errors.New("insert error")},
			err: repository.ErrFailedToInsert,
		},
		"fetch_error": {
			input: repository.CreateOptions{CampaignID: "campaign-1", Name: "Project A"},
			mock: struct {
				insertErr error
				fetchErr  error
			}{fetchErr: errors.New("fetch error")},
			err: repository.ErrFailedToGet,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			insertQuery := `INSERT INTO project\.projects`
			if tc.mock.insertErr != nil {
				mockDB.ExpectQuery(insertQuery).WillReturnError(tc.mock.insertErr)
			} else {
				mockDB.ExpectQuery(insertQuery).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("project-1"))
				fetchQuery := `SELECT id, campaign_id, name, description, brand, entity_type, entity_name, domain_type_code, status, config_status, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.projects`
				if tc.mock.fetchErr != nil {
					mockDB.ExpectQuery(fetchQuery).WillReturnError(tc.mock.fetchErr)
				} else {
					mockDB.ExpectQuery(fetchQuery).WillReturnRows(sqlmock.NewRows(projectColumns()).AddRow(projectRowValues()...))
				}
			}

			output, err := r.Create(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				require.NoError(t, mockDB.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output.ID, output.ID)
			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestGet(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input repository.GetOptions
		mock  struct {
			countErr error
			listErr  error
			scanErr  bool
			rowsErr  bool
		}
		output []model.Project
		err    error
	}{
		"success": {
			input:  repository.GetOptions{Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
			output: []model.Project{{ID: "project-1"}},
		},
		"count_error": {
			input: repository.GetOptions{Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
			mock: struct {
				countErr error
				listErr  error
				scanErr  bool
				rowsErr  bool
			}{countErr: errors.New("count error")},
			err: repository.ErrFailedToList,
		},
		"list_error": {
			input: repository.GetOptions{Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
			mock: struct {
				countErr error
				listErr  error
				scanErr  bool
				rowsErr  bool
			}{listErr: errors.New("list error")},
			err: repository.ErrFailedToList,
		},
		"scan_error": {
			input: repository.GetOptions{Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
			mock: struct {
				countErr error
				listErr  error
				scanErr  bool
				rowsErr  bool
			}{scanErr: true},
			err: repository.ErrFailedToList,
		},
		"rows_error": {
			input: repository.GetOptions{Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
			mock: struct {
				countErr error
				listErr  error
				scanErr  bool
				rowsErr  bool
			}{rowsErr: true},
			err: repository.ErrFailedToList,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			if tc.mock.countErr != nil {
				mockDB.ExpectQuery(`SELECT COUNT\(\*\) FROM project\.projects WHERE deleted_at IS NULL`).WillReturnError(tc.mock.countErr)
			} else {
				mockDB.ExpectQuery(`SELECT COUNT\(\*\) FROM project\.projects WHERE deleted_at IS NULL`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			}
			if tc.mock.listErr != nil {
				mockDB.ExpectQuery(`SELECT id, campaign_id, name, description, brand, entity_type, entity_name, domain_type_code, status, config_status, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.projects`).WillReturnError(tc.mock.listErr)
			} else if tc.mock.scanErr {
				values := projectRowValues()
				values[0] = nil
				mockDB.ExpectQuery(`SELECT id, campaign_id, name, description, brand, entity_type, entity_name, domain_type_code, status, config_status, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.projects`).WillReturnRows(sqlmock.NewRows(projectColumns()).AddRow(values...))
			} else if tc.mock.rowsErr {
				mockDB.ExpectQuery(`SELECT id, campaign_id, name, description, brand, entity_type, entity_name, domain_type_code, status, config_status, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.projects`).WillReturnRows(sqlmock.NewRows(projectColumns()).AddRow(projectRowValues()...).RowError(0, errors.New("rows error")))
			} else if tc.mock.countErr == nil {
				mockDB.ExpectQuery(`SELECT id, campaign_id, name, description, brand, entity_type, entity_name, domain_type_code, status, config_status, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.projects`).WillReturnRows(sqlmock.NewRows(projectColumns()).AddRow(projectRowValues()...))
			}

			output, _, err := r.Get(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				require.NoError(t, mockDB.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Len(t, output, len(tc.output))
			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input repository.UpdateOptions
		mock  struct {
			execErr      error
			rowsAffected int64
			rowsErr      error
			fetchErr     error
		}
		output model.Project
		err    error
	}{
		"success": {
			input: repository.UpdateOptions{ID: "project-1", Name: "Project B", Description: "Desc B", Brand: "Brand B", EntityType: "product", EntityName: "VF9", DomainTypeCode: "ev"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
				fetchErr     error
			}{rowsAffected: 1},
			output: model.Project{ID: "project-1"},
		},
		"fetch_error": {
			input: repository.UpdateOptions{ID: "project-1", Name: "Project B"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
				fetchErr     error
			}{rowsAffected: 1, fetchErr: errors.New("fetch error")},
			err: repository.ErrFailedToGet,
		},
		"exec_error": {
			input: repository.UpdateOptions{ID: "project-1", Name: "Project B"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
				fetchErr     error
			}{execErr: errors.New("exec error")},
			err: repository.ErrFailedToUpdate,
		},
		"rows_error": {
			input: repository.UpdateOptions{ID: "project-1", Name: "Project B"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
				fetchErr     error
			}{rowsErr: errors.New("rows error")},
			err: repository.ErrFailedToUpdate,
		},
		"not_found": {
			input: repository.UpdateOptions{ID: "project-1", Name: "Project B"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
				fetchErr     error
			}{rowsAffected: 0},
			err: repository.ErrNotFound,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			expect := mockDB.ExpectExec(`UPDATE project\.projects\s+SET`)
			if tc.mock.execErr != nil {
				expect.WillReturnError(tc.mock.execErr)
			} else if tc.mock.rowsErr != nil {
				expect.WillReturnResult(sqlmock.NewErrorResult(tc.mock.rowsErr))
			} else {
				expect.WillReturnResult(sqlmock.NewResult(0, tc.mock.rowsAffected))
			}
			if tc.mock.rowsAffected > 0 && tc.mock.execErr == nil && tc.mock.rowsErr == nil {
				if tc.mock.fetchErr != nil {
					mockDB.ExpectQuery(`SELECT id, campaign_id, name, description, brand, entity_type, entity_name, domain_type_code, status, config_status, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.projects`).WillReturnError(tc.mock.fetchErr)
				} else {
					mockDB.ExpectQuery(`SELECT id, campaign_id, name, description, brand, entity_type, entity_name, domain_type_code, status, config_status, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.projects`).WillReturnRows(sqlmock.NewRows(projectColumns()).AddRow(projectRowValues()...))
				}
			}

			output, err := r.Update(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				require.NoError(t, mockDB.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output.ID, output.ID)
			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input repository.UpdateStatusOptions
		mock  struct {
			findErr      error
			updateErr    error
			rowsAffected int64
			fetchErr     error
		}
		output model.Project
		err    error
	}{
		"success_without_expected": {
			input: repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive)},
			mock: struct {
				findErr      error
				updateErr    error
				rowsAffected int64
				fetchErr     error
			}{rowsAffected: 1},
			output: model.Project{ID: "project-1"},
		},
		"success_with_expected": {
			input: repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive), ExpectedStatuses: []string{"PENDING", " "}},
			mock: struct {
				findErr      error
				updateErr    error
				rowsAffected int64
				fetchErr     error
			}{rowsAffected: 1},
			output: model.Project{ID: "project-1"},
		},
		"not_found": {
			input: repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive)},
			mock: struct {
				findErr      error
				updateErr    error
				rowsAffected int64
				fetchErr     error
			}{findErr: sql.ErrNoRows},
			err: repository.ErrNotFound,
		},
		"find_error": {
			input: repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive)},
			mock: struct {
				findErr      error
				updateErr    error
				rowsAffected int64
				fetchErr     error
			}{findErr: errors.New("find error")},
			err: repository.ErrFailedToUpdate,
		},
		"update_error": {
			input: repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive)},
			mock: struct {
				findErr      error
				updateErr    error
				rowsAffected int64
				fetchErr     error
			}{updateErr: errors.New("update error")},
			err: repository.ErrFailedToUpdate,
		},
		"expected_update_error": {
			input: repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive), ExpectedStatuses: []string{"PENDING"}},
			mock: struct {
				findErr      error
				updateErr    error
				rowsAffected int64
				fetchErr     error
			}{updateErr: errors.New("update error")},
			err: repository.ErrFailedToUpdate,
		},
		"expected_conflict": {
			input: repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive), ExpectedStatuses: []string{"PENDING"}},
			mock: struct {
				findErr      error
				updateErr    error
				rowsAffected int64
				fetchErr     error
			}{rowsAffected: 0},
			err: repository.ErrStatusConflict,
		},
		"expected_conflict_not_found": {
			input: repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive), ExpectedStatuses: []string{"PENDING"}},
			mock: struct {
				findErr      error
				updateErr    error
				rowsAffected int64
				fetchErr     error
			}{rowsAffected: 0, findErr: sql.ErrNoRows},
			err: repository.ErrNotFound,
		},
		"expected_conflict_find_error": {
			input: repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive), ExpectedStatuses: []string{"PENDING"}},
			mock: struct {
				findErr      error
				updateErr    error
				rowsAffected int64
				fetchErr     error
			}{rowsAffected: 0, findErr: errors.New("find error")},
			err: repository.ErrFailedToUpdate,
		},
		"fetch_error": {
			input: repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive)},
			mock: struct {
				findErr      error
				updateErr    error
				rowsAffected int64
				fetchErr     error
			}{rowsAffected: 1, fetchErr: errors.New("fetch error")},
			err: repository.ErrFailedToGet,
		},
		"expected_fetch_error": {
			input: repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive), ExpectedStatuses: []string{"PENDING"}},
			mock: struct {
				findErr      error
				updateErr    error
				rowsAffected int64
				fetchErr     error
			}{rowsAffected: 1, fetchErr: errors.New("fetch error")},
			err: repository.ErrFailedToGet,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			findQuery := `select \* from "project"\."projects" where "id"=\$1 and "deleted_at" is null`
			fetchQuery := `SELECT id, campaign_id, name, description, brand, entity_type, entity_name, domain_type_code, status, config_status, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.projects`
			if len(tc.input.ExpectedStatuses) > 0 {
				expect := mockDB.ExpectExec(`UPDATE "project"\."projects" SET`)
				if tc.mock.updateErr != nil {
					expect.WillReturnError(tc.mock.updateErr)
				} else {
					expect.WillReturnResult(sqlmock.NewResult(0, tc.mock.rowsAffected))
				}
				if tc.mock.updateErr == nil && tc.mock.rowsAffected == 0 {
					if tc.mock.findErr != nil {
						mockDB.ExpectQuery(findQuery).WithArgs(tc.input.ID).WillReturnError(tc.mock.findErr)
					} else {
						mockDB.ExpectQuery(findQuery).WithArgs(tc.input.ID).WillReturnRows(sqlboilerProjectRows(tc.input.ID))
					}
				}
				if tc.mock.updateErr == nil && tc.mock.rowsAffected > 0 {
					if tc.mock.fetchErr != nil {
						mockDB.ExpectQuery(fetchQuery).WillReturnError(tc.mock.fetchErr)
					} else {
						mockDB.ExpectQuery(fetchQuery).WillReturnRows(sqlmock.NewRows(projectColumns()).AddRow(projectRowValues()...))
					}
				}
			} else if tc.mock.findErr != nil {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input.ID).WillReturnError(tc.mock.findErr)
			} else {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input.ID).WillReturnRows(sqlboilerProjectRows(tc.input.ID))
				expect := mockDB.ExpectExec(`UPDATE "project"\."projects" SET`)
				if tc.mock.updateErr != nil {
					expect.WillReturnError(tc.mock.updateErr)
				} else {
					expect.WillReturnResult(sqlmock.NewResult(0, tc.mock.rowsAffected))
				}
				if tc.mock.updateErr == nil {
					if tc.mock.fetchErr != nil {
						mockDB.ExpectQuery(fetchQuery).WillReturnError(tc.mock.fetchErr)
					} else {
						mockDB.ExpectQuery(fetchQuery).WillReturnRows(sqlmock.NewRows(projectColumns()).AddRow(projectRowValues()...))
					}
				}
			}

			output, err := r.UpdateStatus(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				require.NoError(t, mockDB.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output.ID, output.ID)
			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestFavorite(t *testing.T) {
	testFavoriteCommand(t, "favorite", func(r *implRepository, ctx context.Context, id, userID string) error {
		return r.Favorite(ctx, id, userID)
	}, `UPDATE project\.projects\s+SET favorite_user_ids`, repository.ErrNotFound)
}

func TestArchive(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input string
		mock  struct {
			findErr   error
			updateErr error
		}
		output struct{}
		err    error
	}{
		"success": {
			input: "project-1",
		},
		"not_found": {
			input: "project-1",
			mock: struct {
				findErr   error
				updateErr error
			}{findErr: sql.ErrNoRows},
			err: repository.ErrNotFound,
		},
		"find_error": {
			input: "project-1",
			mock: struct {
				findErr   error
				updateErr error
			}{findErr: errors.New("find error")},
			err: repository.ErrFailedToDelete,
		},
		"update_error": {
			input: "project-1",
			mock: struct {
				findErr   error
				updateErr error
			}{updateErr: errors.New("update error")},
			err: repository.ErrFailedToDelete,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			findQuery := `select \* from "project"\."projects" where "id"=\$1 and "deleted_at" is null`
			if tc.mock.findErr != nil {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input).WillReturnError(tc.mock.findErr)
			} else {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input).WillReturnRows(sqlboilerProjectRows(tc.input))
				expect := mockDB.ExpectExec(`UPDATE "project"\."projects" SET`)
				if tc.mock.updateErr != nil {
					expect.WillReturnError(tc.mock.updateErr)
				} else {
					expect.WillReturnResult(sqlmock.NewResult(0, 1))
				}
			}

			err := r.Archive(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				require.NoError(t, mockDB.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestUnfavorite(t *testing.T) {
	testFavoriteCommand(t, "unfavorite", func(r *implRepository, ctx context.Context, id, userID string) error {
		return r.Unfavorite(ctx, id, userID)
	}, `UPDATE project\.projects\s+SET favorite_user_ids`, repository.ErrNotFound)
}

func testFavoriteCommand(t *testing.T, name string, call func(*implRepository, context.Context, string, string) error, query string, notFoundErr error) {
	t.Helper()
	ctx := context.Background()

	tcs := map[string]struct {
		input struct{ id, userID string }
		mock  struct {
			execErr      error
			rowsAffected int64
			rowsErr      error
		}
		output struct{}
		err    error
	}{
		"success": {
			input: struct{ id, userID string }{id: "project-1", userID: "user-1"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
			}{rowsAffected: 1},
		},
		"exec_error": {
			input: struct{ id, userID string }{id: "project-1", userID: "user-1"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
			}{execErr: errors.New("exec error")},
			err: repository.ErrFailedToUpdate,
		},
		"rows_error": {
			input: struct{ id, userID string }{id: "project-1", userID: "user-1"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
			}{rowsErr: errors.New("rows error")},
			err: repository.ErrFailedToUpdate,
		},
		"not_found": {
			input: struct{ id, userID string }{id: "project-1", userID: "user-1"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
			}{rowsAffected: 0},
			err: notFoundErr,
		},
	}

	for caseName, tc := range tcs {
		t.Run(name+"_"+caseName, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			expect := mockDB.ExpectExec(query)
			if tc.mock.execErr != nil {
				expect.WillReturnError(tc.mock.execErr)
			} else if tc.mock.rowsErr != nil {
				expect.WillReturnResult(sqlmock.NewErrorResult(tc.mock.rowsErr))
			} else {
				expect.WillReturnResult(sqlmock.NewResult(0, tc.mock.rowsAffected))
			}

			err := call(r, ctx, tc.input.id, tc.input.userID)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				require.NoError(t, mockDB.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}
