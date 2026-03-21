package usecase

import "project-srv/internal/project"

func (uc *implUseCase) normalizeActivationReadinessCommand(command project.ActivationReadinessCommand) project.ActivationReadinessCommand {
	switch command {
	case project.ActivationReadinessCommandResume:
		return project.ActivationReadinessCommandResume
	default:
		return project.ActivationReadinessCommandActivate
	}
}

func (uc *implUseCase) mapReadinessBlockedError(readiness project.ActivationReadiness) error {
	if len(readiness.Errors) == 0 {
		return project.ErrReadinessFailed
	}

	switch readiness.Errors[0].Code {
	case project.ActivationReadinessCodeDatasourceRequired:
		return project.ErrReadinessDatasourceRequired
	case project.ActivationReadinessCodePassiveUnconfirmed:
		return project.ErrReadinessPassiveUnconfirmed
	case project.ActivationReadinessCodeTargetDryrunMissing:
		return project.ErrReadinessTargetDryrunMissing
	case project.ActivationReadinessCodeTargetDryrunFailed:
		return project.ErrReadinessTargetDryrunFailed
	case project.ActivationReadinessCodeActiveTargetRequired:
		return project.ErrReadinessActiveTargetMissing
	case project.ActivationReadinessCodeDatasourceStatus:
		return project.ErrReadinessDatasourceStatus
	default:
		return project.ErrReadinessFailed
	}
}
