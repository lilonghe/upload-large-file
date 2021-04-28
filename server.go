package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type UploadItem struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	SuccessSlice []int  `json:"success_slice"`
	TotalSlice   int    `json:"total_slice"`
	Suffix       string `json:"suffix"`
	SaveFileName string `json:"save_file_name"`
	Size         int64  `json:"size"`
}

type Response struct {
	Success bool `json:"success"`
}

type FailResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type DataResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

var store = make(map[string]UploadItem, 0)
var tempPath = "./temp/"
var uploadPath = "./upload/"
var lock sync.RWMutex

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
	item.Size, _ = strconv.ParseInt(r.FormValue("size"), 10, 64)

	if len(item.Name) == 0 || item.TotalSlice == 0 {
		failResponse(w, "It's wrong")
		return
	}

	randomNum, _ := rand.Int(rand.Reader, big.NewInt(time.Now().Unix()))
	id := strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + randomNum.String()
	item.Id = id
	item.SaveFileName = id + "." + item.Suffix
	lock.Lock()
	store[id] = item
	lock.Unlock()

	fmt.Println(store)
	dataResponse(w, id)
	return
}

func handlerUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")

	id := r.FormValue("id")
	currentSlice, _ := strconv.Atoi(r.FormValue("current"))
	hasId := false
	lock.Lock()
	for _, v := range store {
		if v.Id == id {
			hasId = true
			break
		}
	}
	lock.Unlock()
	if !hasId {
		failResponse(w, "It's error id")
		fmt.Println("It's error id", id)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		failResponse(w, "It's error read file")
		fmt.Println(err)
		return
	}

	dataArr, err := io.ReadAll(file)
	if err != nil {
		failResponse(w, "It's error read file")
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile(tempPath+getTempFileId(id, currentSlice), dataArr, 0777)
	if err != nil {
		failResponse(w, "It's error write file")
		fmt.Println(err)
		return
	}

	lock.Lock()
	var item UploadItem
	for _, v := range store {
		if v.Id == id {
			item = store[id]
			item.SuccessSlice = append(item.SuccessSlice, currentSlice)
			store[id] = item
			break
		}
	}
	lock.Unlock()
	if len(item.SuccessSlice) == item.TotalSlice {
		err = mergeFile(item, item.SaveFileName)
		if err != nil {
			failResponse(w, "It's error merge file")
			fmt.Println(err)
			return
		}
	}

	successResponse(w)
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

func successResponse(w http.ResponseWriter) {
	resByte, _ := json.Marshal(Response{Success: true})
	fmt.Fprintf(w, string(resByte))
}

func failResponse(w http.ResponseWriter, msg string) {
	resByte, _ := json.Marshal(FailResponse{Success: false, Message: msg})
	fmt.Fprintf(w, string(resByte))
}

func dataResponse(w http.ResponseWriter, data interface{}) {
	resByte, _ := json.Marshal(DataResponse{Success: true, Data: data})
	fmt.Fprintf(w, string(resByte))
}
