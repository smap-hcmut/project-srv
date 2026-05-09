package usecase

import (
	"context"
	"strings"

	"project-srv/internal/crisis/repository"
	"project-srv/internal/model"
)

func (uc *implUseCase) ensureCrisisConfig(ctx context.Context, projectID, domainTypeCode string) {
	if uc.crisisRepo == nil {
		return
	}

	if _, err := uc.crisisRepo.Detail(ctx, projectID); err == nil {
		return
	}

	opt := buildCrisisPreset(projectID, domainTypeCode)
	if _, err := uc.crisisRepo.Upsert(ctx, opt); err != nil {
		uc.l.Warnf(ctx, "project.usecase.ensureCrisisConfig.Upsert: project_id=%s domain_type_code=%s err=%v", projectID, domainTypeCode, err)
	}
}

func (uc *implUseCase) attachCrisisConfig(ctx context.Context, p model.Project) model.Project {
	if uc.crisisRepo == nil || p.ID == "" {
		return p
	}

	uc.ensureCrisisConfig(ctx, p.ID, p.DomainTypeCode)
	cfg, err := uc.crisisRepo.Detail(ctx, p.ID)
	if err != nil {
		uc.l.Warnf(ctx, "project.usecase.attachCrisisConfig.Detail: project_id=%s err=%v", p.ID, err)
		return p
	}
	p.CrisisConfig = &cfg
	return p
}

func buildCrisisPreset(projectID, domainTypeCode string) repository.UpsertOptions {
	switch strings.ToLower(strings.TrimSpace(domainTypeCode)) {
	case "ahamove":
		return buildAhamoveLogisticsCrisisPreset(projectID)
	default:
		return buildDefaultCrisisPreset(projectID)
	}
}

func buildDefaultCrisisPreset(projectID string) repository.UpsertOptions {
	status := model.CrisisStatusNormal
	keywords := model.KeywordsTrigger{
		Enabled: true,
		Logic:   "OR",
		Groups: []model.KeywordGroup{
			{
				Name:     "Service risk",
				Keywords: []string{"complaint", "issue", "bad experience", "delay", "broken", "refund"},
				Weight:   6,
			},
		},
	}
	volume := model.VolumeTrigger{
		Enabled: true,
		Metric:  "MENTIONS",
		Rules: []model.VolumeRule{
			{
				Level:                  "WARNING",
				ThresholdPercentGrowth: 150,
				ComparisonWindowHours:  6,
				Baseline:               "AVERAGE_7D",
			},
			{
				Level:                  "CRITICAL",
				ThresholdPercentGrowth: 300,
				ComparisonWindowHours:  2,
				Baseline:               "AVERAGE_7D",
			},
		},
	}
	sentiment := model.SentimentTrigger{
		Enabled:       true,
		MinSampleSize: 10,
		Rules: []model.SentimentRule{
			{
				Type:             "NEGATIVE_SPIKE",
				ThresholdPercent: 35,
			},
		},
	}
	influencer := model.InfluencerTrigger{
		Enabled: true,
		Logic:   "OR",
		Rules: []model.InfluencerRule{
			{
				Type:              "HIGH_REACH",
				MinFollowers:      50000,
				RequiredSentiment: "NEGATIVE",
			},
			{
				Type:        "VIRAL_NEGATIVE",
				MinShares:   300,
				MinComments: 150,
			},
		},
	}
	responsePolicy := model.DefaultCrisisResponsePolicy()

	return repository.UpsertOptions{
		ProjectID:         projectID,
		Status:            &status,
		KeywordsTrigger:   &keywords,
		VolumeTrigger:     &volume,
		SentimentTrigger:  &sentiment,
		InfluencerTrigger: &influencer,
		ResponsePolicy:    &responsePolicy,
	}
}

func buildAhamoveLogisticsCrisisPreset(projectID string) repository.UpsertOptions {
	opt := buildDefaultCrisisPreset(projectID)

	keywords := model.KeywordsTrigger{
		Enabled: true,
		Logic:   "OR",
		Groups: []model.KeywordGroup{
			{
				Name: "Service failure",
				Keywords: []string{
					"giao cham",
					"tre don",
					"khong co tai xe",
					"huy don",
					"that lac",
					"mat hang",
					"vo hang",
					"giao sai",
					"khong giao duoc",
				},
				Weight: 10,
			},
			{
				Name: "Payment and COD",
				Keywords: []string{
					"cod loi",
					"thu ho sai",
					"thu them tien",
					"tinh sai tien",
					"khong hoan tien",
					"nap tien loi",
					"rut tien loi",
				},
				Weight: 8,
			},
			{
				Name: "Trust and safety",
				Keywords: []string{
					"lua dao",
					"scam",
					"gia mao",
					"tai xe vo y thuc",
					"khong an toan",
					"tai nan",
					"khieu nai",
				},
				Weight: 9,
			},
		},
	}
	volume := model.VolumeTrigger{
		Enabled: true,
		Metric:  "MENTIONS",
		Rules: []model.VolumeRule{
			{
				Level:                  "WARNING",
				ThresholdPercentGrowth: 180,
				ComparisonWindowHours:  6,
				Baseline:               "AVERAGE_7D",
			},
			{
				Level:                  "CRITICAL",
				ThresholdPercentGrowth: 350,
				ComparisonWindowHours:  2,
				Baseline:               "AVERAGE_7D",
			},
		},
	}
	sentiment := model.SentimentTrigger{
		Enabled:       true,
		MinSampleSize: 12,
		Rules: []model.SentimentRule{
			{
				Type:             "NEGATIVE_SPIKE",
				ThresholdPercent: 30,
			},
			{
				Type:                     "ASPECT_NEGATIVE",
				CriticalAspects:          []string{"delivery_speed", "delivery_fee", "driver_quality", "package_safety", "payment", "support_resolution", "trust_safety", "coverage"},
				NegativeThresholdPercent: 55,
			},
		},
	}
	influencer := model.InfluencerTrigger{
		Enabled: true,
		Logic:   "OR",
		Rules: []model.InfluencerRule{
			{
				Type:              "HIGH_REACH",
				MinFollowers:      50000,
				RequiredSentiment: "NEGATIVE",
			},
			{
				Type:        "VIRAL_NEGATIVE",
				MinShares:   300,
				MinComments: 150,
			},
		},
	}

	opt.KeywordsTrigger = &keywords
	opt.VolumeTrigger = &volume
	opt.SentimentTrigger = &sentiment
	opt.InfluencerTrigger = &influencer
	return opt
}
