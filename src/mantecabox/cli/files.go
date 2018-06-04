package cli

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"mantecabox/models"

	"github.com/go-resty/resty"
	"github.com/mitchellh/go-homedir"
	"github.com/phayes/permbits"
	"github.com/tidwall/gjson"
	"gopkg.in/AlecAivazis/survey.v1"
)

func uploadFile(filePath string, token string) (string, error) {
	permissionBits, err := permbits.Stat(filePath)
	if err != nil {
		return "", err
	}
	s := GetSpinner()
	response, err := resty.R().
		SetFiles(map[string]string{
			"file": filePath,
		}).
		SetFormData(map[string]string{
			"permissions": permissionBits.String(),
		}).
		SetAuthToken(token).
		Post("/files")
	s.Stop()

	if err != nil {
		return "", err
	}

	fileName := gjson.Get(response.String(), "name")

	if response.StatusCode() != http.StatusCreated && response.StatusCode() != http.StatusOK {
		return "", errors.New(ErrorMessage("error uploading file '%v'.", fileName.Str))
	}

	return fileName.Str, nil
}

func downloadFile(selectedFile string, token string) error {
	fileUrl := "/files/" + selectedFile
	var fileDto models.FileDTO
	var serverError models.ServerError

	s := GetSpinner()
	response, err := resty.R().
		SetAuthToken(token).
		SetResult(&fileDto).
		SetError(&serverError).
		Get(fileUrl)
	if err != nil {
		return err
	}
	if serverError.Message != "" {
		return errors.New(serverError.Message)
	}
	if response.StatusCode() != http.StatusOK {
		return errors.New(ErrorMessage("file details did not return 200 OK for '%v'.", selectedFile))
	}

	response, err = resty.R().
		SetAuthToken(token).
		SetOutput(selectedFile).
		Get(fileUrl + "/download")
	s.Stop()
	if err != nil {
		return err
	}

	if response.StatusCode() != http.StatusOK {
		return errors.New(ErrorMessage("error downloading file '%v'.", selectedFile))
	}
	setFilePermissions(selectedFile, fileDto.PermissionsStr)
	return nil
}
func setFilePermissions(selectedFile string, permissionsStr string) error {
	if len(permissionsStr) != 9 {
		return errors.New("wrong permissions string (must contain 9 characters)")
	}

	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	filePath := home + "/Mantecabox/" + selectedFile
	permissionBits, err := permbits.Stat(filePath)
	if err != nil {
		return err
	}
	permissionBits.SetUserRead(permissionsStr[0] != '-')
	permissionBits.SetUserWrite(permissionsStr[1] != '-')
	permissionBits.SetUserExecute(permissionsStr[2] != '-')
	permissionBits.SetGroupRead(permissionsStr[3] != '-')
	permissionBits.SetGroupWrite(permissionsStr[4] != '-')
	permissionBits.SetGroupExecute(permissionsStr[5] != '-')
	permissionBits.SetOtherRead(permissionsStr[6] != '-')
	permissionBits.SetOtherWrite(permissionsStr[7] != '-')
	permissionBits.SetOtherExecute(permissionsStr[8] != '-')

	permbits.Chmod(filePath, permissionBits)
	return nil
}

func deleteFile(filePath string, token string) error {
	s := GetSpinner()
	response, err := resty.R().
		SetAuthToken(token).
		Delete("/files/" + filePath)
	s.Stop()
	if err != nil {
		return err
	}

	if response.StatusCode() != http.StatusNoContent {
		return errors.New(ErrorMessage("error removing file '%v'.", filePath))
	}

	return nil
}

func Transfer(transferActions []string) error {
	token, err := GetToken()
	if err != nil {
		return err
	}

	lengthActions := len(transferActions)

	if lengthActions > 0 {
		switch transferActions[0] {
		case "list":
			names, dates, permissions, err := getFiles(token)
			if err != nil {
				fmt.Printf(err.Error())
			}

			for i := 0; i < len(names); i++ {
				fmt.Printf("%v %v %v\n", permissions[i], dates[i].Time().Format(time.RFC822), names[i])
			}
		case "upload":
			if lengthActions > 1 {
				for i := 1; i < len(transferActions); i++ {
					fileName, err := uploadFile(transferActions[i], token)
					if err != nil {
						fmt.Printf(ErrorMessage("Error uploading file '%v'\n", transferActions[i]))
					} else {
						fmt.Printf(SuccesMessage("File '%v' uploaded correctly.\n", fileName))
					}
				}
			} else {
				return errors.New("params not found")
			}
		case "download":
			if lengthActions > 1 {
				for i := 1; i < len(transferActions); i++ {
					err := downloadFile(transferActions[i], token)
					if err != nil {
						fmt.Printf(ErrorMessage("Error downloading file '%v'.\n", transferActions[i]))
					} else {
						fmt.Printf(SuccesMessage("File '%v' downloaded correctly.\n", transferActions[i]))
					}
				}
			} else {

				fileSelected, err := getFileList(token)
				if err != nil {
					return err
				}

				err = downloadFile(fileSelected, token)
				if err != nil {
					return err
				}
				fmt.Println(SuccesMessage("File '%v' downloaded correctly.", fileSelected))
			}
			fmt.Println("Remember: all your files are located in your Mantecabox User's directory")
		case "remove":
			if lengthActions > 1 {
				for i := 1; i < len(transferActions); i++ {
					err := deleteFile(transferActions[i], token)
					if err != nil {
						return err
					}

					fmt.Println(SuccesMessage("File '%v' removed correctly.\n", transferActions[i]))
				}
			} else {
				fileSelected, err := getFileList(token)
				if err != nil {
					return err
				}

				err = deleteFile(fileSelected, token)
				if err != nil {
					return err
				}
				fmt.Println(SuccesMessage("File '%v' remove correctly.", fileSelected))
			}
		default:
			return errors.New(ErrorMessage("action '%v' not exist", transferActions[0]))
		}
	} else {
		return errors.New(ErrorMessage("action '%v' not found", transferActions[0]))
	}

	return nil
}

func getFileList(token string) (string, error) {
	names, _, _, err := getFiles(token)
	if err != nil {
		return "", err
	}

	var list []string
	for _, f := range names {
		list = append(list, f.Str)
	}

	fileSelected := ""
	prompt := &survey.Select{
		Message: "Please, choose one file: ",
		Options: list,
	}

	err = survey.AskOne(prompt, &fileSelected, nil)
	if err != nil {
		return "", err
	}

	return fileSelected, err
}

func getFiles(token string) ([]gjson.Result, []gjson.Result, []gjson.Result, error) {
	s := GetSpinner()
	response, err := resty.R().
		SetAuthToken(token).
		Get("/files")
	s.Stop()
	if err != nil {
		return nil, nil, nil, err
	}

	if response.StatusCode() == http.StatusOK {
		names := gjson.Get(response.String(), "#.name").Array()
		dates := gjson.Get(response.String(), "#.updated_at").Array()
		permissions := gjson.Get(response.String(), "#.permissions").Array()
		if !(len(names) > 0) {
			return nil, nil, nil, errors.New("there are no files in our servers. Upload one")
		}
		return names, dates, permissions, nil
	} else {
		return nil, nil, nil, errors.New(ErrorMessage("server did not sent HTTP 200 OK status. ") + response.String())
	}
}
