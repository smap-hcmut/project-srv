package postgre

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"

	"project-srv/internal/campaign/repository"
	"project-srv/internal/model"

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

func campaignColumns() []string {
	return []string{"id", "name", "description", "status", "start_date", "end_date", "favorite_user_ids", "created_by", "created_at", "updated_at"}
}

func campaignRowValues() []driver.Value {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return []driver.Value{"campaign-1", "Campaign A", "Desc", "PENDING", now, now, pq.StringArray{"user-1"}, "user-1", now, now}
}

func sqlboilerCampaignColumns() []string {
	return []string{"id", "name", "description", "status", "start_date", "end_date", "created_by", "favorite_user_ids", "created_at", "updated_at", "deleted_at"}
}

func sqlboilerCampaignRows(id string) *sqlmock.Rows {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return sqlmock.NewRows(sqlboilerCampaignColumns()).AddRow(id, "Campaign A", "Desc", "PENDING", now, now, "user-1", pq.StringArray{"user-1"}, now, now, nil)
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

func TestToCampaignModel(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		input  campaignRow
		mock   struct{}
		output model.Campaign
		err    error
	}{
		"success": {
			input: campaignRow{
				ID:              "campaign-1",
				Name:            "Campaign A",
				Description:     sql.NullString{String: "Desc", Valid: true},
				Status:          "PENDING",
				StartDate:       sql.NullTime{Time: now, Valid: true},
				EndDate:         sql.NullTime{Time: now, Valid: true},
				FavoriteUserIDs: pq.StringArray{"user-1"},
				CreatedBy:       "user-1",
				CreatedAt:       sql.NullTime{Time: now, Valid: true},
				UpdatedAt:       sql.NullTime{Time: now, Valid: true},
			},
			output: model.Campaign{ID: "campaign-1", Name: "Campaign A", Description: "Desc", Status: model.CampaignStatusPending, StartDate: &now, EndDate: &now, FavoriteUserIDs: []string{"user-1"}, CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now},
		},
		"nullable_fields": {
			input:  campaignRow{ID: "campaign-1", Name: "Campaign A", Status: "ACTIVE"},
			output: model.Campaign{ID: "campaign-1", Name: "Campaign A", Status: model.CampaignStatusActive},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := toCampaignModel(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestBuildCampaignFilters(t *testing.T) {
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
			input:  repository.GetOptions{Status: "PENDING", Name: "Cam", CreatedBy: "user-1", FavoriteOnly: true, CurrentUserID: "user-1"},
			output: "deleted_at IS NULL AND status = $1 AND name ILIKE $2 AND created_by = $3 AND favorite_user_ids @> $4::uuid[]",
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

			output, _ := r.buildCampaignFilters(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestBuildCampaignOrderBy(t *testing.T) {
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

			output := r.buildCampaignOrderBy(tc.input, &args)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestDetail(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input  string
		mock   struct{ err error }
		output model.Campaign
		err    error
	}{
		"success": {
			input:  "campaign-1",
			output: model.Campaign{ID: "campaign-1", Name: "Campaign A", Description: "Desc", Status: model.CampaignStatusPending, FavoriteUserIDs: []string{"user-1"}, CreatedBy: "user-1", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		"not_found": {
			input: "campaign-1",
			mock:  struct{ err error }{err: sql.ErrNoRows},
			err:   repository.ErrFailedToGet,
		},
		"query_error": {
			input: "campaign-1",
			mock:  struct{ err error }{err: errors.New("db error")},
			err:   repository.ErrFailedToGet,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			query := `SELECT id, name, description, status, start_date, end_date, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.campaigns\s+WHERE id = \$1 AND deleted_at IS NULL`
			if tc.mock.err != nil {
				mockDB.ExpectQuery(query).WithArgs(tc.input).WillReturnError(tc.mock.err)
			} else {
				mockDB.ExpectQuery(query).WithArgs(tc.input).WillReturnRows(sqlmock.NewRows(campaignColumns()).AddRow(campaignRowValues()...))
			}

			output, err := r.Detail(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				require.NoError(t, mockDB.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output.ID, output.ID)
			require.Equal(t, tc.output.Name, output.Name)
			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		input repository.CreateOptions
		mock  struct {
			insertErr error
		}
		output model.Campaign
		err    error
	}{
		"success": {
			input:  repository.CreateOptions{Name: "Campaign A", Description: "Desc", StartDate: &start, EndDate: &end, CreatedBy: "user-1"},
			output: model.Campaign{Name: "Campaign A", Description: "Desc", Status: model.CampaignStatusPending, CreatedBy: "user-1"},
		},
		"insert_error": {
			input: repository.CreateOptions{Name: "Campaign A", Description: "Desc", StartDate: &start, EndDate: &end, CreatedBy: "user-1"},
			mock:  struct{ insertErr error }{insertErr: errors.New("insert error")},
			err:   repository.ErrFailedToInsert,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			expect := mockDB.ExpectQuery(`INSERT INTO "project"\."campaigns"`)
			if tc.mock.insertErr != nil {
				expect.WillReturnError(tc.mock.insertErr)
			} else {
				expect.WillReturnRows(sqlmock.NewRows([]string{"id", "favorite_user_ids", "deleted_at"}).AddRow("campaign-1", pq.StringArray{}, nil))
			}

			output, err := r.Create(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				require.Equal(t, model.Campaign{}, output)
				require.NoError(t, mockDB.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output.Name, output.Name)
			require.Equal(t, tc.output.Description, output.Description)
			require.Equal(t, tc.output.Status, output.Status)
			require.Equal(t, tc.output.CreatedBy, output.CreatedBy)
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
		output []model.Campaign
		err    error
	}{
		"success": {
			input:  repository.GetOptions{Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
			output: []model.Campaign{{ID: "campaign-1"}},
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
				mockDB.ExpectQuery(`SELECT COUNT\(\*\) FROM project\.campaigns WHERE deleted_at IS NULL`).WillReturnError(tc.mock.countErr)
			} else {
				mockDB.ExpectQuery(`SELECT COUNT\(\*\) FROM project\.campaigns WHERE deleted_at IS NULL`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			}
			if tc.mock.listErr != nil {
				mockDB.ExpectQuery(`SELECT id, name, description, status, start_date, end_date, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.campaigns`).WillReturnError(tc.mock.listErr)
			} else if tc.mock.scanErr {
				values := campaignRowValues()
				values[0] = nil
				mockDB.ExpectQuery(`SELECT id, name, description, status, start_date, end_date, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.campaigns`).WillReturnRows(sqlmock.NewRows(campaignColumns()).AddRow(values...))
			} else if tc.mock.rowsErr {
				mockDB.ExpectQuery(`SELECT id, name, description, status, start_date, end_date, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.campaigns`).WillReturnRows(sqlmock.NewRows(campaignColumns()).AddRow(campaignRowValues()...).RowError(0, errors.New("rows error")))
			} else if tc.mock.countErr == nil {
				mockDB.ExpectQuery(`SELECT id, name, description, status, start_date, end_date, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.campaigns`).WillReturnRows(sqlmock.NewRows(campaignColumns()).AddRow(campaignRowValues()...))
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
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		input repository.UpdateOptions
		mock  struct {
			findErr   error
			updateErr error
			fetchErr  error
		}
		output model.Campaign
		err    error
	}{
		"success": {
			input:  repository.UpdateOptions{ID: "campaign-1", Name: "Campaign B", Description: "Desc B", Status: string(model.CampaignStatusActive), StartDate: &start, EndDate: &start},
			output: model.Campaign{ID: "campaign-1"},
		},
		"not_found": {
			input: repository.UpdateOptions{ID: "campaign-1", Name: "Campaign B"},
			mock: struct {
				findErr   error
				updateErr error
				fetchErr  error
			}{findErr: sql.ErrNoRows},
			err: repository.ErrFailedToGet,
		},
		"find_error": {
			input: repository.UpdateOptions{ID: "campaign-1", Name: "Campaign B"},
			mock: struct {
				findErr   error
				updateErr error
				fetchErr  error
			}{findErr: errors.New("find error")},
			err: repository.ErrFailedToUpdate,
		},
		"update_error": {
			input: repository.UpdateOptions{ID: "campaign-1", Name: "Campaign B"},
			mock: struct {
				findErr   error
				updateErr error
				fetchErr  error
			}{updateErr: errors.New("update error")},
			err: repository.ErrFailedToUpdate,
		},
		"fetch_error": {
			input: repository.UpdateOptions{ID: "campaign-1", Name: "Campaign B"},
			mock: struct {
				findErr   error
				updateErr error
				fetchErr  error
			}{fetchErr: errors.New("fetch error")},
			err: repository.ErrFailedToGet,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			findQuery := `select \* from "project"\."campaigns" where "id"=\$1 and "deleted_at" is null`
			if tc.mock.findErr != nil {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input.ID).WillReturnError(tc.mock.findErr)
			} else {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input.ID).WillReturnRows(sqlboilerCampaignRows(tc.input.ID))
				expect := mockDB.ExpectExec(`UPDATE "project"\."campaigns" SET`)
				if tc.mock.updateErr != nil {
					expect.WillReturnError(tc.mock.updateErr)
				} else {
					expect.WillReturnResult(sqlmock.NewResult(0, 1))
				}
				if tc.mock.updateErr == nil {
					fetchQuery := `SELECT id, name, description, status, start_date, end_date, favorite_user_ids, created_by, created_at, updated_at\s+FROM project\.campaigns`
					if tc.mock.fetchErr != nil {
						mockDB.ExpectQuery(fetchQuery).WillReturnError(tc.mock.fetchErr)
					} else {
						mockDB.ExpectQuery(fetchQuery).WillReturnRows(sqlmock.NewRows(campaignColumns()).AddRow(campaignRowValues()...))
					}
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

func TestFavorite(t *testing.T) {
	testFavoriteCommand(t, "favorite", func(r *implRepository, ctx context.Context, id, userID string) error {
		return r.Favorite(ctx, id, userID)
	}, `UPDATE project\.campaigns\s+SET favorite_user_ids`)
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
			input: "campaign-1",
		},
		"not_found": {
			input: "campaign-1",
			mock: struct {
				findErr   error
				updateErr error
			}{findErr: sql.ErrNoRows},
			err: repository.ErrFailedToGet,
		},
		"find_error": {
			input: "campaign-1",
			mock: struct {
				findErr   error
				updateErr error
			}{findErr: errors.New("find error")},
			err: repository.ErrFailedToDelete,
		},
		"update_error": {
			input: "campaign-1",
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
			findQuery := `select \* from "project"\."campaigns" where "id"=\$1 and "deleted_at" is null`
			if tc.mock.findErr != nil {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input).WillReturnError(tc.mock.findErr)
			} else {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input).WillReturnRows(sqlboilerCampaignRows(tc.input))
				expect := mockDB.ExpectExec(`UPDATE "project"\."campaigns" SET`)
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
	}, `UPDATE project\.campaigns\s+SET favorite_user_ids`)
}

func testFavoriteCommand(t *testing.T, name string, call func(*implRepository, context.Context, string, string) error, query string) {
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
			input: struct{ id, userID string }{id: "campaign-1", userID: "user-1"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
			}{rowsAffected: 1},
		},
		"exec_error": {
			input: struct{ id, userID string }{id: "campaign-1", userID: "user-1"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
			}{execErr: errors.New("exec error")},
			err: repository.ErrFailedToUpdate,
		},
		"rows_error": {
			input: struct{ id, userID string }{id: "campaign-1", userID: "user-1"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
			}{rowsErr: errors.New("rows error")},
			err: repository.ErrFailedToUpdate,
		},
		"not_found": {
			input: struct{ id, userID string }{id: "campaign-1", userID: "user-1"},
			mock: struct {
				execErr      error
				rowsAffected int64
				rowsErr      error
			}{rowsAffected: 0},
			err: repository.ErrFailedToGet,
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
