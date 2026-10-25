package models

import (
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/ai"
)

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ConvertCourseToResponse(course *Course) ResponseCourse {
	return ResponseCourse{
		ID:         course.ID,
		Name:       course.Name,
		CourseCode: course.CourseCode,
		DayOfWeek:  course.DayOfWeek,
		StartTime:  course.StartTime,
		EndTime:    course.EndTime,
		Room:       course.Room,
		Credits:    course.Credits,
		Section:    course.Section,
		Semester:   course.Semester,
		Teacher:    course.Teacher.EnName,
	}
}

func ConvertCoursesToResponse(courses []Course) []ResponseCourse {
	response := make([]ResponseCourse, 0, len(courses))

	for i := range courses {
		response = append(response, ConvertCourseToResponse(&courses[i]))
	}

	return response
}

func ConvertCourseMemberToResponse(courseMember *CourseMember) ResponseMember {
	return ResponseMember{
		UserID:   courseMember.UserID,
		Username: courseMember.User.Username,
		EnName:   courseMember.User.EnName,
		ThName:   courseMember.User.ThName,
		Email:    courseMember.User.Email,
		Status:   courseMember.Status,
		Role:     courseMember.Role,
	}
}

func ConvertAssignmentToResponse(
	a *Assignment,
	hasOverride bool,
	override *AssignmentOverride,
) ResponseAssignment {
	tagNames := make([]string, len(a.Tags))
	for i, tag := range a.Tags {
		tagNames[i] = tag.Name
	}

	const layout = "2006-01-02"

	attachments := make([]ResponseAttachment, len(a.Attachments))
	for i, att := range a.Attachments {
		attachments[i] = ResponseAttachment{
			ID:        att.ID,
			FileName:  att.FileName,
			FileType:  att.FileType,
			Size:      att.Size,
			CreatedAt: att.CreatedAt.Format(layout),
		}
	}

	var effectiveOverride *AssignmentOverride
	if hasOverride {
		effectiveOverride = override
	}
	dueDate, closeDate := EffectiveDeadlines(a, effectiveOverride)

	return ResponseAssignment{
		ID:          a.ID,
		Title:       a.Title,
		Description: a.Description,
		Point:       a.Point,
		StartDate:   a.StartDate.UTC().Format(time.RFC3339Nano),
		DueDate:     dueDate.UTC().Format(time.RFC3339Nano),
		CloseDate:   closeDate.UTC().Format(time.RFC3339Nano),
		Attachments: attachments,
		Tags:        tagNames,
		AIConfigID:  derefString(a.AIConfigID),
		Prompt:      derefString(a.Prompt),
		Visible:     a.Visible,
	}
}

func ConvertAssignmentsToResponse(
	assignments []Assignment,
	overrides map[string]AssignmentOverride,
) []ResponseAssignment {

	response := make([]ResponseAssignment, 0, len(assignments))

	for i := range assignments {
		a := &assignments[i]

		override, ok := overrides[a.ID]

		response = append(response, ConvertAssignmentToResponse(a, ok, &override))
	}

	return response
}

func ConvertDetailedAssignmentToResponse(
	a *Assignment,
	s *[]Submission,
	o *AssignmentOverride,
) ResponseDetailedAssignment {

	assignmentResponse := ConvertAssignmentToResponse(a, o != nil, o)

	submissionResponses := make([]ResponseSubmission, len(*s))

	for i, sub := range *s {
		var submissionFileName *string
		if sub.Attachment != nil {
			submissionFileName = &sub.Attachment.FileName
		}

		commentsResponse := make([]ResponseComment, len(sub.Comments))
		for j, c := range sub.Comments {
			commentsResponse[j] = ResponseComment{
				ID:        c.ID,
				Comment:   c.Comment,
				CreatedBy: string(c.CreatedByRole),
				Visible:   c.Visible,
			}
		}

		var gradedBy *string
		if sub.GradedBy != nil {
			gb := string(*sub.GradedBy)
			gradedBy = &gb
		}

		submissionResponses[i] = ResponseSubmission{
			ID:           sub.ID,
			Submitter:    sub.Student.Username + "|" + sub.Student.EnName + "|" + sub.Student.ThName,
			Point:        sub.Point,
			AttachmentID: sub.AttachmentID,
			FileName:     submissionFileName,
			Comments:     commentsResponse,
			GradedBy:     gradedBy,
		}
	}

	return ResponseDetailedAssignment{
		Assignment:  assignmentResponse,
		Submissions: submissionResponses,
	}
}

