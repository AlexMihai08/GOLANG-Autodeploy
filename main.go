package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var working_folders sync.Map

func gitHead(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path
	result, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running git rev-parse command: %v", err)
	}
	return strings.TrimSpace(string(result)), nil
}

func readFolder(path string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("Error: could not read folders from", path, "-", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			folder_name := entry.Name()
			if _, inProgress := working_folders.Load(folder_name); !inProgress {
				working_folders.Store(folder_name, true)
				go func(folderName string) {
					defer working_folders.Delete(folderName)
					verifyFolder(path, folderName)
				}(folder_name)
			}

		}
	}
}

func pruneDocker() {
	pruneMu.Lock()
	defer pruneMu.Unlock()

	cmds := [][]string{
		{"image", "prune", "-af", "--filter", "until=72h"},
		{"builder", "prune", "-af", "--filter", "until=48h"},
	}
	for _, args := range cmds {
		out, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			fmt.Println("Error running docker", strings.Join(args, " "), "-", err, string(out))
			continue
		}
		fmt.Println(string(out))
	}
}

func verifyFolder(path string, folderName string) {
	cmd_dir := filepath.Join(path, folderName)

	dockerFile := filepath.Join(cmd_dir, "docker-compose.yml")
	if _, err := os.Stat(dockerFile); os.IsNotExist(err) {
		fmt.Printf("Folder %s does not contain a docker-compose.yml file\n", folderName)
		return
	}

	fmt.Println("Checking folder", folderName, "for git changes...\n")

	result_bef, err := gitHead(cmd_dir)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	cmd := exec.Command("git", "pull")
	cmd.Dir = cmd_dir
	_, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error running git pull command:", err)
		return
	}

	result_aft, err := gitHead(cmd_dir)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if result_aft != result_bef {
		fmt.Println("Git HEAD for folder", folderName, "changed from", result_bef, "to", result_aft, "\n")
		fmt.Println("Running docker-compose up -d for folder", folderName, "\n")

		cmd = exec.Command("docker", "compose", "up", "--build", "-d")
		cmd.Dir = cmd_dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("Error running docker compose up --build -d command:", err)
			return
		}
		pruneDocker()
		fmt.Println(string(output))
	}
}

var pruneMu sync.Mutex

func main() {
	rootPath := flag.String("path", "", "root path containing the project folders")
	flag.Parse()

	if *rootPath == "" {
		fmt.Println("Error: root path was not specified.")
		return
	}

	fmt.Println("autodeploy: checking folders in", *rootPath)

	timer := time.NewTicker(5 * time.Second)
	defer timer.Stop()
	defer fmt.Println("autodeploy: stopped checking folders in", *rootPath)

	for range timer.C {
		readFolder(*rootPath)
	}
}
