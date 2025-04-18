// Code generated by github.com/Yamashou/gqlgenc, DO NOT EDIT.

package gen

import (
	"context"
	"net/http"

	"github.com/99designs/gqlgen/graphql"
	"github.com/Yamashou/gqlgenc/v3/client"
)

type Client struct {
	Client *client.Client
}

func NewClient(cli *http.Client, baseURL string, option *client.Options, interceptors ...client.RequestInterceptor) *Client {
	return &Client{Client: client.NewClient(cli, baseURL, option, interceptors...)}
}

type Query struct {
	FileInfo     FileInfoResult  "json:\"fileInfo\" graphql:\"fileInfo\""
	FilesInfo    FilesInfoResult "json:\"filesInfo\" graphql:\"filesInfo\""
	GetListItems ListItemsResult "json:\"getListItems\" graphql:\"getListItems\""
}

type Mutation struct {
	UploadFile  UploadFileResult  "json:\"uploadFile\" graphql:\"uploadFile\""
	UploadFiles UploadFilesResult "json:\"uploadFiles\" graphql:\"uploadFiles\""
	AddListItem AddListItemResult "json:\"addListItem\" graphql:\"addListItem\""
}

type FileDataFragment struct {
	Mime string "json:\"mime\" graphql:\"mime\""
	Name string "json:\"name\" graphql:\"name\""
	Size string "json:\"size\" graphql:\"size\""
}

type UploadFile_UploadFile struct {
	Typename *string           "json:\"__typename\" graphql:\"__typename\""
	File     *FileDataFragment "json:\"file\" graphql:\"file\""
}

type UploadFiles_UploadFiles_UploadFilesResult struct {
	Files []*FileDataFragment "json:\"files\" graphql:\"files\""
}

type UploadFiles_UploadFiles struct {
	Typename          *string                                   "json:\"__typename\" graphql:\"__typename\""
	UploadFilesResult UploadFiles_UploadFiles_UploadFilesResult "graphql:\"... on UploadFilesResult\""
}

type ManyMutationsInOne_UploadFile_File struct {
	Mime string "json:\"mime\" graphql:\"mime\""
	Name string "json:\"name\" graphql:\"name\""
	Size string "json:\"size\" graphql:\"size\""
}

type ManyMutationsInOne_UploadFile struct {
	Typename *string                            "json:\"__typename\" graphql:\"__typename\""
	File     ManyMutationsInOne_UploadFile_File "json:\"file\" graphql:\"file\""
}

type ManyMutationsInOne_UploadFiles_UploadFilesResult_Files struct {
	Mime string "json:\"mime\" graphql:\"mime\""
	Name string "json:\"name\" graphql:\"name\""
	Size string "json:\"size\" graphql:\"size\""
}

type ManyMutationsInOne_UploadFiles_UploadFilesResult struct {
	Files []*ManyMutationsInOne_UploadFiles_UploadFilesResult_Files "json:\"files\" graphql:\"files\""
}

type ManyMutationsInOne_AddListItem_AddListItemResult_Item struct {
	ID   int    "json:\"id\" graphql:\"id\""
	Name string "json:\"name\" graphql:\"name\""
}

type ManyMutationsInOne_AddListItem_AddListItemResult struct {
	Item ManyMutationsInOne_AddListItem_AddListItemResult_Item "json:\"item\" graphql:\"item\""
}

type ManyMutationsInOne_AddListItem struct {
	Typename          *string                                          "json:\"__typename\" graphql:\"__typename\""
	AddListItemResult ManyMutationsInOne_AddListItem_AddListItemResult "graphql:\"... on AddListItemResult\""
}

type FileInfo_FileInfo struct {
	Typename *string           "json:\"__typename\" graphql:\"__typename\""
	File     *FileDataFragment "json:\"file\" graphql:\"file\""
}

type FilesInfo_FilesInfo_FilesInfoResult struct {
	Files []*FileDataFragment "json:\"files\" graphql:\"files\""
}

type FilesInfo_FilesInfo struct {
	Typename        *string                             "json:\"__typename\" graphql:\"__typename\""
	FilesInfoResult FilesInfo_FilesInfo_FilesInfoResult "graphql:\"... on FilesInfoResult\""
}

type AllFilesInfo_Result struct {
	Typename *string           "json:\"__typename\" graphql:\"__typename\""
	File     *FileDataFragment "json:\"file\" graphql:\"file\""
}

type AllFilesInfo_Result2_FilesInfoResult struct {
	Files []*FileDataFragment "json:\"files\" graphql:\"files\""
}

