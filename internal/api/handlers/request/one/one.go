package one

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	resp "github.com/mrdjeb/trueproxy/internal/api/response"
	"github.com/mrdjeb/trueproxy/internal/logger/sl"
	"github.com/mrdjeb/trueproxy/internal/models"
	"github.com/mrdjeb/trueproxy/internal/storage"
)

type RequestGetter interface {
	ReadRequest(uint) (models.RequestResponse, error)
}

func New(log *slog.Logger, requestGetter RequestGetter) echo.HandlerFunc {
	return func(c echo.Context) error {
		const op = "api.request.one.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", c.Request().Header.Get(echo.HeaderXRequestID)),
		)

		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			log.Error("failed to ParseUint ID", sl.Err(err))

			c.JSON(http.StatusBadRequest, resp.Err("bad id"))
			return err
		}

		request, err := requestGetter.ReadRequest(uint(id))
		if err != nil {
			if errors.Is(err, storage.ErrRequestNotFound) {
				log.Warn("request not found", sl.Err(err))

				c.JSON(http.StatusBadRequest, resp.Err(err.Error()))
				return err
			}
			log.Error("failed to requestListGetter", sl.Err(err))

			c.JSON(http.StatusInternalServerError, resp.Err("internal error"))
			return err
		}

		return c.JSON(http.StatusOK, request)
	}
}
