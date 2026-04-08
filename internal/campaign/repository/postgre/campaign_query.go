package postgre

import (
	"fmt"
	"strings"

	"project-srv/internal/campaign/repository"

	"github.com/lib/pq"
)

func (r *implRepository) buildCampaignFilters(opt repository.GetOptions) (string, []any) {
	clauses := []string{"deleted_at IS NULL"}
	args := make([]any, 0, 4)

	if opt.Status != "" {
		args = append(args, opt.Status)
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}

	if opt.Name != "" {
		args = append(args, "%"+opt.Name+"%")
		clauses = append(clauses, fmt.Sprintf("name ILIKE $%d", len(args)))
	}

	if opt.CreatedBy != "" {
		args = append(args, opt.CreatedBy)
		clauses = append(clauses, fmt.Sprintf("created_by = $%d", len(args)))
	}

	if opt.FavoriteOnly {
		if opt.CurrentUserID == "" {
			clauses = append(clauses, "1 = 0")
		} else {
			args = append(args, pq.Array([]string{opt.CurrentUserID}))
			clauses = append(clauses, fmt.Sprintf("favorite_user_ids @> $%d::uuid[]", len(args)))
		}
	}

	return strings.Join(clauses, " AND "), args
}

func (r *implRepository) buildCampaignOrderBy(opt repository.GetOptions, args *[]any) string {
	if opt.Sort == "favorite_desc" && opt.CurrentUserID != "" {
		*args = append(*args, pq.Array([]string{opt.CurrentUserID}))
		return fmt.Sprintf("CASE WHEN favorite_user_ids @> $%d::uuid[] THEN 0 ELSE 1 END, created_at DESC", len(*args))
	}

	return "created_at DESC"
}