type AllFilesInfoWithListItems_FileInfo_File struct {
	Mime string "json:\"mime\" graphql:\"mime\""
	Name string "json:\"name\" graphql:\"name\""
	Size string "json:\"size\" graphql:\"size\""
}

type AllFilesInfoWithListItems_FileInfo struct {
	Typename *string                                 "json:\"__typename\" graphql:\"__typename\""
	File     AllFilesInfoWithListItems_FileInfo_File "json:\"file\" graphql:\"file\""
}

type AllFilesInfoWithListItems_FilesInfo_FilesInfoResult_Files struct {
	Mime string "json:\"mime\" graphql:\"mime\""
	Name string "json:\"name\" graphql:\"name\""
	Size string "json:\"size\" graphql:\"size\""
}

type AllFilesInfoWithListItems_FilesInfo_FilesInfoResult struct {
	Files []*AllFilesInfoWithListItems_FilesInfo_FilesInfoResult_Files "json:\"files\" graphql:\"files\""
}

type AllFilesInfoWithListItems_GetListItems_ListItemsResult_Items struct {
	ID   int    "json:\"id\" graphql:\"id\""
	Name string "json:\"name\" graphql:\"name\""
}

type AllFilesInfoWithListItems_GetListItems_ListItemsResult struct {
	Items []*AllFilesInfoWithListItems_GetListItems_ListItemsResult_Items "json:\"items\" graphql:\"items\""
}

type AllFilesInfoWithListItems_GetListItems struct {
	Typename        *string                                                "json:\"__typename\" graphql:\"__typename\""
	ListItemsResult AllFilesInfoWithListItems_GetListItems_ListItemsResult "graphql:\"... on ListItemsResult\""
}

type UploadFile struct {
	UploadFile UploadFile_UploadFile "json:\"uploadFile\" graphql:\"uploadFile\""
}

type UploadFiles struct {
	UploadFiles UploadFiles_UploadFiles "json:\"uploadFiles\" graphql:\"uploadFiles\""
}

type ManyMutationsInOne struct {
	UploadFile  ManyMutationsInOne_UploadFile                    "json:\"uploadFile\" graphql:\"uploadFile\""
	UploadFiles ManyMutationsInOne_UploadFiles_UploadFilesResult "json:\"uploadFiles\" graphql:\"uploadFiles\""
	AddListItem ManyMutationsInOne_AddListItem                   "json:\"addListItem\" graphql:\"addListItem\""
}

type FileInfo struct {
	FileInfo FileInfo_FileInfo "json:\"fileInfo\" graphql:\"fileInfo\""
}

type FilesInfo struct {
	FilesInfo FilesInfo_FilesInfo "json:\"filesInfo\" graphql:\"filesInfo\""
}

type AllFilesInfo struct {
	Result  AllFilesInfo_Result                  "json:\"result\" graphql:\"result\""
	Result2 AllFilesInfo_Result2_FilesInfoResult "json:\"result2\" graphql:\"result2\""
}

type AllFilesInfoWithListItems struct {
	FileInfo     AllFilesInfoWithListItems_FileInfo                  "json:\"fileInfo\" graphql:\"fileInfo\""
	FilesInfo    AllFilesInfoWithListItems_FilesInfo_FilesInfoResult "json:\"filesInfo\" graphql:\"filesInfo\""
	GetListItems AllFilesInfoWithListItems_GetListItems              "json:\"getListItems\" graphql:\"getListItems\""
}

const UploadFileDocument = `mutation UploadFile ($file: Upload!) {
	uploadFile(file: $file) {
		__typename
		file {
			... FileDataFragment
		}
	}
}
fragment FileDataFragment on FileData {
	mime
	name
	size
}
`

func (c *Client) UploadFile(ctx context.Context, file graphql.Upload, interceptors ...client.RequestInterceptor) (*UploadFile, error) {
	vars := map[string]any{
		"file": file,
	}

	var res UploadFile
	if err := c.Client.Post(ctx, "UploadFile", UploadFileDocument, &res, vars, interceptors...); err != nil {
		return nil, err
	}

	return &res, nil
}

const UploadFilesDocument = `mutation UploadFiles ($files: [Upload!]!) {
	uploadFiles(files: $files) {
		__typename
		... on UploadFilesResult {
			files {
				... FileDataFragment
			}
		}
	}
}
fragment FileDataFragment on FileData {
	mime
	name
	size
}
`

func (c *Client) UploadFiles(ctx context.Context, files []*graphql.Upload, interceptors ...client.RequestInterceptor) (*UploadFiles, error) {
	vars := map[string]any{
		"files": files,
	}

	var res UploadFiles
	if err := c.Client.Post(ctx, "UploadFiles", UploadFilesDocument, &res, vars, interceptors...); err != nil {
		return nil, err
	}

	return &res, nil
}

