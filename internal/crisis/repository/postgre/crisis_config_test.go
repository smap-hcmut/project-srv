package postgre

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"project-srv/internal/crisis/repository"
	"project-srv/internal/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

func newTestRepo(t *testing.T) (*implRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mockDB, err := sqlmock.New()
	require.NoError(t, err)
	l := log.NewZapLogger(log.ZapConfig{Level: log.LevelFatal, Mode: log.ModeProduction, Encoding: log.EncodingJSON})
	return New(db, l).(*implRepository), mockDB, func() { _ = db.Close() }
}

func crisisColumns() []string {
	return []string{"project_id", "status", "keywords_rules", "volume_rules", "sentiment_rules", "influencer_rules", "created_at", "updated_at"}
}

func crisisRows() *sqlmock.Rows {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return sqlmock.NewRows(crisisColumns()).AddRow(
		"project-1",
		"NORMAL",
		[]byte(`{"enabled":true,"logic":"AND","groups":[{"name":"Pin","keywords":["pin"],"weight":10}]}`),
		[]byte(`{"enabled":true,"metric":"MENTIONS","rules":[{"level":"CRITICAL","threshold_percent_growth":150,"comparison_window_hours":1,"baseline":"PREVIOUS_PERIOD"}]}`),
		[]byte(`{"enabled":true,"min_sample_size":10,"rules":[{"type":"NEGATIVE_SPIKE","threshold_percent":25}]}`),
		[]byte(`{"enabled":true,"logic":"OR","rules":[{"type":"HIGH_REACH","min_followers":1000}]}`),
		now,
		now,
	)
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

func TestDetail(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input  string
		mock   struct{ err error }
		output model.CrisisConfig
		err    error
	}{
		"success": {
			input:  "project-1",
			output: model.CrisisConfig{ProjectID: "project-1", Status: model.CrisisStatusNormal},
		},
		"not_found": {
			input: "project-1",
			mock:  struct{ err error }{err: sql.ErrNoRows},
			err:   repository.ErrFailedToGet,
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
			query := `select \* from "project"\."projects_crisis_config" where "project_id"=\$1`
			if tc.mock.err != nil {
				mockDB.ExpectQuery(query).WithArgs(tc.input).WillReturnError(tc.mock.err)
			} else {
				mockDB.ExpectQuery(query).WithArgs(tc.input).WillReturnRows(crisisRows())
			}

			output, err := r.Detail(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				require.NoError(t, mockDB.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output.ProjectID, output.ProjectID)
			require.Equal(t, tc.output.Status, output.Status)
			require.True(t, output.KeywordsTrigger.Enabled)
			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestUpsert(t *testing.T) {
	ctx := context.Background()
	opt := repository.UpsertOptions{
		ProjectID:        "project-1",
		KeywordsTrigger:  &model.KeywordsTrigger{Enabled: true, Logic: "AND", Groups: []model.KeywordGroup{{Name: "Pin", Keywords: []string{"pin"}, Weight: 10}}},
		VolumeTrigger:    &model.VolumeTrigger{Enabled: true, Metric: "MENTIONS", Rules: []model.VolumeRule{{Level: "CRITICAL", ThresholdPercentGrowth: 150, ComparisonWindowHours: 1}}},
		SentimentTrigger: &model.SentimentTrigger{Enabled: true, Rules: []model.SentimentRule{{Type: "NEGATIVE_SPIKE"}}},
		InfluencerTrigger: &model.InfluencerTrigger{Enabled: true, Logic: "OR", Rules: []model.InfluencerRule{{
			Type:         "HIGH_REACH",
			MinFollowers: 1000,
		}}},
	}

	tcs := map[string]struct {
		input repository.UpsertOptions
		mock  struct {
			findErr   error
			insertErr error
			updateErr error
		}
		output model.CrisisConfig
		err    error
	}{
		"create_success": {
			input: opt,
			mock: struct {
				findErr   error
				insertErr error
				updateErr error
			}{findErr: sql.ErrNoRows},
			output: model.CrisisConfig{ProjectID: "project-1"},
		},
		"create_insert_error": {
			input: opt,
			mock: struct {
				findErr   error
				insertErr error
				updateErr error
			}{findErr: sql.ErrNoRows, insertErr: errors.New("insert error")},
			err: repository.ErrFailedToInsert,
		},
		"find_error": {
			input: opt,
			mock: struct {
				findErr   error
				insertErr error
				updateErr error
			}{findErr: errors.New("find error")},
			err: repository.ErrFailedToInsert,
		},
		"update_success": {
			input:  opt,
			output: model.CrisisConfig{ProjectID: "project-1"},
		},
		"update_error": {
			input: opt,
			mock: struct {
				findErr   error
				insertErr error
				updateErr error
			}{updateErr: errors.New("update error")},
			err: repository.ErrFailedToUpdate,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			findQuery := `select \* from "project"\."projects_crisis_config" where "project_id"=\$1`
			if tc.mock.findErr != nil {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input.ProjectID).WillReturnError(tc.mock.findErr)
				if tc.mock.findErr == sql.ErrNoRows {
					expect := mockDB.ExpectQuery(`INSERT INTO "project"\."projects_crisis_config"`)
					if tc.mock.insertErr != nil {
						expect.WillReturnError(tc.mock.insertErr)
					} else {
						expect.WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("NORMAL"))
					}
				}
			} else {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input.ProjectID).WillReturnRows(crisisRows())
				expect := mockDB.ExpectExec(`UPDATE "project"\."projects_crisis_config" SET`)
				if tc.mock.updateErr != nil {
					expect.WillReturnError(tc.mock.updateErr)
				} else {
					expect.WillReturnResult(sqlmock.NewResult(0, 1))
				}
			}

			output, err := r.Upsert(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				require.NoError(t, mockDB.ExpectationsWereMet())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output.ProjectID, output.ProjectID)
			require.NoError(t, mockDB.ExpectationsWereMet())
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input string
		mock  struct {
			findErr      error
			deleteErr    error
			rowsAffected int64
			rowsErr      error
		}
		output struct{}
		err    error
	}{
		"success": {
			input: "project-1",
			mock: struct {
				findErr      error
				deleteErr    error
				rowsAffected int64
				rowsErr      error
			}{rowsAffected: 1},
		},
		"not_found": {
			input: "project-1",
			mock: struct {
				findErr      error
				deleteErr    error
				rowsAffected int64
				rowsErr      error
			}{findErr: sql.ErrNoRows},
			err: repository.ErrFailedToGet,
		},
		"find_error": {
			input: "project-1",
			mock: struct {
				findErr      error
				deleteErr    error
				rowsAffected int64
				rowsErr      error
			}{findErr: errors.New("find error")},
			err: repository.ErrFailedToDelete,
		},
		"delete_error": {
			input: "project-1",
			mock: struct {
				findErr      error
				deleteErr    error
				rowsAffected int64
				rowsErr      error
			}{deleteErr: errors.New("delete error")},
			err: repository.ErrFailedToDelete,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			r, mockDB, done := newTestRepo(t)
			defer done()
			findQuery := `select \* from "project"\."projects_crisis_config" where "project_id"=\$1`
			if tc.mock.findErr != nil {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input).WillReturnError(tc.mock.findErr)
			} else {
				mockDB.ExpectQuery(findQuery).WithArgs(tc.input).WillReturnRows(crisisRows())
				deleteQuery := `DELETE FROM "project"\."projects_crisis_config" WHERE "project_id"=\$1`
				expect := mockDB.ExpectExec(deleteQuery).WithArgs(tc.input)
				if tc.mock.deleteErr != nil {
					expect.WillReturnError(tc.mock.deleteErr)
				} else if tc.mock.rowsErr != nil {
					expect.WillReturnResult(sqlmock.NewErrorResult(tc.mock.rowsErr))
				} else {
					expect.WillReturnResult(sqlmock.NewResult(0, tc.mock.rowsAffected))
				}
			}

			err := r.Delete(ctx, tc.input)

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
