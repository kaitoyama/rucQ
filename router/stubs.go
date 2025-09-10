package router

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/traPtitech/rucQ/api"
)

// 欠けているメソッドのスタブ実装

// AdminDeleteImage 画像を削除（管理者用）
func (s *Server) AdminDeleteImage(
	_ echo.Context,
	_ api.ImageId,
	_ api.AdminDeleteImageParams,
) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "AdminDeleteImage not implemented")
}

// AdminPostImage 画像をアップロード（管理者用）
func (s *Server) AdminPostImage(_ echo.Context, _ api.CampId, _ api.AdminPostImageParams) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "AdminPostImage not implemented")
}

// GetImages 画像の一覧を取得
func (s *Server) GetImages(_ echo.Context, _ api.CampId) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "GetImages not implemented")
}

// GetImage 画像を取得
func (s *Server) GetImage(_ echo.Context, _ api.ImageId) error {
	return echo.NewHTTPError(http.StatusNotImplemented, "GetImage not implemented")
}
