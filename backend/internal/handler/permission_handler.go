package handler

import (
	"EffiPlat/backend/internal/model"
	"EffiPlat/backend/internal/service"
	"EffiPlat/backend/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PermissionHandler struct {
	permissionService service.PermissionService
	auditService      service.AuditLogService
	logger            *zap.Logger
}

func NewPermissionHandler(ps service.PermissionService, auditSvc service.AuditLogService, logger *zap.Logger) *PermissionHandler {
	return &PermissionHandler{
		permissionService: ps,
		auditService:      auditSvc,
		logger:            logger,
	}
}

// CreatePermission godoc
// @Summary Create a new permission
// @Description Create a new permission with name, resource, and action
// @Tags permissions
// @Accept  json
// @Produce  json
// @Param   permission body model.CreatePermissionRequest true "Permission object"
// @Success 201 {object} map[string]interface{} "Unified response: {code, message, data: model.Permission}"
// @Failure 400 {object} map[string]interface{} "Bad Request (e.g. validation error)"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /permissions [post]
func (h *PermissionHandler) CreatePermission(c *gin.Context) {
	var req model.CreatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("CreatePermission: Failed to bind JSON", zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	permissionToCreate := model.Permission{
		Name:        req.Name,
		Description: req.Description,
		Resource:    req.Resource,
		Action:      req.Action,
	}

	createdPermission, err := h.permissionService.CreatePermission(c.Request.Context(), &permissionToCreate)
	if err != nil {
		h.logger.Error("CreatePermission: Service error", zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, "Failed to create permission")
		return
	}

	// 记录审计日志
	details := map[string]interface{}{
		"id":          createdPermission.ID,
		"name":        createdPermission.Name,
		"description": createdPermission.Description,
		"resource":    createdPermission.Resource,
		"action":      createdPermission.Action,
	}
	_ = h.auditService.LogUserAction(c, string(utils.AuditActionCreate), "PERMISSION", createdPermission.ID, details)

	RespondWithSuccess(c, http.StatusCreated, "Permission created successfully", createdPermission)
}

// GetPermissions godoc
// @Summary Get all permissions
// @Description Retrieve a list of all permissions with pagination and search
// @Tags permissions
// @Produce  json
// @Param page query int false "Page number (default: 1)"
// @Param pageSize query int false "Number of items per page (default: 10)"
// @Param name query string false "Search by permission name (fuzzy)"
// @Param resource query string false "Filter by resource"
// @Param action query string false "Filter by action"
// @Success 200 {object} map[string]interface{} "Unified response: {code, message, data: {items: []model.Permission, total: int, page: int, pageSize: int}}"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /permissions [get]
func (h *PermissionHandler) GetPermissions(c *gin.Context) {
	pageQuery := c.DefaultQuery("page", "1")
	pageSizeQuery := c.DefaultQuery("pageSize", "10")
	nameQuery := c.Query("name")
	resourceQuery := c.Query("resource")
	actionQuery := c.Query("action")

	page, err := strconv.Atoi(pageQuery)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeQuery)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	params := model.PermissionListParams{
		Page:     page,
		PageSize: pageSize,
		Name:     nameQuery,
		Resource: resourceQuery,
		Action:   actionQuery,
	}

	permissions, total, err := h.permissionService.GetPermissions(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("GetPermissions: Service error", zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, "Failed to retrieve permissions")
		return
	}

	RespondWithSuccess(c, http.StatusOK, "Permissions retrieved successfully", gin.H{
		"items":    permissions,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetPermissionByID godoc
// @Summary Get a permission by ID
// @Description Retrieve a specific permission by its ID
// @Tags permissions
// @Produce  json
// @Param   permissionId path int true "Permission ID"
// @Success 200 {object} map[string]interface{} "Unified response: {code, message, data: model.Permission}"
// @Failure 400 {object} map[string]interface{} "Bad Request (Invalid ID)"
// @Failure 404 {object} map[string]interface{} "Not Found (Permission not found)"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /permissions/{permissionId} [get]
func (h *PermissionHandler) GetPermissionByID(c *gin.Context) {
	permissionIDStr := c.Param("permissionId")
	permissionID, err := strconv.ParseUint(permissionIDStr, 10, 32)
	if err != nil {
		h.logger.Error("GetPermissionByID: Invalid permission ID format", zap.String("permissionId", permissionIDStr), zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, "Invalid permission ID format")
		return
	}

	permission, err := h.permissionService.GetPermissionByID(c.Request.Context(), uint(permissionID))
	if err != nil {
		h.logger.Error("GetPermissionByID: Service error", zap.Uint("permissionId", uint(permissionID)), zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, "Failed to retrieve permission")
		return
	}

	RespondWithSuccess(c, http.StatusOK, "Permission retrieved successfully", permission)
}

// UpdatePermission godoc
// @Summary Update an existing permission
// @Description Update an existing permission by its ID
// @Tags permissions
// @Accept  json
// @Produce  json
// @Param   permissionId path int true "Permission ID"
// @Param   permission body model.UpdatePermissionRequest true "Permission object with updated fields"
// @Success 200 {object} map[string]interface{} "Unified response: {code, message, data: model.Permission}"
// @Failure 400 {object} map[string]interface{} "Bad Request (Invalid ID, validation error)"
// @Failure 404 {object} map[string]interface{} "Not Found (Permission not found)"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /permissions/{permissionId} [put]
func (h *PermissionHandler) UpdatePermission(c *gin.Context) {
	permissionIDStr := c.Param("permissionId")
	permissionID, err := strconv.ParseUint(permissionIDStr, 10, 32)
	if err != nil {
		h.logger.Error("UpdatePermission: Invalid permission ID format", zap.String("permissionId", permissionIDStr), zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, "Invalid permission ID format")
		return
	}
	
	// 获取原始权限数据用于审计日志
	origPermission, getErr := h.permissionService.GetPermissionByID(c.Request.Context(), uint(permissionID))
	if getErr != nil {
		h.logger.Warn("Could not get original permission data for audit logging",
			zap.Uint64("permissionID", permissionID), zap.Error(getErr))
	}

	var req model.UpdatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("UpdatePermission: Failed to bind JSON", zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	permissionToUpdate := model.Permission{}
	if req.Name != nil {
		permissionToUpdate.Name = *req.Name
	}
	if req.Description != nil {
		permissionToUpdate.Description = *req.Description
	}
	if req.Resource != nil {
		permissionToUpdate.Resource = *req.Resource
	}
	if req.Action != nil {
		permissionToUpdate.Action = *req.Action
	}

	updatedPermission, err := h.permissionService.UpdatePermission(c.Request.Context(), uint(permissionID), &permissionToUpdate)
	if err != nil {
		h.logger.Error("UpdatePermission: Service error", zap.Uint("permissionId", uint(permissionID)), zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, "Failed to update permission")
		return
	}

	// 记录审计日志
	details := map[string]interface{}{
		"before": origPermission,
		"after":  updatedPermission,
		"changes": req,
	}
	_ = h.auditService.LogUserAction(c, string(utils.AuditActionUpdate), "PERMISSION", updatedPermission.ID, details)

	RespondWithSuccess(c, http.StatusOK, "Permission updated successfully", updatedPermission)
}

// DeletePermission godoc
// @Summary Delete a permission
// @Description Delete a permission by its ID
// @Tags permissions
// @Produce  json
// @Param   permissionId path int true "Permission ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]interface{} "Bad Request (Invalid ID)"
// @Failure 404 {object} map[string]interface{} "Not Found (Permission not found)"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /permissions/{permissionId} [delete]
func (h *PermissionHandler) DeletePermission(c *gin.Context) {
	permissionIDStr := c.Param("permissionId")
	permissionID, err := strconv.ParseUint(permissionIDStr, 10, 32)
	if err != nil {
		h.logger.Error("DeletePermission: Invalid permission ID format", zap.String("permissionId", permissionIDStr), zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, "Invalid permission ID format")
		return
	}
	
	// 获取要删除的权限数据用于审计日志
	permission, getErr := h.permissionService.GetPermissionByID(c.Request.Context(), uint(permissionID))
	if getErr != nil {
		h.logger.Warn("Could not get permission data for audit logging before deletion",
			zap.Uint64("permissionID", permissionID), zap.Error(getErr))
	}

	err = h.permissionService.DeletePermission(c.Request.Context(), uint(permissionID))
	if err != nil {
		h.logger.Error("DeletePermission: Service error", zap.Uint("permissionID", uint(permissionID)), zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, "Failed to delete permission")
		return
	}

	// 记录审计日志
	if permission != nil {
		details := map[string]interface{}{
			"deletedPermission": map[string]interface{}{
				"id":          permission.ID,
				"name":        permission.Name,
				"description": permission.Description,
				"resource":    permission.Resource,
				"action":      permission.Action,
			},
		}
		_ = h.auditService.LogUserAction(c, string(utils.AuditActionDelete), "PERMISSION", uint(permissionID), details)
	}

	c.Status(http.StatusNoContent)
}

// AddPermissionsToRole godoc
// @Summary Add permissions to a role
// @Description Add one or more permissions to a specific role
// @Tags roles permissions
// @Accept  json
// @Produce  json
// @Param   roleId path int true "Role ID"
// @Param   permissionIds body []uint true "List of permission IDs to add"
// @Success 200 {object} map[string]interface{} "Unified response: {code, message, data: null}"
// @Failure 400 {object} map[string]interface{} "Bad Request (Invalid IDs, role/permission not found)"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /roles/{roleId}/permissions [post]
func (h *PermissionHandler) AddPermissionsToRole(c *gin.Context) {
	roleIDStr := c.Param("roleId")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		h.logger.Error("AddPermissionsToRole: Invalid role ID format", zap.String("roleId", roleIDStr), zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, "Invalid role ID format")
		return
	}

	var permissionIDs []uint
	if err := c.ShouldBindJSON(&permissionIDs); err != nil {
		h.logger.Error("AddPermissionsToRole: Failed to bind JSON", zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	if len(permissionIDs) == 0 {
		RespondWithError(c, http.StatusBadRequest, "Permission IDs list cannot be empty")
		return
	}

	// 获取角色原始权限数据用于审计日志
	origPermissions, getErr := h.permissionService.GetPermissionsByRoleID(c.Request.Context(), uint(roleID))
	if getErr != nil {
		h.logger.Warn("Could not get original permissions for role for audit logging",
			zap.Uint64("roleID", roleID), zap.Error(getErr))
		origPermissions = []model.Permission{}
	}

	err = h.permissionService.AddPermissionsToRole(c.Request.Context(), uint(roleID), permissionIDs)
	if err != nil {
		h.logger.Error("AddPermissionsToRole: Service error", zap.Uint("roleId", uint(roleID)), zap.Any("permissionIDs", permissionIDs), zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, "Failed to add permissions to role")
		return
	}

	// 记录审计日志
	details := map[string]interface{}{
		"roleId":           roleID,
		"addedPermissions": permissionIDs,
		"originalPermissions": origPermissions,
	}
	
	_ = h.auditService.LogUserAction(c, string(utils.AuditActionUpdate), "ROLE_PERMISSIONS", uint(roleID), details)

	RespondWithSuccess(c, http.StatusOK, "Permissions added to role successfully", nil)
}

// RemovePermissionsFromRole godoc
// @Summary Remove permissions from a role
// @Description Remove one or more permissions from a specific role
// @Tags roles permissions
// @Accept  json
// @Produce  json
// @Param   roleId path int true "Role ID"
// @Param   permissionIds body []uint true "List of permission IDs to remove"
// @Success 200 {object} map[string]interface{} "Unified response: {code, message, data: null}"
// @Failure 400 {object} map[string]interface{} "Bad Request (Invalid IDs, role/permission not found)"
// @Failure 500 {object} map[string]interface{} "Internal Server Error"
// @Router /roles/{roleId}/permissions [delete]
func (h *PermissionHandler) RemovePermissionsFromRole(c *gin.Context) {
	roleIDStr := c.Param("roleId")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		h.logger.Error("RemovePermissionsFromRole: Invalid role ID format", zap.String("roleId", roleIDStr), zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, "Invalid role ID format")
		return
	}

	var permissionIDs []uint
	if err := c.ShouldBindJSON(&permissionIDs); err != nil {
		h.logger.Error("RemovePermissionsFromRole: Failed to bind JSON", zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	if len(permissionIDs) == 0 {
		RespondWithError(c, http.StatusBadRequest, "Permission IDs list cannot be empty")
		return
	}

	// 获取角色原始权限数据用于审计日志
	origPermissions, getErr := h.permissionService.GetPermissionsByRoleID(c.Request.Context(), uint(roleID))
	if getErr != nil {
		h.logger.Warn("Could not get original permissions for role for audit logging",
			zap.Uint64("roleID", roleID), zap.Error(getErr))
		origPermissions = []model.Permission{}
	}

	err = h.permissionService.RemovePermissionsFromRole(c.Request.Context(), uint(roleID), permissionIDs)
	if err != nil {
		h.logger.Error("RemovePermissionsFromRole: Service error", zap.Uint("roleId", uint(roleID)), zap.Any("permissionIDs", permissionIDs), zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, "Failed to remove permissions from role")
		return
	}

	// 记录审计日志
	details := map[string]interface{}{
		"roleId":             roleID,
		"removedPermissions": permissionIDs,
		"originalPermissions": origPermissions,
	}
	
	_ = h.auditService.LogUserAction(c, string(utils.AuditActionUpdate), "ROLE_PERMISSIONS", uint(roleID), details)

	RespondWithSuccess(c, http.StatusOK, "Permissions removed from role successfully", nil)
}

func (h *PermissionHandler) GetPermissionsByRoleID(c *gin.Context) {
	roleIDStr := c.Param("roleId")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		h.logger.Error("GetPermissionsByRoleID: Invalid role ID format", zap.String("roleId", roleIDStr), zap.Error(err))
		RespondWithError(c, http.StatusBadRequest, "Invalid role ID format")
		return
	}

	permissions, err := h.permissionService.GetPermissionsByRoleID(c.Request.Context(), uint(roleID))
	if err != nil {
		h.logger.Error("GetPermissionsByRoleID: Service error", zap.Uint("roleId", uint(roleID)), zap.Error(err))
		RespondWithError(c, http.StatusInternalServerError, "Failed to get permissions for role")
		return
	}

	RespondWithSuccess(c, http.StatusOK, "Permissions for role retrieved successfully", permissions)
}
