package service

import (
	stderrors "errors"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
)

func CreateOrUpdateComment(
	courseID,
	submissionID,
	userID,
	role string,
	form *models.CommentForm,
) (*models.ResponseComment, error) {

	err := repository.CheckSubmissionInCourse(courseID, submissionID)
	if err != nil {
		return nil, err
	}

	comment, err := repository.GetCommentBySubmissionAndUserAndRole(submissionID, userID, role)

	if stderrors.Is(err, errors.ErrCommentNotFound) {

		newComment := &models.Comment{
			SubmissionID:  submissionID,
			Comment:       form.Comment,
			CreatedByRole: models.Role(role),
			CreatedBy:     userID,
			Visible:       form.Visible,
		}

		comment, err = repository.CreateComment(newComment)
		if err != nil {
			return nil, err
		}

	} else if err != nil {
		return nil, err
	} else {

		comment.Comment = form.Comment
		comment.Visible = form.Visible
		comment.CreatedByRole = models.Role(role)

		if err := repository.UpdateComment(comment); err != nil {
			return nil, err
		}
	}

	response := models.ResponseComment{
		ID:        comment.ID,
		Comment:   comment.Comment,
		CreatedBy: role,
		Visible:   comment.Visible,
	}

	return &response, nil
}

func ToggleComment(
	courseID,
	submissionID,
	commentID,
	userID,
	role string,
) (*models.ResponseComment, error) {

	comment, err := repository.GetCommentByIDAndCourse(
		commentID,
		submissionID,
		courseID,
	)
	if err != nil {
		return nil, err
	}

	allowed :=
		models.HasPermission(role, "assignment:comment_any") ||
			(models.HasPermission(role, "assignment:comment_own") &&
				comment.CreatedBy == userID && string(comment.CreatedByRole) == role)

	if !allowed {
		return nil, errors.ErrForbidden
	}

	comment.Visible = !comment.Visible

	if err := repository.UpdateComment(comment); err != nil {
		return nil, err
	}

	response := models.ResponseComment{
		ID:        comment.ID,
		Comment:   comment.Comment,
		CreatedBy: string(comment.CreatedByRole),
		Visible:   comment.Visible,
	}

	return &response, nil
}
