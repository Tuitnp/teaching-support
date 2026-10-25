package service

import (
	"context"
	"fmt"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/ai"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/config"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/security"
)

func CreateAICredential(userID string, form *models.AICredentialForm) (*models.ResponseAICredential, error) {
	key, err := getEncryptionKey()
	if err != nil {
		return nil, err
	}

	encryptedAPIKey, err := security.Encrypt(form.APIKey, key)
	if err != nil {
		return nil, err
	}

	credential := &models.AICredential{
		Name:            form.Name,
		UserID:          userID,
		Provider:        form.Provider,
		BaseURL:         form.BaseURL,
		EncryptedAPIKey: encryptedAPIKey,
	}

	err = repository.CreateAICredential(credential)
	if err != nil {
		return nil, err
	}

	const layout = "2006-01-02"

	response := &models.ResponseAICredential{
		ID:        credential.ID,
		Name:      credential.Name,
		Provider:  credential.Provider,
		BaseURL:   credential.BaseURL,
		CreatedAt: credential.CreatedAt.Format(layout),
	}

	return response, nil
}

func GetAICredentialsByUser(userID string) ([]models.ResponseAICredential, error) {
	credentials, err := repository.GetAICredentials(userID)
	if err != nil {
		return nil, err
	}

	const layout = "2006-01-02"

	response := make([]models.ResponseAICredential, len(credentials))
	for i, cred := range credentials {
		response[i] = models.ResponseAICredential{
			ID:        cred.ID,
			Name:      cred.Name,
			Provider:  cred.Provider,
			BaseURL:   cred.BaseURL,
			CreatedAt: cred.CreatedAt.Format(layout),
		}
	}

	return response, nil
}

func UpdateAICredentialByID(userID, credentialID string, form *models.AICredentialForm) (*models.ResponseAICredential, error) {
	credential, err := repository.GetAICredential(userID, credentialID)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}

	if form.Name != "" {
		updates["name"] = form.Name
	}

	if form.Provider != "" {
		updates["provider"] = form.Provider
	}

	if form.BaseURL != "" {
		updates["base_url"] = form.BaseURL
	}

	if form.APIKey != "" {
		key, err := getEncryptionKey()
		if err != nil {
			return nil, err
		}

		encryptedAPIKey, err := security.Encrypt(form.APIKey, key)
		if err != nil {
			return nil, err
		}

		updates["encrypted_api_key"] = encryptedAPIKey
	}

	if len(updates) > 0 {
		if err := repository.UpdateAICredential(credential.ID, updates); err != nil {
			return nil, err
		}

		credential, err = repository.GetAICredential(userID, credentialID)
		if err != nil {
			return nil, err
		}
	}

	const layout = "2006-01-02"

	response := &models.ResponseAICredential{
		ID:        credential.ID,
		Name:      credential.Name,
		Provider:  credential.Provider,
		BaseURL:   credential.BaseURL,
		CreatedAt: credential.CreatedAt.Format(layout),
	}

	return response, nil
}

func DeleteAICredentialByID(userID, credentialID string) error {
	credential, err := repository.GetAICredential(userID, credentialID)
	if err != nil {
		return err
	}

	return repository.DeleteAICredential(credential.ID)
}

func getEncryptionKey() ([]byte, error) {
	key := config.GetEnv("AI_SECRET_KEY")

	if len(key) == 0 {
		return nil, fmt.Errorf("AI_SECRET_KEY not set")
	}

	return []byte(key), nil
}

func CreateAIConfig(userID, aiCredentialID string, form *models.AIConfigForm) (*models.ResponseAIConfig, error) {
	credential, err := repository.GetAICredential(userID, aiCredentialID)
	if err != nil {
		return nil, err
	}

	temperature := form.Temperature
	if temperature == 0 {
		temperature = 0.2
	}

	config := &models.AIConfig{
		Name:           form.Name,
		UserID:         userID,
		AICredentialID: credential.ID,
		Model:          form.Model,
		Temperature:    temperature,
	}

	err = repository.CreateAIConfig(config)
	if err != nil {
		return nil, err
	}

	response := models.ResponseAIConfig{
		ConfigName:     form.Name,
		CredentialName: credential.Name,
		Model:          form.Model,
		Temperature:    temperature,
	}

	return &response, nil
}

func GetAIConfigs(userID, aiCredentialID string) ([]models.ResponseAIConfig, error) {
	configs, err := repository.GetAIConfigs(userID, aiCredentialID)
	if err != nil {
		return nil, err
	}

	response := models.ConvertAIConfigsToResponse(configs)

	return response, nil
}

func GetAIConfigByID(userID, aiConfigID string) (*models.ResponseAIConfig, error) {
	config, err := repository.GetAIConfigByID(userID, aiConfigID)
	if err != nil {
		return nil, err
	}

	const layout = "2006-01-02"

	response := &models.ResponseAIConfig{
		ID:             config.ID,
		CredentialID:   config.AICredentialID,
		ConfigName:     config.Name,
		CredentialName: config.AICredential.Name,
		Model:          config.Model,
		Temperature:    config.Temperature,
		CreatedAt:      config.CreatedAt.Format(layout),
	}

	return response, nil
}

