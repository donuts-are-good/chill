package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const Ascii = `
     _   _ _ _ 
 ___| |_|_| | |
|  _|   | | | |
|___|_|_|_|_|_|` + semVerInfo + "\n"

const semVerInfo = "v1.0.0"

// mediafile represents a media file with its name and path.
type MediaFile struct {
	Name string
	Path string
}

// mediagroup represents a group of media files within a specific directory.
type MediaGroup struct {
	Directory string
	Files     []MediaFile
}

// categoryconfig represents the configuration for a media category.
type CategoryConfig struct {
	Name      string
	Directory string
	FileTypes []string
}

func main() {

	// define the configuration file path
	configFile := "config.cfg"

	// load the media directories from the config file
	mediaConfigs, err := LoadMediaDirectories(configFile)
	if err != nil {
		log.Fatal("Failed to load media configurations:", err)
	}

	// create file server handlers for each directory
	fileServers := make(map[string]http.Handler)
	for _, config := range mediaConfigs {
		fileServers[config.Directory] = http.FileServer(http.Dir(config.Directory))
	}

	// create a custom handler for serving the media files and generating the file list
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// check if the request is a specific file
		for _, config := range mediaConfigs {
			filePath := filepath.Join(config.Directory, r.URL.Path[1:])
			if fileInfo, err := os.Stat(filePath); err == nil && !fileInfo.IsDir() {
				fs := fileServers[config.Directory]
				fs.ServeHTTP(w, r)
				return
			}
		}

		// generate the list of media from all directories based on the provided mediaconfigs.
		// each directory is processed separately, and the resulting media files are grouped within mediagroup.
		fileList := make([]MediaGroup, 0)
		for _, config := range mediaConfigs {
			group := MediaGroup{Directory: config.Directory, Files: []MediaFile{}}

			// walk through the files in the directory and its subdirectories
			err := filepath.Walk(config.Directory, func(path string, info os.FileInfo, err error) error {
				if err != nil {

					// handle the error and continue traversal
					log.Println("Error accessing file:", err)
					return nil
				}

				// check if the file is not a directory and has an allowed file type
				if !info.IsDir() && isAllowedFileType(path, config.FileTypes) {

					// get the relative path to the directory
					relPath, _ := filepath.Rel(config.Directory, path)

					// append the mediafile to the group's files
					group.Files = append(group.Files, MediaFile{Name: info.Name(), Path: relPath})
				}
				return nil
			})

			if err != nil {

				// handle the error and return an internal server error response
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// append the group to the list of mediagroup
			fileList = append(fileList, group)
		}

		// render the template with the generated list of media groups
		tmpl, err := template.New("index").Parse(indexTemplate)
		if err != nil {

			// handle the error and return an internal server error response
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// prepare the data to be passed to the template
		data := struct{ Groups []MediaGroup }{Groups: fileList}

		// execute the template with the provided data and write the response to the client
		err = tmpl.Execute(w, data)
		if err != nil {

			// handle the error and log it
			log.Println("Error executing template:", err)
			return
		}

	})

	// start the server on port 8080
	fmt.Println(Ascii + "http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// check if the file has an allowed media file type
func isAllowedFileType(path string, fileTypes []string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	// iterate over each file type in the list
	for _, fileType := range fileTypes {

		// check if the lowercase file extension matches the current file type
		if ext == fileType {
			return true
		}
	}

	// no matching file type found
	return false
}

func LoadMediaDirectories(configFile string) ([]CategoryConfig, error) {

	// initialize an empty slice to store the media configurations
	var mediaConfigs []CategoryConfig

	// open the configuration file
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// initialize the current category index
	var currentCategoryIndex int

	// create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// iterate over each line in the file
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// skip empty lines and lines starting with '#' (comments)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// check if the line represents a new category
		if line[0] == '[' && line[len(line)-1] == ']' {
			currentCategory := line[1 : len(line)-1]
			mediaConfigs = append(mediaConfigs, CategoryConfig{Name: currentCategory})

			// create a new categoryconfig for the category
			currentCategoryIndex = len(mediaConfigs) - 1

			// update the current category index
		} else {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			// extract the key and value from the line
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// process the key-value pair based on the key
			switch key {
			case "Directory":

				// set the directory for the current category
				mediaConfigs[currentCategoryIndex].Directory = value
			case "FileTypes":

				// split the file types into a slice
				fileTypes := strings.Split(value, ",")
				for i := range fileTypes {

					// trim spaces from each file type
					fileTypes[i] = strings.TrimSpace(fileTypes[i])
				}

				// set the file types for the current category
				mediaConfigs[currentCategoryIndex].FileTypes = fileTypes
			}
		}
	}

	// check for any scanner errors
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// return the populated media configurations
	return mediaConfigs, nil
}

// html template for rendering the file list
const indexTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0-alpha3/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-KK94CHFLLe+nY2dmCWGMq91rCGa5gtU4mk92HdvYe+M/SXH301p5ILy+dN9+nJOZ" crossorigin="anonymous">

    <style>
        @media (orientation: portrait) {
            .column-count {
                column-count: 2;
            }
        }

        @media (orientation: landscape) {
            .column-count {
                column-count: 3;
            }
        }
    </style>
		<title>Chill Media Player</title>
</head>
<body>
<div class="container-fluid">
    <div class="row">
        <div class="col">
            <h1>Chill Media Player</h1>
        </div>
    </div>
    <div class="row">
        <div class="col column-count">
            <ul>
                {{range .Groups}}
                <li>
                    <strong>{{.Directory}}</strong>
                    <ul>
                        {{range .Files}}
                        <li>
                            <a href="{{.Path}}" name="{{.Path}}" title="{{.Path}}" target="_blank">{{.Name}}</a>
                        </li>
                        {{end}}
                    </ul>
                </li>
                {{end}}
            </ul>
        </div>
    </div>
</div>
</body>
</html>
`
