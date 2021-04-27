package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"
)

type UploadItem struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	SuccessSlice []int  `json:"success_slice"`
	TotalSlice   int    `json:"total_slice"`
	Suffix       string `json:"suffix"`
	SaveFileName string
}

var store []UploadItem
var tempPath = "./temp/"
var uploadPath = "./upload/"

func main() {
	checkPath(tempPath)
	checkPath(uploadPath)

	http.HandleFunc("/check", handlerUploadCheck)
	http.HandleFunc("/upload", handlerUpload)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func checkPath(path string) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(path, os.ModePerm)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}
}

func handlerUploadCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	item := UploadItem{}
	item.Name = r.FormValue("name")
	item.TotalSlice, _ = strconv.Atoi(r.FormValue("total"))
	item.Suffix = r.FormValue("suffix")
	if len(item.Name) == 0 || item.TotalSlice == 0 {
		fmt.Fprint(w, "It's wrong")
		return
	}

	randomNum, _ := rand.Int(rand.Reader, big.NewInt(time.Now().Unix()))
	id := strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + randomNum.String()
	item.Id = id
	store = append(store, item)

	fmt.Println(store)
	fmt.Fprint(w, id)
	return
}

func handlerUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	id := r.FormValue("id")
	currentSlice, _ := strconv.Atoi(r.FormValue("current"))
	hasId := false
	for _, v := range store {
		if v.Id == id {
			hasId = true
			break
		}
	}
	if !hasId {
		fmt.Fprintf(w, "It's error id")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		fmt.Fprintf(w, "It's error read file")
		fmt.Println(err)
		return
	}

	dataArr, err := io.ReadAll(file)
	if err != nil {
		fmt.Fprintf(w, "It's error read file")
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile(tempPath+getTempFileId(id, currentSlice), dataArr, 0777)
	if err != nil {
		fmt.Fprintf(w, "It's error write file")
		fmt.Println(err)
		return
	}

	for i, v := range store {
		if v.Id == id {
			store[i].SuccessSlice = append(store[i].SuccessSlice, currentSlice)
			if len(store[i].SuccessSlice) == store[i].TotalSlice {
				fileName := store[i].Id + "." + store[i].Suffix
				err = mergeFile(store[i], fileName)
				if err != nil {
					fmt.Fprintf(w, "It's error merge file")
					fmt.Println(err)
					return
				}
				store[i].SaveFileName = fileName
			}

			break
		}
	}

	fmt.Fprintf(w, "You are done!")
}

func mergeFile(targetUploadInfo UploadItem, fileName string) error {
	data := make([]byte, 0)
	for _, v := range targetUploadInfo.SuccessSlice {
		d, err := ioutil.ReadFile(tempPath + getTempFileId(targetUploadInfo.Id, v))
		if err != nil {
			return err
		}
		data = append(data, d...)
	}
	err := ioutil.WriteFile(uploadPath+fileName, data, 0777)
	if err != nil {
		return err
	}
	for _, v := range targetUploadInfo.SuccessSlice {
		err = os.Remove(tempPath + getTempFileId(targetUploadInfo.Id, v))
		if err != nil {
			fmt.Print(err)
		}
	}
	return nil
}

func getTempFileId(id string, sliceNum int) string {
	return id + "-" + strconv.Itoa(sliceNum)
}