func UpdateAIConfig(
	userID,
	aiConfigID string,
	form *models.AIConfigForm,
) (*models.ResponseAIConfig, error) {
	if err := models.ValidateUUIDs(aiConfigID, form.AICredentialID); err != nil {
		return nil, err
	}

	config, err := repository.GetAIConfig(userID, aiConfigID)
	if err != nil {
		return nil, err
	}

	_, err = repository.GetAICredential(userID, form.AICredentialID)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}

	if form.Name != "" {
		updates["name"] = form.Name
	}

	if form.AICredentialID != "" {
		updates["ai_credential_id"] = form.AICredentialID
	}

	if form.Model != "" {
		updates["model"] = form.Model
	}

	updates["temperature"] = form.Temperature

	if len(updates) > 0 {
		if err := repository.UpdateAIConfig(userID, aiConfigID, updates); err != nil {
			return nil, err
		}

		config, err = repository.GetAIConfig(userID, aiConfigID)
		if err != nil {
			return nil, err
		}
	}

	response := models.ResponseAIConfig{
		ID:             config.ID,
		CredentialID:   config.AICredentialID,
		ConfigName:     config.Name,
		CredentialName: config.AICredential.Name,
		Model:          config.Model,
		Temperature:    config.Temperature,
		CreatedAt:      config.CreatedAt.Format("2006-01-02"),
	}
	return &response, nil
}

func DeleteAIConfig(userID, aiConfigID string) error {
	config, err := repository.GetAIConfig(userID, aiConfigID)
	if err != nil {
		return err
	}

	return repository.DeleteAIConfig(config.ID)
}

func GetModels(userID, aiCredentialID string) ([]models.ResponseModel, error) {
	// Get user's AI config
	credential, err := repository.GetAICredential(userID, aiCredentialID)
	if err != nil {
		return nil, err
	}

	// Decrypt API key
	key, err := getEncryptionKey()
	if err != nil {
		return nil, err
	}

	apiKey, err := security.Decrypt(credential.EncryptedAPIKey, key)
	if err != nil {
		return nil, err
	}

	// Create credential
	credentialPayload := ai.Credential{
		Provider: credential.Provider,
		APIKey:   apiKey,
		BaseURL:  credential.BaseURL,
	}

	provider, err := ai.GetProvider(credential.Provider)
	if err != nil {
		return nil, err
	}

	modelList, err := provider.ListModels(context.Background(), credentialPayload)
	if err != nil {
		return nil, err
	}

	return models.ConvertAIModelsToResponse(modelList), nil
}

func GetGradingJobs(teacherID string) (*models.GradingJobsResponse, error) {
	gradingJobs, err := repository.GetGradingJobsByTeacher(teacherID)
	if err != nil {
		return nil, err
	}

	jobs := make([]models.GradingJobResponse, len(gradingJobs))
	for i, job := range gradingJobs {
		jobs[i] = models.GradingJobResponse{
			ID:                   job.ID,
			CourseID:             job.Assignment.CourseID,
			AssignmentID:         job.AssignmentID,
			AssignmentTitle:      job.Assignment.Title,
			Status:               job.Status,
			Progress:             job.Progress,
			TotalSubmissions:     job.TotalSubmissions,
			ProcessedSubmissions: job.ProcessedSubmissions,
			Error:                job.Error,
			StartedAt:            job.StartedAt,
			CompletedAt:          job.CompletedAt,
			CreatedAt:            job.CreatedAt,
			UpdatedAt:            job.UpdatedAt,
		}
	}

	return &models.GradingJobsResponse{Jobs: jobs}, nil
}

func CreatePromptTemplate(userID string, form *models.PromptTemplateForm) (*models.ResponsePrompt, error) {
	prompt := &models.PromptTemplate{
		UserID: userID,
		Name:   form.Name,
		Prompt: form.Prompt,
	}

	if err := repository.CreatePromptTemplate(prompt); err != nil {
		return nil, err
	}

	return &models.ResponsePrompt{
		Name:   prompt.Name,
		Prompt: prompt.Prompt,
	}, nil
}

func GetPromptTemplates(userID string) ([]models.ResponsePrompt, error) {
	prompts, err := repository.GetPromptTemplatesByUser(userID)
	if err != nil {
		return nil, err
	}

	response := make([]models.ResponsePrompt, len(prompts))
	for i, p := range prompts {
		response[i] = models.ResponsePrompt{
			Name:   p.Name,
			Prompt: p.Prompt,
		}
	}

	return response, nil
}

func DeletePromptTemplate(userID, promptID string) error {
	_, err := repository.GetPromptTemplateByID(userID, promptID)
	if err != nil {
		return err
	}

	return repository.DeletePromptTemplateByID(userID, promptID)
}

func DeleteGradingJob(gradingJobID, teacherID string) error {
	// Get the grading job to check its status
	job, err := repository.GetGradingJobByID(gradingJobID, teacherID)
	if err != nil {
		return err
	}

	if job == nil {
		return errors.ErrGradingJobNotFound
	}

	// Only allow deletion of completed or failed jobs
	if job.Status != models.JobCompleted && job.Status != models.JobFailed {
		return errors.ErrCannotDeleteJob
	}

	// Delete the job
	err = repository.DeleteGradingJob(gradingJobID, teacherID)
	if err != nil {
		return err
	}

	return nil
}
