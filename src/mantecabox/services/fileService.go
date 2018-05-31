package services

import (
	"github.com/paveltrufi/mantecabox/config"
	"github.com/paveltrufi/mantecabox/dao/factory"
	"github.com/paveltrufi/mantecabox/dao/interfaces"
	"github.com/paveltrufi/mantecabox/models"
)

var (
	fileDao interfaces.FileDao
)

func init() {
	dao := factory.FileDaoFactory(config.GetServerConf().Engine)
	fileDao = dao
}

func CreateFile (file *models.File) (models.File, error) {
	return fileDao.Create(file)
}

func DeleteFile (file int64) error {
	return fileDao.Delete(file)
}

func DownloadFile (file int64) (models.File, error) {
	return fileDao.GetByPk(file)
}
