package dao

import (
	"database/sql"

	"mantecabox/logs"
	"mantecabox/models"
	"mantecabox/utilities"
)

const (
	getLoginAttemptsByUserQuery = `SELECT
  la.*,
  u.*
FROM login_attempts la
  JOIN users u ON la."user" = u.email
WHERE u.deleted_at IS NULL AND la."user" = $1`
	getLastNLoginAttemptsByUserQuery = `SELECT *
FROM (SELECT
        la.*,
        u.*
      FROM login_attempts la
        JOIN users u ON la."user" = u.email
      WHERE u.deleted_at IS NULL AND la."user" = $1
      ORDER BY la.id DESC
      LIMIT $2) as reversed
ORDER BY reversed.id`
	insertLoginAttemptQuery = `INSERT INTO login_attempts ("user", user_agent, ip, successful) VALUES ($1, $2, $3, $4) RETURNING *;`
)

type (
	LoginAttemptPgDao struct {
	}

	LoginAttempDao interface {
		GetByUser(email string) ([]models.LoginAttempt, error)
		GetLastNByUser(email string, n int) ([]models.LoginAttempt, error)
		GetSimilarAttempts(attempt *models.LoginAttempt) ([]models.LoginAttempt, error)
		Create(attempt *models.LoginAttempt) (models.LoginAttempt, error)
	}
)

func withDb(f func(db *sql.DB) (interface{}, error)) (interface{}, error) {
	logs.DaoLog.Debug("withDb")
	db, err := utilities.GetPgDb()
	if err != nil {
		logs.DaoLog.Fatalf("Unable to connnect with database: %v" + err.Error())
		return models.LoginAttempt{}, err
	}
	defer db.Close()
	return f(db)
}

func scanLoginAttemptWithNestedUser(rows *sql.Rows) ([]models.LoginAttempt, error) {
	logs.DaoLog.Debug("scanLoginAttemptWithNestedUser")
	attempts := make([]models.LoginAttempt, 0)
	for rows.Next() {
		var attempt models.LoginAttempt
		var user string
		err := rows.Scan(&attempt.Id,
			&attempt.CreatedAt,
			&user,
			&attempt.UserAgent,
			&attempt.IP,
			&attempt.Successful,
			&attempt.User.CreatedAt,
			&attempt.User.UpdatedAt,
			&attempt.User.DeletedAt,
			&attempt.User.Email,
			&attempt.User.Password,
			&attempt.User.TwoFactorAuth,
			&attempt.User.TwoFactorTime,
		)
		if err != nil {
			logs.DaoLog.Infof("Unable to execute LoginAttemptPgDao.scanLoginAttemptWithNestedUser() scan. Reason: %v", err)
			return nil, err
		}
		attempts = append(attempts, attempt)
	}

	logs.DaoLog.Info("Queried  ", len(attempts), " login attempts")
	return attempts, nil
}

func (dao LoginAttemptPgDao) GetByUser(email string) ([]models.LoginAttempt, error) {
	logs.DaoLog.Debug("GetByUser")
	res, err := withDb(func(db *sql.DB) (interface{}, error) {
		rows, err := db.Query(getLoginAttemptsByUserQuery, email)
		if err != nil {
			logs.DaoLog.Infof("Unable to execute LoginAttemptPgDao.GetByUser() query. Reason: %v", err)
			return nil, err
		}
		return scanLoginAttemptWithNestedUser(rows)
	})
	return res.([]models.LoginAttempt), err
}

func (dao LoginAttemptPgDao) GetLastNByUser(email string, n int) ([]models.LoginAttempt, error) {
	logs.DaoLog.Debug("GetLastNByUser")
	if n < 0 {
		return dao.GetByUser(email)
	}
	res, err := withDb(func(db *sql.DB) (interface{}, error) {
		rows, err := db.Query(getLastNLoginAttemptsByUserQuery, email, n)
		if err != nil {
			logs.DaoLog.Infof("Unable to execute LoginAttemptPgDao.GetLastNByUser() query. Reason: %v", err)
			return nil, err
		}
		return scanLoginAttemptWithNestedUser(rows)
	})
	return res.([]models.LoginAttempt), err
}

func (dao LoginAttemptPgDao) Create(attempt *models.LoginAttempt) (models.LoginAttempt, error) {
	logs.DaoLog.Debug("Create")
	res, err := withDb(func(db *sql.DB) (interface{}, error) {
		var createdAttempt models.LoginAttempt
		err := db.QueryRow(insertLoginAttemptQuery, attempt.User.Email, attempt.UserAgent, attempt.IP, attempt.Successful).
			Scan(&createdAttempt.Id, &createdAttempt.CreatedAt, &createdAttempt.User.Email,
				&createdAttempt.UserAgent, &createdAttempt.IP, &createdAttempt.Successful)
		if err != nil {
			logs.DaoLog.Infof("Unable to execute FilePgDao.Create(file models.File) query. Reason: %v", err)
			return createdAttempt, err
		} else {
			logs.DaoLog.Infof("Created file: %v", createdAttempt)
		}
		owner, err := UserPgDao{}.GetByPk(createdAttempt.User.Email)
		createdAttempt.User = owner
		return createdAttempt, err
	})
	return res.(models.LoginAttempt), err
}

func (dao LoginAttemptPgDao) GetSimilarAttempts(attempt *models.LoginAttempt) ([]models.LoginAttempt, error) {
	logs.DaoLog.Debug("GetSimilarAttempts")
	res, err := withDb(func(db *sql.DB) (interface{}, error) {
		rows, err := db.Query(`SELECT
		la.*,
		u.*
		FROM login_attempts la
		JOIN users u ON la."user" = u.email
		WHERE u.deleted_at IS NULL
		AND la."user" = $1
		AND la.user_agent = $2
		AND la.ip = $3;`, attempt.User.Email, attempt.UserAgent, attempt.IP)
		if err != nil {
			logs.DaoLog.Infof("Unable to execute LoginAttemptPgDao.GetSimilarAttempts() query. Reason: %v", err)
			return nil, err
		}
		return scanLoginAttemptWithNestedUser(rows)
	})
	return res.([]models.LoginAttempt), err
}
