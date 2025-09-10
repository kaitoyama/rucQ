package router

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/traPtitech/rucQ/api"
	"github.com/traPtitech/rucQ/converter"
	"github.com/traPtitech/rucQ/model"
	"github.com/traPtitech/rucQ/repository"
)

func (s *Server) GetDashboard(
	e echo.Context,
	campID api.CampId,
	params api.GetDashboardParams,
) error {
	isParticipant, err := s.repo.IsCampParticipant(
		e.Request().Context(),
		uint(campID),
		*params.XForwardedUser,
	)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return echo.NewHTTPError(
				http.StatusNotFound,
				"Camp not found",
			)
		}

		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to check camp participation: %w", err))
	}

	if !isParticipant {
		return echo.NewHTTPError(http.StatusNotFound, "User is not a participant of this camp")
	}

	res := api.DashboardResponse{
		Id: *params.XForwardedUser,
	}

	// Get latest roll call for the camp using cache service
	latestRollCall, err := s.rollCallCacheService.GetLatestRollCall(e.Request().Context(), uint(campID))
	if err != nil && !errors.Is(err, repository.ErrRollCallNotFound) {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to get latest roll call: %w", err))
	}

	if latestRollCall != nil {
		apiRollCall, err := converter.Convert[api.RollCallResponse](latestRollCall)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).
				SetInternal(fmt.Errorf("failed to convert latest roll call: %w", err))
		}
		res.LatestRollCall = &apiRollCall
	}

	payment, err := s.repo.GetPaymentByUserID(
		e.Request().Context(),
		uint(campID),
		*params.XForwardedUser,
	)

	if err != nil && !errors.Is(err, repository.ErrPaymentNotFound) {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to get payment: %w", err))
	}

	if payment != nil {
		apiPayment, err := converter.Convert[api.PaymentResponse](payment)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).
				SetInternal(fmt.Errorf("failed to convert payment: %w", err))
		}

		res.Payment = &apiPayment
	}

	room, err := s.repo.GetRoomByUserID(e.Request().Context(), uint(campID), *params.XForwardedUser)

	if err != nil && !errors.Is(err, repository.ErrRoomNotFound) {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to get room: %w", err))
	}

	if room != nil {
		apiRoom, err := converter.Convert[api.RoomResponse](room)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).
				SetInternal(fmt.Errorf("failed to convert room: %w", err))
		}

		res.Room = &apiRoom
	}

	return e.JSON(http.StatusOK, &res)
}
