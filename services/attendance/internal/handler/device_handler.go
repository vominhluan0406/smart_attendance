package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/attendance-service/internal/repository"
	"github.com/smart-attendance/shared/response"
)

type DeviceHandler struct {
	deviceRepo *repository.UserDeviceRepository
}

func NewDeviceHandler(deviceRepo *repository.UserDeviceRepository) *DeviceHandler {
	return &DeviceHandler{deviceRepo: deviceRepo}
}

// ListDevices handles GET /api/users/{id}/devices — list a user's known devices
func (h *DeviceHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	devices, err := h.deviceRepo.ListByUserID(userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, devices)
}

// BlockDevice handles POST /api/users/{id}/devices/{deviceId}/block
func (h *DeviceHandler) BlockDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	device, err := h.deviceRepo.FindByID(deviceID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Không tìm thấy thiết bị")
		return
	}
	device.IsBlocked = true
	device.IsTrusted = false
	if err := h.deviceRepo.Update(device); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "Đã chặn thiết bị"})
}

// UnblockDevice handles POST /api/users/{id}/devices/{deviceId}/unblock
func (h *DeviceHandler) UnblockDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	device, err := h.deviceRepo.FindByID(deviceID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Không tìm thấy thiết bị")
		return
	}
	device.IsBlocked = false
	device.IsTrusted = true
	if err := h.deviceRepo.Update(device); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "Đã bỏ chặn thiết bị"})
}

// DeleteDevice handles DELETE /api/users/{id}/devices/{deviceId}
func (h *DeviceHandler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceId")
	if err := h.deviceRepo.Delete(deviceID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "Đã xoá thiết bị"})
}
