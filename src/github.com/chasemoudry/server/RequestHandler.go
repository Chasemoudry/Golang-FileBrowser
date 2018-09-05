package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

//File ...
type File struct {
	Path  string
	Title string
	Body  []byte
}

//Directory ...
type Directory struct {
	Path       string
	Title      string
	ParentPath string
	Dir        []DirectoryInfo
}

// DirectoryInfo ...
type DirectoryInfo struct {
	FullPath string
	SubPath  string
}

const rootDir = "./files/"
const pagingLimit = 5

var pageTemplates = template.Must(template.ParseFiles("file.html", "directory.html"))
var validFilePath = regexp.MustCompile("[0-9]+$")
var validDirectoryPath = regexp.MustCompile("[0-9]+(\\?page=)([1-9]|[1-9][0-9]+)$")
var validPath = regexp.MustCompile("^\\.\\/files((\\/([0-9]+\\/)*[0-9]+\\?page=([1-9]|[1-9][0-9]+))|(\\/([0-9]+))*\\/?)$")

func getSubPath(path string) string {
	fullPath := strings.Split(path, "/")
	if len(fullPath) == 0 {
		return ""
	}
	return fullPath[(len(fullPath) - 1)]
}

func urlFromPath(path string) string {
	return strings.TrimPrefix(path, "./")
}

func trimPathQueries(path string) string {
	return strings.Split(path, "")[0]
}

func isValidPath(path string) bool {
	return validPath.MatchString(path)
}

func getParentPath(path string) string {
	return strings.TrimSuffix(path, getSubPath(path))
}

func fileToString(startPath string, files []os.FileInfo) []DirectoryInfo {
	dirInfo := make([]DirectoryInfo, len(files), len(files))
	for i, v := range files {
		if startPath == rootDir {
			dirInfo[i].FullPath = startPath + v.Name()
		} else {
			dirInfo[i].FullPath = startPath + "/" + v.Name()
		}
		dirInfo[i].SubPath = v.Name()
	}
	return dirInfo
}

func loadFile(path string) (*File, error) {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &File{Path: urlFromPath(path), Title: getSubPath(path), Body: body}, nil
}

func getPagingSlice(targetDir []os.FileInfo, pageNum int) []os.FileInfo {
	sliceLen := len(targetDir)
	fmt.Printf("DEBUG LOG: SLICING FROM %d:%d\n", (pageNum-1)*pagingLimit, sliceLen-(pageNum*pagingLimit))
	targetDir = targetDir[(pageNum-1)*pagingLimit : sliceLen-(pageNum*pagingLimit)]
	return targetDir
}

func getQueries(rawQuery string) (page int) {
	queries := strings.Split(rawQuery, "&")
	var currentQuery []string
	for _, v := range queries {
		currentQuery = strings.Split(v, "=")
		switch {
		case currentQuery[0] == "page":
			fmt.Println("DEBUG LOG: QUERY page = " + currentQuery[1])
			page, _ = strconv.Atoi(currentQuery[1])
		}
	}
	return
}

func renderDirectory(w http.ResponseWriter, tmpl string, dir *Directory) {
	fmt.Println("DEBUG LOG: Directory template loaded.")
	err := pageTemplates.ExecuteTemplate(w, tmpl+".html", dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderFile(w http.ResponseWriter, tmpl string, f *File) {
	fmt.Println("DEBUG LOG: File template loaded.")
	err := pageTemplates.ExecuteTemplate(w, tmpl+".html", f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func directoryHandler(w http.ResponseWriter, r *http.Request, filePath string) {
	fmt.Println("DEBUG LOG: Directory '" + filePath + "' loaded.")
	if r.URL.RawQuery == "" {
		if filePath == rootDir {
			fmt.Println("DEBUG?WRN: DIRECTORY REDIRECTING to " + "?page=1")
			http.Redirect(w, r, "?page=1", http.StatusFound)
		} else {
			fmt.Println("DEBUG?WRN: DIRECTORY REDIRECTING to " + filePath + "?page=1")
			http.Redirect(w, r, getSubPath(filePath)+"?page=1", http.StatusFound)
		}
		return
	}
	currentPage := getQueries(r.URL.RawQuery)
	files, readErr := ioutil.ReadDir(filePath)
	if readErr != nil {
		fmt.Println("DEBUG!ERR: " + readErr.Error())
		renderFile(w, "file", &File{
			Path:  urlFromPath(filePath),
			Title: "Error",
			Body:  []byte(readErr.Error())})
	}
	files = getPagingSlice(files, currentPage)
	// for _, file := range files {
	// 	fmt.Println("DEBUG LOG: " + file.Name())
	// }
	renderDirectory(w, "directory", &Directory{
		Path:       filePath,
		Title:      urlFromPath(filePath),
		ParentPath: getParentPath(filePath),
		Dir:        fileToString(filePath, files)})
}

func fileHandler(w http.ResponseWriter, r *http.Request, filePath string) {
	fmt.Println("DEBUG LOG: File '" + filePath + "' loaded.")
	file, loadErr := loadFile(filePath)
	if loadErr != nil {
		if strings.Contains(loadErr.Error(), "The system cannot find the path specified.") {
			fmt.Println("DEBUG!ERR: " + loadErr.Error())
			renderFile(w, "file", &File{
				Path:  urlFromPath(filePath),
				Title: "Error",
				Body:  []byte(loadErr.Error())})
			return
		}
	}
	renderFile(w, "file", file)
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("DEBUG LOG: Loading... " + r.URL.Path)
	filePath := "." + r.URL.Path
	// checks validity of URL as path
	if isValid := isValidPath(filePath); isValid == false {
		fmt.Println("DEBUG!ERR: Could not access " + filePath + " - Invalid Path.")
		http.NotFound(w, r)
		return
	}
	fileStat, statErr := os.Stat(filePath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			fmt.Println("DEBUG!ERR: " + statErr.Error())
			renderFile(w, "file", &File{
				Path:  r.URL.Path,
				Title: "Error",
				Body:  []byte(statErr.Error())})
			return
		}
	}
	switch mode := fileStat.Mode(); {
	case mode.IsDir():
		// When path leads to directory
		directoryHandler(w, r, filePath)
	case mode.IsRegular():
		// When path leads to file
		fileHandler(w, r, filePath)
	default:
		fmt.Println("DEBUG?WRN: " + filePath + " mode = " + strconv.Itoa((int)(mode)))
	}
}

func redirectToRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Println("DEBUG LOG: Redirected to root directory!")
	http.Redirect(w, r, "/files/", http.StatusFound)
}

func main() {
	// files, err := ioutil.ReadDir("./files/001/")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// for _, file := range files {
	// 	fmt.Println("DEBUG LOG: " + file.Name())
	// }

	http.HandleFunc("/", redirectToRoot)
	http.HandleFunc("/files/", pageHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
