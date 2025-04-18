package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/99designs/gqlgen/graphql"
)

type FormField struct {
	Name  string
	Value any
}

func multipartRequest(ctx context.Context, endpoint, operationName, query string, variables map[string]any) (*http.Request, error) {
	multipartFilesGroups, mapping, variables := parseMultipartFiles(variables)
	r := &Request{
		Query:         query,
		Variables:     variables,
		OperationName: operationName,
	}

	body := new(bytes.Buffer)
	formFields := []FormField{
		{
			Name:  "operations",
			Value: r,
		},
		{
			Name:  "map",
			Value: mapping,
		},
	}
	contentType, err := prepareMultipartFormBody(body, formFields, multipartFilesGroups)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare form body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create request struct failed: %w", err)
	}

	req.Header = http.Header{"Content-Type": []string{contentType}}

	return req, nil
}

type MultipartFile struct {
	File  graphql.Upload
	Index int
}

type MultipartFilesGroup struct {
	Files      []MultipartFile
	IsMultiple bool
}

func parseMultipartFiles(vars map[string]any) ([]MultipartFilesGroup, map[string][]string, map[string]any) {
	var (
		multipartFilesGroups []MultipartFilesGroup
		mapping              = map[string][]string{}
		i                    = 0
	)

	for k, v := range vars {
		switch item := v.(type) {
		case graphql.Upload:
			iStr := strconv.Itoa(i)
			vars[k] = nil
			mapping[iStr] = []string{fmt.Sprintf("variables.%s", k)}

			multipartFilesGroups = append(multipartFilesGroups, MultipartFilesGroup{
				Files: []MultipartFile{
					{
						Index: i,
						File:  item,
					},
				},
			})

			i++
		case *graphql.Upload:
			// continue if it is empty
			if item == nil {
				continue
			}

			iStr := strconv.Itoa(i)
			vars[k] = nil
			mapping[iStr] = []string{fmt.Sprintf("variables.%s", k)}

			multipartFilesGroups = append(multipartFilesGroups, MultipartFilesGroup{
				Files: []MultipartFile{
					{
						Index: i,
						File:  *item,
					},
				},
			})

			i++
		case []*graphql.Upload:
			vars[k] = make([]struct{}, len(item))
			var groupFiles []MultipartFile

			for itemI, itemV := range item {
				iStr := strconv.Itoa(i)
				mapping[iStr] = []string{fmt.Sprintf("variables.%s.%s", k, strconv.Itoa(itemI))}

				groupFiles = append(groupFiles, MultipartFile{
					Index: i,
					File:  *itemV,
				})

				i++
			}

			multipartFilesGroups = append(multipartFilesGroups, MultipartFilesGroup{
				Files:      groupFiles,
				IsMultiple: true,
			})
		}
	}

	return multipartFilesGroups, mapping, vars
}

func prepareMultipartFormBody(buffer *bytes.Buffer, formFields []FormField, files []MultipartFilesGroup) (string, error) {
	writer := multipart.NewWriter(buffer)
	defer writer.Close()

	// form fields
	for _, field := range formFields {
		fieldBody, err := json.Marshal(field.Value)
		if err != nil {
			return "", fmt.Errorf("encode %s: %w", field.Name, err)
		}

		err = writer.WriteField(field.Name, string(fieldBody))
		if err != nil {
			return "", fmt.Errorf("write %s: %w", field.Name, err)
		}
	}

	// files
	for _, filesGroup := range files {
		for _, file := range filesGroup.Files {
			part, err := writer.CreateFormFile(strconv.Itoa(file.Index), file.File.Filename)
			if err != nil {
				return "", fmt.Errorf("form file %w", err)
			}

			_, err = io.Copy(part, file.File.File)
			if err != nil {
				return "", fmt.Errorf("copy file %w", err)
			}
		}
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("writer close %w", err)
	}

	return writer.FormDataContentType(), nil
}
