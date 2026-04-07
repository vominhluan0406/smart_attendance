package service

import (
	"fmt"
	"log"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/repository"
)

type FraudAlertService struct {
	alertRepo *repository.FraudAlertRepository
}

func NewFraudAlertService(alertRepo *repository.FraudAlertRepository) *FraudAlertService {
	return &FraudAlertService{alertRepo: alertRepo}
}

// GetBranchAlerts returns paginated fraud alerts for a branch with filters.
func (s *FraudAlertService) GetBranchAlerts(branchID string, alertType, severity string, isReviewed *bool, dateFrom, dateTo *time.Time, page, limit int) (*repository.FraudAlertListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	result, err := s.alertRepo.List(branchID, alertType, severity, isReviewed, dateFrom, dateTo, page, limit)
	if err != nil {
		log.Printf("[service][fraud_alert] GetBranchAlerts failed: branchID=%s err=%v", branchID, err)
		return nil, fmt.Errorf("không thể tải danh sách cảnh báo")
	}

	return result, nil
}

// ReviewAlert marks a fraud alert as reviewed by a user.
func (s *FraudAlertService) ReviewAlert(alertID, reviewerID, branchID string) error {
	alert, err := s.alertRepo.FindByID(alertID)
	if err != nil {
		log.Printf("[service][fraud_alert] ReviewAlert: alert not found id=%s err=%v", alertID, err)
		return fmt.Errorf("không tìm thấy cảnh báo")
	}

	// RBAC: manager can only review alerts in their own branch
	if branchID != "" && alert.BranchID != branchID {
		log.Printf("[service][fraud_alert] ReviewAlert: branch mismatch alertBranch=%s userBranch=%s", alert.BranchID, branchID)
		return fmt.Errorf("bạn không có quyền xét duyệt cảnh báo này")
	}

	if alert.IsReviewed {
		return nil // Already reviewed, no-op
	}

	if err := s.alertRepo.MarkReviewed(alertID, reviewerID); err != nil {
		log.Printf("[service][fraud_alert] ReviewAlert: mark failed id=%s err=%v", alertID, err)
		return fmt.Errorf("không thể cập nhật trạng thái cảnh báo")
	}

	log.Printf("[service][fraud_alert] alert reviewed: id=%s by=%s", alertID, reviewerID)
	return nil
}
