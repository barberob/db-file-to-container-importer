package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
)

const helpText = "↑/↓: naviguer  |  /: filtrer  |  Entrée: valider  |  Ctrl+C: quitter"

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Italic(true)

func main() {
	containers, err := listContainers()
	if err != nil || len(containers) == 0 {
		log.Fatal("Aucun conteneur Docker actif: ", err)
	}

	startDir := loadLastDir()
	selectedFile, err := browseForFile(startDir)
	if err != nil {
		os.Exit(0)
	}

	// Save the directory of the selected file for next time
	fileDir := filepath.Dir(selectedFile)
	if err := saveLastDir(fileDir); err != nil {
		log.Printf("Warning: could not save last directory: %v", err)
	}

	var (
		selectedContainer string
		dbType            string
		dbName            string
		dbUser            string
		dbPassword        string
		shouldEmptyDB     = true
	)

	// Step 1: Select container
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("🐳 Conteneur cible").
				Options(containerToOptions(containers)...).
				Value(&selectedContainer).
				Height(10).
				Filtering(true),
		),
	)

	if err := form.Run(); err != nil {
		os.Exit(0)
	}

	// Step 2: Auto-detect or select DB type
	dbType = detectDBType(selectedContainer)
	if dbType == "" {
		selectForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("🗄️  Type de base (auto-détection échouée)").
					Options(
						huh.NewOption("PostgreSQL", "pgsql"),
						huh.NewOption("MySQL", "mysql"),
					).
					Value(&dbType),
			),
		)
		if err := selectForm.Run(); err != nil {
			os.Exit(0)
		}
	}

	// Set default user and password placeholder based on DB type
	passwordPlaceholder := "password"
	if dbType == "pgsql" {
		dbUser = "app"
		passwordPlaceholder = "app"
	} else {
		dbUser = "root"
	}

	// Step 3: Configuration
	restForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Base de données").
				Placeholder("mydb").
				Value(&dbName),

			huh.NewInput().
				Title("Utilisateur DB").
				Placeholder(dbUser).
				Value(&dbUser),

			huh.NewInput().
				Title("Mot de passe DB").
				Placeholder(passwordPlaceholder).
				Password(true).
				Value(&dbPassword),
		),
	)

	if err := restForm.Run(); err != nil {
		os.Exit(0)
	}

	// Apply defaults
	if dbName == "" {
		dbName = "mydb"
	}
	if dbPassword == "" {
		if dbType == "pgsql" {
			dbPassword = "app"
		} else {
			dbPassword = "password"
		}
	}

	// Step 4: Ask if should empty database before import
	shouldEmptyDB = true // Default to yes
	emptyForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[bool]().
				Title(fmt.Sprintf("Vider la base '%s' avant l'import ?", dbName)).
				Options(
					huh.NewOption("Oui", true),
					huh.NewOption("Non", false),
				).
				Value(&shouldEmptyDB).
				Height(5),
		),
	)

	if err := emptyForm.Run(); err != nil {
		os.Exit(0)
	}

	importFn := func() {
		baseName := filepath.Base(selectedFile)
		dest := fmt.Sprintf("%s:/tmp/%s", selectedContainer, baseName)
		run("docker", "cp", selectedFile, dest)

		remoteFile := "/tmp/" + baseName

		if strings.HasSuffix(remoteFile, ".gz") {
			run("docker", "exec", selectedContainer,
				"gunzip", "-f", remoteFile,
			)
			remoteFile = strings.TrimSuffix(remoteFile, ".gz")
		}

		// Empty database if requested
		if shouldEmptyDB {
			fmt.Printf("🗑️  Vidage de la base '%s'...\n", dbName)
			switch dbType {
			case "pgsql":
				// Drop database using dropdb command (doesn't run in transaction)
				dropCmd := []string{"docker", "exec"}
				if dbPassword != "" {
					dropCmd = append(dropCmd, "-e", fmt.Sprintf("PGPASSWORD=%s", dbPassword))
				}
				dropCmd = append(dropCmd, selectedContainer, "dropdb", "--if-exists", "-U", dbUser, dbName)
				run(dropCmd[0], dropCmd[1:]...)
				
				// Create database using createdb command
				createCmd := []string{"docker", "exec"}
				if dbPassword != "" {
					createCmd = append(createCmd, "-e", fmt.Sprintf("PGPASSWORD=%s", dbPassword))
				}
				createCmd = append(createCmd, selectedContainer, "createdb", "-U", dbUser, dbName)
				run(createCmd[0], createCmd[1:]...)
			case "mysql":
				dropCmd := fmt.Sprintf("mysql -u %s", dbUser)
				if dbPassword != "" {
					dropCmd += fmt.Sprintf(" -p%s", dbPassword)
				}
				// Drop and recreate database
				dropCmd += fmt.Sprintf(" -e \"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s;\"", dbName, dbName)
				run("docker", "exec", selectedContainer, "sh", "-c", dropCmd)
			}
		}

		switch dbType {
		case "pgsql":
			pgCmd := []string{"docker", "exec"}
			if dbPassword != "" {
				pgCmd = append(pgCmd, "-e", fmt.Sprintf("PGPASSWORD=%s", dbPassword))
			}
			pgCmd = append(pgCmd, selectedContainer, "psql", "-U", dbUser, "-d", dbName, "-f", remoteFile)
			run(pgCmd[0], pgCmd[1:]...)
		case "mysql":
			mysqlCmd := fmt.Sprintf("mysql -u %s", dbUser)
			if dbPassword != "" {
				mysqlCmd += fmt.Sprintf(" -p%s", dbPassword)
			}
			mysqlCmd += fmt.Sprintf(" %s < %s", dbName, remoteFile)
			run(
				"docker", "exec", selectedContainer,
				"sh", "-c", mysqlCmd,
			)
		}

		run("docker", "exec", selectedContainer, "rm", "-f", remoteFile)
	}

	err = spinner.New().
		Title("Import en cours...").
		Action(importFn).
		Run()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("✅ Import terminé!")
}

