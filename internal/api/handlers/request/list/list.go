package list

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	resp "github.com/mrdjeb/trueproxy/internal/api/response"
	"github.com/mrdjeb/trueproxy/internal/logger/sl"
	"github.com/mrdjeb/trueproxy/internal/models"
)

/*type RequestsRepo interface {
	CreateRequest(*models.DumpRequest) error
	ReadRequest(id int) (models.DumpRequest, error)
	ReadAllRequest() ([]models.DumpRequest, error)
}*/

type RequestListGetter interface {
	ReadAllRequest() ([]models.RequestResponse, error)
}

func New(log *slog.Logger, requestListGetter RequestListGetter) echo.HandlerFunc {
	return func(c echo.Context) error {
		const op = "api.request.list.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", c.Request().Header.Get(echo.HeaderXRequestID)),
		)

		requests, err := requestListGetter.ReadAllRequest()
		if err != nil {
			log.Error("failed to requestListGetter", sl.Err(err))

			c.JSON(http.StatusInternalServerError, resp.Err("internal error"))
			return err
		}

		//log.Debug("Get this to render worker", (*workers))

		return c.JSON(http.StatusOK, requests)
	}
}
