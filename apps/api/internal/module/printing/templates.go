package printing

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/apihttp"
	"github.com/shyun-cc/lathe-base/apps/api/internal/store"
)

type Provider struct {
	Definition Definition
	Load       func(context.Context, int64) (json.RawMessage, error)
}

type templateInput struct {
	Code          string        `json:"code"`
	Name          string        `json:"name"`
	DocumentType  string        `json:"document_type"`
	Configuration Configuration `json:"configuration"`
}

type service struct {
	db        *sql.DB
	providers map[string]Provider
}

var codePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,39}$`)

func Bind(group *ghttp.RouterGroup, db *sql.DB, providers map[string]Provider) {
	s := service{db, providers}
	definitions := make(map[string]Definition, len(providers))
	for key, provider := range providers {
		definitions[key] = provider.Definition
	}
	group.GET("/print-definitions", apihttp.ID(func(context.Context, int64) (map[string]Definition, error) {
		return definitions, nil
	}))
	group.GET("/print-documents/:document_type/:id", func(r *ghttp.Request) {
		provider, ok := providers[r.GetRouter("document_type").String()]
		if !ok {
			apihttp.ID(func(context.Context, int64) (any, error) {
				return nil, store.Invalid("unknown print document type")
			})(r)
			return
		}
		apihttp.ID(provider.Load)(r)
	})
	group.GET("/print-templates", apihttp.List(store.List(db, `SELECT * FROM printing_templates WHERE id > $1 ORDER BY id LIMIT $2`)))
	group.POST("/print-templates", apihttp.Body(s.create))
	group.POST("/print-templates/:id/publish", apihttp.ID(s.publish))
}

func (s service) create(ctx context.Context, input templateInput) (store.ID, error) {
	provider, ok := s.providers[input.DocumentType]
	input.Name = strings.TrimSpace(input.Name)
	if !ok || !codePattern.MatchString(input.Code) || input.Name == "" || len([]rune(input.Name)) > 80 {
		return store.ID{}, store.Invalid("valid document type, template code and name are required")
	}
	if err := input.Configuration.validate(provider.Definition); err != nil {
		return store.ID{}, err
	}
	if input.Configuration.Header == nil {
		input.Configuration.Header = []Cell{}
	}
	if input.Configuration.Footer == nil {
		input.Configuration.Footer = []Cell{}
	}
	configuration, err := json.Marshal(input.Configuration)
	if err != nil {
		return store.ID{}, err
	}
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		if err := lockTemplate(ctx, tx, input.DocumentType, input.Code); err != nil {
			return store.ID{}, err
		}
		return store.Insert(ctx, tx, `
			INSERT INTO printing_templates (document_type, code, version, name, configuration)
			SELECT $1, $2, coalesce(max(version), 0) + 1, $3, $4::jsonb
			FROM printing_templates WHERE document_type = $1 AND code = $2
			RETURNING id`, input.DocumentType, input.Code, input.Name, configuration)
	})
}

func (s service) publish(ctx context.Context, id int64) (store.ID, error) {
	return store.Transaction(ctx, s.db, func(tx *sql.Tx) (store.ID, error) {
		var documentType, code string
		err := tx.QueryRowContext(ctx, "SELECT document_type, code FROM printing_templates WHERE id = $1", id).
			Scan(&documentType, &code)
		if err != nil {
			return store.ID{}, err
		}
		if err := lockTemplate(ctx, tx, documentType, code); err != nil {
			return store.ID{}, err
		}
		_, err = tx.ExecContext(ctx, `UPDATE printing_templates SET active = false
			WHERE document_type = $1 AND code = $2 AND active AND id <> $3`, documentType, code, id)
		if err != nil {
			return store.ID{}, err
		}
		return store.Insert(ctx, tx, "UPDATE printing_templates SET active = true WHERE id = $1 RETURNING id", id)
	})
}

func lockTemplate(ctx context.Context, tx *sql.Tx, documentType, code string) error {
	_, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", "printing:"+documentType+":"+code)
	return err
}
