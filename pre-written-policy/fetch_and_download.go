package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func downloadFile(filepath, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func unzipFile(zip_path, dest_dir string) error {
	r, err := zip.OpenReader(zip_path)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest_dir, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		inFile, err := f.Open()
		if err != nil {
			return err
		}
		defer inFile.Close()

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, inFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func moveNestedContents(temp_dir, dest_dir string) error {
	entries, err := os.ReadDir(temp_dir)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("no folders found in temp dir")
	}

	firstLevel := filepath.Join(temp_dir, entries[0].Name())
	secondLevelEntries, err := os.ReadDir(firstLevel)
	if err != nil {
		return err
	}

	for _, e := range secondLevelEntries {
		src_path := filepath.Join(firstLevel, e.Name())
		dst_path := filepath.Join(dest_dir, e.Name())

		err := os.Rename(src_path, dst_path)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleDownload(input map[string]string) {
	var_name := input["name"]
	policy_owner := input["owner"]
	policy_github_repository := input["repo"]

	download_dir := filepath.Join("downloads-"+var_name)
	unzip_dir := filepath.Join("unzipped-"+var_name)
	temp_dir := filepath.Join("temp_unzip-"+var_name)

	// Create directories if they do not exist
	err := os.MkdirAll(download_dir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error on creating download directory: %v\n", err)
		os.Exit(1)
	}
	err = os.MkdirAll(temp_dir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error on creating temp directory: %v\n", err)
		os.Exit(1)
	}
	err = os.MkdirAll(unzip_dir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error on creating unzip directory: %v\n", err)
		os.Exit(1)
	}

	tags_url := fmt.Sprintf("https://api.github.com/repos/%s/%s/tags", policy_owner, policy_github_repository)

	// Fetch the latest tags from the GitHub API
	res, err := http.Get(tags_url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching tags: %v\n", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: received status code %d\n", res.StatusCode)
		return
	}
	tags, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		return
	}

	// Unmarshal the JSON response
	var jsonData []map[string]any
	if err := json.Unmarshal(tags, &jsonData); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling JSON: %v\n", err)
		return
	}
	if len(jsonData) == 0 {
		fmt.Fprintf(os.Stderr, "No tags found.")
		return
	}

	latest_zip_url := jsonData[0]["zipball_url"].(string)
	latest_tag := jsonData[0]["name"].(string)
	latest_zip_path := filepath.Join(download_dir, latest_tag+".zip")

	// Download the latest zip file
	err = downloadFile(latest_zip_path, latest_zip_url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading file: %v\n", err)
		os.Exit(1)
	}

	// Unzip the downloaded file
	err = unzipFile(latest_zip_path, temp_dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unzipping file: %v\n", err)
		os.Exit(1)
	}

	// Move nested contents to the final directory
	err = moveNestedContents(temp_dir, unzip_dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error moving nested contents: %v\n", err)
		os.Exit(1)
	}

	// Cleanup
	os.RemoveAll(temp_dir)
	os.RemoveAll(download_dir)

	output := map[string]string{
		"message": "Download and extraction completed successfully",
		"latest_tag": latest_tag,
		"unzip_dir": unzip_dir,
	}

	jsonOutput, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonOutput))
}

func handleCleanup(input map[string]string) {
	unzipDir := input["unzip_dir"]
	err := os.RemoveAll(unzipDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"error": "failed to cleanup: %s"}`, err)
		os.Exit(1)
	}

	output := map[string]string{
		"message": "cleanup successful",
	}
	jsonOutput, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonOutput))
}

func main() {
	var input map[string]string
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to parse input: %s", err)
		os.Exit(1)
	}

	action := input["action"]

	switch action {
	case "download":
		handleDownload(input)
	case "cleanup":
		handleCleanup(input)
	default:
		fmt.Fprintf(os.Stderr, `{"error": "unknown action: %s"}`, action)
		os.Exit(1)
	}
}