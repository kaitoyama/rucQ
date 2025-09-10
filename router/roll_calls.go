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

func (s *Server) GetRollCalls(e echo.Context, campID api.CampId) error {
	rollCalls, err := s.repo.GetRollCalls(e.Request().Context(), uint(campID))

	if err != nil {
		if errors.Is(err, repository.ErrCampNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Camp not found")
		}

		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to get roll calls: %w", err))
	}

	res, err := converter.Convert[[]api.RollCallResponse](rollCalls)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to convert roll calls: %w", err))
	}

	return e.JSON(http.StatusOK, res)
}

func (s *Server) AdminPostRollCall(
	e echo.Context,
	campID api.CampId,
	params api.AdminPostRollCallParams,
) error {
	if params.XForwardedUser == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "X-Forwarded-User header is required")
	}

	user, err := s.repo.GetOrCreateUser(e.Request().Context(), *params.XForwardedUser)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to get or create user: %w", err))
	}

	if !user.IsStaff {
		return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
	}

	var req api.AdminPostRollCallJSONRequestBody

	if err := e.Bind(&req); err != nil {
		return err
	}

	rollCall, err := converter.Convert[model.RollCall](req)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to convert request body: %w", err))
	}

	rollCall.CampID = uint(campID)

	if err := s.repo.CreateRollCall(e.Request().Context(), &rollCall); err != nil {
		if errors.Is(err, repository.ErrCampNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Camp not found")
		}

		if errors.Is(err, repository.ErrUserNotFound) {
			return echo.NewHTTPError(http.StatusBadRequest, "One or more subject users not found")
		}

		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to create roll call: %w", err))
	}

	// Invalidate cache for this camp since we created a new roll call
	s.rollCallCacheService.InvalidateCache(uint(campID))

	res, err := converter.Convert[api.RollCallResponse](rollCall)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to convert roll call: %w", err))
	}

	return e.JSON(http.StatusCreated, res)
}

func (s *Server) GetRollCallReactions(e echo.Context, rollCallID api.RollCallId) error {
	reactions, err := s.repo.GetRollCallReactions(e.Request().Context(), uint(rollCallID))

	if err != nil {
		if errors.Is(err, repository.ErrRollCallNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Roll call not found")
		}

		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to get roll call reactions: %w", err))
	}

	res, err := converter.Convert[[]api.RollCallReactionResponse](reactions)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to convert roll call reactions: %w", err))
	}

	return e.JSON(http.StatusOK, res)
}

func (s *Server) PostRollCallReaction(
	e echo.Context,
	rollCallID api.RollCallId,
	params api.PostRollCallReactionParams,
) error {
	if params.XForwardedUser == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "X-Forwarded-User header is required")
	}

	user, err := s.repo.GetOrCreateUser(e.Request().Context(), *params.XForwardedUser)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to get or create user: %w", err))
	}

	var req api.PostRollCallReactionJSONRequestBody

	if err := e.Bind(&req); err != nil {
		return err
	}

	reaction, err := converter.Convert[model.RollCallReaction](req)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to convert request body: %w", err))
	}

	reaction.RollCallID = uint(rollCallID)
	reaction.UserID = user.ID

	if err := s.repo.CreateRollCallReaction(e.Request().Context(), &reaction); err != nil {
		if errors.Is(err, repository.ErrRollCallNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Roll call not found")
		}

		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to create roll call reaction: %w", err))
	}

	var eventData api.RollCallReactionEvent

	if err := eventData.FromRollCallReactionCreatedEvent(api.RollCallReactionCreatedEvent{
		Id:      int(reaction.ID),
		Type:    api.Created,
		UserId:  user.ID,
		Content: reaction.Content,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to create event data: %w", err))
	}

	go s.reactionPubSub.Send(reactionEvent{
		rollCallID: uint(rollCallID),
		data:       eventData,
	})

	res, err := converter.Convert[api.RollCallReactionResponse](reaction)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to convert roll call reaction: %w", err))
	}

	return e.JSON(http.StatusCreated, res)
}

