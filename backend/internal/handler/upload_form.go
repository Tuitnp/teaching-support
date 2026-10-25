package handler

import (
	"mime/multipart"
	"strings"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/gofiber/fiber/v3"
)

func uploadedFiles(c fiber.Ctx, field string) ([]*multipart.FileHeader, error) {
	if !strings.HasPrefix(strings.ToLower(c.Get("Content-Type")), "multipart/form-data") {
		return nil, nil
	}
	form, err := c.MultipartForm()
	if err != nil {
		return nil, errors.ErrBadRequest
	}
	return form.File[field], nil
}

func submissionFile(c fiber.Ctx) (*multipart.FileHeader, error) {
	files, err := uploadedFiles(c, "file")
	if err != nil {
		return nil, err
	}
	if len(files) > 1 {
		return nil, errors.ErrBadRequest
	}
	if len(files) == 0 {
		return nil, nil
	}
	return files[0], nil
}
