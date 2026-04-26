package ldtkgo

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// World is the top-level entry point for working with an LDtk project's simplified export.
// It loads everything in one call: project definitions, tileset images, levels with their
// layers, background images, and entity-to-definition links.
//
// Use LoadWorld to create a World from an .ldtk project file.
type World struct {
	// Project contains all definitions (tilesets, entities, layers) from the .ldtk file.
	Project *Project

	// Levels contains all loaded levels with entities linked and images resolved.
	Levels []*Level

	levelsByID map[string]*Level
}

// LoadWorld loads an LDtk project and all its simplified level exports in a single call.
//
// projectPath is the path to the .ldtk file relative to the file system root
// (e.g., "ldtk/pong.ldtk").
//
// LoadWorld automatically:
//   - Parses the .ldtk project for tileset, entity, and layer definitions
//   - Loads all tileset PNG images
//   - Discovers and loads each level's simplified data.json
//   - Links entities to their definitions and tilesets
//   - Loads background images and layer PNGs for every level
//
// The simplified export directory is expected at the standard LDtk location:
//
//	<dir>/<project_name>/simplified/<level_identifier>/data.json
//
// where <dir>/<project_name>.ldtk is the project file path.
func LoadWorld(projectPath string, fileSystem fs.FS) (*World, error) {
	// Parse paths: "ldtk/pong.ldtk" -> baseDir="ldtk", projectName="pong"
	baseDir := path.Dir(projectPath)
	if baseDir == "." {
		baseDir = ""
	}
	projectFile := path.Base(projectPath)
	projectName := strings.TrimSuffix(projectFile, path.Ext(projectFile))

	// Load project definitions
	project, err := OpenProject(projectPath, fileSystem)
	if err != nil {
		return nil, fmt.Errorf("loading project %s: %w", projectPath, err)
	}

	// Load all tileset images
	if err := project.LoadAllTilesetImages(baseDir, fileSystem); err != nil {
		return nil, fmt.Errorf("loading tileset images: %w", err)
	}

	world := &World{
		Project:    project,
		levelsByID: make(map[string]*Level, len(project.LevelRefs)),
	}

	// Load each level from its simplified export
	for _, ref := range project.LevelRefs {
		var levelDir string
		if baseDir == "" {
			levelDir = path.Join(projectName, "simplified", ref.Identifier)
		} else {
			levelDir = path.Join(baseDir, projectName, "simplified", ref.Identifier)
		}
		dataPath := path.Join(levelDir, "data.json")

		level, err := Open(dataPath, fileSystem)
		if err != nil {
			return nil, fmt.Errorf("loading level %s: %w", ref.Identifier, err)
		}

		// Link entity definitions and tilesets
		level.LinkProject(project)

		// Load background image
		if err := level.LoadBGImage(baseDir, fileSystem); err != nil {
			return nil, fmt.Errorf("loading bg image for %s: %w", ref.Identifier, err)
		}

		// Load layer images
		layers, err := level.LoadLayers(levelDir, fileSystem)
		if err != nil {
			return nil, fmt.Errorf("loading layers for %s: %w", ref.Identifier, err)
		}
		level.LoadedLayers = layers

		// Load IntGrid CSV layers
		if err := level.LoadIntGrids(levelDir, fileSystem, project); err != nil {
			return nil, fmt.Errorf("loading intgrids for %s: %w", ref.Identifier, err)
		}

		world.Levels = append(world.Levels, level)
		world.levelsByID[level.Identifier] = level
	}

	return world, nil
}

// Level returns the level with the given identifier, or nil if not found.
func (w *World) Level(identifier string) *Level {
	return w.levelsByID[identifier]
}

// LevelNames returns the identifiers of all loaded levels, in load order.
func (w *World) LevelNames() []string {
	names := make([]string, len(w.Levels))
	for i, l := range w.Levels {
		names[i] = l.Identifier
	}
	return names
}

// Tileset returns the tileset definition with the given identifier, or nil.
func (w *World) Tileset(identifier string) *TilesetDef {
	for _, t := range w.Project.Tilesets {
		if t.Identifier == identifier {
			return t
		}
	}
	return nil
}

// EntityDef returns the entity definition with the given identifier, or nil.
func (w *World) EntityDef(identifier string) *EntityDef {
	return w.Project.EntityDefByID(identifier)
}
