# dbimport

CLI interactive pour importer des fichiers SQL dans des conteneurs Docker.

## Description

`dbimport` est un outil en ligne de commande qui facilite l'import de dumps SQL dans des bases de données tournant dans des conteneurs Docker. Il fournit une interface interactive pour sélectionner le fichier, le conteneur et configurer les paramètres de connexion.

## Fonctionnalités

- 🔍 **Navigation de fichiers** : Parcourez vos dossiers pour trouver les fichiers `.sql`, `.sql.gz`, `.dump`
- ☁️ **Import S3** : Téléchargez et importez directement depuis un bucket S3
- 🐳 **Détection automatique** : Détecte les conteneurs de bases de données (MySQL, PostgreSQL, MongoDB, Redis, etc.)
- 🗄️ **Multi-SGBD** : Supporte PostgreSQL et MySQL
- ⚡ **Auto-configuration** : Détecte le type de base depuis l'image Docker
- 🧹 **Vidage optionnel** : Possibilité de vider la base avant import
- 💾 **Mémoire** : Se souvient du dernier dossier utilisé et des connexions par conteneur
- 📅 **Tri automatique** : Fichiers triés par date (plus récents en premier)

## Prérequis

- Docker
- Conteneur de base de données actif

## Installation

### Méthode 1 : Depuis les releases (recommandée)

Téléchargez la dernière version pour votre architecture depuis la [page des releases](https://github.com/barberob/db-file-to-container-importer/releases).

```bash
# Linux (AMD64)
cd /tmp
curl -LO https://github.com/barberob/db-file-to-container-importer/releases/latest/download/dbimport_0.1.0_linux_amd64.tar.gz
tar -xzf dbimport_0.1.0_linux_amd64.tar.gz
sudo mv dbimport /usr/local/bin/
```

```bash
# Linux (ARM64)
cd /tmp
curl -LO https://github.com/barberob/db-file-to-container-importer/releases/latest/download/dbimport_0.1.0_linux_arm64.tar.gz
tar -xzf dbimport_0.1.0_linux_arm64.tar.gz
sudo mv dbimport /usr/local/bin/
```

### Méthode 2 : Depuis les sources (développeurs)

Nécessite Go 1.21+

```bash
# Cloner le repository
git clone https://github.com/barberob/db-file-to-container-importer.git
cd dbimport

# Télécharger les dépendances
go mod tidy

# Compiler
go build -o dbimport .

# Optionnel : installer dans le PATH
sudo cp dbimport /usr/local/bin/
```

## Utilisation

```bash
dbimport
```

### Flow interactif

1. **Source** : Choisissez entre fichier local ou bucket S3
2. **Sélection du fichier** : Naviguez pour choisir un fichier SQL (local ou S3)
3. **Sélection du conteneur** : Liste filtrée des conteneurs de bases de données
4. **Type de base** : Auto-détecté ou sélection manuelle (PostgreSQL/MySQL)
5. **Configuration** (pré-remplie si import précédent) :
   - Nom de la base
   - Utilisateur
   - Mot de passe
6. **Vidage** : Choisissez de vider la base avant import (Oui par défaut)
7. **Import** : Visualisez la progression avec un spinner

### Raccourcis

- `↑/↓` : Naviguer dans les listes
- `/` : Activer le filtre de recherche
- `Entrée` : Valider la sélection
- `Ctrl+C` : Quitter



## Structure du projet

```
dbimport/
├── main.go       # Logique principale et interface
├── docker.go     # Fonctions Docker (liste, détection)
├── config.go     # Gestion de la configuration
├── files.go      # Fonctions utilitaires fichiers
├── s3.go         # Support S3 (téléchargement, navigation)
├── go.mod        # Dépendances Go
├── go.sum        # Checksums
└── README.md     # Documentation
```

## Dépendances

- [charmbracelet/huh](https://github.com/charmbracelet/huh) - Composants interactifs
- [charmbracelet/huh/spinner](https://github.com/charmbracelet/huh) - Indicateur de progression
- [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) - Styles
- [aws/aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) - Client AWS/S3

## Docker supporté

Les conteneurs suivants sont automatiquement détectés :
- MySQL / MariaDB
- PostgreSQL
- MongoDB
- Redis
- Elasticsearch
- Cassandra
- CouchDB

## Configuration

Le CLI stocke sa configuration dans `~/.config/dbimport/` :

```
~/.config/dbimport/
├── lastdir              # Dernier dossier utilisé pour la navigation locale
├── s3config             # Configuration S3 (endpoint, clés, bucket, région)
└── containers/           # Configurations par conteneur
    ├── container1.json   # dbName, dbUser, dbPassword
    ├── container2.json
    └── ...
```

### S3 Support

Vous pouvez importer des fichiers directement depuis un bucket S3 :
- Les fichiers `.sql`, `.sql.gz`, `.dump` sont listés et triés par date
- La configuration S3 est sauvegardée après la première utilisation
- Supporte n'importe quel provider S3 compatible (AWS, Scaleway, MinIO, etc.)

### Mémoire par conteneur

Pour chaque conteneur, dbimport mémorise :
- Nom de la base de données
- Nom d'utilisateur
- Mot de passe

Ces informations sont pré-remplies automatiquement lors des imports ultérieurs.

## Développement

```bash
# Mode développement
go run .

# Tests
go test ./...

# Build
go build -o dbimport .
```

## Licence

MIT
