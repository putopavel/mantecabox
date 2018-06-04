package webservice

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"mantecabox/models"
	"mantecabox/services"

	"github.com/appleboy/gin-jwt"
	"github.com/benashford/go-func"
	"github.com/gin-gonic/gin"
)

type (
	FileController interface {
		GetAllFiles(context *gin.Context)
		GetAllFileVersions(context *gin.Context)
		GetFile(context *gin.Context)
		GetFileVersion(context *gin.Context)
		DownloadFile(context *gin.Context)
		DownloadFileVersion(context *gin.Context)
		UploadFile(context *gin.Context)
		DeleteFile(context *gin.Context)
	}

	FileControllerImpl struct {
		configuration *models.Configuration
		fileService   services.FileService
	}
)

func NewFileController(configuration *models.Configuration) FileController {
	fileService := services.NewFileService(configuration)
	if fileService == nil {
		return nil
	}
	return FileControllerImpl{
		configuration: configuration,
		fileService:   fileService,
	}
}

func (fileController FileControllerImpl) GetAllFiles(context *gin.Context) {
	files, err := fileController.fileService.GetAllFiles(getUser(context))
	if err != nil {
		sendJsonMsg(context, http.StatusInternalServerError, "Unable to retrieve files: "+err.Error())
		return
	}
	filesDto := funcs.Maps(files, models.FileToDto).([]models.FileDTO)
	context.JSON(http.StatusOK, filesDto)
}

func (fileController FileControllerImpl) GetAllFileVersions(context *gin.Context) {
	filename := context.Param("file")
	user := getUser(context)
	files, err := fileController.fileService.GetFileVersionsByNameAndOwner(filename, &user)
	if err != nil {
		sendJsonMsg(context, http.StatusInternalServerError, "Unable to retrieve files: "+err.Error())
		return
	}
	filesDto := funcs.Maps(files, models.FileToDto).([]models.FileDTO)
	context.JSON(http.StatusOK, filesDto)
}

func (fileController FileControllerImpl) GetFile(context *gin.Context) {
	filename := context.Param("file")
	user := getUser(context)
	file, err := fileController.fileService.GetLastVersionFileByNameAndOwner(filename, &user)

	if err != nil {
		if err == sql.ErrNoRows {
			sendJsonMsg(context, http.StatusNotFound, fmt.Sprintf(`Unable to find file "%v": %v`, filename, err))
			return
		} else {
			sendJsonMsg(context, http.StatusInternalServerError, fmt.Sprintf(`Unable to find file "%v": %v`, filename, err))
			return
		}
	}
	context.JSON(200, models.FileToDto(file))
}

func (fileController FileControllerImpl) GetFileVersion(context *gin.Context) {
	filename := context.Param("file")
	versionStr := context.Param("version")
	user := getUser(context)
	version, err := strconv.ParseInt(versionStr, 10, 64)
	if err != nil {
		sendJsonMsg(context, http.StatusInternalServerError, "Unable to parse version number: "+err.Error())
		return
	}
	file, err := fileController.fileService.GetFileByVersion(filename, version, &user)
	if err != nil {
		if err == sql.ErrNoRows {
			sendJsonMsg(context, http.StatusNotFound, fmt.Sprintf(`Unable to find file "%v" version %v: %v`, filename, version, err))
			return
		} else {
			sendJsonMsg(context, http.StatusInternalServerError, fmt.Sprintf(`Unable to find file "%v" version %v: %v`, filename, version, err))
			return
		}
	}
	context.JSON(200, models.FileToDto(file))
}