func ConvertSubmissionToResponse(s *Submission) ResponseSubmission {

	var fileName *string
	if s.Attachment != nil {
		fileName = &s.Attachment.FileName
	}

	return ResponseSubmission{
		ID:           s.ID,
		Point:        s.Point,
		GradedBy:     (*string)(s.GradedBy),
		AttachmentID: s.AttachmentID,
		FileName:     fileName,
	}
}

func ConvertAssignmentTemplatesToResponse(
	templates []AssignmentTemplate,
) []ResponseAssignmentTemplates {

	response := make([]ResponseAssignmentTemplates, 0, len(templates))

	for _, template := range templates {

		tagNames := make([]string, len(template.Tags))
		for i, tag := range template.Tags {
			tagNames[i] = tag.Name
		}

		const layout = "2006-01-02"

		attachments := make([]ResponseAttachment, len(template.Attachments))
		for i, att := range template.Attachments {
			attachments[i] = ResponseAttachment{
				ID:        att.ID,
				FileName:  att.FileName,
				FileType:  att.FileType,
				Size:      att.Size,
				CreatedAt: att.CreatedAt.Format(layout),
			}
		}

		data := ResponseAssignmentTemplates{
			ID:          template.ID,
			Title:       template.Title,
			Description: template.Description,
			Point:       template.Point,
			Attachments: attachments,
			Tags:        tagNames,
			AIConfigID:  derefString(template.AIConfigID),
			Prompt:      derefString(template.Prompt),
		}

		response = append(response, data)
	}

	return response
}

func ConvertAIModelToResponse(model ai.Model) ResponseModel {
	return ResponseModel{
		ID:   model.ID,
		Name: model.Name,
	}
}

func ConvertAIModelsToResponse(models []ai.Model) []ResponseModel {
	response := make([]ResponseModel, 0, len(models))

	for _, model := range models {
		response = append(response, ConvertAIModelToResponse(model))
	}

	return response
}

func ConvertAssignmentTemplatesToShortResponse(
	templates []AssignmentTemplate,
) []ResponseShortAssignmentTemplates {

	response := make([]ResponseShortAssignmentTemplates, 0, len(templates))

	for _, template := range templates {

		tagNames := make([]string, len(template.Tags))
		for i, tag := range template.Tags {
			tagNames[i] = tag.Name
		}

		data := ResponseShortAssignmentTemplates{
			ID:    template.ID,
			Title: template.Title,
			Tags:  tagNames,
		}

		response = append(response, data)
	}

	return response
}

func ConvertAssignmentTemplateToResponse(
	template *AssignmentTemplate,
) *ResponseAssignmentTemplate {

	tagNames := make([]string, len(template.Tags))
	for i, tag := range template.Tags {
		tagNames[i] = tag.Name
	}

	const layout = "2006-01-02"

	attachments := make([]ResponseAttachment, len(template.Attachments))
	for i, att := range template.Attachments {
		attachments[i] = ResponseAttachment{
			ID:        att.ID,
			FileName:  att.FileName,
			FileType:  att.FileType,
			Size:      att.Size,
			CreatedAt: att.CreatedAt.Format(layout),
		}
	}

	response := ResponseAssignmentTemplate{
		ID:          template.ID,
		Title:       template.Title,
		Description: template.Description,
		Point:       template.Point,
		Attachments: attachments,
		Tags:        tagNames,
		AIConfigID:  derefString(template.AIConfigID),
		Prompt:      derefString(template.Prompt),
		Visible:     true,
	}

	return &response
}

func ConvertAttachmentsToResponse(
	attachments []Attachment,
) []ResponseAttachment {
	const layout = "2006-01-02"

	response := make([]ResponseAttachment, len(attachments))
	for i, att := range attachments {
		response[i] = ResponseAttachment{
			ID:        att.ID,
			FileName:  att.FileName,
			FileType:  att.FileType,
			Size:      att.Size,
			CreatedAt: att.CreatedAt.Format(layout),
		}
	}

	return response
}

func ConvertAIConfigsToResponse(
	configs []AIConfig,
) []ResponseAIConfig {
	const layout = "2006-01-02"

	response := make([]ResponseAIConfig, len(configs))
	for i, cf := range configs {
		response[i] = ResponseAIConfig{
			ID:             cf.ID,
			CredentialID:   cf.AICredentialID,
			ConfigName:     cf.Name,
			CredentialName: cf.AICredential.Name,
			Model:          cf.Model,
			Temperature:    cf.Temperature,
			CreatedAt:      cf.CreatedAt.Format(layout),
		}
	}

	return response
}
