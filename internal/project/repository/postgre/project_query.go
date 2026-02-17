package postgre

import (
	"project-srv/internal/project/repository"
	"project-srv/internal/sqlboiler"

	"github.com/aarondl/sqlboiler/v4/queries/qm"
)

// buildGetQuery builds query mods for listing projects.
func (r *implRepository) buildGetQuery(opt repository.GetOptions) []qm.QueryMod {
	var mods []qm.QueryMod

	if opt.CampaignID != "" {
		mods = append(mods, sqlboiler.ProjectWhere.CampaignID.EQ(opt.CampaignID))
	}

	if opt.Status != "" {
		mods = append(mods, sqlboiler.ProjectWhere.Status.EQ(sqlboiler.ProjectStatus(opt.Status)))
	}

	if opt.Name != "" {
		mods = append(mods, qm.Where(sqlboiler.ProjectColumns.Name+" ILIKE ?", "%"+opt.Name+"%"))
	}

	if opt.Brand != "" {
		mods = append(mods, qm.Where(sqlboiler.ProjectColumns.Brand+" ILIKE ?", "%"+opt.Brand+"%"))
	}

	if opt.EntityType != "" {
		mods = append(mods, sqlboiler.ProjectWhere.EntityType.EQ(sqlboiler.EntityType(opt.EntityType)))
	}

	return mods
}
