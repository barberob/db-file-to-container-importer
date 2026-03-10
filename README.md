# dbimport

CLI interactive pour importer des fichiers SQL dans des conteneurs Docker.

## Description

`dbimport` est un outil en ligne de commande qui facilite l'import de dumps SQL dans des bases de données tournant dans des conteneurs Docker. Il fournit une interface interactive pour sélectionner le fichier, le conteneur et configurer les paramètres de connexion.

## Fonctionnalités

- 🔍 **Navigation de fichiers** : Parcourez vos dossiers pour trouver les fichiers `.sql`, `.sql.gz`, `.dump`
- 🐳 **Détection automatique** : Détecte les conteneurs de bases de données (MySQL, PostgreSQL, MongoDB, Redis, etc.)
- 🗄️ **Multi-SGBD** : Supporte PostgreSQL et MySQL
- ⚡ **Auto-configuration** : Détecte le type de base depuis l'image Docker
- 🧹 **Vidage optionnel** : Possibilité de vider la base avant import
- 💾 **Mémoire** : Se souvient du dernier dossier utilisé

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
./dbimport
```

### Flow interactif

1. **Sélection du fichier** : Naviguez dans vos dossiers pour choisir un fichier SQL
2. **Sélection du conteneur** : Liste filtrée des conteneurs de bases de données
3. **Type de base** : Auto-détecté ou sélection manuelle (PostgreSQL/MySQL)
4. **Configuration** :
   - Nom de la base (défaut: `mydb`)
   - Utilisateur (défaut: `app` pour PostgreSQL, `root` pour MySQL)
   - Mot de passe (défaut: `app` pour PostgreSQL, `password` pour MySQL)
5. **Vidage** : Choisissez de vider la base avant import (Oui par défaut)
6. **Import** : Visualisez la progression avec un spinner

### Raccourcis

- `↑/↓` : Naviguer dans les listes
- `/` : Activer le filtre de recherche
- `Entrée` : Valider la sélection
- `Ctrl+C` : Quitter

## Valeurs par défaut

| Type | Utilisateur | Mot de passe |
|------|-------------|--------------|
| PostgreSQL | `app` | `app` |
| MySQL | `root` | `password` |

## Structure du projet

```
dbimport/
├── main.go       # Logique principale et interface
├── docker.go     # Fonctions Docker (liste, détection)
├── config.go     # Gestion de la configuration
├── files.go      # Fonctions utilitaires fichiers
├── go.mod        # Dépendances Go
├── go.sum        # Checksums
└── README.md     # Documentation
```

## Dépendances

- [charmbracelet/huh](https://github.com/charmbracelet/huh) - Composants interactifs
- [charmbracelet/huh/spinner](https://github.com/charmbracelet/huh) - Indicateur de progression
- [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) - Styles

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

Le CLI stocke le dernier dossier utilisé dans :
```
~/.config/dbimport/lastdir
```

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
