package apihttp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
	"github.com/shyun-cc/shw-gf-lib/response"
)

func Authorize(token string) ghttp.HandlerFunc {
	expected := sha256.Sum256([]byte("Bearer " + token))
	return func(r *ghttp.Request) {
		actual := sha256.Sum256([]byte(r.Header.Get("Authorization")))
		if token == "" || subtle.ConstantTimeCompare(actual[:], expected[:]) != 1 {
			response.Unauthorized(r, "a valid API token is required")
			return
		}
		r.Middleware.Next()
	}
}

func Body[I, O any](handler func(context.Context, I) (O, error)) ghttp.HandlerFunc {
	return BodyID(func(ctx context.Context, _ int64, input I) (O, error) {
		return handler(ctx, input)
	})
}

func BodyID[I, O any](handler func(context.Context, int64, I) (O, error)) ghttp.HandlerFunc {
	return serve(func(r *ghttp.Request, id int64) (O, error) {
		var input I
		// GoFrame enforces the server request body size limit.
		decoder := json.NewDecoder(bytes.NewReader(r.GetBody()))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			return inputError[O]("invalid JSON request body")
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			return inputError[O]("request body must contain one JSON value")
		}
		return handler(r.Context(), id, input)
	})
}

func inputError[O any](message string) (O, error) {
	var zero O
	return zero, store.Invalid(message)
}

func ID[O any](handler func(context.Context, int64) (O, error)) ghttp.HandlerFunc {
	return serve(func(r *ghttp.Request, id int64) (O, error) {
		return handler(r.Context(), id)
	})
}

func List[O any](handler func(context.Context, int64, int64) (O, error)) ghttp.HandlerFunc {
	return serve(func(r *ghttp.Request, _ int64) (O, error) {
		after, err := strconv.ParseInt(r.GetQuery("after_id", "0").String(), 10, 64)
		if err != nil || after < 0 {
			return inputError[O]("after_id must be a nonnegative integer")
		}
		limit, err := strconv.ParseInt(r.GetQuery("limit", "50").String(), 10, 64)
		if err != nil || limit < 1 || limit > 200 {
			return inputError[O]("limit must be between 1 and 200")
		}
		return handler(r.Context(), after, limit)
	})
}

func serve[O any](handler func(*ghttp.Request, int64) (O, error)) ghttp.HandlerFunc {
	return func(r *ghttp.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		r.SetCtx(ctx)
		var id int64
		if value := r.GetRouter("id"); value != nil {
			var err error
			id, err = strconv.ParseInt(value.String(), 10, 64)
			if err != nil || id <= 0 {
				writeError(r, store.Invalid("id must be a positive integer"))
				return
			}
		}
		result, err := handler(r, id)
		if err != nil {
			writeError(r, err)
			return
		}
		response.Success(r, result)
	}
}

func writeError(r *ghttp.Request, err error) {
	var input store.InputError
	var conflict store.ConflictError
	var pg *pgconn.PgError
	status, message := http.StatusInternalServerError, "internal server error"
	switch {
	case errors.As(err, &input):
		status, message = http.StatusBadRequest, input.Message
	case errors.As(err, &conflict):
		status, message = http.StatusConflict, conflict.Message
	case errors.Is(err, sql.ErrNoRows):
		status, message = http.StatusNotFound, "record not found"
	case errors.As(err, &pg):
		switch pg.Code {
		case "23505", "23503":
			status, message = http.StatusConflict, "document number is already used or a reference prevents the change"
		case "23514", "23502", "22007", "22008", "22P02", "22003", "22021", "22P05":
			status, message = http.StatusBadRequest, "invalid field value"
		case "40001", "40P01":
			status, message = http.StatusConflict, "concurrent update, retry the request"
		}
	}
	if status == http.StatusInternalServerError {
		// Database errors can embed entered values, so record only the error type/state.
		if pg != nil {
			r.Server.Logger().Errorf(r.Context(), "database request failed (SQLSTATE %s)", pg.Code)
		} else {
			r.Server.Logger().Errorf(r.Context(), "API request failed (%T)", err)
		}
	}
	r.Response.WriteHeader(status)
	response.Error(r, message, status)
}
