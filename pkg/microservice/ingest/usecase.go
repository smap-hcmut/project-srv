package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"project-srv/pkg/microservice"
	"strings"
)

func (uc *implUseCase) GetActivationReadiness(ctx context.Context, input microservice.ActivationReadinessInput) (microservice.ActivationReadiness, error) {
	projectID := strings.TrimSpace(input.ProjectID)
	endpoint := uc.buildEndpoint(fmt.Sprintf("/projects/%s/activation-readiness", url.PathEscape(projectID)))
	if command := strings.TrimSpace(string(input.Command)); command != "" {
		values := url.Values{}
		values.Set("command", command)
		endpoint = endpoint + "?" + values.Encode()
	}
	body, status, err := uc.doRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return microservice.ActivationReadiness{}, err
	}

	if status != http.StatusOK {
		return microservice.ActivationReadiness{}, mapStatusError(status, body)
	}

	var envelope responseEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return microservice.ActivationReadiness{}, fmt.Errorf("%w: unmarshal readiness envelope: %v", microservice.ErrRequestFailed, err)
	}

	var dto readinessRespDTO
	if err := json.Unmarshal(envelope.Data, &dto); err != nil {
		return microservice.ActivationReadiness{}, fmt.Errorf("%w: unmarshal readiness data: %v", microservice.ErrRequestFailed, err)
	}

	out := microservice.ActivationReadiness{
		ProjectID:                dto.ProjectID,
		DataSourceCount:          dto.DataSourceCount,
		HasDatasource:            dto.HasDatasource,
		PassiveUnconfirmedCount:  dto.PassiveUnconfirmedCount,
		MissingTargetDryrunCount: dto.MissingTargetDryrunCount,
		FailedTargetDryrunCount:  dto.FailedTargetDryrunCount,
		CanProceed:               dto.CanProceed,
		Errors:                   make([]microservice.ActivationReadinessError, 0, len(dto.Errors)),
	}

	for _, e := range dto.Errors {
		out.Errors = append(out.Errors, microservice.ActivationReadinessError{
			Code:         e.Code,
			Message:      e.Message,
			DataSourceID: e.DataSourceID,
			TargetID:     e.TargetID,
		})
	}

	if out.ProjectID == "" {
		out.ProjectID = strings.TrimSpace(projectID)
	}

	return out, nil
}

func (uc *implUseCase) Activate(ctx context.Context, projectID string) error {
	endpoint := uc.buildEndpoint(fmt.Sprintf("/projects/%s/activate", url.PathEscape(strings.TrimSpace(projectID))))
	body, status, err := uc.doRequest(ctx, http.MethodPost, endpoint)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return mapStatusError(status, body)
	}
	return nil
}

func (uc *implUseCase) Pause(ctx context.Context, projectID string) error {
	endpoint := uc.buildEndpoint(fmt.Sprintf("/projects/%s/pause", url.PathEscape(strings.TrimSpace(projectID))))
	body, status, err := uc.doRequest(ctx, http.MethodPost, endpoint)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return mapStatusError(status, body)
	}
	return nil
}

func (uc *implUseCase) Resume(ctx context.Context, projectID string) error {
	endpoint := uc.buildEndpoint(fmt.Sprintf("/projects/%s/resume", url.PathEscape(strings.TrimSpace(projectID))))
	body, status, err := uc.doRequest(ctx, http.MethodPost, endpoint)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return mapStatusError(status, body)
	}
	return nil
}

func (uc *implUseCase) doRequest(ctx context.Context, method, endpoint string) ([]byte, int, error) {
	headers := map[string]string{
		"Accept": "application/json",
	}
	if uc.internalKey != "" {
		headers[internalAuthHeader] = uc.internalKey
	}

	switch method {
	case http.MethodGet:
		body, status, err := uc.client.Get(ctx, endpoint, headers)
		if err != nil {
			return nil, status, fmt.Errorf("%w: %v", microservice.ErrRequestFailed, err)
		}
		return body, status, nil
	case http.MethodPost:
		body, status, err := uc.client.Post(ctx, endpoint, nil, headers)
		if err != nil {
			return nil, status, fmt.Errorf("%w: %v", microservice.ErrRequestFailed, err)
		}
		return body, status, nil
	default:
		return nil, 0, fmt.Errorf("%w: unsupported method=%s", microservice.ErrRequestFailed, method)
	}
}

func mapStatusError(status int, body []byte) error {
	trimmedBody := strings.TrimSpace(string(body))
	switch status {
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", microservice.ErrBadRequest, trimmedBody)
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", microservice.ErrUnauthorized, trimmedBody)
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", microservice.ErrForbidden, trimmedBody)
	default:
		return fmt.Errorf("%w: status=%d body=%s", microservice.ErrRequestFailed, status, trimmedBody)
	}
}