const ManyMutationsInOneDocument = `mutation ManyMutationsInOne ($file: Upload!, $files: [Upload!]!, $name: String!) {
	uploadFile(file: $file) {
		__typename
		file {
			mime
			name
			size
		}
	}
	uploadFiles(files: $files) {
		... on UploadFilesResult {
			files {
				mime
				name
				size
			}
		}
	}
	addListItem(name: $name) {
		__typename
		... on AddListItemResult {
			item {
				id
				name
			}
		}
	}
}
`

func (c *Client) ManyMutationsInOne(ctx context.Context, file graphql.Upload, files []*graphql.Upload, name string, interceptors ...client.RequestInterceptor) (*ManyMutationsInOne, error) {
	vars := map[string]any{
		"file":  file,
		"files": files,
		"name":  name,
	}

	var res ManyMutationsInOne
	if err := c.Client.Post(ctx, "ManyMutationsInOne", ManyMutationsInOneDocument, &res, vars, interceptors...); err != nil {
		return nil, err
	}

	return &res, nil
}

const FileInfoDocument = `query FileInfo ($file: Upload!) {
	fileInfo(file: $file) {
		__typename
		file {
			... FileDataFragment
		}
	}
}
fragment FileDataFragment on FileData {
	mime
	name
	size
}
`

func (c *Client) FileInfo(ctx context.Context, file graphql.Upload, interceptors ...client.RequestInterceptor) (*FileInfo, error) {
	vars := map[string]any{
		"file": file,
	}

	var res FileInfo
	if err := c.Client.Post(ctx, "FileInfo", FileInfoDocument, &res, vars, interceptors...); err != nil {
		return nil, err
	}

	return &res, nil
}

const FilesInfoDocument = `query FilesInfo ($files: [Upload!]!) {
	filesInfo(files: $files) {
		__typename
		... on FilesInfoResult {
			files {
				... FileDataFragment
			}
		}
	}
}
fragment FileDataFragment on FileData {
	mime
	name
	size
}
`

func (c *Client) FilesInfo(ctx context.Context, files []*graphql.Upload, interceptors ...client.RequestInterceptor) (*FilesInfo, error) {
	vars := map[string]any{
		"files": files,
	}

	var res FilesInfo
	if err := c.Client.Post(ctx, "FilesInfo", FilesInfoDocument, &res, vars, interceptors...); err != nil {
		return nil, err
	}

	return &res, nil
}

const AllFilesInfoDocument = `query AllFilesInfo ($file: Upload!, $files: [Upload!]!) {
	result: fileInfo(file: $file) {
		__typename
		file {
			... FileDataFragment
		}
	}
	result2: filesInfo(files: $files) {
		... on FilesInfoResult {
			files {
				... FileDataFragment
			}
		}
	}
}
fragment FileDataFragment on FileData {
	mime
	name
	size
}
`

func (c *Client) AllFilesInfo(ctx context.Context, file graphql.Upload, files []*graphql.Upload, interceptors ...client.RequestInterceptor) (*AllFilesInfo, error) {
	vars := map[string]any{
		"file":  file,
		"files": files,
	}

	var res AllFilesInfo
	if err := c.Client.Post(ctx, "AllFilesInfo", AllFilesInfoDocument, &res, vars, interceptors...); err != nil {
		return nil, err
	}

	return &res, nil
}

const AllFilesInfoWithListItemsDocument = `query AllFilesInfoWithListItems ($file: Upload!, $files: [Upload!]!, $input: ListItemsInput) {
	fileInfo(file: $file) {
		__typename
		file {
			mime
			name
			size
		}
	}
	filesInfo(files: $files) {
		... on FilesInfoResult {
			files {
				mime
				name
				size
			}
		}
	}
	getListItems(input: $input) {
		__typename
		... on ListItemsResult {
			items {
				id
				name
			}
		}
	}
}
`

func (c *Client) AllFilesInfoWithListItems(ctx context.Context, file graphql.Upload, files []*graphql.Upload, input *ListItemsInput, interceptors ...client.RequestInterceptor) (*AllFilesInfoWithListItems, error) {
	vars := map[string]any{
		"file":  file,
		"files": files,
		"input": input,
	}

	var res AllFilesInfoWithListItems
	if err := c.Client.Post(ctx, "AllFilesInfoWithListItems", AllFilesInfoWithListItemsDocument, &res, vars, interceptors...); err != nil {
		return nil, err
	}

	return &res, nil
}
