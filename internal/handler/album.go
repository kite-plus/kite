package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kite-plus/kite/internal/i18n"
	"github.com/kite-plus/kite/internal/middleware"
	"github.com/kite-plus/kite/internal/model"
	"github.com/kite-plus/kite/internal/repo"
)

// AlbumHandler handles album (folder) management HTTP requests.
type AlbumHandler struct {
	albumRepo *repo.AlbumRepo
	fileRepo  *repo.FileRepo
}

func NewAlbumHandler(albumRepo *repo.AlbumRepo, fileRepo *repo.FileRepo) *AlbumHandler {
	return &AlbumHandler{albumRepo: albumRepo, fileRepo: fileRepo}
}

type createAlbumRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Description string `json:"description" binding:"max=500"`
	IsPublic    bool   `json:"is_public"`
	ParentID    string `json:"parent_id"`
}

type albumListData struct {
	Items         []model.Album `json:"items"`
	Total         int64         `json:"total"`
	Page          int           `json:"page"`
	Size          int           `json:"size"`
	CurrentFolder *model.Album  `json:"current_folder,omitempty"`
	Ancestors     []model.Album `json:"ancestors,omitempty"`
}

// Create creates a new album.
func (h *AlbumHandler) Create(c *gin.Context) {
	var req createAlbumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeyAlbumInvalidData, err.Error()))
		return
	}

	userID := c.GetString(middleware.ContextKeyUserID)

	album := &model.Album{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		IsPublic:    req.IsPublic,
	}
	if req.ParentID != "" {
		parent, err := h.albumRepo.GetByID(c.Request.Context(), req.ParentID)
		if err != nil || parent.UserID != userID {
			BadRequest(c, M(c, i18n.KeyAlbumInvalidParentFolder))
			return
		}
		album.ParentID = &req.ParentID
	}

	if err := h.albumRepo.Create(c.Request.Context(), album); err != nil {
		ServerError(c, M(c, i18n.KeyAlbumCreateFailed))
		return
	}

	Created(c, album)
}

// List returns the current user's folders under the given parent directory.
func (h *AlbumHandler) List(c *gin.Context) {
	userID := c.GetString(middleware.ContextKeyUserID)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	parentID := c.Query("parent_id")
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	var parentIDPtr *string
	var currentFolder *model.Album
	var ancestors []model.Album
	if parentID != "" {
		parentIDPtr = &parentID
		folder, err := h.albumRepo.GetByID(c.Request.Context(), parentID)
		if err != nil || folder.UserID != userID {
			NotFound(c, M(c, i18n.KeyAlbumFolderNotFound))
			return
		}
		currentFolder = folder
		ancestors, err = h.albumRepo.ListAncestors(c.Request.Context(), userID, parentID)
		if err != nil {
			ServerError(c, M(c, i18n.KeyAlbumLoadFolderFailed))
			return
		}
	}

	albums, total, err := h.albumRepo.ListByUser(c.Request.Context(), userID, parentIDPtr, page, size)
	if err != nil {
		ServerError(c, M(c, i18n.KeyAlbumListFailed))
		return
	}

	// Populate per-folder counts of child folders and files.
	for i := range albums {
		count, _ := h.fileRepo.CountByAlbum(c.Request.Context(), albums[i].ID)
		albums[i].FileCount = count
		folderCount, _ := h.albumRepo.CountChildren(c.Request.Context(), albums[i].ID)
		albums[i].FolderCount = folderCount
	}

	Success(c, albumListData{
		Items:         albums,
		Total:         total,
		Page:          page,
		Size:          size,
		CurrentFolder: currentFolder,
		Ancestors:     ancestors,
	})
}

type updateAlbumRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	IsPublic    *bool   `json:"is_public"`
	ParentID    *string `json:"parent_id"`
}

// Update modifies an album.
func (h *AlbumHandler) Update(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)

	album, err := h.albumRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		NotFound(c, M(c, i18n.KeyAlbumNotFound))
		return
	}

	if album.UserID != userID {
		Forbidden(c, M(c, i18n.KeyAlbumNotOwner))
		return
	}

	var req updateAlbumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, M(c, i18n.KeyAlbumInvalidData, err.Error()))
		return
	}

	if req.Name != nil {
		album.Name = *req.Name
	}
	if req.Description != nil {
		album.Description = *req.Description
	}
	if req.IsPublic != nil {
		album.IsPublic = *req.IsPublic
	}
	if req.ParentID != nil {
		if *req.ParentID == "" {
			album.ParentID = nil
		} else {
			if *req.ParentID == album.ID {
				BadRequest(c, M(c, i18n.KeyAlbumFolderSelfParent))
				return
			}
			parent, err := h.albumRepo.GetByID(c.Request.Context(), *req.ParentID)
			if err != nil || parent.UserID != userID {
				BadRequest(c, M(c, i18n.KeyAlbumInvalidParentFolder))
				return
			}
			ancestors, err := h.albumRepo.ListAncestors(c.Request.Context(), userID, *req.ParentID)
			if err != nil {
				BadRequest(c, M(c, i18n.KeyAlbumInvalidParentFolder))
				return
			}
			for _, ancestor := range ancestors {
				if ancestor.ID == album.ID {
					BadRequest(c, M(c, i18n.KeyAlbumDescendantMove))
					return
				}
			}
			album.ParentID = req.ParentID
		}
	}

	if err := h.albumRepo.Update(c.Request.Context(), album); err != nil {
		ServerError(c, M(c, i18n.KeyAlbumUpdateFailed))
		return
	}

	Success(c, album)
}

// Delete removes a folder.
func (h *AlbumHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString(middleware.ContextKeyUserID)

	album, err := h.albumRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		NotFound(c, M(c, i18n.KeyAlbumNotFound))
		return
	}

	if album.UserID != userID {
		Forbidden(c, M(c, i18n.KeyAlbumNotOwner))
		return
	}

	if err := h.albumRepo.Delete(c.Request.Context(), id); err != nil {
		ServerError(c, M(c, i18n.KeyAlbumDeleteFailed))
		return
	}

	Success(c, nil)
}
