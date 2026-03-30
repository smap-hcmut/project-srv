package postgre

import (
	"fmt"
	"strings"

	"project-srv/internal/project/repository"

	"github.com/lib/pq"
)

func (r *implRepository) buildProjectFilters(opt repository.GetOptions) (string, []any) {
	clauses := []string{"deleted_at IS NULL"}
	args := make([]any, 0, 8)

	if opt.CampaignID != "" {
		args = append(args, opt.CampaignID)
		clauses = append(clauses, fmt.Sprintf("campaign_id = $%d", len(args)))
	}

	if opt.Status != "" {
		args = append(args, opt.Status)
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}

	if opt.Name != "" {
		args = append(args, "%"+opt.Name+"%")
		clauses = append(clauses, fmt.Sprintf("name ILIKE $%d", len(args)))
	}

	if opt.Brand != "" {
		args = append(args, "%"+opt.Brand+"%")
		clauses = append(clauses, fmt.Sprintf("brand ILIKE $%d", len(args)))
	}

	if opt.EntityType != "" {
		args = append(args, opt.EntityType)
		clauses = append(clauses, fmt.Sprintf("entity_type = $%d", len(args)))
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

func (r *implRepository) buildProjectOrderBy(opt repository.GetOptions, args *[]any) string {
	if opt.Sort == "favorite_desc" && opt.CurrentUserID != "" {
		*args = append(*args, pq.Array([]string{opt.CurrentUserID}))
		return fmt.Sprintf("CASE WHEN favorite_user_ids @> $%d::uuid[] THEN 0 ELSE 1 END, created_at DESC", len(*args))
	}

	return "created_at DESC"
}
