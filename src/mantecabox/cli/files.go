package cli

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty"
	"github.com/tidwall/gjson"
	"gopkg.in/AlecAivazis/survey.v1"
)

func uploadFile(filePath string, token string) (string, error) {

	s := GetSpinner()
	response, err := resty.R().
		SetFiles(map[string]string{
			"file": filePath,
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

func downloadFile(fileSelected string, token string) error {
	s := GetSpinner()
	response, err := resty.R().
		SetAuthToken(token).
		SetOutput(fileSelected).
		Get("/files/" + fileSelected)
	s.Stop()
	if err != nil {
		return err
	}

	if response.StatusCode() != http.StatusOK {
		return errors.New(ErrorMessage("error downloading file '%v'.", fileSelected))
	}

	return nil
}

func deleteFile(filename string, token string) error {
	s := GetSpinner()
	response, err := resty.R().
		SetAuthToken(token).
		Delete("/files/" + filename)
	s.Stop()
	if err != nil {
		return err
	}

	if response.StatusCode() != http.StatusNoContent {
		return errors.New(ErrorMessage("error removing file '%v'.", filename))
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
			list, listUpdates, err := getFiles(token)
			if err != nil {
				fmt.Printf(err.Error())
			}

			for i := 0; i < len(list); i++  {
				fmt.Printf(" - %v\t%v\n", listUpdates[i].Time().Format(time.RFC822), list[i])
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

func getFileList(token string) (string, error){
	list, _, err := getFiles(token)
	if err != nil {
		return "", err
	}

	var listaString []string
	for _, f := range list {
		listaString = append(listaString, f.Str)
	}

	fileSelected := ""
	prompt := &survey.Select{
		Message: "Please, choose one file: ",
		Options: listaString,
	}

	err = survey.AskOne(prompt, &fileSelected, nil)
	if err != nil {
		return "", err
	}

	return fileSelected, err
}

func getFiles(token string) ([]gjson.Result, []gjson.Result, error) {
	s := GetSpinner()
	response, err := resty.R().
		SetAuthToken(token).
		Get("/files")
	s.Stop()
	if err != nil {
		return nil, nil, err
	}

	if response.StatusCode() == http.StatusOK {
		list := gjson.Get(response.String(), "#.name").Array()
		listUpdates := gjson.Get(response.String(), "#.updated_at").Array()
		if !(len(list) > 0) {
			return nil, nil, errors.New("there are no files in the database. Upload one")
		}
		return list, listUpdates, nil
	} else {
		return nil, nil, errors.New(ErrorMessage("server did not sent HTTP 200 OK status. ") + response.String())
	}
}