func toOptions(items []string) []huh.Option[string] {
	opts := make([]huh.Option[string], len(items))
	for i, item := range items {
		opts[i] = huh.NewOption(item, item)
	}
	return opts
}

func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Erreur: %s %v → %v", name, args, err)
	}
}

func browseForFile(startDir string) (string, error) {
	currentDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		entries, err := os.ReadDir(currentDir)
		if err != nil {
			return "", fmt.Errorf("impossible de lire le répertoire %s: %v", currentDir, err)
		}

		var options []huh.Option[string]

		// Option to go up
		parentDir := filepath.Dir(currentDir)
		if parentDir != currentDir {
			options = append(options, huh.NewOption("📁 .. (dossier parent)", "__parent__"))
		}

		// Directories
		for _, e := range entries {
			if e.IsDir() {
				name := e.Name()
				if !strings.HasPrefix(name, ".") {
					options = append(options, huh.NewOption("📁 "+name, "dir:"+name))
				}
			}
		}

		// SQL files
		validExts := []string{".sql", ".sql.gz", ".dump"}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			for _, ext := range validExts {
				if strings.HasSuffix(name, ext) {
					options = append(options, huh.NewOption("📄 "+name, "file:"+name))
					break
				}
			}
		}

		if len(options) == 0 {
			return "", fmt.Errorf("aucun fichier SQL trouvé dans %s", currentDir)
		}

		var selected string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("📂 %s\n%s", currentDir, helpStyle.Render(helpText))).
					Options(options...).
					Value(&selected).
					Height(20).
					Filtering(true),
			),
		)

		if err := form.Run(); err != nil {
			return "", err
		}

		switch {
		case selected == "__parent__":
			currentDir = parentDir
		case strings.HasPrefix(selected, "dir:"):
			dirName := strings.TrimPrefix(selected, "dir:")
			currentDir = filepath.Join(currentDir, dirName)
		case strings.HasPrefix(selected, "file:"):
			fileName := strings.TrimPrefix(selected, "file:")
			return filepath.Join(currentDir, fileName), nil
		}
	}
}
