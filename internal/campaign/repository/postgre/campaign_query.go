package postgre

import (
	"project-srv/internal/campaign/repository"
	"project-srv/internal/sqlboiler"

	"github.com/aarondl/sqlboiler/v4/queries/qm"
)

// buildGetQuery builds query mods for listing campaigns.
func (r *implRepository) buildGetQuery(opt repository.GetOptions) []qm.QueryMod {
	var mods []qm.QueryMod

	if opt.Status != "" {
		mods = append(mods, sqlboiler.CampaignWhere.Status.EQ(sqlboiler.CampaignStatus(opt.Status)))
	}

	if opt.Name != "" {
		mods = append(mods, qm.Where(sqlboiler.CampaignColumns.Name+" ILIKE ?", "%"+opt.Name+"%"))
	}

	if opt.CreatedBy != "" {
		mods = append(mods, sqlboiler.CampaignWhere.CreatedBy.EQ(opt.CreatedBy))
	}

	return mods
}
