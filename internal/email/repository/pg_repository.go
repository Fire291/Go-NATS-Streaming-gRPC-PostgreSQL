package repository

import (
	"context"

	"github.com/AleksK1NG/nats-streaming/internal/models"
	"github.com/AleksK1NG/nats-streaming/pkg/utils"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

type emailPGRepository struct {
	db *pgxpool.Pool
}

func NewEmailPGRepository(db *pgxpool.Pool) *emailPGRepository {
	return &emailPGRepository{db: db}
}

func (e *emailPGRepository) Create(ctx context.Context, email *models.Email) (*models.Email, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "emailPGRepository.Create")
	defer span.Finish()

	createEmailQuery := `INSERT INTO emails (addressfrom, addressto, subject, message) 
	VALUES ($1, $2, $3, $4) 
	RETURNING email_id, addressfrom, addressto, subject, message, created_at`

	var mail models.Email
	if err := e.db.QueryRow(ctx, createEmailQuery, &email.From, &email.To, &email.Subject, &email.Message).Scan(&mail); err != nil {
		return nil, errors.Wrap(err, "Scan")
	}

	return &mail, nil
}

func (e *emailPGRepository) GetByID(ctx context.Context, emailID uuid.UUID) (*models.Email, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "emailPGRepository.GetByID")
	defer span.Finish()

	getByIDQuery := `SELECT email_id, addressfrom, addressto, subject, message, created_at FROM emails WHERE email_id = $1`
	var mail models.Email
	if err := e.db.QueryRow(ctx, getByIDQuery, emailID).Scan(&mail); err != nil {
		return nil, errors.Wrap(err, "Scan")
	}

	return &mail, nil
}

func (e *emailPGRepository) Search(ctx context.Context, search string, pagination *utils.Pagination) (*models.EmailsList, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "emailPGRepository.Search")
	defer span.Finish()

	searchTotalCountQuery := `SELECT count(email_id) FROM emails WHERE addressto ILIKE '%' || $1 || '%' 
	ORDER BY created_at OFFSET $2 LIMIT  $3`

	var count int
	if err := e.db.QueryRow(ctx, searchTotalCountQuery, search, pagination.GetOffset(), pagination.GetLimit()).Scan(&count); err != nil {
		return nil, errors.Wrap(err, "QueryRow")
	}
	if count == 0 {
		return &models.EmailsList{
			TotalCount: 0,
			TotalPages: 0,
			Page:       0,
			Size:       0,
			HasMore:    false,
			Emails:     make([]*models.Email, 0),
		}, nil
	}

	searchQuery := `SELECT email_id, addressfrom, addressto, subject, message, created_at 
	FROM emails WHERE addressto ILIKE '%' || $1 || '%' ORDER BY created_at OFFSET $2 LIMIT  $3`
	rows, err := e.db.Query(ctx, searchQuery, searchQuery, pagination.GetOffset(), pagination.GetLimit())
	if err != nil {
		return nil, errors.Wrap(err, "db.Query")
	}
	defer rows.Close()

	emailList := make([]*models.Email, 0, count)
	for rows.Next() {
		var m models.Email
		if err := rows.Scan(&m.EmailID, &m.From, &m.EmailID, &m.To, &m.Subject, &m.Message, &m.CreatedAt); err != nil {
			return nil, errors.Wrap(err, " rows.Scan")
		}
		emailList = append(emailList, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows.Err")
	}

	return &models.EmailsList{
		TotalCount: int64(count),
		TotalPages: int64(pagination.GetTotalPages(count)),
		Page:       int64(pagination.GetPage()),
		Size:       int64(pagination.GetSize()),
		HasMore:    pagination.GetHasMore(count),
		Emails:     emailList,
	}, nil
}