func (s *Server) PutReaction(
	e echo.Context,
	reactionID api.ReactionId,
	params api.PutReactionParams,
) error {
	if params.XForwardedUser == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "X-Forwarded-User header is required")
	}

	// リアクションの存在確認と所有者チェック
	existingReaction, err := s.repo.GetRollCallReactionByID(e.Request().Context(), uint(reactionID))

	if err != nil {
		if errors.Is(err, repository.ErrRollCallReactionNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Reaction not found")
		}

		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to get roll call reaction: %w", err))
	}

	// ユーザーの所有者チェック
	if existingReaction.UserID != *params.XForwardedUser {
		return echo.NewHTTPError(http.StatusForbidden, "You can only edit your own reactions")
	}

	var req api.PutReactionJSONRequestBody

	if err := e.Bind(&req); err != nil {
		return err
	}

	updateData, err := converter.Convert[model.RollCallReaction](req)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to convert request body: %w", err))
	}

	if err := s.repo.UpdateRollCallReaction(e.Request().Context(), uint(reactionID), &updateData); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to update roll call reaction: %w", err))
	}

	// 更新後のデータを取得
	updatedReaction, err := s.repo.GetRollCallReactionByID(e.Request().Context(), uint(reactionID))

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to get updated roll call reaction: %w", err))
	}

	var eventData api.RollCallReactionEvent

	if err := eventData.FromRollCallReactionUpdatedEvent(api.RollCallReactionUpdatedEvent{
		Id:      int(updatedReaction.ID),
		Type:    api.Updated,
		UserId:  updatedReaction.UserID,
		Content: updatedReaction.Content,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to create event data: %w", err))
	}

	go s.reactionPubSub.Send(reactionEvent{
		rollCallID: uint(updatedReaction.RollCallID),
		data:       eventData,
	})

	res, err := converter.Convert[api.RollCallReactionResponse](*updatedReaction)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to convert roll call reaction: %w", err))
	}

	return e.JSON(http.StatusOK, res)
}

func (s *Server) DeleteReaction(
	e echo.Context,
	reactionID api.ReactionId,
	params api.DeleteReactionParams,
) error {
	if params.XForwardedUser == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "X-Forwarded-User header is required")
	}

	// リアクションの存在確認と所有者チェック
	existingReaction, err := s.repo.GetRollCallReactionByID(e.Request().Context(), uint(reactionID))

	if err != nil {
		if errors.Is(err, repository.ErrRollCallReactionNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "Reaction not found")
		}

		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to get roll call reaction: %w", err))
	}

	// ユーザーの所有者チェック
	if existingReaction.UserID != *params.XForwardedUser {
		return echo.NewHTTPError(http.StatusForbidden, "You can only delete your own reactions")
	}

	if err := s.repo.DeleteRollCallReaction(e.Request().Context(), uint(reactionID)); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to delete roll call reaction: %w", err))
	}

	var eventData api.RollCallReactionEvent

	if err := eventData.FromRollCallReactionDeletedEvent(api.RollCallReactionDeletedEvent{
		Id:     int(existingReaction.ID),
		Type:   api.Deleted,
		UserId: existingReaction.UserID,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError).
			SetInternal(fmt.Errorf("failed to create event data: %w", err))
	}

	go s.reactionPubSub.Send(reactionEvent{
		rollCallID: uint(existingReaction.RollCallID),
		data:       eventData,
	})

	return e.NoContent(http.StatusNoContent)
}

func (s *Server) StreamRollCallReactions(e echo.Context, rollCallID api.RollCallId) error {
	res := e.Response()

	res.Header().Set(echo.HeaderContentType, "text/event-stream")

	sub := s.reactionPubSub.Subscribe(e.Request().Context(), maxReactionEventBuffer)

	for event := range sub {
		if event.rollCallID != uint(rollCallID) {
			continue
		}

		b, err := event.data.MarshalJSON()

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).
				SetInternal(fmt.Errorf("failed to marshal event data: %w", err))
		}

		if _, err := fmt.Fprintf(res, "data: %s\n\n", b); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError).
				SetInternal(fmt.Errorf("failed to write event data: %w", err))
		}

		res.Flush()
	}

	return nil
}
