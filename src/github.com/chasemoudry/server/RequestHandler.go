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
	Path       string
	Title      string
	ParentPath string
	Body       []byte
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
	pathLen := len(fullPath)
	if pathLen == 0 {
		return ""
	}
	if fullPath[(pathLen-1)] == "" {
		fmt.Printf("DEBUG LOG: subpath for %s = %s\n", path, fullPath[(pathLen-2)])
		return fullPath[(pathLen - 2)]
	}
	fmt.Printf("DEBUG LOG: subpath for %s = %s\n", path, fullPath[(pathLen-1)])
	return fullPath[(pathLen - 1)]
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

func getDirectoryInfo(path string, files []os.FileInfo) []DirectoryInfo {
	dirInfo := make([]DirectoryInfo, len(files), len(files))
	for i, v := range files {
		if path == rootDir {
			dirInfo[i].FullPath = path + v.Name()
		} else {
			dirInfo[i].FullPath = path + "/" + v.Name()
		}
		dirInfo[i].SubPath = v.Name()
	}
	return dirInfo
}

func getPagingSlice(targetDir []os.FileInfo, pageNum int) []os.FileInfo {
	sliceLen := len(targetDir)
	if pageNum*pagingLimit <= sliceLen {
		targetDir = targetDir[(pageNum-1)*pagingLimit : (pageNum * pagingLimit)]
	} else {
		targetDir = targetDir[(pageNum-1)*pagingLimit : sliceLen]
	}
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

func directoryHandler(w http.ResponseWriter, r *http.Request, filePath string) {
	fmt.Println("DEBUG LOG: Directory '" + filePath + "' loaded.")
	if r.URL.RawQuery == "" {
		if filePath == rootDir {
			filePath = strings.TrimSuffix(filePath, "/")
			fmt.Println("DEBUG?WRN: DIRECTORY REDIRECTING to ?page=1")
			http.Redirect(w, r, "?page=1", http.StatusFound)
		} else {
			filePath = strings.TrimSuffix(filePath, "/")
			fmt.Println("DEBUG?WRN: DIRECTORY REDIRECTING to " + filePath + "?page=1")
			// http.Redirect(w, r, getSubPath(filePath)+"?page=1", http.StatusFound)
			http.Redirect(w, r, "?page=1", http.StatusFound)
		}
		return
	}
	currentPage := getQueries(r.URL.RawQuery)
	files, readErr := ioutil.ReadDir(filePath)
	if readErr != nil {
		fmt.Println("DEBUG!ERR: " + readErr.Error())
		renderFile(w, "file", &File{
			Path:       urlFromPath(filePath),
			Title:      "Error",
			ParentPath: getParentPath(filePath),
			Body:       []byte(readErr.Error())})
	}
	files = getPagingSlice(files, currentPage)
	// for _, file := range files {
	// 	fmt.Println("DEBUG LOG: " + file.Name())
	// }
	renderDirectory(w, "directory", &Directory{
		Path:       urlFromPath(filePath),
		Title:      getSubPath(filePath),
		ParentPath: getParentPath(filePath),
		Dir:        getDirectoryInfo(filePath, files)})
}

func renderFile(w http.ResponseWriter, tmpl string, f *File) {
	fmt.Println("DEBUG LOG: File template loaded.")
	err := pageTemplates.ExecuteTemplate(w, tmpl+".html", f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func loadFile(filePath string) (*File, error) {
	body, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return &File{
			Path:       urlFromPath(filePath),
			Title:      getSubPath(filePath),
			ParentPath: getParentPath(filePath),
			Body:       body},
		nil
}

func fileHandler(w http.ResponseWriter, r *http.Request, filePath string) {
	fmt.Println("DEBUG LOG: File '" + filePath + "' loaded.")
	file, loadErr := loadFile(filePath)
	if loadErr != nil {
		if strings.Contains(loadErr.Error(), "The system cannot find the path specified.") {
			fmt.Println("DEBUG!ERR: " + loadErr.Error())
			renderFile(w, "file", &File{
				Path:       urlFromPath(filePath),
				Title:      "Error",
				ParentPath: getParentPath(filePath),
				Body:       []byte(loadErr.Error())})
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
				Path:       r.URL.Path,
				Title:      "Error",
				ParentPath: getParentPath(filePath),
				Body:       []byte(statErr.Error())})
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
		fmt.Println("DEBUG?WRN: " + filePath + " unrecognized mode = " + strconv.Itoa((int)(mode)))
	}
}

func redirectToRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Println("DEBUG LOG: Redirected to root directory!")
	http.Redirect(w, r, "/files/", http.StatusFound)
}

func main() {
	http.HandleFunc("/", redirectToRoot)
	http.HandleFunc("/files/", pageHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