func (fileController FileControllerImpl) DownloadFile(context *gin.Context) {
	filename := context.Param("file")
	user := getUser(context)
	file, err := fileController.fileService.GetLastVersionFileByNameAndOwner(filename, &user)

	if err != nil {
		if err == sql.ErrNoRows {
			sendJsonMsg(context, http.StatusNotFound, fmt.Sprintf(`Unable to find file "%v": %v`, filename, err))
			return
		} else {
			sendJsonMsg(context, http.StatusInternalServerError, fmt.Sprintf(`Unable to find file "%v": %v`, filename, err))
			return
		}
	}

	fileDecrypt, err := fileController.fileService.GetDecryptedLocalFile(file)
	if err != nil {
		sendJsonMsg(context, http.StatusInternalServerError, fmt.Sprintf(`Unable to find file "%v": %v`, filename, err))
	}

	contentLength, contentType, reader, extraHeaders := fileController.fileService.GetFileStream(fileDecrypt, file)
	context.DataFromReader(http.StatusOK, contentLength, contentType, reader, extraHeaders)
}

func (fileController FileControllerImpl) DownloadFileVersion(context *gin.Context) {
	filename := context.Param("file")
	versionStr := context.Param("version")
	user := getUser(context)
	version, err := strconv.ParseInt(versionStr, 10, 64)
	if err != nil {
		sendJsonMsg(context, http.StatusInternalServerError, "Unable to parse version number: "+err.Error())
		return
	}

	file, err := fileController.fileService.GetFileByVersion(filename, version, &user)
	if err != nil {
		if err == sql.ErrNoRows {
			sendJsonMsg(context, http.StatusNotFound, fmt.Sprintf(`Unable to find file "%v": %v`, filename, err))
			return
		} else {
			sendJsonMsg(context, http.StatusInternalServerError, fmt.Sprintf(`Unable to find file "%v": %v`, filename, err))
			return
		}
	}

	fileDecrypt, err := fileController.fileService.GetDecryptedLocalFile(file)
	if err != nil {
		sendJsonMsg(context, http.StatusInternalServerError, fmt.Sprintf(`Unable to find file "%v": %v`, filename, err))
	}

	contentLength, contentType, reader, extraHeaders := fileController.fileService.GetFileStream(fileDecrypt, file)
	context.DataFromReader(http.StatusOK, contentLength, contentType, reader, extraHeaders)
}

// UploadFile se encarga de la subida y cifrado de los ficheros.
func (fileController FileControllerImpl) UploadFile(context *gin.Context) {
	// Obtención del fichero desde el post
	file, header, err := context.Request.FormFile("file")
	permissionsStr, _ := context.GetPostForm("permissions")
	if permissionsStr != "" && len(permissionsStr) != 9 {
		sendJsonMsg(context, http.StatusBadRequest, "Wrong permissions flags (must have 9 characters exactly)")
		return
	}
	if err != nil {
		sendJsonMsg(context, http.StatusBadRequest, err.Error())
		return
	}

	fileModel, err := fileController.fileService.CreateFile(&models.File{
		Name:           header.Filename,
		Owner:          getUser(context),
		PermissionsStr: permissionsStr,
	})
	if err != nil {
		sendJsonMsg(context, http.StatusInternalServerError, err.Error())
		return
	}
	err = fileController.fileService.SaveFile(file, fileModel)
	if err != nil {
		sendJsonMsg(context, http.StatusInternalServerError, err.Error())
		return
	}

	context.JSON(http.StatusCreated, models.FileToDto(fileModel))
}

func (fileController FileControllerImpl) DeleteFile(context *gin.Context) {
	filename := context.Param("file")
	user := getUser(context)
	err := fileController.fileService.DeleteFile(filename, &user)
	if err != nil {
		if err == sql.ErrNoRows {
			sendJsonMsg(context, http.StatusNotFound, "Unable to find file: "+filename)
		} else {
			sendJsonMsg(context, http.StatusBadRequest, "Unable to delete file: "+err.Error())
		}
		return
	}

	context.Writer.WriteHeader(http.StatusNoContent)
}

func getUser(context *gin.Context) models.User {
	var user models.User
	user.Email = jwt.ExtractClaims(context)["id"].(string)
	return user
}
